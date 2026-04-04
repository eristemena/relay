package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eristemena/relay/internal/agents"
	"github.com/eristemena/relay/internal/app"
	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
)

func TestRepositoryFileTree_RequestHydratesConnectedRepositoryTree(t *testing.T) {
	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.MkdirAll(filepath.Join(repoRoot, "cmd", "relay"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("relay\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "cmd", "relay", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	server, _, _ := newIntegrationServer(t, app.Options{
		NoBrowser:   true,
		ProjectRoot: repoRoot,
	})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	_ = readUntilType(t, connection, "workspace.bootstrap")

	writeMessage(t, connection, map[string]any{
		"type": "repository.tree.request",
		"payload": map[string]any{
			"session_id": "",
		},
	})

	treeResult := readUntilType(t, connection, "repository.tree.result")
	payload := treeResult["payload"].(map[string]any)
	if payload["repository_root"] != repoRoot {
		t.Fatalf("repository_root = %v, want %q", payload["repository_root"], repoRoot)
	}
	if payload["status"] != "ready" {
		t.Fatalf("status = %v, want ready", payload["status"])
	}
	paths := payload["paths"].([]any)
	if len(paths) < 3 {
		t.Fatalf("len(paths) = %d, want at least 3", len(paths))
	}
	assertContainsPath(t, paths, "README.md")
	assertContainsPath(t, paths, "cmd")
	assertContainsPath(t, paths, "cmd/relay")
	assertContainsPath(t, paths, "cmd/relay/main.go")
	if touchedFiles, ok := payload["touched_files"]; ok && touchedFiles != nil {
		if items, ok := touchedFiles.([]any); !ok || len(items) != 0 {
			t.Fatalf("touched_files = %#v, want omitted or empty", touchedFiles)
		}
	}
}

func TestRepositoryFileTree_StreamsTouchEventsAndReconnectRestoresTouchedSnapshot(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = repoRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Repository tree streaming")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
		repoRoot:     repoRoot,
		writePath:    "README.md",
		writeContent: "after\n",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Emit repository tree activity",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	_ = readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	_ = readUntilStreamingType(t, connection, "tool_call")
	fileTouched := readUntilStreamingType(t, connection, "file_touched")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")

	touchPayload := fileTouched["payload"].(map[string]any)
	if touchPayload["run_id"] != runID {
		t.Fatalf("file_touched.run_id = %v, want %q", touchPayload["run_id"], runID)
	}
	if touchPayload["agent_id"] != string(sqlite.RoleCoder) {
		t.Fatalf("file_touched.agent_id = %v, want %q", touchPayload["agent_id"], sqlite.RoleCoder)
	}
	if touchPayload["file_path"] != "README.md" {
		t.Fatalf("file_touched.file_path = %v, want README.md", touchPayload["file_path"])
	}
	if touchPayload["touch_type"] != sqlite.TouchTypeProposed {
		t.Fatalf("file_touched.touch_type = %v, want %q", touchPayload["touch_type"], sqlite.TouchTypeProposed)
	}
	if touchPayload["replay"] != false {
		t.Fatalf("file_touched.replay = %v, want false", touchPayload["replay"])
	}

	approvalPayload := approvalRequest["payload"].(map[string]any)
	if approvalPayload["tool_call_id"] != "call_write" {
		t.Fatalf("approval_request.tool_call_id = %v, want call_write", approvalPayload["tool_call_id"])
	}

	if err := store.RecordTouchedFile(context.Background(), sqlite.TouchedFile{
		RunID:      runID,
		AgentID:    string(sqlite.RoleCoder),
		FilePath:   "README.md",
		TouchType:  sqlite.TouchTypeRead,
		RecordedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RecordTouchedFile(read) error = %v", err)
	}

	reconnected := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, reconnected, map[string]any{
		"type": "repository.tree.request",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     runID,
		},
	})

	treeResult := readUntilStreamingType(t, reconnected, "repository.tree.result")
	treePayload := treeResult["payload"].(map[string]any)
	if treePayload["repository_root"] != repoRoot {
		t.Fatalf("repository_root = %v, want %q", treePayload["repository_root"], repoRoot)
	}
	touchedFiles := treePayload["touched_files"].([]any)
	if len(touchedFiles) != 2 {
		t.Fatalf("len(touched_files) = %d, want 2", len(touchedFiles))
	}
	assertTouchedFileKinds(t, touchedFiles, "README.md", []string{sqlite.TouchTypeProposed, sqlite.TouchTypeRead})
	treePaths := treePayload["paths"].([]any)
	assertContainsPath(t, treePaths, "README.md")
}

func TestRepositoryFileTree_PreservesAgentAttributionForFiltering(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.MkdirAll(filepath.Join(repoRoot, "docs"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "docs", "review.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(review.md) error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = repoRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Repository tree agent filtering")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
		repoRoot:     repoRoot,
		writePath:    "README.md",
		writeContent: "after\n",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Emit attributed repository tree activity",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	_ = readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	_ = readUntilStreamingType(t, connection, "tool_call")
	fileTouched := readUntilStreamingType(t, connection, "file_touched")
	_ = readUntilStreamingType(t, connection, "approval_request")

	touchPayload := fileTouched["payload"].(map[string]any)
	if touchPayload["agent_id"] != string(sqlite.RoleCoder) {
		t.Fatalf("file_touched.agent_id = %v, want %q", touchPayload["agent_id"], sqlite.RoleCoder)
	}
	if touchPayload["file_path"] != "README.md" {
		t.Fatalf("file_touched.file_path = %v, want README.md", touchPayload["file_path"])
	}

	if err := store.RecordTouchedFile(context.Background(), sqlite.TouchedFile{
		RunID:      runID,
		AgentID:    "agent_reviewer_1",
		FilePath:   "docs/review.md",
		TouchType:  sqlite.TouchTypeRead,
		RecordedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("RecordTouchedFile(reviewer) error = %v", err)
	}

	reconnected := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, reconnected, map[string]any{
		"type": "repository.tree.request",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     runID,
		},
	})

	treeResult := readUntilStreamingType(t, reconnected, "repository.tree.result")
	treePayload := treeResult["payload"].(map[string]any)
	touchedFiles := treePayload["touched_files"].([]any)
	assertTouchedFileEntry(t, touchedFiles, "README.md", string(sqlite.RoleCoder), sqlite.TouchTypeProposed)
	assertTouchedFileEntry(t, touchedFiles, "docs/review.md", "agent_reviewer_1", sqlite.TouchTypeRead)
	treePaths := treePayload["paths"].([]any)
	assertContainsPath(t, treePaths, "docs/review.md")
}

func assertContainsPath(t *testing.T, paths []any, want string) {
	t.Helper()
	for _, path := range paths {
		if path == want {
			return
		}
	}
	t.Fatalf("paths = %#v, want %q", paths, want)
}

func assertTouchedFileKinds(t *testing.T, touchedFiles []any, filePath string, wantKinds []string) {
	t.Helper()
	seen := make(map[string]bool, len(wantKinds))
	for _, item := range touchedFiles {
		touchedFile := item.(map[string]any)
		if touchedFile["file_path"] != filePath {
			continue
		}
		kind, _ := touchedFile["touch_type"].(string)
		seen[kind] = true
	}
	for _, kind := range wantKinds {
		if !seen[kind] {
			t.Fatalf("touched_files = %#v, want %q for %s", touchedFiles, kind, filePath)
		}
	}
}

func assertTouchedFileEntry(t *testing.T, touchedFiles []any, filePath string, agentID string, touchType string) {
	t.Helper()
	for _, item := range touchedFiles {
		touchedFile := item.(map[string]any)
		if touchedFile["file_path"] == filePath && touchedFile["agent_id"] == agentID && touchedFile["touch_type"] == touchType {
			return
		}
	}
	t.Fatalf("touched_files = %#v, want file=%q agent=%q touch_type=%q", touchedFiles, filePath, agentID, touchType)
}