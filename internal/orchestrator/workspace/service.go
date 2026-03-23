package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

type Store interface {
	ListSessions(ctx context.Context) ([]sqlite.Session, error)
	GetSession(ctx context.Context, sessionID string) (sqlite.Session, error)
	CreateSession(ctx context.Context, displayName string) (sqlite.Session, error)
	OpenSession(ctx context.Context, sessionID string) (sqlite.Session, error)
	CreateAgentRun(ctx context.Context, sessionID string, taskText string, role sqlite.AgentRole, model string) (sqlite.AgentRun, error)
	GetActiveRun(ctx context.Context) (sqlite.AgentRun, error)
	GetAgentRun(ctx context.Context, runID string) (sqlite.AgentRun, error)
	ListRunSummaries(ctx context.Context, sessionID string) ([]sqlite.RunSummary, error)
	AppendRunEvent(ctx context.Context, runID string, eventType string, role sqlite.AgentRole, model string, payloadJSON string) (sqlite.AgentRunEvent, error)
	ListRunEvents(ctx context.Context, runID string) ([]sqlite.AgentRunEvent, error)
	UpdateAgentRun(ctx context.Context, run sqlite.AgentRun) error
}

type Service struct {
	store Store
	paths config.Paths
	mu    sync.Mutex
	runnerFactory func(cfg config.Config, task string) agents.Runner
	activeRuns    map[string]activeRunRegistration
	pendingApprovals map[string]pendingApproval
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
	SessionID  string
	RunID      string
	ToolCallID string
	Respond    chan ApprovalDecision
	Payload    map[string]any
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
	ActiveSessionID string                 `json:"active_session_id"`
	Sessions        []SessionSummary       `json:"sessions"`
	Preferences     config.SafePreferences `json:"preferences"`
	UIState         UIState                `json:"ui_state"`
	ActiveRunID     string                 `json:"active_run_id,omitempty"`
	RunSummaries    []RunSummary           `json:"run_summaries,omitempty"`
	CredentialStatus CredentialStatus      `json:"credential_status"`
	Warnings        []string               `json:"warnings,omitempty"`
}

type CredentialStatus struct {
	Configured bool `json:"configured"`
}

type RunSummary struct {
	ID              string             `json:"id"`
	TaskTextPreview string             `json:"task_text_preview"`
	Role            sqlite.AgentRole   `json:"role"`
	Model           string             `json:"model"`
	State           string             `json:"state"`
	StartedAt       time.Time          `json:"started_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	HasToolActivity bool               `json:"has_tool_activity"`
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
		store:      store,
		paths:      paths,
		activeRuns: make(map[string]activeRunRegistration),
		pendingApprovals: make(map[string]pendingApproval),
	}
	service.runnerFactory = func(cfg config.Config, task string) agents.Runner {
		registry := agents.NewRegistry(cfg.Agents.WithDefaults())
		return registry.NewRunner(cfg.OpenRouter.APIKey, task, newCatalogToolExecutor(cfg.ProjectRoot, service))
	}
	return service
}

func (s *Service) SetRunnerFactory(factory func(cfg config.Config, task string) agents.Runner) {
	if factory == nil {
		return
	}
	s.runnerFactory = factory
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

func (s *Service) pendingApprovalPayload(runID string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, pending := range s.pendingApprovals {
		if pending.RunID == runID {
			payload := make(map[string]any, len(pending.Payload))
			for key, value := range pending.Payload {
				payload[key] = value
			}
			return payload
		}
	}
	return nil
}

func approvalKey(runID string, toolCallID string) string {
	return strings.TrimSpace(runID) + ":" + strings.TrimSpace(toolCallID)
}

func (s *Service) RequestApproval(ctx context.Context, request ApprovalRequest) (ApprovalDecision, error) {
	runContext, ok := runExecutionContextFromContext(ctx)
	if !ok {
		return ApprovalDecision{}, fmt.Errorf("approval flow is unavailable for this run")
	}

	key := approvalKey(request.RunID, request.ToolCallID)
	pending := pendingApproval{
		SessionID:  request.SessionID,
		RunID:      request.RunID,
		ToolCallID: request.ToolCallID,
		Respond:    make(chan ApprovalDecision, 1),
		Payload: map[string]any{
			"session_id":    request.SessionID,
			"run_id":        request.RunID,
			"role":          request.Role,
			"model":         request.Model,
			"tool_call_id":  request.ToolCallID,
			"tool_name":     string(request.ToolName),
			"input_preview": request.InputPreview,
			"message":       request.Message,
			"occurred_at":   request.OccurredAt.UTC().Format(time.RFC3339),
		},
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
		Type: "approval_request",
		Payload: pending.Payload,
	}); err != nil {
		return ApprovalDecision{}, err
	}

	select {
	case decision := <-pending.Respond:
		return decision, nil
	case <-ctx.Done():
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
		return WorkspaceSnapshot{}, fmt.Errorf("approval request %s is no longer pending", input.ToolCallID)
	}

	select {
	case pending.Respond <- ApprovalDecision{Approved: decision == "approved"}:
	default:
		return WorkspaceSnapshot{}, fmt.Errorf("approval request %s could not accept another decision", input.ToolCallID)
	}

	return s.Bootstrap(ctx, input.SessionID)
}

func (s *Service) Bootstrap(ctx context.Context, lastSessionID string) (WorkspaceSnapshot, error) {
	cfg, warnings, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

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
	if activeSessionID != "" {
		storedRuns, err := s.store.ListRunSummaries(ctx, activeSessionID)
		if err != nil {
			return WorkspaceSnapshot{}, fmt.Errorf("load run summaries: %w", err)
		}
		runSummaries = summarizeRuns(storedRuns)
		activeRun, err := s.store.GetActiveRun(ctx)
		if err == nil && activeRun.SessionID == activeSessionID {
			activeRunID = activeRun.ID
		}
	}

	return WorkspaceSnapshot{
		ActiveSessionID: activeSessionID,
		Sessions:        summarizeSessions(sessions),
		Preferences:     cfg.SafePreferences(),
		UIState: UIState{
			HistoryState: "ready",
			CanvasState:  canvasStateForSessions(sessions, activeSessionID),
			SaveState:    "idle",
		},
		ActiveRunID:      activeRunID,
		RunSummaries:     runSummaries,
		CredentialStatus: CredentialStatus{Configured: cfg.HasOpenRouterKey()},
		Warnings: warnings,
	}, nil
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
