package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"slices"
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

func TestService_BootstrapRecoversOrphanedActiveRun(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Recover orphaned run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	run, err := store.CreateAgentRun(ctx, session.ID, "Stuck task", sqlite.RoleTester, config.DefaultTesterModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	run.State = sqlite.RunStateToolRunning
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}

	snapshot, err := service.Bootstrap(ctx, session.ID)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if snapshot.ActiveRunID != "" {
		t.Fatalf("snapshot.ActiveRunID = %q, want empty after orphan recovery", snapshot.ActiveRunID)
	}
	if len(snapshot.RunSummaries) != 1 {
		t.Fatalf("len(snapshot.RunSummaries) = %d, want 1", len(snapshot.RunSummaries))
	}
	if snapshot.RunSummaries[0].ErrorCode != "run_interrupted" {
		t.Fatalf("snapshot.RunSummaries[0].ErrorCode = %q, want run_interrupted", snapshot.RunSummaries[0].ErrorCode)
	}

	storedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if storedRun.State != sqlite.RunStateErrored {
		t.Fatalf("storedRun.State = %q, want %q", storedRun.State, sqlite.RunStateErrored)
	}
	if storedRun.ErrorCode != "run_interrupted" {
		t.Fatalf("storedRun.ErrorCode = %q, want run_interrupted", storedRun.ErrorCode)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected interrupt recovery error event")
	}
	last := events[len(events)-1]
	if last.EventType != sqlite.EventTypeError {
		t.Fatalf("last.EventType = %q, want %q", last.EventType, sqlite.EventTypeError)
	}
	if !strings.Contains(last.PayloadJSON, "run_interrupted") {
		t.Fatalf("last.PayloadJSON = %q, want run_interrupted code", last.PayloadJSON)
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

func TestService_EmitRunErrorLogsTerminalFailureDetails(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	var logBuffer bytes.Buffer
	service.SetLogger(slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Log session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Fail planner", sqlite.RolePlanner, "anthropic/claude-opus-4")
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	execution, err := store.CreateAgentExecution(ctx, sqlite.AgentExecution{
		ID:         "agent_planner_1",
		RunID:      run.ID,
		Role:       sqlite.RolePlanner,
		Model:      "anthropic/claude-opus-4",
		State:      sqlite.AgentExecutionStateErrored,
		TaskText:   "Fail planner",
		SpawnOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateAgentExecution() error = %v", err)
	}

	err = service.emitRunError(ctx, run, "run_stage_failed", "The run stopped because Relay could not finish the planner stage.", &execution)
	if err != nil {
		t.Fatalf("emitRunError() error = %v", err)
	}

	logged := logBuffer.String()
	if !strings.Contains(logged, "orchestration run halted") {
		t.Fatalf("log output = %q, want halt message", logged)
	}
	for _, want := range []string{
		"run_stage_failed",
		"The run stopped because Relay could not finish the planner stage.",
		run.ID,
		session.ID,
		"agent_planner_1",
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output = %q, want substring %q", logged, want)
		}
	}
	if strings.Contains(logged, "level=INFO") {
		t.Fatalf("log output = %q, want terminal failure logged above info level", logged)
	}
	if !strings.Contains(logged, "level=ERROR") {
		t.Fatalf("log output = %q, want error level", logged)
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

	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return blockingRunner{}
	})

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

func TestService_SubmitRunRejectsSecondActiveOrchestrationRun(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Single active orchestration run")
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

	plannerStarted := make(chan struct{})
	plannerRelease := make(chan struct{})
	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{
				profile: orchestrationProfile(role),
				run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
					close(plannerStarted)
					<-plannerRelease
					if handlers.OnComplete != nil {
						handlers.OnComplete("stop")
					}
					return nil
				},
			}
		default:
			return scriptedPromptOnlyAgent{
				profile: orchestrationProfile(role),
				run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
					if handlers.OnComplete != nil {
						handlers.OnComplete("stop")
					}
					return nil
				},
			}
		}
	})

	_, err = service.SubmitRun(ctx, SubmitRunInput{
		SessionID: session.ID,
		Task:      "Hold the orchestration run open",
	}, func(StreamEnvelope) error { return nil })
	if err != nil {
		t.Fatalf("SubmitRun() first error = %v", err)
	}

	select {
	case <-plannerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("planner did not start")
	}

	_, err = service.SubmitRun(ctx, SubmitRunInput{
		SessionID: session.ID,
		Task:      "Attempt a second orchestration run",
	}, func(StreamEnvelope) error { return nil })
	if err == nil {
		t.Fatal("SubmitRun() second error = nil, want active-run rejection")
	}
	if !strings.Contains(err.Error(), "already has an active run") {
		t.Fatalf("SubmitRun() second error = %q, want active-run rejection", err.Error())
	}

	activeRunID, err := waitForRegisteredActiveRun(service)
	if err != nil {
		t.Fatalf("waitForRegisteredActiveRun() error = %v", err)
	}

	close(plannerRelease)
	cancel, ok := service.activeRunCancel(activeRunID)
	if ok {
		cancel()
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := service.activeRunCancel(activeRunID); !ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("active run %q remained registered after release", activeRunID)
}

func TestService_ExecuteOrchestrationRunOrchestratesPlannerParallelReviewerAndExplainer(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Orchestration path")
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

	run, err := store.CreateAgentRun(ctx, session.ID, "Explain the orchestration DAG", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	plannerStarted := make(chan struct{})
	plannerRelease := make(chan struct{})
	coderStarted := make(chan struct{})
	testerStarted := make(chan struct{})
	parallelRelease := make(chan struct{})
	runComplete := make(chan struct{})
	errCh := make(chan error, 1)

	var (
		orderMu    sync.Mutex
		startOrder []sqlite.AgentRole
	)
	recordStart := func(role sqlite.AgentRole) {
		orderMu.Lock()
		defer orderMu.Unlock()
		startOrder = append(startOrder, role)
	}

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				close(plannerStarted)
				<-plannerRelease
				if handlers.OnToken != nil {
					handlers.OnToken("planner output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleCoder:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				close(coderStarted)
				<-parallelRelease
				if handlers.OnToken != nil {
					handlers.OnToken("coder output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleTester:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				close(testerStarted)
				<-parallelRelease
				if handlers.OnToken != nil {
					handlers.OnToken("tester output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleReviewer:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		default:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnToken != nil {
					handlers.OnToken("explainer output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				close(runComplete)
				return nil
			}}
		}
	})
	go func() {
		errCh <- service.executeOrchestrationRun(context.Background(), run, run.TaskText, cfg)
	}()

	select {
	case <-plannerStarted:
		close(plannerRelease)
	case <-time.After(2 * time.Second):
		t.Fatal("planner did not start")
	}

	select {
	case <-testerStarted:
	case err := <-errCh:
		t.Fatalf("executeOrchestrationRun() returned before tester start: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatalf("tester did not start; started roles so far = %v", startOrder)
	}
	close(parallelRelease)

	select {
	case <-runComplete:
	case <-time.After(2 * time.Second):
		t.Fatal("orchestration did not complete")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("executeOrchestrationRun() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("executeOrchestrationRun() did not return")
	}

	runSummaries, err := store.ListRunSummaries(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListRunSummaries() error = %v", err)
	}
	if len(runSummaries) != 1 {
		t.Fatalf("len(runSummaries) = %d, want 1", len(runSummaries))
	}
	if runSummaries[0].State != sqlite.RunStateCompleted {
		t.Fatalf("runSummaries[0].State = %q, want %q", runSummaries[0].State, sqlite.RunStateCompleted)
	}

	executions, err := store.ListAgentExecutions(ctx, runSummaries[0].ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 5 {
		t.Fatalf("len(executions) = %d, want 5", len(executions))
	}
	for index, execution := range executions {
		if execution.SpawnOrder != index+1 {
			t.Fatalf("execution[%d].SpawnOrder = %d, want %d", index, execution.SpawnOrder, index+1)
		}
		if strings.TrimSpace(execution.TaskText) == "" {
			t.Fatalf("execution[%d].TaskText = empty, want persisted assignment text", index)
		}
	}

	events, err := store.ListRunEvents(ctx, runSummaries[0].ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	taskAssignedCount := 0
	for _, event := range events {
		if event.EventType == sqlite.EventTypeTaskAssigned {
			taskAssignedCount++
		}
	}
	if taskAssignedCount != 5 {
		t.Fatalf("taskAssignedCount = %d, want 5", taskAssignedCount)
	}

	orderMu.Lock()
	defer orderMu.Unlock()
	if len(startOrder) != 5 {
		t.Fatalf("len(startOrder) = %d, want 5", len(startOrder))
	}
	plannerIndex := indexOfRole(startOrder, sqlite.RolePlanner)
	coderIndex := indexOfRole(startOrder, sqlite.RoleCoder)
	testerIndex := indexOfRole(startOrder, sqlite.RoleTester)
	reviewerIndex := indexOfRole(startOrder, sqlite.RoleReviewer)
	explainerIndex := indexOfRole(startOrder, sqlite.RoleExplainer)
	if !(plannerIndex < coderIndex && plannerIndex < testerIndex && coderIndex < reviewerIndex && testerIndex < reviewerIndex && reviewerIndex < explainerIndex) {
		t.Fatalf("startOrder = %v, want planner first, coder/tester before reviewer, and explainer last", startOrder)
	}
}

func TestNewServiceDefaultAgentFactoryUsesToolEnabledRoleProfiles(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	agent := service.agentFactory(config.Config{
		OpenRouter:  config.OpenRouter{APIKey: "or-test-key"},
		ProjectRoot: t.TempDir(),
		Agents: config.AgentModels{
			Planner:   config.DefaultPlannerModel,
			Coder:     config.DefaultCoderModel,
			Reviewer:  config.DefaultReviewerModel,
			Tester:    config.DefaultTesterModel,
			Explainer: config.DefaultExplainerModel,
		},
	}, sqlite.RolePlanner)

	profile := agent.Profile()
	if profile.Role != sqlite.RolePlanner {
		t.Fatalf("profile.Role = %q, want %q", profile.Role, sqlite.RolePlanner)
	}
	if len(profile.AllowedTools) == 0 {
		t.Fatal("profile.AllowedTools = empty, want tool-enabled orchestration planner")
	}
	if !slices.Equal(profile.AllowedTools, []agents.ToolName{agents.ToolReadFile, agents.ToolSearchCodebase}) {
		t.Fatalf("profile.AllowedTools = %v, want planner read/search tools", profile.AllowedTools)
	}
}

func TestService_ExecuteOrchestrationRunLogsUnderlyingPlannerStageError(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	var logBuffer bytes.Buffer
	service.SetLogger(slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo})))

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Planner hard failure")
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

	run, err := store.CreateAgentRun(ctx, session.ID, "Trigger planner hard failure", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, _ agents.StreamEventHandlers) error {
			if role == sqlite.RolePlanner {
				return errors.New("planner stream blew up before agent error emission")
			}
			return nil
		}}
	})

	err = service.executeOrchestrationRun(ctx, run, "Trigger planner hard failure", cfg)
	if err != nil {
		t.Fatalf("executeOrchestrationRun() error = %v, want nil because the service emits terminal run state internally", err)
	}

	logged := logBuffer.String()
	for _, want := range []string{
		"orchestration stage execution failed",
		"planner stream blew up before agent error emission",
		"role=planner",
		"run_id=" + run.ID,
		"session_id=" + session.ID,
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("log output = %q, want substring %q", logged, want)
		}
	}
	if !strings.Contains(logged, "orchestration run halted") {
		t.Fatalf("log output = %q, want terminal run halt log as well", logged)
	}

	storedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if storedRun.ErrorCode != "run_stage_failed" {
		t.Fatalf("storedRun.ErrorCode = %q, want run_stage_failed", storedRun.ErrorCode)
	}
}

func TestService_ExecuteOrchestrationRunAllowsConsecutiveRunsWithoutExecutionIDCollisions(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Consecutive orchestration runs")
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

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
			if handlers.OnToken != nil {
				handlers.OnToken(string(role) + " output")
			}
			if handlers.OnComplete != nil {
				handlers.OnComplete("stop")
			}
			return nil
		}}
	})

	runOne, err := store.CreateAgentRun(ctx, session.ID, "First run", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() runOne error = %v", err)
	}
	if err := service.executeOrchestrationRun(ctx, runOne, runOne.TaskText, cfg); err != nil {
		t.Fatalf("executeOrchestrationRun() runOne error = %v", err)
	}

	runTwo, err := store.CreateAgentRun(ctx, session.ID, "Second run", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() runTwo error = %v", err)
	}
	if err := service.executeOrchestrationRun(ctx, runTwo, runTwo.TaskText, cfg); err != nil {
		t.Fatalf("executeOrchestrationRun() runTwo error = %v", err)
	}

	executionsOne, err := store.ListAgentExecutions(ctx, runOne.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() runOne error = %v", err)
	}
	executionsTwo, err := store.ListAgentExecutions(ctx, runTwo.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() runTwo error = %v", err)
	}
	if len(executionsOne) != 5 || len(executionsTwo) != 5 {
		t.Fatalf("execution counts = (%d, %d), want (5, 5)", len(executionsOne), len(executionsTwo))
	}
	if executionsOne[0].ID == executionsTwo[0].ID {
		t.Fatalf("planner execution IDs collided: %q", executionsOne[0].ID)
	}
	if !strings.Contains(executionsOne[0].ID, runOne.ID) {
		t.Fatalf("executionsOne[0].ID = %q, want run-scoped ID containing %q", executionsOne[0].ID, runOne.ID)
	}
	if !strings.Contains(executionsTwo[0].ID, runTwo.ID) {
		t.Fatalf("executionsTwo[0].ID = %q, want run-scoped ID containing %q", executionsTwo[0].ID, runTwo.ID)
	}

	eventsTwo, err := store.ListRunEvents(ctx, runTwo.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() runTwo error = %v", err)
	}
	if len(eventsTwo) == 0 {
		t.Fatal("expected orchestration events for second run")
	}
	if !strings.Contains(eventsTwo[0].PayloadJSON, runTwo.ID) {
		t.Fatalf("eventsTwo[0].PayloadJSON = %q, want run-scoped agent ids", eventsTwo[0].PayloadJSON)
	}
}

func TestService_ExecuteOrchestrationRunContinuesAfterCoderAgentError(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Agent error continuation")
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

	run, err := store.CreateAgentRun(ctx, session.ID, "Preserve partial failures", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	var (
		orderMu    sync.Mutex
		startOrder []sqlite.AgentRole
	)
	recordStart := func(role sqlite.AgentRole) {
		orderMu.Lock()
		defer orderMu.Unlock()
		startOrder = append(startOrder, role)
	}

	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnToken != nil {
					handlers.OnToken("planner output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleCoder:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnToken != nil {
					handlers.OnToken("partial coder output")
				}
				if handlers.OnError != nil {
					handlers.OnError("agent_generation_failed", "Coder could not finish the draft, but Relay can continue with preserved output.")
				}
				return nil
			}}
		case sqlite.RoleTester:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnToken != nil {
					handlers.OnToken("tester output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleReviewer:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		default:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				recordStart(role)
				if handlers.OnToken != nil {
					handlers.OnToken("explainer output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		}
	})

	if err := service.executeOrchestrationRun(context.Background(), run, run.TaskText, cfg); err != nil {
		t.Fatalf("executeOrchestrationRun() error = %v", err)
	}

	runSummaries, err := store.ListRunSummaries(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListRunSummaries() error = %v", err)
	}
	if len(runSummaries) != 1 {
		t.Fatalf("len(runSummaries) = %d, want 1", len(runSummaries))
	}
	if runSummaries[0].State != sqlite.RunStateCompleted {
		t.Fatalf("runSummaries[0].State = %q, want %q", runSummaries[0].State, sqlite.RunStateCompleted)
	}

	executions, err := store.ListAgentExecutions(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 5 {
		t.Fatalf("len(executions) = %d, want 5", len(executions))
	}
	if executions[1].Role != sqlite.RoleCoder || executions[1].State != sqlite.AgentExecutionStateErrored {
		t.Fatalf("coder execution = %#v, want errored coder execution", executions[1])
	}
	if executions[3].Role != sqlite.RoleReviewer || executions[3].State != sqlite.AgentExecutionStateCompleted {
		t.Fatalf("reviewer execution = %#v, want completed reviewer execution", executions[3])
	}
	if executions[4].Role != sqlite.RoleExplainer || executions[4].State != sqlite.AgentExecutionStateCompleted {
		t.Fatalf("explainer execution = %#v, want completed explainer execution", executions[4])
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	var sawAgentError, sawRunComplete, sawRunError bool
	for _, event := range events {
		switch event.EventType {
		case sqlite.EventTypeAgentError:
			sawAgentError = true
		case sqlite.EventTypeRunComplete:
			sawRunComplete = true
		case sqlite.EventTypeRunError:
			sawRunError = true
		}
	}
	if !sawAgentError {
		t.Fatal("expected agent_error event for coder failure")
	}
	if !sawRunComplete {
		t.Fatal("expected run_complete event after preserved agent failure")
	}
	if sawRunError {
		t.Fatal("did not expect run_error event when only coder failed")
	}

	orderMu.Lock()
	defer orderMu.Unlock()
	if indexOfRole(startOrder, sqlite.RoleReviewer) <= indexOfRole(startOrder, sqlite.RoleTester) {
		t.Fatalf("startOrder = %v, want reviewer after tester terminal state", startOrder)
	}
	if indexOfRole(startOrder, sqlite.RoleExplainer) <= indexOfRole(startOrder, sqlite.RoleReviewer) {
		t.Fatalf("startOrder = %v, want explainer after reviewer", startOrder)
	}
}

func TestService_ExecuteOrchestrationRunHaltsAfterCoderClarificationRequest(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Coder clarification halt")
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

	run, err := store.CreateAgentRun(ctx, session.ID, "Halt after coder clarification", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	startedRoles := make([]sqlite.AgentRole, 0, 5)
	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		switch role {
		case sqlite.RolePlanner:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				startedRoles = append(startedRoles, role)
				if handlers.OnToken != nil {
					handlers.OnToken("planner output")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleCoder:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				startedRoles = append(startedRoles, role)
				if handlers.OnToken != nil {
					handlers.OnToken("Would you like me to review your specific .env.example file and add appropriate comments?")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		case sqlite.RoleTester:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				startedRoles = append(startedRoles, role)
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		default:
			return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
				startedRoles = append(startedRoles, role)
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}}
		}
	})

	if err := service.executeOrchestrationRun(context.Background(), run, run.TaskText, cfg); err != nil {
		t.Fatalf("executeOrchestrationRun() error = %v", err)
	}

	runSummary, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if runSummary.State != sqlite.RunStateHalted {
		t.Fatalf("run.State = %q, want %q", runSummary.State, sqlite.RunStateHalted)
	}
	if runSummary.ErrorCode != "coder_clarification_required" {
		t.Fatalf("run.ErrorCode = %q, want coder_clarification_required", runSummary.ErrorCode)
	}

	executions, err := store.ListAgentExecutions(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 3 {
		t.Fatalf("len(executions) = %d, want 3", len(executions))
	}
	if executions[1].Role != sqlite.RoleCoder || executions[1].State != sqlite.AgentExecutionStateErrored {
		t.Fatalf("coder execution = %#v, want errored coder execution", executions[1])
	}
	if executions[1].ErrorCode != "coder_clarification_required" {
		t.Fatalf("coder execution error code = %q, want coder_clarification_required", executions[1].ErrorCode)
	}
	if len(startedRoles) != 3 || indexOfRole(startedRoles, sqlite.RoleReviewer) != -1 || indexOfRole(startedRoles, sqlite.RoleExplainer) != -1 {
		t.Fatalf("startedRoles = %v, want planner/coder/tester only before halt", startedRoles)
	}
}

func TestService_ExecuteOrchestrationRunHaltsAfterPlannerFailure(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Planner halt")
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

	run, err := store.CreateAgentRun(ctx, session.ID, "Stop after planner", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	startedRoles := make([]sqlite.AgentRole, 0, 5)
	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
			startedRoles = append(startedRoles, role)
			if role == sqlite.RolePlanner {
				if handlers.OnError != nil {
					handlers.OnError("agent_generation_failed", "Planner could not break the goal into stages.")
				}
				return nil
			}
			if handlers.OnComplete != nil {
				handlers.OnComplete("stop")
			}
			return nil
		}}
	})

	if err := service.executeOrchestrationRun(context.Background(), run, run.TaskText, cfg); err != nil {
		t.Fatalf("executeOrchestrationRun() error = %v", err)
	}

	runSummary, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if runSummary.State != sqlite.RunStateHalted {
		t.Fatalf("run.State = %q, want %q", runSummary.State, sqlite.RunStateHalted)
	}
	if runSummary.ErrorCode != "planner_required" {
		t.Fatalf("run.ErrorCode = %q, want planner_required", runSummary.ErrorCode)
	}

	executions, err := store.ListAgentExecutions(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 1 {
		t.Fatalf("len(executions) = %d, want 1", len(executions))
	}
	if executions[0].Role != sqlite.RolePlanner || executions[0].State != sqlite.AgentExecutionStateErrored {
		t.Fatalf("planner execution = %#v, want errored planner execution", executions[0])
	}
	if len(startedRoles) != 1 || startedRoles[0] != sqlite.RolePlanner {
		t.Fatalf("startedRoles = %v, want only planner", startedRoles)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected persisted orchestration events")
	}
	if events[len(events)-1].EventType != sqlite.EventTypeRunError {
		t.Fatalf("last event = %q, want %q", events[len(events)-1].EventType, sqlite.EventTypeRunError)
	}
	if !strings.Contains(events[len(events)-1].PayloadJSON, executions[0].ID) {
		t.Fatalf("run_error payload = %q, want planner agent_id %q", events[len(events)-1].PayloadJSON, executions[0].ID)
	}
}

func TestService_ExecuteOrchestrationRunHaltsAfterPlannerClarificationQuestion(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Planner clarification halt")
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

	run, err := store.CreateAgentRun(ctx, session.ID, "Stop after planner clarification", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	startedRoles := make([]sqlite.AgentRole, 0, 5)
	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
			startedRoles = append(startedRoles, role)
			if role == sqlite.RolePlanner {
				if handlers.OnToken != nil {
					handlers.OnToken("Would you like me to search for any related configuration files or documentation that might need updating to match these comments?")
				}
				if handlers.OnComplete != nil {
					handlers.OnComplete("stop")
				}
				return nil
			}
			if handlers.OnComplete != nil {
				handlers.OnComplete("stop")
			}
			return nil
		}}
	})

	if err := service.executeOrchestrationRun(context.Background(), run, run.TaskText, cfg); err != nil {
		t.Fatalf("executeOrchestrationRun() error = %v", err)
	}

	runSummary, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if runSummary.State != sqlite.RunStateHalted {
		t.Fatalf("run.State = %q, want %q", runSummary.State, sqlite.RunStateHalted)
	}
	if runSummary.ErrorCode != "planner_clarification_required" {
		t.Fatalf("run.ErrorCode = %q, want planner_clarification_required", runSummary.ErrorCode)
	}

	executions, err := store.ListAgentExecutions(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 1 {
		t.Fatalf("len(executions) = %d, want 1", len(executions))
	}
	if executions[0].Role != sqlite.RolePlanner || executions[0].State != sqlite.AgentExecutionStateErrored {
		t.Fatalf("planner execution = %#v, want errored planner execution", executions[0])
	}
	if executions[0].ErrorCode != "planner_clarification_required" {
		t.Fatalf("planner execution error code = %q, want planner_clarification_required", executions[0].ErrorCode)
	}
	if len(startedRoles) != 1 || startedRoles[0] != sqlite.RolePlanner {
		t.Fatalf("startedRoles = %v, want only planner", startedRoles)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected persisted orchestration events")
	}
	hasAgentError := false
	for _, event := range events {
		if event.EventType == sqlite.EventTypeAgentError {
			hasAgentError = true
			break
		}
	}
	if !hasAgentError {
		t.Fatal("expected planner agent_error event")
	}
	if events[len(events)-1].EventType != sqlite.EventTypeRunError {
		t.Fatalf("last event = %q, want %q", events[len(events)-1].EventType, sqlite.EventTypeRunError)
	}
	if !strings.Contains(events[len(events)-1].PayloadJSON, "planner_clarification_required") {
		t.Fatalf("run_error payload = %q, want planner_clarification_required", events[len(events)-1].PayloadJSON)
	}
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
		profile:      agents.Profile{Role: sqlite.RoleCoder, Model: config.DefaultCoderModel},
		started:      make(chan struct{}),
		releaseState: make(chan struct{}),
		release:      make(chan struct{}),
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

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

func TestService_OpenRunReplaysStoredOrchestrationEvents(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Replay orchestration")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay node details", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeAgentSpawned, sqlite.RolePlanner, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","role":"planner","model":"`+run.Model+`","label":"Planner","spawn_order":1,"occurred_at":"2026-03-24T12:00:00Z"}`); err != nil {
		t.Fatalf("AppendRunEvent() agent_spawned error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeTaskAssigned, sqlite.RolePlanner, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","role":"planner","model":"`+run.Model+`","task_text":"Break the goal into stages.","occurred_at":"2026-03-24T12:00:01Z"}`); err != nil {
		t.Fatalf("AppendRunEvent() task_assigned error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, sqlite.RolePlanner, run.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","role":"planner","model":"`+run.Model+`","text":"planner transcript","occurred_at":"2026-03-24T12:00:02Z"}`); err != nil {
		t.Fatalf("AppendRunEvent() token error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeRunComplete, sqlite.RoleExplainer, config.DefaultExplainerModel, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_explainer_5","summary":"Finished orchestration.","occurred_at":"2026-03-24T12:00:03Z"}`); err != nil {
		t.Fatalf("AppendRunEvent() run_complete error = %v", err)
	}

	envelopes := make([]StreamEnvelope, 0, 4)
	_, err = service.OpenRun(ctx, OpenRunInput{SessionID: session.ID, RunID: run.ID}, func(envelope StreamEnvelope) error {
		envelopes = append(envelopes, envelope)
		return nil
	})
	if err != nil {
		t.Fatalf("OpenRun() error = %v", err)
	}
	if len(envelopes) != 4 {
		t.Fatalf("len(envelopes) = %d, want 4", len(envelopes))
	}
	if envelopes[0].Type != sqlite.EventTypeAgentSpawned || envelopes[1].Type != sqlite.EventTypeTaskAssigned || envelopes[2].Type != sqlite.EventTypeToken || envelopes[3].Type != sqlite.EventTypeRunComplete {
		t.Fatalf("envelopes = %#v, want orchestration replay order", envelopes)
	}
	for index, envelope := range envelopes {
		payload, ok := envelope.Payload.(map[string]any)
		if !ok {
			t.Fatalf("envelopes[%d].Payload = %#v, want map payload", index, envelope.Payload)
		}
		if payload["replay"] != true {
			t.Fatalf("envelopes[%d].payload.replay = %v, want true", index, payload["replay"])
		}
		if payload["sequence"] != float64(index+1) && payload["sequence"] != int64(index+1) {
			t.Fatalf("envelopes[%d].payload.sequence = %v, want %d", index, payload["sequence"], index+1)
		}
	}
	spawnPayload := envelopes[0].Payload.(map[string]any)
	if spawnPayload["agent_id"] != "agent_planner_1" {
		t.Fatalf("spawnPayload.agent_id = %v, want agent_planner_1", spawnPayload["agent_id"])
	}
	tokenPayload := envelopes[2].Payload.(map[string]any)
	if tokenPayload["text"] != "planner transcript" {
		t.Fatalf("tokenPayload.text = %v, want planner transcript", tokenPayload["text"])
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

	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return blockingRunner{}
	})

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

func TestService_CancelRunFinalizesOrphanedActiveRun(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Cancel orphaned run")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	run, err := store.CreateAgentRun(ctx, session.ID, "Cancel me after restart", sqlite.RoleTester, config.DefaultTesterModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	run.State = sqlite.RunStateToolRunning
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}

	snapshot, err := service.CancelRun(ctx, CancelRunInput{SessionID: session.ID, RunID: run.ID}, nil)
	if err != nil {
		t.Fatalf("CancelRun() error = %v", err)
	}
	if snapshot.ActiveRunID != "" {
		t.Fatalf("snapshot.ActiveRunID = %q, want empty", snapshot.ActiveRunID)
	}

	storedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if storedRun.State != sqlite.RunStateErrored {
		t.Fatalf("storedRun.State = %q, want %q", storedRun.State, sqlite.RunStateErrored)
	}
	if storedRun.ErrorCode != "run_cancelled" {
		t.Fatalf("storedRun.ErrorCode = %q, want run_cancelled", storedRun.ErrorCode)
	}
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

func TestServiceEmitToolEventsUseStageRoleAndModelFromRunContext(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Stage tool events")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Record tester tool events", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	stageCtx := withRunExecutionContext(ctx, runExecutionContext{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Role:      sqlite.RoleTester,
		Model:     config.DefaultTesterModel,
		Emit:      func(StreamEnvelope) error { return nil },
	})

	if err := service.emitToolCall(stageCtx, run.ID, agents.ToolCallEvent{
		ToolCallID:   "call_stage",
		ToolName:     agents.ToolWriteFile,
		InputPreview: map[string]any{"path": "tests/generated/smoke_test.sh"},
	}, nil); err != nil {
		t.Fatalf("emitToolCall() error = %v", err)
	}
	if err := service.emitToolResult(stageCtx, run.ID, agents.ToolResultEvent{
		ToolCallID:    "call_stage",
		ToolName:      agents.ToolWriteFile,
		Status:        "completed",
		ResultPreview: map[string]any{"path": "tests/generated/smoke_test.sh"},
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
	if events[0].Role != sqlite.RoleTester || events[0].Model != config.DefaultTesterModel {
		t.Fatalf("tool call event = %#v, want tester role/model", events[0])
	}
	if events[1].Role != sqlite.RoleTester || events[1].Model != config.DefaultTesterModel {
		t.Fatalf("tool result event = %#v, want tester role/model", events[1])
	}
}

func TestExecuteStageEmitsToolEventsForOrchestrationAgents(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Execute stage tool events")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Tester uses a tool", sqlite.RolePlanner, config.DefaultPlannerModel)
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	service.SetAgentFactory(func(cfg config.Config, role sqlite.AgentRole) agents.Agent {
		return scriptedPromptOnlyAgent{profile: orchestrationProfile(role), run: func(_ context.Context, _ string, handlers agents.StreamEventHandlers) error {
			if handlers.OnToolCall != nil {
				handlers.OnToolCall(agents.ToolCallEvent{
					ToolCallID:   "call_exec_stage",
					ToolName:     agents.ToolWriteFile,
					InputPreview: map[string]any{"path": "tests/generated/smoke_test.sh"},
				})
			}
			if handlers.OnToolResult != nil {
				handlers.OnToolResult(agents.ToolResultEvent{
					ToolCallID:    "call_exec_stage",
					ToolName:      agents.ToolWriteFile,
					Status:        "completed",
					ResultPreview: map[string]any{"path": "tests/generated/smoke_test.sh"},
				})
			}
			if handlers.OnComplete != nil {
				handlers.OnComplete("stop")
			}
			return nil
		}}
	})

	if _, err := service.executeStage(ctx, run, cfg, 2, sqlite.RoleTester, "Emit tool events"); err != nil {
		t.Fatalf("executeStage() error = %v", err)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	toolCallSeen := false
	toolResultSeen := false
	for _, event := range events {
		if event.EventType == sqlite.EventTypeToolCall && event.Role == sqlite.RoleTester {
			toolCallSeen = true
		}
		if event.EventType == sqlite.EventTypeToolResult && event.Role == sqlite.RoleTester {
			toolResultSeen = true
		}
	}
	if !toolCallSeen || !toolResultSeen {
		t.Fatalf("events = %#v, want tester tool_call and tool_result", events)
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
	profile      agents.Profile
	started      chan struct{}
	releaseState chan struct{}
	release      chan struct{}
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
		if handlers.OnComplete != nil {
			handlers.OnComplete("stop")
		}
		return nil
	}
	return a.run(ctx, task, handlers)
}

func orchestrationProfile(role sqlite.AgentRole) agents.Profile {
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

func indexOfRole(roles []sqlite.AgentRole, target sqlite.AgentRole) int {
	for index, role := range roles {
		if role == target {
			return index
		}
	}
	return -1
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
