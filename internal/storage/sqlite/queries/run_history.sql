-- name: UpsertRunHistoryDocument :exec
INSERT INTO run_history_documents (
  run_id,
  session_id,
  generated_title,
  goal_text,
  final_status,
  agent_count,
  started_at,
  completed_at,
  first_event_at,
  last_event_at,
  summary_text,
  touched_file_count,
  has_file_changes,
  exported_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(run_id) DO UPDATE SET
  session_id = excluded.session_id,
  generated_title = excluded.generated_title,
  goal_text = excluded.goal_text,
  final_status = excluded.final_status,
  agent_count = excluded.agent_count,
  started_at = excluded.started_at,
  completed_at = excluded.completed_at,
  first_event_at = excluded.first_event_at,
  last_event_at = excluded.last_event_at,
  summary_text = excluded.summary_text,
  touched_file_count = excluded.touched_file_count,
  has_file_changes = excluded.has_file_changes,
  exported_at = excluded.exported_at;

-- name: SearchRunHistoryDocuments :many
SELECT d.run_id,
       d.session_id,
       d.generated_title,
       d.goal_text,
       d.final_status,
       d.agent_count,
       d.started_at,
       d.completed_at,
       d.first_event_at,
       d.last_event_at,
       d.summary_text,
       d.touched_file_count,
       d.has_file_changes,
       d.exported_at
FROM run_history_documents d
LEFT JOIN run_history_search_fts fts ON fts.run_id = d.run_id
WHERE d.session_id = ?;