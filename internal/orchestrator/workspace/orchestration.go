package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

type stageResult struct {
	execution  sqlite.AgentExecution
	transcript string
	completion agents.CompletionMetadata
	failed     bool
	errCode    string
	errMessage string
	runErr     error
}

func (s *Service) executeOrchestrationRun(ctx context.Context, run sqlite.AgentRun, task string, cfg config.Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := s.transitionRun(ctx, run.ID, sqlite.RunStateActive, "Relay is coordinating the planner, coder, tester, reviewer, and explainer.", nil); err != nil {
		return err
	}

	transcripts := make(map[sqlite.AgentRole]string)
	results := make(map[sqlite.AgentRole]stageResult)

	planner, err := s.executeStage(ctx, run, cfg, 1, sqlite.RolePlanner, task)
	if err != nil {
		s.logStageExecutionFailure(run, sqlite.RolePlanner, executionRef(planner.execution), err)
		return s.emitRunError(context.WithoutCancel(ctx), run, "run_stage_failed", runStageFailureMessage(sqlite.RolePlanner), executionRef(planner.execution))
	}
	results[sqlite.RolePlanner] = planner
	transcripts[sqlite.RolePlanner] = planner.transcript
	if planner.failed {
		if clarificationRequiredCode(sqlite.RolePlanner) == planner.errCode {
			return s.emitRunError(context.WithoutCancel(ctx), run, planner.errCode, planner.errMessage, executionRef(planner.execution))
		}
		return s.emitRunError(context.WithoutCancel(ctx), run, "planner_required", "The run stopped because the planner did not complete and downstream work could not continue.", executionRef(planner.execution))
	}
	if stageNeedsClarification(sqlite.RolePlanner, planner.transcript, planner.errMessage) {
		return s.emitRunError(context.WithoutCancel(ctx), run, clarificationRequiredCode(sqlite.RolePlanner), clarificationRequiredRunMessage(sqlite.RolePlanner), executionRef(planner.execution))
	}

	parallelSpecs := []struct {
		order int
		role  sqlite.AgentRole
		task  string
	}{
		{order: 2, role: sqlite.RoleCoder, task: buildRoleTask(sqlite.RoleCoder, task, transcripts)},
		{order: 3, role: sqlite.RoleTester, task: buildRoleTask(sqlite.RoleTester, task, transcripts)},
	}

	for _, spec := range parallelSpecs {
		if err := s.emitHandoffEvent(ctx, run, sqlite.RolePlanner, spec.role, sqlite.EventTypeHandoffStart, "planner_completed"); err != nil {
			return err
		}
	}

	resultsCh := make(chan stageResult, len(parallelSpecs))
	var wg sync.WaitGroup
	for _, spec := range parallelSpecs {
		spec := spec
		wg.Go(func() {
			result, stageErr := s.executeStage(ctx, run, cfg, spec.order, spec.role, spec.task)
			if stageErr != nil {
				result.runErr = stageErr
				if !errors.Is(stageErr, context.Canceled) {
					cancel()
				}
			}
			resultsCh <- result
		})
	}
	wg.Wait()
	close(resultsCh)

	parallelResults := make([]stageResult, 0, len(parallelSpecs))
	var parallelFailure *stageResult
	for result := range resultsCh {
		if result.execution.Role != "" {
			results[result.execution.Role] = result
			transcripts[result.execution.Role] = result.transcript
			parallelResults = append(parallelResults, result)
		}
		if result.failed && strings.HasSuffix(result.errCode, "_clarification_required") && parallelFailure == nil {
			failedResult := result
			parallelFailure = &failedResult
			continue
		}
		if result.runErr != nil && !errors.Is(result.runErr, context.Canceled) && parallelFailure == nil {
			failedResult := result
			parallelFailure = &failedResult
		}
	}
	if parallelFailure != nil {
		if strings.HasSuffix(parallelFailure.errCode, "_clarification_required") {
			return s.emitRunError(context.WithoutCancel(ctx), run, parallelFailure.errCode, parallelFailure.errMessage, executionRef(parallelFailure.execution))
		}
		s.logStageExecutionFailure(run, parallelFailure.execution.Role, executionRef(parallelFailure.execution), parallelFailure.runErr)
		return s.emitRunError(context.WithoutCancel(ctx), run, "run_stage_failed", runStageFailureMessage(parallelFailure.execution.Role), executionRef(parallelFailure.execution))
	}
	for _, result := range parallelResults {
		if err := s.emitHandoffEvent(ctx, run, sqlite.RolePlanner, result.execution.Role, sqlite.EventTypeHandoffComplete, "planner_completed"); err != nil {
			return err
		}
	}

	if err := s.emitHandoffEvent(ctx, run, sqlite.RoleCoder, sqlite.RoleReviewer, sqlite.EventTypeHandoffStart, "parallel_agents_completed"); err != nil {
		return err
	}
	if err := s.emitHandoffEvent(ctx, run, sqlite.RoleTester, sqlite.RoleReviewer, sqlite.EventTypeHandoffStart, "parallel_agents_completed"); err != nil {
		return err
	}

	reviewer, err := s.executeStage(ctx, run, cfg, 4, sqlite.RoleReviewer, buildRoleTask(sqlite.RoleReviewer, task, transcripts))
	if err != nil {
		s.logStageExecutionFailure(run, sqlite.RoleReviewer, executionRef(reviewer.execution), err)
		return s.emitRunError(context.WithoutCancel(ctx), run, "run_stage_failed", runStageFailureMessage(sqlite.RoleReviewer), executionRef(reviewer.execution))
	}
	results[sqlite.RoleReviewer] = reviewer
	transcripts[sqlite.RoleReviewer] = reviewer.transcript
	if reviewer.failed && strings.HasSuffix(reviewer.errCode, "_clarification_required") {
		return s.emitRunError(context.WithoutCancel(ctx), run, reviewer.errCode, reviewer.errMessage, executionRef(reviewer.execution))
	}
	if err := s.emitHandoffEvent(ctx, run, sqlite.RoleCoder, sqlite.RoleReviewer, sqlite.EventTypeHandoffComplete, "parallel_agents_completed"); err != nil {
		return err
	}
	if err := s.emitHandoffEvent(ctx, run, sqlite.RoleTester, sqlite.RoleReviewer, sqlite.EventTypeHandoffComplete, "parallel_agents_completed"); err != nil {
		return err
	}

	if err := s.emitHandoffEvent(ctx, run, sqlite.RoleReviewer, sqlite.RoleExplainer, sqlite.EventTypeHandoffStart, "reviewer_finished"); err != nil {
		return err
	}

	explainer, err := s.executeStage(ctx, run, cfg, 5, sqlite.RoleExplainer, buildRoleTask(sqlite.RoleExplainer, task, transcripts))
	if err != nil {
		s.logStageExecutionFailure(run, sqlite.RoleExplainer, executionRef(explainer.execution), err)
		return s.emitRunError(context.WithoutCancel(ctx), run, "run_stage_failed", runStageFailureMessage(sqlite.RoleExplainer), executionRef(explainer.execution))
	}
	results[sqlite.RoleExplainer] = explainer
	transcripts[sqlite.RoleExplainer] = explainer.transcript
	if explainer.failed && strings.HasSuffix(explainer.errCode, "_clarification_required") {
		return s.emitRunError(context.WithoutCancel(ctx), run, explainer.errCode, explainer.errMessage, executionRef(explainer.execution))
	}
	if err := s.emitHandoffEvent(ctx, run, sqlite.RoleReviewer, sqlite.RoleExplainer, sqlite.EventTypeHandoffComplete, "reviewer_finished"); err != nil {
		return err
	}

	return s.emitRunComplete(ctx, run, buildRunSummary(transcripts, results), executionRef(explainer.execution), &explainer.completion)
}

func (s *Service) executeStage(ctx context.Context, run sqlite.AgentRun, cfg config.Config, spawnOrder int, role sqlite.AgentRole, task string) (stageResult, error) {
	model := modelForRole(cfg.Agents.WithDefaults(), role)
	execution := sqlite.AgentExecution{
		ID:         executionID(run.ID, role, spawnOrder),
		RunID:      run.ID,
		Role:       role,
		Model:      model,
		State:      sqlite.AgentExecutionStateQueued,
		TaskText:   strings.TrimSpace(task),
		SpawnOrder: spawnOrder,
	}
	storedExecution, err := s.store.CreateAgentExecution(ctx, execution)
	if err != nil {
		return stageResult{}, err
	}
	if err := s.emitAgentSpawned(ctx, run, storedExecution); err != nil {
		return stageResult{}, err
	}

	now := time.Now().UTC()
	storedExecution.State = sqlite.AgentExecutionStateAssigned
	storedExecution.StartedAt = &now
	if err := s.store.UpdateAgentExecution(ctx, storedExecution); err != nil {
		return stageResult{}, err
	}
	if err := s.emitTaskAssigned(ctx, run, storedExecution); err != nil {
		return stageResult{}, err
	}

	agent := s.agentFactory(cfg, role)
	stageCtx := withRunExecutionContext(ctx, runExecutionContext{
		SessionID: run.SessionID,
		RunID:     run.ID,
		Emit: func(envelope StreamEnvelope) error {
			return s.dispatchRunEnvelope(run.ID, envelope)
		},
		Role:  role,
		Model: model,
	})
	result := stageResult{execution: storedExecution}
	firstToken := true
	completed := false
	err = agent.Run(stageCtx, task, agents.StreamEventHandlers{
		OnStateChange: func(message string) {
			if strings.TrimSpace(message) == "" {
				message = sqlite.AgentExecutionStateThinking
			}
			storedExecution.State = sqlite.AgentExecutionStateThinking
			_ = s.store.UpdateAgentExecution(stageCtx, storedExecution)
			_ = s.emitAgentStateChanged(stageCtx, run, storedExecution, sqlite.AgentExecutionStateThinking, roleProgressMessage(role, message), nil)
		},
		OnToken: func(text string) {
			result.transcript += text
			if firstToken {
				firstToken = false
				storedExecution.State = sqlite.AgentExecutionStateStreaming
				_ = s.store.UpdateAgentExecution(stageCtx, storedExecution)
				_ = s.emitAgentStateChanged(stageCtx, run, storedExecution, sqlite.AgentExecutionStateStreaming, roleStreamingMessage(role), nil)
			}
			_ = s.emitAgentToken(stageCtx, run, storedExecution, text)
		},
		OnToolCall: func(event agents.ToolCallEvent) {
			_ = s.emitToolCall(stageCtx, run.ID, event, nil)
		},
		OnToolResult: func(event agents.ToolResultEvent) {
			_ = s.emitToolResult(stageCtx, run.ID, event, nil)
		},
		OnComplete: func(metadata agents.CompletionMetadata) {
			completed = true
			if metadata.ContextLimit == nil {
				metadata.ContextLimit = s.resolveModelContextLimit(context.WithoutCancel(stageCtx), cfg, model)
			}
			result.completion = metadata
		},
		OnError: func(code string, message string) {
			completedAt := time.Now().UTC()
			storedExecution.State = sqlite.AgentExecutionStateErrored
			storedExecution.CompletedAt = &completedAt
			storedExecution.ErrorCode = code
			storedExecution.ErrorMessage = message
			result.failed = true
			result.errCode = code
			result.errMessage = message
			_ = s.store.UpdateAgentExecution(stageCtx, storedExecution)
			_ = s.emitAgentError(stageCtx, run, storedExecution, code, message)
		},
	})
	if err != nil && result.failed {
		err = nil
	}
	if err == nil && completed && !result.failed {
		if stageNeedsClarification(role, result.transcript, result.errMessage) {
			completedAt := time.Now().UTC()
			storedExecution.State = sqlite.AgentExecutionStateErrored
			storedExecution.CompletedAt = &completedAt
			storedExecution.ErrorCode = clarificationRequiredCode(role)
			storedExecution.ErrorMessage = clarificationRequiredRunMessage(role)
			result.failed = true
			result.errCode = storedExecution.ErrorCode
			result.errMessage = storedExecution.ErrorMessage
			_ = s.store.UpdateAgentExecution(ctx, storedExecution)
			_ = s.emitAgentError(ctx, run, storedExecution, storedExecution.ErrorCode, storedExecution.ErrorMessage)
		} else {
			completedAt := time.Now().UTC()
			storedExecution.State = sqlite.AgentExecutionStateCompleted
			storedExecution.CompletedAt = &completedAt
			_ = s.store.UpdateAgentExecution(ctx, storedExecution)
			_ = s.emitAgentStateChanged(ctx, run, storedExecution, sqlite.AgentExecutionStateCompleted, roleCompletedMessage(role), &result.completion)
		}
	}
	if result.failed && result.errCode == "agent_generation_failed" && stageNeedsClarification(role, result.transcript, result.errMessage) {
		storedExecution.ErrorCode = clarificationRequiredCode(role)
		storedExecution.ErrorMessage = clarificationRequiredRunMessage(role)
		result.errCode = storedExecution.ErrorCode
		result.errMessage = storedExecution.ErrorMessage
		_ = s.store.UpdateAgentExecution(ctx, storedExecution)
	}
	result.execution = storedExecution
	result.runErr = err
	return result, err
}

func stageNeedsClarification(role sqlite.AgentRole, transcript string, errMessage string) bool {
	text := strings.ToLower(strings.TrimSpace(strings.Join([]string{transcript, errMessage}, "\n")))
	if text == "" {
		return false
	}

	phrases := []string{
		"would you like me to",
		"would you like me",
		"do you want me to",
		"do you want me",
		"should i ",
		"can you provide",
		"can you share",
		"could you provide",
		"could you share",
		"please share",
		"please provide",
		"if you'd like, i can",
		"if you want, i can",
		"let me know if you'd like me",
	}
	for _, phrase := range phrases {
		if strings.Contains(text, phrase) {
			return true
		}
	}

	return false
}

func clarificationRequiredCode(role sqlite.AgentRole) string {
	return string(role) + "_clarification_required"
}

func clarificationRequiredRunMessage(role sqlite.AgentRole) string {
	return fmt.Sprintf("The run stopped because the %s asked for user clarification instead of producing actionable output.", strings.ToLower(labelForRole(role)))
}

func buildRoleTask(role sqlite.AgentRole, originalTask string, transcripts map[sqlite.AgentRole]string) string {
	plannerOutput := strings.TrimSpace(transcripts[sqlite.RolePlanner])
	coderOutput := strings.TrimSpace(transcripts[sqlite.RoleCoder])
	testerOutput := strings.TrimSpace(transcripts[sqlite.RoleTester])
	reviewerOutput := strings.TrimSpace(transcripts[sqlite.RoleReviewer])

	switch role {
	case sqlite.RoleCoder:
		return fmt.Sprintf("Original goal:\n%s\n\nPlanner output:\n%s\n\nRespond with a focused implementation approach only.", originalTask, plannerOutput)
	case sqlite.RoleTester:
		return fmt.Sprintf("Original goal:\n%s\n\nPlanner output:\n%s\n\nRespond with validation strategy, edge cases, and concrete checks only.", originalTask, plannerOutput)
	case sqlite.RoleReviewer:
		return fmt.Sprintf("Original goal:\n%s\n\nPlanner output:\n%s\n\nCoder output:\n%s\n\nTester output:\n%s\n\nReview the combined work for risks, regressions, and gaps.", originalTask, plannerOutput, coderOutput, testerOutput)
	case sqlite.RoleExplainer:
		return fmt.Sprintf("Original goal:\n%s\n\nPlanner output:\n%s\n\nCoder output:\n%s\n\nTester output:\n%s\n\nReviewer output:\n%s\n\nSummarize the orchestration in plain language.", originalTask, plannerOutput, coderOutput, testerOutput, reviewerOutput)
	default:
		return originalTask
	}
}

func buildRunSummary(transcripts map[sqlite.AgentRole]string, results map[sqlite.AgentRole]stageResult) string {
	if text := strings.TrimSpace(transcripts[sqlite.RoleExplainer]); text != "" {
		return text
	}
	failedRoles := make([]string, 0)
	for _, role := range []sqlite.AgentRole{sqlite.RoleCoder, sqlite.RoleTester, sqlite.RoleReviewer, sqlite.RoleExplainer} {
		if result, ok := results[role]; ok && result.failed {
			failedRoles = append(failedRoles, string(role))
		}
	}
	if len(failedRoles) == 0 {
		return "The orchestration completed with planner, coder, tester, reviewer, and explainer stages."
	}
	return fmt.Sprintf("The orchestration finished with preserved partial failures in: %s.", strings.Join(failedRoles, ", "))
}

func modelForRole(models config.AgentModels, role sqlite.AgentRole) string {
	switch role {
	case sqlite.RolePlanner:
		return models.Planner
	case sqlite.RoleReviewer:
		return models.Reviewer
	case sqlite.RoleTester:
		return models.Tester
	case sqlite.RoleExplainer:
		return models.Explainer
	default:
		return models.Coder
	}
}

func labelForRole(role sqlite.AgentRole) string {
	switch role {
	case sqlite.RolePlanner:
		return "Planner"
	case sqlite.RoleReviewer:
		return "Reviewer"
	case sqlite.RoleTester:
		return "Tester"
	case sqlite.RoleExplainer:
		return "Explainer"
	default:
		return "Coder"
	}
}

func roleProgressMessage(role sqlite.AgentRole, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return fmt.Sprintf("%s is working through the current handoff.", labelForRole(role))
}

func roleStreamingMessage(role sqlite.AgentRole) string {
	return fmt.Sprintf("%s is streaming visible output.", labelForRole(role))
}

func roleCompletedMessage(role sqlite.AgentRole) string {
	return fmt.Sprintf("%s completed its task.", labelForRole(role))
}

func runStageFailureMessage(role sqlite.AgentRole) string {
	label := strings.ToLower(labelForRole(role))
	if strings.TrimSpace(label) == "" {
		return "The run stopped because Relay could not finish the orchestration stage."
	}
	return fmt.Sprintf("The run stopped because Relay could not finish the %s stage.", label)
}

func executionRef(execution sqlite.AgentExecution) *sqlite.AgentExecution {
	if strings.TrimSpace(execution.ID) == "" {
		return nil
	}
	executionCopy := execution
	return &executionCopy
}

func executionID(runID string, role sqlite.AgentRole, spawnOrder int) string {
	return fmt.Sprintf("agent_%s_%s_%d", runID, role, spawnOrder)
}

func (s *Service) logStageExecutionFailure(run sqlite.AgentRun, role sqlite.AgentRole, execution *sqlite.AgentExecution, err error) {
	if err == nil {
		return
	}

	loggedRole := role
	loggedModel := run.Model
	args := []any{
		"session_id", run.SessionID,
		"run_id", run.ID,
		"role", loggedRole,
		"error", err,
	}
	if execution != nil {
		if execution.Role != "" {
			loggedRole = execution.Role
			args[5] = loggedRole
		}
		if execution.Model != "" {
			loggedModel = execution.Model
		}
		if strings.TrimSpace(execution.ID) != "" {
			args = append(args, "agent_id", execution.ID)
		}
	}
	if strings.TrimSpace(loggedModel) != "" {
		args = append(args, "model", loggedModel)
	}

	s.logger.Error("orchestration stage execution failed", args...)
}

func (s *Service) emitAgentSpawned(ctx context.Context, run sqlite.AgentRun, execution sqlite.AgentExecution) error {
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    execution.ID,
		"replay":      false,
		"role":        execution.Role,
		"model":       execution.Model,
		"label":       labelForRole(execution.Role),
		"spawn_order": execution.SpawnOrder,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeAgentSpawned, execution.Role, execution.Model, payload, nil)
}

func (s *Service) emitAgentStateChanged(ctx context.Context, run sqlite.AgentRun, execution sqlite.AgentExecution, state string, message string, completion *agents.CompletionMetadata) error {
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    execution.ID,
		"replay":      false,
		"role":        execution.Role,
		"model":       execution.Model,
		"state":       state,
		"message":     message,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	}
	if completion != nil {
		if completion.TokensUsed != nil {
			payload["tokens_used"] = *completion.TokensUsed
		}
		if completion.ContextLimit != nil {
			payload["context_limit"] = *completion.ContextLimit
		}
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeAgentStateChanged, execution.Role, execution.Model, payload, completion)
}

func (s *Service) emitTaskAssigned(ctx context.Context, run sqlite.AgentRun, execution sqlite.AgentExecution) error {
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    execution.ID,
		"replay":      false,
		"role":        execution.Role,
		"model":       execution.Model,
		"task_text":   execution.TaskText,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeTaskAssigned, execution.Role, execution.Model, payload, nil)
}

func (s *Service) emitHandoffEvent(ctx context.Context, run sqlite.AgentRun, fromRole sqlite.AgentRole, toRole sqlite.AgentRole, eventType string, reason string) error {
	payload := map[string]any{
		"session_id":    run.SessionID,
		"run_id":        run.ID,
		"agent_id":      executionID(run.ID, fromRole, spawnOrderForRole(fromRole)),
		"replay":        false,
		"from_agent_id": executionID(run.ID, fromRole, spawnOrderForRole(fromRole)),
		"to_agent_id":   executionID(run.ID, toRole, spawnOrderForRole(toRole)),
		"reason":        reason,
		"occurred_at":   time.Now().UTC().Format(time.RFC3339),
	}
	return s.appendAndDispatchEvent(ctx, run.ID, eventType, fromRole, "", payload, nil)
}

func (s *Service) emitAgentToken(ctx context.Context, run sqlite.AgentRun, execution sqlite.AgentExecution, text string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    execution.ID,
		"replay":      false,
		"role":        execution.Role,
		"model":       execution.Model,
		"text":        text,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeToken, execution.Role, execution.Model, payload, nil)
}

func (s *Service) emitAgentError(ctx context.Context, run sqlite.AgentRun, execution sqlite.AgentExecution, code string, message string) error {
	s.logger.Error("orchestration agent failed",
		"session_id", run.SessionID,
		"run_id", run.ID,
		"agent_id", execution.ID,
		"role", execution.Role,
		"model", execution.Model,
		"code", code,
		"message", message,
	)

	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    execution.ID,
		"replay":      false,
		"role":        execution.Role,
		"model":       execution.Model,
		"code":        code,
		"message":     message,
		"terminal":    true,
		"occurred_at": time.Now().UTC().Format(time.RFC3339),
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeAgentError, execution.Role, execution.Model, payload, nil)
}

func (s *Service) emitRunComplete(ctx context.Context, run sqlite.AgentRun, summary string, execution *sqlite.AgentExecution, completion *agents.CompletionMetadata) error {
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateCompleted
	run.CompletedAt = &completedAt
	if err := s.store.UpdateAgentRun(ctx, run); err != nil {
		return err
	}
	agentID := executionID(run.ID, sqlite.RoleExplainer, spawnOrderForRole(sqlite.RoleExplainer))
	role := sqlite.RoleExplainer
	model := ""
	if execution != nil {
		agentID = execution.ID
		role = execution.Role
		model = execution.Model
	}
	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    agentID,
		"replay":      false,
		"role":        role,
		"model":       model,
		"summary":     summary,
		"occurred_at": completedAt.Format(time.RFC3339),
	}
	if completion != nil {
		if completion.TokensUsed != nil {
			payload["tokens_used"] = *completion.TokensUsed
		}
		if completion.ContextLimit != nil {
			payload["context_limit"] = *completion.ContextLimit
		}
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeRunComplete, role, model, payload, completion)
}

func (s *Service) emitRunError(ctx context.Context, run sqlite.AgentRun, code string, message string, execution *sqlite.AgentExecution) error {
	completedAt := time.Now().UTC()
	run.State = sqlite.RunStateHalted
	run.ErrorCode = code
	run.ErrorMessage = message
	run.CompletedAt = &completedAt
	if err := s.store.UpdateAgentRun(ctx, run); err != nil {
		return err
	}
	agentID := ""
	role := run.Role
	model := run.Model
	if execution != nil {
		agentID = execution.ID
		role = execution.Role
		model = execution.Model
	}
	logArgs := []any{
		"session_id", run.SessionID,
		"run_id", run.ID,
		"role", role,
		"model", model,
		"code", code,
		"message", message,
	}
	if strings.TrimSpace(agentID) != "" {
		logArgs = append(logArgs, "agent_id", agentID)
	}
	if code == "run_cancelled" {
		s.logger.Info("orchestration run cancelled", logArgs...)
	} else {
		s.logger.Error("orchestration run halted", logArgs...)
	}

	payload := map[string]any{
		"session_id":  run.SessionID,
		"run_id":      run.ID,
		"agent_id":    agentID,
		"replay":      false,
		"role":        role,
		"model":       model,
		"code":        code,
		"message":     message,
		"terminal":    true,
		"occurred_at": completedAt.Format(time.RFC3339),
	}
	return s.appendAndDispatchEvent(ctx, run.ID, sqlite.EventTypeRunError, role, model, payload, nil)
}

func (s *Service) appendAndDispatchEvent(ctx context.Context, runID string, eventType string, role sqlite.AgentRole, model string, payload map[string]any, completion *agents.CompletionMetadata) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal orchestration payload: %w", err)
	}
	var tokensUsed *int
	var contextLimit *int
	if completion != nil {
		tokensUsed = completion.TokensUsed
		contextLimit = completion.ContextLimit
	}
	event, err := s.store.AppendRunEvent(ctx, runID, eventType, role, model, string(encoded), tokensUsed, contextLimit)
	if err != nil {
		return err
	}
	payload["sequence"] = event.Sequence
	return s.dispatchRunEnvelope(runID, StreamEnvelope{Type: eventType, Payload: payload})
}

func spawnOrderForRole(role sqlite.AgentRole) int {
	switch role {
	case sqlite.RolePlanner:
		return 1
	case sqlite.RoleCoder:
		return 2
	case sqlite.RoleTester:
		return 3
	case sqlite.RoleReviewer:
		return 4
	case sqlite.RoleExplainer:
		return 5
	default:
		return 0
	}
}
