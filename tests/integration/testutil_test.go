package integration_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eristemena/relay/internal/app"
	"github.com/eristemena/relay/internal/config"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type stubLauncher struct {
	openedURL string
	err       error
}

func (s *stubLauncher) Open(_ context.Context, targetURL string) error {
	s.openedURL = targetURL
	return s.err
}

func newIntegrationServer(t *testing.T, options app.Options) (*app.Server, config.Paths, *stubLauncher) {
	t.Helper()

	homeDir := t.TempDir()
	paths, err := config.EnsurePaths(homeDir)
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	launcher := &stubLauncher{}
	if options.Launcher == nil {
		options.Launcher = launcher
	}
	if options.Logger == nil {
		options.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	options.HomeDir = homeDir

	ctx, cancel := context.WithCancel(context.Background())
	server, err := app.NewServer(ctx, options)
	if err != nil {
		cancel()
		t.Fatalf("NewServer() error = %v", err)
	}

	go func() {
		_ = server.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		_ = server.Close()
	})

	waitForHealth(t, server.BaseURL())
	return server, paths, launcher
}

func waitForHealth(t *testing.T, baseURL string) {
	t.Helper()
	client := &http.Client{Timeout: 250 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		response, err := client.Get(baseURL + "/api/healthz")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("health endpoint never became ready at %s", baseURL)
}

func websocketURL(baseURL string) string {
	return "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
}

func dialWorkspace(t *testing.T, baseURL string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connection, _, err := websocket.Dial(ctx, websocketURL(baseURL), nil)
	if err != nil {
		t.Fatalf("websocket.Dial() error = %v", err)
	}
	t.Cleanup(func() {
		_ = connection.Close(websocket.StatusNormalClosure, "test complete")
	})
	return connection
}

func writeMessage(t *testing.T, connection *websocket.Conn, payload map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, connection, payload); err != nil {
		t.Fatalf("wsjson.Write() error = %v", err)
	}
}

func readUntilType(t *testing.T, connection *websocket.Conn, messageType string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		var envelope map[string]any
		if err := wsjson.Read(ctx, connection, &envelope); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("timed out waiting for websocket message %q", messageType)
			}
			t.Fatalf("wsjson.Read() error = %v", err)
		}
		if fmt.Sprint(envelope["type"]) == messageType {
			return envelope
		}
	}
}
