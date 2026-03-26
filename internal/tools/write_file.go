package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

const RequestKindFileWrite = "file_write"

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

func BuildWriteFilePreview(projectRoot string, input WriteFileInput) (map[string]any, error) {
	resolvedRoot, err := resolveWithinRoot(projectRoot, ".")
	if err != nil {
		return nil, err
	}
	resolvedPath, err := resolveWithinRoot(projectRoot, input.Path)
	if err != nil {
		return nil, err
	}
	originalContent, err := readWriteFileBaseContent(resolvedPath)
	if err != nil {
		return nil, err
	}
	relativePath, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("resolve relative write path: %w", err)
	}
	normalizedPath := filepath.ToSlash(relativePath)
	return map[string]any{
		"path":            normalizedPath,
		"request_kind":    RequestKindFileWrite,
		"repository_root": resolvedRoot,
		"target_path":     normalizedPath,
		"diff_preview": map[string]any{
			"target_path":       normalizedPath,
			"original_content":  originalContent,
			"proposed_content":  input.Content,
			"base_content_hash": hashWriteFileContent(originalContent),
		},
	}, nil
}

func CurrentWriteFileBaseHash(projectRoot string, requestedPath string) (string, error) {
	resolvedPath, err := resolveWithinRoot(projectRoot, requestedPath)
	if err != nil {
		return "", err
	}
	originalContent, err := readWriteFileBaseContent(resolvedPath)
	if err != nil {
		return "", err
	}
	return hashWriteFileContent(originalContent), nil
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

func readWriteFileBaseContent(resolvedPath string) (string, error) {
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read existing file content: %w", err)
	}
	return string(content), nil
}

func hashWriteFileContent(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}
