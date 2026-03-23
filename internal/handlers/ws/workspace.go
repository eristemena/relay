package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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

	ctx := request.Context()
	for {
		var envelope Envelope
		if err := wsjson.Read(ctx, connection, &envelope); err != nil {
			if websocket.CloseStatus(err) != -1 && !errors.Is(err, context.Canceled) {
				h.logger.Debug("websocket closed", "error", err)
			}
			return
		}

		if err := h.handleMessage(ctx, connection, envelope); err != nil {
			h.logger.Error("workspace websocket error", "type", envelope.Type, "error", err)
			_ = writeJSON(ctx, connection, TypeError, envelope.RequestID, ErrorPayload{
				Code:    "request_failed",
				Message: err.Error(),
			})
		}
	}
}

func (h *Handler) handleMessage(ctx context.Context, connection *websocket.Conn, envelope Envelope) error {
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

		if err := writeJSON(ctx, connection, TypeWorkspaceBootstrap, envelope.RequestID, toPayload(snapshot)); err != nil {
			return err
		}

		return h.sendRuntimeEvents(ctx, connection)
	case TypeSessionCreate:
		var payload SessionCreatePayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.CreateSession(ctx, payload.DisplayName)
		if err != nil {
			return err
		}

		return writeJSON(ctx, connection, TypeSessionCreated, envelope.RequestID, toPayload(snapshot))
	case TypeSessionOpen:
		var payload SessionOpenPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		snapshot, err := h.service.OpenSession(ctx, payload.SessionID)
		if err != nil {
			if errors.Is(err, sqlite.ErrSessionNotFound) {
				return writeJSON(ctx, connection, TypeError, envelope.RequestID, ErrorPayload{
					Code:    "session_not_found",
					Message: "That session is no longer available. Choose another session or start a new one.",
				})
			}
			return err
		}

		return writeJSON(ctx, connection, TypeSessionOpened, envelope.RequestID, toPayload(snapshot))
	case TypePreferencesSave:
		var payload PreferencesSavePayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return err
		}

		if err := writeJSON(ctx, connection, TypeWorkspaceStatus, envelope.RequestID, WorkspaceStatusPayload{
			Phase:   "preferences-saving",
			Message: "Saving your Relay preferences locally.",
		}); err != nil {
			return err
		}

		input := workspaceorchestrator.PreferencesInput{
			PreferredPort:      payload.PreferredPort,
			AppearanceVariant:  payload.AppearanceVariant,
			ReplaceCredentials: len(payload.Credentials) > 0,
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

		return writeJSON(ctx, connection, TypePreferencesSaved, envelope.RequestID, toPayload(snapshot))
	default:
		return writeJSON(ctx, connection, TypeError, envelope.RequestID, ErrorPayload{
			Code:    "unsupported_message",
			Message: "Relay did not recognize that workspace message type.",
		})
	}
}

func (h *Handler) sendRuntimeEvents(ctx context.Context, connection *websocket.Conn) error {
	if h.runtimeEvents == nil {
		return nil
	}

	for _, event := range h.runtimeEvents.RuntimeEvents() {
		if strings.TrimSpace(event.Message) == "" {
			continue
		}
		if err := writeJSON(ctx, connection, TypeWorkspaceStatus, "", WorkspaceStatusPayload{
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
			PreferredPort:      snapshot.Preferences.PreferredPort,
			AppearanceVariant:  snapshot.Preferences.AppearanceVariant,
			HasCredentials:     snapshot.Preferences.HasCredentials,
			OpenBrowserOnStart: snapshot.Preferences.OpenBrowserOnStart,
		},
		UIState: UIState{
			HistoryState: snapshot.UIState.HistoryState,
			CanvasState:  snapshot.UIState.CanvasState,
			SaveState:    snapshot.UIState.SaveState,
		},
		Warnings: snapshot.Warnings,
	}
}
