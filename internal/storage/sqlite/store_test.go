package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_CreateListAndOpenSession(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	first, err := store.CreateSession(ctx, "First session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	second, err := store.CreateSession(ctx, "Second session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	seen := map[string]bool{}
	for _, session := range sessions {
		seen[session.ID] = true
	}
	if !seen[first.ID] || !seen[second.ID] {
		t.Fatalf("sessions = %+v, want both created sessions present", sessions)
	}

	opened, err := store.OpenSession(ctx, first.ID)
	if err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if opened.ID != first.ID {
		t.Fatalf("opened.ID = %q, want %q", opened.ID, first.ID)
	}
	if opened.Status != StatusActive {
		t.Fatalf("opened.Status = %q, want %q", opened.Status, StatusActive)
	}
}

func TestStore_ListRunSummariesAndReplayEvents(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Run history")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	longTask := strings.Repeat("review ", 20)
	firstRun, err := store.CreateAgentRun(ctx, session.ID, longTask, RoleReviewer, "model-a")
	if err != nil {
		t.Fatalf("CreateAgentRun() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, firstRun.ID, EventTypeToolCall, firstRun.Role, firstRun.Model, `{"tool_name":"read_file"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() tool call error = %v", err)
	}
	secondEvent, err := store.AppendRunEvent(ctx, firstRun.ID, EventTypeToolResult, firstRun.Role, firstRun.Model, `{"status":"completed"}`, nil, nil)
	if err != nil {
		t.Fatalf("AppendRunEvent() tool result error = %v", err)
	}
	now := time.Now().UTC()
	firstRun.State = RunStateCompleted
	firstRun.CompletedAt = &now
	if err := store.UpdateAgentRun(ctx, firstRun); err != nil {
		t.Fatalf("UpdateAgentRun() first error = %v", err)
	}

	secondRun, err := store.CreateAgentRun(ctx, session.ID, "Explain the replay path", RoleExplainer, "model-b")
	if err != nil {
		t.Fatalf("CreateAgentRun() second error = %v", err)
	}
	secondRun.State = RunStateErrored
	secondRun.ErrorCode = "run_failed"
	secondRun.ErrorMessage = "provider failure"
	secondRun.CompletedAt = &now
	if err := store.UpdateAgentRun(ctx, secondRun); err != nil {
		t.Fatalf("UpdateAgentRun() second error = %v", err)
	}

	summaries, err := store.ListRunSummaries(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListRunSummaries() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}
	if summaries[0].ID != secondRun.ID {
		t.Fatalf("summaries[0].ID = %q, want latest run %q", summaries[0].ID, secondRun.ID)
	}
	if summaries[0].HasToolActivity {
		t.Fatal("summaries[0].HasToolActivity = true, want false")
	}
	if summaries[0].ErrorCode != "run_failed" {
		t.Fatalf("summaries[0].ErrorCode = %q, want run_failed", summaries[0].ErrorCode)
	}
	if summaries[1].ID != firstRun.ID {
		t.Fatalf("summaries[1].ID = %q, want first run %q", summaries[1].ID, firstRun.ID)
	}
	if !summaries[1].HasToolActivity {
		t.Fatal("summaries[1].HasToolActivity = false, want true")
	}
	if len(summaries[1].TaskTextPreview) > 96 {
		t.Fatalf("TaskTextPreview length = %d, want <= 96", len(summaries[1].TaskTextPreview))
	}

	events, err := store.ListRunEvents(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Sequence != 1 || events[1].Sequence != secondEvent.Sequence {
		t.Fatalf("event sequences = [%d, %d], want [1, 2]", events[0].Sequence, events[1].Sequence)
	}
	if events[0].EventType != EventTypeToolCall || events[1].EventType != EventTypeToolResult {
		t.Fatalf("event types = [%q, %q], want tool_call/tool_result", events[0].EventType, events[1].EventType)
	}
}

func TestStore_ListAgentExecutionsAndOrchestrationReplayData(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Orchestration replay")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Replay per-agent output", RolePlanner, "planner-model")
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	startedAt := time.Now().UTC()
	planner, err := store.CreateAgentExecution(ctx, AgentExecution{
		ID:         "agent_planner_1",
		RunID:      run.ID,
		Role:       RolePlanner,
		Model:      "planner-model",
		State:      AgentExecutionStateAssigned,
		TaskText:   "Break the goal into stages.",
		SpawnOrder: 1,
		StartedAt:  &startedAt,
	})
	if err != nil {
		t.Fatalf("CreateAgentExecution() planner error = %v", err)
	}
	coder, err := store.CreateAgentExecution(ctx, AgentExecution{
		ID:           "agent_coder_2",
		RunID:        run.ID,
		Role:         RoleCoder,
		Model:        "coder-model",
		State:        AgentExecutionStateErrored,
		TaskText:     "Draft the implementation from the planner output.",
		SpawnOrder:   2,
		StartedAt:    &startedAt,
		ErrorCode:    "agent_generation_failed",
		ErrorMessage: "Coder could not finish the draft.",
	})
	if err != nil {
		t.Fatalf("CreateAgentExecution() coder error = %v", err)
	}

	completedAt := time.Now().UTC()
	planner.State = AgentExecutionStateCompleted
	planner.CompletedAt = &completedAt
	if err := store.UpdateAgentExecution(ctx, planner); err != nil {
		t.Fatalf("UpdateAgentExecution() planner error = %v", err)
	}
	if err := store.UpdateAgentExecution(ctx, coder); err != nil {
		t.Fatalf("UpdateAgentExecution() coder error = %v", err)
	}

	if _, err := store.AppendRunEvent(ctx, run.ID, EventTypeTaskAssigned, RolePlanner, planner.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","task_text":"Break the goal into stages.","occurred_at":"2026-03-24T12:00:00Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() task_assigned error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, EventTypeToken, RolePlanner, planner.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_planner_1","text":"planner transcript","occurred_at":"2026-03-24T12:00:01Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() token error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, run.ID, EventTypeAgentError, RoleCoder, coder.Model, `{"session_id":"`+session.ID+`","run_id":"`+run.ID+`","agent_id":"agent_coder_2","code":"agent_generation_failed","message":"Coder could not finish the draft.","terminal":true,"occurred_at":"2026-03-24T12:00:02Z"}`, nil, nil); err != nil {
		t.Fatalf("AppendRunEvent() agent_error error = %v", err)
	}

	executions, err := store.ListAgentExecutions(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 2 {
		t.Fatalf("len(executions) = %d, want 2", len(executions))
	}
	if executions[0].TaskText != "Break the goal into stages." {
		t.Fatalf("executions[0].TaskText = %q, want persisted planner assignment", executions[0].TaskText)
	}
	if executions[1].ErrorMessage != "Coder could not finish the draft." {
		t.Fatalf("executions[1].ErrorMessage = %q, want persisted coder failure", executions[1].ErrorMessage)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if !strings.Contains(events[0].PayloadJSON, `"agent_id":"agent_planner_1"`) {
		t.Fatalf("events[0].PayloadJSON = %q, want planner agent id", events[0].PayloadJSON)
	}
	if !strings.Contains(events[1].PayloadJSON, `"text":"planner transcript"`) {
		t.Fatalf("events[1].PayloadJSON = %q, want planner transcript chunk", events[1].PayloadJSON)
	}
	if !strings.Contains(events[2].PayloadJSON, `"agent_id":"agent_coder_2"`) {
		t.Fatalf("events[2].PayloadJSON = %q, want coder agent id", events[2].PayloadJSON)
	}
}

func TestStore_GetSessionAndRunLookups(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Lookup session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	loadedSession, err := store.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if loadedSession.ID != session.ID {
		t.Fatalf("loadedSession.ID = %q, want %q", loadedSession.ID, session.ID)
	}

	run, err := store.CreateAgentRun(ctx, session.ID, "Inspect active lookup", RolePlanner, "planner-model")
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	if !run.Active() {
		t.Fatal("run.Active() = false, want true for accepted run")
	}

	activeRun, err := store.GetActiveRun(ctx)
	if err != nil {
		t.Fatalf("GetActiveRun() error = %v", err)
	}
	if activeRun.ID != run.ID {
		t.Fatalf("activeRun.ID = %q, want %q", activeRun.ID, run.ID)
	}

	loadedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() error = %v", err)
	}
	if loadedRun.ID != run.ID {
		t.Fatalf("loadedRun.ID = %q, want %q", loadedRun.ID, run.ID)
	}

	run.State = RunStateCompleted
	completedAt := time.Now().UTC()
	firstTokenAt := completedAt.Add(-time.Second)
	run.CompletedAt = &completedAt
	run.FirstTokenAt = &firstTokenAt
	if err := store.UpdateAgentRun(ctx, run); err != nil {
		t.Fatalf("UpdateAgentRun() error = %v", err)
	}
	if run.Active() {
		t.Fatal("run.Active() = true, want false for completed run")
	}

	completedRun, err := store.GetAgentRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetAgentRun() completed error = %v", err)
	}
	if completedRun.CompletedAt == nil || completedRun.CompletedAt.Format(time.RFC3339) != completedAt.Format(time.RFC3339) {
		t.Fatalf("completedRun.CompletedAt = %#v, want %v", completedRun.CompletedAt, completedAt)
	}
	if completedRun.FirstTokenAt == nil || completedRun.FirstTokenAt.Format(time.RFC3339) != firstTokenAt.Format(time.RFC3339) {
		t.Fatalf("completedRun.FirstTokenAt = %#v, want %v", completedRun.FirstTokenAt, firstTokenAt)
	}

	if _, err := store.GetActiveRun(ctx); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("GetActiveRun() error = %v, want ErrRunNotFound", err)
	}
}

func TestStore_ReturnsNotFoundErrorsForMissingRows(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	if _, err := store.GetSession(ctx, "session_missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("GetSession() error = %v, want ErrSessionNotFound", err)
	}
	if _, err := store.OpenSession(ctx, "session_missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("OpenSession() error = %v, want ErrSessionNotFound", err)
	}
	if _, err := store.GetAgentRun(ctx, "run_missing"); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("GetAgentRun() error = %v, want ErrRunNotFound", err)
	}
	if _, err := store.GetApprovalRequest(ctx, "run_missing", "tool_missing"); !errors.Is(err, ErrApprovalRequestNotFound) {
		t.Fatalf("GetApprovalRequest() error = %v, want ErrApprovalRequestNotFound", err)
	}
	if err := store.UpdateAgentRun(ctx, AgentRun{ID: "run_missing", State: RunStateCompleted}); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("UpdateAgentRun() error = %v, want ErrRunNotFound", err)
	}
	if err := store.UpdateApprovalRequestState(ctx, "run_missing", "tool_missing", ApprovalStateRejected, nil, nil); !errors.Is(err, ErrApprovalRequestNotFound) {
		t.Fatalf("UpdateApprovalRequestState() error = %v, want ErrApprovalRequestNotFound", err)
	}
	if err := store.UpdateAgentExecution(ctx, AgentExecution{ID: "agent_missing", State: AgentExecutionStateCompleted}); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("UpdateAgentExecution() error = %v, want ErrRunNotFound", err)
	}
}

func TestStore_CreateAndResolveApprovalRequests(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Approval persistence")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Persist approval state", RoleCoder, "coder-model")
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}
	occurredAt := time.Now().UTC().Add(-time.Minute)
	approval, err := store.CreateApprovalRequest(ctx, ApprovalRequest{
		SessionID:        session.ID,
		RunID:            run.ID,
		ToolCallID:       "call_1",
		ToolName:         "write_file",
		Role:             RoleCoder,
		Model:            "coder-model",
		InputPreviewJSON: `{"path":"README.md"}`,
		Message:          "Approve the write.",
		State:            ApprovalStateProposed,
		OccurredAt:       occurredAt,
	})
	if err != nil {
		t.Fatalf("CreateApprovalRequest() error = %v", err)
	}
	loaded, err := store.GetApprovalRequest(ctx, run.ID, "call_1")
	if err != nil {
		t.Fatalf("GetApprovalRequest() error = %v", err)
	}
	if loaded.ID != approval.ID || loaded.State != ApprovalStateProposed {
		t.Fatalf("loaded approval = %#v, want persisted proposed approval", loaded)
	}
	pending, err := store.ListPendingApprovalRequests(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListPendingApprovalRequests() error = %v", err)
	}
	if len(pending) != 1 || pending[0].ToolCallID != "call_1" {
		t.Fatalf("pending approvals = %#v, want one approval", pending)
	}
	reviewedAt := time.Now().UTC()
	if err := store.UpdateApprovalRequestState(ctx, run.ID, "call_1", ApprovalStateApproved, &reviewedAt, nil); err != nil {
		t.Fatalf("UpdateApprovalRequestState(approved) error = %v", err)
	}
	appliedAt := reviewedAt.Add(time.Second)
	if err := store.UpdateApprovalRequestState(ctx, run.ID, "call_1", ApprovalStateApplied, &reviewedAt, &appliedAt); err != nil {
		t.Fatalf("UpdateApprovalRequestState(applied) error = %v", err)
	}
	resolved, err := store.GetApprovalRequest(ctx, run.ID, "call_1")
	if err != nil {
		t.Fatalf("GetApprovalRequest() resolved error = %v", err)
	}
	if resolved.State != ApprovalStateApplied {
		t.Fatalf("resolved.State = %q, want %q", resolved.State, ApprovalStateApplied)
	}
	if resolved.AppliedAt == nil || resolved.AppliedAt.Format(time.RFC3339) != appliedAt.Format(time.RFC3339) {
		t.Fatalf("resolved.AppliedAt = %#v, want %v", resolved.AppliedAt, appliedAt)
	}
	if err := store.ResolvePendingApprovalRequestsForRun(ctx, run.ID, ApprovalStateExpired, &reviewedAt); err != nil {
		t.Fatalf("ResolvePendingApprovalRequestsForRun() error = %v", err)
	}
	remaining, err := store.ListPendingApprovalRequestsForRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListPendingApprovalRequestsForRun() error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining approvals = %#v, want none", remaining)
	}
}

func TestStore_CreateAgentExecutionAppliesDefaults(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Execution defaults")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	run, err := store.CreateAgentRun(ctx, session.ID, "Assign defaults", RolePlanner, "planner-model")
	if err != nil {
		t.Fatalf("CreateAgentRun() error = %v", err)
	}

	execution, err := store.CreateAgentExecution(ctx, AgentExecution{
		RunID:      run.ID,
		Role:       RoleTester,
		Model:      "tester-model",
		TaskText:   "Validate the plan.",
		SpawnOrder: 1,
	})
	if err != nil {
		t.Fatalf("CreateAgentExecution() error = %v", err)
	}
	if !strings.HasPrefix(execution.ID, "agent_") {
		t.Fatalf("execution.ID = %q, want generated agent_ prefix", execution.ID)
	}
	if execution.State != AgentExecutionStateQueued {
		t.Fatalf("execution.State = %q, want %q", execution.State, AgentExecutionStateQueued)
	}

	executions, err := store.ListAgentExecutions(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListAgentExecutions() error = %v", err)
	}
	if len(executions) != 1 {
		t.Fatalf("len(executions) = %d, want 1", len(executions))
	}
	if executions[0].ID != execution.ID {
		t.Fatalf("executions[0].ID = %q, want %q", executions[0].ID, execution.ID)
	}
}
