package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGitLogToolReturnsRecentCommits(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	commitRepositoryFile(t, projectRoot, "README.md", "hello\n", "Add readme")
	commitRepositoryFile(t, projectRoot, "docs/guide.md", "guide\n", "Add guide")

	tool := NewGitLogTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"max_results":1}`))
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
	if entries[0]["subject"] != "Add guide" {
		t.Fatalf("entries[0][subject] = %q, want %q", entries[0]["subject"], "Add guide")
	}
	if entries[0]["author_name"] != "Relay Test" {
		t.Fatalf("entries[0][author_name] = %q, want Relay Test", entries[0]["author_name"])
	}
}

func TestGitLogToolRejectsInvalidProjectRoot(t *testing.T) {
	tool := NewGitLogTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"max_results":1}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want repository validation failure")
	}
	if !strings.Contains(err.Error(), "local Git repository root") {
		t.Fatalf("Execute() error = %q, want git repository guidance", err.Error())
	}
}
