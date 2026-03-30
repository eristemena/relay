package ws

import "encoding/json"

const (
	TypeWorkspaceBootstrapRequest = "workspace.bootstrap.request"
	TypeWorkspaceBootstrap        = "workspace.bootstrap"
	TypeWorkspaceStatus           = "workspace.status"
	TypeRepositoryBrowseRequest   = "repository.browse.request"
	TypeRepositoryBrowseResult    = "repository.browse.result"
	TypeRepositoryTreeRequest     = "repository.tree.request"
	TypeRepositoryTreeResult      = "repository.tree.result"
	TypeFileTouched               = "file_touched"
	TypeRepositoryGraphStatus     = "repository_graph_status"
	TypeSessionCreate             = "session.create"
	TypeSessionCreated            = "session.created"
	TypeSessionOpen               = "session.open"
	TypeSessionOpened             = "session.opened"
	TypePreferencesSave           = "preferences.save"
	TypePreferencesSaved          = "preferences.saved"
	TypeAgentRunSubmit            = "agent.run.submit"
	TypeAgentRunOpen              = "agent.run.open"
	TypeRunHistoryQuery           = "run.history.query"
	TypeRunHistoryResult          = "run.history.result"
	TypeRunHistoryDetailsRequest  = "run.history.details.request"
	TypeRunHistoryDetailsResult   = "run.history.details.result"
	TypeRunHistoryExportRequest   = "run.history.export.request"
	TypeRunHistoryExportResult    = "run.history.export.result"
	TypeAgentRunReplayControl     = "agent.run.replay.control"
	TypeAgentRunReplayState       = "agent.run.replay.state"
	TypeAgentRunCancel            = "agent.run.cancel"
	TypeAgentRunApprovalRespond   = "agent.run.approval.respond"
	TypeApprovalRequest           = "approval_request"
	TypeApprovalStateChanged      = "approval_state_changed"
	TypeStateChange               = "state_change"
	TypeToken                     = "token"
	TypeToolCall                  = "tool_call"
	TypeToolResult                = "tool_result"
	TypeComplete                  = "complete"
	TypeAgentSpawned              = "agent_spawned"
	TypeAgentStateChanged         = "agent_state_changed"
	TypeTaskAssigned              = "task_assigned"
	TypeHandoffStart              = "handoff_start"
	TypeHandoffComplete           = "handoff_complete"
	TypeAgentError                = "agent_error"
	TypeRunComplete               = "run_complete"
	TypeRunError                  = "run_error"
	TypeError                     = "error"
)

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type OutboundEnvelope[T any] struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   T      `json:"payload"`
}

type BootstrapRequestPayload struct {
	LastSessionID string `json:"last_session_id,omitempty"`
}

type RepositoryBrowseRequestPayload struct {
	Path       string `json:"path,omitempty"`
	ShowHidden bool   `json:"show_hidden,omitempty"`
}

type RepositoryDirectoryPayload struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	IsGitRepository bool   `json:"is_git_repository"`
}

type RepositoryBrowseResultPayload struct {
	Path        string                       `json:"path"`
	Directories []RepositoryDirectoryPayload `json:"directories"`
}

type RepositoryTreeRequestPayload struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id,omitempty"`
}

type TouchedFilePayload struct {
	RunID     string `json:"run_id"`
	AgentID   string `json:"agent_id"`
	FilePath  string `json:"file_path"`
	TouchType string `json:"touch_type"`
}

type RepositoryTreeResultPayload struct {
	SessionID      string               `json:"session_id,omitempty"`
	RunID          string               `json:"run_id,omitempty"`
	RepositoryRoot string               `json:"repository_root,omitempty"`
	Status         string               `json:"status"`
	Message        string               `json:"message,omitempty"`
	Paths          []string             `json:"paths,omitempty"`
	TouchedFiles   []TouchedFilePayload `json:"touched_files,omitempty"`
}

type FileTouchedPayload struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
	AgentID   string `json:"agent_id"`
	Role      string `json:"role"`
	FilePath  string `json:"file_path"`
	TouchType string `json:"touch_type"`
	Replay    bool   `json:"replay"`
}

type RepositoryGraphStatusPayload struct {
	RepositoryRoot string                       `json:"repository_root,omitempty"`
	Status         string                       `json:"status"`
	Message        string                       `json:"message,omitempty"`
	Nodes          []RepositoryGraphNodePayload `json:"nodes,omitempty"`
	Edges          []RepositoryGraphEdgePayload `json:"edges,omitempty"`
}

type RepositoryGraphNodePayload struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type RepositoryGraphEdgePayload struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind,omitempty"`
}

type ConnectedRepositoryView struct {
	Path    string `json:"path,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type SessionCreatePayload struct {
	DisplayName string `json:"display_name,omitempty"`
}

type SessionOpenPayload struct {
	SessionID string `json:"session_id"`
}

type CredentialPayload struct {
	Provider string `json:"provider"`
	Label    string `json:"label,omitempty"`
	Secret   string `json:"secret"`
}

type PreferencesSavePayload struct {
	PreferredPort      *int                `json:"preferred_port,omitempty"`
	AppearanceVariant  *string             `json:"appearance_variant,omitempty"`
	Credentials        []CredentialPayload `json:"credentials,omitempty"`
	OpenRouterAPIKey   *string             `json:"openrouter_api_key,omitempty"`
	ProjectRoot        *string             `json:"project_root,omitempty"`
	OpenBrowserOnStart *bool               `json:"open_browser_on_start,omitempty"`
}

type AgentRunSubmitPayload struct {
	SessionID string `json:"session_id"`
	Task      string `json:"task"`
}

type AgentRunOpenPayload struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
}

type RunHistoryQueryPayload struct {
	SessionID string `json:"session_id"`
	Query     string `json:"query,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	DateFrom  string `json:"date_from,omitempty"`
	DateTo    string `json:"date_to,omitempty"`
}

type RunHistoryDetailsRequestPayload struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
}

type RunHistoryExportRequestPayload struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
}

type AgentRunReplayControlPayload struct {
	SessionID string  `json:"session_id"`
	RunID     string  `json:"run_id"`
	Action    string  `json:"action"`
	CursorMS  int     `json:"cursor_ms,omitempty"`
	Speed     float64 `json:"speed,omitempty"`
}

type AgentRunCancelPayload struct {
	SessionID string `json:"session_id"`
	RunID     string `json:"run_id"`
}

type AgentRunApprovalRespondPayload struct {
	SessionID  string `json:"session_id"`
	RunID      string `json:"run_id"`
	ToolCallID string `json:"tool_call_id"`
	Decision   string `json:"decision"`
}

type SessionSummary struct {
	ID           string `json:"id"`
	DisplayName  string `json:"display_name"`
	CreatedAt    string `json:"created_at"`
	LastOpenedAt string `json:"last_opened_at"`
	Status       string `json:"status"`
	HasActivity  bool   `json:"has_activity"`
}

type PreferencesView struct {
	PreferredPort         int             `json:"preferred_port"`
	AppearanceVariant     string          `json:"appearance_variant"`
	HasCredentials        bool            `json:"has_credentials"`
	OpenRouterConfigured  bool            `json:"openrouter_configured"`
	ProjectRoot           string          `json:"project_root"`
	ProjectRootConfigured bool            `json:"project_root_configured"`
	ProjectRootValid      bool            `json:"project_root_valid"`
	ProjectRootMessage    string          `json:"project_root_message,omitempty"`
	AgentModels           AgentModelsView `json:"agent_models"`
	OpenBrowserOnStart    bool            `json:"open_browser_on_start"`
}

type AgentModelsView struct {
	Planner   string `json:"planner"`
	Coder     string `json:"coder"`
	Reviewer  string `json:"reviewer"`
	Tester    string `json:"tester"`
	Explainer string `json:"explainer"`
}

type UIState struct {
	HistoryState string `json:"history_state"`
	CanvasState  string `json:"canvas_state"`
	SaveState    string `json:"save_state"`
}

type WorkspaceSnapshotPayload struct {
	ActiveSessionID     string                   `json:"active_session_id"`
	Sessions            []SessionSummary         `json:"sessions"`
	Preferences         PreferencesView          `json:"preferences"`
	ConnectedRepository ConnectedRepositoryView  `json:"connected_repository"`
	UIState             UIState                  `json:"ui_state"`
	ActiveRunID         string                   `json:"active_run_id,omitempty"`
	RunSummaries        []AgentRunSummary        `json:"run_summaries,omitempty"`
	PendingApprovals    []ApprovalRequestPayload `json:"pending_approvals,omitempty"`
	CredentialStatus    CredentialStatusView     `json:"credential_status"`
	Warnings            []string                 `json:"warnings,omitempty"`
}

type AgentRunSummary struct {
	ID              string `json:"id"`
	GeneratedTitle  string `json:"generated_title,omitempty"`
	TaskTextPreview string `json:"task_text_preview"`
	Role            string `json:"role"`
	Model           string `json:"model"`
	State           string `json:"state"`
	ErrorCode       string `json:"error_code,omitempty"`
	StartedAt       string `json:"started_at"`
	CompletedAt     string `json:"completed_at,omitempty"`
	HasToolActivity bool   `json:"has_tool_activity"`
	AgentCount      int    `json:"agent_count,omitempty"`
	FinalStatus     string `json:"final_status,omitempty"`
	HasFileChanges  bool   `json:"has_file_changes,omitempty"`
}

type RunChangeRecordPayload struct {
	ToolCallID     string `json:"tool_call_id"`
	Path           string `json:"path"`
	OriginalContent string `json:"original_content,omitempty"`
	ProposedContent string `json:"proposed_content,omitempty"`
	BaseContentHash string `json:"base_content_hash,omitempty"`
	ApprovalState   string `json:"approval_state,omitempty"`
	OccurredAt      string `json:"occurred_at,omitempty"`
}

type RunHistoryResultPayload struct {
	SessionID string            `json:"session_id"`
	Query     string            `json:"query,omitempty"`
	FilePath  string            `json:"file_path,omitempty"`
	DateFrom  string            `json:"date_from,omitempty"`
	DateTo    string            `json:"date_to,omitempty"`
	Runs      []AgentRunSummary `json:"runs"`
}

type RunHistoryDetailsResultPayload struct {
	SessionID      string                   `json:"session_id"`
	RunID          string                   `json:"run_id"`
	GeneratedTitle string                   `json:"generated_title,omitempty"`
	FinalStatus    string                   `json:"final_status,omitempty"`
	AgentCount     int                      `json:"agent_count,omitempty"`
	ChangeRecords  []RunChangeRecordPayload `json:"change_records,omitempty"`
}

type AgentRunReplayStatePayload struct {
	SessionID         string  `json:"session_id"`
	RunID             string  `json:"run_id"`
	Status            string  `json:"status"`
	CursorMS          int     `json:"cursor_ms"`
	DurationMS        int     `json:"duration_ms"`
	Speed             float64 `json:"speed"`
	SelectedTimestamp string  `json:"selected_timestamp,omitempty"`
}

type RunHistoryExportResultPayload struct {
	SessionID   string `json:"session_id"`
	RunID       string `json:"run_id"`
	Status      string `json:"status"`
	ExportPath  string `json:"export_path,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
}

type CredentialStatusView struct {
	Configured bool `json:"configured"`
}

type WorkspaceStatusPayload struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
}

type ErrorPayload struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	SessionID  string `json:"session_id,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	AgentID    string `json:"agent_id,omitempty"`
	Sequence   int64  `json:"sequence,omitempty"`
	Replay     bool   `json:"replay,omitempty"`
	Role       string `json:"role,omitempty"`
	Model      string `json:"model,omitempty"`
	Terminal   bool   `json:"terminal,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type ApprovalRequestPayload struct {
	SessionID      string         `json:"session_id"`
	RunID          string         `json:"run_id"`
	Role           string         `json:"role,omitempty"`
	Model          string         `json:"model,omitempty"`
	ToolCallID     string         `json:"tool_call_id"`
	ToolName       string         `json:"tool_name"`
	RequestKind    string         `json:"request_kind,omitempty"`
	Status         string         `json:"status,omitempty"`
	RepositoryRoot string         `json:"repository_root,omitempty"`
	InputPreview   map[string]any `json:"input_preview"`
	DiffPreview    map[string]any `json:"diff_preview,omitempty"`
	CommandPreview map[string]any `json:"command_preview,omitempty"`
	Message        string         `json:"message"`
	OccurredAt     string         `json:"occurred_at,omitempty"`
}

type ApprovalStateChangedPayload struct {
	SessionID  string `json:"session_id"`
	RunID      string `json:"run_id"`
	Role       string `json:"role,omitempty"`
	Model      string `json:"model,omitempty"`
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	OccurredAt string `json:"occurred_at,omitempty"`
	Sequence   int64  `json:"sequence,omitempty"`
	Replay     bool   `json:"replay,omitempty"`
}

type OrchestrationEventBase struct {
	SessionID  string `json:"session_id"`
	RunID      string `json:"run_id"`
	AgentID    string `json:"agent_id,omitempty"`
	Sequence   int64  `json:"sequence,omitempty"`
	Replay     bool   `json:"replay,omitempty"`
	Role       string `json:"role,omitempty"`
	Model      string `json:"model,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

type AgentSpawnedPayload struct {
	OrchestrationEventBase
	Label      string `json:"label"`
	SpawnOrder int    `json:"spawn_order"`
}

type AgentStateChangedPayload struct {
	OrchestrationEventBase
	State        string `json:"state"`
	Message      string `json:"message"`
	TokensUsed   *int   `json:"tokens_used,omitempty"`
	ContextLimit *int   `json:"context_limit,omitempty"`
}

type TaskAssignedPayload struct {
	OrchestrationEventBase
	TaskText string `json:"task_text"`
}

type HandoffPayload struct {
	OrchestrationEventBase
	FromAgentID string `json:"from_agent_id"`
	ToAgentID   string `json:"to_agent_id"`
	Reason      string `json:"reason"`
}

type RunCompletePayload struct {
	OrchestrationEventBase
	Summary      string `json:"summary"`
	TokensUsed   *int   `json:"tokens_used,omitempty"`
	ContextLimit *int   `json:"context_limit,omitempty"`
}

type CompletePayload struct {
	OrchestrationEventBase
	FinishReason string `json:"finish_reason"`
	TokensUsed   *int   `json:"tokens_used,omitempty"`
	ContextLimit *int   `json:"context_limit,omitempty"`
}
