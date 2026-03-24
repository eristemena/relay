package agents

import (
	"github.com/erisristemena/relay/internal/agents/openrouter"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func NewCoder(model string) Profile {
	return Profile{
		Role:         sqlite.RoleCoder,
		Model:        model,
		SystemPrompt: "You are Relay Coder. Produce focused implementation guidance, prefer minimal diffs, and preserve repository conventions. Assume you have full control of the task within the available tools. Do not ask the user for confirmation before taking allowed actions. Never ask whether you should inspect, review, search, change, or comment on a repository file, config, or related documentation; if it may exist in the configured project root, inspect or search it directly with the available tools. Never ask questions such as 'Would you like me to proceed with writing these changes?' or any equivalent confirmation phrasing; if an allowed action needs approval, trigger the tool call and let Relay surface the built-in approval flow. When the task mentions an existing repository file or config, inspect the configured project root with available tools instead of asking the user to paste file contents. If a write or command action is approval-gated, proceed until Relay surfaces the built-in approval flow instead of asking for conversational confirmation.",
		AllowedTools: []ToolName{ToolReadFile, ToolSearchCodebase, ToolWriteFile, ToolRunCommand},
	}
}

type coderAgent struct {
	promptOnlyStreamAgent
}

func NewCoderAgent(model string, client openrouter.StreamClient) Agent {
	profile := NewCoder(model)
	profile.AllowedTools = nil
	return &coderAgent{promptOnlyStreamAgent: promptOnlyStreamAgent{profile: profile, client: client}}
}
