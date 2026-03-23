package integration_test

import "testing"

import "github.com/erisristemena/relay/internal/app"

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
