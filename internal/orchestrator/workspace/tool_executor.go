package workspace

import (
	"context"
	"encoding/json"
	"time"

	"github.com/erisristemena/relay/internal/agents"
	toolspkg "github.com/erisristemena/relay/internal/tools"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type catalogToolExecutor struct {
	catalog   *toolspkg.Catalog
	approvals interface {
		RequestApproval(ctx context.Context, request ApprovalRequest) (ApprovalDecision, error)
	}
}

func newCatalogToolExecutor(projectRoot string, approvals interface {
	RequestApproval(ctx context.Context, request ApprovalRequest) (ApprovalDecision, error)
}) agents.ToolExecutor {
	return &catalogToolExecutor{catalog: toolspkg.NewCatalog(projectRoot), approvals: approvals}
}

func (e *catalogToolExecutor) Definitions(allowedTools []agents.ToolName) []agents.ToolDefinition {
	definitions := make([]agents.ToolDefinition, 0, len(allowedTools))
	for _, toolName := range allowedTools {
		tool, ok := e.catalog.Lookup(string(toolName))
		if !ok {
			continue
		}
		definition := tool.Definition()
		if definition.RequiresApproval && e.approvals == nil {
			continue
		}
		definitions = append(definitions, agents.ToolDefinition{
			Name:        toolName,
			Description: definition.Description,
			Parameters:  toolSchema(toolName),
		})
	}
	return definitions
}

func (e *catalogToolExecutor) PreviewToolCall(_ agents.ToolName, arguments json.RawMessage) map[string]any {
	if len(arguments) == 0 {
		return toolspkg.SafePreview("Tool call received.", map[string]any{})
	}

	var decoded any
	if err := json.Unmarshal(arguments, &decoded); err != nil {
		return toolspkg.SafePreview("Tool call received.", map[string]any{"arguments": toolspkg.RedactText(string(arguments))})
	}
	preview, ok := sanitizePreviewValue(decoded).(map[string]any)
	if !ok {
		return toolspkg.SafePreview("Tool call received.", map[string]any{"arguments": toolspkg.RedactText(string(arguments))})
	}
	return preview
}

func (e *catalogToolExecutor) ExecuteTool(ctx context.Context, toolCallID string, name agents.ToolName, arguments json.RawMessage) (agents.ToolExecutionResult, error) {
	tool, ok := e.catalog.Lookup(string(name))
	if !ok {
		message := toolspkg.RedactText(toolspkg.UnsupportedToolError(string(name)).Error())
		return agents.ToolExecutionResult{
			ToolCallID:    toolCallID,
			ToolName:      name,
			Status:        "error",
			Content:       message,
			ResultPreview: toolspkg.SafePreview("Tool failed.", map[string]any{"message": message}),
		}, nil
	}

	definition := tool.Definition()
	if definition.RequiresApproval {
		if e.approvals == nil {
			message := "Relay blocked the tool call because approval is unavailable."
			return agents.ToolExecutionResult{
				ToolCallID:    toolCallID,
				ToolName:      name,
				Status:        "rejected",
				Content:       message,
				ResultPreview: toolspkg.SafePreview("Tool blocked.", map[string]any{"message": message}),
			}, nil
		}

		runContext, ok := runExecutionContextFromContext(ctx)
		if !ok {
			message := "Relay blocked the tool call because the active run context is missing."
			return agents.ToolExecutionResult{
				ToolCallID:    toolCallID,
				ToolName:      name,
				Status:        "rejected",
				Content:       message,
				ResultPreview: toolspkg.SafePreview("Tool blocked.", map[string]any{"message": message}),
			}, nil
		}

		decision, err := e.approvals.RequestApproval(ctx, ApprovalRequest{
			SessionID:    runContext.SessionID,
			RunID:        runContext.RunID,
			ToolCallID:   toolCallID,
			ToolName:     name,
			InputPreview: e.PreviewToolCall(name, arguments),
			Message:      approvalMessage(name),
			OccurredAt:   time.Now().UTC(),
		})
		if err != nil {
			message := toolspkg.RedactText(err.Error())
			return agents.ToolExecutionResult{
				ToolCallID:    toolCallID,
				ToolName:      name,
				Status:        "error",
				Content:       message,
				ResultPreview: toolspkg.SafePreview("Tool failed.", map[string]any{"message": message}),
			}, nil
		}
		if !decision.Approved {
			message := ErrApprovalRejected.Error()
			return agents.ToolExecutionResult{
				ToolCallID:    toolCallID,
				ToolName:      name,
				Status:        "rejected",
				Content:       message,
				ResultPreview: toolspkg.SafePreview("Tool blocked.", map[string]any{"message": message}),
			}, nil
		}
	}

	result, err := tool.Execute(ctx, arguments)
	if err != nil {
		message := toolspkg.RedactText(err.Error())
		return agents.ToolExecutionResult{
			ToolCallID:    toolCallID,
			ToolName:      name,
			Status:        "error",
			Content:       message,
			ResultPreview: toolspkg.SafePreview("Tool failed.", map[string]any{"message": message}),
		}, nil
	}

	return agents.ToolExecutionResult{
		ToolCallID:    toolCallID,
		ToolName:      name,
		Status:        "completed",
		Content:       result.Output,
		ResultPreview: result.Preview,
	}, nil
}

func sanitizePreviewValue(value any) any {
	switch typed := value.(type) {
	case string:
		return toolspkg.RedactText(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizePreviewValue(item))
		}
		return items
	case map[string]any:
		preview := make(map[string]any, len(typed))
		for key, item := range typed {
			preview[key] = sanitizePreviewValue(item)
		}
		return preview
	default:
		return typed
	}
}

func toolSchema(name agents.ToolName) jsonschema.Definition {
	switch name {
	case agents.ToolReadFile:
		return jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"path":       {Type: jsonschema.String, Description: "Path relative to the configured project root."},
				"start_line": {Type: jsonschema.Number, Description: "Optional 1-based line to start reading from."},
				"end_line":   {Type: jsonschema.Number, Description: "Optional 1-based line to stop reading at."},
			},
			Required: []string{"path"},
		}
	case agents.ToolSearchCodebase:
		return jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"query":           {Type: jsonschema.String, Description: "Text or filename fragment to search for in the repository."},
				"include_pattern": {Type: jsonschema.String, Description: "Optional filename glob to limit the search."},
				"max_results":     {Type: jsonschema.Number, Description: "Optional maximum number of matches to return."},
			},
			Required: []string{"query"},
		}
	case agents.ToolWriteFile:
		return jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"path":    {Type: jsonschema.String, Description: "Path relative to the configured project root."},
				"content": {Type: jsonschema.String, Description: "Full file contents to write."},
			},
			Required: []string{"path", "content"},
		}
	case agents.ToolRunCommand:
		return jsonschema.Definition{
			Type: jsonschema.Object,
			Properties: map[string]jsonschema.Definition{
				"command": {Type: jsonschema.String, Description: "Command to execute from the configured project root."},
				"args": {
					Type: jsonschema.Array,
					Items: &jsonschema.Definition{Type: jsonschema.String},
					Description: "Optional command arguments.",
				},
			},
			Required: []string{"command"},
		}
	default:
		return jsonschema.Definition{Type: jsonschema.Object}
	}
}

func approvalMessage(name agents.ToolName) string {
	switch name {
	case agents.ToolWriteFile:
		return "Relay needs approval before it can write files inside the configured project root."
	case agents.ToolRunCommand:
		return "Relay needs approval before it can run a shell command from the configured project root."
	default:
		return "Relay needs approval before it can continue with this tool call."
	}
}