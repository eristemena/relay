package workspace

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/erisristemena/relay/internal/repository"
)

type RepositoryBrowseInput struct {
	Path       string
	ShowHidden bool
}

type RepositoryDirectory struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	IsGitRepository bool   `json:"is_git_repository"`
}

type RepositoryBrowseResult struct {
	Path        string                `json:"path"`
	Directories []RepositoryDirectory `json:"directories"`
}

func (s *Service) BrowseRepository(_ context.Context, input RepositoryBrowseInput) (RepositoryBrowseResult, error) {
	targetPath := strings.TrimSpace(input.Path)
	if targetPath == "" {
		targetPath = s.paths.HomeDir
	}
	if !filepath.IsAbs(targetPath) {
		return RepositoryBrowseResult{}, fmt.Errorf("Relay can only browse absolute local directories")
	}

	absolutePath, err := filepath.Abs(targetPath)
	if err != nil {
		return RepositoryBrowseResult{}, fmt.Errorf("Relay could not normalize that directory path")
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return RepositoryBrowseResult{}, fmt.Errorf("Relay could not read that directory path")
	}
	if !info.IsDir() {
		return RepositoryBrowseResult{}, fmt.Errorf("Relay can only browse directories")
	}

	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return RepositoryBrowseResult{}, fmt.Errorf("Relay could not list directories under that path")
	}

	directories := make([]RepositoryDirectory, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !input.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		childPath := filepath.Join(absolutePath, name)
		rootStatus := repository.ValidateRoot(childPath)
		directories = append(directories, RepositoryDirectory{
			Name:            name,
			Path:            childPath,
			IsGitRepository: rootStatus.Valid,
		})
	}

	slices.SortFunc(directories, func(left RepositoryDirectory, right RepositoryDirectory) int {
		return strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})

	return RepositoryBrowseResult{Path: absolutePath, Directories: directories}, nil
}
