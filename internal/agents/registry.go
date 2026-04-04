package agents

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/eristemena/relay/internal/agents/openrouter"
	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type Registry struct {
	clientFactory func(apiKey string) openrouter.StreamClient
	models        config.AgentModels
}

func NewRegistry(models config.AgentModels) *Registry {
	return &Registry{
		clientFactory: func(apiKey string) openrouter.StreamClient {
			return openrouter.NewClient(apiKey)
		},
		models: models.WithDefaults(),
	}
}

func (r *Registry) WithClientFactory(factory func(apiKey string) openrouter.StreamClient) *Registry {
	r.clientFactory = factory
	return r
}

func (r *Registry) SelectProfile(task string) Profile {
	text := strings.ToLower(strings.TrimSpace(task))
	switch {
	case strings.Contains(text, "review") || strings.Contains(text, "regression"):
		return NewReviewer(r.models.Reviewer)
	case strings.Contains(text, "test") || strings.Contains(text, "coverage"):
		return NewTester(r.models.Tester)
	case strings.Contains(text, "explain") || strings.Contains(text, "why"):
		return NewExplainer(r.models.Explainer)
	case strings.Contains(text, "plan") || strings.Contains(text, "steps") || strings.Contains(text, "design"):
		return NewPlanner(r.models.Planner)
	default:
		return NewCoder(r.models.Coder)
	}
}

func (r *Registry) NewRunner(apiKey string, task string, toolExecutor ToolExecutor) Runner {
	profile := r.SelectProfile(task)
	client := r.clientFactory(apiKey)
	return &profileRunner{profile: profile, client: client, toolExecutor: toolExecutor}
}

func (r *Registry) NewAgent(apiKey string, role sqlite.AgentRole, toolExecutor ToolExecutor) Agent {
	profile := r.profileForRole(role)
	client := r.clientFactory(apiKey)
	return &profileRunner{profile: profile, client: client, toolExecutor: toolExecutor}
}

func (r *Registry) NewPromptOnlyAgent(apiKey string, role sqlite.AgentRole) Agent {
	client := r.clientFactory(apiKey)
	switch role {
	case sqlite.RolePlanner:
		return NewPlannerAgent(r.models.Planner, client)
	case sqlite.RoleReviewer:
		return NewReviewerAgent(r.models.Reviewer, client)
	case sqlite.RoleTester:
		return NewTesterAgent(r.models.Tester, client)
	case sqlite.RoleExplainer:
		return NewExplainerAgent(r.models.Explainer, client)
	default:
		return NewCoderAgent(r.models.Coder, client)
	}
}

func (r *Registry) profileForRole(role sqlite.AgentRole) Profile {
	switch role {
	case sqlite.RolePlanner:
		return NewPlanner(r.models.Planner)
	case sqlite.RoleReviewer:
		return NewReviewer(r.models.Reviewer)
	case sqlite.RoleTester:
		return NewTester(r.models.Tester)
	case sqlite.RoleExplainer:
		return NewExplainer(r.models.Explainer)
	default:
		return NewCoder(r.models.Coder)
	}
}

type profileRunner struct {
	profile Profile
	client  openrouter.StreamClient
	toolExecutor ToolExecutor
}

func (r *profileRunner) Profile() Profile {
	return r.profile
}

func (r *profileRunner) Run(ctx context.Context, task string, handlers StreamEventHandlers) error {
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(string(sqlite.RunStateThinking))
	}

	request := openrouter.StreamRequest{
		Model:        r.profile.Model,
		SystemPrompt: r.profile.SystemPrompt,
		UserPrompt:   task,
	}
	if r.toolExecutor != nil {
		request.Tools = toOpenRouterTools(r.toolExecutor.Definitions(r.profile.AllowedTools))
	}

	return r.client.Stream(ctx, request, openrouter.StreamHandlers{
		OnToken: func(text string) {
			if handlers.OnToken != nil {
				handlers.OnToken(text)
			}
		},
		ExecuteTool: func(ctx context.Context, call openrouter.ToolCall) (openrouter.ToolResult, error) {
			if r.toolExecutor == nil {
				return openrouter.ToolResult{}, nil
			}

			toolName := ToolName(call.Name)
			if handlers.OnToolCall != nil {
				handlers.OnToolCall(ToolCallEvent{
					ToolCallID:   call.ID,
					ToolName:     toolName,
					InputPreview: r.toolExecutor.PreviewToolCall(toolName, json.RawMessage(call.Arguments)),
				})
			}

			execution, err := r.toolExecutor.ExecuteTool(ctx, call.ID, toolName, json.RawMessage(call.Arguments))
			if err != nil {
				return openrouter.ToolResult{}, err
			}
			if handlers.OnToolResult != nil {
				handlers.OnToolResult(ToolResultEvent{
					ToolCallID:    execution.ToolCallID,
					ToolName:      execution.ToolName,
					Status:        execution.Status,
					ResultPreview: execution.ResultPreview,
				})
			}

			return openrouter.ToolResult{
				ToolCallID: execution.ToolCallID,
				Name:       string(execution.ToolName),
				Content:    execution.Content,
			}, nil
		},
		OnComplete: func(metadata openrouter.CompletionMetadata) {
			if handlers.OnComplete != nil {
				handlers.OnComplete(CompletionMetadata{
					FinishReason: metadata.FinishReason,
					TokensUsed:   metadata.TokensUsed,
				})
			}
		},
		OnError: func(code string, message string) {
			if handlers.OnError != nil {
				handlers.OnError(code, message)
			}
		},
	})
}

func toOpenRouterTools(definitions []ToolDefinition) []openrouter.ToolDefinition {
	if len(definitions) == 0 {
		return nil
	}

	tools := make([]openrouter.ToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, openrouter.ToolDefinition{
			Name:        string(definition.Name),
			Description: definition.Description,
			Parameters:  definition.Parameters,
		})
	}
	return tools
}

func stringProperty(description string) jsonschema.Definition {
	return jsonschema.Definition{Type: jsonschema.String, Description: description}
}

type promptOnlyStreamAgent struct {
	profile Profile
	client  openrouter.StreamClient
}

func (a *promptOnlyStreamAgent) Profile() Profile {
	return a.profile
}

func (a *promptOnlyStreamAgent) Run(ctx context.Context, task string, handlers StreamEventHandlers) error {
	if handlers.OnStateChange != nil {
		handlers.OnStateChange(sqlite.AgentExecutionStateThinking)
	}

	return a.client.Stream(ctx, openrouter.StreamRequest{
		Model:        a.profile.Model,
		SystemPrompt: a.profile.SystemPrompt,
		UserPrompt:   task,
	}, openrouter.StreamHandlers{
		OnToken: func(text string) {
			if handlers.OnToken != nil {
				handlers.OnToken(text)
			}
		},
		OnComplete: func(metadata openrouter.CompletionMetadata) {
			if handlers.OnComplete != nil {
				handlers.OnComplete(CompletionMetadata{
					FinishReason: metadata.FinishReason,
					TokensUsed:   metadata.TokensUsed,
				})
			}
		},
		OnError: func(code string, message string) {
			if handlers.OnError != nil {
				handlers.OnError(code, message)
			}
		},
	})
}