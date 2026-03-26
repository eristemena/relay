package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBuildWriteFilePreviewIncludesDiffMetadata(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	if err := os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	preview, err := BuildWriteFilePreview(projectRoot, WriteFileInput{Path: "README.md", Content: "after\n"})
	if err != nil {
		t.Fatalf("BuildWriteFilePreview() error = %v", err)
	}
	if preview["request_kind"] != RequestKindFileWrite {
		t.Fatalf("preview[request_kind] = %v, want %q", preview["request_kind"], RequestKindFileWrite)
	}
	if preview["repository_root"] != projectRoot {
		t.Fatalf("preview[repository_root] = %v, want %q", preview["repository_root"], projectRoot)
	}
	diffPreview, ok := preview["diff_preview"].(map[string]any)
	if !ok {
		t.Fatalf("preview[diff_preview] = %#v, want diff preview map", preview["diff_preview"])
	}
	if diffPreview["original_content"] != "before\n" {
		t.Fatalf("diffPreview[original_content] = %#v, want original file content", diffPreview["original_content"])
	}
	if diffPreview["proposed_content"] != "after\n" {
		t.Fatalf("diffPreview[proposed_content] = %#v, want proposed file content", diffPreview["proposed_content"])
	}
	if diffPreview["base_content_hash"] == "" {
		t.Fatal("diffPreview[base_content_hash] = empty, want populated hash")
	}
}

func TestCurrentWriteFileBaseHashTracksExistingAndMissingFiles(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	if err := os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	existingHash, err := CurrentWriteFileBaseHash(projectRoot, "README.md")
	if err != nil {
		t.Fatalf("CurrentWriteFileBaseHash(existing) error = %v", err)
	}
	missingHash, err := CurrentWriteFileBaseHash(projectRoot, "missing.txt")
	if err != nil {
		t.Fatalf("CurrentWriteFileBaseHash(missing) error = %v", err)
	}
	if existingHash == missingHash {
		t.Fatalf("existingHash = %q, missingHash = %q, want different hashes", existingHash, missingHash)
	}
}

func TestWriteFileToolWritesContentWithinProjectRoot(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
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
	tool := NewWriteFileTool(initRepositoryRoot(t))
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../notes.txt","content":"hello"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want path guard error")
	}
	if err.Error() != "Relay blocked access outside the configured project_root" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}
