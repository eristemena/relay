package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/app"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	git "github.com/go-git/go-git/v5"
)

func TestWorkspaceSessions_CreatePersistAndOpen(t *testing.T) {
	server, _, _ := newIntegrationServer(t, app.Options{NoBrowser: true})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type":    "session.create",
		"payload": map[string]any{"display_name": "Investigate startup"},
	})
	created := readUntilType(t, connection, "session.created")
	payload := created["payload"].(map[string]any)
	sessions := payload["sessions"].([]any)
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}

	reconnected := dialWorkspace(t, server.BaseURL())
	writeMessage(t, reconnected, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	bootstrapped := readUntilType(t, reconnected, "workspace.bootstrap")
	bootstrapPayload := bootstrapped["payload"].(map[string]any)
	bootstrappedSessions := bootstrapPayload["sessions"].([]any)
	if len(bootstrappedSessions) != 1 {
		t.Fatalf("len(bootstrappedSessions) = %d, want 1", len(bootstrappedSessions))
	}

	sessionID := bootstrapPayload["active_session_id"].(string)
	writeMessage(t, reconnected, map[string]any{
		"type":    "session.open",
		"payload": map[string]any{"session_id": sessionID},
	})
	opened := readUntilType(t, reconnected, "session.opened")
	openedPayload := opened["payload"].(map[string]any)
	if openedPayload["active_session_id"].(string) != sessionID {
		t.Fatalf("opened active_session_id = %q, want %q", openedPayload["active_session_id"], sessionID)
	}
}

func TestWorkspaceSessions_BootstrapReconnectPreservesConnectedRepository(t *testing.T) {
	parentDir := t.TempDir()
	repoDir := filepath.Join(parentDir, "relay-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(repoDir) error = %v", err)
	}
	if _, err := git.PlainInit(repoDir, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}

	server, _, _ := newIntegrationServer(t, app.Options{
		NoBrowser:   true,
		ProjectRoot: repoDir,
	})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	bootstrap := readUntilType(t, connection, "workspace.bootstrap")
	payload := bootstrap["payload"].(map[string]any)
	connectedRepository := payload["connected_repository"].(map[string]any)
	if connectedRepository["path"] != repoDir {
		t.Fatalf("connected_repository.path = %v, want %q", connectedRepository["path"], repoDir)
	}
	if connectedRepository["status"] != "connected" {
		t.Fatalf("connected_repository.status = %v, want connected", connectedRepository["status"])
	}

	reconnected := dialWorkspace(t, server.BaseURL())
	writeMessage(t, reconnected, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	reconnectedBootstrap := readUntilType(t, reconnected, "workspace.bootstrap")
	reconnectedPayload := reconnectedBootstrap["payload"].(map[string]any)
	reconnectedRepository := reconnectedPayload["connected_repository"].(map[string]any)
	if reconnectedRepository["path"] != repoDir {
		t.Fatalf("reconnected connected_repository.path = %v, want %q", reconnectedRepository["path"], repoDir)
	}
	if reconnectedRepository["status"] != "connected" {
		t.Fatalf("reconnected connected_repository.status = %v, want connected", reconnectedRepository["status"])
	}
}

func TestWorkspaceSessions_RestartOpenRunReplaysRepositoryAwareActivity(t *testing.T) {
	homeDir := t.TempDir()
	repoDir := filepath.Join(homeDir, "relay-repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(repoDir) error = %v", err)
	}
	if _, err := git.PlainInit(repoDir, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}

	firstServer, firstCancel := startServerAtHome(t, homeDir)
	firstCancel()
	_ = firstServer.Close()

	paths, err := config.EnsurePaths(homeDir)
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}
	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("sqlite.NewStore() error = %v", err)
	}

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay repository activity after restart")
	if err != nil {
		store.Close()
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay repository-aware activity", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		store.Close()
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		store.Close()
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToolCall, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","role":"coder","model":"`+run.Model+`","tool_call_id":"call_read","tool_name":"read_file","input_preview":{"path":"README.md"},"occurred_at":"2026-03-24T12:00:00Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() read tool_call error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToolResult, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","role":"coder","model":"`+run.Model+`","tool_call_id":"call_read","tool_name":"read_file","status":"completed","result_preview":{"summary":"Loaded file content."},"occurred_at":"2026-03-24T12:00:01Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() read tool_result error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToolCall, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","role":"coder","model":"`+run.Model+`","tool_call_id":"call_write","tool_name":"write_file","input_preview":{"path":"README.md","request_kind":"file_write","repository_root":"`+repoDir+`","diff_preview":{"target_path":"README.md","original_content":"before\n","proposed_content":"after\n","base_content_hash":"sha256:abc"}},"occurred_at":"2026-03-24T12:00:02Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() write tool_call error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeApprovalStateChanged, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","role":"coder","model":"`+run.Model+`","tool_call_id":"call_write","tool_name":"write_file","status":"approved","message":"Tool approved. Relay is resuming the run.","occurred_at":"2026-03-24T12:00:03Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() approval approved error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeApprovalStateChanged, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","role":"coder","model":"`+run.Model+`","tool_call_id":"call_write","tool_name":"write_file","status":"applied","message":"Relay applied the approved change.","occurred_at":"2026-03-24T12:00:04Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() approval applied error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToolResult, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","role":"coder","model":"`+run.Model+`","tool_call_id":"call_write","tool_name":"write_file","status":"completed","result_preview":{"summary":"Wrote file content."},"occurred_at":"2026-03-24T12:00:05Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() write tool_result error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeComplete, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","finish_reason":"stop","occurred_at":"2026-03-24T12:00:06Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() complete error = %v", err)
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
	cfg.ProjectRoot = repoDir
	cfg.LastSessionID = session.ID
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	secondServer, secondCancel := startServerAtHome(t, homeDir)
	defer func() {
		secondCancel()
		_ = secondServer.Close()
	}()

	connection := dialWorkspace(t, secondServer.BaseURL())
	writeMessage(t, connection, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	bootstrap := readUntilType(t, connection, "workspace.bootstrap")
	bootstrapPayload := bootstrap["payload"].(map[string]any)
	connectedRepository := bootstrapPayload["connected_repository"].(map[string]any)
	if connectedRepository["path"] != repoDir {
		t.Fatalf("connected_repository.path = %v, want %q", connectedRepository["path"], repoDir)
	}
	if connectedRepository["status"] != "connected" {
		t.Fatalf("connected_repository.status = %v, want connected", connectedRepository["status"])
	}

	writeMessage(t, connection, map[string]any{
		"type": "agent.run.open",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     run.ID,
		},
	})

	readCall := readUntilType(t, connection, "tool_call")
	_ = readUntilType(t, connection, "tool_result")
	writeCall := readUntilType(t, connection, "tool_call")
	approved := readUntilType(t, connection, "approval_state_changed")
	applied := readUntilType(t, connection, "approval_state_changed")
	writeResult := readUntilType(t, connection, "tool_result")
	complete := readUntilType(t, connection, "complete")

	replayed := []map[string]any{readCall, writeCall, approved, applied, writeResult, complete}
	for _, envelope := range replayed {
		payload := envelope["payload"].(map[string]any)
		if payload["replay"] != true {
			t.Fatalf("replayed payload = %#v, want replay=true", payload)
		}
	}
	if readCall["payload"].(map[string]any)["input_preview"].(map[string]any)["path"] != "README.md" {
		t.Fatalf("read call input_preview = %#v, want README.md", readCall["payload"].(map[string]any)["input_preview"])
	}
	writePreview := writeCall["payload"].(map[string]any)["input_preview"].(map[string]any)
	if writePreview["repository_root"] != repoDir {
		t.Fatalf("write call repository_root = %v, want %q", writePreview["repository_root"], repoDir)
	}
	if writePreview["diff_preview"].(map[string]any)["target_path"] != "README.md" {
		t.Fatalf("write call diff_preview.target_path = %v, want README.md", writePreview["diff_preview"].(map[string]any)["target_path"])
	}
	if approved["payload"].(map[string]any)["status"] != "approved" {
		t.Fatalf("approved status = %v, want approved", approved["payload"].(map[string]any)["status"])
	}
	if applied["payload"].(map[string]any)["status"] != "applied" {
		t.Fatalf("applied status = %v, want applied", applied["payload"].(map[string]any)["status"])
	}
	if writeResult["payload"].(map[string]any)["status"] != "completed" {
		t.Fatalf("write result status = %v, want completed", writeResult["payload"].(map[string]any)["status"])
	}
}
