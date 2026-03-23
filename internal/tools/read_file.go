package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFileInput struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

type ReadFileTool struct {
	projectRoot string
}

func NewReadFileTool(projectRoot string) *ReadFileTool {
	return &ReadFileTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *ReadFileTool) Definition() Definition {
	return Definition{
		Name:        ReadFileName,
		Description: "Read a text file inside Relay's configured project root.",
		Parameters: map[string]any{"path": "string", "start_line": "number", "end_line": "number"},
	}
}

func (t *ReadFileTool) Execute(_ context.Context, arguments json.RawMessage) (Result, error) {
	var input ReadFileInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode read_file arguments: %w", err)
	}

	resolvedPath, err := resolveWithinRoot(t.projectRoot, input.Path)
	if err != nil {
		return Result{}, err
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return Result{}, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	startLine := input.StartLine
	if startLine < 1 {
		startLine = 1
	}
	endLine := input.EndLine
	if endLine != 0 && endLine < startLine {
		endLine = startLine
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if lineNumber < startLine {
			continue
		}
		if endLine != 0 && lineNumber > endLine {
			break
		}
		builder.WriteString(scanner.Text())
		builder.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("scan file: %w", err)
	}

	relativePath, _ := filepath.Rel(t.projectRoot, resolvedPath)
	content := strings.TrimRight(builder.String(), "\n")
	return Result{Output: content, Preview: SafePreview("Loaded file content.", map[string]any{"path": relativePath})}, nil
}