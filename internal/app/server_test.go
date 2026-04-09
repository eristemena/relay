package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eristemena/relay/internal/config"
)

func TestNewServerUsesExplicitStartupRoot(t *testing.T) {
	homeDir := t.TempDir()
	projectRoot := t.TempDir()
	server, err := NewServer(context.Background(), Options{
		HomeDir:    homeDir,
		NoBrowser:  true,
		ProjectRoot: projectRoot,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	defer server.Close()

	loaded, _, err := config.Load(server.configPaths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if loaded.ProjectRoot != filepath.Clean(projectRoot) {
		t.Fatalf("loaded.ProjectRoot = %q, want %q", loaded.ProjectRoot, filepath.Clean(projectRoot))
	}
}

func TestNewServerFallsBackToWorkingDirectory(t *testing.T) {
	homeDir := t.TempDir()
	workingDir := t.TempDir()
	originalGetWorkingDirectory := getWorkingDirectory
	t.Cleanup(func() {
		getWorkingDirectory = originalGetWorkingDirectory
	})
	getWorkingDirectory = func() (string, error) {
		return workingDir, nil
	}

	server, err := NewServer(context.Background(), Options{
		HomeDir:   homeDir,
		NoBrowser: true,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	defer server.Close()

	loaded, _, err := config.Load(server.configPaths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if loaded.ProjectRoot != filepath.Clean(workingDir) {
		t.Fatalf("loaded.ProjectRoot = %q, want %q", loaded.ProjectRoot, filepath.Clean(workingDir))
	}
}

func TestNewServerRejectsInvalidStartupRoot(t *testing.T) {
	homeDir := t.TempDir()
	missingRoot := filepath.Join(t.TempDir(), "missing-root")
	_, err := NewServer(context.Background(), Options{
		HomeDir:    homeDir,
		NoBrowser:  true,
		ProjectRoot: missingRoot,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err == nil {
		t.Fatal("NewServer() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "relay could not start with the selected project root") {
		t.Fatalf("NewServer() error = %q, want startup root guidance", err)
	}
	if !strings.Contains(err.Error(), "validate project root") {
		t.Fatalf("NewServer() error = %q, want validate project root detail", err)
	}
}

func TestGetenvUsesFallbackForBlankValues(t *testing.T) {
	t.Setenv("RELAY_TEST_ENV", "")
	if got := getenv("RELAY_TEST_ENV", "fallback"); got != "fallback" {
		t.Fatalf("getenv() = %q, want fallback", got)
	}

	t.Setenv("RELAY_TEST_ENV", "value")
	if got := getenv("RELAY_TEST_ENV", "fallback"); got != "value" {
		t.Fatalf("getenv() = %q, want value", got)
	}

	_ = os.Unsetenv("RELAY_TEST_ENV")
}