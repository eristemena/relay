package integration_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	ws "github.com/erisristemena/relay/internal/handlers/ws"
	workspaceorchestrator "github.com/erisristemena/relay/internal/orchestrator/workspace"
	"github.com/erisristemena/relay/internal/storage/sqlite"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

const streamingIOTimeout = 15 * time.Second

var streamingEnvelopeBuffer sync.Map

func TestAgentStreaming_SubmitDeliversOrderedEventsAndRejectsSecondActiveRun(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	runner := &scriptedRunner{
		profile: agents.Profile{
			Role:         sqlite.RoleCoder,
			Model:        config.DefaultCoderModel,
			SystemPrompt: "test prompt",
			AllowedTools: []agents.ToolName{agents.ToolReadFile},
		},
		ready: make(chan struct{}),
		release: make(chan struct{}),
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	session, err := store.CreateSession(context.Background(), "Streaming session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Implement websocket integration coverage",
		},
	})

	bootstrap := readStreamingEnvelope(t, connection)
	if bootstrap["type"] != "workspace.bootstrap" {
		t.Fatalf("first message type = %v, want workspace.bootstrap", bootstrap["type"])
	}
	bootstrapPayload := bootstrap["payload"].(map[string]any)
	if bootstrapPayload["active_session_id"] != session.ID {
		t.Fatalf("bootstrap active_session_id = %v, want %q", bootstrapPayload["active_session_id"], session.ID)
	}

	<-runner.ready

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Attempt a second active run",
		},
	})

	rejection := readUntilStreamingType(t, connection, "error")
	rejectionPayload := rejection["payload"].(map[string]any)
	if rejectionPayload["code"] != "agent_run_submit_failed" {
		t.Fatalf("rejection code = %v, want agent_run_submit_failed", rejectionPayload["code"])
	}
	if !strings.Contains(rejectionPayload["message"].(string), "already has an active run") {
		t.Fatalf("rejection message = %q, want active-run guidance", rejectionPayload["message"])
	}

	close(runner.release)

	stateChange := readUntilStreamingType(t, connection, "state_change")
	tokenOne := readUntilStreamingType(t, connection, "token")
	tokenTwo := readUntilStreamingType(t, connection, "token")
	complete := readUntilStreamingType(t, connection, "complete")

	statePayload := stateChange["payload"].(map[string]any)
	tokenOnePayload := tokenOne["payload"].(map[string]any)
	tokenTwoPayload := tokenTwo["payload"].(map[string]any)
	completePayload := complete["payload"].(map[string]any)

	if statePayload["state"] != string(sqlite.RunStateThinking) {
		t.Fatalf("state change = %v, want %q", statePayload["state"], sqlite.RunStateThinking)
	}
	if tokenOnePayload["text"] != "alpha" {
		t.Fatalf("first token = %v, want alpha", tokenOnePayload["text"])
	}
	if tokenTwoPayload["text"] != "beta" {
		t.Fatalf("second token = %v, want beta", tokenTwoPayload["text"])
	}
	if _, ok := tokenOnePayload["first_token_latency_ms"]; !ok {
		t.Fatalf("first token payload missing first_token_latency_ms: %#v", tokenOnePayload)
	}
	if _, ok := tokenTwoPayload["first_token_latency_ms"]; ok {
		t.Fatalf("second token payload unexpectedly contained first_token_latency_ms: %#v", tokenTwoPayload)
	}
	if completePayload["finish_reason"] != "stop" {
		t.Fatalf("finish_reason = %v, want stop", completePayload["finish_reason"])
	}

	stateSequence := int(statePayload["sequence"].(float64))
	firstTokenSequence := int(tokenOnePayload["sequence"].(float64))
	secondTokenSequence := int(tokenTwoPayload["sequence"].(float64))
	completeSequence := int(completePayload["sequence"].(float64))
	if !(stateSequence < firstTokenSequence && firstTokenSequence < secondTokenSequence && secondTokenSequence < completeSequence) {
		t.Fatalf("unexpected sequence order: state=%d token1=%d token2=%d complete=%d", stateSequence, firstTokenSequence, secondTokenSequence, completeSequence)
	}
}

func TestAgentStreaming_OpenRunReattachesActiveStreamAfterReconnect(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	runner := &scriptedReconnectRunner{
		profile: agents.Profile{
			Role:         sqlite.RoleCoder,
			Model:        config.DefaultCoderModel,
			SystemPrompt: "test prompt",
			AllowedTools: []agents.ToolName{agents.ToolReadFile},
		},
		ready:   make(chan struct{}),
		stateReady: make(chan struct{}),
		release: make(chan struct{}),
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	session, err := store.CreateSession(context.Background(), "Reconnect session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	first := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, first, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Reconnect the live stream",
		},
	})

	_ = readUntilStreamingType(t, first, "workspace.bootstrap")
	close(runner.stateReady)
	state := readUntilStreamingType(t, first, "state_change")
	runID := state["payload"].(map[string]any)["run_id"].(string)
	<-runner.ready
	_ = first.Close(websocket.StatusGoingAway, "simulate reconnect")

	second := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, second, map[string]any{
		"type": "agent.run.open",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     runID,
		},
	})

	replayed := readUntilStreamingType(t, second, "state_change")
	if replayed["payload"].(map[string]any)["replay"] != true {
		t.Fatalf("replayed state_change replay = %v, want true", replayed["payload"].(map[string]any)["replay"])
	}

	close(runner.release)

	var sawToken, sawComplete bool
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !(sawToken && sawComplete) {
		envelope := readStreamingEnvelope(t, second)
		switch envelope["type"] {
		case "token":
			payload := envelope["payload"].(map[string]any)
			if payload["text"] == "after-reconnect" && payload["replay"] == false {
				sawToken = true
			}
		case "complete":
			payload := envelope["payload"].(map[string]any)
			if payload["replay"] == false {
				sawComplete = true
			}
		}
	}
	if !sawToken || !sawComplete {
		t.Fatalf("reattached stream missing events: token=%v complete=%v", sawToken, sawComplete)
	}
}

type scriptedRunner struct {
	profile agents.Profile
	ready   chan struct{}
	release chan struct{}
}

type scriptedReconnectRunner struct {
	profile agents.Profile
	ready   chan struct{}
	stateReady chan struct{}
	release chan struct{}
}

func (r *scriptedReconnectRunner) Profile() agents.Profile {
	return r.profile
}

func (r *scriptedReconnectRunner) Run(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
	<-r.stateReady
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(string(sqlite.RunStateThinking))
	}
	close(r.ready)
	<-r.release
	if handlers.OnToken != nil {
		handlers.OnToken("after-reconnect")
	}
	if handlers.OnComplete != nil {
		handlers.OnComplete("stop")
	}
	return nil
}

func (r *scriptedRunner) Profile() agents.Profile {
	return r.profile
}

func (r *scriptedRunner) Run(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
	close(r.ready)
	<-r.release
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(string(sqlite.RunStateThinking))
	}
	if handlers.OnToken != nil {
		handlers.OnToken("alpha")
		handlers.OnToken("beta")
	}
	if handlers.OnComplete != nil {
		handlers.OnComplete("stop")
	}
	return nil
}

func newStreamingTestService(t *testing.T) (*workspaceorchestrator.Service, *sqlite.Store, config.Paths) {
	t.Helper()

	paths, err := config.EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("sqlite.NewStore() error = %v", err)
	}

	return workspaceorchestrator.NewService(store, paths), store, paths
}

func newStreamingTestServer(t *testing.T, service *workspaceorchestrator.Service) *httptest.Server {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mux := http.NewServeMux()
	mux.Handle("/ws", ws.NewHandler(service, nil, logger))

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func dialStreamingSocket(t *testing.T, baseURL string) *websocket.Conn {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connection, _, err := websocket.Dial(ctx, websocketURL(baseURL), nil)
	if err != nil {
		t.Fatalf("websocket.Dial() error = %v", err)
	}
	t.Cleanup(func() {
		streamingEnvelopeBuffer.Delete(connection)
		_ = connection.Close(websocket.StatusNormalClosure, "done")
	})
	return connection
}

func writeStreamingMessage(t *testing.T, connection *websocket.Conn, payload map[string]any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), streamingIOTimeout)
	defer cancel()
	if err := wsjson.Write(ctx, connection, payload); err != nil {
		t.Fatalf("wsjson.Write() error = %v", err)
	}
}

func readStreamingEnvelope(t *testing.T, connection *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), streamingIOTimeout)
	defer cancel()

	var envelope map[string]any
	if err := wsjson.Read(ctx, connection, &envelope); err != nil {
		t.Fatalf("wsjson.Read() error = %v", err)
	}
	return envelope
}

func readUntilStreamingType(t *testing.T, connection *websocket.Conn, messageType string) map[string]any {
	t.Helper()
	if buffered, ok := popBufferedStreamingEnvelope(connection, messageType); ok {
		return buffered
	}

	deadline := time.Now().Add(streamingIOTimeout)
	for time.Now().Before(deadline) {
		envelope := readStreamingEnvelope(t, connection)
		if envelope["type"] == messageType {
			return envelope
		}
		pushBufferedStreamingEnvelope(connection, envelope)
	}
	t.Fatalf("timed out waiting for message type %q", messageType)
	return nil
}

func popBufferedStreamingEnvelope(connection *websocket.Conn, messageType string) (map[string]any, bool) {
	value, ok := streamingEnvelopeBuffer.Load(connection)
	if !ok {
		return nil, false
	}

	envelopes, ok := value.([]map[string]any)
	if !ok || len(envelopes) == 0 {
		streamingEnvelopeBuffer.Delete(connection)
		return nil, false
	}

	for index, envelope := range envelopes {
		if envelope["type"] != messageType {
			continue
		}
		next := append([]map[string]any{}, envelopes[:index]...)
		next = append(next, envelopes[index+1:]...)
		if len(next) == 0 {
			streamingEnvelopeBuffer.Delete(connection)
		} else {
			streamingEnvelopeBuffer.Store(connection, next)
		}
		return envelope, true
	}

	return nil, false
}

func pushBufferedStreamingEnvelope(connection *websocket.Conn, envelope map[string]any) {
	value, _ := streamingEnvelopeBuffer.Load(connection)
	envelopes, _ := value.([]map[string]any)
	envelopes = append(envelopes, envelope)
	streamingEnvelopeBuffer.Store(connection, envelopes)
}