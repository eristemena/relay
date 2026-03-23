package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Name string

const (
	ReadFileName       Name = "read_file"
	SearchCodebaseName Name = "search_codebase"
	WriteFileName      Name = "write_file"
	RunCommandName     Name = "run_command"
)

type Definition struct {
	Name             Name           `json:"name"`
	Description      string         `json:"description"`
	RequiresApproval bool           `json:"requires_approval"`
	Parameters       map[string]any `json:"parameters,omitempty"`
}

type Result struct {
	Output  string         `json:"output"`
	Preview map[string]any `json:"preview"`
}

type Tool interface {
	Definition() Definition
	Execute(ctx context.Context, arguments json.RawMessage) (Result, error)
}

type Catalog struct {
	definitions map[string]Tool
}

func NewCatalog(projectRoot string) *Catalog {
	return &Catalog{definitions: map[string]Tool{
		string(ReadFileName):       NewReadFileTool(projectRoot),
		string(SearchCodebaseName): NewSearchCodebaseTool(projectRoot),
		string(WriteFileName):      NewWriteFileTool(projectRoot),
		string(RunCommandName):     NewRunCommandTool(projectRoot),
	}}
}

func (c *Catalog) Definitions() []Definition {
	definitions := make([]Definition, 0, len(c.definitions))
	for _, tool := range c.definitions {
		definitions = append(definitions, tool.Definition())
	}
	return definitions
}

func (c *Catalog) Lookup(name string) (Tool, bool) {
	tool, ok := c.definitions[name]
	return tool, ok
}

func RedactText(input string) string {
	text := strings.TrimSpace(input)
	if text == "" {
		return text
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)([^\s]+)`),
		regexp.MustCompile(`(?i)(token\s*[:=]\s*)([^\s]+)`),
		regexp.MustCompile(`(?i)(authorization:\s*bearer\s+)([^\s]+)`),
	}
	for _, pattern := range patterns {
		text = pattern.ReplaceAllString(text, `${1}[redacted]`)
	}
	if len(text) > 400 {
		return text[:400] + "..."
	}
	return text
}

func SafePreview(summary string, extra map[string]any) map[string]any {
	preview := map[string]any{"summary": RedactText(summary)}
	for key, value := range extra {
		switch typed := value.(type) {
		case string:
			preview[key] = RedactText(typed)
		default:
			preview[key] = typed
		}
	}
	return preview
}

func UnsupportedToolError(name string) error {
	return fmt.Errorf("unsupported tool: %s", name)
}