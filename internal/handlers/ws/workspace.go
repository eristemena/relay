package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	workspaceorchestrator "github.com/erisristemena/relay/internal/orchestrator/workspace"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type Service interface {
	Bootstrap(ctx context.Context, lastSessionID string) (workspaceorchestrator.WorkspaceSnapshot, error)
	AttachWorkspaceSubscriber(ctx context.Context, emit func(workspaceorchestrator.StreamEnvelope) error)
	BrowseRepository(ctx context.Context, input workspaceorchestrator.RepositoryBrowseInput) (workspaceorchestrator.RepositoryBrowseResult, error)
	GetRepositoryTree(ctx context.Context, input workspaceorchestrator.RepositoryTreeRequestInput) (workspaceorchestrator.RepositoryTreeResult, error)
	CreateSession(ctx context.Context, displayName string) (workspaceorchestrator.WorkspaceSnapshot, error)
	OpenSession(ctx context.Context, sessionID string) (workspaceorchestrator.WorkspaceSnapshot, error)
	SavePreferences(ctx context.Context, input workspaceorchestrator.PreferencesInput) (workspaceorchestrator.WorkspaceSnapshot, error)
	SubmitRun(ctx context.Context, input workspaceorchestrator.SubmitRunInput, emit func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error)
	OpenRun(ctx context.Context, input workspaceorchestrator.OpenRunInput, emit func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error)
	ReplayControl(ctx context.Context, input workspaceorchestrator.ReplayControlInput, emit func(workspaceorchestrator.StreamEnvelope) error) error
	QueryRunHistory(ctx context.Context, input workspaceorchestrator.RunHistoryQueryInput) ([]workspaceorchestrator.RunSummary, error)
	GetRunHistoryDetails(ctx context.Context, sessionID string, runID string) (workspaceorchestrator.RunHistoryDetails, error)
	ExportRunHistory(ctx context.Context, input workspaceorchestrator.RunHistoryExportRequest) (workspaceorchestrator.RunHistoryExportResult, error)
	CancelRun(ctx context.Context, input workspaceorchestrator.CancelRunInput, emit func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error)
	ResolveApproval(ctx context.Context, input workspaceorchestrator.ApprovalResponseInput) (workspaceorchestrator.WorkspaceSnapshot, error)
}

type RuntimeEvent struct {
	Phase   string
	Message string
}

type RuntimeEventProvider interface {
	RuntimeEvents() []RuntimeEvent
}

type Handler struct {
	service       Service
	runtimeEvents RuntimeEventProvider
	logger        *slog.Logger
}

func NewHandler(service Service, runtimeEvents RuntimeEventProvider, logger *slog.Logger) *Handler {
	return &Handler{service: service, runtimeEvents: runtimeEvents, logger: logger}
}

func (h *Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	connection, err := websocket.Accept(responseWriter, request, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer connection.Close(websocket.StatusNormalClosure, "closing websocket")
	ctx := workspaceorchestrator.WithStreamSubscriber(request.Context(), fmt.Sprintf("conn_%p", connection))
	var writeMu sync.Mutex
	write := func(messageType string, requestID string, payload any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writeJSON(ctx, connection, messageType, requestID, payload)
	}
	h.service.AttachWorkspaceSubscriber(ctx, func(envelope workspaceorchestrator.StreamEnvelope) error {
		return write(envelope.Type, "", envelope.Payload)
	})
	for {
		var envelope Envelope
		if err := wsjson.Read(ctx, connection, &envelope); err != nil {
			if websocket.CloseStatus(err) != -1 && !errors.Is(err, context.Canceled) {
				h.logger.Debug("websocket closed", "error", err)
			}
			return
		}

		if err := h.handleMessage(ctx, envelope, write); err != nil {
			h.logger.Error("workspace websocket error", "type", envelope.Type, "error", err)
			_ = write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "request_failed",
				Message: err.Error(),
			})
		}
	}
}

func (h *Handler) handleMessage(ctx context.Context, envelope Envelope, write func(messageType string, requestID string, payload any) error) error {
	switch envelope.Type {
	case TypeWorkspaceBootstrapRequest:
		var payload BootstrapRequestPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.Bootstrap(ctx, payload.LastSessionID)
		if err != nil {
			return err
		}

		if err := write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		if err := write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot)); err != nil {
			return err
		}

		return h.sendRuntimeEvents(envelope.RequestID, write)
	case TypeRepositoryBrowseRequest:
		var payload RepositoryBrowseRequestPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		result, err := h.service.BrowseRepository(ctx, workspaceorchestrator.RepositoryBrowseInput{
			Path:       payload.Path,
			ShowHidden: payload.ShowHidden,
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "repository_browse_failed",
				Message: err.Error(),
			})
		}

		return write(TypeRepositoryBrowseResult, envelope.RequestID, toRepositoryBrowsePayload(result))
	case TypeRepositoryTreeRequest:
		var payload RepositoryTreeRequestPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		result, err := h.service.GetRepositoryTree(ctx, workspaceorchestrator.RepositoryTreeRequestInput{
			SessionID: payload.SessionID,
			RunID:     payload.RunID,
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "repository_tree_failed",
				Message: err.Error(),
			})
		}

		return write(TypeRepositoryTreeResult, envelope.RequestID, RepositoryTreeResultPayload{
			SessionID:      result.SessionID,
			RunID:          result.RunID,
			RepositoryRoot: result.RepositoryRoot,
			Status:         result.Status,
			Message:        result.Message,
			Paths:          result.Paths,
			TouchedFiles:   toTouchedFilePayloads(result.TouchedFiles),
		})
	case TypeSessionCreate:
		var payload SessionCreatePayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.CreateSession(ctx, payload.DisplayName)
		if err != nil {
			return err
		}

		if err := write(TypeSessionCreated, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		return write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot))
	case TypeSessionOpen:
		var payload SessionOpenPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.OpenSession(ctx, payload.SessionID)
		if err != nil {
			if errors.Is(err, sqlite.ErrSessionNotFound) {
				return write(TypeError, envelope.RequestID, ErrorPayload{
					Code:    "session_not_found",
					Message: "That session is no longer available. Choose another session or start a new one.",
				})
			}
			return err
		}

		if err := write(TypeSessionOpened, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		return write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot))
	case TypePreferencesSave:
		var payload PreferencesSavePayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		if err := write(TypeWorkspaceStatus, envelope.RequestID, WorkspaceStatusPayload{
			Phase:   "preferences-saving",
			Message: "Saving your Relay preferences locally.",
		}); err != nil {
			return err
		}

		input := workspaceorchestrator.PreferencesInput{
			PreferredPort:      payload.PreferredPort,
			AppearanceVariant:  payload.AppearanceVariant,
			ReplaceCredentials: len(payload.Credentials) > 0,
			OpenRouterAPIKey:   payload.OpenRouterAPIKey,
			ProjectRoot:        payload.ProjectRoot,
			OpenBrowserOnStart: payload.OpenBrowserOnStart,
		}
		for _, credential := range payload.Credentials {
			input.Credentials = append(input.Credentials, workspaceorchestrator.CredentialInput{
				Provider: credential.Provider,
				Label:    credential.Label,
				Secret:   credential.Secret,
			})
		}

		snapshot, err := h.service.SavePreferences(ctx, input)
		if err != nil {
			return err
		}

		if err := write(TypePreferencesSaved, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		return write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot))
	case TypeAgentRunSubmit:
		var payload AgentRunSubmitPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.SubmitRun(ctx, workspaceorchestrator.SubmitRunInput{
			SessionID: payload.SessionID,
			Task:      payload.Task,
		}, func(stream workspaceorchestrator.StreamEnvelope) error {
			return write(stream.Type, envelope.RequestID, stream.Payload)
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "agent_run_submit_failed",
				Message: err.Error(),
			})
		}

		if err := write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		return write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot))
	case TypeAgentRunOpen:
		var payload AgentRunOpenPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.OpenRun(ctx, workspaceorchestrator.OpenRunInput{
			SessionID: payload.SessionID,
			RunID:     payload.RunID,
		}, func(stream workspaceorchestrator.StreamEnvelope) error {
			return write(stream.Type, envelope.RequestID, stream.Payload)
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "agent_run_open_failed",
				Message: err.Error(),
			})
		}

		if err := write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		return write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot))
	case TypeRunHistoryQuery:
		var payload RunHistoryQueryPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}
		dateFrom, err := parseOptionalTimestamp(payload.DateFrom)
		if err != nil {
			return err
		}
		dateTo, err := parseOptionalTimestamp(payload.DateTo)
		if err != nil {
			return err
		}
		runs, err := h.service.QueryRunHistory(ctx, workspaceorchestrator.RunHistoryQueryInput{
			SessionID: payload.SessionID,
			Query:     payload.Query,
			FilePath:  payload.FilePath,
			DateFrom:  dateFrom,
			DateTo:    dateTo,
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{Code: "run_history_query_failed", Message: err.Error()})
		}
		return write(TypeRunHistoryResult, envelope.RequestID, RunHistoryResultPayload{
			SessionID: payload.SessionID,
			Query:     payload.Query,
			FilePath:  payload.FilePath,
			DateFrom:  payload.DateFrom,
			DateTo:    payload.DateTo,
			Runs:      summarizeRunPayload(runs),
		})
	case TypeRunHistoryDetailsRequest:
		var payload RunHistoryDetailsRequestPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}
		details, err := h.service.GetRunHistoryDetails(ctx, payload.SessionID, payload.RunID)
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{Code: "run_history_details_failed", Message: err.Error()})
		}
		return write(TypeRunHistoryDetailsResult, envelope.RequestID, toRunHistoryDetailsPayload(details))
	case TypeRunHistoryExportRequest:
		var payload RunHistoryExportRequestPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}
		if err := write(TypeRunHistoryExportResult, envelope.RequestID, RunHistoryExportResultPayload{
			SessionID: payload.SessionID,
			RunID:     payload.RunID,
			Status:    "started",
		}); err != nil {
			return err
		}
		result, err := h.service.ExportRunHistory(ctx, workspaceorchestrator.RunHistoryExportRequest{
			SessionID:   payload.SessionID,
			RunID:       payload.RunID,
			DirectUser:  true,
			RequestedAt: time.Now().UTC(),
		})
		if err != nil {
			_ = write(TypeRunHistoryExportResult, envelope.RequestID, RunHistoryExportResultPayload{
				SessionID: payload.SessionID,
				RunID:     payload.RunID,
				Status:    "error",
			})
			return write(TypeError, envelope.RequestID, ErrorPayload{Code: "run_history_export_failed", Message: err.Error(), RunID: payload.RunID, SessionID: payload.SessionID})
		}
		return write(TypeRunHistoryExportResult, envelope.RequestID, RunHistoryExportResultPayload{
			SessionID:   result.SessionID,
			RunID:       result.RunID,
			Status:      result.Status,
			ExportPath:  result.ExportPath,
			GeneratedAt: result.GeneratedAt.Format(time.RFC3339),
		})
	case TypeAgentRunReplayControl:
		var payload AgentRunReplayControlPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}
		if err := h.service.ReplayControl(ctx, workspaceorchestrator.ReplayControlInput{
			SessionID:  payload.SessionID,
			RunID:      payload.RunID,
			Action:     workspaceorchestrator.ReplayAction(payload.Action),
			CursorMS:   payload.CursorMS,
			Speed:      payload.Speed,
			DirectUser: true,
		}, func(stream workspaceorchestrator.StreamEnvelope) error {
			return write(stream.Type, envelope.RequestID, stream.Payload)
		}); err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{Code: "agent_run_replay_control_failed", Message: err.Error()})
		}
		return nil
	case TypeAgentRunCancel:
		var payload AgentRunCancelPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.CancelRun(ctx, workspaceorchestrator.CancelRunInput{
			SessionID: payload.SessionID,
			RunID:     payload.RunID,
		}, func(stream workspaceorchestrator.StreamEnvelope) error {
			return write(stream.Type, envelope.RequestID, stream.Payload)
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "agent_run_cancel_failed",
				Message: err.Error(),
			})
		}

		if err := write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}
		return write(TypeRepositoryGraphStatus, envelope.RequestID, repositoryGraphStatusPayload(snapshot))
	case TypeAgentRunApprovalRespond:
		var payload AgentRunApprovalRespondPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.ResolveApproval(ctx, workspaceorchestrator.ApprovalResponseInput{
			SessionID:  payload.SessionID,
			RunID:      payload.RunID,
			ToolCallID: payload.ToolCallID,
			Decision:   payload.Decision,
		})
		if err != nil {
			return write(TypeError, envelope.RequestID, ErrorPayload{
				Code:    "agent_run_approval_failed",
				Message: err.Error(),
			})
		}

		return write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot))
	default:
		return write(TypeError, envelope.RequestID, ErrorPayload{
			Code:    "unsupported_message",
			Message: "Relay did not recognize that workspace message type.",
		})
	}
}

func parseOptionalTimestamp(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid timestamp %q", value)
}

func toRepositoryBrowsePayload(result workspaceorchestrator.RepositoryBrowseResult) RepositoryBrowseResultPayload {
	directories := make([]RepositoryDirectoryPayload, 0, len(result.Directories))
	for _, directory := range result.Directories {
		directories = append(directories, RepositoryDirectoryPayload{
			Name:            directory.Name,
			Path:            directory.Path,
			IsGitRepository: directory.IsGitRepository,
		})
	}
	return RepositoryBrowseResultPayload{Path: result.Path, Directories: directories}
}

func toTouchedFilePayloads(items []workspaceorchestrator.TouchedFileSummary) []TouchedFilePayload {
	payloads := make([]TouchedFilePayload, 0, len(items))
	for _, item := range items {
		payloads = append(payloads, TouchedFilePayload{
			RunID:     item.RunID,
			AgentID:   item.AgentID,
			FilePath:  item.FilePath,
			TouchType: item.TouchType,
		})
	}
	return payloads
}

func repositoryGraphStatusPayload(snapshot workspaceorchestrator.WorkspaceSnapshot) RepositoryGraphStatusPayload {
	graph := snapshot.RepositoryGraph
	if strings.TrimSpace(graph.Status) == "" {
		connected := snapshot.ConnectedRepository
		switch connected.Status {
		case "connected":
			return RepositoryGraphStatusPayload{
				RepositoryRoot: connected.Path,
				Status:         "loading",
				Message:        "Building repository graph in the background.",
			}
		case "invalid":
			message := strings.TrimSpace(connected.Message)
			if message == "" {
				message = "Relay could not build the repository graph yet. The rest of the workspace remains available."
			}
			return RepositoryGraphStatusPayload{RepositoryRoot: connected.Path, Status: "error", Message: message}
		default:
			return RepositoryGraphStatusPayload{Status: "idle", Message: "Connect a repository to load the background-built codebase graph."}
		}
	}

	payload := RepositoryGraphStatusPayload{
		RepositoryRoot: graph.RepositoryRoot,
		Status:         graph.Status,
		Message:        graph.Message,
	}
	if graph.Status != "ready" {
		return payload
	}
	payload.Nodes = make([]RepositoryGraphNodePayload, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		payload.Nodes = append(payload.Nodes, RepositoryGraphNodePayload{
			ID:    node.ID,
			Label: node.Label,
			Kind:  node.Kind,
		})
	}
	payload.Edges = make([]RepositoryGraphEdgePayload, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		payload.Edges = append(payload.Edges, RepositoryGraphEdgePayload{
			ID:     edge.ID,
			Source: edge.Source,
			Target: edge.Target,
			Kind:   edge.Kind,
		})
	}
	return payload
}

func (h *Handler) sendRuntimeEvents(requestID string, write func(messageType string, requestID string, payload any) error) error {
	if h.runtimeEvents == nil {
		return nil
	}

	for _, event := range h.runtimeEvents.RuntimeEvents() {
		if strings.TrimSpace(event.Message) == "" {
			continue
		}
		if err := write(TypeWorkspaceStatus, requestID, WorkspaceStatusPayload{
			Phase:   event.Phase,
			Message: event.Message,
		}); err != nil {
			return err
		}
	}

	return nil
}

func decodePayload[T any](raw []byte, target *T) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func writeJSON[T any](ctx context.Context, connection *websocket.Conn, messageType string, requestID string, payload T) error {
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return wsjson.Write(writeCtx, connection, OutboundEnvelope[T]{
		Type:      messageType,
		RequestID: requestID,
		Payload:   payload,
	})
}

func toPayload(snapshot workspaceorchestrator.WorkspaceSnapshot) WorkspaceSnapshotPayload {
	sessions := make([]SessionSummary, 0, len(snapshot.Sessions))
	for _, session := range snapshot.Sessions {
		sessions = append(sessions, SessionSummary{
			ID:           session.ID,
			DisplayName:  session.DisplayName,
			CreatedAt:    session.CreatedAt.Format(time.RFC3339),
			LastOpenedAt: session.LastOpenedAt.Format(time.RFC3339),
			Status:       session.Status,
			HasActivity:  session.HasActivity,
		})
	}

	return WorkspaceSnapshotPayload{
		ActiveSessionID: snapshot.ActiveSessionID,
		Sessions:        sessions,
		Preferences: PreferencesView{
			PreferredPort:         snapshot.Preferences.PreferredPort,
			AppearanceVariant:     snapshot.Preferences.AppearanceVariant,
			HasCredentials:        snapshot.Preferences.HasCredentials,
			OpenRouterConfigured:  snapshot.Preferences.OpenRouterConfigured,
			ProjectRoot:           snapshot.Preferences.ProjectRoot,
			ProjectRootConfigured: snapshot.Preferences.ProjectRootConfigured,
			ProjectRootValid:      snapshot.Preferences.ProjectRootValid,
			ProjectRootMessage:    snapshot.Preferences.ProjectRootMessage,
			AgentModels: AgentModelsView{
				Planner:   snapshot.Preferences.AgentModels.Planner,
				Coder:     snapshot.Preferences.AgentModels.Coder,
				Reviewer:  snapshot.Preferences.AgentModels.Reviewer,
				Tester:    snapshot.Preferences.AgentModels.Tester,
				Explainer: snapshot.Preferences.AgentModels.Explainer,
			},
			OpenBrowserOnStart: snapshot.Preferences.OpenBrowserOnStart,
		},
		ConnectedRepository: ConnectedRepositoryView{
			Path:    snapshot.ConnectedRepository.Path,
			Status:  snapshot.ConnectedRepository.Status,
			Message: snapshot.ConnectedRepository.Message,
		},
		UIState: UIState{
			HistoryState: snapshot.UIState.HistoryState,
			CanvasState:  snapshot.UIState.CanvasState,
			SaveState:    snapshot.UIState.SaveState,
		},
		ActiveRunID:      snapshot.ActiveRunID,
		RunSummaries:     summarizeRunPayload(snapshot.RunSummaries),
		PendingApprovals: summarizeApprovalPayload(snapshot.PendingApprovals),
		CredentialStatus: CredentialStatusView{Configured: snapshot.CredentialStatus.Configured},
		Warnings:         snapshot.Warnings,
	}
}

func summarizeRunPayload(runs []workspaceorchestrator.RunSummary) []AgentRunSummary {
	items := make([]AgentRunSummary, 0, len(runs))
	for _, run := range runs {
		item := AgentRunSummary{
			ID:              run.ID,
			GeneratedTitle:  run.GeneratedTitle,
			TaskTextPreview: run.TaskTextPreview,
			Role:            string(run.Role),
			Model:           run.Model,
			State:           run.State,
			ErrorCode:       run.ErrorCode,
			StartedAt:       run.StartedAt.Format(time.RFC3339),
			HasToolActivity: run.HasToolActivity,
			AgentCount:      run.AgentCount,
			FinalStatus:     run.FinalStatus,
			HasFileChanges:  run.HasFileChanges,
		}
		if run.CompletedAt != nil {
			item.CompletedAt = run.CompletedAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	return items
}

func toRunHistoryDetailsPayload(details workspaceorchestrator.RunHistoryDetails) RunHistoryDetailsResultPayload {
	records := make([]RunChangeRecordPayload, 0, len(details.ChangeRecords))
	for _, record := range details.ChangeRecords {
		records = append(records, RunChangeRecordPayload{
			ToolCallID:      record.ToolCallID,
			Path:            record.Path,
			OriginalContent: record.OriginalContent,
			ProposedContent: record.ProposedContent,
			BaseContentHash: record.BaseContentHash,
			ApprovalState:   record.ApprovalState,
			OccurredAt:      record.OccurredAt.Format(time.RFC3339),
		})
	}
	return RunHistoryDetailsResultPayload{
		SessionID:      details.SessionID,
		RunID:          details.RunID,
		GeneratedTitle: details.GeneratedTitle,
		FinalStatus:    details.FinalStatus,
		AgentCount:     details.AgentCount,
		ChangeRecords:  records,
	}
}

func summarizeApprovalPayload(approvals []workspaceorchestrator.ApprovalSummary) []ApprovalRequestPayload {
	items := make([]ApprovalRequestPayload, 0, len(approvals))
	for _, approval := range approvals {
		items = append(items, ApprovalRequestPayload{
			SessionID:      approval.SessionID,
			RunID:          approval.RunID,
			Role:           string(approval.Role),
			Model:          approval.Model,
			ToolCallID:     approval.ToolCallID,
			ToolName:       approval.ToolName,
			RequestKind:    approval.RequestKind,
			Status:         approval.Status,
			RepositoryRoot: approval.RepositoryRoot,
			InputPreview:   approval.InputPreview,
			DiffPreview:    approval.DiffPreview,
			CommandPreview: approval.CommandPreview,
			Message:        approval.Message,
			OccurredAt:     approval.OccurredAt.Format(time.RFC3339),
		})
	}
	return items
}
