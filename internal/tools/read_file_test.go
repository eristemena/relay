package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReadFileToolReadsRequestedRange(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	writeTextFile(t, filepath.Join(projectRoot, "README.md"), "one\ntwo\nthree\n")

	tool := NewReadFileTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md","start_line":2,"end_line":3}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Output != "two\nthree" {
		t.Fatalf("result.Output = %q, want %q", result.Output, "two\nthree")
	}
	wantPreview := map[string]any{"summary": "Loaded file content.", "path": "README.md"}
	if !reflect.DeepEqual(result.Preview, wantPreview) {
		t.Fatalf("result.Preview = %#v, want %#v", result.Preview, wantPreview)
	}
}

func TestReadFileToolDefaultsToWholeFile(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	writeTextFile(t, filepath.Join(projectRoot, "docs", "guide.md"), "alpha\nbeta\ngamma\n")

	tool := NewReadFileTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"docs/guide.md","start_line":0,"end_line":0}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Output != "alpha\nbeta\ngamma" {
		t.Fatalf("result.Output = %q, want full file content", result.Output)
	}
}

func TestReadFileToolPreviewPreservesRelativePathMetadata(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	writeTextFile(t, filepath.Join(projectRoot, "docs", "guide.md"), "alpha\n")

	tool := NewReadFileTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"docs/guide.md"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Preview["path"] != "docs/guide.md" {
		t.Fatalf("result.Preview[path] = %v, want docs/guide.md", result.Preview["path"])
	}
}

func TestReadFileToolRejectsPathsOutsideRoot(t *testing.T) {
	tool := NewReadFileTool(initRepositoryRoot(t))
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../secrets.txt"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want path guard error")
	}
	if err.Error() != "Relay blocked access outside the configured project_root" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}

func TestReadFileToolRejectsInvalidProjectRoot(t *testing.T) {
	tool := NewReadFileTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"README.md"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want repository validation failure")
	}
	if !strings.Contains(err.Error(), "local Git repository root") {
		t.Fatalf("Execute() error = %q, want git repository guidance", err.Error())
	}
}