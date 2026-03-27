package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	git "github.com/go-git/go-git/v5"
)

func TestLoad_FieldLevelFallbackAndRedaction(t *testing.T) {
	paths, err := EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	content := []byte(`port = 99999
open_browser_on_start = true
appearance_variant = "sunrise"
last_session_id = "session_alpha"

[[credentials]]
provider = "openai"
label = "Personal"
secret = "top-secret"
`)
	if err := os.WriteFile(paths.ConfigPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, warnings, err := Load(paths)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != DefaultPort {
		t.Fatalf("cfg.Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.AppearanceVariant != DefaultAppearanceVariant {
		t.Fatalf("cfg.AppearanceVariant = %q, want %q", cfg.AppearanceVariant, DefaultAppearanceVariant)
	}
	if cfg.LastSessionID != "session_alpha" {
		t.Fatalf("cfg.LastSessionID = %q, want session_alpha", cfg.LastSessionID)
	}
	if len(cfg.Credentials) != 1 {
		t.Fatalf("len(cfg.Credentials) = %d, want 1", len(cfg.Credentials))
	}
	if len(warnings) < 2 {
		t.Fatalf("len(warnings) = %d, want at least 2", len(warnings))
	}

	redacted := cfg.RedactedJSON()
	if strings.Contains(redacted, "top-secret") {
		t.Fatalf("RedactedJSON() leaked the secret: %s", redacted)
	}
	if !strings.Contains(redacted, "[redacted]") {
		t.Fatalf("RedactedJSON() = %s, want redacted marker", redacted)
	}
}

func TestLoad_ProjectRootAndAgentModelFallback(t *testing.T) {
	paths, err := EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	content := []byte(`project_root = "relative/path"

[agents]
planner = "custom/planner-model"
coder = ""
tester = 42
`)
	if err := os.WriteFile(paths.ConfigPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, warnings, err := Load(paths)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Agents.Planner != "custom/planner-model" {
		t.Fatalf("cfg.Agents.Planner = %q, want custom override", cfg.Agents.Planner)
	}
	if cfg.Agents.Coder != DefaultCoderModel {
		t.Fatalf("cfg.Agents.Coder = %q, want %q", cfg.Agents.Coder, DefaultCoderModel)
	}
	if cfg.Agents.Tester != DefaultTesterModel {
		t.Fatalf("cfg.Agents.Tester = %q, want %q", cfg.Agents.Tester, DefaultTesterModel)
	}

	projectRoot := cfg.ProjectRootState()
	if !projectRoot.Configured || projectRoot.Valid {
		t.Fatalf("ProjectRootState() = %+v, want configured invalid state", projectRoot)
	}
	if !strings.Contains(projectRoot.Message, "absolute path") {
		t.Fatalf("ProjectRootState().Message = %q, want absolute-path guidance", projectRoot.Message)
	}
	if len(warnings) == 0 {
		t.Fatal("Load() warnings = 0, want warnings for invalid agent overrides")
	}
	if !containsWarning(warnings, "coder model override") {
		t.Fatalf("warnings = %v, want coder fallback warning", warnings)
	}
	if !containsWarning(warnings, "tester model override") {
		t.Fatalf("warnings = %v, want tester fallback warning", warnings)
	}
}

func TestProjectRootStateValidation(t *testing.T) {
	t.Parallel()

	fileRoot := filepath.Join(t.TempDir(), "relay.txt")
	if err := os.WriteFile(fileRoot, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	testCases := []struct {
		name           string
		projectRoot    string
		wantConfigured bool
		wantValid      bool
		wantMessage    string
	}{
		{
			name:           "missing project root",
			projectRoot:    "",
			wantConfigured: false,
			wantValid:      false,
			wantMessage:    "valid project_root",
		},
		{
			name:           "relative project root",
			projectRoot:    "relative/path",
			wantConfigured: true,
			wantValid:      false,
			wantMessage:    "absolute path",
		},
		{
			name:           "missing directory",
			projectRoot:    filepath.Join(t.TempDir(), "does-not-exist"),
			wantConfigured: true,
			wantValid:      false,
			wantMessage:    "could not read",
		},
		{
			name:           "file path",
			projectRoot:    fileRoot,
			wantConfigured: true,
			wantValid:      false,
			wantMessage:    "must point to a directory",
		},
		{
			name:           "plain directory",
			projectRoot:    t.TempDir(),
			wantConfigured: true,
			wantValid:      false,
			wantMessage:    "local Git repository root",
		},
		{
			name:           "git repository root",
			projectRoot:    initGitRepository(t),
			wantConfigured: true,
			wantValid:      true,
			wantMessage:    "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ProjectRoot = testCase.projectRoot

			status := cfg.ProjectRootState()
			if status.Configured != testCase.wantConfigured || status.Valid != testCase.wantValid {
				t.Fatalf("ProjectRootState() = %+v, want configured=%t valid=%t", status, testCase.wantConfigured, testCase.wantValid)
			}
			if testCase.wantMessage == "" {
				if status.Message != "" {
					t.Fatalf("ProjectRootState().Message = %q, want empty", status.Message)
				}
				return
			}
			if !strings.Contains(status.Message, testCase.wantMessage) {
				t.Fatalf("ProjectRootState().Message = %q, want substring %q", status.Message, testCase.wantMessage)
			}
		})
	}
}

func TestSaveAndLoadPreserveOpenRouterAndProjectRoot(t *testing.T) {
	paths, err := EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	projectRoot := initGitRepository(t)
	configToSave := DefaultConfig()
	configToSave.ProjectRoot = projectRoot
	configToSave.OpenRouter.APIKey = "or-test-key"

	if err := Save(paths, configToSave); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, warnings, err := Load(paths)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if loaded.ProjectRoot != projectRoot {
		t.Fatalf("loaded.ProjectRoot = %q, want %q", loaded.ProjectRoot, projectRoot)
	}
	if loaded.OpenRouter.APIKey != "or-test-key" {
		t.Fatalf("loaded.OpenRouter.APIKey = %q, want saved key", loaded.OpenRouter.APIKey)
	}
	preferences := loaded.SafePreferences()
	if !preferences.OpenRouterConfigured {
		t.Fatal("preferences.OpenRouterConfigured = false, want true")
	}
	if !preferences.ProjectRootConfigured || !preferences.ProjectRootValid {
		t.Fatalf("project root preferences = %+v, want configured valid root", preferences)
	}
	if preferences.ProjectRoot != projectRoot {
		t.Fatalf("preferences.ProjectRoot = %q, want %q", preferences.ProjectRoot, projectRoot)
	}
}


func TestModelContextLimitHint(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		model string
		want  *int
	}{
		{
			name:  "parses k suffix",
			model: "anthropic/claude-3.7-sonnet-32k",
			want:  intPtr(32000),
		},
		{
			name:  "parses m suffix",
			model: "google/gemini-2.0-flash-1m",
			want:  intPtr(1000000),
		},
		{
			name:  "ignores invalid hints",
			model: "anthropic/claude-sonnet-latest",
			want:  nil,
		},
		{
			name:  "ignores zero values",
			model: "custom/model-0k",
			want:  nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ModelContextLimitHint(testCase.model)
			if !equalIntPtr(got, testCase.want) {
				t.Fatalf("ModelContextLimitHint(%q) = %v, want %v", testCase.model, valueOfIntPtr(got), valueOfIntPtr(testCase.want))
			}
		})
	}
}

func intPtr(value int) *int {
	return &value
}

func equalIntPtr(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func valueOfIntPtr(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}

func initGitRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := git.PlainInit(root, false); err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	return root
}
