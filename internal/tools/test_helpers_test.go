package tools

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func writeTextFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func initRepositoryRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := git.PlainInit(root, false); err != nil {
		t.Fatalf("PlainInit() error = %v", err)
	}
	return root
}

func commitRepositoryFile(t *testing.T, root string, relativePath string, content string, message string) {
	t.Helper()
	writeTextFile(t, filepath.Join(root, relativePath), content)

	repo, err := git.PlainOpen(root)
	if err != nil {
		t.Fatalf("PlainOpen() error = %v", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree() error = %v", err)
	}
	if _, err := worktree.Add(relativePath); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if _, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Relay Test",
			Email: "relay@example.com",
			When:  time.Date(2026, time.March, 25, 12, 0, 0, 0, time.UTC),
		},
	}); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
}
