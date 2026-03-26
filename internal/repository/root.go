package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	git "github.com/go-git/go-git/v5"
)

type RootStatus struct {
	Configured bool
	Valid      bool
	Message    string
	Root       string
}

func ValidateRoot(root string) RootStatus {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return RootStatus{
			Configured: false,
			Valid:      false,
			Message:    "Repository-reading tools stay disabled until Relay has a valid project_root in config.toml.",
		}
	}

	if !filepath.IsAbs(trimmedRoot) {
		return RootStatus{
			Configured: true,
			Valid:      false,
			Message:    "The saved project_root must be an absolute path.",
		}
	}

	absoluteRoot, err := filepath.Abs(trimmedRoot)
	if err != nil {
		return RootStatus{
			Configured: true,
			Valid:      false,
			Message:    "Relay could not normalize the saved project_root. Update config.toml to point at an accessible local Git repository.",
		}
	}

	info, err := os.Stat(absoluteRoot)
	if err != nil {
		return RootStatus{
			Configured: true,
			Valid:      false,
			Message:    "Relay could not read the saved project_root. Update config.toml to point at an accessible local Git repository.",
		}
	}

	if !info.IsDir() {
		return RootStatus{
			Configured: true,
			Valid:      false,
			Message:    "The saved project_root must point to a directory.",
		}
	}

	repo, err := git.PlainOpen(absoluteRoot)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return RootStatus{
				Configured: true,
				Valid:      false,
				Message:    "Relay only enables repository-aware tools when project_root points to a local Git repository root.",
			}
		}
		return RootStatus{
			Configured: true,
			Valid:      false,
			Message:    fmt.Sprintf("Relay could not open the saved project_root as a local Git repository: %v", err),
		}
	}

	if _, err := repo.Worktree(); err != nil {
		return RootStatus{
			Configured: true,
			Valid:      false,
			Message:    "Relay only supports local Git working trees for project_root.",
		}
	}

	return RootStatus{Configured: true, Valid: true, Root: absoluteRoot}
}
