package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunCommandToolExecutesFromProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
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
	projectRoot := t.TempDir()
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
