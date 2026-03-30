CREATE TABLE IF NOT EXISTS touched_files (
  run_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  file_path TEXT NOT NULL,
  touch_type TEXT NOT NULL,
  recorded_at TEXT NOT NULL,
  PRIMARY KEY (run_id, agent_id, file_path, touch_type),
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_touched_files_run_id ON touched_files(run_id);
CREATE INDEX IF NOT EXISTS idx_touched_files_run_agent_id ON touched_files(run_id, agent_id);