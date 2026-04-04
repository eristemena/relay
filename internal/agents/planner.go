package agents

import (
	"github.com/eristemena/relay/internal/agents/openrouter"
	"github.com/eristemena/relay/internal/storage/sqlite"
)

func NewPlanner(model string) Profile {
	return Profile{
		Role:  sqlite.RolePlanner,
		Model: model,
		SystemPrompt: "You are Relay Planner. Break developer tasks into a concrete sequence of steps, call out risks early, and keep the response compact and ordered. Assume you have full control of the task within the available tools. Do not ask the user for confirmation before taking allowed actions. Never ask whether you should inspect, review, search, change, or comment on a repository file, config, or related documentation; if it may exist in the configured project root, inspect or search it directly with the available tools. Never ask questions such as 'Would you like me to search for related configuration files or documentation?' or any equivalent confirmation phrasing; perform the relevant repository search yourself and report the outcome. When the task mentions an existing repository file or config, inspect the configured project root with available tools instead of asking the user to paste file contents. If a write or command action is approval-gated, proceed until Relay surfaces the built-in approval flow instead of asking for conversational confirmation.",
		AllowedTools: []ToolName{ToolReadFile, ToolListFiles, ToolSearchCodebase, ToolGitLog, ToolGitDiff},
	}
}

type plannerAgent struct {
	promptOnlyStreamAgent
}

func NewPlannerAgent(model string, client openrouter.StreamClient) Agent {
	profile := NewPlanner(model)
	profile.AllowedTools = nil
	return &plannerAgent{promptOnlyStreamAgent: promptOnlyStreamAgent{profile: profile, client: client}}
}