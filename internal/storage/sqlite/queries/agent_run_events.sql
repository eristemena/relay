-- name: ListRunEvents :many
SELECT run_id, sequence, event_type, role, model, payload_json, created_at
FROM agent_run_events
WHERE run_id = ?
ORDER BY sequence ASC;

-- name: AppendRunEvent :exec
INSERT INTO agent_run_events (run_id, sequence, event_type, role, model, payload_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);