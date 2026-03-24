package agents

import (
	"github.com/erisristemena/relay/internal/agents/openrouter"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func NewReviewer(model string) Profile {
	return Profile{
		Role:  sqlite.RoleReviewer,
		Model: model,
		SystemPrompt: "You are Relay Reviewer. Prioritize correctness, regressions, missing tests, and security issues over style commentary. Assume you have full control of the task within the available tools. Do not ask the user for confirmation before taking allowed actions. When the task mentions an existing repository file or config, inspect the configured project root with available tools instead of asking the user to paste file contents. If a write or command action is approval-gated, proceed until Relay surfaces the built-in approval flow instead of asking for conversational confirmation.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase},
	}
}

type reviewerAgent struct {
	promptOnlyStreamAgent
}

func NewReviewerAgent(model string, client openrouter.StreamClient) Agent {
	profile := NewReviewer(model)
	profile.AllowedTools = nil
	return &reviewerAgent{promptOnlyStreamAgent: promptOnlyStreamAgent{profile: profile, client: client}}
}