package integration_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/app"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func TestRunHistoryReplay_RestartHydratesBootstrapAndReplaysRun(t *testing.T) {
	homeDir := t.TempDir()

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
	session, err := store.CreateSession(ctx, "Replay after restart")
	if err != nil {
		store.Close()
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Review the saved run", sqlite.RoleReviewer, config.DefaultReviewerModel)
	if err != nil {
		store.Close()
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","state":"thinking","message":"thinking","occurred_at":"2026-03-23T12:00:00Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() state error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","text":"alpha","occurred_at":"2026-03-23T12:00:01Z"}`); err != nil {
		store.Close()
		t.Fatalf("AppendRunEvent() token error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		store.Close()
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeComplete, run.Role, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","finish_reason":"stop","occurred_at":"2026-03-23T12:00:02Z"}`); err != nil {
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
	cfg.ProjectRoot = homeDir
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
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	bootstrap := readUntilType(t, connection, "workspace.bootstrap")
	payload := bootstrap["payload"].(map[string]any)
	if payload["active_session_id"] != session.ID {
		t.Fatalf("active_session_id = %v, want %q", payload["active_session_id"], session.ID)
	}
	if payload["active_run_id"] != nil {
		t.Fatalf("active_run_id = %v, want nil for completed replay-only run", payload["active_run_id"])
	}
	credentialStatus := payload["credential_status"].(map[string]any)
	if credentialStatus["configured"] != true {
		t.Fatalf("credential_status.configured = %v, want true", credentialStatus["configured"])
	}
	preferences := payload["preferences"].(map[string]any)
	if preferences["openrouter_configured"] != true {
		t.Fatalf("preferences.openrouter_configured = %v, want true", preferences["openrouter_configured"])
	}
	if preferences["project_root"] != homeDir {
		t.Fatalf("preferences.project_root = %v, want %q", preferences["project_root"], homeDir)
	}
	if preferences["project_root_valid"] != true {
		t.Fatalf("preferences.project_root_valid = %v, want true", preferences["project_root_valid"])
	}
	runSummaries := payload["run_summaries"].([]any)
	if len(runSummaries) != 1 {
		t.Fatalf("len(run_summaries) = %d, want 1", len(runSummaries))
	}
	if runSummaries[0].(map[string]any)["id"] != run.ID {
		t.Fatalf("run_summaries[0].id = %v, want %q", runSummaries[0].(map[string]any)["id"], run.ID)
	}

	writeMessage(t, connection, map[string]any{
		"type": "agent.run.open",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     run.ID,
		},
	})

	stateChange := readUntilType(t, connection, "state_change")
	token := readUntilType(t, connection, "token")
	complete := readUntilType(t, connection, "complete")

	statePayload := stateChange["payload"].(map[string]any)
	tokenPayload := token["payload"].(map[string]any)
	completePayload := complete["payload"].(map[string]any)
	if statePayload["replay"] != true || tokenPayload["replay"] != true || completePayload["replay"] != true {
		t.Fatalf("replay flags = state:%v token:%v complete:%v, want all true", statePayload["replay"], tokenPayload["replay"], completePayload["replay"])
	}
	if statePayload["sequence"] != float64(1) || tokenPayload["sequence"] != float64(2) || completePayload["sequence"] != float64(3) {
		t.Fatalf("sequences = [%v, %v, %v], want [1, 2, 3]", statePayload["sequence"], tokenPayload["sequence"], completePayload["sequence"])
	}
	if tokenPayload["text"] != "alpha" {
		t.Fatalf("token text = %v, want alpha", tokenPayload["text"])
	}
	if completePayload["finish_reason"] != "stop" {
		t.Fatalf("finish_reason = %v, want stop", completePayload["finish_reason"])
	}
}

func startServerAtHome(t *testing.T, homeDir string) (*app.Server, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	server, err := app.NewServer(ctx, app.Options{
		HomeDir:   homeDir,
		NoBrowser: true,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		cancel()
		t.Fatalf("app.NewServer() error = %v", err)
	}
	go func() {
		_ = server.Run(ctx)
	}()
	waitForHealth(t, server.BaseURL())
	return server, cancel
}