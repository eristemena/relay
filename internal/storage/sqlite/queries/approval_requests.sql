-- name: CreateApprovalRequest :exec
INSERT INTO approval_requests (
  id,
  session_id,
  run_id,
  tool_call_id,
  tool_name,
  role,
  model,
  input_preview_json,
  message,
  state,
  occurred_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetApprovalRequest :one
SELECT
  id,
  session_id,
  run_id,
  tool_call_id,
  tool_name,
  role,
  model,
  input_preview_json,
  message,
  state,
  occurred_at,
  reviewed_at,
  applied_at
FROM approval_requests
WHERE run_id = ? AND tool_call_id = ?;

-- name: ListPendingApprovalRequests :many
SELECT
  id,
  session_id,
  run_id,
  tool_call_id,
  tool_name,
  role,
  model,
  input_preview_json,
  message,
  state,
  occurred_at,
  reviewed_at,
  applied_at
FROM approval_requests
WHERE session_id = ? AND state = 'proposed'
ORDER BY datetime(occurred_at) ASC;

-- name: UpdateApprovalRequestState :exec
UPDATE approval_requests
SET state = ?, reviewed_at = ?, applied_at = ?
WHERE run_id = ? AND tool_call_id = ?;