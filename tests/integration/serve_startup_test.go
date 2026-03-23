package integration_test

import (
	"errors"
	"net"
	"testing"

	"github.com/erisristemena/relay/internal/app"
)

func TestServeStartup_FallsBackToFreePort(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:4747")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer occupied.Close()

	server, _, launcher := newIntegrationServer(t, app.Options{PreferredPort: 4747, NoBrowser: false})
	status := server.StatusSnapshot()

	if status.ActivePort == 4747 {
		t.Fatalf("status.ActivePort = %d, want fallback port", status.ActivePort)
	}
	if launcher.openedURL == "" {
		t.Fatalf("launcher.openedURL = %q, want opened url", launcher.openedURL)
	}
}

func TestServeStartup_ReportsBrowserLaunchFailure(t *testing.T) {
	launcher := &stubLauncher{err: errors.New("launch blocked")}
	server, _, _ := newIntegrationServer(t, app.Options{PreferredPort: 4747, Launcher: launcher})
	status := server.StatusSnapshot()

	if status.BrowserLaunch != "failed" {
		t.Fatalf("status.BrowserLaunch = %q, want failed", status.BrowserLaunch)
	}
	if len(server.RuntimeEvents()) == 0 {
		t.Fatalf("RuntimeEvents() returned no events")
	}
}
