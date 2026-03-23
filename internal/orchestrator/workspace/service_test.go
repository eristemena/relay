package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func TestService_BootstrapUsesSavedSession(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	created, err := store.CreateSession(ctx, "Resume me")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.LastSessionID = created.ID
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	snapshot, err := service.Bootstrap(ctx, "")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if snapshot.ActiveSessionID != created.ID {
		t.Fatalf("snapshot.ActiveSessionID = %q, want %q", snapshot.ActiveSessionID, created.ID)
	}
	if len(snapshot.Sessions) != 1 {
		t.Fatalf("len(snapshot.Sessions) = %d, want 1", len(snapshot.Sessions))
	}
}

func TestService_CreateSessionAndOpenSessionPersistSelection(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()

	createdSnapshot, err := service.CreateSession(ctx, "Fresh session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if createdSnapshot.ActiveSessionID == "" {
		t.Fatal("CreateSession() returned empty ActiveSessionID")
	}

	archived, err := store.CreateSession(ctx, "Archived session")
	if err != nil {
		t.Fatalf("CreateSession() store error = %v", err)
	}

	openedSnapshot, err := service.OpenSession(ctx, archived.ID)
	if err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if openedSnapshot.ActiveSessionID != archived.ID {
		t.Fatalf("openedSnapshot.ActiveSessionID = %q, want %q", openedSnapshot.ActiveSessionID, archived.ID)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.LastSessionID != archived.ID {
		t.Fatalf("cfg.LastSessionID = %q, want %q", cfg.LastSessionID, archived.ID)
	}
}

func TestService_SetRunnerFactoryAndEnvelopeHelpers(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	originalFactory := reflect.ValueOf(service.runnerFactory).Pointer()
	service.SetRunnerFactory(nil)
	if reflect.ValueOf(service.runnerFactory).Pointer() != originalFactory {
		t.Fatal("SetRunnerFactory(nil) changed the existing runnerFactory")
	}

	customFactory := func(config.Config, string) agents.Runner { return blockingRunner{} }
	service.SetRunnerFactory(customFactory)
	if reflect.ValueOf(service.runnerFactory).Pointer() != reflect.ValueOf(customFactory).Pointer() {
		t.Fatal("SetRunnerFactory(custom) did not replace the runnerFactory")
	}

	sequenceCases := []struct {
		name     string
		envelope StreamEnvelope
		want     int64
		ok       bool
	}{
		{name: "int64", envelope: StreamEnvelope{Payload: map[string]any{"sequence": int64(4)}}, want: 4, ok: true},
		{name: "int", envelope: StreamEnvelope{Payload: map[string]any{"sequence": 5}}, want: 5, ok: true},
		{name: "float64", envelope: StreamEnvelope{Payload: map[string]any{"sequence": float64(6)}}, want: 6, ok: true},
		{name: "missing", envelope: StreamEnvelope{Payload: map[string]any{}}, ok: false},
		{name: "wrong-payload", envelope: StreamEnvelope{Payload: "ignored"}, ok: false},
	}
	for _, test := range sequenceCases {
		t.Run(test.name, func(t *testing.T) {
			got, ok := sequenceFromEnvelope(test.envelope)
			if ok != test.ok || got != test.want {
				t.Fatalf("sequenceFromEnvelope() = (%d, %v), want (%d, %v)", got, ok, test.want, test.ok)
			}
		})
	}

	service.pendingApprovals[approvalKey("run_1", "call_1")] = pendingApproval{
		RunID: "run_1",
		Payload: map[string]any{
			"tool_name": "write_file",
		},
	}
	payload := service.pendingApprovalPayload("run_1")
	if payload["tool_name"] != "write_file" {
		t.Fatalf("pendingApprovalPayload() = %#v, want copied payload", payload)
	}
	payload["tool_name"] = "mutated"
	if service.pendingApprovals[approvalKey("run_1", "call_1")].Payload["tool_name"] != "write_file" {
		t.Fatal("pendingApprovalPayload() returned aliased map, want defensive copy")
	}
}

func TestService_SubmitRunRequiresOpenRouterKey(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Missing key")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	_, err = service.SubmitRun(ctx, SubmitRunInput{
		SessionID: session.ID,
		Task:      "Review the config path",
	}, func(StreamEnvelope) error { return nil })
	if err == nil {
		t.Fatal("SubmitRun() error = nil, want missing-key error")
	}
	if !strings.Contains(err.Error(), "OpenRouter is not configured yet") {
		t.Fatalf("SubmitRun() error = %q, want missing-key guidance", err.Error())
	}
}

func TestService_SubmitRunRejectsSecondActiveRun(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Single active run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.OpenRouter.UpdatedAt = time.Now().UTC()
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	service.runnerFactory = func(config.Config, string) agents.Runner {
		return blockingRunner{}
	}

	snapshot, err := service.SubmitRun(ctx, SubmitRunInput{
		SessionID: session.ID,
		Task:      "Keep the active run alive",
	}, func(StreamEnvelope) error { return nil })
	if err != nil {
		t.Fatalf("SubmitRun() first error = %v", err)
	}
	if snapshot.ActiveSessionID != session.ID {
		t.Fatalf("snapshot.ActiveSessionID = %q, want %q", snapshot.ActiveSessionID, session.ID)
	}

	activeRunID, err := waitForRegisteredActiveRun(service)
	if err != nil {
		t.Fatalf("waitForRegisteredActiveRun() error = %v", err)
	}

	_, err = service.SubmitRun(ctx, SubmitRunInput{
		SessionID: session.ID,
		Task:      "Attempt a second run",
	}, func(StreamEnvelope) error { return nil })
	if err == nil {
		t.Fatal("SubmitRun() second error = nil, want active-run rejection")
	}
	if !strings.Contains(err.Error(), "already has an active run") {
		t.Fatalf("SubmitRun() second error = %q, want active-run rejection", err.Error())
	}

	cancel, ok := service.activeRunCancel(activeRunID)
	if !ok {
		t.Fatalf("activeRunCancel(%q) = missing, want registered cancel func", activeRunID)
	}
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := service.activeRunCancel(activeRunID); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("active run %q remained registered after cancellation", activeRunID)
}

func TestService_ResolveApprovalUnblocksPendingRequest(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Approval flow")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Update the README", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	approvalCtx := withRunExecutionContext(context.Background(), runExecutionContext{
		SessionID: session.ID,
		RunID:     run.ID,
		Emit: func(StreamEnvelope) error {
			return nil
		},
	})

	var (
		decision ApprovalDecision
		requestErr error
		wait sync.WaitGroup
	)
	wait.Add(1)
	go func() {
		defer wait.Done()
		decision, requestErr = service.RequestApproval(approvalCtx, ApprovalRequest{
			SessionID:    session.ID,
			RunID:        run.ID,
			Role:         run.Role,
			Model:        run.Model,
			ToolCallID:   "call_1",
			ToolName:     agents.ToolWriteFile,
			InputPreview: map[string]any{"path": "README.md"},
			Message:      "Relay needs approval before it can write files inside the configured project root.",
			OccurredAt:   time.Now().UTC(),
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		service.mu.Lock()
		_, ok := service.pendingApprovals[approvalKey(run.ID, "call_1")]
		service.mu.Unlock()
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	snapshot, err := service.ResolveApproval(ctx, ApprovalResponseInput{
		SessionID:  session.ID,
		RunID:      run.ID,
		ToolCallID: "call_1",
		Decision:   "approved",
	})
	if err != nil {
		t.Fatalf("ResolveApproval() error = %v", err)
	}
	wait.Wait()

	if requestErr != nil {
		t.Fatalf("RequestApproval() error = %v", requestErr)
	}
	if !decision.Approved {
		t.Fatalf("decision.Approved = %v, want true", decision.Approved)
	}
	if snapshot.ActiveSessionID != session.ID {
		t.Fatalf("snapshot.ActiveSessionID = %q, want %q", snapshot.ActiveSessionID, session.ID)
	}
}

func TestService_ResolveApprovalRejectsPendingRequest(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Approval reject")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Do not allow command execution", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	approvalCtx := withRunExecutionContext(context.Background(), runExecutionContext{
		SessionID: session.ID,
		RunID:     run.ID,
		Emit:      func(StreamEnvelope) error { return nil },
	})

	var (
		decision   ApprovalDecision
		requestErr error
		wait       sync.WaitGroup
	)
	wait.Add(1)
	go func() {
		defer wait.Done()
		decision, requestErr = service.RequestApproval(approvalCtx, ApprovalRequest{
			SessionID:    session.ID,
			RunID:        run.ID,
			Role:         run.Role,
			Model:        run.Model,
			ToolCallID:   "call_reject",
			ToolName:     agents.ToolRunCommand,
			InputPreview: map[string]any{"command": "pwd"},
			Message:      "Relay needs approval before it can run a shell command from the configured project root.",
			OccurredAt:   time.Now().UTC(),
		})
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		service.mu.Lock()
		_, ok := service.pendingApprovals[approvalKey(run.ID, "call_reject")]
		service.mu.Unlock()
		if ok {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, err = service.ResolveApproval(ctx, ApprovalResponseInput{
		SessionID:  session.ID,
		RunID:      run.ID,
		ToolCallID: "call_reject",
		Decision:   "rejected",
	})
	if err != nil {
		t.Fatalf("ResolveApproval() error = %v", err)
	}
	wait.Wait()

	if requestErr != nil {
		t.Fatalf("RequestApproval() error = %v", requestErr)
	}
	if decision.Approved {
		t.Fatalf("decision.Approved = %v, want false", decision.Approved)
	}

	service.mu.Lock()
	_, ok := service.pendingApprovals[approvalKey(run.ID, "call_reject")]
	service.mu.Unlock()
	if ok {
		t.Fatal("pending approval remained registered after rejection")
	}
}

func TestService_OpenRunReplaysHistoryAndStreamsFutureEvents(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Reconnect run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.OpenRouter.UpdatedAt = time.Now().UTC()
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	runner := &reconnectRunner{
		profile: agents.Profile{Role: sqlite.RoleCoder, Model: config.DefaultCoderModel},
		started: make(chan struct{}),
		releaseState: make(chan struct{}),
		release: make(chan struct{}),
	}
	service.runnerFactory = func(config.Config, string) agents.Runner {
		return runner
	}

	firstCtx, cancelFirst := context.WithCancel(WithStreamSubscriber(context.Background(), "socket-1"))
	firstStream := make(chan StreamEnvelope, 8)
	_, err = service.SubmitRun(firstCtx, SubmitRunInput{
		SessionID: session.ID,
		Task:      "Resume the active stream",
	}, func(envelope StreamEnvelope) error {
		firstStream <- envelope
		return nil
	})
	if err != nil {
		t.Fatalf("SubmitRun() error = %v", err)
	}

	close(runner.releaseState)
	<-runner.started
	state := <-firstStream
	if state.Type != sqlite.EventTypeStateChange {
		t.Fatalf("first stream event = %q, want %q", state.Type, sqlite.EventTypeStateChange)
	}
	statePayload := state.Payload.(map[string]any)
	runID := statePayload["run_id"].(string)
	cancelFirst()

	openCtx, cancelOpen := context.WithCancel(WithStreamSubscriber(context.Background(), "socket-2"))
	defer cancelOpen()
	secondStream := make(chan StreamEnvelope, 8)
	_, err = service.OpenRun(openCtx, OpenRunInput{
		SessionID: session.ID,
		RunID:     runID,
	}, func(envelope StreamEnvelope) error {
		secondStream <- envelope
		return nil
	})
	if err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}

	replayed := <-secondStream
	if replayed.Type != sqlite.EventTypeStateChange {
		t.Fatalf("replayed event = %q, want %q", replayed.Type, sqlite.EventTypeStateChange)
	}
	replayedPayload := replayed.Payload.(map[string]any)
	if replayedPayload["replay"] != true {
		t.Fatalf("replayed state replay = %v, want true", replayedPayload["replay"])
	}

	close(runner.release)

	var seenToken, seenComplete bool
	deadline := time.After(2 * time.Second)
	for !(seenToken && seenComplete) {
		select {
		case envelope := <-secondStream:
			payload := envelope.Payload.(map[string]any)
			switch envelope.Type {
			case sqlite.EventTypeToken:
				if payload["text"] == "reconnected" && payload["replay"] == false {
					seenToken = true
				}
			case sqlite.EventTypeComplete:
				if payload["replay"] == false {
					seenComplete = true
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for live events after reattaching")
		}
	}
}

func TestService_CancelRunPersistsCancelledTerminalEvent(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Cancel run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.OpenRouter.UpdatedAt = time.Now().UTC()
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	service.runnerFactory = func(config.Config, string) agents.Runner {
		return blockingRunner{}
	}

	_, err = service.SubmitRun(WithStreamSubscriber(context.Background(), "socket-cancel"), SubmitRunInput{
		SessionID: session.ID,
		Task:      "Cancel me",
	}, func(StreamEnvelope) error {
		return nil
	})
	if err != nil {
		t.Fatalf("SubmitRun() error = %v", err)
	}

	runID, err := waitForRegisteredActiveRun(service)
	if err != nil {
		t.Fatalf("waitForRegisteredActiveRun() error = %v", err)
	}

	if _, err := service.CancelRun(context.Background(), CancelRunInput{SessionID: session.ID, RunID: runID}, nil); err != nil {
		t.Fatalf("CancelRun() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := store.GetAgentRun(context.Background(), runID)
		if err != nil {
			t.Fatalf("GetAgentRun() error = %v", err)
		}
		if run.State == sqlite.RunStateErrored {
			if run.ErrorCode != "run_cancelled" {
				t.Fatalf("run.ErrorCode = %q, want run_cancelled", run.ErrorCode)
			}
			events, err := store.ListRunEvents(context.Background(), runID)
			if err != nil {
				t.Fatalf("ListRunEvents() error = %v", err)
			}
			if len(events) == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			last := events[len(events)-1]
			if last.EventType != sqlite.EventTypeError {
				t.Fatalf("last.EventType = %q, want %q", last.EventType, sqlite.EventTypeError)
			}
			if !strings.Contains(last.PayloadJSON, "run_cancelled") {
				t.Fatalf("last.PayloadJSON = %q, want run_cancelled code", last.PayloadJSON)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("timed out waiting for cancelled run terminal state")
}

func TestService_EmitTokenAnnotatesFirstTokenLatencyOnlyOnce(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Token latency")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Measure token latency", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	service.registerActiveRun(run.ID, func() {})
	defer service.clearActiveRun(run.ID)

	subscriberCtx, cancel := context.WithCancel(WithStreamSubscriber(context.Background(), "token-latency"))
	defer cancel()
	envelopes := make(chan StreamEnvelope, 4)
	if _, ok := service.attachRunSubscriber(subscriberCtx, run.ID, func(envelope StreamEnvelope) error {
		envelopes <- envelope
		return nil
	}, false); !ok {
		t.Fatal("attachRunSubscriber() = false, want active subscriber")
	}

	time.Sleep(5 * time.Millisecond)
	if err := service.emitToken(ctx, run.ID, "alpha", nil); err != nil {
		t.Fatalf("emitToken(first) error = %v", err)
	}
	if err := service.emitToken(ctx, run.ID, "beta", nil); err != nil {
		t.Fatalf("emitToken(second) error = %v", err)
	}

	first := (<-envelopes).Payload.(map[string]any)
	second := (<-envelopes).Payload.(map[string]any)
	latency, ok := first["first_token_latency_ms"].(int64)
	if !ok {
		latencyFloat, floatOK := first["first_token_latency_ms"].(float64)
		if !floatOK {
			t.Fatalf("first token payload missing first_token_latency_ms: %#v", first)
		}
		latency = int64(latencyFloat)
	}
	if latency < 0 {
		t.Fatalf("first_token_latency_ms = %d, want non-negative latency", latency)
	}
	if _, exists := second["first_token_latency_ms"]; exists {
		t.Fatalf("second token payload unexpectedly contained first_token_latency_ms: %#v", second)
	}

	storedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if storedRun.FirstTokenAt == nil {
		t.Fatal("storedRun.FirstTokenAt = nil, want first token timestamp")
	}
}

func TestService_EmitToolEventsPersistAndDispatch(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Tool event run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Inspect README", sqlite.RoleCoder, config.DefaultCoderModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	service.registerActiveRun(run.ID, func() {})
	defer service.clearActiveRun(run.ID)

	subscriberCtx, cancel := context.WithCancel(WithStreamSubscriber(context.Background(), "socket-tool-events"))
	defer cancel()
	var envelopes []StreamEnvelope
	if _, ok := service.attachRunSubscriber(subscriberCtx, run.ID, func(envelope StreamEnvelope) error {
		in, err := json.Marshal(envelope.Payload)
		if err != nil {
			return err
		}
		var payload map[string]any
		if err := json.Unmarshal(in, &payload); err != nil {
			return err
		}
		envelopes = append(envelopes, StreamEnvelope{Type: envelope.Type, Payload: payload})
		return nil
	}, false); !ok {
		t.Fatal("attachRunSubscriber() = not attached, want active subscriber")
	}

	if err := service.emitToolCall(ctx, run.ID, agents.ToolCallEvent{
		ToolCallID:   "call_1",
		ToolName:     agents.ToolReadFile,
		InputPreview: map[string]any{"path": "README.md"},
	}, nil); err != nil {
		t.Fatalf("emitToolCall() error = %v", err)
	}
	if err := service.emitToolResult(ctx, run.ID, agents.ToolResultEvent{
		ToolCallID:    "call_1",
		ToolName:      agents.ToolReadFile,
		Status:        "completed",
		ResultPreview: map[string]any{"summary": "Loaded file content."},
	}, nil); err != nil {
		t.Fatalf("emitToolResult() error = %v", err)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].EventType != sqlite.EventTypeToolCall {
		t.Fatalf("events[0].EventType = %q, want %q", events[0].EventType, sqlite.EventTypeToolCall)
	}
	if events[1].EventType != sqlite.EventTypeToolResult {
		t.Fatalf("events[1].EventType = %q, want %q", events[1].EventType, sqlite.EventTypeToolResult)
	}
	if len(envelopes) != 2 {
		t.Fatalf("len(envelopes) = %d, want 2", len(envelopes))
	}
	if envelopes[0].Type != sqlite.EventTypeToolCall || envelopes[1].Type != sqlite.EventTypeToolResult {
		t.Fatalf("envelopes = %#v, want tool call then tool result", envelopes)
	}

	storedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if storedRun.State != sqlite.RunStateToolRunning {
		t.Fatalf("storedRun.State = %q, want %q", storedRun.State, sqlite.RunStateToolRunning)
	}
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, _ string, handlers agents.StreamEventHandlers) error {
	<-ctx.Done()
	return ctx.Err()
}

func (blockingRunner) Profile() agents.Profile {
	return agents.Profile{Role: sqlite.RoleCoder, Model: config.DefaultCoderModel}
}

type reconnectRunner struct {
	profile agents.Profile
	started chan struct{}
	releaseState chan struct{}
	release chan struct{}
}

func (r *reconnectRunner) Profile() agents.Profile {
	return r.profile
}

func (r *reconnectRunner) Run(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
	close(r.started)
	<-r.releaseState
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(string(sqlite.RunStateThinking))
	}
	<-r.release
	if handlers.OnToken != nil {
		handlers.OnToken("reconnected")
	}
	if handlers.OnComplete != nil {
		handlers.OnComplete("stop")
	}
	return nil
}

func waitForRegisteredActiveRun(service *Service) (string, error) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		service.mu.Lock()
		for runID := range service.activeRuns {
			service.mu.Unlock()
			return runID, nil
		}
		service.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	return "", errors.New("timed out waiting for registered active run")
}

func newTestServiceStore(t *testing.T) (config.Paths, *sqlite.Store) {
	t.Helper()
	paths, err := config.EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	return paths, store
}
