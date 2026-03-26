package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var ErrSessionNotFound = errors.New("session not found")
var ErrRunNotFound = errors.New("agent run not found")
var ErrActiveRunExists = errors.New("agent run already active")
var ErrApprovalRequestNotFound = errors.New("approval request not found")

type Store struct {
	db      *sql.DB
	writeMu sync.Mutex
}

func NewStore(databasePath string) (*Store, error) {
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite busy timeout: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite journal mode: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		ORDER BY datetime(last_opened_at) DESC, datetime(created_at) DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		WHERE id = ?
	`, sessionID)

	session, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}

	return session, nil
}

func (s *Store) CreateSession(ctx context.Context, displayName string) (Session, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if strings.TrimSpace(displayName) == "" {
		displayName = fmt.Sprintf("Session %s", time.Now().Format("2006-01-02 15:04"))
	}

	now := time.Now().UTC()
	sessionID, err := generateSessionID()
	if err != nil {
		return Session{}, err
	}

	snapshot := EmptySnapshot()
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return Session{}, fmt.Errorf("marshal snapshot: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET status = ?, updated_at = ? WHERE status = ?`, StatusIdle, now.Format(time.RFC3339), StatusActive); err != nil {
		return Session{}, fmt.Errorf("clear active sessions: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sessionID, strings.TrimSpace(displayName), now.Format(time.RFC3339), now.Format(time.RFC3339), now.Format(time.RFC3339), StatusActive, string(snapshotJSON)); err != nil {
		return Session{}, fmt.Errorf("insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Session{}, fmt.Errorf("commit session creation: %w", err)
	}

	return Session{
		ID:           sessionID,
		DisplayName:  strings.TrimSpace(displayName),
		CreatedAt:    now,
		UpdatedAt:    now,
		LastOpenedAt: now,
		Status:       StatusActive,
		Snapshot:     snapshot,
	}, nil
}

func (s *Store) OpenSession(ctx context.Context, sessionID string) (Session, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET status = ?, updated_at = ? WHERE status = ?`, StatusIdle, now.Format(time.RFC3339), StatusActive); err != nil {
		return Session{}, fmt.Errorf("clear active sessions: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, updated_at = ?, last_opened_at = ?
		WHERE id = ?
	`, StatusActive, now.Format(time.RFC3339), now.Format(time.RFC3339), sessionID)
	if err != nil {
		return Session{}, fmt.Errorf("open session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Session{}, fmt.Errorf("read updated rows: %w", err)
	}
	if rowsAffected == 0 {
		return Session{}, ErrSessionNotFound
	}

	session, err := scanSession(tx.QueryRowContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		WHERE id = ?
	`, sessionID))
	if err != nil {
		return Session{}, err
	}

	if err := tx.Commit(); err != nil {
		return Session{}, fmt.Errorf("commit session open: %w", err)
	}

	return session, nil
}

func (s *Store) CreateAgentRun(ctx context.Context, sessionID string, taskText string, role AgentRole, model string) (AgentRun, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	now := time.Now().UTC()
	runID, err := generateID("run")
	if err != nil {
		return AgentRun{}, err
	}

	taskText = strings.TrimSpace(taskText)
	if taskText == "" {
		return AgentRun{}, fmt.Errorf("create agent run: task text is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AgentRun{}, fmt.Errorf("begin run transaction: %w", err)
	}
	defer tx.Rollback()

	var activeCount int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM agent_runs
		WHERE state IN (?, ?, ?, ?)
	`, RunStateAccepted, RunStateActive, RunStateThinking, RunStateToolRunning).Scan(&activeCount); err != nil {
		return AgentRun{}, fmt.Errorf("check active run: %w", err)
	}
	if activeCount > 0 {
		return AgentRun{}, ErrActiveRunExists
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO agent_runs (id, session_id, task_text, role, model, state, started_at, completed_at, error_code, error_message, first_token_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NULL, '', '', NULL)
	`, runID, sessionID, taskText, string(role), strings.TrimSpace(model), RunStateAccepted, now.Format(time.RFC3339)); err != nil {
		return AgentRun{}, fmt.Errorf("insert agent run: %w", err)
	}

	if err := s.updateSessionActivity(ctx, tx, sessionID, true); err != nil {
		return AgentRun{}, err
	}

	if err := tx.Commit(); err != nil {
		return AgentRun{}, fmt.Errorf("commit agent run: %w", err)
	}

	return AgentRun{
		ID:        runID,
		SessionID: sessionID,
		TaskText:  taskText,
		Role:      role,
		Model:     strings.TrimSpace(model),
		State:     RunStateAccepted,
		StartedAt: now,
		Mode:      RunModeOrchestration,
	}, nil
}

func (s *Store) GetActiveRun(ctx context.Context) (AgentRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, task_text, role, model, state, started_at, completed_at, error_code, error_message, first_token_at
		FROM agent_runs
		WHERE state IN (?, ?, ?, ?)
		ORDER BY datetime(started_at) DESC
		LIMIT 1
	`, RunStateAccepted, RunStateActive, RunStateThinking, RunStateToolRunning)

	run, err := scanAgentRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentRun{}, ErrRunNotFound
	}
	if err != nil {
		return AgentRun{}, err
	}

	return run, nil
}

func (s *Store) GetAgentRun(ctx context.Context, runID string) (AgentRun, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, task_text, role, model, state, started_at, completed_at, error_code, error_message, first_token_at
		FROM agent_runs
		WHERE id = ?
	`, runID)

	run, err := scanAgentRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return AgentRun{}, ErrRunNotFound
	}
	if err != nil {
		return AgentRun{}, err
	}

	return run, nil
}

func (s *Store) ListRunSummaries(ctx context.Context, sessionID string) ([]RunSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id,
		       CASE
		         WHEN length(r.task_text) > 96 THEN substr(r.task_text, 1, 93) || '...'
		         ELSE r.task_text
		       END AS task_text_preview,
		       r.role,
		       r.model,
		       r.state,
		       r.started_at,
		       r.completed_at,
		       r.error_code,
		       EXISTS (
		         SELECT 1 FROM agent_run_events e
		         WHERE e.run_id = r.id AND e.event_type IN (?, ?)
		       ) AS has_tool_activity
		FROM agent_runs r
		WHERE r.session_id = ?
			ORDER BY datetime(r.started_at) DESC, r.rowid DESC
	`, EventTypeToolCall, EventTypeToolResult, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list run summaries: %w", err)
	}
	defer rows.Close()

	summaries := make([]RunSummary, 0)
	for rows.Next() {
		var (
			summary     RunSummary
			role        string
			errorCode   sql.NullString
			startedAt   string
			completedAt sql.NullString
		)
		if err := rows.Scan(&summary.ID, &summary.TaskTextPreview, &role, &summary.Model, &summary.State, &startedAt, &completedAt, &errorCode, &summary.HasToolActivity); err != nil {
			return nil, fmt.Errorf("scan run summary: %w", err)
		}
		parsedStartedAt, err := time.Parse(time.RFC3339, startedAt)
		if err != nil {
			return nil, fmt.Errorf("parse run summary started_at: %w", err)
		}
		summary.StartedAt = parsedStartedAt
		summary.Role = AgentRole(role)
		if completedAt.Valid {
			parsedCompletedAt, err := time.Parse(time.RFC3339, completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("parse run summary completed_at: %w", err)
			}
			summary.CompletedAt = &parsedCompletedAt
		}
		if errorCode.Valid {
			summary.ErrorCode = errorCode.String
		}
		if summary.TaskTextPreview == "" {
			summary.TaskTextPreview = "Untitled task"
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run summaries: %w", err)
	}

	return summaries, nil
}

func (s *Store) AppendRunEvent(ctx context.Context, runID string, eventType string, role AgentRole, model string, payloadJSON string) (AgentRunEvent, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AgentRunEvent{}, fmt.Errorf("begin append run event: %w", err)
	}
	defer tx.Rollback()

	var nextSequence int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(sequence), 0) + 1
		FROM agent_run_events
		WHERE run_id = ?
	`, runID).Scan(&nextSequence); err != nil {
		return AgentRunEvent{}, fmt.Errorf("compute next run event sequence: %w", err)
	}

	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO agent_run_events (run_id, sequence, event_type, role, model, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, runID, nextSequence, eventType, string(role), strings.TrimSpace(model), payloadJSON, now.Format(time.RFC3339)); err != nil {
		return AgentRunEvent{}, fmt.Errorf("insert run event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return AgentRunEvent{}, fmt.Errorf("commit run event: %w", err)
	}

	return AgentRunEvent{
		RunID:       runID,
		Sequence:    nextSequence,
		EventType:   eventType,
		Role:        role,
		Model:       strings.TrimSpace(model),
		PayloadJSON: payloadJSON,
		CreatedAt:   now,
	}, nil
}

func (s *Store) ListRunEvents(ctx context.Context, runID string) ([]AgentRunEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT run_id, sequence, event_type, role, model, payload_json, created_at
		FROM agent_run_events
		WHERE run_id = ?
		ORDER BY sequence ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list run events: %w", err)
	}
	defer rows.Close()

	events := make([]AgentRunEvent, 0)
	for rows.Next() {
		var (
			event     AgentRunEvent
			role      string
			createdAt string
		)
		if err := rows.Scan(&event.RunID, &event.Sequence, &event.EventType, &role, &event.Model, &event.PayloadJSON, &createdAt); err != nil {
			return nil, fmt.Errorf("scan run event: %w", err)
		}
		parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse run event created_at: %w", err)
		}
		event.Role = AgentRole(role)
		event.CreatedAt = parsedCreatedAt
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run events: %w", err)
	}

	return events, nil
}

func (s *Store) CreateApprovalRequest(ctx context.Context, approval ApprovalRequest) (ApprovalRequest, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if strings.TrimSpace(approval.ID) == "" {
		id, err := generateID("approval")
		if err != nil {
			return ApprovalRequest{}, err
		}
		approval.ID = id
	}
	if strings.TrimSpace(approval.SessionID) == "" {
		return ApprovalRequest{}, fmt.Errorf("create approval request: session id is required")
	}
	if strings.TrimSpace(approval.RunID) == "" {
		return ApprovalRequest{}, fmt.Errorf("create approval request: run id is required")
	}
	if strings.TrimSpace(approval.ToolCallID) == "" {
		return ApprovalRequest{}, fmt.Errorf("create approval request: tool call id is required")
	}
	if strings.TrimSpace(approval.ToolName) == "" {
		return ApprovalRequest{}, fmt.Errorf("create approval request: tool name is required")
	}
	if strings.TrimSpace(approval.InputPreviewJSON) == "" {
		approval.InputPreviewJSON = "{}"
	}
	if strings.TrimSpace(approval.Message) == "" {
		return ApprovalRequest{}, fmt.Errorf("create approval request: message is required")
	}
	if strings.TrimSpace(approval.State) == "" {
		approval.State = ApprovalStateProposed
	}
	if approval.OccurredAt.IsZero() {
		approval.OccurredAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO approval_requests (id, session_id, run_id, tool_call_id, tool_name, role, model, input_preview_json, message, state, occurred_at, reviewed_at, applied_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, approval.ID, approval.SessionID, approval.RunID, approval.ToolCallID, approval.ToolName, string(approval.Role), strings.TrimSpace(approval.Model), approval.InputPreviewJSON, approval.Message, approval.State, approval.OccurredAt.Format(time.RFC3339), nullableRFC3339(approval.ReviewedAt), nullableRFC3339(approval.AppliedAt))
	if err != nil {
		return ApprovalRequest{}, fmt.Errorf("insert approval request: %w", err)
	}

	return approval, nil
}

func (s *Store) GetApprovalRequest(ctx context.Context, runID string, toolCallID string) (ApprovalRequest, error) {
	approval, err := scanApprovalRequest(s.db.QueryRowContext(ctx, `
		SELECT id, session_id, run_id, tool_call_id, tool_name, role, model, input_preview_json, message, state, occurred_at, reviewed_at, applied_at
		FROM approval_requests
		WHERE run_id = ? AND tool_call_id = ?
	`, runID, toolCallID))
	if errors.Is(err, sql.ErrNoRows) {
		return ApprovalRequest{}, ErrApprovalRequestNotFound
	}
	if err != nil {
		return ApprovalRequest{}, fmt.Errorf("get approval request: %w", err)
	}
	return approval, nil
}

func (s *Store) ListPendingApprovalRequests(ctx context.Context, sessionID string) ([]ApprovalRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, run_id, tool_call_id, tool_name, role, model, input_preview_json, message, state, occurred_at, reviewed_at, applied_at
		FROM approval_requests
		WHERE session_id = ? AND state = ?
		ORDER BY datetime(occurred_at) ASC
	`, sessionID, ApprovalStateProposed)
	if err != nil {
		return nil, fmt.Errorf("list pending approval requests: %w", err)
	}
	defer rows.Close()

	approvals := make([]ApprovalRequest, 0)
	for rows.Next() {
		approval, err := scanApprovalRequest(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate approval requests: %w", err)
	}
	return approvals, nil
}

func (s *Store) ListPendingApprovalRequestsForRun(ctx context.Context, runID string) ([]ApprovalRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, run_id, tool_call_id, tool_name, role, model, input_preview_json, message, state, occurred_at, reviewed_at, applied_at
		FROM approval_requests
		WHERE run_id = ? AND state = ?
		ORDER BY datetime(occurred_at) ASC
	`, runID, ApprovalStateProposed)
	if err != nil {
		return nil, fmt.Errorf("list pending approval requests for run: %w", err)
	}
	defer rows.Close()

	approvals := make([]ApprovalRequest, 0)
	for rows.Next() {
		approval, err := scanApprovalRequest(rows)
		if err != nil {
			return nil, err
		}
		approvals = append(approvals, approval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate run approval requests: %w", err)
	}
	return approvals, nil
}

func (s *Store) UpdateApprovalRequestState(ctx context.Context, runID string, toolCallID string, state string, reviewedAt *time.Time, appliedAt *time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	result, err := s.db.ExecContext(ctx, `
		UPDATE approval_requests
		SET state = ?, reviewed_at = ?, applied_at = ?
		WHERE run_id = ? AND tool_call_id = ?
	`, strings.TrimSpace(state), nullableRFC3339(reviewedAt), nullableRFC3339(appliedAt), runID, toolCallID)
	if err != nil {
		return fmt.Errorf("update approval request state: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated approval request rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrApprovalRequestNotFound
	}
	return nil
}

func (s *Store) ResolvePendingApprovalRequestsForRun(ctx context.Context, runID string, state string, reviewedAt *time.Time) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		UPDATE approval_requests
		SET state = ?, reviewed_at = COALESCE(reviewed_at, ?)
		WHERE run_id = ? AND state = ?
	`, strings.TrimSpace(state), nullableRFC3339(reviewedAt), runID, ApprovalStateProposed)
	if err != nil {
		return fmt.Errorf("resolve pending approval requests for run: %w", err)
	}
	return nil
}

func (s *Store) UpdateAgentRun(ctx context.Context, run AgentRun) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	completedAt := sql.NullString{}
	if run.CompletedAt != nil {
		completedAt.Valid = true
		completedAt.String = run.CompletedAt.Format(time.RFC3339)
	}
	firstTokenAt := sql.NullString{}
	if run.FirstTokenAt != nil {
		firstTokenAt.Valid = true
		firstTokenAt.String = run.FirstTokenAt.Format(time.RFC3339)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_runs
		SET state = ?, completed_at = ?, error_code = ?, error_message = ?, first_token_at = ?
		WHERE id = ?
	`, run.State, completedAt, strings.TrimSpace(run.ErrorCode), strings.TrimSpace(run.ErrorMessage), firstTokenAt, run.ID)
	if err != nil {
		return fmt.Errorf("update agent run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated run rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (s *Store) CreateAgentExecution(ctx context.Context, execution AgentExecution) (AgentExecution, error) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if strings.TrimSpace(execution.ID) == "" {
		id, err := generateID("agent")
		if err != nil {
			return AgentExecution{}, err
		}
		execution.ID = id
	}

	if strings.TrimSpace(execution.RunID) == "" {
		return AgentExecution{}, fmt.Errorf("create agent execution: run id is required")
	}
	if strings.TrimSpace(execution.TaskText) == "" {
		return AgentExecution{}, fmt.Errorf("create agent execution: task text is required")
	}
	if strings.TrimSpace(execution.State) == "" {
		execution.State = AgentExecutionStateQueued
	}

	var startedAt any
	if execution.StartedAt != nil {
		startedAt = execution.StartedAt.Format(time.RFC3339)
	}
	var completedAt any
	if execution.CompletedAt != nil {
		completedAt = execution.CompletedAt.Format(time.RFC3339)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_executions (id, run_id, role, model, state, task_text, spawn_order, started_at, completed_at, error_code, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, execution.ID, execution.RunID, string(execution.Role), strings.TrimSpace(execution.Model), execution.State, execution.TaskText, execution.SpawnOrder, startedAt, completedAt, strings.TrimSpace(execution.ErrorCode), strings.TrimSpace(execution.ErrorMessage)); err != nil {
		return AgentExecution{}, fmt.Errorf("insert agent execution: %w", err)
	}

	return execution, nil
}

func (s *Store) UpdateAgentExecution(ctx context.Context, execution AgentExecution) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	var startedAt any
	if execution.StartedAt != nil {
		startedAt = execution.StartedAt.Format(time.RFC3339)
	}
	var completedAt any
	if execution.CompletedAt != nil {
		completedAt = execution.CompletedAt.Format(time.RFC3339)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE agent_executions
		SET state = ?, started_at = ?, completed_at = ?, error_code = ?, error_message = ?
		WHERE id = ?
	`, execution.State, startedAt, completedAt, strings.TrimSpace(execution.ErrorCode), strings.TrimSpace(execution.ErrorMessage), execution.ID)
	if err != nil {
		return fmt.Errorf("update agent execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated agent execution rows: %w", err)
	}
	if rowsAffected == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (s *Store) ListAgentExecutions(ctx context.Context, runID string) ([]AgentExecution, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, run_id, role, model, state, task_text, spawn_order, started_at, completed_at, error_code, error_message
		FROM agent_executions
		WHERE run_id = ?
		ORDER BY spawn_order ASC
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list agent executions: %w", err)
	}
	defer rows.Close()

	executions := make([]AgentExecution, 0)
	for rows.Next() {
		execution, err := scanAgentExecution(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent executions: %w", err)
	}

	return executions, nil
}

func (s *Store) migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		body, err := migrationFiles.ReadFile(path.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if _, err := s.db.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(scanner rowScanner) (Session, error) {
	var (
		session      Session
		createdAt    string
		updatedAt    string
		lastOpenedAt string
		snapshotJSON string
	)

	if err := scanner.Scan(&session.ID, &session.DisplayName, &createdAt, &updatedAt, &lastOpenedAt, &session.Status, &snapshotJSON); err != nil {
		return Session{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse created_at: %w", err)
	}
	parsedUpdatedAt, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse updated_at: %w", err)
	}
	parsedLastOpenedAt, err := time.Parse(time.RFC3339, lastOpenedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse last_opened_at: %w", err)
	}

	session.CreatedAt = parsedCreatedAt
	session.UpdatedAt = parsedUpdatedAt
	session.LastOpenedAt = parsedLastOpenedAt
	session.Snapshot = EmptySnapshot()
	if strings.TrimSpace(snapshotJSON) != "" {
		if err := json.Unmarshal([]byte(snapshotJSON), &session.Snapshot); err != nil {
			return Session{}, fmt.Errorf("decode snapshot: %w", err)
		}
	}

	return session, nil
}

func scanAgentExecution(scanner rowScanner) (AgentExecution, error) {
	var (
		execution   AgentExecution
		role        string
		startedAt   sql.NullString
		completedAt sql.NullString
	)

	if err := scanner.Scan(&execution.ID, &execution.RunID, &role, &execution.Model, &execution.State, &execution.TaskText, &execution.SpawnOrder, &startedAt, &completedAt, &execution.ErrorCode, &execution.ErrorMessage); err != nil {
		return AgentExecution{}, err
	}

	execution.Role = AgentRole(role)
	if startedAt.Valid {
		parsedStartedAt, err := time.Parse(time.RFC3339, startedAt.String)
		if err != nil {
			return AgentExecution{}, fmt.Errorf("parse agent execution started_at: %w", err)
		}
		execution.StartedAt = &parsedStartedAt
	}
	if completedAt.Valid {
		parsedCompletedAt, err := time.Parse(time.RFC3339, completedAt.String)
		if err != nil {
			return AgentExecution{}, fmt.Errorf("parse agent execution completed_at: %w", err)
		}
		execution.CompletedAt = &parsedCompletedAt
	}

	return execution, nil
}

func scanApprovalRequest(scanner rowScanner) (ApprovalRequest, error) {
	var (
		approval      ApprovalRequest
		role          string
		occurredAt    string
		reviewedAt    sql.NullString
		appliedAt     sql.NullString
	)

	if err := scanner.Scan(&approval.ID, &approval.SessionID, &approval.RunID, &approval.ToolCallID, &approval.ToolName, &role, &approval.Model, &approval.InputPreviewJSON, &approval.Message, &approval.State, &occurredAt, &reviewedAt, &appliedAt); err != nil {
		return ApprovalRequest{}, err
	}

	parsedOccurredAt, err := time.Parse(time.RFC3339, occurredAt)
	if err != nil {
		return ApprovalRequest{}, fmt.Errorf("parse approval occurred_at: %w", err)
	}
	approval.OccurredAt = parsedOccurredAt
	approval.Role = AgentRole(role)
	if reviewedAt.Valid {
		parsedReviewedAt, err := time.Parse(time.RFC3339, reviewedAt.String)
		if err != nil {
			return ApprovalRequest{}, fmt.Errorf("parse approval reviewed_at: %w", err)
		}
		approval.ReviewedAt = &parsedReviewedAt
	}
	if appliedAt.Valid {
		parsedAppliedAt, err := time.Parse(time.RFC3339, appliedAt.String)
		if err != nil {
			return ApprovalRequest{}, fmt.Errorf("parse approval applied_at: %w", err)
		}
		approval.AppliedAt = &parsedAppliedAt
	}

	return approval, nil
}

func scanAgentRun(scanner rowScanner) (AgentRun, error) {
	var (
		run          AgentRun
		role         string
		startedAt    string
		completedAt  sql.NullString
		firstTokenAt sql.NullString
	)

	if err := scanner.Scan(&run.ID, &run.SessionID, &run.TaskText, &role, &run.Model, &run.State, &startedAt, &completedAt, &run.ErrorCode, &run.ErrorMessage, &firstTokenAt); err != nil {
		return AgentRun{}, err
	}

	parsedStartedAt, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return AgentRun{}, fmt.Errorf("parse agent run started_at: %w", err)
	}
	run.StartedAt = parsedStartedAt
	run.Role = AgentRole(role)
	if completedAt.Valid {
		parsedCompletedAt, err := time.Parse(time.RFC3339, completedAt.String)
		if err != nil {
			return AgentRun{}, fmt.Errorf("parse agent run completed_at: %w", err)
		}
		run.CompletedAt = &parsedCompletedAt
	}
	if firstTokenAt.Valid {
		parsedFirstTokenAt, err := time.Parse(time.RFC3339, firstTokenAt.String)
		if err != nil {
			return AgentRun{}, fmt.Errorf("parse agent run first_token_at: %w", err)
		}
		run.FirstTokenAt = &parsedFirstTokenAt
	}

	return run, nil
}

func generateSessionID() (string, error) {
	return generateID("session")
}

func generateID(prefix string) (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate %s id: %w", prefix, err)
	}

	return prefix + "_" + hex.EncodeToString(buffer), nil
}

func nullableRFC3339(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *Store) updateSessionActivity(ctx context.Context, tx *sql.Tx, sessionID string, hasActivity bool) error {
	session, err := scanSession(tx.QueryRowContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		WHERE id = ?
	`, sessionID))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrSessionNotFound
	}
	if err != nil {
		return err
	}

	session.Snapshot.HasActivity = hasActivity
	snapshotJSON, err := json.Marshal(session.Snapshot)
	if err != nil {
		return fmt.Errorf("marshal updated session snapshot: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE sessions
		SET snapshot_json = ?, updated_at = ?
		WHERE id = ?
	`, string(snapshotJSON), time.Now().UTC().Format(time.RFC3339), sessionID); err != nil {
		return fmt.Errorf("update session snapshot activity: %w", err)
	}

	return nil
}
