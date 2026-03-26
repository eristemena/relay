package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListFilesToolListsImmediateEntries(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	writeTextFile(t, filepath.Join(projectRoot, "README.md"), "hello\n")
	writeTextFile(t, filepath.Join(projectRoot, "nested", "guide.md"), "nested\n")

	tool := NewListFilesTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"max_results":10}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var entries []string
	if err := json.Unmarshal([]byte(result.Output), &entries); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []string{"README.md", "nested/"}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("entries = %#v, want %#v", entries, want)
	}
}

func TestListFilesToolListsRecursively(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	writeTextFile(t, filepath.Join(projectRoot, "README.md"), "hello\n")
	writeTextFile(t, filepath.Join(projectRoot, "nested", "guide.md"), "nested\n")
	writeTextFile(t, filepath.Join(projectRoot, "node_modules", "ignore.js"), "console.log('ignore')\n")

	tool := NewListFilesTool(projectRoot)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"recursive":true,"max_results":10}`))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var entries []string
	if err := json.Unmarshal([]byte(result.Output), &entries); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	want := []string{"README.md", "nested/", "nested/guide.md"}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("entries = %#v, want %#v", entries, want)
	}
}

func TestListFilesToolRejectsPathsOutsideRoot(t *testing.T) {
	tool := NewListFilesTool(initRepositoryRoot(t))
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../private","recursive":true}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want path guard error")
	}
	if err.Error() != "Relay blocked access outside the configured project_root" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}

func TestListFilesToolRejectsInvalidProjectRoot(t *testing.T) {
	tool := NewListFilesTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"recursive":true}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want repository validation failure")
	}
	if !strings.Contains(err.Error(), "local Git repository root") {
		t.Fatalf("Execute() error = %q, want git repository guidance", err.Error())
	}
}
