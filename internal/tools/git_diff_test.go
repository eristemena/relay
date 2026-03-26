package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGitDiffToolReturnsWorkingTreePatch(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	commitRepositoryFile(t, projectRoot, "README.md", "hello\n", "Add readme")
	writeTextFile(t, projectRoot+"/README.md", "hello\nupdated\n")

	tool := NewGitDiffTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var entries []map[string]string
	if err := json.Unmarshal([]byte(result.Output), &entries); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}
	if entries[0]["path"] != "README.md" {
		t.Fatalf("entries[0][path] = %q, want README.md", entries[0]["path"])
	}
	if !strings.Contains(entries[0]["status"], "worktree:modified") {
		t.Fatalf("entries[0][status] = %q, want worktree modified", entries[0]["status"])
	}
	if !strings.Contains(entries[0]["patch"], "updated") {
		t.Fatalf("entries[0][patch] = %q, want updated content in patch", entries[0]["patch"])
	}
}

func TestGitDiffToolRejectsPathsOutsideRoot(t *testing.T) {
	tool := NewGitDiffTool(initRepositoryRoot(t))
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../README.md"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want path guard error")
	}
	if err.Error() != "Relay blocked access outside the configured project_root" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}

func TestGitDiffToolRejectsInvalidProjectRoot(t *testing.T) {
	tool := NewGitDiffTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want repository validation failure")
	}
	if !strings.Contains(err.Error(), "local Git repository root") {
		t.Fatalf("Execute() error = %q, want git repository guidance", err.Error())
	}
}
