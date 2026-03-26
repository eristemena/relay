package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/repository"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	toolspkg "github.com/erisristemena/relay/internal/tools"
)

type Store interface {
	ListSessions(ctx context.Context) ([]sqlite.Session, error)
	GetSession(ctx context.Context, sessionID string) (sqlite.Session, error)
	CreateSession(ctx context.Context, displayName string) (sqlite.Session, error)
	OpenSession(ctx context.Context, sessionID string) (sqlite.Session, error)
	CreateAgentRun(ctx context.Context, sessionID string, taskText string, role sqlite.AgentRole, model string) (sqlite.AgentRun, error)
	CreateAgentExecution(ctx context.Context, execution sqlite.AgentExecution) (sqlite.AgentExecution, error)
	GetActiveRun(ctx context.Context) (sqlite.AgentRun, error)
	GetAgentRun(ctx context.Context, runID string) (sqlite.AgentRun, error)
	ListAgentExecutions(ctx context.Context, runID string) ([]sqlite.AgentExecution, error)
	ListRunSummaries(ctx context.Context, sessionID string) ([]sqlite.RunSummary, error)
	AppendRunEvent(ctx context.Context, runID string, eventType string, role sqlite.AgentRole, model string, payloadJSON string) (sqlite.AgentRunEvent, error)
	ListRunEvents(ctx context.Context, runID string) ([]sqlite.AgentRunEvent, error)
	CreateApprovalRequest(ctx context.Context, approval sqlite.ApprovalRequest) (sqlite.ApprovalRequest, error)
	GetApprovalRequest(ctx context.Context, runID string, toolCallID string) (sqlite.ApprovalRequest, error)
	ListPendingApprovalRequests(ctx context.Context, sessionID string) ([]sqlite.ApprovalRequest, error)
	ListPendingApprovalRequestsForRun(ctx context.Context, runID string) ([]sqlite.ApprovalRequest, error)
	UpdateApprovalRequestState(ctx context.Context, runID string, toolCallID string, state string, reviewedAt *time.Time, appliedAt *time.Time) error
	ResolvePendingApprovalRequestsForRun(ctx context.Context, runID string, state string, reviewedAt *time.Time) error
	UpdateAgentRun(ctx context.Context, run sqlite.AgentRun) error
	UpdateAgentExecution(ctx context.Context, execution sqlite.AgentExecution) error
}

type Service struct {
	store                    Store
	paths                    config.Paths
	logger                   *slog.Logger
	mu                       sync.Mutex
	runnerFactory            func(cfg config.Config, task string) agents.Runner
	agentFactory             func(cfg config.Config, role sqlite.AgentRole) agents.Agent
	forceLegacyRunnerPath    bool
	activeRuns               map[string]activeRunRegistration
	pendingApprovals         map[string]pendingApproval
	repositoryGraphs         map[string]repositoryGraphCacheEntry
	repositoryGraphBuilds    map[string]context.CancelFunc
	repositoryGraphBuilder   repositoryGraphBuilder
	repositoryGraphSignature repositoryGraphSignatureFunc
	workspaceSubscribers     map[string]func(StreamEnvelope) error
}

type activeRunRegistration struct {
	cancel      context.CancelFunc
	subscribers map[string]*runSubscriber
}

type runSubscriber struct {
	emit      func(StreamEnvelope) error
	replaying bool
	buffered  []StreamEnvelope
}

type ApprovalResponseInput struct {
	SessionID  string
	RunID      string
	ToolCallID string
	Decision   string
}

type ApprovalRequest struct {
	SessionID    string
	RunID        string
	Role         sqlite.AgentRole
	Model        string
	ToolCallID   string
	ToolName     agents.ToolName
	InputPreview map[string]any
	Message      string
	OccurredAt   time.Time
}

type ApprovalDecision struct {
	Approved bool
}

type pendingApproval struct {
	RunID      string
	ToolCallID string
	Respond    chan ApprovalDecision
}

var ErrApprovalRejected = errors.New("Relay blocked the tool call because approval was rejected")

type SessionSummary struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
	LastOpenedAt time.Time `json:"last_opened_at"`
	Status       string    `json:"status"`
	HasActivity  bool      `json:"has_activity"`
}

type UIState struct {
	HistoryState string `json:"history_state"`
	CanvasState  string `json:"canvas_state"`
	SaveState    string `json:"save_state"`
}

type WorkspaceSnapshot struct {
	ActiveSessionID     string                     `json:"active_session_id"`
	Sessions            []SessionSummary           `json:"sessions"`
	Preferences         config.SafePreferences     `json:"preferences"`
	ConnectedRepository ConnectedRepositorySummary `json:"connected_repository"`
	RepositoryGraph     RepositoryGraphState       `json:"repository_graph"`
	UIState             UIState                    `json:"ui_state"`
	ActiveRunID         string                     `json:"active_run_id,omitempty"`
	RunSummaries        []RunSummary               `json:"run_summaries,omitempty"`
	PendingApprovals    []ApprovalSummary          `json:"pending_approvals,omitempty"`
	CredentialStatus    CredentialStatus           `json:"credential_status"`
	Warnings            []string                   `json:"warnings,omitempty"`
}

type ConnectedRepositorySummary struct {
	Path    string `json:"path,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ApprovalSummary struct {
	SessionID      string           `json:"session_id"`
	RunID          string           `json:"run_id"`
	Role           sqlite.AgentRole `json:"role,omitempty"`
	Model          string           `json:"model,omitempty"`
	ToolCallID     string           `json:"tool_call_id"`
	ToolName       string           `json:"tool_name"`
	RequestKind    string           `json:"request_kind,omitempty"`
	Status         string           `json:"status,omitempty"`
	RepositoryRoot string           `json:"repository_root,omitempty"`
	InputPreview   map[string]any   `json:"input_preview"`
	DiffPreview    map[string]any   `json:"diff_preview,omitempty"`
	CommandPreview map[string]any   `json:"command_preview,omitempty"`
	Message        string           `json:"message"`
	OccurredAt     time.Time        `json:"occurred_at"`
}

type CredentialStatus struct {
	Configured bool `json:"configured"`
}

type RunSummary struct {
	ID              string           `json:"id"`
	TaskTextPreview string           `json:"task_text_preview"`
	Role            sqlite.AgentRole `json:"role"`
	Model           string           `json:"model"`
	State           string           `json:"state"`
	ErrorCode       string           `json:"error_code,omitempty"`
	StartedAt       time.Time        `json:"started_at"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty"`
	HasToolActivity bool             `json:"has_tool_activity"`
}

type SubmitRunInput struct {
	SessionID string
	Task      string
}

type OpenRunInput struct {
	SessionID string
	RunID     string
}

type CancelRunInput struct {
	SessionID string
	RunID     string
}

type StreamEnvelope struct {
	Type    string
	Payload any
}

type CredentialInput struct {
	Provider string `json:"provider"`
	Label    string `json:"label,omitempty"`
	Secret   string `json:"secret"`
}

type PreferencesInput struct {
	PreferredPort      *int
	AppearanceVariant  *string
	Credentials        []CredentialInput
	ReplaceCredentials bool
	OpenRouterAPIKey   *string
	ProjectRoot        *string
	OpenBrowserOnStart *bool
}

func NewService(store Store, paths config.Paths) *Service {
	service := &Service{
		store:                 store,
		paths:                 paths,
		logger:                slog.Default(),
		activeRuns:            make(map[string]activeRunRegistration),
		pendingApprovals:      make(map[string]pendingApproval),
		repositoryGraphs:      make(map[string]repositoryGraphCacheEntry),
		repositoryGraphBuilds: make(map[string]context.CancelFunc),
		workspaceSubscribers:  make(map[string]func(StreamEnvelope) error),
	}
	service.repositoryGraphBuilder = defaultRepositoryGraphBuilder
	service.repositoryGraphSignature = defaultRepositoryGraphSignature
	service.runnerFactory = func(cfg config.Config, task string) agents.Runner {
		registry := agents.NewRegistry(cfg.Agents.WithDefaults())
		return registry.NewRunner(cfg.OpenRouter.APIKey, task, newCatalogToolExecutor(cfg.ProjectRoot, service))
	}
	service.agentFactory = func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		registry := agents.NewRegistry(cfg.Agents.WithDefaults())
		return registry.NewAgent(cfg.OpenRouter.APIKey, role, newCatalogToolExecutor(cfg.ProjectRoot, service))
	}
	return service
}

func (s *Service) SetLogger(logger *slog.Logger) {
	if logger == nil {
		return
	}
	s.logger = logger
}

func (s *Service) SetRunnerFactory(factory func(cfg config.Config, task string) agents.Runner) {
	if factory == nil {
		return
	}
	s.runnerFactory = factory
	s.forceLegacyRunnerPath = true
}

func (s *Service) SetAgentFactory(factory func(cfg config.Config, role sqlite.AgentRole) agents.Agent) {
	if factory == nil {
		return
	}
	s.agentFactory = factory
	s.forceLegacyRunnerPath = false
}

func (s *Service) registerActiveRun(runID string, cancel context.CancelFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeRuns[runID] = activeRunRegistration{
		cancel:      cancel,
		subscribers: make(map[string]*runSubscriber),
	}
}

func (s *Service) activeRunCancel(runID string) (context.CancelFunc, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	registration, ok := s.activeRuns[runID]
	if !ok {
		return nil, false
	}
	return registration.cancel, true
}

func (s *Service) hasTrackedActiveRun(runID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.activeRuns[runID]
	return ok
}

func (s *Service) clearActiveRun(runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeRuns, runID)
	for key, pending := range s.pendingApprovals {
		if pending.RunID == runID {
			delete(s.pendingApprovals, key)
		}
	}
}

func (s *Service) attachRunSubscriber(ctx context.Context, runID string, emit func(StreamEnvelope) error, replaying bool) (string, bool) {
	subscriberID, ok := streamSubscriberFromContext(ctx)
	if !ok || strings.TrimSpace(subscriberID) == "" || emit == nil {
		return "", false
	}

	s.mu.Lock()
	registration, exists := s.activeRuns[runID]
	if !exists {
		s.mu.Unlock()
		return "", false
	}
	registration.subscribers[subscriberID] = &runSubscriber{
		emit:      emit,
		replaying: replaying,
	}
	s.activeRuns[runID] = registration
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.detachRunSubscriber(runID, subscriberID)
	}()

	return subscriberID, true
}

func (s *Service) detachRunSubscriber(runID string, subscriberID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	registration, ok := s.activeRuns[runID]
	if !ok {
		return
	}
	delete(registration.subscribers, subscriberID)
	s.activeRuns[runID] = registration
}

func (s *Service) dispatchRunEnvelope(runID string, envelope StreamEnvelope) error {
	type delivery struct {
		subscriberID string
		emit         func(StreamEnvelope) error
		envelope     StreamEnvelope
	}

	deliveries := make([]delivery, 0)
	s.mu.Lock()
	registration, ok := s.activeRuns[runID]
	if ok {
		for subscriberID, subscriber := range registration.subscribers {
			if subscriber.replaying {
				subscriber.buffered = append(subscriber.buffered, envelope)
				continue
			}
			deliveries = append(deliveries, delivery{
				subscriberID: subscriberID,
				emit:         subscriber.emit,
				envelope:     envelope,
			})
		}
	}
	s.mu.Unlock()

	for _, delivery := range deliveries {
		if err := delivery.emit(delivery.envelope); err != nil {
			s.detachRunSubscriber(runID, delivery.subscriberID)
		}
	}
	return nil
}

func (s *Service) AttachWorkspaceSubscriber(ctx context.Context, emit func(StreamEnvelope) error) {
	subscriberID, ok := streamSubscriberFromContext(ctx)
	if !ok || strings.TrimSpace(subscriberID) == "" || emit == nil {
		return
	}

	s.mu.Lock()
	s.workspaceSubscribers[subscriberID] = emit
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		delete(s.workspaceSubscribers, subscriberID)
		s.mu.Unlock()
	}()
}

func (s *Service) dispatchWorkspaceEnvelope(envelope StreamEnvelope) {
	type delivery struct {
		subscriberID string
		emit         func(StreamEnvelope) error
	}

	deliveries := make([]delivery, 0)
	s.mu.Lock()
	for subscriberID, emit := range s.workspaceSubscribers {
		deliveries = append(deliveries, delivery{subscriberID: subscriberID, emit: emit})
	}
	s.mu.Unlock()

	for _, delivery := range deliveries {
		if err := delivery.emit(envelope); err != nil {
			s.mu.Lock()
			delete(s.workspaceSubscribers, delivery.subscriberID)
			s.mu.Unlock()
		}
	}
}

func (s *Service) finishReplay(runID string, subscriberID string, replayMaxSequence int64) error {
	type delivery struct {
		emit     func(StreamEnvelope) error
		envelope StreamEnvelope
	}

	var deliveries []delivery
	s.mu.Lock()
	registration, ok := s.activeRuns[runID]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	subscriber, ok := registration.subscribers[subscriberID]
	if !ok {
		s.mu.Unlock()
		return nil
	}
	buffered := subscriber.buffered
	subscriber.buffered = nil
	subscriber.replaying = false
	for _, envelope := range buffered {
		sequence, hasSequence := sequenceFromEnvelope(envelope)
		if hasSequence && sequence <= replayMaxSequence {
			continue
		}
		deliveries = append(deliveries, delivery{emit: subscriber.emit, envelope: envelope})
	}
	s.activeRuns[runID] = registration
	s.mu.Unlock()

	for _, delivery := range deliveries {
		if err := delivery.emit(delivery.envelope); err != nil {
			s.detachRunSubscriber(runID, subscriberID)
			return err
		}
	}
	return nil
}

func sequenceFromEnvelope(envelope StreamEnvelope) (int64, bool) {
	payload, ok := envelope.Payload.(map[string]any)
	if !ok {
		return 0, false
	}
	value, ok := payload["sequence"]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case float64:
		return int64(typed), true
	default:
		return 0, false
	}
}

func approvalPayload(summary ApprovalSummary) map[string]any {
	payload := map[string]any{
		"session_id":    summary.SessionID,
		"run_id":        summary.RunID,
		"tool_call_id":  summary.ToolCallID,
		"tool_name":     summary.ToolName,
		"status":        sqlite.ApprovalStateProposed,
		"input_preview": summary.InputPreview,
		"message":       summary.Message,
		"occurred_at":   summary.OccurredAt.UTC().Format(time.RFC3339),
	}
	if requestKind, ok := summary.InputPreview["request_kind"].(string); ok && strings.TrimSpace(requestKind) != "" {
		payload["request_kind"] = requestKind
	}
	if repositoryRoot, ok := summary.InputPreview["repository_root"].(string); ok && strings.TrimSpace(repositoryRoot) != "" {
		payload["repository_root"] = repositoryRoot
	}
	if diffPreview, ok := summary.InputPreview["diff_preview"].(map[string]any); ok {
		payload["diff_preview"] = diffPreview
	}
	if commandPreview, ok := summary.InputPreview["command_preview"].(map[string]any); ok {
		payload["command_preview"] = commandPreview
	}
	if summary.Role != "" {
		payload["role"] = summary.Role
	}
	if strings.TrimSpace(summary.Model) != "" {
		payload["model"] = summary.Model
	}
	return payload
}

func approvalStateChangedPayload(summary ApprovalSummary, state string, occurredAt time.Time, message string) map[string]any {
	payload := approvalPayload(summary)
	payload["status"] = state
	payload["message"] = strings.TrimSpace(message)
	payload["occurred_at"] = occurredAt.UTC().Format(time.RFC3339)
	return payload
}

func approvalStateChangeMessage(state string) string {
	switch state {
	case sqlite.ApprovalStateApproved:
		return "Relay recorded approval and is revalidating before apply."
	case sqlite.ApprovalStateApplied:
		return "Relay applied the approved tool call successfully."
	case sqlite.ApprovalStateRejected:
		return "Relay recorded the rejection and blocked the tool call."
	case sqlite.ApprovalStateBlocked:
		return "Relay blocked the tool call because the approval was no longer safe to continue."
	case sqlite.ApprovalStateExpired:
		return "Relay expired the approval because the request was no longer actionable."
	default:
		return "Relay updated the approval state."
	}
}

func (s *Service) emitApprovalStateChanged(ctx context.Context, summary ApprovalSummary, state string, occurredAt time.Time, message string) error {
	run, err := s.store.GetAgentRun(ctx, summary.RunID)
	if err != nil {
		return err
	}
	role := run.Role
	model := run.Model
	if summary.Role != "" {
		role = summary.Role
	}
	if strings.TrimSpace(summary.Model) != "" {
		model = summary.Model
	}
	payload := approvalStateChangedPayload(ApprovalSummary{
		SessionID:    summary.SessionID,
		RunID:        summary.RunID,
		Role:         role,
		Model:        model,
		ToolCallID:   summary.ToolCallID,
		ToolName:     summary.ToolName,
		InputPreview: summary.InputPreview,
		Message:      summary.Message,
		OccurredAt:   summary.OccurredAt,
	}, state, occurredAt, message)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal approval state payload: %w", err)
	}
	storedEvent, err := s.store.AppendRunEvent(ctx, summary.RunID, sqlite.EventTypeApprovalStateChanged, role, model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = storedEvent.Sequence
	if err := s.dispatchRunEnvelope(summary.RunID, StreamEnvelope{Type: sqlite.EventTypeApprovalStateChanged, Payload: payload}); err != nil {
		return nil
	}
	return nil
}

func (s *Service) pendingApprovalPayloads(ctx context.Context, runID string) ([]map[string]any, error) {
	approvals, err := s.store.ListPendingApprovalRequestsForRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	payloads := make([]map[string]any, 0, len(approvals))
	for _, approval := range approvals {
		summary, err := approvalSummaryFromRecord(approval)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, approvalPayload(summary))
	}
	return payloads, nil
}

func approvalKey(runID string, toolCallID string) string {
	return strings.TrimSpace(runID) + ":" + strings.TrimSpace(toolCallID)
}

func approvalSummaryFromRecord(record sqlite.ApprovalRequest) (ApprovalSummary, error) {
	inputPreview := make(map[string]any)
	if strings.TrimSpace(record.InputPreviewJSON) != "" {
		if err := json.Unmarshal([]byte(record.InputPreviewJSON), &inputPreview); err != nil {
			return ApprovalSummary{}, fmt.Errorf("decode approval input preview: %w", err)
		}
	}

	return ApprovalSummary{
		SessionID:      record.SessionID,
		RunID:          record.RunID,
		Role:           record.Role,
		Model:          record.Model,
		ToolCallID:     record.ToolCallID,
		ToolName:       record.ToolName,
		RequestKind:    stringValueFromPreview(inputPreview, "request_kind"),
		Status:         record.State,
		RepositoryRoot: stringValueFromPreview(inputPreview, "repository_root"),
		InputPreview:   inputPreview,
		DiffPreview:    mapValueFromPreview(inputPreview, "diff_preview"),
		CommandPreview: mapValueFromPreview(inputPreview, "command_preview"),
		Message:        record.Message,
		OccurredAt:     record.OccurredAt,
	}, nil
}

func stringValueFromPreview(preview map[string]any, key string) string {
	value, _ := preview[key].(string)
	return strings.TrimSpace(value)
}

func mapValueFromPreview(preview map[string]any, key string) map[string]any {
	value, _ := preview[key].(map[string]any)
	return value
}

func (s *Service) RequestApproval(ctx context.Context, request ApprovalRequest) (ApprovalDecision, error) {
	runContext, ok := runExecutionContextFromContext(ctx)
	if !ok {
		return ApprovalDecision{}, fmt.Errorf("approval flow is unavailable for this run")
	}

	key := approvalKey(request.RunID, request.ToolCallID)
	inputPreviewJSON, err := json.Marshal(request.InputPreview)
	if err != nil {
		return ApprovalDecision{}, fmt.Errorf("marshal approval input preview: %w", err)
	}
	record, err := s.store.CreateApprovalRequest(ctx, sqlite.ApprovalRequest{
		SessionID:        request.SessionID,
		RunID:            request.RunID,
		ToolCallID:       request.ToolCallID,
		ToolName:         string(request.ToolName),
		Role:             request.Role,
		Model:            request.Model,
		InputPreviewJSON: string(inputPreviewJSON),
		Message:          request.Message,
		State:            sqlite.ApprovalStateProposed,
		OccurredAt:       request.OccurredAt.UTC(),
	})
	if err != nil {
		return ApprovalDecision{}, err
	}
	summary, err := approvalSummaryFromRecord(record)
	if err != nil {
		return ApprovalDecision{}, err
	}
	pending := pendingApproval{
		RunID:      request.RunID,
		ToolCallID: request.ToolCallID,
		Respond:    make(chan ApprovalDecision, 1),
	}

	s.mu.Lock()
	if _, exists := s.pendingApprovals[key]; exists {
		s.mu.Unlock()
		return ApprovalDecision{}, fmt.Errorf("approval is already pending for tool call %s", request.ToolCallID)
	}
	s.pendingApprovals[key] = pending
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pendingApprovals, key)
		s.mu.Unlock()
	}()

	if err := runContext.Emit(StreamEnvelope{
		Type:    "approval_request",
		Payload: approvalPayload(summary),
	}); err != nil {
		now := time.Now().UTC()
		_ = s.store.UpdateApprovalRequestState(context.WithoutCancel(ctx), request.RunID, request.ToolCallID, sqlite.ApprovalStateBlocked, &now, nil)
		_ = s.emitApprovalStateChanged(context.WithoutCancel(ctx), summary, sqlite.ApprovalStateBlocked, now, approvalStateChangeMessage(sqlite.ApprovalStateBlocked))
		return ApprovalDecision{}, err
	}

	select {
	case decision := <-pending.Respond:
		return decision, nil
	case <-ctx.Done():
		now := time.Now().UTC()
		_ = s.store.UpdateApprovalRequestState(context.WithoutCancel(ctx), request.RunID, request.ToolCallID, sqlite.ApprovalStateBlocked, &now, nil)
		_ = s.emitApprovalStateChanged(context.WithoutCancel(ctx), summary, sqlite.ApprovalStateBlocked, now, approvalStateChangeMessage(sqlite.ApprovalStateBlocked))
		return ApprovalDecision{}, ctx.Err()
	}
}

func (s *Service) ResolveApproval(ctx context.Context, input ApprovalResponseInput) (WorkspaceSnapshot, error) {
	run, err := s.store.GetAgentRun(ctx, input.RunID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	if run.SessionID != input.SessionID {
		return WorkspaceSnapshot{}, fmt.Errorf("run %s does not belong to session %s", input.RunID, input.SessionID)
	}

	decision := strings.TrimSpace(strings.ToLower(input.Decision))
	if decision != "approved" && decision != "rejected" {
		return WorkspaceSnapshot{}, fmt.Errorf("unsupported approval decision %q", input.Decision)
	}

	key := approvalKey(input.RunID, input.ToolCallID)
	s.mu.Lock()
	pending, ok := s.pendingApprovals[key]
	s.mu.Unlock()
	if !ok {
		approval, approvalErr := s.store.GetApprovalRequest(ctx, input.RunID, input.ToolCallID)
		if approvalErr != nil || approval.State != sqlite.ApprovalStateProposed {
			return WorkspaceSnapshot{}, fmt.Errorf("approval request %s is no longer pending", input.ToolCallID)
		}
		pending = pendingApproval{RunID: input.RunID, ToolCallID: input.ToolCallID}
	}
	reviewedAt := time.Now().UTC()
	nextState := sqlite.ApprovalStateRejected
	if decision == "approved" {
		approval, approvalErr := s.store.GetApprovalRequest(ctx, input.RunID, input.ToolCallID)
		if approvalErr != nil {
			return WorkspaceSnapshot{}, approvalErr
		}
		validatedState, validatedMessage, stale, validationErr := s.revalidateApprovalRequest(ctx, approval)
		if validationErr != nil {
			return WorkspaceSnapshot{}, validationErr
		}
		if stale {
			nextState = validatedState
			if err := s.store.UpdateApprovalRequestState(ctx, input.RunID, input.ToolCallID, nextState, &reviewedAt, nil); err != nil {
				return WorkspaceSnapshot{}, err
			}
			summary, err := approvalSummaryFromRecord(approval)
			if err != nil {
				return WorkspaceSnapshot{}, err
			}
			if err := s.emitApprovalStateChanged(ctx, summary, nextState, reviewedAt, validatedMessage); err != nil {
				return WorkspaceSnapshot{}, err
			}
			if ok {
				select {
				case pending.Respond <- ApprovalDecision{Approved: false}:
				default:
					return WorkspaceSnapshot{}, fmt.Errorf("approval request %s could not accept another decision", input.ToolCallID)
				}
			}
			return s.Bootstrap(ctx, input.SessionID)
		}
		_ = validatedMessage
		nextState = sqlite.ApprovalStateApproved
	}
	if err := s.store.UpdateApprovalRequestState(ctx, input.RunID, input.ToolCallID, nextState, &reviewedAt, nil); err != nil {
		return WorkspaceSnapshot{}, err
	}
	approval, err := s.store.GetApprovalRequest(ctx, input.RunID, input.ToolCallID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	summary, err := approvalSummaryFromRecord(approval)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	if err := s.emitApprovalStateChanged(ctx, summary, nextState, reviewedAt, approvalStateChangeMessage(nextState)); err != nil {
		return WorkspaceSnapshot{}, err
	}

	if ok {
		select {
		case pending.Respond <- ApprovalDecision{Approved: decision == "approved"}:
		default:
			return WorkspaceSnapshot{}, fmt.Errorf("approval request %s could not accept another decision", input.ToolCallID)
		}
	}

	return s.Bootstrap(ctx, input.SessionID)
}

func (s *Service) revalidateApprovalRequest(_ context.Context, approval sqlite.ApprovalRequest) (string, string, bool, error) {
	inputPreview := make(map[string]any)
	if strings.TrimSpace(approval.InputPreviewJSON) != "" {
		if err := json.Unmarshal([]byte(approval.InputPreviewJSON), &inputPreview); err != nil {
			return "", "", false, fmt.Errorf("decode approval input preview: %w", err)
		}
	}

	repositoryRoot, _ := inputPreview["repository_root"].(string)
	if strings.TrimSpace(repositoryRoot) == "" {
		return "", "", false, nil
	}

	cfg, _, err := config.Load(s.paths)
	if err != nil {
		return "", "", false, err
	}
	rootStatus := repository.ValidateRoot(cfg.ProjectRoot)
	if !rootStatus.Valid || rootStatus.Root != repositoryRoot {
		return sqlite.ApprovalStateBlocked, "Relay blocked the approval because the connected repository changed or is no longer valid.", true, nil
	}

	requestKind, _ := inputPreview["request_kind"].(string)
	if requestKind != toolspkg.RequestKindFileWrite {
		return "", "", false, nil
	}

	diffPreview, ok := inputPreview["diff_preview"].(map[string]any)
	if !ok {
		return "", "", false, nil
	}
	targetPath, _ := diffPreview["target_path"].(string)
	expectedHash, _ := diffPreview["base_content_hash"].(string)
	if strings.TrimSpace(targetPath) == "" || strings.TrimSpace(expectedHash) == "" {
		return "", "", false, nil
	}
	currentHash, err := toolspkg.CurrentWriteFileBaseHash(rootStatus.Root, targetPath)
	if err != nil {
		return sqlite.ApprovalStateBlocked, "Relay blocked the approval because it could not revalidate the file before apply.", true, nil
	}
	if currentHash != expectedHash {
		return sqlite.ApprovalStateExpired, "Relay expired the approval because the file changed after the diff was prepared.", true, nil
	}
	return "", "", false, nil
}

func (s *Service) Bootstrap(ctx context.Context, lastSessionID string) (WorkspaceSnapshot, error) {
	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	s.syncRepositoryGraph(ctx, cfg)

	sessions, err := s.store.ListSessions(ctx)
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("load saved sessions: %w", err)
	}

	activeSessionID := strings.TrimSpace(lastSessionID)
	if activeSessionID == "" {
		activeSessionID = strings.TrimSpace(cfg.LastSessionID)
	}

	if activeSessionID != "" {
		if _, err := s.store.GetSession(ctx, activeSessionID); err != nil {
			activeSessionID = ""
			warnings = append(warnings, "The last active session is no longer available, so Relay opened the workspace without restoring it.")
		}
	}

	if activeSessionID == "" && len(sessions) > 0 {
		activeSessionID = sessions[0].ID
	}

	runSummaries := make([]RunSummary, 0)
	activeRunID := ""
	pendingApprovals := make([]ApprovalSummary, 0)
	if activeSessionID != "" {
		activeRun, hasActiveRun, err := s.reconcileActiveRun(ctx)
		if err != nil {
			return WorkspaceSnapshot{}, fmt.Errorf("reconcile active run: %w", err)
		}

		storedRuns, err := s.store.ListRunSummaries(ctx, activeSessionID)
		if err != nil {
			return WorkspaceSnapshot{}, fmt.Errorf("load run summaries: %w", err)
		}
		runSummaries = summarizeRuns(storedRuns)
		if hasActiveRun && activeRun.SessionID == activeSessionID {
			activeRunID = activeRun.ID
		}

		storedApprovals, err := s.store.ListPendingApprovalRequests(ctx, activeSessionID)
		if err != nil {
			return WorkspaceSnapshot{}, fmt.Errorf("load pending approvals: %w", err)
		}
		pendingApprovals = make([]ApprovalSummary, 0, len(storedApprovals))
		for _, approval := range storedApprovals {
			summary, err := approvalSummaryFromRecord(approval)
			if err != nil {
				return WorkspaceSnapshot{}, err
			}
			pendingApprovals = append(pendingApprovals, summary)
		}
	}

	connectedRepository := connectedRepositorySummary(cfg.SafePreferences())
	graphState := s.currentRepositoryGraph(connectedRepository)

	return WorkspaceSnapshot{
		ActiveSessionID:     activeSessionID,
		Sessions:            summarizeSessions(sessions),
		Preferences:         cfg.SafePreferences(),
		ConnectedRepository: connectedRepository,
		RepositoryGraph:     graphState,
		UIState: UIState{
			HistoryState: "ready",
			CanvasState:  canvasStateForSessions(sessions, activeSessionID),
			SaveState:    "idle",
		},
		ActiveRunID:      activeRunID,
		RunSummaries:     runSummaries,
		PendingApprovals: pendingApprovals,
		CredentialStatus: CredentialStatus{Configured: cfg.HasOpenRouterKey()},
		Warnings:         warnings,
	}, nil
}

func (s *Service) currentRepositoryGraph(connected ConnectedRepositorySummary) RepositoryGraphState {
	if strings.TrimSpace(connected.Path) == "" {
		return repositoryGraphStateFromConnection(connected, repositoryGraphCacheEntry{}, false)
	}

	s.mu.Lock()
	entry, ok := s.repositoryGraphs[connected.Path]
	s.mu.Unlock()
	return repositoryGraphStateFromConnection(connected, entry, ok)
}

func connectedRepositorySummary(preferences config.SafePreferences) ConnectedRepositorySummary {
	if preferences.ProjectRootValid && strings.TrimSpace(preferences.ProjectRoot) != "" {
		return ConnectedRepositorySummary{
			Path:    preferences.ProjectRoot,
			Status:  "connected",
			Message: "Repository-aware reads stay inside this local Git worktree.",
		}
	}

	if preferences.ProjectRootConfigured {
		message := strings.TrimSpace(preferences.ProjectRootMessage)
		if message == "" {
			message = "Relay could not use the saved project root. Choose a valid local Git repository."
		}
		return ConnectedRepositorySummary{
			Path:    preferences.ProjectRoot,
			Status:  "invalid",
			Message: message,
		}
	}

	return ConnectedRepositorySummary{
		Status:  "not_configured",
		Message: "Choose a local Git repository to enable repository-aware tools.",
	}
}

func (s *Service) reconcileActiveRun(ctx context.Context) (sqlite.AgentRun, bool, error) {
	activeRun, err := s.store.GetActiveRun(ctx)
	if errors.Is(err, sqlite.ErrRunNotFound) {
		return sqlite.AgentRun{}, false, nil
	}
	if err != nil {
		return sqlite.AgentRun{}, false, err
	}
	if s.hasTrackedActiveRun(activeRun.ID) {
		return activeRun, true, nil
	}
	if activeRun.State != sqlite.RunStateToolRunning {
		return activeRun, true, nil
	}
	if err := s.failRun(
		ctx,
		activeRun.ID,
		"run_interrupted",
		"Relay lost the active run before it reached a terminal state. The run was marked as interrupted so you can start another task.",
		nil,
	); err != nil {
		return sqlite.AgentRun{}, false, err
	}
	return sqlite.AgentRun{}, false, nil
}

func (s *Service) CreateSession(ctx context.Context, displayName string) (WorkspaceSnapshot, error) {
	session, err := s.store.CreateSession(ctx, displayName)
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("create session: %w", err)
	}

	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	cfg.LastSessionID = session.ID
	if err := config.Save(s.paths, cfg); err != nil {
		return WorkspaceSnapshot{}, err
	}

	snapshot, err := s.Bootstrap(ctx, session.ID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	snapshot.Warnings = append(snapshot.Warnings, warnings...)
	return snapshot, nil
}

func (s *Service) OpenSession(ctx context.Context, sessionID string) (WorkspaceSnapshot, error) {
	session, err := s.store.OpenSession(ctx, sessionID)
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("open session: %w", err)
	}

	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	cfg.LastSessionID = session.ID
	if err := config.Save(s.paths, cfg); err != nil {
		return WorkspaceSnapshot{}, err
	}

	snapshot, err := s.Bootstrap(ctx, session.ID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	snapshot.Warnings = append(snapshot.Warnings, warnings...)
	return snapshot, nil
}

func (s *Service) SavePreferences(ctx context.Context, input PreferencesInput) (WorkspaceSnapshot, error) {
	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

	if input.PreferredPort != nil {
		if *input.PreferredPort < 1 || *input.PreferredPort > 65535 {
			warnings = append(warnings, "Ignored invalid preferred port and kept the existing saved value.")
		} else {
			cfg.Port = *input.PreferredPort
		}
	}

	if input.AppearanceVariant != nil {
		variant := strings.TrimSpace(*input.AppearanceVariant)
		if variant == "" {
			warnings = append(warnings, "Ignored invalid appearance variant.")
		} else if variant != "midnight" && variant != "graphite" {
			warnings = append(warnings, "Ignored unsupported appearance variant and kept the existing dark-mode setting.")
		} else {
			cfg.AppearanceVariant = variant
		}
	}

	if input.OpenBrowserOnStart != nil {
		cfg.OpenBrowserOnStart = *input.OpenBrowserOnStart
	}

	if input.ProjectRoot != nil {
		cfg.ProjectRoot = strings.TrimSpace(*input.ProjectRoot)
	}

	if input.OpenRouterAPIKey != nil {
		cfg.OpenRouter.APIKey = strings.TrimSpace(*input.OpenRouterAPIKey)
		if cfg.OpenRouter.APIKey != "" {
			cfg.OpenRouter.UpdatedAt = time.Now().UTC()
		}
	}

	if input.ReplaceCredentials {
		credentials := make([]config.Credential, 0, len(input.Credentials))
		for _, item := range input.Credentials {
			if strings.TrimSpace(item.Provider) == "" || strings.TrimSpace(item.Secret) == "" {
				warnings = append(warnings, "Ignored an incomplete credential entry.")
				continue
			}
			credentials = append(credentials, config.Credential{
				Provider:  strings.TrimSpace(item.Provider),
				Label:     strings.TrimSpace(item.Label),
				Secret:    item.Secret,
				UpdatedAt: time.Now().UTC(),
			})
		}
		cfg.Credentials = credentials
	}

	if err := config.Save(s.paths, cfg); err != nil {
		return WorkspaceSnapshot{}, err
	}

	snapshot, err := s.Bootstrap(ctx, cfg.LastSessionID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	snapshot.Warnings = append(snapshot.Warnings, warnings...)
	snapshot.UIState.SaveState = "saved"
	return snapshot, nil
}

func summarizeRuns(runs []sqlite.RunSummary) []RunSummary {
	items := make([]RunSummary, 0, len(runs))
	for _, run := range runs {
		items = append(items, RunSummary{
			ID:              run.ID,
			TaskTextPreview: run.TaskTextPreview,
			Role:            run.Role,
			Model:           run.Model,
			State:           run.State,
			ErrorCode:       run.ErrorCode,
			StartedAt:       run.StartedAt,
			CompletedAt:     run.CompletedAt,
			HasToolActivity: run.HasToolActivity,
		})
	}

	return items
}

func summarizeSessions(sessions []sqlite.Session) []SessionSummary {
	items := make([]SessionSummary, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, SessionSummary{
			ID:           session.ID,
			DisplayName:  session.DisplayName,
			CreatedAt:    session.CreatedAt,
			LastOpenedAt: session.LastOpenedAt,
			Status:       session.Status,
			HasActivity:  session.Snapshot.HasActivity,
		})
	}

	return items
}

func canvasStateForSessions(sessions []sqlite.Session, activeSessionID string) string {
	if len(sessions) == 0 {
		return "empty"
	}

	for _, session := range sessions {
		if session.ID == activeSessionID {
			if session.Snapshot.HasActivity {
				return "ready"
			}
			return "empty"
		}
	}

	return "ready"
}
