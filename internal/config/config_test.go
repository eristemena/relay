package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestProjectRootStateReportsMissingAndUnreadableRoots(t *testing.T) {
	missing := DefaultConfig().ProjectRootState()
	if missing.Configured || missing.Valid {
		t.Fatalf("missing ProjectRootState() = %+v, want unconfigured invalid state", missing)
	}
	if !strings.Contains(missing.Message, "valid project_root") {
		t.Fatalf("missing ProjectRootState().Message = %q, want setup guidance", missing.Message)
	}

	unreadable := DefaultConfig()
	unreadable.ProjectRoot = filepath.Join(t.TempDir(), "does-not-exist")
	status := unreadable.ProjectRootState()
	if !status.Configured || status.Valid {
		t.Fatalf("unreadable ProjectRootState() = %+v, want configured invalid state", status)
	}
	if !strings.Contains(status.Message, "could not read") {
		t.Fatalf("unreadable ProjectRootState().Message = %q, want unreadable guidance", status.Message)
	}

	valid := DefaultConfig()
	valid.ProjectRoot = t.TempDir()
	validStatus := valid.ProjectRootState()
	if !validStatus.Configured || !validStatus.Valid {
		t.Fatalf("valid ProjectRootState() = %+v, want configured valid state", validStatus)
	}
}

func TestSaveAndLoadPreserveOpenRouterAndProjectRoot(t *testing.T) {
	paths, err := EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	projectRoot := t.TempDir()
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

func containsWarning(warnings []string, needle string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return true
		}
	}
	return false
}
