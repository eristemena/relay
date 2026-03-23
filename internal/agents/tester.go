package agents

import "github.com/erisristemena/relay/internal/storage/sqlite"

func NewTester(model string) Profile {
	return Profile{
		Role:  sqlite.RoleTester,
		Model: model,
		SystemPrompt: "You are Relay Tester. Focus on validation strategy, failure modes, and concrete commands or cases that prove the change works.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase, ToolWriteFile, ToolRunCommand},
	}
}