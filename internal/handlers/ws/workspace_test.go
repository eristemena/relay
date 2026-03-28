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
	if bootstrap.Payload.ConnectedRepository.Status != "connected" {
		t.Fatalf("bootstrap.Payload.ConnectedRepository.Status = %q, want connected", bootstrap.Payload.ConnectedRepository.Status)
	}

	var graphStatus OutboundEnvelope[RepositoryGraphStatusPayload]
	if err := wsjson.Read(ctx, conn, &graphStatus); err != nil {
		t.Fatalf("read repository graph status error = %v", err)
	}
	if graphStatus.Type != TypeRepositoryGraphStatus {
		t.Fatalf("graphStatus.Type = %q, want %q", graphStatus.Type, TypeRepositoryGraphStatus)
	}
	if graphStatus.Payload.Status != "loading" {
		t.Fatalf("graphStatus.Payload.Status = %q, want loading", graphStatus.Payload.Status)
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
		repositoryBrowseResult: workspaceorchestrator.RepositoryBrowseResult{
			Path: "/tmp/repos",
			Directories: []workspaceorchestrator.RepositoryDirectory{{
				Name:            "relay",
				Path:            "/tmp/repos/relay",
				IsGitRepository: true,
			}},
		},
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
		assertWriteTypes(t, calls, TypeSessionCreated, TypeRepositoryGraphStatus)
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
		if len(calls) != 3 {
			t.Fatalf("len(calls) = %d, want 3", len(calls))
		}
		if calls[0].messageType != TypeWorkspaceStatus || calls[1].messageType != TypePreferencesSaved || calls[2].messageType != TypeRepositoryGraphStatus {
			t.Fatalf("message types = [%q, %q, %q], want workspace.status/preferences.saved/repository_graph_status", calls[0].messageType, calls[1].messageType, calls[2].messageType)
		}
	})

	t.Run("submit open cancel approval and unsupported", func(t *testing.T) {
		cases := []struct {
			name        string
			envelope    Envelope
			wantTypes   []string
			configure   func()
			assertAfter func(t *testing.T)
		}{
			{
				name:      "submit failure maps to error payload",
				envelope:  Envelope{Type: TypeAgentRunSubmit, RequestID: "req_submit", Payload: mustMarshalJSON(t, AgentRunSubmitPayload{SessionID: "session_alpha", Task: "Inspect"})},
				wantTypes: []string{TypeError},
				configure: func() { service.submitErr = errors.New("submit failed") },
				assertAfter: func(t *testing.T) {
					if service.submitInput.Task != "Inspect" {
						t.Fatalf("service.submitInput.Task = %q, want Inspect", service.submitInput.Task)
					}
					service.submitErr = nil
				},
			},
			{
				name:      "open run success",
				envelope:  Envelope{Type: TypeAgentRunOpen, RequestID: "req_open_run", Payload: mustMarshalJSON(t, AgentRunOpenPayload{SessionID: "session_alpha", RunID: "run_1"})},
				wantTypes: []string{TypeWorkspaceBootstrap, TypeRepositoryGraphStatus},
				assertAfter: func(t *testing.T) {
					if service.openRunInput.RunID != "run_1" {
						t.Fatalf("service.openRunInput.RunID = %q, want run_1", service.openRunInput.RunID)
					}
				},
			},
			{
				name:      "cancel run success",
				envelope:  Envelope{Type: TypeAgentRunCancel, RequestID: "req_cancel", Payload: mustMarshalJSON(t, AgentRunCancelPayload{SessionID: "session_alpha", RunID: "run_2"})},
				wantTypes: []string{TypeWorkspaceBootstrap, TypeRepositoryGraphStatus},
				assertAfter: func(t *testing.T) {
					if service.cancelInput.RunID != "run_2" {
						t.Fatalf("service.cancelInput.RunID = %q, want run_2", service.cancelInput.RunID)
					}
				},
			},
			{
				name:      "approval respond success",
				envelope:  Envelope{Type: TypeAgentRunApprovalRespond, RequestID: "req_approval", Payload: mustMarshalJSON(t, AgentRunApprovalRespondPayload{SessionID: "session_alpha", RunID: "run_3", ToolCallID: "tool_1", Decision: "approve"})},
				wantTypes: []string{TypeWorkspaceBootstrap},
				assertAfter: func(t *testing.T) {
					if service.approvalInput.ToolCallID != "tool_1" || service.approvalInput.Decision != "approve" {
						t.Fatalf("approval input = %#v, want tool_1/approve", service.approvalInput)
					}
				},
			},
			{
				name:      "unsupported message",
				envelope:  Envelope{Type: "workspace.unknown", RequestID: "req_unknown"},
				wantTypes: []string{TypeError},
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
				assertWriteTypes(t, calls, testCase.wantTypes...)
				if testCase.assertAfter != nil {
					testCase.assertAfter(t)
				}
			})
		}
	})

	t.Run("repository browse success and failure", func(t *testing.T) {
		calls := make([]capturedWrite, 0, 1)
		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypeRepositoryBrowseRequest,
			RequestID: "req_browse",
			Payload:   mustMarshalJSON(t, RepositoryBrowseRequestPayload{Path: "/tmp/repos", ShowHidden: true}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() browse error = %v", err)
		}
		assertWriteType(t, calls, TypeRepositoryBrowseResult)
		payload := calls[0].payload.(RepositoryBrowseResultPayload)
		if payload.Path != "/tmp/repos" || len(payload.Directories) != 1 || !payload.Directories[0].IsGitRepository {
			t.Fatalf("browse payload = %#v, want repo browse result", payload)
		}

		service.browseErr = errors.New("browse failed")
		defer func() { service.browseErr = nil }()
		calls = make([]capturedWrite, 0, 1)
		err = handler.handleMessage(context.Background(), Envelope{
			Type:      TypeRepositoryBrowseRequest,
			RequestID: "req_browse_fail",
			Payload:   mustMarshalJSON(t, RepositoryBrowseRequestPayload{Path: "/tmp/missing"}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() browse failure error = %v", err)
		}
		assertWriteType(t, calls, TypeError)
		if calls[0].payload.(ErrorPayload).Code != "repository_browse_failed" {
			t.Fatalf("browse failure payload = %#v, want repository_browse_failed", calls[0].payload)
		}
	})

	t.Run("run history query and details", func(t *testing.T) {
		service.runHistoryRuns = []workspaceorchestrator.RunSummary{{
			ID:              "run_history_1",
			GeneratedTitle:  "Review approval flow",
			TaskTextPreview: "Audit approval review flow",
			Role:            sqlite.RoleReviewer,
			Model:           "anthropic/claude-sonnet-4-5",
			State:           sqlite.RunStateCompleted,
			StartedAt:       time.Date(2026, time.March, 24, 12, 0, 0, 0, time.UTC),
			HasToolActivity: true,
			AgentCount:      3,
			FinalStatus:     "completed",
			HasFileChanges:  true,
		}}
		service.runHistoryDetails = workspaceorchestrator.RunHistoryDetails{
			SessionID:      "session_alpha",
			RunID:          "run_history_1",
			GeneratedTitle: "Review approval flow",
			FinalStatus:    "completed",
			AgentCount:     3,
			ChangeRecords: []workspaceorchestrator.RunChangeRecord{{
				ToolCallID:      "call_1",
				Path:            "README.md",
				OriginalContent: "before\n",
				ProposedContent: "after\n",
				BaseContentHash: "sha256:abc",
				ApprovalState:   sqlite.ApprovalStateApplied,
				OccurredAt:      time.Date(2026, time.March, 24, 12, 1, 0, 0, time.UTC),
			}},
		}

		queryCalls := make([]capturedWrite, 0, 1)
		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypeRunHistoryQuery,
			RequestID: "req_history_query",
			Payload:   mustMarshalJSON(t, RunHistoryQueryPayload{SessionID: "session_alpha", Query: "approval", FilePath: "README.md", DateFrom: "2026-03-24"}),
		}, captureWrites(&queryCalls))
		if err != nil {
			t.Fatalf("handleMessage() history query error = %v", err)
		}
		assertWriteType(t, queryCalls, TypeRunHistoryResult)
		queryPayload := queryCalls[0].payload.(RunHistoryResultPayload)
		if len(queryPayload.Runs) != 1 || queryPayload.Runs[0].GeneratedTitle != "Review approval flow" {
			t.Fatalf("queryPayload = %#v, want generated title in run history result", queryPayload)
		}
		if service.runHistoryQueryInput.Query != "approval" || service.runHistoryQueryInput.FilePath != "README.md" {
			t.Fatalf("runHistoryQueryInput = %#v, want approval + README.md", service.runHistoryQueryInput)
		}

		detailCalls := make([]capturedWrite, 0, 1)
		err = handler.handleMessage(context.Background(), Envelope{
			Type:      TypeRunHistoryDetailsRequest,
			RequestID: "req_history_details",
			Payload:   mustMarshalJSON(t, RunHistoryDetailsRequestPayload{SessionID: "session_alpha", RunID: "run_history_1"}),
		}, captureWrites(&detailCalls))
		if err != nil {
			t.Fatalf("handleMessage() history details error = %v", err)
		}
		assertWriteType(t, detailCalls, TypeRunHistoryDetailsResult)
		detailsPayload := detailCalls[0].payload.(RunHistoryDetailsResultPayload)
		if len(detailsPayload.ChangeRecords) != 1 || detailsPayload.ChangeRecords[0].Path != "README.md" {
			t.Fatalf("detailsPayload = %#v, want one README.md change record", detailsPayload)
		}
		if service.runHistoryDetailsSessionID != "session_alpha" || service.runHistoryDetailsRunID != "run_history_1" {
			t.Fatalf("history details input = %q/%q, want session_alpha/run_history_1", service.runHistoryDetailsSessionID, service.runHistoryDetailsRunID)
		}
	})

	t.Run("run history export", func(t *testing.T) {
		service.runHistoryExportResult = workspaceorchestrator.RunHistoryExportResult{
			SessionID:   "session_alpha",
			RunID:       "run_history_1",
			Status:      "completed",
			ExportPath:  "/Users/example/.relay/exports/review-approval-flow.md",
			GeneratedAt: time.Date(2026, time.March, 24, 12, 3, 0, 0, time.UTC),
		}
		calls := make([]capturedWrite, 0, 3)
		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypeRunHistoryExportRequest,
			RequestID: "req_history_export",
			Payload:   mustMarshalJSON(t, RunHistoryExportRequestPayload{SessionID: "session_alpha", RunID: "run_history_1"}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() history export error = %v", err)
		}
		assertWriteTypes(t, calls, TypeRunHistoryExportResult, TypeRunHistoryExportResult)
		startedPayload := calls[0].payload.(RunHistoryExportResultPayload)
		if startedPayload.Status != "started" {
			t.Fatalf("startedPayload.Status = %q, want started", startedPayload.Status)
		}
		completedPayload := calls[1].payload.(RunHistoryExportResultPayload)
		if completedPayload.Status != "completed" || completedPayload.ExportPath == "" {
			t.Fatalf("completedPayload = %#v, want completed export result", completedPayload)
		}
		if service.runHistoryExportInput.RunID != "run_history_1" || !service.runHistoryExportInput.DirectUser {
			t.Fatalf("runHistoryExportInput = %#v, want direct export request for run_history_1", service.runHistoryExportInput)
		}
	})

	t.Run("run history replay control", func(t *testing.T) {
		calls := make([]capturedWrite, 0, 1)
		err := handler.handleMessage(context.Background(), Envelope{
			Type:      TypeAgentRunReplayControl,
			RequestID: "req_replay",
			Payload: mustMarshalJSON(t, AgentRunReplayControlPayload{
				SessionID: "session_alpha",
				RunID:     "run_history_1",
				Action:    "seek",
				CursorMS:  2500,
				Speed:     1,
			}),
		}, captureWrites(&calls))
		if err != nil {
			t.Fatalf("handleMessage() replay control error = %v", err)
		}
		if service.replayControlInput.RunID != "run_history_1" || service.replayControlInput.CursorMS != 2500 {
			t.Fatalf("replayControlInput = %#v, want run_history_1 cursor 2500", service.replayControlInput)
		}
		if service.replayControlInput.Action != workspaceorchestrator.ReplayActionSeek || !service.replayControlInput.DirectUser {
			t.Fatalf("replayControlInput = %#v, want seek direct-user request", service.replayControlInput)
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
	if len(payload.PendingApprovals) != 1 || payload.PendingApprovals[0].ToolCallID != "call_1" {
		t.Fatalf("payload.PendingApprovals = %#v, want one persisted approval", payload.PendingApprovals)
	}
	if payload.ConnectedRepository.Path != "/tmp/project" {
		t.Fatalf("payload.ConnectedRepository.Path = %q, want /tmp/project", payload.ConnectedRepository.Path)
	}
}

func TestRepositoryGraphStatusPayloadIncludesReadyGraphData(t *testing.T) {
	t.Parallel()

	payload := repositoryGraphStatusPayload(workspaceorchestrator.WorkspaceSnapshot{
		ConnectedRepository: workspaceorchestrator.ConnectedRepositorySummary{
			Path:   "/tmp/project",
			Status: "connected",
		},
		RepositoryGraph: workspaceorchestrator.RepositoryGraphState{
			RepositoryRoot: "/tmp/project",
			Status:         "ready",
			Message:        "Repository graph ready.",
			Nodes: []workspaceorchestrator.RepositoryGraphNode{
				{ID: "src/index.ts", Label: "src/index.ts", Kind: "file"},
			},
			Edges: []workspaceorchestrator.RepositoryGraphEdge{
				{ID: "src/index.ts->src/lib/util.ts", Source: "src/index.ts", Target: "src/lib/util.ts", Kind: "import"},
			},
		},
	})

	if payload.Status != "ready" {
		t.Fatalf("payload.Status = %q, want ready", payload.Status)
	}
	if len(payload.Nodes) != 1 || payload.Nodes[0].ID != "src/index.ts" {
		t.Fatalf("payload.Nodes = %#v, want ready graph nodes", payload.Nodes)
	}
	if len(payload.Edges) != 1 || payload.Edges[0].Target != "src/lib/util.ts" {
		t.Fatalf("payload.Edges = %#v, want ready graph edges", payload.Edges)
	}
}

func TestHandlerHandleMessage_RepositoryBrowseResponses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		service       *stubService
		request       RepositoryBrowseRequestPayload
		wantType      string
		wantErrorCode string
		assertPayload func(t *testing.T, payload RepositoryBrowseResultPayload)
		assertService func(t *testing.T, service *stubService)
	}{
		{
			name: "success forwards browse request and payload",
			service: &stubService{repositoryBrowseResult: workspaceorchestrator.RepositoryBrowseResult{
				Path: "/tmp/repos",
				Directories: []workspaceorchestrator.RepositoryDirectory{{
					Name:            "relay",
					Path:            "/tmp/repos/relay",
					IsGitRepository: true,
				}},
			}},
			request:  RepositoryBrowseRequestPayload{Path: "/tmp/repos", ShowHidden: true},
			wantType: TypeRepositoryBrowseResult,
			assertPayload: func(t *testing.T, payload RepositoryBrowseResultPayload) {
				if payload.Path != "/tmp/repos" || len(payload.Directories) != 1 || !payload.Directories[0].IsGitRepository {
					t.Fatalf("payload = %#v, want one Git repository directory", payload)
				}
			},
			assertService: func(t *testing.T, service *stubService) {
				if service.browseInput.Path != "/tmp/repos" || !service.browseInput.ShowHidden {
					t.Fatalf("browseInput = %#v, want /tmp/repos with ShowHidden", service.browseInput)
				}
			},
		},
		{
			name:          "failure maps to repository browse error",
			service:       &stubService{browseErr: errors.New("browse failed")},
			request:       RepositoryBrowseRequestPayload{Path: "/tmp/missing"},
			wantType:      TypeError,
			wantErrorCode: "repository_browse_failed",
			assertService: func(t *testing.T, service *stubService) {
				if service.browseInput.Path != "/tmp/missing" || service.browseInput.ShowHidden {
					t.Fatalf("browseInput = %#v, want /tmp/missing without ShowHidden", service.browseInput)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler := NewHandler(testCase.service, nil, slog.Default())
			calls := make([]capturedWrite, 0, 1)
			err := handler.handleMessage(context.Background(), Envelope{
				Type:      TypeRepositoryBrowseRequest,
				RequestID: "req_browse",
				Payload:   mustMarshalJSON(t, testCase.request),
			}, captureWrites(&calls))
			if err != nil {
				t.Fatalf("handleMessage() error = %v", err)
			}
			assertWriteType(t, calls, testCase.wantType)
			if testCase.assertService != nil {
				testCase.assertService(t, testCase.service)
			}
			if testCase.wantType == TypeRepositoryBrowseResult {
				if testCase.assertPayload != nil {
					testCase.assertPayload(t, calls[0].payload.(RepositoryBrowseResultPayload))
				}
				return
			}
			if got := calls[0].payload.(ErrorPayload).Code; got != testCase.wantErrorCode {
				t.Fatalf("ErrorPayload.Code = %q, want %q", got, testCase.wantErrorCode)
			}
		})
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
	repositoryBrowseResult workspaceorchestrator.RepositoryBrowseResult
	bootstrapLastSessionID string
	createDisplayName      string
	openSessionID          string
	preferencesInput       workspaceorchestrator.PreferencesInput
	submitInput            workspaceorchestrator.SubmitRunInput
	openRunInput           workspaceorchestrator.OpenRunInput
	cancelInput            workspaceorchestrator.CancelRunInput
	approvalInput          workspaceorchestrator.ApprovalResponseInput
	browseInput            workspaceorchestrator.RepositoryBrowseInput
	runHistoryQueryInput   workspaceorchestrator.RunHistoryQueryInput
	runHistoryRuns         []workspaceorchestrator.RunSummary
	runHistoryDetails      workspaceorchestrator.RunHistoryDetails
	runHistoryDetailsSessionID string
	runHistoryDetailsRunID string
	runHistoryExportInput  workspaceorchestrator.RunHistoryExportRequest
	runHistoryExportResult workspaceorchestrator.RunHistoryExportResult
	replayControlInput     workspaceorchestrator.ReplayControlInput
	openErr                error
	submitErr              error
	browseErr              error
}

func (s *stubService) Bootstrap(_ context.Context, lastSessionID string) (workspaceorchestrator.WorkspaceSnapshot, error) {
	s.bootstrapLastSessionID = lastSessionID
	return s.bootstrapSnapshot, nil
}

func (s *stubService) AttachWorkspaceSubscriber(_ context.Context, _ func(workspaceorchestrator.StreamEnvelope) error) {
}

func (s *stubService) BrowseRepository(_ context.Context, input workspaceorchestrator.RepositoryBrowseInput) (workspaceorchestrator.RepositoryBrowseResult, error) {
	s.browseInput = input
	if s.browseErr != nil {
		return workspaceorchestrator.RepositoryBrowseResult{}, s.browseErr
	}
	return s.repositoryBrowseResult, nil
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

func (s *stubService) ReplayControl(_ context.Context, input workspaceorchestrator.ReplayControlInput, _ func(workspaceorchestrator.StreamEnvelope) error) error {
	s.replayControlInput = input
	return nil
}

func (s *stubService) ExportRunHistory(_ context.Context, input workspaceorchestrator.RunHistoryExportRequest) (workspaceorchestrator.RunHistoryExportResult, error) {
	s.runHistoryExportInput = input
	return s.runHistoryExportResult, nil
}

func (s *stubService) QueryRunHistory(_ context.Context, input workspaceorchestrator.RunHistoryQueryInput) ([]workspaceorchestrator.RunSummary, error) {
	s.runHistoryQueryInput = input
	return s.runHistoryRuns, nil
}

func (s *stubService) GetRunHistoryDetails(_ context.Context, sessionID string, runID string) (workspaceorchestrator.RunHistoryDetails, error) {
	s.runHistoryDetailsSessionID = sessionID
	s.runHistoryDetailsRunID = runID
	return s.runHistoryDetails, nil
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

func assertWriteTypes(t *testing.T, calls []capturedWrite, wantTypes ...string) {
	t.Helper()
	if len(calls) != len(wantTypes) {
		t.Fatalf("len(calls) = %d, want %d", len(calls), len(wantTypes))
	}
	for index, wantType := range wantTypes {
		if calls[index].messageType != wantType {
			t.Fatalf("calls[%d].messageType = %q, want %q", index, calls[index].messageType, wantType)
		}
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
		ConnectedRepository: workspaceorchestrator.ConnectedRepositorySummary{
			Path:    "/tmp/project",
			Status:  "connected",
			Message: "Repository-aware reads stay inside this local Git worktree.",
		},
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
		PendingApprovals: []workspaceorchestrator.ApprovalSummary{{
			SessionID:    "session_alpha",
			RunID:        "run_1",
			Role:         sqlite.RoleCoder,
			Model:        "anthropic/claude-sonnet-4-5",
			ToolCallID:   "call_1",
			ToolName:     "write_file",
			InputPreview: map[string]any{"path": "README.md"},
			Message:      "Relay needs approval before it can write files inside the configured project root.",
			OccurredAt:   startedAt.Add(time.Second),
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
