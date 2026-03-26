package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	git "github.com/go-git/go-git/v5"
)

func TestService_BrowseRepositoryListsDirectoriesAndFlagsGitRoots(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	browseRoot := filepath.Join(paths.HomeDir, "browse-root")
	if err := os.MkdirAll(filepath.Join(browseRoot, "plain-dir"), 0o755); err != nil {
		t.Fatalf("MkdirAll(plain-dir) error = %v", err)
	}
	gitDir := filepath.Join(browseRoot, "repo-dir")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(repo-dir) error = %v", err)
	}
	if _, err := git.PlainInit(gitDir, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(browseRoot, ".hidden-dir"), 0o755); err != nil {
		t.Fatalf("MkdirAll(.hidden-dir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(browseRoot, "README.md"), []byte("ignore me\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	result, err := service.BrowseRepository(context.Background(), RepositoryBrowseInput{Path: browseRoot})
	if err != nil {
		t.Fatalf("BrowseRepository() error = %v", err)
	}
	if result.Path != browseRoot {
		t.Fatalf("result.Path = %q, want %q", result.Path, browseRoot)
	}
	if len(result.Directories) != 2 {
		t.Fatalf("len(result.Directories) = %d, want 2", len(result.Directories))
	}
	if result.Directories[0].Name != "plain-dir" || result.Directories[0].IsGitRepository {
		t.Fatalf("result.Directories[0] = %#v, want plain non-git directory", result.Directories[0])
	}
	if result.Directories[1].Name != "repo-dir" || !result.Directories[1].IsGitRepository {
		t.Fatalf("result.Directories[1] = %#v, want repo git directory", result.Directories[1])
	}

	withHidden, err := service.BrowseRepository(context.Background(), RepositoryBrowseInput{Path: browseRoot, ShowHidden: true})
	if err != nil {
		t.Fatalf("BrowseRepository(show hidden) error = %v", err)
	}
	if len(withHidden.Directories) != 3 {
		t.Fatalf("len(withHidden.Directories) = %d, want 3", len(withHidden.Directories))
	}
}

func TestService_BrowseRepositoryRejectsInvalidPaths(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	if _, err := service.BrowseRepository(context.Background(), RepositoryBrowseInput{Path: "relative/path"}); err == nil {
		t.Fatal("BrowseRepository(relative) error = nil, want validation failure")
	}
	filePath := filepath.Join(paths.HomeDir, "file.txt")
	if err := os.WriteFile(filePath, []byte("nope\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := service.BrowseRepository(context.Background(), RepositoryBrowseInput{Path: filePath}); err == nil {
		t.Fatal("BrowseRepository(file) error = nil, want validation failure")
	}
}