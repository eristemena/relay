-- name: ListSessions :many
SELECT id, display_name, project_root, created_at, updated_at, last_opened_at, status, snapshot_json
FROM sessions
ORDER BY datetime(last_opened_at) DESC, datetime(created_at) DESC;

-- name: GetSession :one
SELECT id, display_name, project_root, created_at, updated_at, last_opened_at, status, snapshot_json
FROM sessions
WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (id, display_name, project_root, created_at, updated_at, last_opened_at, status, snapshot_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetSessionByProjectRoot :one
SELECT id, display_name, project_root, created_at, updated_at, last_opened_at, status, snapshot_json
FROM sessions
WHERE project_root = ?
ORDER BY datetime(last_opened_at) DESC, datetime(created_at) DESC
LIMIT 1;

-- name: ListKnownProjects :many
SELECT id, display_name, project_root, created_at, updated_at, last_opened_at, status, snapshot_json
FROM sessions
WHERE project_root <> ''
ORDER BY datetime(last_opened_at) DESC, datetime(created_at) DESC;

-- name: DeactivateActiveSessions :exec
UPDATE sessions
SET status = ?, updated_at = ?
WHERE status = ?;

-- name: OpenSession :exec
UPDATE sessions
SET status = ?, updated_at = ?, last_opened_at = ?
WHERE id = ?;
