package main

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/eristemena/relay/internal/app"
)

func TestServeCommandPassesStartupRoot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	originalRunRelay := runRelay
	t.Cleanup(func() {
		runRelay = originalRunRelay
	})

	var captured app.Options
	runRelay = func(_ context.Context, options app.Options) error {
		captured = options
		return nil
	}

	command := newRootCommand(logger)
	command.SetArgs([]string{"serve", "--root", "/tmp/project-root", "--no-browser"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.ProjectRoot != "/tmp/project-root" {
		t.Fatalf("captured.ProjectRoot = %q, want /tmp/project-root", captured.ProjectRoot)
	}
	if !captured.NoBrowser {
		t.Fatal("captured.NoBrowser = false, want true")
	}
}

func TestServeCommandAcceptsDeprecatedProjectRootFlag(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	originalRunRelay := runRelay
	t.Cleanup(func() {
		runRelay = originalRunRelay
	})

	var captured app.Options
	runRelay = func(_ context.Context, options app.Options) error {
		captured = options
		return nil
	}

	command := newRootCommand(logger)
	command.SetArgs([]string{"serve", "--project-root", "/tmp/legacy-root"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if captured.ProjectRoot != "/tmp/legacy-root" {
		t.Fatalf("captured.ProjectRoot = %q, want /tmp/legacy-root", captured.ProjectRoot)
	}
}