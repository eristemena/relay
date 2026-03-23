package integration_test

import "testing"

import "github.com/erisristemena/relay/internal/app"

func TestWorkspaceSessions_CreatePersistAndOpen(t *testing.T) {
	server, _, _ := newIntegrationServer(t, app.Options{NoBrowser: true})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type": "session.create",
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
		"type": "workspace.bootstrap.request",
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
		"type": "session.open",
		"payload": map[string]any{"session_id": sessionID},
	})
	opened := readUntilType(t, reconnected, "session.opened")
	openedPayload := opened["payload"].(map[string]any)
	if openedPayload["active_session_id"].(string) != sessionID {
		t.Fatalf("opened active_session_id = %q, want %q", openedPayload["active_session_id"], sessionID)
	}
}
