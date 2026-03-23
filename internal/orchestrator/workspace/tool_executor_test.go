package workspace

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/agents"
)

func TestCatalogToolExecutorDefinitionsSkipApprovalTools(t *testing.T) {
	executor := newCatalogToolExecutor(t.TempDir(), nil)
	definitions := executor.Definitions([]agents.ToolName{
		agents.ToolReadFile,
		agents.ToolSearchCodebase,
		agents.ToolWriteFile,
		agents.ToolRunCommand,
	})

	if len(definitions) != 2 {
		t.Fatalf("len(definitions) = %d, want 2", len(definitions))
	}
	if definitions[0].Name != agents.ToolReadFile {
		t.Fatalf("definitions[0].Name = %q, want %q", definitions[0].Name, agents.ToolReadFile)
	}
	if definitions[1].Name != agents.ToolSearchCodebase {
		t.Fatalf("definitions[1].Name = %q, want %q", definitions[1].Name, agents.ToolSearchCodebase)
	}
}

func TestCatalogToolExecutorExecutesReadFileWithinProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	readmePath := filepath.Join(projectRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	executor := newCatalogToolExecutor(projectRoot, nil)
	preview := executor.PreviewToolCall(agents.ToolReadFile, json.RawMessage(`{"path":"README.md","secret":"api_key=top-secret"}`))
	if preview["secret"] != "api_key=[redacted]" {
		t.Fatalf("preview[secret] = %v, want redacted secret", preview["secret"])
	}

	result, err := executor.ExecuteTool(context.Background(), "call_123", agents.ToolReadFile, json.RawMessage(`{"path":"README.md","start_line":2}`))
	if err != nil {
		t.Fatalf("ExecuteTool() error = %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("result.Status = %q, want completed", result.Status)
	}
	if result.Content != "beta" {
		t.Fatalf("result.Content = %q, want beta", result.Content)
	}
	wantPreview := map[string]any{"summary": "Loaded file content.", "path": "README.md"}
	if !reflect.DeepEqual(result.ResultPreview, wantPreview) {
		t.Fatalf("result.ResultPreview = %v, want %v", result.ResultPreview, wantPreview)
	}
	if result.ToolCallID != "call_123" {
		t.Fatalf("result.ToolCallID = %q, want call_123", result.ToolCallID)
	}
}

func TestCatalogToolExecutorRequestsApprovalBeforeWriteFile(t *testing.T) {
	projectRoot := t.TempDir()
	approvals := &stubApprovalManager{decision: ApprovalDecision{Approved: true}}
	executor := newCatalogToolExecutor(projectRoot, approvals)

	ctx := withRunExecutionContext(context.Background(), runExecutionContext{
		SessionID: "session_alpha",
		RunID:     "run_1",
		Emit:      func(StreamEnvelope) error { return nil },
	})

	result, err := executor.ExecuteTool(ctx, "call_approve", agents.ToolWriteFile, json.RawMessage(`{"path":"README.md","content":"hello"}`))
	if err != nil {
		t.Fatalf("ExecuteTool() error = %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("result.Status = %q, want completed", result.Status)
	}
	if approvals.request.ToolCallID != "call_approve" {
		t.Fatalf("approvals.request.ToolCallID = %q, want call_approve", approvals.request.ToolCallID)
	}
	if approvals.request.ToolName != agents.ToolWriteFile {
		t.Fatalf("approvals.request.ToolName = %q, want %q", approvals.request.ToolName, agents.ToolWriteFile)
	}
	if approvals.request.InputPreview["path"] != "README.md" {
		t.Fatalf("approvals.request.InputPreview[path] = %v, want README.md", approvals.request.InputPreview["path"])
	}
	content, err := os.ReadFile(filepath.Join(projectRoot, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("written content = %q, want hello", string(content))
	}
}

func TestCatalogToolExecutorReturnsRejectedResultWhenApprovalDenied(t *testing.T) {
	executor := newCatalogToolExecutor(t.TempDir(), &stubApprovalManager{decision: ApprovalDecision{Approved: false}})
	ctx := withRunExecutionContext(context.Background(), runExecutionContext{
		SessionID: "session_alpha",
		RunID:     "run_1",
		Emit:      func(StreamEnvelope) error { return nil },
	})

	result, err := executor.ExecuteTool(ctx, "call_reject", agents.ToolRunCommand, json.RawMessage(`{"command":"pwd"}`))
	if err != nil {
		t.Fatalf("ExecuteTool() error = %v", err)
	}
	if result.Status != "rejected" {
		t.Fatalf("result.Status = %q, want rejected", result.Status)
	}
	if result.Content != ErrApprovalRejected.Error() {
		t.Fatalf("result.Content = %q, want %q", result.Content, ErrApprovalRejected.Error())
	}
	if result.ResultPreview["message"] != ErrApprovalRejected.Error() {
		t.Fatalf("result.ResultPreview[message] = %v, want rejection message", result.ResultPreview["message"])
	}
}

func TestCatalogToolExecutorHandlesUnsupportedAndBlockedTools(t *testing.T) {
	executor := newCatalogToolExecutor(t.TempDir(), nil)

	unsupported, err := executor.ExecuteTool(context.Background(), "call_missing", agents.ToolName("missing_tool"), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("ExecuteTool() unsupported error = %v", err)
	}
	if unsupported.Status != "error" {
		t.Fatalf("unsupported.Status = %q, want error", unsupported.Status)
	}
	if unsupported.Content != "unsupported tool: missing_tool" {
		t.Fatalf("unsupported.Content = %q, want unsupported tool message", unsupported.Content)
	}

	blocked, err := executor.ExecuteTool(context.Background(), "call_blocked", agents.ToolWriteFile, json.RawMessage(`{"path":"README.md","content":"hello"}`))
	if err != nil {
		t.Fatalf("ExecuteTool() blocked error = %v", err)
	}
	if blocked.Status != "rejected" {
		t.Fatalf("blocked.Status = %q, want rejected", blocked.Status)
	}
	if blocked.Content != "Relay blocked the tool call because approval is unavailable." {
		t.Fatalf("blocked.Content = %q, want approval unavailable message", blocked.Content)
	}

	approvalExecutor := newCatalogToolExecutor(t.TempDir(), &stubApprovalManager{decision: ApprovalDecision{Approved: true}})
	missingContext, err := approvalExecutor.ExecuteTool(context.Background(), "call_context", agents.ToolRunCommand, json.RawMessage(`{"command":"pwd"}`))
	if err != nil {
		t.Fatalf("ExecuteTool() missing-context error = %v", err)
	}
	if missingContext.Status != "rejected" {
		t.Fatalf("missingContext.Status = %q, want rejected", missingContext.Status)
	}
	if missingContext.Content != "Relay blocked the tool call because the active run context is missing." {
		t.Fatalf("missingContext.Content = %q, want missing context message", missingContext.Content)
	}
}

func TestCatalogToolExecutorPreviewAndSchemaHelpers(t *testing.T) {
	executor := newCatalogToolExecutor(t.TempDir(), nil)

	emptyPreview := executor.PreviewToolCall(agents.ToolReadFile, nil)
	if emptyPreview["summary"] != "Tool call received." {
		t.Fatalf("emptyPreview = %#v, want default summary", emptyPreview)
	}

	invalidPreview := executor.PreviewToolCall(agents.ToolReadFile, json.RawMessage(`{"path":`))
	if invalidPreview["arguments"] == nil {
		t.Fatalf("invalidPreview = %#v, want redacted arguments fallback", invalidPreview)
	}

	arrayPreview := sanitizePreviewValue(map[string]any{
		"items": []any{"token=secret", map[string]any{"authorization": "bearer abc123"}},
	}).(map[string]any)
	items, ok := arrayPreview["items"].([]any)
	if !ok || items[0] != "token=[redacted]" {
		t.Fatalf("sanitizePreviewValue() items = %#v, want redacted array values", arrayPreview["items"])
	}

	readSchema := toolSchema(agents.ToolReadFile)
	if readSchema.Required[0] != "path" || readSchema.Properties["path"].Type != "string" {
		t.Fatalf("toolSchema(read_file) = %#v, want path requirement", readSchema)
	}
	commandSchema := toolSchema(agents.ToolRunCommand)
	if commandSchema.Properties["args"].Type != "array" {
		t.Fatalf("toolSchema(run_command) args = %#v, want array", commandSchema.Properties["args"])
	}
	unknownSchema := toolSchema(agents.ToolName("unknown"))
	if unknownSchema.Type != "object" {
		t.Fatalf("toolSchema(unknown) = %#v, want object fallback", unknownSchema)
	}

	if approvalMessage(agents.ToolWriteFile) == approvalMessage(agents.ToolRunCommand) {
		t.Fatal("approvalMessage() returned the same string for distinct approval-gated tools")
	}
	if approvalMessage(agents.ToolName("other")) != "Relay needs approval before it can continue with this tool call." {
		t.Fatalf("approvalMessage(default) = %q, want generic approval message", approvalMessage(agents.ToolName("other")))
	}
}

type stubApprovalManager struct {
	decision ApprovalDecision
	request  ApprovalRequest
}

func (s *stubApprovalManager) RequestApproval(_ context.Context, request ApprovalRequest) (ApprovalDecision, error) {
	if request.OccurredAt.IsZero() {
		request.OccurredAt = time.Now().UTC()
	}
	s.request = request
	return s.decision, nil
}