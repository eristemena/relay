CREATE TABLE IF NOT EXISTS approval_requests (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  tool_call_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  input_preview_json TEXT NOT NULL,
  message TEXT NOT NULL,
  state TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  reviewed_at TEXT,
  applied_at TEXT,
  FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE,
  UNIQUE (run_id, tool_call_id)
);

CREATE INDEX IF NOT EXISTS idx_approval_requests_session_state_occurred
ON approval_requests(session_id, state, occurred_at ASC);

CREATE INDEX IF NOT EXISTS idx_approval_requests_run_state
ON approval_requests(run_id, state);