package config

import (
	"os"
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
