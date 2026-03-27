package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	workspaceorchestrator "github.com/erisristemena/relay/internal/orchestrator/workspace"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	toolspkg "github.com/erisristemena/relay/internal/tools"
	git "github.com/go-git/go-git/v5"
	"nhooyr.io/websocket"
)

func TestToolCallOrdering_ApprovalRejectionReplayAndRedaction(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Tool ordering session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Edit the readme after checking the repository",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	statePayload := stateChange["payload"].(map[string]any)
	runID := statePayload["run_id"].(string)
	runner.runIDReady <- runID
	firstToolCall := readUntilStreamingType(t, connection, "tool_call")
	firstToolResult := readUntilStreamingType(t, connection, "tool_result")
	secondToolCall := readUntilStreamingType(t, connection, "tool_call")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")

	firstToolCallPayload := firstToolCall["payload"].(map[string]any)
	firstToolResultPayload := firstToolResult["payload"].(map[string]any)
	secondToolCallPayload := secondToolCall["payload"].(map[string]any)
	approvalPayload := approvalRequest["payload"].(map[string]any)

	if secondToolCallPayload["input_preview"].(map[string]any)["content"] != "api_key=[redacted]" {
		t.Fatalf("second tool call content preview = %#v, want redacted content", secondToolCallPayload["input_preview"])
	}
	if approvalPayload["input_preview"].(map[string]any)["content"] != "api_key=[redacted]" {
		t.Fatalf("approval preview = %#v, want redacted content", approvalPayload["input_preview"])
	}

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.approval.respond",
		"payload": map[string]any{
			"session_id":   session.ID,
			"run_id":       runID,
			"tool_call_id": "call_write",
			"decision":     "rejected",
		},
	})

	rejectedToolResult := readUntilStreamingType(t, connection, "tool_result")
	terminalError := readUntilStreamingType(t, connection, "error")

	rejectedPayload := rejectedToolResult["payload"].(map[string]any)
	errorPayload := terminalError["payload"].(map[string]any)

	sequences := []int{
		int(statePayload["sequence"].(float64)),
		int(firstToolCallPayload["sequence"].(float64)),
		int(firstToolResultPayload["sequence"].(float64)),
		int(secondToolCallPayload["sequence"].(float64)),
		int(rejectedPayload["sequence"].(float64)),
		int(errorPayload["sequence"].(float64)),
	}
	for index := 1; index < len(sequences); index++ {
		if sequences[index-1] >= sequences[index] {
			t.Fatalf("sequence order = %v, want strictly increasing", sequences)
		}
	}

	if rejectedPayload["status"] != "rejected" {
		t.Fatalf("rejected tool result status = %v, want rejected", rejectedPayload["status"])
	}
	if rejectedPayload["result_preview"].(map[string]any)["message"] != workspaceorchestrator.ErrApprovalRejected.Error() {
		t.Fatalf("rejected tool result preview = %#v, want rejection message", rejectedPayload["result_preview"])
	}
	if errorPayload["code"] != "run_failed" {
		t.Fatalf("terminal error code = %v, want run_failed", errorPayload["code"])
	}
	if errorPayload["message"] != workspaceorchestrator.ErrApprovalRejected.Error() {
		t.Fatalf("terminal error message = %v, want rejection message", errorPayload["message"])
	}

	replayConnection := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, replayConnection, map[string]any{
		"type": "agent.run.open",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     runID,
		},
	})

	replayState := readUntilStreamingType(t, replayConnection, "state_change")
	replayFirstToolCall := readUntilStreamingType(t, replayConnection, "tool_call")
	replayFirstToolResult := readUntilStreamingType(t, replayConnection, "tool_result")
	replaySecondToolCall := readUntilStreamingType(t, replayConnection, "tool_call")
	replayRejected := readUntilStreamingType(t, replayConnection, "tool_result")
	replayError := readUntilStreamingType(t, replayConnection, "error")

	replayEnvelopes := []map[string]any{replayState, replayFirstToolCall, replayFirstToolResult, replaySecondToolCall, replayRejected, replayError}
	for _, envelope := range replayEnvelopes {
		payload := envelope["payload"].(map[string]any)
		if payload["replay"] != true {
			t.Fatalf("replayed payload = %#v, want replay=true", payload)
		}
	}
	if replayError["payload"].(map[string]any)["message"] != workspaceorchestrator.ErrApprovalRejected.Error() {
		t.Fatalf("replayed terminal error = %#v, want preserved rejection message", replayError["payload"])
	}
}

func TestToolCallOrdering_TesterApprovalAllowsRunToContinue(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Tester approval session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleTester,
			Model: config.DefaultTesterModel,
		},
		writePath:    "tests/generated/smoke_test.sh",
		writeContent: "#!/bin/sh\necho ok\n",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Create a smoke test script and continue after approval",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	_ = readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	_ = readUntilStreamingType(t, connection, "tool_call")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")
	approvalPayload := approvalRequest["payload"].(map[string]any)

	if approvalPayload["role"] != string(sqlite.RoleTester) {
		t.Fatalf("approval role = %v, want tester", approvalPayload["role"])
	}
	if approvalPayload["tool_name"] != string(agents.ToolWriteFile) {
		t.Fatalf("approval tool_name = %v, want write_file", approvalPayload["tool_name"])
	}
	if approvalPayload["input_preview"].(map[string]any)["path"] != "tests/generated/smoke_test.sh" {
		t.Fatalf("approval input preview = %#v, want test path", approvalPayload["input_preview"])
	}

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.approval.respond",
		"payload": map[string]any{
			"session_id":   session.ID,
			"run_id":       runID,
			"tool_call_id": "call_write",
			"decision":     "approved",
		},
	})

	toolResult := readUntilStreamingType(t, connection, "tool_result")

	toolResultPayload := toolResult["payload"].(map[string]any)
	if toolResultPayload["status"] != "completed" {
		t.Fatalf("tool result status = %v, want completed", toolResultPayload["status"])
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := store.GetAgentRun(context.Background(), runID)
		if err != nil {
			t.Fatalf("GetAgentRun() error = %v", err)
		}
		if run.State == sqlite.RunStateCompleted {
			if run.ErrorCode != "" {
				t.Fatalf("run.ErrorCode = %q, want empty", run.ErrorCode)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for tester run completion after approval")
}

func TestToolCallOrdering_PendingApprovalRehydratesOnBootstrapReconnect(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = repoRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Approval reconnect session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
		repoRoot:     repoRoot,
		writePath:    "README.md",
		writeContent: "after\n",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Prepare a file diff and wait for approval",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	_ = readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	_ = readUntilStreamingType(t, connection, "tool_call")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")
	approvalPayload := approvalRequest["payload"].(map[string]any)
	if approvalPayload["request_kind"] != toolspkg.RequestKindFileWrite {
		t.Fatalf("approval request_kind = %v, want %q", approvalPayload["request_kind"], toolspkg.RequestKindFileWrite)
	}
	if approvalPayload["repository_root"] != repoRoot {
		t.Fatalf("approval repository_root = %v, want %q", approvalPayload["repository_root"], repoRoot)
	}
	if _, ok := approvalPayload["diff_preview"].(map[string]any); !ok {
		t.Fatalf("approval diff_preview = %#v, want diff preview map", approvalPayload["diff_preview"])
	}

	reconnected := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, reconnected, map[string]any{
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{
			"last_session_id": session.ID,
		},
	})

	bootstrap := readUntilStreamingType(t, reconnected, "workspace.bootstrap")
	bootstrapPayload := bootstrap["payload"].(map[string]any)
	pendingApprovals := bootstrapPayload["pending_approvals"].([]any)
	if len(pendingApprovals) != 1 {
		t.Fatalf("len(pending_approvals) = %d, want 1", len(pendingApprovals))
	}
	restored := pendingApprovals[0].(map[string]any)
	if restored["tool_call_id"] != "call_write" {
		t.Fatalf("restored tool_call_id = %v, want call_write", restored["tool_call_id"])
	}
	if restored["request_kind"] != toolspkg.RequestKindFileWrite {
		t.Fatalf("restored request_kind = %v, want %q", restored["request_kind"], toolspkg.RequestKindFileWrite)
	}
	if restored["repository_root"] != repoRoot {
		t.Fatalf("restored repository_root = %v, want %q", restored["repository_root"], repoRoot)
	}
	diffPreview, ok := restored["diff_preview"].(map[string]any)
	if !ok {
		t.Fatalf("restored diff_preview = %#v, want diff preview map", restored["diff_preview"])
	}
	if diffPreview["proposed_content"] != "after\n" {
		t.Fatalf("restored proposed_content = %#v, want updated content", diffPreview["proposed_content"])
	}
}

func TestToolCallOrdering_StaleCommandApprovalIsBlockedAfterRepositoryChange(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	repoRoot := initIntegrationRepositoryRoot(t)
	replacementRoot := initIntegrationRepositoryRoot(t)

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = repoRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Approval stale session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
		repoRoot: repoRoot,
		toolName: agents.ToolRunCommand,
		command:  "pwd",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Prepare a command and wait for approval",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	_ = readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	_ = readUntilStreamingType(t, connection, "tool_call")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")
	approvalPayload := approvalRequest["payload"].(map[string]any)
	if approvalPayload["request_kind"] != toolspkg.RequestKindCommand {
		t.Fatalf("approval request_kind = %v, want %q", approvalPayload["request_kind"], toolspkg.RequestKindCommand)
	}

	cfg, _, err = config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() reload error = %v", err)
	}
	cfg.ProjectRoot = replacementRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() replacement error = %v", err)
	}

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.approval.respond",
		"payload": map[string]any{
			"session_id":   session.ID,
			"run_id":       runID,
			"tool_call_id": "call_write",
			"decision":     "approved",
		},
	})

	approvalStateChanged := readUntilStreamingType(t, connection, "approval_state_changed")
	approvalStatePayload := approvalStateChanged["payload"].(map[string]any)
	if approvalStatePayload["status"] != sqlite.ApprovalStateBlocked {
		t.Fatalf("approval state status = %v, want %q", approvalStatePayload["status"], sqlite.ApprovalStateBlocked)
	}
	if approvalStatePayload["message"] == "" {
		t.Fatal("approval state message = empty, want plain-language blocking explanation")
	}

	toolResultPayload := readUntilStreamingToolResult(t, connection, "call_write")
	if toolResultPayload["tool_call_id"] != "call_write" {
		t.Fatalf("tool result tool_call_id = %v, want call_write", toolResultPayload["tool_call_id"])
	}
	if toolResultPayload["status"] != "rejected" {
		t.Fatalf("tool result status = %v, want rejected", toolResultPayload["status"])
	}

	storedApproval, err := store.GetApprovalRequest(context.Background(), runID, "call_write")
	if err != nil {
		t.Fatalf("GetApprovalRequest() error = %v", err)
	}
	if storedApproval.State != sqlite.ApprovalStateBlocked {
		t.Fatalf("storedApproval.State = %q, want %q", storedApproval.State, sqlite.ApprovalStateBlocked)
	}
}

type approvalFlowRunner struct {
	service      *workspaceorchestrator.Service
	sessionID    string
	runIDReady   chan string
	profile      agents.Profile
	repoRoot      string
	toolName      agents.ToolName
	command       string
	commandArgs   []string
	writePath    string
	writeContent string
}

func (r *approvalFlowRunner) Profile() agents.Profile {
	return r.profile
}

func (r *approvalFlowRunner) Run(ctx context.Context, _ string, handlers agents.StreamEventHandlers) error {
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(string(sqlite.RunStateThinking))
	}

	readPreview := toolspkg.SafePreview("Tool call received.", map[string]any{"path": "README.md"})
	if handlers.OnToolCall != nil {
		handlers.OnToolCall(agents.ToolCallEvent{
			ToolCallID:   "call_read",
			ToolName:     agents.ToolReadFile,
			InputPreview: readPreview,
		})
	}
	if handlers.OnToolResult != nil {
		handlers.OnToolResult(agents.ToolResultEvent{
			ToolCallID:    "call_read",
			ToolName:      agents.ToolReadFile,
			Status:        "completed",
			ResultPreview: toolspkg.SafePreview("Loaded file content.", map[string]any{"path": "README.md"}),
		})
	}

	writePath := r.writePath
	if writePath == "" {
		writePath = "README.md"
	}
	writeContent := r.writeContent
	if writeContent == "" {
		writeContent = "api_key=super-secret"
	}
	toolName := r.toolName
	if toolName == "" {
		toolName = agents.ToolWriteFile
	}
	var inputPreview map[string]any
	if strings.TrimSpace(r.repoRoot) != "" {
		switch toolName {
		case agents.ToolRunCommand:
			commandName := r.command
			if commandName == "" {
				commandName = "pwd"
			}
			preview, err := toolspkg.BuildRunCommandPreview(r.repoRoot, toolspkg.RunCommandInput{Command: commandName, Args: r.commandArgs})
			if err != nil {
				return err
			}
			inputPreview = preview
		default:
			preview, err := toolspkg.BuildWriteFilePreview(r.repoRoot, toolspkg.WriteFileInput{Path: writePath, Content: writeContent})
			if err != nil {
				return err
			}
			inputPreview = preview
		}
	}
	if inputPreview == nil {
		if toolName == agents.ToolRunCommand {
			commandName := r.command
			if commandName == "" {
				commandName = "pwd"
			}
			inputPreview = toolspkg.SafePreview("Tool call received.", map[string]any{"command": commandName, "args": r.commandArgs})
		} else {
			inputPreview = toolspkg.SafePreview("Tool call received.", map[string]any{"path": writePath, "content": writeContent})
		}
	}
	if handlers.OnToolCall != nil {
		handlers.OnToolCall(agents.ToolCallEvent{
			ToolCallID:   "call_write",
			ToolName:     toolName,
			InputPreview: inputPreview,
		})
	}
	runID := <-r.runIDReady

	decision, err := r.service.RequestApproval(ctx, workspaceorchestrator.ApprovalRequest{
		SessionID:    r.sessionID,
		RunID:        runID,
		ToolCallID:   "call_write",
		ToolName:     toolName,
		Role:         r.profile.Role,
		Model:        r.profile.Model,
		InputPreview: inputPreview,
		Message:      approvalFlowMessage(toolName),
		OccurredAt:   time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	if !decision.Approved {
		if handlers.OnToolResult != nil {
			handlers.OnToolResult(agents.ToolResultEvent{
				ToolCallID:    "call_write",
				ToolName:      toolName,
				Status:        "rejected",
				ResultPreview: toolspkg.SafePreview("Tool blocked.", map[string]any{"message": workspaceorchestrator.ErrApprovalRejected.Error()}),
			})
		}
		if handlers.OnError != nil {
			handlers.OnError("run_failed", workspaceorchestrator.ErrApprovalRejected.Error())
		}
		return nil
	}

	if handlers.OnToolResult != nil {
		resultPreview := toolspkg.SafePreview("Wrote file content.", map[string]any{"path": writePath})
		if toolName == agents.ToolRunCommand {
			commandName := r.command
			if commandName == "" {
				commandName = "pwd"
			}
			resultPreview = toolspkg.SafePreview("Command completed.", map[string]any{"command": commandName})
		}
		handlers.OnToolResult(agents.ToolResultEvent{
			ToolCallID:    "call_write",
			ToolName:      toolName,
			Status:        "completed",
			ResultPreview: resultPreview,
		})
	}

	if handlers.OnComplete != nil {
		handlers.OnComplete(agents.CompletionMetadata{FinishReason: "stop"})
	}
	return nil
}

func initIntegrationRepositoryRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := git.PlainInit(root, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	return root
}

func approvalFlowMessage(toolName agents.ToolName) string {
	if toolName == agents.ToolRunCommand {
		return "Relay needs approval before it can run a shell command from the configured project root."
	}
	return "Relay needs approval before it can write files inside the configured project root."
}

func readUntilStreamingToolResult(t *testing.T, connection *websocket.Conn, toolCallID string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(streamingIOTimeout)
	for time.Now().Before(deadline) {
		envelope := readUntilStreamingType(t, connection, "tool_result")
		payload := envelope["payload"].(map[string]any)
		if payload["tool_call_id"] == toolCallID {
			return payload
		}
	}
	t.Fatalf("timed out waiting for tool_result with tool_call_id %q", toolCallID)
	return nil
}
