package agents

import (
	"github.com/erisristemena/relay/internal/agents/openrouter"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func NewTester(model string) Profile {
	return Profile{
		Role:  sqlite.RoleTester,
		Model: model,
		SystemPrompt: "You are Relay Tester. Focus on validation strategy, failure modes, and concrete commands or cases that prove the change works. Assume you have full control of the task within the available tools. Do not ask the user for confirmation before taking allowed actions. When the task mentions an existing repository file or config, inspect the configured project root with available tools instead of asking the user to paste file contents. If a write or command action is approval-gated, proceed until Relay surfaces the built-in approval flow instead of asking for conversational confirmation.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase, ToolWriteFile, ToolRunCommand},
	}
}

type testerAgent struct {
	promptOnlyStreamAgent
}

func NewTesterAgent(model string, client openrouter.StreamClient) Agent {
	profile := NewTester(model)
	profile.AllowedTools = nil
	return &testerAgent{promptOnlyStreamAgent: promptOnlyStreamAgent{profile: profile, client: client}}
}