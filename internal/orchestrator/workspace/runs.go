package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func (s *Service) SubmitRun(ctx context.Context, input SubmitRunInput, emit func(StreamEnvelope) error) (WorkspaceSnapshot, error) {
	if strings.TrimSpace(input.Task) == "" {
		return WorkspaceSnapshot{}, fmt.Errorf("Relay needs a task before it can start an agent run")
	}
	if _, err := s.store.GetSession(ctx, input.SessionID); err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("load target session: %w", err)
	}

	cfg, _, err := config.Load(s.paths)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	if !cfg.HasOpenRouterKey() {
		return WorkspaceSnapshot{}, fmt.Errorf("OpenRouter is not configured yet. Save an API key in preferences before starting a run")
	}

	agent := s.agentFactory(cfg, sqlite.RolePlanner)
	profile := agent.Profile()

	run, err := s.store.CreateAgentRun(ctx, input.SessionID, input.Task, profile.Role, profile.Model)
	if errors.Is(err, sqlite.ErrActiveRunExists) {
		return WorkspaceSnapshot{}, fmt.Errorf("Relay already has an active run. Wait for it to finish before starting another task")
	}
	if err != nil {
		return WorkspaceSnapshot{}, fmt.Errorf("create agent run: %w", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.registerActiveRun(run.ID, cancel)
	s.attachRunSubscriber(ctx, run.ID, emit, false)
	go s.executeRun(runCtx, run, input.Task, cfg)

	snapshot, err := s.Bootstrap(ctx, input.SessionID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}

	return snapshot, nil
}

func (s *Service) CancelRun(ctx context.Context, input CancelRunInput, _ func(StreamEnvelope) error) (WorkspaceSnapshot, error) {
	run, err := s.store.GetAgentRun(ctx, input.RunID)
	if err != nil {
		return WorkspaceSnapshot{}, err
	}
	if run.SessionID != input.SessionID {
		return WorkspaceSnapshot{}, fmt.Errorf("run %s does not belong to session %s", input.RunID, input.SessionID)
	}
	if !run.Active() {
		return WorkspaceSnapshot{}, fmt.Errorf("run %s is no longer active", input.RunID)
	}

	cancel, ok := s.activeRunCancel(run.ID)
	if !ok {
		if err := s.failRun(ctx, run.ID, "run_cancelled", "Relay cancelled the active run.", nil); err != nil {
			return WorkspaceSnapshot{}, err
		}
		return s.Bootstrap(ctx, input.SessionID)
	}
	cancel()

	return s.Bootstrap(ctx, input.SessionID)
}

func (s *Service) executeRun(runCtx context.Context, run sqlite.AgentRun, task string, cfg config.Config) {
	defer s.clearActiveRun(run.ID)
	runCtx = withRunExecutionContext(runCtx, runExecutionContext{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Emit: func(envelope StreamEnvelope) error {
			return s.dispatchRunEnvelope(run.ID, envelope)
		},
		Role:  run.Role,
		Model: run.Model,
	})
	if s.forceLegacyRunnerPath {
		s.executeLegacyRun(runCtx, run, task, s.runnerFactory(cfg, task))
		return
	}
	if err := s.executeOrchestrationRun(runCtx, run, task, cfg); err == nil {
		return
	} else if errors.Is(err, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
		_ = s.emitRunError(context.WithoutCancel(runCtx), run, "run_cancelled", "Relay cancelled the active run.", nil)
		return
	} else {
		_ = s.emitRunError(context.WithoutCancel(runCtx), run, "run_failed", "Relay could not complete the orchestration run.", nil)
	}
}

func (s *Service) executeLegacyRun(runCtx context.Context, run sqlite.AgentRun, task string, runner agents.Runner) {
	profile := runner.Profile()
	runCtx = withRunExecutionContext(runCtx, runExecutionContext{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Emit: func(envelope StreamEnvelope) error {
			return s.dispatchRunEnvelope(run.ID, envelope)
		},
		Role:  profile.Role,
		Model: profile.Model,
	})

	terminalEventWritten := false
	markTerminal := func() {
		terminalEventWritten = true
	}

	err := runner.Run(runCtx, task, agents.StreamEventHandlers{
		OnStateChange: func(message string) {
			_ = s.transitionRun(runCtx, run.ID, strings.TrimSpace(message), message, nil)
		},
		OnToken: func(text string) {
			_ = s.emitToken(runCtx, run.ID, text, nil)
		},
		OnToolCall: func(event agents.ToolCallEvent) {
			_ = s.emitToolCall(runCtx, run.ID, event, nil)
		},
		OnToolResult: func(event agents.ToolResultEvent) {
			_ = s.emitToolResult(runCtx, run.ID, event, nil)
		},
		OnComplete: func(finishReason string) {
			markTerminal()
			_ = s.completeRun(runCtx, run.ID, finishReason, nil)
		},
		OnError: func(code string, message string) {
			markTerminal()
			_ = s.failRun(runCtx, run.ID, code, message, nil)
		},
	})
	if err == nil || terminalEventWritten {
		return
	}

	if errors.Is(err, context.Canceled) || errors.Is(runCtx.Err(), context.Canceled) {
		_ = s.failRun(context.WithoutCancel(runCtx), run.ID, "run_cancelled", "Relay cancelled the active run.", nil)
		return
	}

	_ = s.failRun(runCtx, run.ID, "run_failed", "Relay could not complete the agent run.", nil)
}

func (s *Service) transitionRun(ctx context.Context, runID string, nextState string, message string, emit func(StreamEnvelope) error) error {
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return err
	}
	run.State = nextState
	if err := s.store.UpdateAgentRun(ctx, run); err != nil {
		return err
	}
	return s.emitStateChange(ctx, run, nextState, message, false, emit)
}

func (s *Service) emitStateChange(ctx context.Context, run sqlite.AgentRun, state string, message string, replay bool, emit func(StreamEnvelope) error) error {
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"replay":      replay,
		"role":        run.Role,
		"model":       run.Model,
		"state":       state,
		"message":     message,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal state payload: %w", err)
	}
	event, err := s.store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeStateChange, run.Role, run.Model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = event.Sequence
	if err := s.dispatchRunEnvelope(run.ID, StreamEnvelope{Type: sqlite.EventTypeStateChange, Payload: payload}); err != nil {
		return nil
	}
	return nil
}

func (s *Service) emitToken(ctx context.Context, runID string, text string, emit func(StreamEnvelope) error) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	firstVisibleToken := run.FirstTokenAt == nil
	if run.FirstTokenAt == nil || run.State == sqlite.RunStateToolRunning {
		if run.FirstTokenAt == nil {
			run.FirstTokenAt = &now
		}
		run.State = sqlite.RunStateThinking
		if err := s.store.UpdateAgentRun(ctx, run); err != nil {
			return err
		}
	}
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"replay":      false,
		"role":        run.Role,
		"model":       run.Model,
		"text":        text,
		"occurred_at": now.Format(time.RFC3339),
	}
	if firstVisibleToken {
		payload["first_token_latency_ms"] = now.Sub(run.StartedAt).Milliseconds()
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal token payload: %w", err)
	}
	event, err := s.store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToken, run.Role, run.Model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = event.Sequence
	if err := s.dispatchRunEnvelope(run.ID, StreamEnvelope{Type: sqlite.EventTypeToken, Payload: payload}); err != nil {
		return nil
	}
	return nil
}

func (s *Service) emitToolCall(ctx context.Context, runID string, event agents.ToolCallEvent, emit func(StreamEnvelope) error) error {
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return err
	}
	role := run.Role
	model := run.Model
	if runContext, ok := runExecutionContextFromContext(ctx); ok {
		if runContext.Role != "" {
			role = runContext.Role
		}
		if strings.TrimSpace(runContext.Model) != "" {
			model = runContext.Model
		}
	}
	run.State = sqlite.RunStateToolRunning
	if err := s.store.UpdateAgentRun(ctx, run); err != nil {
		return err
	}

	payload := map[string]any{
		"session_id":    run.SessionID,
		"run_id":        run.ID,
		"replay":        false,
		"role":          role,
		"model":         model,
		"tool_call_id":  event.ToolCallID,
		"tool_name":     string(event.ToolName),
		"input_preview": event.InputPreview,
		"occurred_at":   time.Now().UTC().Format(time.RFC3339),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal tool call payload: %w", err)
	}
	storedEvent, err := s.store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToolCall, role, model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = storedEvent.Sequence
	if err := s.dispatchRunEnvelope(run.ID, StreamEnvelope{Type: sqlite.EventTypeToolCall, Payload: payload}); err != nil {
		return nil
	}
	return nil
}

func (s *Service) emitToolResult(ctx context.Context, runID string, event agents.ToolResultEvent, emit func(StreamEnvelope) error) error {
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return err
	}
	role := run.Role
	model := run.Model
	if runContext, ok := runExecutionContextFromContext(ctx); ok {
		if runContext.Role != "" {
			role = runContext.Role
		}
		if strings.TrimSpace(runContext.Model) != "" {
			model = runContext.Model
		}
	}

	payload := map[string]any{
		"session_id":     run.SessionID,
		"run_id":         run.ID,
		"replay":         false,
		"role":           role,
		"model":          model,
		"tool_call_id":   event.ToolCallID,
		"tool_name":      string(event.ToolName),
		"status":         event.Status,
		"result_preview": event.ResultPreview,
		"occurred_at":    time.Now().UTC().Format(time.RFC3339),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal tool result payload: %w", err)
	}
	storedEvent, err := s.store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeToolResult, role, model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = storedEvent.Sequence
	if err := s.dispatchRunEnvelope(run.ID, StreamEnvelope{Type: sqlite.EventTypeToolResult, Payload: payload}); err != nil {
		return nil
	}
	return nil
}

func (s *Service) completeRun(ctx context.Context, runID string, finishReason string, emit func(StreamEnvelope) error) error {
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &now
	if err := s.store.UpdateAgentRun(ctx, run); err != nil {
		return err
	}
	payload := map[string]any{
		"session_id":    run.SessionID,
		"run_id":        run.ID,
		"replay":        false,
		"role":          run.Role,
		"model":         run.Model,
		"finish_reason": finishReason,
		"occurred_at":   now.Format(time.RFC3339),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal complete payload: %w", err)
	}
	event, err := s.store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeComplete, run.Role, run.Model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = event.Sequence
	if err := s.dispatchRunEnvelope(run.ID, StreamEnvelope{Type: sqlite.EventTypeComplete, Payload: payload}); err != nil {
		return nil
	}
	return nil
}

func (s *Service) failRun(ctx context.Context, runID string, code string, message string, emit func(StreamEnvelope) error) error {
	run, err := s.store.GetAgentRun(ctx, runID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	run.State = sqlite.RunStateErrored
	run.CompletedAt = &now
	run.ErrorCode = strings.TrimSpace(code)
	run.ErrorMessage = strings.TrimSpace(message)
	if err := s.store.UpdateAgentRun(ctx, run); err != nil {
		return err
	}
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"replay":      false,
		"role":        run.Role,
		"model":       run.Model,
		"code":        code,
		"message":     message,
		"terminal":    true,
		"occurred_at": now.Format(time.RFC3339),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal error payload: %w", err)
	}
	event, err := s.store.AppendRunEvent(ctx, run.ID, sqlite.EventTypeError, run.Role, run.Model, string(encoded))
	if err != nil {
		return err
	}
	payload["sequence"] = event.Sequence
	if err := s.dispatchRunEnvelope(run.ID, StreamEnvelope{Type: sqlite.EventTypeError, Payload: payload}); err != nil {
		return nil
	}
	return nil
}
