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

	"github.com/eristemena/relay/internal/agents"
	"github.com/eristemena/relay/internal/config"
	ws "github.com/eristemena/relay/internal/handlers/ws"
	workspaceorchestrator "github.com/eristemena/relay/internal/orchestrator/workspace"
	"github.com/eristemena/relay/internal/storage/sqlite"
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

func TestAgentStreaming_CompletePreservesTokenCountWithoutContextLimit(t *testing.T) {
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
			Model:        "custom/local-model",
			SystemPrompt: "test prompt",
			AllowedTools: []agents.ToolName{agents.ToolReadFile},
		},
		ready:   make(chan struct{}),
		release: make(chan struct{}),
		onComplete: func(handlers agents.StreamEventHandlers) {
			if handlers.OnComplete == nil {
				return
			}
			tokensUsed := 4812
			handlers.OnComplete(agents.CompletionMetadata{
				FinishReason: "stop",
				TokensUsed:   &tokensUsed,
			})
		},
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	session, err := store.CreateSession(context.Background(), "Streaming token count only")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Stream a run without a context limit",
		},
	})

	_ = readStreamingEnvelope(t, connection)
	<-runner.ready
	close(runner.release)

	_ = readUntilStreamingType(t, connection, "state_change")
	_ = readUntilStreamingType(t, connection, "token")
	_ = readUntilStreamingType(t, connection, "token")
	complete := readUntilStreamingType(t, connection, "complete")
	completePayload := complete["payload"].(map[string]any)
	if completePayload["tokens_used"] != float64(4812) {
		t.Fatalf("tokens_used = %v, want 4812", completePayload["tokens_used"])
	}
	if _, ok := completePayload["context_limit"]; ok {
		t.Fatalf("context_limit = %v, want omitted when unavailable", completePayload["context_limit"])
	}
	if completePayload["finish_reason"] != "stop" {
		t.Fatalf("finish_reason = %v, want stop", completePayload["finish_reason"])
	}
}

func TestAgentStreaming_SubmitDeliversOrderedOrchestrationEvents(t *testing.T) {
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

	plannerStarted := make(chan struct{})
	plannerRelease := make(chan struct{})
	coderStarted := make(chan struct{})
	testerStarted := make(chan struct{})
	parallelRelease := make(chan struct{})
	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				close(plannerStarted)
				<-plannerRelease
				if handlers.OnToken != nil {
					handlers.OnToken("planner")
				}
				completeStreamingRun(handlers)
				return nil
			}}
		case sqlite.RoleCoder:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				close(coderStarted)
				<-parallelRelease
				if handlers.OnToken != nil {
					handlers.OnToken("coder")
				}
				completeStreamingRun(handlers)
				return nil
			}}
		case sqlite.RoleTester:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				close(testerStarted)
				<-parallelRelease
				if handlers.OnToken != nil {
					handlers.OnToken("tester")
				}
				completeStreamingRun(handlers)
				return nil
			}}
		default:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				if handlers.OnToken != nil && role == sqlite.RoleExplainer {
					handlers.OnToken("explainer")
				}
				completeStreamingRun(handlers)
				return nil
			}}
		}
	})

	session, err := store.CreateSession(context.Background(), "Orchestration streaming")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Run the orchestration graph",
		},
	})

	bootstrap := readUntilStreamingType(t, connection, "workspace.bootstrap")
	bootstrapPayload := bootstrap["payload"].(map[string]any)
	if bootstrapPayload["active_session_id"] != session.ID {
		t.Fatalf("bootstrap active_session_id = %v, want %q", bootstrapPayload["active_session_id"], session.ID)
	}

	select {
	case <-plannerStarted:
		close(plannerRelease)
	case <-time.After(2 * time.Second):
		t.Fatal("planner did not start")
	}
	select {
	case <-coderStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("coder did not start")
	}
	select {
	case <-testerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("tester did not start")
	}
	close(parallelRelease)

	spawnedRoles := make([]string, 0, 5)
	taskAssignedCount := 0
	seenRunComplete := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !seenRunComplete {
		envelope := readStreamingEnvelope(t, connection)
		switch envelope["type"] {
		case "agent_spawned":
			payload := envelope["payload"].(map[string]any)
			spawnedRoles = append(spawnedRoles, payload["role"].(string))
			if payload["agent_id"] == "" {
				t.Fatalf("agent_spawned payload missing agent_id: %#v", payload)
			}
		case "task_assigned":
			taskAssignedCount++
		case "run_complete":
			seenRunComplete = true
		}
	}

	if !seenRunComplete {
		t.Fatal("run_complete was not delivered")
	}
	if len(spawnedRoles) != 5 {
		t.Fatalf("len(spawnedRoles) = %d, want 5 (%v)", len(spawnedRoles), spawnedRoles)
	}
	if spawnedRoles[0] != string(sqlite.RolePlanner) {
		t.Fatalf("spawnedRoles[0] = %q, want planner", spawnedRoles[0])
	}
	if indexOfRole(spawnedRoles, string(sqlite.RoleReviewer)) <= indexOfRole(spawnedRoles, string(sqlite.RoleCoder)) || indexOfRole(spawnedRoles, string(sqlite.RoleReviewer)) <= indexOfRole(spawnedRoles, string(sqlite.RoleTester)) {
		t.Fatalf("spawnedRoles = %v, want reviewer after coder and tester", spawnedRoles)
	}
	if spawnedRoles[len(spawnedRoles)-1] != string(sqlite.RoleExplainer) {
		t.Fatalf("last spawned role = %q, want explainer", spawnedRoles[len(spawnedRoles)-1])
	}
	if taskAssignedCount != 5 {
		t.Fatalf("taskAssignedCount = %d, want 5", taskAssignedCount)
	}
}

func TestAgentStreaming_OrchestrationAgentErrorContinuesToRunComplete(t *testing.T) {
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

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				completeStreamingRun(handlers)
				return nil
			}}
		case sqlite.RoleCoder:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				if handlers.OnToken != nil {
					handlers.OnToken("partial coder output")
				}
				if handlers.OnError != nil {
					handlers.OnError("agent_generation_failed", "Coder could not finish the draft, but Relay can continue with preserved output.")
				}
				return nil
			}}
		case sqlite.RoleTester:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				completeStreamingRun(handlers)
				return nil
			}}
		default:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				completeStreamingRun(handlers)
				return nil
			}}
		}
	})

	session, err := store.CreateSession(context.Background(), "Agent error orchestration")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Preserve partial orchestration failures",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")

	spawnedRoles := make([]string, 0, 5)
	sawAgentError := false
	sawRunComplete := false
	sawRunError := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !sawRunComplete {
		envelope := readStreamingEnvelope(t, connection)
		switch envelope["type"] {
		case "agent_spawned":
			payload := envelope["payload"].(map[string]any)
			spawnedRoles = append(spawnedRoles, payload["role"].(string))
		case "agent_error":
			payload := envelope["payload"].(map[string]any)
			agentID, _ := payload["agent_id"].(string)
			if strings.HasSuffix(agentID, "_coder_2") {
				sawAgentError = true
			}
		case "run_error":
			sawRunError = true
		case "run_complete":
			sawRunComplete = true
		}
	}

	if !sawAgentError {
		t.Fatal("expected agent_error event for coder")
	}
	if !sawRunComplete {
		t.Fatal("expected run_complete after coder agent_error")
	}
	if sawRunError {
		t.Fatal("did not expect run_error when orchestration preserved coder failure")
	}
	if indexOfRole(spawnedRoles, string(sqlite.RoleReviewer)) == -1 || indexOfRole(spawnedRoles, string(sqlite.RoleExplainer)) == -1 {
		t.Fatalf("spawnedRoles = %v, want reviewer and explainer after agent error", spawnedRoles)
	}
}

func TestAgentStreaming_OrchestrationCoderClarificationHaltsBeforeReviewerAndExplainer(t *testing.T) {
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

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				if handlers.OnToken != nil {
					handlers.OnToken("planner output")
				}
				completeStreamingRun(handlers)
				return nil
			}}
		case sqlite.RoleCoder:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				if handlers.OnToken != nil {
					handlers.OnToken("Would you like me to review your specific .env.example file and add appropriate comments?")
				}
				completeStreamingRun(handlers)
				return nil
			}}
		default:
			return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				completeStreamingRun(handlers)
				return nil
			}}
		}
	})

	session, err := store.CreateSession(context.Background(), "Coder clarification halt orchestration")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Stop after coder clarification",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")

	spawnedRoles := make([]string, 0, 5)
	sawAgentError := false
	sawRunError := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !sawRunError {
		envelope := readStreamingEnvelope(t, connection)
		switch envelope["type"] {
		case "agent_spawned":
			payload := envelope["payload"].(map[string]any)
			spawnedRoles = append(spawnedRoles, payload["role"].(string))
		case "agent_error":
			payload := envelope["payload"].(map[string]any)
			if payload["code"] == "coder_clarification_required" {
				sawAgentError = true
			}
		case "run_error":
			payload := envelope["payload"].(map[string]any)
			if payload["code"] != "coder_clarification_required" {
				t.Fatalf("run_error code = %v, want coder_clarification_required", payload["code"])
			}
			sawRunError = true
		}
	}

	if !sawAgentError {
		t.Fatal("expected coder agent_error before run_error")
	}
	if !sawRunError {
		t.Fatal("expected run_error after coder clarification")
	}
	if indexOfRole(spawnedRoles, string(sqlite.RoleReviewer)) != -1 || indexOfRole(spawnedRoles, string(sqlite.RoleExplainer)) != -1 {
		t.Fatalf("spawnedRoles = %v, want no reviewer or explainer after coder clarification", spawnedRoles)
	}
}

func TestAgentStreaming_OrchestrationPlannerFailureHaltsBeforeDownstreamSpawn(t *testing.T) {
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

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
			if role == sqlite.RolePlanner {
				if handlers.OnError != nil {
					handlers.OnError("agent_generation_failed", "Planner could not break the goal into stages.")
				}
				return nil
			}
			completeStreamingRun(handlers)
			return nil
		}}
	})

	session, err := store.CreateSession(context.Background(), "Planner halt orchestration")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Stop after planner failure",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")

	spawnedRoles := make([]string, 0, 5)
	sawAgentError := false
	sawRunError := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !sawRunError {
		envelope := readStreamingEnvelope(t, connection)
		switch envelope["type"] {
		case "agent_spawned":
			payload := envelope["payload"].(map[string]any)
			spawnedRoles = append(spawnedRoles, payload["role"].(string))
		case "agent_error":
			sawAgentError = true
		case "run_error":
			payload := envelope["payload"].(map[string]any)
			agentID, _ := payload["agent_id"].(string)
			if !strings.HasSuffix(agentID, "_planner_1") {
				t.Fatalf("run_error agent_id = %v, want planner stage agent id suffix _planner_1", payload["agent_id"])
			}
			sawRunError = true
		}
	}

	if !sawAgentError {
		t.Fatal("expected planner agent_error before run_error")
	}
	if !sawRunError {
		t.Fatal("expected run_error after planner failure")
	}
	if len(spawnedRoles) != 1 || spawnedRoles[0] != string(sqlite.RolePlanner) {
		t.Fatalf("spawnedRoles = %v, want only planner before halt", spawnedRoles)
	}
}

func TestAgentStreaming_OrchestrationPlannerClarificationHaltsBeforeDownstreamSpawn(t *testing.T) {
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

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationIntegrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
			if role == sqlite.RolePlanner {
				if handlers.OnToken != nil {
					handlers.OnToken("Would you like me to review your specific .env.example file and add appropriate comments?")
				}
				completeStreamingRun(handlers)
				return nil
			}
			completeStreamingRun(handlers)
			return nil
		}}
	})

	session, err := store.CreateSession(context.Background(), "Planner clarification halt orchestration")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Stop after planner clarification",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")

	spawnedRoles := make([]string, 0, 5)
	sawAgentError := false
	sawRunError := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !sawRunError {
		envelope := readStreamingEnvelope(t, connection)
		switch envelope["type"] {
		case "agent_spawned":
			payload := envelope["payload"].(map[string]any)
			spawnedRoles = append(spawnedRoles, payload["role"].(string))
		case "agent_error":
			payload := envelope["payload"].(map[string]any)
			if payload["code"] == "planner_clarification_required" {
				sawAgentError = true
			}
		case "run_error":
			payload := envelope["payload"].(map[string]any)
			if payload["code"] != "planner_clarification_required" {
				t.Fatalf("run_error code = %v, want planner_clarification_required", payload["code"])
			}
			sawRunError = true
		}
	}

	if !sawAgentError {
		t.Fatal("expected planner agent_error before run_error")
	}
	if !sawRunError {
		t.Fatal("expected run_error after planner clarification")
	}
	if len(spawnedRoles) != 1 || spawnedRoles[0] != string(sqlite.RolePlanner) {
		t.Fatalf("spawnedRoles = %v, want only planner before halt", spawnedRoles)
	}
}

type scriptedRunner struct {
	profile agents.Profile
	ready   chan struct{}
	release chan struct{}
	onComplete func(handlers agents.StreamEventHandlers)
}

type scriptedPromptOnlyAgent struct {
	profile agents.Profile
	run     func(ctx context.Context, task string, handlers agents.StreamEventHandlers) error
}

func (a scriptedPromptOnlyAgent) Profile() agents.Profile {
	return a.profile
}

func (a scriptedPromptOnlyAgent) Run(ctx context.Context, task string, handlers agents.StreamEventHandlers) error {
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(sqlite.AgentExecutionStateThinking)
	}
	if a.run == nil {
		completeStreamingRun(handlers)
		return nil
	}
	return a.run(ctx, task, handlers)
}

func completeStreamingRun(handlers agents.StreamEventHandlers) {
	if handlers.OnComplete != nil {
		handlers.OnComplete(agents.CompletionMetadata{FinishReason: "stop"})
	}
}

func orchestrationIntegrationProfile(role sqlite.AgentRole) agents.Profile {
	switch role {
	case sqlite.RolePlanner:
		return agents.NewPlanner(config.DefaultPlannerModel)
	case sqlite.RoleReviewer:
		return agents.NewReviewer(config.DefaultReviewerModel)
	case sqlite.RoleTester:
		return agents.NewTester(config.DefaultTesterModel)
	case sqlite.RoleExplainer:
		return agents.NewExplainer(config.DefaultExplainerModel)
	default:
		return agents.NewCoder(config.DefaultCoderModel)
	}
}

func indexOfRole(roles []string, target string) int {
	for index, role := range roles {
		if role == target {
			return index
		}
	}
	return -1
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
	completeStreamingRun(handlers)
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
	if r.onComplete != nil {
		r.onComplete(handlers)
	} else {
		completeStreamingRun(handlers)
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