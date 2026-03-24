-- name: ListAgentExecutionsByRun :many
SELECT id, run_id, role, model, state, task_text, spawn_order, started_at, completed_at, error_code, error_message
FROM agent_executions
WHERE run_id = ?
ORDER BY spawn_order ASC;