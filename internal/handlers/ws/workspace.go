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
	CreateSession(ctx context.Context, displayName string) (workspaceorchestrator.WorkspaceSnapshot, error)
	OpenSession(ctx context.Context, sessionID string) (workspaceorchestrator.WorkspaceSnapshot, error)
	SavePreferences(ctx context.Context, input workspaceorchestrator.PreferencesInput) (workspaceorchestrator.WorkspaceSnapshot, error)
	SubmitRun(ctx context.Context, input workspaceorchestrator.SubmitRunInput, emit func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error)
	OpenRun(ctx context.Context, input workspaceorchestrator.OpenRunInput, emit func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error)
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

		return h.sendRuntimeEvents(envelope.RequestID, write)
	case TypeSessionCreate:
		var payload SessionCreatePayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.CreateSession(ctx, payload.DisplayName)
		if err != nil {
			return err
		}

		return write(TypeSessionCreated, envelope.RequestID, toPayload(snapshot))
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

		return write(TypeSessionOpened, envelope.RequestID, toPayload(snapshot))
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

		return write(TypePreferencesSaved, envelope.RequestID, toPayload(snapshot))
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

		return write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot))
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

		return write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot))
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

		return write(TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot))
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
			PreferredPort:        snapshot.Preferences.PreferredPort,
			AppearanceVariant:    snapshot.Preferences.AppearanceVariant,
			HasCredentials:       snapshot.Preferences.HasCredentials,
			OpenRouterConfigured: snapshot.Preferences.OpenRouterConfigured,
			ProjectRoot:          snapshot.Preferences.ProjectRoot,
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
		UIState: UIState{
			HistoryState: snapshot.UIState.HistoryState,
			CanvasState:  snapshot.UIState.CanvasState,
			SaveState:    snapshot.UIState.SaveState,
		},
		ActiveRunID: snapshot.ActiveRunID,
		RunSummaries: summarizeRunPayload(snapshot.RunSummaries),
		CredentialStatus: CredentialStatusView{Configured: snapshot.CredentialStatus.Configured},
		Warnings: snapshot.Warnings,
	}
}

func summarizeRunPayload(runs []workspaceorchestrator.RunSummary) []AgentRunSummary {
	items := make([]AgentRunSummary, 0, len(runs))
	for _, run := range runs {
		item := AgentRunSummary{
			ID:              run.ID,
			TaskTextPreview: run.TaskTextPreview,
			Role:            string(run.Role),
			Model:           run.Model,
			State:           run.State,
			StartedAt:       run.StartedAt.Format(time.RFC3339),
			HasToolActivity: run.HasToolActivity,
		}
		if run.CompletedAt != nil {
			item.CompletedAt = run.CompletedAt.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	return items
}
