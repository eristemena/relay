package tools

import (
	"reflect"
	"sort"
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

func TestCatalogDefinitionsLookupAndUnsupportedTool(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	catalog := NewCatalog(projectRoot)
	definitions := catalog.Definitions()
	if len(definitions) != 7 {
		t.Fatalf("len(definitions) = %d, want 7", len(definitions))
	}

	lookupCases := map[string]Name{
		"read_file":       ReadFileName,
		"list_files":      ListFilesName,
		"search_codebase": SearchCodebaseName,
		"git_log":         GitLogName,
		"git_diff":        GitDiffName,
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

func TestCatalogDefinitionsIncludeRepositoryAwareReadOnlyTools(t *testing.T) {
	t.Parallel()

	definitions := NewCatalog(initRepositoryRoot(t)).Definitions()
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, string(definition.Name))
	}
	sort.Strings(names)

	want := []string{"git_diff", "git_log", "list_files", "read_file", "run_command", "search_codebase", "write_file"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("catalog definition names = %#v, want %#v", names, want)
	}
}

func TestToolDefinitionsDescribeParametersAndApproval(t *testing.T) {
	projectRoot := initRepositoryRoot(t)
	definitions := []Definition{
		NewReadFileTool(projectRoot).Definition(),
		NewListFilesTool(projectRoot).Definition(),
		NewSearchCodebaseTool(projectRoot).Definition(),
		NewGitLogTool(projectRoot).Definition(),
		NewGitDiffTool(projectRoot).Definition(),
		NewWriteFileTool(projectRoot).Definition(),
		NewRunCommandTool(projectRoot).Definition(),
	}

	if definitions[0].Name != ReadFileName || definitions[0].RequiresApproval {
		t.Fatalf("read_file definition = %#v, want read-only tool", definitions[0])
	}
	if definitions[1].Name != ListFilesName || definitions[1].RequiresApproval {
		t.Fatalf("list_files definition = %#v, want read-only tool", definitions[1])
	}
	if definitions[2].Name != SearchCodebaseName || definitions[2].RequiresApproval {
		t.Fatalf("search_codebase definition = %#v, want read-only tool", definitions[2])
	}
	if definitions[3].Name != GitLogName || definitions[3].RequiresApproval {
		t.Fatalf("git_log definition = %#v, want read-only tool", definitions[3])
	}
	if definitions[4].Name != GitDiffName || definitions[4].RequiresApproval {
		t.Fatalf("git_diff definition = %#v, want read-only tool", definitions[4])
	}
	if definitions[5].Name != WriteFileName || !definitions[5].RequiresApproval {
		t.Fatalf("write_file definition = %#v, want approval-gated tool", definitions[5])
	}
	if definitions[6].Name != RunCommandName || !definitions[6].RequiresApproval {
		t.Fatalf("run_command definition = %#v, want approval-gated tool", definitions[6])
	}

	for _, definition := range definitions {
		if len(definition.Parameters) == 0 {
			t.Fatalf("definition %q missing parameters", definition.Name)
		}
	}
}
