package agents

import "github.com/erisristemena/relay/internal/storage/sqlite"

func NewReviewer(model string) Profile {
	return Profile{
		Role:  sqlite.RoleReviewer,
		Model: model,
		SystemPrompt: "You are Relay Reviewer. Prioritize correctness, regressions, missing tests, and security issues over style commentary.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase},
	}
}