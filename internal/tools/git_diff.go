package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type GitDiffInput struct {
	Path string `json:"path,omitempty"`
}

type GitDiffTool struct {
	projectRoot string
}

type gitDiffEntry struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Patch  string `json:"patch"`
}

func NewGitDiffTool(projectRoot string) *GitDiffTool {
	return &GitDiffTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *GitDiffTool) Definition() Definition {
	return Definition{
		Name:        GitDiffName,
		Description: "Show working tree diff details for Relay's configured Git repository.",
		Parameters:  map[string]any{"path": "string"},
	}
}

func (t *GitDiffTool) Execute(_ context.Context, arguments json.RawMessage) (Result, error) {
	var input GitDiffInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode git_diff arguments: %w", err)
	}

	resolvedRoot, err := resolveWithinRoot(t.projectRoot, ".")
	if err != nil {
		return Result{}, err
	}

	repo, err := git.PlainOpen(resolvedRoot)
	if err != nil {
		return Result{}, fmt.Errorf("open repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return Result{}, fmt.Errorf("load repository worktree: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return Result{}, fmt.Errorf("load repository status: %w", err)
	}

	filterPrefix := ""
	if strings.TrimSpace(input.Path) != "" {
		resolvedPath, resolveErr := resolveWithinRoot(t.projectRoot, input.Path)
		if resolveErr != nil {
			return Result{}, resolveErr
		}
		relativePath, relErr := filepath.Rel(resolvedRoot, resolvedPath)
		if relErr != nil {
			return Result{}, fmt.Errorf("resolve diff path: %w", relErr)
		}
		filterPrefix = filepath.ToSlash(relativePath)
	}

	changedPaths := make([]string, 0, len(status))
	for path, fileStatus := range status {
		if fileStatus.Staging == git.Unmodified && fileStatus.Worktree == git.Unmodified {
			continue
		}
		normalizedPath := filepath.ToSlash(path)
		if filterPrefix != "" && normalizedPath != filterPrefix && !strings.HasPrefix(normalizedPath, filterPrefix+"/") {
			continue
		}
		changedPaths = append(changedPaths, path)
	}
	sort.Strings(changedPaths)

	headCommit, headErr := repositoryHeadCommit(repo)
	if headErr != nil && headErr != plumbing.ErrReferenceNotFound {
		return Result{}, fmt.Errorf("read repository head commit: %w", headErr)
	}

	entries := make([]gitDiffEntry, 0, len(changedPaths))
	for _, changedPath := range changedPaths {
		fileStatus := status[changedPath]
		beforeContent, beforeErr := gitTrackedFileContent(headCommit, changedPath)
		if beforeErr != nil {
			return Result{}, fmt.Errorf("load tracked file %q: %w", changedPath, beforeErr)
		}
		afterContent, afterErr := gitWorkingTreeFileContent(resolvedRoot, changedPath)
		if afterErr != nil {
			return Result{}, fmt.Errorf("load working tree file %q: %w", changedPath, afterErr)
		}

		entries = append(entries, gitDiffEntry{
			Path:   filepath.ToSlash(changedPath),
			Status: describeGitStatus(*fileStatus),
			Patch:  buildRepositoryPatch(beforeContent, afterContent),
		})
	}

	encoded, err := json.Marshal(entries)
	if err != nil {
		return Result{}, fmt.Errorf("encode git diff results: %w", err)
	}

	return Result{Output: string(encoded), Preview: SafePreview("Loaded repository diff.", map[string]any{"count": len(entries)})}, nil
}

func repositoryHeadCommit(repo *git.Repository) (*object.Commit, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	return repo.CommitObject(head.Hash())
}

func gitTrackedFileContent(headCommit *object.Commit, path string) (string, error) {
	if headCommit == nil {
		return "", nil
	}
	tree, err := headCommit.Tree()
	if err != nil {
		return "", err
	}
	file, err := tree.File(path)
	if err != nil {
		if err == object.ErrFileNotFound {
			return "", nil
		}
		return "", err
	}
	content, err := file.Contents()
	if err != nil {
		return "", err
	}
	return content, nil
}

func gitWorkingTreeFileContent(root string, path string) (string, error) {
	body, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(body), nil
}

func describeGitStatus(status git.FileStatus) string {
	parts := make([]string, 0, 2)
	if status.Staging != git.Unmodified {
		parts = append(parts, "index:"+describeGitStatusCode(status.Staging))
	}
	if status.Worktree != git.Unmodified {
		parts = append(parts, "worktree:"+describeGitStatusCode(status.Worktree))
	}
	if len(parts) == 0 {
		return "unmodified"
	}
	return strings.Join(parts, ", ")
}

func describeGitStatusCode(code git.StatusCode) string {
	switch code {
	case git.Unmodified:
		return "unmodified"
	case git.Untracked:
		return "untracked"
	case git.Modified:
		return "modified"
	case git.Added:
		return "added"
	case git.Deleted:
		return "deleted"
	case git.Renamed:
		return "renamed"
	case git.Copied:
		return "copied"
	case git.UpdatedButUnmerged:
		return "conflict"
	default:
		return string(code)
	}
}

func buildRepositoryPatch(before string, after string) string {
	differ := diffmatchpatch.New()
	patches := differ.PatchMake(before, after)
	return differ.PatchToText(patches)
}
