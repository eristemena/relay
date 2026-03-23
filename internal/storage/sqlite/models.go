package sqlite

import "time"

const (
	StatusActive   = "active"
	StatusIdle     = "idle"
	StatusArchived = "archived"
)

type Snapshot struct {
	ActivePanel      string         `json:"active_panel,omitempty"`
	CanvasState      map[string]any `json:"canvas_state,omitempty"`
	HasActivity      bool           `json:"has_activity"`
	RecoverableError *SnapshotError `json:"recoverable_error,omitempty"`
}

type SnapshotError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Session struct {
	ID           string    `json:"id"`
	DisplayName  string    `json:"display_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	LastOpenedAt time.Time `json:"last_opened_at"`
	Status       string    `json:"status"`
	Snapshot     Snapshot  `json:"snapshot"`
}

func EmptySnapshot() Snapshot {
	return Snapshot{
		ActivePanel: "canvas",
		CanvasState: map[string]any{"variant": "empty"},
		HasActivity: false,
	}
}
