package workspace

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestDefaultRepositoryTreeBuilderBuildsSortedRelativePaths(t *testing.T) {
	repositoryRoot := t.TempDir()
	if _, err := git.PlainInit(repositoryRoot, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repositoryRoot, "docs", "guides"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "README.md"), []byte("relay\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(README.md) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "docs", "guides", "setup.md"), []byte("setup\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(setup.md) error = %v", err)
	}

	snapshot, err := defaultRepositoryTreeBuilder(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatalf("defaultRepositoryTreeBuilder() error = %v", err)
	}
	if snapshot.RepositoryRoot != repositoryRoot {
		t.Fatalf("snapshot.RepositoryRoot = %q, want %q", snapshot.RepositoryRoot, repositoryRoot)
	}
	wantPaths := []string{"README.md", "docs", "docs/guides", "docs/guides/setup.md"}
	if !reflect.DeepEqual(snapshot.Paths, wantPaths) {
		t.Fatalf("snapshot.Paths = %#v, want %#v", snapshot.Paths, wantPaths)
	}
}

func TestDefaultRepositoryTreeBuilderSkipsGitIgnoredPaths(t *testing.T) {
	repositoryRoot := t.TempDir()
	if _, err := git.PlainInit(repositoryRoot, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repositoryRoot, "node_modules", "left-pad"), 0o755); err != nil {
		t.Fatalf("MkdirAll(node_modules) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repositoryRoot, "src"), 0o755); err != nil {
		t.Fatalf("MkdirAll(src) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(.gitignore) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "node_modules", "left-pad", "index.js"), []byte("module.exports = {};\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.js) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "src", "app.ts"), []byte("export const app = true;\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(app.ts) error = %v", err)
	}

	snapshot, err := defaultRepositoryTreeBuilder(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatalf("defaultRepositoryTreeBuilder() error = %v", err)
	}
	if reflect.DeepEqual(snapshot.Paths, []string{}) {
		t.Fatal("snapshot.Paths is empty, want tracked paths")
	}
	for _, path := range snapshot.Paths {
		if strings.HasPrefix(path, "node_modules") {
			t.Fatalf("snapshot.Paths contains ignored path %q", path)
		}
	}
	wantPaths := []string{".gitignore", "src", "src/app.ts"}
	if !reflect.DeepEqual(snapshot.Paths, wantPaths) {
		t.Fatalf("snapshot.Paths = %#v, want %#v", snapshot.Paths, wantPaths)
	}
}