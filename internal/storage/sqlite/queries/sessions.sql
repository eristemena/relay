-- name: ListSessions :many
SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
FROM sessions
ORDER BY datetime(last_opened_at) DESC, datetime(created_at) DESC;

-- name: GetSession :one
SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
FROM sessions
WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeactivateActiveSessions :exec
UPDATE sessions
SET status = ?, updated_at = ?
WHERE status = ?;

-- name: OpenSession :exec
UPDATE sessions
SET status = ?, updated_at = ?, last_opened_at = ?
WHERE id = ?;
