package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
)

type GitLogInput struct {
	MaxResults int `json:"max_results,omitempty"`
}

type gitLogEntry struct {
	Hash       string `json:"hash"`
	AuthorName string `json:"author_name"`
	AuthoredAt string `json:"authored_at"`
	Subject    string `json:"subject"`
}

type GitLogTool struct {
	projectRoot string
}

func NewGitLogTool(projectRoot string) *GitLogTool {
	return &GitLogTool{projectRoot: strings.TrimSpace(projectRoot)}
}

func (t *GitLogTool) Definition() Definition {
	return Definition{
		Name:        GitLogName,
		Description: "Show recent commit history for Relay's configured Git repository.",
		Parameters:  map[string]any{"max_results": "number"},
	}
}

func (t *GitLogTool) Execute(_ context.Context, arguments json.RawMessage) (Result, error) {
	var input GitLogInput
	if err := json.Unmarshal(arguments, &input); err != nil {
		return Result{}, fmt.Errorf("decode git_log arguments: %w", err)
	}

	resolvedRoot, err := resolveWithinRoot(t.projectRoot, ".")
	if err != nil {
		return Result{}, err
	}

	maxResults := input.MaxResults
	if maxResults <= 0 || maxResults > 25 {
		maxResults = 10
	}

	repo, err := git.PlainOpen(resolvedRoot)
	if err != nil {
		return Result{}, fmt.Errorf("open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return Result{}, fmt.Errorf("read repository head: %w", err)
	}

	iterator, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		return Result{}, fmt.Errorf("read repository log: %w", err)
	}
	defer iterator.Close()

	entries := make([]gitLogEntry, 0, maxResults)
	for len(entries) < maxResults {
		commit, nextErr := iterator.Next()
		if nextErr != nil {
			if nextErr == io.EOF {
				break
			}
			return Result{}, fmt.Errorf("iterate repository log: %w", nextErr)
		}

		subject := strings.TrimSpace(strings.Split(commit.Message, "\n")[0])
		entries = append(entries, gitLogEntry{
			Hash:       commit.Hash.String(),
			AuthorName: commit.Author.Name,
			AuthoredAt: commit.Author.When.UTC().Format(time.RFC3339),
			Subject:    subject,
		})
	}

	encoded, err := json.Marshal(entries)
	if err != nil {
		return Result{}, fmt.Errorf("encode git log results: %w", err)
	}

	return Result{Output: string(encoded), Preview: SafePreview("Loaded recent repository commits.", map[string]any{"count": len(entries)})}, nil
}
