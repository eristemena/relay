package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	workspaceorchestrator "github.com/erisristemena/relay/internal/orchestrator/workspace"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	toolspkg "github.com/erisristemena/relay/internal/tools"
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

type approvalFlowRunner struct {
	service   *workspaceorchestrator.Service
	sessionID string
	runIDReady chan string
	profile   agents.Profile
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

	writePreview := toolspkg.SafePreview("Tool call received.", map[string]any{"path": "README.md", "content": "api_key=super-secret"})
	if handlers.OnToolCall != nil {
		handlers.OnToolCall(agents.ToolCallEvent{
			ToolCallID:   "call_write",
			ToolName:     agents.ToolWriteFile,
			InputPreview: writePreview,
		})
	}
	runID := <-r.runIDReady

	decision, err := r.service.RequestApproval(ctx, workspaceorchestrator.ApprovalRequest{
		SessionID:    r.sessionID,
		RunID:        runID,
		ToolCallID:   "call_write",
		ToolName:     agents.ToolWriteFile,
		Role:         r.profile.Role,
		Model:        r.profile.Model,
		InputPreview: writePreview,
		Message:      "Relay needs approval before it can write files inside the configured project root.",
		OccurredAt:   time.Now().UTC(),
	})
	if err != nil {
		return err
	}
	if !decision.Approved {
		if handlers.OnToolResult != nil {
			handlers.OnToolResult(agents.ToolResultEvent{
				ToolCallID:    "call_write",
				ToolName:      agents.ToolWriteFile,
				Status:        "rejected",
				ResultPreview: toolspkg.SafePreview("Tool blocked.", map[string]any{"message": workspaceorchestrator.ErrApprovalRejected.Error()}),
			})
		}
		if handlers.OnError != nil {
			handlers.OnError("run_failed", workspaceorchestrator.ErrApprovalRejected.Error())
		}
		return nil
	}

	if handlers.OnComplete != nil {
		handlers.OnComplete("stop")
	}
	return nil
}