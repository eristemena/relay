ALTER TABLE sessions ADD COLUMN project_root TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_sessions_project_root_last_opened_at
ON sessions(project_root, last_opened_at DESC);