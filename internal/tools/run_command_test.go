package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildRunCommandPreviewIncludesEffectiveDirectory(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	preview, err := BuildRunCommandPreview(projectRoot, RunCommandInput{Command: "go", Args: []string{"test", "./..."}})
	if err != nil {
		t.Fatalf("BuildRunCommandPreview() error = %v", err)
	}
	if preview["request_kind"] != RequestKindCommand {
		t.Fatalf("preview[request_kind] = %v, want %q", preview["request_kind"], RequestKindCommand)
	}
	commandPreview, ok := preview["command_preview"].(map[string]any)
	if !ok {
		t.Fatalf("preview[command_preview] = %#v, want command preview map", preview["command_preview"])
	}
	if commandPreview["effective_dir"] != projectRoot {
		t.Fatalf("commandPreview[effective_dir] = %v, want %q", commandPreview["effective_dir"], projectRoot)
	}
}

func TestRunCommandToolExecutesFromProjectRoot(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	tool := NewRunCommandTool(projectRoot)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"pwd"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.TrimSpace(result.Output) != projectRoot {
		t.Fatalf("result.Output = %q, want %q", strings.TrimSpace(result.Output), projectRoot)
	}
	if got := result.Preview["command"]; got != "pwd" {
		t.Fatalf("result.Preview[command] = %v, want pwd", got)
	}
}

func TestRunCommandToolReturnsExecutionError(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	tool := NewRunCommandTool(projectRoot)

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"sh","args":["-c","exit 7"]}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want command failure")
	}
	if !strings.Contains(err.Error(), "run command") {
		t.Fatalf("Execute() error = %q, want wrapped run command error", err.Error())
	}
	if !strings.Contains(err.Error(), "exit status 7") {
		t.Fatalf("Execute() error = %q, want exit status", err.Error())
	}
}

func TestRunCommandToolRejectsInvalidProjectRoot(t *testing.T) {
	tool := NewRunCommandTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"pwd"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid project root error")
	}
	if !strings.Contains(err.Error(), "local Git repository root") {
		t.Fatalf("Execute() error = %q, want git repository guidance", err.Error())
	}
}
