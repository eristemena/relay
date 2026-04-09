package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eristemena/relay/internal/agents"
	"github.com/eristemena/relay/internal/app"
	"github.com/eristemena/relay/internal/config"
	"github.com/eristemena/relay/internal/storage/sqlite"
	toolspkg "github.com/eristemena/relay/internal/tools"
	git "github.com/go-git/go-git/v5"
)

func TestCodebaseAwareness_BootstrapAndBrowseExposeRepositoryState(t *testing.T) {
	parentDir := t.TempDir()
	repoDir := filepath.Join(parentDir, "relay-repo")
	plainDir := filepath.Join(parentDir, "plain-dir")

	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(repoDir) error = %v", err)
	}
	if _, err := git.PlainInit(repoDir, false); err != nil {
		t.Fatalf("git.PlainInit() error = %v", err)
	}
	if err := os.MkdirAll(plainDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(plainDir) error = %v", err)
	}

	server, _, _ := newIntegrationServer(t, app.Options{
		NoBrowser:   true,
		ProjectRoot: repoDir,
	})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})

	bootstrap := readUntilType(t, connection, "workspace.bootstrap")
	payload := bootstrap["payload"].(map[string]any)
	preferences := payload["preferences"].(map[string]any)
	connectedRepository := payload["connected_repository"].(map[string]any)

	if preferences["project_root"] != repoDir {
		t.Fatalf("preferences.project_root = %v, want %q", preferences["project_root"], repoDir)
	}
	if preferences["project_root_valid"] != true {
		t.Fatalf("preferences.project_root_valid = %v, want true", preferences["project_root_valid"])
	}
	if connectedRepository["path"] != repoDir {
		t.Fatalf("connected_repository.path = %v, want %q", connectedRepository["path"], repoDir)
	}
	if connectedRepository["status"] != "connected" {
		t.Fatalf("connected_repository.status = %v, want connected", connectedRepository["status"])
	}

	writeMessage(t, connection, map[string]any{
		"type": "repository.browse.request",
		"payload": map[string]any{
			"path": parentDir,
		},
	})

	browse := readUntilType(t, connection, "repository.browse.result")
	browsePayload := browse["payload"].(map[string]any)
	directories := browsePayload["directories"].([]any)
	if browsePayload["path"] != parentDir {
		t.Fatalf("browse.path = %v, want %q", browsePayload["path"], parentDir)
	}
	if len(directories) != 2 {
		t.Fatalf("len(directories) = %d, want 2", len(directories))
	}

	seenRepo := false
	seenPlain := false
	for _, item := range directories {
		directory := item.(map[string]any)
		switch directory["name"] {
		case "plain-dir":
			seenPlain = true
			if directory["is_git_repository"] != false {
				t.Fatalf("plain-dir is_git_repository = %v, want false", directory["is_git_repository"])
			}
		case "relay-repo":
			seenRepo = true
			if directory["is_git_repository"] != true {
				t.Fatalf("relay-repo is_git_repository = %v, want true", directory["is_git_repository"])
			}
		}
	}

	if !seenPlain || !seenRepo {
		t.Fatalf("browse directories missing expected entries: seenPlain=%v seenRepo=%v", seenPlain, seenRepo)
	}
}

func TestCodebaseAwareness_BootstrapReconnectRestoresPendingApprovalForConnectedRepository(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = repoRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateProjectSession(
		context.Background(),
		"Codebase awareness approval session",
		repoRoot,
	)
	if err != nil {
		t.Fatalf("CreateProjectSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
		repoRoot:     repoRoot,
		writePath:    "README.md",
		writeContent: "after\n",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Prepare a file diff and wait for approval",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	_ = readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	_ = readUntilStreamingType(t, connection, "tool_call")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")
	approvalPayload := approvalRequest["payload"].(map[string]any)

	if approvalPayload["request_kind"] != toolspkg.RequestKindFileWrite {
		t.Fatalf("approval request_kind = %v, want %q", approvalPayload["request_kind"], toolspkg.RequestKindFileWrite)
	}
	if approvalPayload["repository_root"] != repoRoot {
		t.Fatalf("approval repository_root = %v, want %q", approvalPayload["repository_root"], repoRoot)
	}
	if _, ok := approvalPayload["diff_preview"].(map[string]any); !ok {
		t.Fatalf("approval diff_preview = %#v, want diff preview map", approvalPayload["diff_preview"])
	}

	reconnected := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, reconnected, map[string]any{
		"type": "workspace.bootstrap.request",
		"payload": map[string]any{
			"last_session_id": session.ID,
		},
	})

	bootstrap := readUntilStreamingType(t, reconnected, "workspace.bootstrap")
	bootstrapPayload := bootstrap["payload"].(map[string]any)
	preferences := bootstrapPayload["preferences"].(map[string]any)
	connectedRepository := bootstrapPayload["connected_repository"].(map[string]any)
	pendingApprovals := bootstrapPayload["pending_approvals"].([]any)

	if preferences["project_root"] != repoRoot {
		t.Fatalf("preferences.project_root = %v, want %q", preferences["project_root"], repoRoot)
	}
	if preferences["project_root_valid"] != true {
		t.Fatalf("preferences.project_root_valid = %v, want true", preferences["project_root_valid"])
	}
	if connectedRepository["path"] != repoRoot {
		t.Fatalf("connected_repository.path = %v, want %q", connectedRepository["path"], repoRoot)
	}
	if connectedRepository["status"] != "connected" {
		t.Fatalf("connected_repository.status = %v, want connected", connectedRepository["status"])
	}
	if len(pendingApprovals) != 1 {
		t.Fatalf("len(pending_approvals) = %d, want 1", len(pendingApprovals))
	}

	restored := pendingApprovals[0].(map[string]any)
	if restored["tool_call_id"] != "call_write" {
		t.Fatalf("restored tool_call_id = %v, want call_write", restored["tool_call_id"])
	}
	if restored["request_kind"] != toolspkg.RequestKindFileWrite {
		t.Fatalf("restored request_kind = %v, want %q", restored["request_kind"], toolspkg.RequestKindFileWrite)
	}
	if restored["repository_root"] != repoRoot {
		t.Fatalf("restored repository_root = %v, want %q", restored["repository_root"], repoRoot)
	}
	diffPreview, ok := restored["diff_preview"].(map[string]any)
	if !ok {
		t.Fatalf("restored diff_preview = %#v, want diff preview map", restored["diff_preview"])
	}
	if diffPreview["target_path"] != "README.md" {
		t.Fatalf("restored target_path = %v, want README.md", diffPreview["target_path"])
	}
	if diffPreview["proposed_content"] != "after\n" {
		t.Fatalf("restored proposed_content = %#v, want updated content", diffPreview["proposed_content"])
	}
}

func TestCodebaseAwareness_RepositoryGraphEventsStreamFromLoadingToReady(t *testing.T) {
	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.MkdirAll(filepath.Join(repoRoot, "src", "lib"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "src", "index.ts"), []byte("import util from './lib/util'\nexport default util\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(index.ts) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "src", "lib", "util.ts"), []byte("export default 'ok'\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(util.ts) error = %v", err)
	}

	server, _, _ := newIntegrationServer(t, app.Options{
		NoBrowser:   true,
		ProjectRoot: repoRoot,
	})
	connection := dialWorkspace(t, server.BaseURL())

	writeMessage(t, connection, map[string]any{
		"type":    "workspace.bootstrap.request",
		"payload": map[string]any{},
	})

	_ = readUntilType(t, connection, "workspace.bootstrap")
	loading := readUntilType(t, connection, "repository_graph_status")
	loadingPayload := loading["payload"].(map[string]any)
	if loadingPayload["repository_root"] != repoRoot {
		t.Fatalf("loading repository_root = %v, want %q", loadingPayload["repository_root"], repoRoot)
	}
	if loadingPayload["status"] == "ready" {
		nodes := loadingPayload["nodes"].([]any)
		edges := loadingPayload["edges"].([]any)
		if len(nodes) != 2 {
			t.Fatalf("len(nodes) = %d, want 2", len(nodes))
		}
		if len(edges) != 1 {
			t.Fatalf("len(edges) = %d, want 1", len(edges))
		}
		edge := edges[0].(map[string]any)
		if edge["source"] != "src/index.ts" || edge["target"] != "src/lib/util.ts" {
			t.Fatalf("edge = %#v, want src/index.ts -> src/lib/util.ts", edge)
		}
		return
	}
	if loadingPayload["status"] != "loading" {
		t.Fatalf("loading status = %v, want loading or ready", loadingPayload["status"])
	}

	ready := readUntilType(t, connection, "repository_graph_status")
	readyPayload := ready["payload"].(map[string]any)
	if readyPayload["status"] != "ready" {
		t.Fatalf("ready status = %v, want ready", readyPayload["status"])
	}
	nodes := readyPayload["nodes"].([]any)
	edges := readyPayload["edges"].([]any)
	if len(nodes) != 2 {
		t.Fatalf("len(nodes) = %d, want 2", len(nodes))
	}
	if len(edges) != 1 {
		t.Fatalf("len(edges) = %d, want 1", len(edges))
	}
	edge := edges[0].(map[string]any)
	if edge["source"] != "src/index.ts" || edge["target"] != "src/lib/util.ts" {
		t.Fatalf("edge = %#v, want src/index.ts -> src/lib/util.ts", edge)
	}
}

func TestCodebaseAwareness_OpenRunReplaysRepositoryAwareAgentFileActivity(t *testing.T) {
	service, store, paths := newStreamingTestService(t)
	defer store.Close()

	repoRoot := initIntegrationRepositoryRoot(t)
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("before\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, _, err := config.Load(paths)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.OpenRouter.APIKey = "or-test-key"
	cfg.ProjectRoot = repoRoot
	if err := config.Save(paths, cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	session, err := store.CreateSession(context.Background(), "Codebase awareness replay session")
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	runner := &approvalFlowRunner{
		service:    service,
		sessionID:  session.ID,
		runIDReady: make(chan string, 1),
		profile: agents.Profile{
			Role:  sqlite.RoleCoder,
			Model: config.DefaultCoderModel,
		},
		repoRoot:     repoRoot,
		writePath:    "README.md",
		writeContent: "after\n",
	}
	service.SetRunnerFactory(func(config.Config, string) agents.Runner {
		return runner
	})

	server := newStreamingTestServer(t, service)
	connection := dialStreamingSocket(t, server.URL)

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.submit",
		"payload": map[string]any{
			"session_id": session.ID,
			"task":       "Prepare a file diff, approve it, and preserve replay activity",
		},
	})

	_ = readUntilStreamingType(t, connection, "workspace.bootstrap")
	stateChange := readUntilStreamingType(t, connection, "state_change")
	runID := stateChange["payload"].(map[string]any)["run_id"].(string)
	runner.runIDReady <- runID
	firstToolCall := readUntilStreamingType(t, connection, "tool_call")
	_ = readUntilStreamingType(t, connection, "tool_result")
	secondToolCall := readUntilStreamingType(t, connection, "tool_call")
	approvalRequest := readUntilStreamingType(t, connection, "approval_request")

	firstToolCallPayload := firstToolCall["payload"].(map[string]any)
	if firstToolCallPayload["tool_name"] != string(agents.ToolReadFile) {
		t.Fatalf("first tool_call tool_name = %v, want read_file", firstToolCallPayload["tool_name"])
	}
	if firstToolCallPayload["input_preview"].(map[string]any)["path"] != "README.md" {
		t.Fatalf("first tool_call input_preview = %#v, want README.md", firstToolCallPayload["input_preview"])
	}

	approvalPayload := approvalRequest["payload"].(map[string]any)
	if approvalPayload["repository_root"] != repoRoot {
		t.Fatalf("approval repository_root = %v, want %q", approvalPayload["repository_root"], repoRoot)
	}
	if approvalPayload["diff_preview"].(map[string]any)["target_path"] != "README.md" {
		t.Fatalf("approval diff_preview.target_path = %v, want README.md", approvalPayload["diff_preview"].(map[string]any)["target_path"])
	}

	secondToolCallPayload := secondToolCall["payload"].(map[string]any)
	if secondToolCallPayload["tool_name"] != string(agents.ToolWriteFile) {
		t.Fatalf("second tool_call tool_name = %v, want write_file", secondToolCallPayload["tool_name"])
	}
	writePreview := secondToolCallPayload["input_preview"].(map[string]any)
	if writePreview["repository_root"] != repoRoot {
		t.Fatalf("write tool input_preview.repository_root = %v, want %q", writePreview["repository_root"], repoRoot)
	}
	if writePreview["path"] != "README.md" {
		t.Fatalf("write tool input_preview.path = %v, want README.md", writePreview["path"])
	}

	writeStreamingMessage(t, connection, map[string]any{
		"type": "agent.run.approval.respond",
		"payload": map[string]any{
			"session_id":   session.ID,
			"run_id":       runID,
			"tool_call_id": "call_write",
			"decision":     "approved",
		},
	})

	approved := readUntilStreamingType(t, connection, "approval_state_changed")
	applied := readUntilStreamingType(t, connection, "approval_state_changed")
	writeResult := readUntilStreamingType(t, connection, "tool_result")
	complete := readUntilStreamingType(t, connection, "complete")

	if approved["payload"].(map[string]any)["status"] != sqlite.ApprovalStateApproved {
		t.Fatalf("approved status = %v, want approved", approved["payload"].(map[string]any)["status"])
	}
	if applied["payload"].(map[string]any)["status"] != sqlite.ApprovalStateApplied {
		t.Fatalf("applied status = %v, want applied", applied["payload"].(map[string]any)["status"])
	}
	if writeResult["payload"].(map[string]any)["status"] != "completed" {
		t.Fatalf("write result status = %v, want completed", writeResult["payload"].(map[string]any)["status"])
	}
	if complete["payload"].(map[string]any)["finish_reason"] != "stop" {
		t.Fatalf("complete finish_reason = %v, want stop", complete["payload"].(map[string]any)["finish_reason"])
	}

	replayConnection := dialStreamingSocket(t, server.URL)
	writeStreamingMessage(t, replayConnection, map[string]any{
		"type": "agent.run.open",
		"payload": map[string]any{
			"session_id": session.ID,
			"run_id":     runID,
		},
	})

	replayState := readUntilStreamingType(t, replayConnection, "state_change")
	replayReadCall := readUntilStreamingType(t, replayConnection, "tool_call")
	_ = readUntilStreamingType(t, replayConnection, "tool_result")
	replayWriteCall := readUntilStreamingType(t, replayConnection, "tool_call")
	replayApproved := readUntilStreamingType(t, replayConnection, "approval_state_changed")
	replayApplied := readUntilStreamingType(t, replayConnection, "approval_state_changed")
	replayWriteResult := readUntilStreamingType(t, replayConnection, "tool_result")
	replayComplete := readUntilStreamingType(t, replayConnection, "complete")

	replayEnvelopes := []map[string]any{replayState, replayReadCall, replayWriteCall, replayApproved, replayApplied, replayWriteResult, replayComplete}
	for _, envelope := range replayEnvelopes {
		payload := envelope["payload"].(map[string]any)
		if payload["replay"] != true {
			t.Fatalf("replayed payload = %#v, want replay=true", payload)
		}
	}

	if replayReadCall["payload"].(map[string]any)["input_preview"].(map[string]any)["path"] != "README.md" {
		t.Fatalf("replayed read input_preview = %#v, want README.md", replayReadCall["payload"].(map[string]any)["input_preview"])
	}
	replayedWritePreview := replayWriteCall["payload"].(map[string]any)["input_preview"].(map[string]any)
	if replayedWritePreview["repository_root"] != repoRoot {
		t.Fatalf("replayed write repository_root = %v, want %q", replayedWritePreview["repository_root"], repoRoot)
	}
	if replayedWritePreview["diff_preview"].(map[string]any)["target_path"] != "README.md" {
		t.Fatalf("replayed write diff_preview.target_path = %v, want README.md", replayedWritePreview["diff_preview"].(map[string]any)["target_path"])
	}
	if replayApproved["payload"].(map[string]any)["status"] != sqlite.ApprovalStateApproved {
		t.Fatalf("replayed approved status = %v, want approved", replayApproved["payload"].(map[string]any)["status"])
	}
	if replayApplied["payload"].(map[string]any)["status"] != sqlite.ApprovalStateApplied {
		t.Fatalf("replayed applied status = %v, want applied", replayApplied["payload"].(map[string]any)["status"])
	}
	if replayWriteResult["payload"].(map[string]any)["status"] != "completed" {
		t.Fatalf("replayed write result status = %v, want completed", replayWriteResult["payload"].(map[string]any)["status"])
	}
}
