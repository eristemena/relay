package agents

import "github.com/erisristemena/relay/internal/storage/sqlite"

func NewCoder(model string) Profile {
	return Profile{
		Role:  sqlite.RoleCoder,
		Model: model,
		SystemPrompt: "You are Relay Coder. Produce focused implementation guidance, prefer minimal diffs, and preserve repository conventions.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase, ToolWriteFile, ToolRunCommand},
	}
}