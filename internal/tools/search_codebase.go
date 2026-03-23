package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type SearchCodebaseInput struct {
	Query          string `json:"query"`
	IncludePattern string `json:"include_pattern,omitempty"`
	MaxResults     int    `json:"max_results,omitempty"`
}

type SearchCodebaseTool struct {
	projectRoot string
}

func NewSearchCodebaseTool(projectRoot string) *SearchCodebaseTool {
	return &SearchCodebaseTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *SearchCodebaseTool) Definition() Definition {
	return Definition{
		Name:        SearchCodebaseName,
		Description: "Search text files inside Relay's configured project root.",
		Parameters:  map[string]any{"query": "string", "include_pattern": "string", "max_results": "number"},
	}
}

func (t *SearchCodebaseTool) Execute(_ context.Context, arguments json.RawMessage) (Result, error) {
	var input SearchCodebaseInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode search_codebase arguments: %w", err)
	}
	if strings.TrimSpace(input.Query) == "" {
		return Result{}, fmt.Errorf("search query is required")
	}
	if _, err := resolveWithinRoot(t.projectRoot, "."); err != nil {
		return Result{}, err
	}

	maxResults := input.MaxResults
	if maxResults <= 0 || maxResults > 25 {
		maxResults = 10
	}

	matches := make([]string, 0, maxResults)
	needle := strings.ToLower(input.Query)
	err := filepath.WalkDir(t.projectRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			name := entry.Name()
			if name == ".git" || name == "node_modules" || name == ".next" {
				return filepath.SkipDir
			}
			return nil
		}
		if input.IncludePattern != "" {
			matched, err := filepath.Match(input.IncludePattern, filepath.Base(path))
			if err != nil || !matched {
				return nil
			}
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(string(body)), needle) {
			relativePath, _ := filepath.Rel(t.projectRoot, path)
			matches = append(matches, relativePath)
			if len(matches) >= maxResults {
				return fs.SkipAll
			}
		}
		return nil
	})
	if err != nil && err != fs.SkipAll {
		return Result{}, fmt.Errorf("walk project root: %w", err)
	}

	encoded, err := json.Marshal(matches)
	if err != nil {
		return Result{}, fmt.Errorf("encode search results: %w", err)
	}
	return Result{Output: string(encoded), Preview: SafePreview("Search completed.", map[string]any{"match_count": len(matches)})}, nil
}