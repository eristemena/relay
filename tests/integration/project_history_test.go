package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
	git "github.com/go-git/go-git/v5"
)

func TestProjectHistory_RunHistoryQueryScopesToActiveProjectUnlessAllProjects(t *testing.T) {
	homeDir := t.TempDir()
	projectARoot := initReplayRepositoryAtHome(t, homeDir)
	projectBRoot := filepath.Join(homeDir, "relay-repo-b")
	if err := os.MkdirAll(projectBRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll(projectBRoot) error = %v", err)
	}
	if _, err := git.PlainInit(projectBRoot, false); err != nil {
		t.Fatalf("git.PlainInit(projectBRoot) error = %v", err)
	}

	paths, err := config.EnsurePaths(homeDir)
	if err != nil {
		t.Fatalf("EnsurePaths() error = %v", err)
	}
	store, err := sqlite.NewStore(paths.Database)
	if err != nil {
		t.Fatalf("sqlite.NewStore() error = %v", err)
	}

	ctx := context.Background()
	sessionA, err := store.CreateProjectSession(ctx, "relay-repo", projectARoot)
	if err != nil {
		store.Close()
		t.Fatalf("CreateProjectSession(project A) error = %v", err)
	}
	sessionB, err := store.CreateProjectSession(ctx, "relay-repo-b", projectBRoot)
	if err != nil {
		store.Close()
		t.Fatalf("CreateProjectSession(project B) error = %v", err)
	}

	runA, err := store.CreateAgentRun(ctx, sessionA.ID, "Inspect relay-repo history", sqlite.RoleReviewer, config.DefaultReviewerModel)
	if err != nil {
		store.Close()
		t.Fatalf("CreateAgentRun(project A) error = %v", err)
	}
	completedA := time.Date(2026, 4, 5, 8, 59, 0, 0, time.UTC)
	runA.State = sqlite.RunStateCompleted
	runA.CompletedAt = &completedA
	if err := store.UpdateAgentRun(ctx, runA); err != nil {
		store.Close()
		t.Fatalf("UpdateAgentRun(project A) error = %v", err)
	}

	runB, err := store.CreateAgentRun(ctx, sessionB.ID, "Inspect relay-repo-b history", sqlite.RoleReviewer, config.DefaultReviewerModel)
	if err != nil {
		store.Close()
		t.Fatalf("CreateAgentRun(project B) error = %v", err)
	}
	completedB := time.Date(2026, 4, 5, 9, 1, 0, 0, time.UTC)
	runB.State = sqlite.RunStateCompleted
	runB.CompletedAt = &completedB
	if err := store.UpdateAgentRun(ctx, runB); err != nil {
		store.Close()
		t.Fatalf("UpdateAgentRun(project B) error = %v", err)
	}

	for _, item := range []struct {
		runID     string
		sessionID string
		title     string
		goal      string
		startedAt time.Time
	}{
		{runID: runA.ID, sessionID: sessionA.ID, title: "Project A audit", goal: "Inspect project A history", startedAt: time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)},
		{runID: runB.ID, sessionID: sessionB.ID, title: "Project B audit", goal: "Inspect project B history", startedAt: time.Date(2026, 4, 5, 9, 2, 0, 0, time.UTC)},
	} {
		if err := store.UpsertRunHistoryDocument(ctx, sqlite.RunHistoryDocument{
			RunID:          item.runID,
			SessionID:      item.sessionID,
			GeneratedTitle: item.title,
			GoalText:       item.goal,
			FinalStatus:    "completed",
			AgentCount:     1,
			StartedAt:      item.startedAt,
			HasFileChanges: false,
		}); err != nil {
			store.Close()
			t.Fatalf("UpsertRunHistoryDocument(%s) error = %v", item.runID, err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close() error = %v", err)
	}

	cfg, warnings, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("config warnings = %v, want none", warnings)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = projectARoot
	cfg.LastSessionID = sessionA.ID
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	server, cancel := startServerAtHome(t, homeDir)
	defer func() {
		cancel()
		_ = server.Close()
	}()

	connection := dialWorkspace(t, server.BaseURL())
	writeMessage(t, connection, map[string]any{
		"type": "run.history.query",
		"payload": map[string]any{
			"session_id": sessionA.ID,
		},
	})
	projectScoped := readUntilType(t, connection, "run.history.result")
	projectScopedPayload := projectScoped["payload"].(map[string]any)
	if projectScopedPayload["all_projects"] != nil {
		t.Fatalf("projectScoped all_projects = %v, want omitted or false", projectScopedPayload["all_projects"])
	}
	projectScopedRuns := projectScopedPayload["runs"].([]any)
	if len(projectScopedRuns) != 1 {
		t.Fatalf("len(projectScopedRuns) = %d, want 1", len(projectScopedRuns))
	}
	projectScopedRun := projectScopedRuns[0].(map[string]any)
	if projectScopedRun["project_root"] != projectARoot {
		t.Fatalf("projectScoped project_root = %v, want %q", projectScopedRun["project_root"], projectARoot)
	}
	if projectScopedRun["project_label"] != "relay-repo" {
		t.Fatalf("projectScoped project_label = %v, want relay-repo", projectScopedRun["project_label"])
	}

	writeMessage(t, connection, map[string]any{
		"type": "run.history.query",
		"payload": map[string]any{
			"session_id":    sessionA.ID,
			"all_projects": true,
		},
	})
	allProjects := readUntilType(t, connection, "run.history.result")
	allProjectsPayload := allProjects["payload"].(map[string]any)
	if allProjectsPayload["all_projects"] != true {
		t.Fatalf("allProjectsPayload.all_projects = %v, want true", allProjectsPayload["all_projects"])
	}
	runs := allProjectsPayload["runs"].([]any)
	if len(runs) != 2 {
		t.Fatalf("len(allProjects runs) = %d, want 2", len(runs))
	}
	roots := make([]string, 0, len(runs))
	labels := make([]string, 0, len(runs))
	for _, item := range runs {
		run := item.(map[string]any)
		roots = append(roots, run["project_root"].(string))
		labels = append(labels, run["project_label"].(string))
	}
	sort.Strings(roots)
	sort.Strings(labels)
	if roots[0] != projectARoot || roots[1] != projectBRoot {
		t.Fatalf("project roots = %v, want [%q %q]", roots, projectARoot, projectBRoot)
	}
	if labels[0] != "relay-repo" || labels[1] != "relay-repo-b" {
		t.Fatalf("project labels = %v, want [relay-repo relay-repo-b]", labels)
	}
}