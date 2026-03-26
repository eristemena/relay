package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ListFilesInput struct {
	Path       string `json:"path,omitempty"`
	Recursive  bool   `json:"recursive,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type ListFilesTool struct {
	projectRoot string
}

func NewListFilesTool(projectRoot string) *ListFilesTool {
	return &ListFilesTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *ListFilesTool) Definition() Definition {
	return Definition{
		Name:        ListFilesName,
		Description: "List files and directories inside Relay's configured project root.",
		Parameters:  map[string]any{"path": "string", "recursive": "boolean", "max_results": "number"},
	}
}

func (t *ListFilesTool) Execute(_ context.Context, arguments json.RawMessage) (Result, error) {
	var input ListFilesInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode list_files arguments: %w", err)
	}

	resolvedPath, err := resolveWithinRoot(t.projectRoot, input.Path)
	if err != nil {
		return Result{}, err
	}

	maxResults := input.MaxResults
	if maxResults <= 0 || maxResults > 200 {
		maxResults = 100
	}

	entries := make([]string, 0, maxResults)
	if input.Recursive {
		err = filepath.WalkDir(resolvedPath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == resolvedPath {
				return nil
			}
			name := entry.Name()
			if entry.IsDir() && shouldSkipRepositoryDir(name) {
				return filepath.SkipDir
			}
			relative, relErr := filepath.Rel(t.projectRoot, path)
			if relErr != nil {
				return nil
			}
			if entry.IsDir() {
				relative += "/"
			}
			entries = append(entries, filepath.ToSlash(relative))
			if len(entries) >= maxResults {
				return fs.SkipAll
			}
			return nil
		})
		if err != nil && err != fs.SkipAll {
			return Result{}, fmt.Errorf("walk directory: %w", err)
		}
	} else {
		dirEntries, err := os.ReadDir(resolvedPath)
		if err != nil {
			return Result{}, fmt.Errorf("read directory: %w", err)
		}
		for _, entry := range dirEntries {
			if shouldSkipRepositoryDir(entry.Name()) {
				continue
			}
			fullPath := filepath.Join(resolvedPath, entry.Name())
			relative, relErr := filepath.Rel(t.projectRoot, fullPath)
			if relErr != nil {
				continue
			}
			if entry.IsDir() {
				relative += "/"
			}
			entries = append(entries, filepath.ToSlash(relative))
			if len(entries) >= maxResults {
				break
			}
		}
	}

	sort.Strings(entries)
	encoded, err := json.Marshal(entries)
	if err != nil {
		return Result{}, fmt.Errorf("encode list results: %w", err)
	}

	return Result{Output: string(encoded), Preview: SafePreview("Listed repository files.", map[string]any{"count": len(entries), "path": filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(resolvedPath, t.projectRoot), string(filepath.Separator)))})}, nil
}

func shouldSkipRepositoryDir(name string) bool {
	trimmed := strings.TrimSpace(name)
	return trimmed == ".git" || trimmed == "node_modules" || trimmed == ".next"
}
