package sqlite

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var ErrSessionNotFound = errors.New("session not found")

type Store struct {
	db *sql.DB
}

func NewStore(databasePath string) (*Store, error) {
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		ORDER BY datetime(last_opened_at) DESC, datetime(created_at) DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]Session, 0)
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (Session, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		WHERE id = ?
	`, sessionID)

	session, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}

	return session, nil
}

func (s *Store) CreateSession(ctx context.Context, displayName string) (Session, error) {
	if strings.TrimSpace(displayName) == "" {
		displayName = fmt.Sprintf("Session %s", time.Now().Format("2006-01-02 15:04"))
	}

	now := time.Now().UTC()
	sessionID, err := generateSessionID()
	if err != nil {
		return Session{}, err
	}

	snapshot := EmptySnapshot()
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return Session{}, fmt.Errorf("marshal snapshot: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET status = ?, updated_at = ? WHERE status = ?`, StatusIdle, now.Format(time.RFC3339), StatusActive); err != nil {
		return Session{}, fmt.Errorf("clear active sessions: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sessionID, strings.TrimSpace(displayName), now.Format(time.RFC3339), now.Format(time.RFC3339), now.Format(time.RFC3339), StatusActive, string(snapshotJSON)); err != nil {
		return Session{}, fmt.Errorf("insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Session{}, fmt.Errorf("commit session creation: %w", err)
	}

	return Session{
		ID:           sessionID,
		DisplayName:  strings.TrimSpace(displayName),
		CreatedAt:    now,
		UpdatedAt:    now,
		LastOpenedAt: now,
		Status:       StatusActive,
		Snapshot:     snapshot,
	}, nil
}

func (s *Store) OpenSession(ctx context.Context, sessionID string) (Session, error) {
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET status = ?, updated_at = ? WHERE status = ?`, StatusIdle, now.Format(time.RFC3339), StatusActive); err != nil {
		return Session{}, fmt.Errorf("clear active sessions: %w", err)
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, updated_at = ?, last_opened_at = ?
		WHERE id = ?
	`, StatusActive, now.Format(time.RFC3339), now.Format(time.RFC3339), sessionID)
	if err != nil {
		return Session{}, fmt.Errorf("open session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Session{}, fmt.Errorf("read updated rows: %w", err)
	}
	if rowsAffected == 0 {
		return Session{}, ErrSessionNotFound
	}

	session, err := scanSession(tx.QueryRowContext(ctx, `
		SELECT id, display_name, created_at, updated_at, last_opened_at, status, snapshot_json
		FROM sessions
		WHERE id = ?
	`, sessionID))
	if err != nil {
		return Session{}, err
	}

	if err := tx.Commit(); err != nil {
		return Session{}, fmt.Errorf("commit session open: %w", err)
	}

	return session, nil
}

func (s *Store) migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		body, err := migrationFiles.ReadFile(path.Join("migrations", entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if _, err := s.db.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(scanner rowScanner) (Session, error) {
	var (
		session      Session
		createdAt    string
		updatedAt    string
		lastOpenedAt string
		snapshotJSON string
	)

	if err := scanner.Scan(&session.ID, &session.DisplayName, &createdAt, &updatedAt, &lastOpenedAt, &session.Status, &snapshotJSON); err != nil {
		return Session{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse created_at: %w", err)
	}
	parsedUpdatedAt, err := time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse updated_at: %w", err)
	}
	parsedLastOpenedAt, err := time.Parse(time.RFC3339, lastOpenedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse last_opened_at: %w", err)
	}

	session.CreatedAt = parsedCreatedAt
	session.UpdatedAt = parsedUpdatedAt
	session.LastOpenedAt = parsedLastOpenedAt
	session.Snapshot = EmptySnapshot()
	if strings.TrimSpace(snapshotJSON) != "" {
		if err := json.Unmarshal([]byte(snapshotJSON), &session.Snapshot); err != nil {
			return Session{}, fmt.Errorf("decode snapshot: %w", err)
		}
	}

	return session, nil
}

func generateSessionID() (string, error) {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}

	return "session_" + hex.EncodeToString(buffer), nil
}
