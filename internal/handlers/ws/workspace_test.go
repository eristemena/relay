package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/config"
	workspaceorchestrator "github.com/erisristemena/relay/internal/orchestrator/workspace"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func TestHandlerServeHTTP_BootstrapSendsSnapshotAndRuntimeEvents(t *testing.T) {
	t.Parallel()

	service := &stubService{
		bootstrapSnapshot: sampleWorkspaceSnapshot(),
	}
	handler := NewHandler(service, stubRuntimeEvents{events: []RuntimeEvent{{Phase: "", Message: ""}, {Phase: "boot", Message: "Relay booted"}}}, slog.Default())
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+server.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("websocket.Dial() error = %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	request := Envelope{Type: TypeWorkspaceBootstrapRequest, RequestID: "req_bootstrap", Payload: mustMarshalJSON(t, BootstrapRequestPayload{LastSessionID: "session_alpha"})}
	if err := wsjson.Write(ctx, conn, request); err != nil {
		t.Fatalf("wsjson.Write() error = %v", err)
	}

	var bootstrap OutboundEnvelope[WorkspaceSnapshotPayload]
	if err := wsjson.Read(ctx, conn, &bootstrap); err != nil {
		t.Fatalf("read bootstrap error = %v", err)
	}
	if bootstrap.Type != TypeWorkspaceBootstrap {
		t.Fatalf("bootstrap.Type = %q, want %q", bootstrap.Type, TypeWorkspaceBootstrap)
	}
	if bootstrap.Payload.ActiveSessionID != "session_alpha" {
		t.Fatalf("bootstrap.Payload.ActiveSessionID = %q, want session_alpha", bootstrap.Payload.ActiveSessionID)
	}

	var status OutboundEnvelope[WorkspaceStatusPayload]
	if err := wsjson.Read(ctx, conn, &status); err != nil {
		t.Fatalf("read runtime status error = %v", err)
	}
	if status.Type != TypeWorkspaceStatus {
		t.Fatalf("status.Type = %q, want %q", status.Type, TypeWorkspaceStatus)
	}
	if status.Payload.Message != "Relay booted" {
		t.Fatalf("status.Payload.Message = %q, want Relay booted", status.Payload.Message)
	}
	if service.bootstrapLastSessionID != "session_alpha" {
		t.Fatalf("service.bootstrapLastSessionID = %q, want session_alpha", service.bootstrapLastSessionID)
	}
}

func TestHandlerHandleMessage_RoutesServiceActionsAndMapsErrors(t *testing.T) {
	t.Parallel()

	service := &stubService{
		createSnapshot:      sampleWorkspaceSnapshot(),
		openSnapshot:        sampleWorkspaceSnapshot(),
		preferencesSnapshot: sampleWorkspaceSnapshot(),
		submitSnapshot:      sampleWorkspaceSnapshot(),
		openRunSnapshot:     sampleWorkspaceSnapshot(),
		cancelSnapshot:      sampleWorkspaceSnapshot(),
		approvalSnapshot:    sampleWorkspaceSnapshot(),
	}
	handler := NewHandler(service, nil, slog.Default())

	t.Run("session create", func(t *testing.T) {
		calls := make([]capturedWrite, 0, 1)
		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypeSessionCreate,
			RequestID: "req_create",
			Payload:   mustMarshalJSON(t, SessionCreatePayload{DisplayName: "New session"}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() error = %v", err)
		}
		if service.createDisplayName != "New session" {
			t.Fatalf("service.createDisplayName = %q, want New session", service.createDisplayName)
		}
		assertWriteType(t, calls, TypeSessionCreated)
	})

	t.Run("session open not found", func(t *testing.T) {
		calls := make([]capturedWrite, 0, 1)
		service.openErr = sqlite.ErrSessionNotFound
		defer func() { service.openErr = nil }()

		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypeSessionOpen,
			RequestID: "req_open",
			Payload:   mustMarshalJSON(t, SessionOpenPayload{SessionID: "session_missing"}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() error = %v", err)
		}
		assertWriteType(t, calls, TypeError)
		payload := calls[0].payload.(ErrorPayload)
		if payload.Code != "session_not_found" {
			t.Fatalf("payload.Code = %q, want session_not_found", payload.Code)
		}
	})

	t.Run("preferences save", func(t *testing.T) {
		calls := make([]capturedWrite, 0, 2)
		preferredPort := 4747
		appearance := "midnight"
		openBrowser := true
		projectRoot := "/tmp/project"
		apiKey := "or-test-key"

		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypePreferencesSave,
			RequestID: "req_prefs",
			Payload: mustMarshalJSON(t, PreferencesSavePayload{
				PreferredPort:      &preferredPort,
				AppearanceVariant:  &appearance,
				OpenBrowserOnStart: &openBrowser,
				ProjectRoot:        &projectRoot,
				OpenRouterAPIKey:   &apiKey,
				Credentials:        []CredentialPayload{{Provider: "openrouter", Label: "primary", Secret: "secret"}},
			}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() error = %v", err)
		}
		if service.preferencesInput.ProjectRoot == nil || *service.preferencesInput.ProjectRoot != "/tmp/project" {
			t.Fatalf("preferences project root = %#v, want /tmp/project", service.preferencesInput.ProjectRoot)
		}
		if len(calls) != 2 {
			t.Fatalf("len(calls) = %d, want 2", len(calls))
		}
		if calls[0].messageType != TypeWorkspaceStatus || calls[1].messageType != TypePreferencesSaved {
			t.Fatalf("message types = [%q, %q], want workspace.status/preferences.saved", calls[0].messageType, calls[1].messageType)
		}
	})

	t.Run("submit open cancel approval and unsupported", func(t *testing.T) {
		cases := []struct {
			name        string
			envelope    Envelope
			wantType    string
			configure   func()
			assertAfter func(t *testing.T)
		}{
			{
				name:      "submit failure maps to error payload",
				envelope:  Envelope{Type: TypeAgentRunSubmit, RequestID: "req_submit", Payload: mustMarshalJSON(t, AgentRunSubmitPayload{SessionID: "session_alpha", Task: "Inspect"})},
				wantType:  TypeError,
				configure: func() { service.submitErr = errors.New("submit failed") },
				assertAfter: func(t *testing.T) {
					if service.submitInput.Task != "Inspect" {
						t.Fatalf("service.submitInput.Task = %q, want Inspect", service.submitInput.Task)
					}
					service.submitErr = nil
				},
			},
			{
				name:     "open run success",
				envelope: Envelope{Type: TypeAgentRunOpen, RequestID: "req_open_run", Payload: mustMarshalJSON(t, AgentRunOpenPayload{SessionID: "session_alpha", RunID: "run_1"})},
				wantType: TypeWorkspaceBootstrap,
				assertAfter: func(t *testing.T) {
					if service.openRunInput.RunID != "run_1" {
						t.Fatalf("service.openRunInput.RunID = %q, want run_1", service.openRunInput.RunID)
					}
				},
			},
			{
				name:     "cancel run success",
				envelope: Envelope{Type: TypeAgentRunCancel, RequestID: "req_cancel", Payload: mustMarshalJSON(t, AgentRunCancelPayload{SessionID: "session_alpha", RunID: "run_2"})},
				wantType: TypeWorkspaceBootstrap,
				assertAfter: func(t *testing.T) {
					if service.cancelInput.RunID != "run_2" {
						t.Fatalf("service.cancelInput.RunID = %q, want run_2", service.cancelInput.RunID)
					}
				},
			},
			{
				name:     "approval respond success",
				envelope: Envelope{Type: TypeAgentRunApprovalRespond, RequestID: "req_approval", Payload: mustMarshalJSON(t, AgentRunApprovalRespondPayload{SessionID: "session_alpha", RunID: "run_3", ToolCallID: "tool_1", Decision: "approve"})},
				wantType: TypeWorkspaceBootstrap,
				assertAfter: func(t *testing.T) {
					if service.approvalInput.ToolCallID != "tool_1" || service.approvalInput.Decision != "approve" {
						t.Fatalf("approval input = %#v, want tool_1/approve", service.approvalInput)
					}
				},
			},
			{
				name:     "unsupported message",
				envelope: Envelope{Type: "workspace.unknown", RequestID: "req_unknown"},
				wantType: TypeError,
			},
		}

		for _, testCase := range cases {
			t.Run(testCase.name, func(t *testing.T) {
				if testCase.configure != nil {
					testCase.configure()
				}
				calls := make([]capturedWrite, 0, 1)
				err := handler.handleMessage(context.Background(), testCase.envelope, captureWrites(&calls))
				if err != nil {
					t.Fatalf("handleMessage() error = %v", err)
				}
				assertWriteType(t, calls, testCase.wantType)
				if testCase.assertAfter != nil {
					testCase.assertAfter(t)
				}
			})
		}
	})
}

func TestToPayloadAndSummarizeRunPayload(t *testing.T) {
	t.Parallel()

	snapshot := sampleWorkspaceSnapshot()
	payload := toPayload(snapshot)
	if payload.Preferences.AgentModels.Planner != "anthropic/claude-opus-4" {
		t.Fatalf("payload.Preferences.AgentModels.Planner = %q, want anthropic/claude-opus-4", payload.Preferences.AgentModels.Planner)
	}
	if len(payload.RunSummaries) != 1 {
		t.Fatalf("len(payload.RunSummaries) = %d, want 1", len(payload.RunSummaries))
	}
	if payload.RunSummaries[0].CompletedAt == "" {
		t.Fatal("payload.RunSummaries[0].CompletedAt = empty, want populated timestamp")
	}
}

type stubService struct {
	bootstrapSnapshot      workspaceorchestrator.WorkspaceSnapshot
	createSnapshot         workspaceorchestrator.WorkspaceSnapshot
	openSnapshot           workspaceorchestrator.WorkspaceSnapshot
	preferencesSnapshot    workspaceorchestrator.WorkspaceSnapshot
	submitSnapshot         workspaceorchestrator.WorkspaceSnapshot
	openRunSnapshot        workspaceorchestrator.WorkspaceSnapshot
	cancelSnapshot         workspaceorchestrator.WorkspaceSnapshot
	approvalSnapshot       workspaceorchestrator.WorkspaceSnapshot
	bootstrapLastSessionID string
	createDisplayName      string
	openSessionID          string
	preferencesInput       workspaceorchestrator.PreferencesInput
	submitInput            workspaceorchestrator.SubmitRunInput
	openRunInput           workspaceorchestrator.OpenRunInput
	cancelInput            workspaceorchestrator.CancelRunInput
	approvalInput          workspaceorchestrator.ApprovalResponseInput
	openErr                error
	submitErr              error
}

func (s *stubService) Bootstrap(_ context.Context, lastSessionID string) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.bootstrapLastSessionID = lastSessionID
	return s.bootstrapSnapshot, nil
}

func (s *stubService) CreateSession(_ context.Context, displayName string) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.createDisplayName = displayName
	return s.createSnapshot, nil
}

func (s *stubService) OpenSession(_ context.Context, sessionID string) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.openSessionID = sessionID
	if s.openErr != nil {
		return workspaceorchestrator.WorkspaceSnapshot{}, s.openErr
	}
	return s.openSnapshot, nil
}

func (s *stubService) SavePreferences(_ context.Context, input workspaceorchestrator.PreferencesInput) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.preferencesInput = input
	return s.preferencesSnapshot, nil
}

func (s *stubService) SubmitRun(_ context.Context, input workspaceorchestrator.SubmitRunInput, _ func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.submitInput = input
	if s.submitErr != nil {
		return workspaceorchestrator.WorkspaceSnapshot{}, s.submitErr
	}
	return s.submitSnapshot, nil
}

func (s *stubService) OpenRun(_ context.Context, input workspaceorchestrator.OpenRunInput, _ func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.openRunInput = input
	return s.openRunSnapshot, nil
}

func (s *stubService) CancelRun(_ context.Context, input workspaceorchestrator.CancelRunInput, _ func(workspaceorchestrator.StreamEnvelope) error) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.cancelInput = input
	return s.cancelSnapshot, nil
}

func (s *stubService) ResolveApproval(_ context.Context, input workspaceorchestrator.ApprovalResponseInput) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.approvalInput = input
	return s.approvalSnapshot, nil
}

type stubRuntimeEvents struct {
	events []RuntimeEvent
}

func (s stubRuntimeEvents) RuntimeEvents() []RuntimeEvent {
	return s.events
}

type capturedWrite struct {
	messageType string
	requestID   string
	payload     any
}

func captureWrites(calls *[]capturedWrite) func(string, string, any) error {
	return func(messageType string, requestID string, payload any) error {
		*calls = append(*calls, capturedWrite{messageType: messageType, requestID: requestID, payload: payload})
		return nil
	}
}

func assertWriteType(t *testing.T, calls []capturedWrite, wantType string) {
	t.Helper()
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}
	if calls[0].messageType != wantType {
		t.Fatalf("calls[0].messageType = %q, want %q", calls[0].messageType, wantType)
	}
}

func mustMarshalJSON(t *testing.T, payload any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return encoded
}

func sampleWorkspaceSnapshot() workspaceorchestrator.WorkspaceSnapshot {
	startedAt := time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC)
	completedAt := startedAt.Add(3 * time.Second)
	return workspaceorchestrator.WorkspaceSnapshot{
		ActiveSessionID: "session_alpha",
		Sessions: []workspaceorchestrator.SessionSummary{{
			ID:           "session_alpha",
			DisplayName:  "Relay",
			CreatedAt:    startedAt,
			LastOpenedAt: startedAt,
			Status:       sqlite.StatusActive,
			HasActivity:  true,
		}},
		Preferences: samplePreferences(),
		UIState:     workspaceorchestrator.UIState{HistoryState: "ready", CanvasState: "idle", SaveState: "idle"},
		ActiveRunID: "run_1",
		RunSummaries: []workspaceorchestrator.RunSummary{{
			ID:              "run_1",
			TaskTextPreview: "Inspect orchestration status",
			Role:            sqlite.RolePlanner,
			Model:           "anthropic/claude-opus-4",
			State:           sqlite.RunStateCompleted,
			ErrorCode:       "",
			StartedAt:       startedAt,
			CompletedAt:     &completedAt,
			HasToolActivity: false,
		}},
		CredentialStatus: workspaceorchestrator.CredentialStatus{Configured: true},
	}
}

func samplePreferences() config.SafePreferences {
	return config.SafePreferences{
		PreferredPort:         4747,
		AppearanceVariant:     "midnight",
		HasCredentials:        true,
		OpenRouterConfigured:  true,
		ProjectRoot:           "/tmp/project",
		ProjectRootConfigured: true,
		ProjectRootValid:      true,
		AgentModels: config.AgentModels{
			Planner:   "anthropic/claude-opus-4",
			Coder:     "anthropic/claude-sonnet-4-5",
			Reviewer:  "anthropic/claude-sonnet-4-5",
			Tester:    "deepseek/deepseek-chat",
			Explainer: "google/gemini-2.0-flash-001",
		},
		OpenBrowserOnStart: true,
	}
}
