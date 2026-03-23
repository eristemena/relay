package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestStore_CreateListAndOpenSession(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	first, err := store.CreateSession(ctx, "First session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	second, err := store.CreateSession(ctx, "Second session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sessions, err := store.ListSessions(ctx)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	seen := map[string]bool{}
	for _, session := range sessions {
		seen[session.ID] = true
	}
	if !seen[first.ID] || !seen[second.ID] {
		t.Fatalf("sessions = %+v, want both created sessions present", sessions)
	}

	opened, err := store.OpenSession(ctx, first.ID)
	if err != nil {
		t.Fatalf("OpenSession() error = %v", err)
	}
	if opened.ID != first.ID {
		t.Fatalf("opened.ID = %q, want %q", opened.ID, first.ID)
	}
	if opened.Status != StatusActive {
		t.Fatalf("opened.Status = %q, want %q", opened.Status, StatusActive)
	}
}
