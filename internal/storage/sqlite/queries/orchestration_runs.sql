-- name: ListOrchestrationRuns :many
SELECT id, session_id, task_text, role, model, state, started_at, completed_at
FROM agent_runs
ORDER BY started_at DESC;