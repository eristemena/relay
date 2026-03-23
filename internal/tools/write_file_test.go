package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteFileToolWritesContentWithinProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	tool := NewWriteFileTool(projectRoot)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"nested/notes.txt","content":"token=secret"}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	body, err := os.ReadFile(filepath.Join(projectRoot, "nested", "notes.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(body) != "token=secret" {
		t.Fatalf("written file = %q, want %q", string(body), "token=secret")
	}
	wantPreview := map[string]any{"summary": "Wrote file content.", "path": "nested/notes.txt"}
	if !reflect.DeepEqual(result.Preview, wantPreview) {
		t.Fatalf("result.Preview = %#v, want %#v", result.Preview, wantPreview)
	}
}

func TestWriteFileToolRejectsPathOutsideRoot(t *testing.T) {
	tool := NewWriteFileTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../notes.txt","content":"hello"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want path guard error")
	}
	if err.Error() != "Relay blocked access outside the configured project_root" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}
