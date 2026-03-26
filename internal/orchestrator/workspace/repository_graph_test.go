package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryGraphSignatureChangesWhenWorktreeChanges(t *testing.T) {
	repositoryRoot := initWorkspaceRepositoryRoot(t)

	first, err := defaultRepositoryGraphSignature(repositoryRoot)
	if err != nil {
		t.Fatalf("defaultRepositoryGraphSignature() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	second, err := defaultRepositoryGraphSignature(repositoryRoot)
	if err != nil {
		t.Fatalf("defaultRepositoryGraphSignature() second error = %v", err)
	}
	if first == second {
		t.Fatalf("signature before and after worktree change = %q, want different value", first)
	}
}

func TestRepositoryGraphBuilderCollectsRelativeImportEdges(t *testing.T) {
	repositoryRoot := initWorkspaceRepositoryRoot(t)
	if err := os.MkdirAll(filepath.Join(repositoryRoot, "src", "lib"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "src", "index.ts"), []byte("import util from './lib/util'\nexport default util\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.ts) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repositoryRoot, "src", "lib", "util.ts"), []byte("export default 'ok'\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(util.ts) error = %v", err)
	}

	snapshot, err := defaultRepositoryGraphBuilder(context.Background(), repositoryRoot)
	if err != nil {
		t.Fatalf("defaultRepositoryGraphBuilder() error = %v", err)
	}
	if snapshot.RepositoryRoot != repositoryRoot {
		t.Fatalf("snapshot.RepositoryRoot = %q, want %q", snapshot.RepositoryRoot, repositoryRoot)
	}
	if len(snapshot.Nodes) != 2 {
		t.Fatalf("len(snapshot.Nodes) = %d, want 2", len(snapshot.Nodes))
	}
	if len(snapshot.Edges) != 1 {
		t.Fatalf("len(snapshot.Edges) = %d, want 1", len(snapshot.Edges))
	}
	if snapshot.Edges[0].Source != "src/index.ts" || snapshot.Edges[0].Target != "src/lib/util.ts" {
		t.Fatalf("snapshot.Edges[0] = %#v, want src/index.ts -> src/lib/util.ts", snapshot.Edges[0])
	}
}
