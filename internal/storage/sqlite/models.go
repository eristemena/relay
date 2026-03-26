package sqlite

import "time"

const (
	StatusActive                  = "active"
	StatusIdle                    = "idle"
	StatusArchived                = "archived"
	RunStateAccepted              = "accepted"
	RunStateActive                = "active"
	RunStateThinking              = "thinking"
	RunStateToolRunning           = "tool_running"
	RunStateCompleted             = "completed"
	RunStateCancelled             = "cancelled"
	RunStateHalted                = "halted"
	RunStateErrored               = "errored"
	RunModeSingleAgent            = "single_agent"
	RunModeOrchestration          = "orchestration"
	EventTypeStateChange          = "state_change"
	EventTypeToken                = "token"
	EventTypeToolCall             = "tool_call"
	EventTypeToolResult           = "tool_result"
	EventTypeComplete             = "complete"
	EventTypeError                = "error"
	EventTypeAgentSpawned         = "agent_spawned"
	EventTypeAgentStateChanged    = "agent_state_changed"
	EventTypeTaskAssigned         = "task_assigned"
	EventTypeHandoffStart         = "handoff_start"
	EventTypeHandoffComplete      = "handoff_complete"
	EventTypeAgentError           = "agent_error"
	EventTypeRunComplete          = "run_complete"
	EventTypeRunError             = "run_error"
	EventTypeApprovalStateChanged = "approval_state_changed"
)

const (
	ApprovalStateProposed = "proposed"
	ApprovalStateApproved = "approved"
	ApprovalStateRejected = "rejected"
	ApprovalStateApplied  = "applied"
	ApprovalStateBlocked  = "blocked"
	ApprovalStateExpired  = "expired"
)

const (
	AgentExecutionStateQueued    = "queued"
	AgentExecutionStateAssigned  = "assigned"
	AgentExecutionStateThinking  = "thinking"
	AgentExecutionStateStreaming = "streaming"
	AgentExecutionStateCompleted = "completed"
	AgentExecutionStateErrored   = "errored"
	AgentExecutionStateCancelled = "cancelled"
	AgentExecutionStateBlocked   = "blocked"
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
	Mode         string     `json:"mode,omitempty"`
}

type AgentExecution struct {
	ID           string     `json:"id"`
	RunID        string     `json:"run_id"`
	Role         AgentRole  `json:"role"`
	Model        string     `json:"model"`
	State        string     `json:"state"`
	TaskText     string     `json:"task_text"`
	SpawnOrder   int        `json:"spawn_order"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorCode    string     `json:"error_code,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

type AgentRunEvent struct {
	RunID       string    `json:"run_id"`
	Sequence    int64     `json:"sequence"`
	EventType   string    `json:"event_type"`
	AgentID     string    `json:"agent_id,omitempty"`
	Role        AgentRole `json:"role"`
	Model       string    `json:"model"`
	PayloadJSON string    `json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}

type ApprovalRequest struct {
	ID               string     `json:"id"`
	SessionID        string     `json:"session_id"`
	RunID            string     `json:"run_id"`
	ToolCallID       string     `json:"tool_call_id"`
	ToolName         string     `json:"tool_name"`
	Role             AgentRole  `json:"role,omitempty"`
	Model            string     `json:"model,omitempty"`
	InputPreviewJSON string     `json:"input_preview_json"`
	Message          string     `json:"message"`
	State            string     `json:"state"`
	OccurredAt       time.Time  `json:"occurred_at"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	AppliedAt        *time.Time `json:"applied_at,omitempty"`
}

type RunSummary struct {
	ID              string     `json:"id"`
	TaskTextPreview string     `json:"task_text_preview"`
	Role            AgentRole  `json:"role"`
	Model           string     `json:"model"`
	State           string     `json:"state"`
	ErrorCode       string     `json:"error_code,omitempty"`
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
	return run.State == RunStateAccepted || run.State == RunStateActive || run.State == RunStateThinking || run.State == RunStateToolRunning
}
