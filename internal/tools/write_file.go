package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WriteFileInput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WriteFileTool struct {
	projectRoot string
}

func NewWriteFileTool(projectRoot string) *WriteFileTool {
	return &WriteFileTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *WriteFileTool) Definition() Definition {
	return Definition{
		Name:             WriteFileName,
		Description:      "Write a file inside Relay's configured project root.",
		RequiresApproval: true,
		Parameters:       map[string]any{"path": "string", "content": "string"},
	}
}

func (t *WriteFileTool) Execute(_ context.Context, arguments json.RawMessage) (Result, error) {
	var input WriteFileInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode write_file arguments: %w", err)
	}

	resolvedPath, err := resolveWithinRoot(t.projectRoot, input.Path)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create destination directory: %w", err)
	}
	if err := os.WriteFile(resolvedPath, []byte(input.Content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	relativePath, _ := filepath.Rel(t.projectRoot, resolvedPath)
	return Result{Output: "ok", Preview: SafePreview("Wrote file content.", map[string]any{"path": relativePath})}, nil
}