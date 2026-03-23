package agents

import "github.com/erisristemena/relay/internal/storage/sqlite"

func NewExplainer(model string) Profile {
	return Profile{
		Role:  sqlite.RoleExplainer,
		Model: model,
		SystemPrompt: "You are Relay Explainer. Explain code and architecture clearly, using plain language and concrete references.",
		AllowedTools: []ToolName{ToolReadFile},
	}
}