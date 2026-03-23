package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRedactTextRedactsSecretsAndTruncates(t *testing.T) {
	input := "api_key=secret-token authorization: bearer abc123 token: xyz"
	got := RedactText(input)
	if got != "api_key=[redacted] authorization: bearer [redacted] token: [redacted]" {
		t.Fatalf("RedactText() = %q", got)
	}

	long := strings.Repeat("a", 450)
	truncated := RedactText(long)
	if len(truncated) != 403 {
		t.Fatalf("len(truncated) = %d, want 403", len(truncated))
	}
	if truncated[len(truncated)-3:] != "..." {
		t.Fatalf("truncated suffix = %q, want ...", truncated[len(truncated)-3:])
	}
}

func TestSafePreviewRedactsStringFields(t *testing.T) {
	preview := SafePreview("authorization: bearer secret", map[string]any{
		"message": "token=super-secret",
		"count":   2,
	})

	want := map[string]any{
		"summary": "authorization: bearer [redacted]",
		"message": "token=[redacted]",
		"count":   2,
	}
	if !reflect.DeepEqual(preview, want) {
		t.Fatalf("SafePreview() = %#v, want %#v", preview, want)
	}
}

func TestReadFileToolReadsRequestedRange(t *testing.T) {
	projectRoot := t.TempDir()
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

func TestReadFileToolRejectsPathsOutsideRoot(t *testing.T) {
	tool := NewReadFileTool(t.TempDir())
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"path":"../secrets.txt"}`))
	if err == nil {
		t.Fatal("Execute() error = nil, want path guard error")
	}
	if err.Error() != "Relay blocked access outside the configured project_root" {
		t.Fatalf("Execute() error = %q", err.Error())
	}
}

func TestCatalogDefinitionsLookupAndUnsupportedTool(t *testing.T) {
	projectRoot := t.TempDir()
	catalog := NewCatalog(projectRoot)
	definitions := catalog.Definitions()
	if len(definitions) != 4 {
		t.Fatalf("len(definitions) = %d, want 4", len(definitions))
	}

	lookupCases := map[string]Name{
		"read_file":       ReadFileName,
		"search_codebase": SearchCodebaseName,
		"write_file":      WriteFileName,
		"run_command":     RunCommandName,
	}
	for name, expected := range lookupCases {
		tool, ok := catalog.Lookup(name)
		if !ok {
			t.Fatalf("Lookup(%q) = missing, want tool", name)
		}
		if tool.Definition().Name != expected {
			t.Fatalf("Lookup(%q).Definition().Name = %q, want %q", name, tool.Definition().Name, expected)
		}
	}
	if _, ok := catalog.Lookup("missing_tool"); ok {
		t.Fatal("Lookup(missing_tool) = found, want missing")
	}

	err := UnsupportedToolError("missing_tool")
	if err == nil || err.Error() != "unsupported tool: missing_tool" {
		t.Fatalf("UnsupportedToolError() = %v, want unsupported tool message", err)
	}
}

func TestToolDefinitionsDescribeParametersAndApproval(t *testing.T) {
	projectRoot := t.TempDir()
	definitions := []Definition{
		NewReadFileTool(projectRoot).Definition(),
		NewSearchCodebaseTool(projectRoot).Definition(),
		NewWriteFileTool(projectRoot).Definition(),
		NewRunCommandTool(projectRoot).Definition(),
	}

	if definitions[0].Name != ReadFileName || definitions[0].RequiresApproval {
		t.Fatalf("read_file definition = %#v, want read-only tool", definitions[0])
	}
	if definitions[1].Name != SearchCodebaseName || definitions[1].RequiresApproval {
		t.Fatalf("search_codebase definition = %#v, want read-only tool", definitions[1])
	}
	if definitions[2].Name != WriteFileName || !definitions[2].RequiresApproval {
		t.Fatalf("write_file definition = %#v, want approval-gated tool", definitions[2])
	}
	if definitions[3].Name != RunCommandName || !definitions[3].RequiresApproval {
		t.Fatalf("run_command definition = %#v, want approval-gated tool", definitions[3])
	}

	for _, definition := range definitions {
		if len(definition.Parameters) == 0 {
			t.Fatalf("definition %q missing parameters", definition.Name)
		}
	}
}
