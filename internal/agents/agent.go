package agents

import (
	"context"
	"encoding/json"

	"github.com/erisristemena/relay/internal/storage/sqlite"
)

type ToolName string

const (
	ToolReadFile       ToolName = "read_file"
	ToolListFiles      ToolName = "list_files"
	ToolSearchCodebase ToolName = "search_codebase"
	ToolGitLog         ToolName = "git_log"
	ToolGitDiff        ToolName = "git_diff"
	ToolWriteFile      ToolName = "write_file"
	ToolRunCommand     ToolName = "run_command"
)

type Profile struct {
	Role         sqlite.AgentRole `json:"role"`
	SystemPrompt string           `json:"system_prompt"`
	Model        string           `json:"model"`
	AllowedTools []ToolName       `json:"allowed_tools"`
}

type ToolDefinition struct {
	Name        ToolName
	Description string
	Parameters  any
}

type ToolCallEvent struct {
	ToolCallID   string
	ToolName     ToolName
	InputPreview map[string]any
}

type ToolResultEvent struct {
	ToolCallID    string
	ToolName      ToolName
	Status        string
	ResultPreview map[string]any
}

type ToolExecutionResult struct {
	ToolCallID    string
	ToolName      ToolName
	Status        string
	Content       string
	ResultPreview map[string]any
}

type CompletionMetadata struct {
	FinishReason string
	TokensUsed   *int
	ContextLimit *int
}

type ToolExecutor interface {
	Definitions(allowedTools []ToolName) []ToolDefinition
	PreviewToolCall(name ToolName, arguments json.RawMessage) map[string]any
	ExecuteTool(ctx context.Context, toolCallID string, name ToolName, arguments json.RawMessage) (ToolExecutionResult, error)
}

type StreamEventHandlers struct {
	OnStateChange func(message string)
	OnToken       func(text string)
	OnToolCall    func(event ToolCallEvent)
	OnToolResult  func(event ToolResultEvent)
	OnComplete    func(metadata CompletionMetadata)
	OnError       func(code string, message string)
}

type Agent interface {
	Run(ctx context.Context, task string, handlers StreamEventHandlers) error
	Profile() Profile
}

type Runner = Agent

type promptOnlyAgentRunner interface {
	stream(ctx context.Context, request PromptOnlyRequest, handlers StreamEventHandlers) error
}

type PromptOnlyRequest struct {
	Model        string
	SystemPrompt string
	Task         string
}