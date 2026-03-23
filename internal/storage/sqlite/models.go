package sqlite

import "time"

const (
	StatusActive   = "active"
	StatusIdle     = "idle"
	StatusArchived = "archived"
	RunStateAccepted   = "accepted"
	RunStateThinking   = "thinking"
	RunStateToolRunning = "tool_running"
	RunStateCompleted  = "completed"
	RunStateErrored    = "errored"
	EventTypeStateChange = "state_change"
	EventTypeToken      = "token"
	EventTypeToolCall   = "tool_call"
	EventTypeToolResult = "tool_result"
	EventTypeComplete   = "complete"
	EventTypeError      = "error"
)

type AgentRole string

const (
	RolePlanner   AgentRole = "planner"
	RoleCoder     AgentRole = "coder"
	RoleReviewer  AgentRole = "reviewer"
	RoleTester    AgentRole = "tester"
	RoleExplainer AgentRole = "explainer"
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

type AgentRun struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"session_id"`
	TaskText     string     `json:"task_text"`
	Role         AgentRole  `json:"role"`
	Model        string     `json:"model"`
	State        string     `json:"state"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorCode    string     `json:"error_code,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	FirstTokenAt *time.Time `json:"first_token_at,omitempty"`
}

type AgentRunEvent struct {
	RunID       string    `json:"run_id"`
	Sequence    int64     `json:"sequence"`
	EventType   string    `json:"event_type"`
	Role        AgentRole `json:"role"`
	Model       string    `json:"model"`
	PayloadJSON string    `json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}

type RunSummary struct {
	ID              string     `json:"id"`
	TaskTextPreview string     `json:"task_text_preview"`
	Role            AgentRole  `json:"role"`
	Model           string     `json:"model"`
	State           string     `json:"state"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	HasToolActivity bool       `json:"has_tool_activity"`
}

func EmptySnapshot() Snapshot {
	return Snapshot{
		ActivePanel: "canvas",
		CanvasState: map[string]any{"variant": "empty"},
		HasActivity: false,
	}
}

func (run AgentRun) Active() bool {
	return run.State == RunStateAccepted || run.State == RunStateThinking || run.State == RunStateToolRunning
}
