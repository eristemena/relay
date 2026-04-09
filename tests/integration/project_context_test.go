package integration_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/eristemena/relay/internal/app"
	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
	git "github.com/go-git/go-git/v5"
)

func TestProjectContext_ProjectSwitchRequestSwitchesKnownProjects(t *testing.T) {
	homeDir := t.TempDir()
	projectARoot := filepath.Join(homeDir, "relay-project-a")
	projectBRoot := filepath.Join(homeDir, "relay-project-b")
	for _, root := range []string{projectARoot, projectBRoot} {
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", root, err)
		}
		if _, err := git.PlainInit(root, false); err != nil {
			t.Fatalf("git.PlainInit(%s) error = %v", root, err)
		}
	}

	paths, err := config.EnsurePaths(homeDir)
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}
	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("sqlite.NewStore() error = %v", err)
	}

	ctx := context.Background()
	sessionA, err := store.CreateProjectSession(ctx, "relay-project-a", projectARoot)
	if err != nil {
		store.Close()
		t.Fatalf("CreateProjectSession(project A) error = %v", err)
	}
	sessionB, err := store.CreateProjectSession(ctx, "relay-project-b", projectBRoot)
	if err != nil {
		store.Close()
		t.Fatalf("CreateProjectSession(project B) error = %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}

	cfg, warnings, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("config warnings = %v, want none", warnings)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = projectARoot
	cfg.LastSessionID = sessionA.ID
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server, err := app.NewServer(ctx, app.Options{
		HomeDir:     homeDir,
		ProjectRoot: projectARoot,
		NoBrowser:   true,
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		cancel()
		t.Fatalf("app.NewServer() error = %v", err)
	}
	go func() {
		_ = server.Run(ctx)
	}()
	waitForHealth(t, server.BaseURL())
	defer func() {
		cancel()
		_ = server.Close()
	}()

	connection := dialWorkspace(t, server.BaseURL())
	writeMessage(t, connection, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	bootstrap := readUntilType(t, connection, "workspace.bootstrap")
	payload := bootstrap["payload"].(map[string]any)
	if payload["active_project_root"] != projectARoot {
		t.Fatalf("active_project_root = %v, want %q", payload["active_project_root"], projectARoot)
	}
	knownProjects := payload["known_projects"].([]any)
	if len(knownProjects) != 2 {
		t.Fatalf("len(known_projects) = %d, want 2", len(knownProjects))
	}

	writeMessage(t, connection, map[string]any{
		"type": "project.switch.request",
		"payload": map[string]any{
			"project_root": projectBRoot,
		},
	})
	switched := readUntilType(t, connection, "workspace.bootstrap")
	switchedPayload := switched["payload"].(map[string]any)
	if switchedPayload["active_project_root"] != projectBRoot {
		t.Fatalf("switched active_project_root = %v, want %q", switchedPayload["active_project_root"], projectBRoot)
	}
	if switchedPayload["active_session_id"] != sessionB.ID {
		t.Fatalf("switched active_session_id = %v, want %q", switchedPayload["active_session_id"], sessionB.ID)
	}
	connectedRepository := switchedPayload["connected_repository"].(map[string]any)
	if connectedRepository["path"] != projectBRoot {
		t.Fatalf("connected_repository.path = %v, want %q", connectedRepository["path"], projectBRoot)
	}
	if connectedRepository["status"] != "connected" {
		t.Fatalf("connected_repository.status = %v, want connected", connectedRepository["status"])
	}
	projects := switchedPayload["known_projects"].([]any)
	activeCount := 0
	for _, item := range projects {
		project := item.(map[string]any)
		if project["is_active"] == true {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("active known project count = %d, want 1", activeCount)
	}
}
