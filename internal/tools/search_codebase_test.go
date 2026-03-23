package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestSearchCodebaseToolFindsMatchesAndRespectsPattern(t *testing.T) {
	projectRoot := t.TempDir()
	writeTextFile(t, filepath.Join(projectRoot, "README.md"), "alpha beta\n")
	writeTextFile(t, filepath.Join(projectRoot, "notes.txt"), "alpha only\n")
	writeTextFile(t, filepath.Join(projectRoot, "nested", "guide.md"), "beta alpha\n")

	tool := NewSearchCodebaseTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"alpha","include_pattern":"*.md","max_results":5}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Output != `["README.md","nested/guide.md"]` {
		t.Fatalf("result.Output = %q", result.Output)
	}
	if got := result.Preview["match_count"]; got != 2 {
		t.Fatalf("result.Preview[match_count] = %v, want 2", got)
	}
}

func TestSearchCodebaseToolRequiresQuery(t *testing.T) {
	tool := NewSearchCodebaseTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"  "}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want query validation error")
	}
	if err.Error() != "search query is required" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}
