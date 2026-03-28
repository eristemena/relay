CREATE TABLE IF NOT EXISTS run_history_documents (
  run_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  generated_title TEXT NOT NULL,
  goal_text TEXT NOT NULL,
  final_status TEXT NOT NULL,
  agent_count INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  completed_at TEXT,
  first_event_at TEXT,
  last_event_at TEXT,
  summary_text TEXT,
  touched_file_count INTEGER NOT NULL DEFAULT 0,
  has_file_changes INTEGER NOT NULL DEFAULT 0,
  exported_at TEXT,
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE,
  FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_history_documents_session_started_at ON run_history_documents(session_id, started_at DESC);

CREATE TABLE IF NOT EXISTS run_history_search_documents (
  run_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  title_text TEXT NOT NULL,
  goal_text TEXT NOT NULL,
  summary_text TEXT NOT NULL DEFAULT '',
  transcript_text TEXT NOT NULL DEFAULT '',
  file_names_text TEXT NOT NULL DEFAULT '',
  participant_text TEXT NOT NULL DEFAULT '',
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE,
  FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE VIRTUAL TABLE IF NOT EXISTS run_history_search_fts USING fts5(
  run_id UNINDEXED,
  title_text,
  goal_text,
  summary_text,
  transcript_text,
  file_names_text,
  participant_text,
  tokenize = 'porter unicode61'
);

CREATE TABLE IF NOT EXISTS run_change_records (
  run_id TEXT NOT NULL,
  tool_call_id TEXT NOT NULL,
  path TEXT NOT NULL,
  original_content TEXT,
  proposed_content TEXT,
  base_content_hash TEXT NOT NULL,
  approval_state TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  occurred_at TEXT NOT NULL,
  PRIMARY KEY (run_id, tool_call_id, path),
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_change_records_run_id ON run_change_records(run_id, occurred_at ASC);
CREATE INDEX IF NOT EXISTS idx_run_change_records_path ON run_change_records(path);

CREATE TABLE IF NOT EXISTS run_export_documents (
  run_id TEXT NOT NULL,
  export_path TEXT NOT NULL,
  generated_at TEXT NOT NULL,
  title TEXT NOT NULL,
  final_status TEXT NOT NULL,
  participants_json TEXT NOT NULL,
  timeline_markdown TEXT NOT NULL,
  changes_markdown TEXT,
  requested_by TEXT NOT NULL,
  PRIMARY KEY (run_id, export_path),
  FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_run_export_documents_run_id_generated_at ON run_export_documents(run_id, generated_at DESC);