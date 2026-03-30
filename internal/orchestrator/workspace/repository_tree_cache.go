package workspace

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/erisristemena/relay/internal/config"
	billy "github.com/go-git/go-billy/v5"
	git "github.com/go-git/go-git/v5"
	gitignore "github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

const (
	repositoryTreeStatusIdle  = "idle"
	repositoryTreeStatusReady = "ready"
	repositoryTreeStatusError = "error"
)

type repositoryTreeBuilder func(ctx context.Context, repositoryRoot string) (repositoryTreeSnapshot, error)

type repositoryTreeCacheEntry struct {
	RepositoryRoot string
	Status         string
	Snapshot       repositoryTreeSnapshot
	GeneratedAt    *time.Time
	ErrorMessage   string
}

type repositoryTreeSnapshot struct {
	RepositoryRoot string
	Paths          []string
}

func defaultRepositoryTreeBuilder(ctx context.Context, repositoryRoot string) (repositoryTreeSnapshot, error) {
	repo, err := git.PlainOpen(repositoryRoot)
	if err != nil {
		return repositoryTreeSnapshot{}, fmt.Errorf("open repository: %w", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return repositoryTreeSnapshot{}, fmt.Errorf("open repository worktree: %w", err)
	}

	paths := make([]string, 0)
	patterns, err := gitignore.ReadPatterns(worktree.Filesystem, nil)
	if err != nil {
		return repositoryTreeSnapshot{}, fmt.Errorf("read repository ignore patterns: %w", err)
	}
	matcher := gitignore.NewMatcher(patterns)
	if err := walkRepositoryTree(ctx, worktree.Filesystem, ".", matcher, &paths); err != nil {
		return repositoryTreeSnapshot{}, err
	}
	sort.Strings(paths)
	return repositoryTreeSnapshot{RepositoryRoot: repositoryRoot, Paths: paths}, nil
}

func walkRepositoryTree(ctx context.Context, filesystem billy.Filesystem, current string, matcher gitignore.Matcher, paths *[]string) error {
	entries, err := filesystem.ReadDir(current)
	if err != nil {
		return fmt.Errorf("read repository tree at %s: %w", current, err)
	}
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		name := strings.TrimSpace(entry.Name())
		if name == "" || name == ".git" {
			continue
		}
		relPath := path.Clean(path.Join(current, name))
		if relPath == "." {
			continue
		}
		if matcher != nil && matcher.Match(strings.Split(relPath, "/"), entry.IsDir()) {
			continue
		}
		*paths = append(*paths, relPath)
		if entry.IsDir() {
			if err := walkRepositoryTree(ctx, filesystem, relPath, matcher, paths); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) syncRepositoryTree(ctx context.Context, cfg config.Config) {
	connected := connectedRepositorySummary(cfg.SafePreferences())
	if connected.Status != "connected" || strings.TrimSpace(connected.Path) == "" {
		return
	}

	s.mu.Lock()
	entry, ok := s.repositoryTrees[connected.Path]
	s.mu.Unlock()
	if ok && entry.Status == repositoryTreeStatusReady && entry.RepositoryRoot == connected.Path {
		return
	}

	snapshot, err := s.repositoryTreeBuilder(ctx, connected.Path)
	now := time.Now().UTC()
	if err != nil {
		s.mu.Lock()
		s.repositoryTrees[connected.Path] = repositoryTreeCacheEntry{
			RepositoryRoot: connected.Path,
			Status:         repositoryTreeStatusError,
			ErrorMessage:   "Relay could not build the repository tree for the connected repository.",
			GeneratedAt:    &now,
		}
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.repositoryTrees[connected.Path] = repositoryTreeCacheEntry{
		RepositoryRoot: connected.Path,
		Status:         repositoryTreeStatusReady,
		Snapshot:       snapshot,
		GeneratedAt:    &now,
	}
	s.mu.Unlock()
}

func (s *Service) currentRepositoryTree(connected ConnectedRepositorySummary) repositoryTreeCacheEntry {
	if strings.TrimSpace(connected.Path) == "" {
		return repositoryTreeCacheEntry{Status: repositoryTreeStatusIdle}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.repositoryTrees[connected.Path]
	if !ok {
		return repositoryTreeCacheEntry{
			RepositoryRoot: connected.Path,
			Status:         repositoryTreeStatusError,
			ErrorMessage:   "Relay could not build the repository tree for the connected repository.",
		}
	}
	return entry
}