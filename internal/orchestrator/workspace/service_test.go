package workspace

import (
	"context"
	"testing"

	"github.com/erisristemena/relay/internal/config"
	"github.com/erisristemena/relay/internal/storage/sqlite"
)

func TestService_BootstrapUsesSavedSession(t *testing.T) {
	paths, store := newTestServiceStore(t)
	defer store.Close()

	service := NewService(store, paths)
	ctx := context.Background()
	created, err := store.CreateSession(ctx, "Resume me")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.LastSessionID = created.ID
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	snapshot, err := service.Bootstrap(ctx, "")
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	if snapshot.ActiveSessionID != created.ID {
		t.Fatalf("snapshot.ActiveSessionID = %q, want %q", snapshot.ActiveSessionID, created.ID)
	}
	if len(snapshot.Sessions) != 1 {
		t.Fatalf("len(snapshot.Sessions) = %d, want 1", len(snapshot.Sessions))
	}
}

func newTestServiceStore(t *testing.T) (config.Paths, *sqlite.Store) {
	t.Helper()
	paths, err := config.EnsurePaths(t.TempDir())
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}

	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	return paths, store
}
