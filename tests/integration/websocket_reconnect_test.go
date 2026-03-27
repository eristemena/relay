package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/app"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func TestWorkspaceWebSocket_ReconnectBootstrapsAgain(t *testing.T) {
	server, _, _ := newIntegrationServer(t, app.Options{NoBrowser: true})

	first := dialWorkspace(t, server.BaseURL())
	writeMessage(t, first, map[string]any{
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	envelope := readUntilType(t, first, "workspace.bootstrap")
	if envelope["payload"] == nil {
		t.Fatalf("bootstrap payload missing on first connection")
	}
	_ = first.Close(1000, "reconnect")

	second := dialWorkspace(t, server.BaseURL())
	writeMessage(t, second, map[string]any{
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{},
	})
	envelope = readUntilType(t, second, "workspace.bootstrap")
	if envelope["payload"] == nil {
		t.Fatalf("bootstrap payload missing after reconnect")
	}
}

func TestWorkspaceWebSocket_ReconnectPreservesActiveRunIdentity(t *testing.T) {
	server, paths, _ := newIntegrationServer(t, app.Options{NoBrowser: true})

	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("sqlite.NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Reconnect orchestration")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Keep the active run visible", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	run.State = sqlite.RunStateActive
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeAgentSpawned, sqlite.RolePlanner, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","role":"planner","model":"`+run.Model+`","label":"Planner","spawn_order":1,"occurred_at":"`+time.Now().UTC().Format(time.RFC3339)+`"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() error = %v", err)
	}

	first := dialWorkspace(t, server.BaseURL())
	writeMessage(t, first, map[string]any{
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{
			"last_session_id": session.ID,
		},
	})
	firstBootstrap := readUntilType(t, first, "workspace.bootstrap")
	firstPayload := firstBootstrap["payload"].(map[string]any)
	if firstPayload["active_run_id"] != run.ID {
		t.Fatalf("first active_run_id = %v, want %q", firstPayload["active_run_id"], run.ID)
	}
	_ = first.Close(1000, "reconnect")

	second := dialWorkspace(t, server.BaseURL())
	writeMessage(t, second, map[string]any{
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{
			"last_session_id": session.ID,
		},
	})
	secondBootstrap := readUntilType(t, second, "workspace.bootstrap")
	secondPayload := secondBootstrap["payload"].(map[string]any)
	if secondPayload["active_run_id"] != run.ID {
		t.Fatalf("second active_run_id = %v, want %q", secondPayload["active_run_id"], run.ID)
	}
}
