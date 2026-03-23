package sqlite

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestStore_ListRunSummariesAndReplayEvents(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "relay.db"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	session, err := store.CreateSession(ctx, "Run history")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	longTask := strings.Repeat("review ", 20)
	firstRun, err := store.CreateAgentRun(ctx, session.ID, longTask, RoleReviewer, "model-a")
	if err != nil {
		t.Fatalf("CreateAgentRun() first error = %v", err)
	}
	if _, err := store.AppendRunEvent(ctx, firstRun.ID, EventTypeToolCall, firstRun.Role, firstRun.Model, `{"tool_name":"read_file"}`); err != nil {
		t.Fatalf("AppendRunEvent() tool call error = %v", err)
	}
	secondEvent, err := store.AppendRunEvent(ctx, firstRun.ID, EventTypeToolResult, firstRun.Role, firstRun.Model, `{"status":"completed"}`)
	if err != nil {
		t.Fatalf("AppendRunEvent() tool result error = %v", err)
	}
	now := time.Now().UTC()
	firstRun.State = RunStateCompleted
	firstRun.CompletedAt = &now
	if err := store.UpdateAgentRun(ctx, firstRun); err != nil {
		t.Fatalf("UpdateAgentRun() first error = %v", err)
	}

	secondRun, err := store.CreateAgentRun(ctx, session.ID, "Explain the replay path", RoleExplainer, "model-b")
	if err != nil {
		t.Fatalf("CreateAgentRun() second error = %v", err)
	}
	secondRun.State = RunStateErrored
	secondRun.ErrorCode = "run_failed"
	secondRun.ErrorMessage = "provider failure"
	secondRun.CompletedAt = &now
	if err := store.UpdateAgentRun(ctx, secondRun); err != nil {
		t.Fatalf("UpdateAgentRun() second error = %v", err)
	}

	summaries, err := store.ListRunSummaries(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListRunSummaries() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}
	if summaries[0].ID != secondRun.ID {
		t.Fatalf("summaries[0].ID = %q, want latest run %q", summaries[0].ID, secondRun.ID)
	}
	if summaries[0].HasToolActivity {
		t.Fatal("summaries[0].HasToolActivity = true, want false")
	}
	if summaries[1].ID != firstRun.ID {
		t.Fatalf("summaries[1].ID = %q, want first run %q", summaries[1].ID, firstRun.ID)
	}
	if !summaries[1].HasToolActivity {
		t.Fatal("summaries[1].HasToolActivity = false, want true")
	}
	if len(summaries[1].TaskTextPreview) > 96 {
		t.Fatalf("TaskTextPreview length = %d, want <= 96", len(summaries[1].TaskTextPreview))
	}

	events, err := store.ListRunEvents(ctx, firstRun.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Sequence != 1 || events[1].Sequence != secondEvent.Sequence {
		t.Fatalf("event sequences = [%d, %d], want [1, 2]", events[0].Sequence, events[1].Sequence)
	}
	if events[0].EventType != EventTypeToolCall || events[1].EventType != EventTypeToolResult {
		t.Fatalf("event types = [%q, %q], want tool_call/tool_result", events[0].EventType, events[1].EventType)
	}
}
