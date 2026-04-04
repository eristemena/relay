package agents

import (
	"github.com/eristemena/relay/internal/agents/openrouter"
	"github.com/eristemena/relay/internal/storage/sqlite"
)

func NewExplainer(model string) Profile {
	return Profile{
		Role:  sqlite.RoleExplainer,
		Model: model,
		SystemPrompt: "You are Relay Explainer. Explain code and architecture clearly, using plain language and concrete references. Assume you have full control of the task within the available tools. Do not ask the user for confirmation before taking allowed actions. When the task mentions an existing repository file or config, inspect the configured project root with available tools instead of asking the user to paste file contents. If a write or command action is approval-gated, proceed until Relay surfaces the built-in approval flow instead of asking for conversational confirmation.",
		AllowedTools: []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff},
	}
}

type explainerAgent struct {
	promptOnlyStreamAgent
}

func NewExplainerAgent(model string, client openrouter.StreamClient) Agent {
	profile := NewExplainer(model)
	profile.AllowedTools = nil
	return &explainerAgent{promptOnlyStreamAgent: promptOnlyStreamAgent{profile: profile, client: client}}
}