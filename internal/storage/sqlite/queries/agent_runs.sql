-- name: GetActiveRun :one
SELECT id, session_id, task_text, role, model, state, started_at, completed_at, error_code, error_message, first_token_at
FROM agent_runs
WHERE state IN ('accepted', 'thinking', 'tool_running')
ORDER BY datetime(started_at) DESC
LIMIT 1;

-- name: GetAgentRun :one
SELECT id, session_id, task_text, role, model, state, started_at, completed_at, error_code, error_message, first_token_at
FROM agent_runs
WHERE id = ?;

-- name: ListRunSummaries :many
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
       EXISTS (
         SELECT 1 FROM agent_run_events e
         WHERE e.run_id = r.id AND e.event_type IN ('tool_call', 'tool_result')
       ) AS has_tool_activity
FROM agent_runs r
WHERE r.session_id = ?
ORDER BY datetime(r.started_at) DESC;