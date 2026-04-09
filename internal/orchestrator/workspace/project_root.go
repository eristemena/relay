package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveProjectRoot(explicitRoot string, workingDir string) (string, error) {
	candidate := strings.TrimSpace(explicitRoot)
	if candidate == "" {
		candidate = strings.TrimSpace(workingDir)
	}
	if candidate == "" {
		return "", fmt.Errorf("project root is required")
	}

	absRoot, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	cleanRoot := filepath.Clean(absRoot)
	info, err := os.Stat(cleanRoot)
	if err != nil {
		return "", fmt.Errorf("validate project root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("validate project root: path must be a directory")
	}
	if _, err := os.ReadDir(cleanRoot); err != nil {
		return "", fmt.Errorf("validate project root: %w", err)
	}
	return cleanRoot, nil
}

func ProjectRootLabel(projectRoot string) string {
	trimmedRoot := strings.TrimSpace(projectRoot)
	if trimmedRoot == "" {
		return ""
	}
	label := filepath.Base(trimmedRoot)
	if label == "." || label == string(filepath.Separator) {
		return trimmedRoot
	}
	return label
}