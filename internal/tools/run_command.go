package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type RunCommandInput struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

type RunCommandTool struct {
	projectRoot string
}

func NewRunCommandTool(projectRoot string) *RunCommandTool {
	return &RunCommandTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *RunCommandTool) Definition() Definition {
	return Definition{
		Name:             RunCommandName,
		Description:      "Run a shell command from Relay's configured project root.",
		RequiresApproval: true,
		Parameters:       map[string]any{"command": "string", "args": "string[]"},
	}
}

func (t *RunCommandTool) Execute(ctx context.Context, arguments json.RawMessage) (Result, error) {
	var input RunCommandInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode run_command arguments: %w", err)
	}
	if _, err := resolveWithinRoot(t.projectRoot, "."); err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(input.Command) == "" {
		return Result{}, fmt.Errorf("command is required")
	}

	command := exec.CommandContext(ctx, input.Command, input.Args...)
	command.Dir = t.projectRoot
	output, err := command.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("run command: %w", err)
	}

	return Result{Output: string(output), Preview: SafePreview("Command completed.", map[string]any{"command": input.Command})}, nil
}