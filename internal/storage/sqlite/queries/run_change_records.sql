-- name: ReplaceRunChangeRecords :exec
DELETE FROM run_change_records WHERE run_id = ?;

-- name: InsertRunChangeRecord :exec
INSERT INTO run_change_records (
  run_id,
  tool_call_id,
  path,
  original_content,
  proposed_content,
  base_content_hash,
  approval_state,
  role,
  model,
  occurred_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListRunChangeRecords :many
SELECT run_id,
       tool_call_id,
       path,
       original_content,
       proposed_content,
       base_content_hash,
       approval_state,
       role,
       model,
       occurred_at
FROM run_change_records
WHERE run_id = ?
ORDER BY datetime(occurred_at) ASC, tool_call_id ASC, path ASC;