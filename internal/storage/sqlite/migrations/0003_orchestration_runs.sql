CREATE TABLE IF NOT EXISTS agent_executions (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  role TEXT NOT NULL,
  model TEXT NOT NULL,
  state TEXT NOT NULL,
  task_text TEXT NOT NULL,
  spawn_order INTEGER NOT NULL,
  started_at TEXT,
  completed_at TEXT,
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_executions_run_spawn_order
ON agent_executions(run_id, spawn_order ASC);

CREATE INDEX IF NOT EXISTS idx_agent_executions_run_state
ON agent_executions(run_id, state);