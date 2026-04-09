package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveProjectRoot(t *testing.T) {
	t.Run("prefers explicit root", func(t *testing.T) {
		workingDir := t.TempDir()
		explicitRoot := t.TempDir()

		resolved, err := ResolveProjectRoot(explicitRoot, workingDir)
		if err != nil {
			t.Fatalf("ResolveProjectRoot() error = %v", err)
		}
		if resolved != filepath.Clean(explicitRoot) {
			t.Fatalf("ResolveProjectRoot() = %q, want %q", resolved, filepath.Clean(explicitRoot))
		}
	})

	t.Run("falls back to working directory", func(t *testing.T) {
		workingDir := t.TempDir()

		resolved, err := ResolveProjectRoot("", workingDir)
		if err != nil {
			t.Fatalf("ResolveProjectRoot() error = %v", err)
		}
		if resolved != filepath.Clean(workingDir) {
			t.Fatalf("ResolveProjectRoot() = %q, want %q", resolved, filepath.Clean(workingDir))
		}
	})

	t.Run("rejects missing root", func(t *testing.T) {
		if _, err := ResolveProjectRoot("", ""); err == nil {
			t.Fatal("ResolveProjectRoot() error = nil, want error")
		}
	})

	t.Run("rejects non-directory path", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(filePath, []byte("relay"), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}

		if _, err := ResolveProjectRoot(filePath, ""); err == nil {
			t.Fatal("ResolveProjectRoot() error = nil, want error")
		}
	})
}

func TestProjectRootLabel(t *testing.T) {
	if got := ProjectRootLabel("/tmp/relay"); got != "relay" {
		t.Fatalf("ProjectRootLabel() = %q, want relay", got)
	}
	if got := ProjectRootLabel(""); got != "" {
		t.Fatalf("ProjectRootLabel() = %q, want empty", got)
	}
}