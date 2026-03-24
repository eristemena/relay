-- name: ListOrchestrationEventsByRun :many
SELECT run_id, sequence, event_type, role, model, payload_json, created_at
FROM agent_run_events
WHERE run_id = ?
ORDER BY sequence ASC;