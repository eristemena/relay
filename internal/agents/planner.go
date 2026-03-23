package agents

import "github.com/erisristemena/relay/internal/storage/sqlite"

func NewPlanner(model string) Profile {
	return Profile{
		Role:  sqlite.RolePlanner,
		Model: model,
		SystemPrompt: "You are Relay Planner. Break developer tasks into a concrete sequence of steps, call out risks early, and keep the response compact and ordered.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase},
	}
}