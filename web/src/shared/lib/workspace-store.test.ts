import { act } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  resetWorkspaceStore,
  selectWorkspaceCanvasNode,
  toggleWorkspaceRepositoryTreePath,
  workspaceStore,
} from "@/shared/lib/workspace-store";
import { buildWorkspaceSnapshot, primeWorkspaceStore } from "@/shared/lib/test-helpers";

describe("workspaceStore", () => {
  it("returns to thinking after approval-required receives a tool result", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_1",
        run_summaries: [
          {
            id: "run_1",
            task_text_preview: "Update the README",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "tool_running",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: true,
          },
        ],
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "approval_request",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          tool_call_id: "call_1",
          tool_name: "write_file",
          input_preview: { path: "README.md" },
          message:
            "Relay needs approval before it can write files inside the configured project root.",
          occurred_at: "2026-03-23T12:00:01Z",
        },
      } as never);
    });

    act(() => {
      workspaceStore.handleEnvelope({
        type: "tool_result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          sequence: 3,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          tool_call_id: "call_1",
          tool_name: "write_file",
          status: "completed",
          result_preview: { summary: "Wrote file content." },
          occurred_at: "2026-03-23T12:00:03Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.pendingApprovals).toEqual({});
    expect(state.runSummaries[0]?.state).toBe("thinking");

    resetWorkspaceStore();
  });

  it("updates orchestration node state for approval and tool result events", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(buildWorkspaceSnapshot({ active_run_id: "run_9" }));

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_9",
          sequence: 1,
          replay: false,
          role: "tester",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_tester_3",
          label: "Tester",
          spawn_order: 3,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
    });

    act(() => {
      workspaceStore.handleEnvelope({
        type: "approval_request",
        payload: {
          session_id: "session_alpha",
          run_id: "run_9",
          role: "tester",
          model: "anthropic/claude-sonnet-4-5",
          tool_call_id: "call_1",
          tool_name: "write_file",
          input_preview: { path: "tests/generated/smoke_test.sh" },
          message:
            "Relay needs approval before it can write files inside the configured project root.",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
    });

    let state = workspaceStore.getSnapshot();
    expect(state.orchestrationDocuments.run_9?.nodes[0]?.state).toBe(
      "approval_required",
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "tool_result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_9",
          sequence: 3,
          replay: false,
          role: "tester",
          model: "anthropic/claude-sonnet-4-5",
          tool_call_id: "call_1",
          tool_name: "write_file",
          status: "completed",
          result_preview: { summary: "Wrote file content." },
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    state = workspaceStore.getSnapshot();
    expect(state.orchestrationDocuments.run_9?.nodes[0]?.state).toBe(
      "thinking",
    );

    resetWorkspaceStore();
  });

  it("derives node file activity from replayed tool and approval events", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(buildWorkspaceSnapshot({ active_run_id: "run_10" }));

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_10",
          sequence: 1,
          replay: true,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_coder_2",
          label: "Coder",
          spawn_order: 2,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "tool_call",
        payload: {
          session_id: "session_alpha",
          run_id: "run_10",
          sequence: 2,
          replay: true,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          tool_call_id: "call_read",
          tool_name: "read_file",
          input_preview: { path: "internal/agents/coder.go" },
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "approval_request",
        payload: {
          session_id: "session_alpha",
          run_id: "run_10",
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          tool_call_id: "call_write",
          tool_name: "write_file",
          request_kind: "file_write",
          status: "proposed",
          repository_root: "/tmp/project",
          input_preview: { path: "README.md" },
          diff_preview: {
            target_path: "README.md",
            original_content: "before\n",
            proposed_content: "after\n",
            base_content_hash: "sha256:abc",
          },
          message: "Relay needs approval before it can write files.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "approval_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_10",
          sequence: 4,
          replay: true,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          tool_call_id: "call_write",
          tool_name: "write_file",
          status: "applied",
          message: "Relay applied the approved change.",
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(
      state.orchestrationDocuments.run_10?.nodes[0]?.details.readPaths,
    ).toEqual(["internal/agents/coder.go"]);
    expect(
      state.orchestrationDocuments.run_10?.nodes[0]?.details.proposedChanges,
    ).toEqual([
      {
        path: "README.md",
        toolCallId: "call_write",
        approvalState: "applied",
      },
    ]);

    resetWorkspaceStore();
  });

  it("deduplicates repeated run summaries from snapshot updates", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          active_run_id: "run_1",
          run_summaries: [
            {
              id: "run_1",
              task_text_preview: "Inspect relay startup",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:00:00Z",
              has_tool_activity: true,
            },
            {
              id: "run_1",
              task_text_preview: "Inspect relay startup",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:00:00Z",
              has_tool_activity: true,
            },
          ],
        }),
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.runSummaries).toHaveLength(1);
    expect(state.runSummaries[0]?.id).toBe("run_1");

    resetWorkspaceStore();
  });

  it("clears project-scoped caches when the active project changes", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_project_root: "/tmp/relay-a",
        known_projects: [
          {
            project_root: "/tmp/relay-a",
            label: "relay-a",
            is_active: true,
            is_available: true,
            last_opened_at: "2026-03-23T12:15:00Z",
          },
          {
            project_root: "/tmp/relay-b",
            label: "relay-b",
            is_active: false,
            is_available: true,
            last_opened_at: "2026-03-23T12:20:00Z",
          },
        ],
        active_run_id: "run_a",
        run_summaries: [
          {
            id: "run_a",
            task_text_preview: "Inspect relay-a",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "thinking",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: true,
          },
        ],
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_a",
          sequence: 1,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_coder_1",
          label: "Coder",
          spawn_order: 1,
          occurred_at: "2026-03-23T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run.history.result",
        payload: {
          session_id: "session_alpha",
          all_projects: true,
          query: "relay",
          runs: [
            {
              id: "run_history_a",
              task_text_preview: "Inspect relay-a history",
              project_root: "/tmp/relay-a",
              project_label: "relay-a",
              role: "reviewer",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:02:00Z",
              has_tool_activity: true,
            },
          ],
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "repository.tree.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_a",
          repository_root: "/tmp/relay-a",
          status: "ready",
          paths: ["README.md"],
          touched_files: [],
        },
      } as never);
    });

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          active_session_id: "session_beta",
          active_project_root: "/tmp/relay-b",
          known_projects: [
            {
              project_root: "/tmp/relay-a",
              label: "relay-a",
              is_active: false,
              is_available: true,
              last_opened_at: "2026-03-23T12:15:00Z",
            },
            {
              project_root: "/tmp/relay-b",
              label: "relay-b",
              is_active: true,
              is_available: true,
              last_opened_at: "2026-03-23T12:20:00Z",
            },
          ],
          sessions: [
            {
              id: "session_beta",
              display_name: "relay-b",
              created_at: "2026-03-23T12:10:00Z",
              last_opened_at: "2026-03-23T12:20:00Z",
              status: "active",
              has_activity: false,
            },
          ],
          preferences: {
            ...buildWorkspaceSnapshot().preferences,
            project_root: "/tmp/relay-b",
            project_root_configured: true,
            project_root_valid: true,
          },
          run_summaries: [],
        }),
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.activeProjectRoot).toBe("/tmp/relay-b");
    expect(state.knownProjects).toHaveLength(2);
    expect(state.runEvents).toEqual({});
    expect(state.orchestrationDocuments).toEqual({});
    expect(state.runHistoryResults).toEqual([]);
    expect(state.runHistoryQuery?.all_projects).toBe(true);
    expect(state.repositoryTree.status).toBe("idle");
    expect(state.repositoryTree.paths).toEqual([]);

    resetWorkspaceStore();
  });

  it("stores project metadata for all-project history results", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "run.history.result",
        payload: {
          session_id: "session_alpha",
          all_projects: true,
          runs: [
            {
              id: "run_history_a",
              task_text_preview: "Inspect relay-a history",
              project_root: "/tmp/relay-a",
              project_label: "relay-a",
              role: "reviewer",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:02:00Z",
              has_tool_activity: true,
            },
          ],
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.runHistoryQuery?.all_projects).toBe(true);
    expect(state.runHistoryResults).toEqual([
      expect.objectContaining({
        id: "run_history_a",
        project_root: "/tmp/relay-a",
        project_label: "relay-a",
      }),
    ]);

    resetWorkspaceStore();
  });

  it("rehydrates pending approvals from bootstrap snapshots", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          active_run_id: "run_1",
          pending_approvals: [
            {
              session_id: "session_alpha",
              run_id: "run_1",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              tool_call_id: "call_1",
              tool_name: "write_file",
              input_preview: { path: "README.md" },
              message:
                "Relay needs approval before it can write files inside the configured project root.",
              occurred_at: "2026-03-23T12:00:01Z",
            },
          ],
        }),
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.pendingApprovals.call_1).toMatchObject({
      runId: "run_1",
      toolName: "write_file",
      message:
        "Relay needs approval before it can write files inside the configured project root.",
    });

    resetWorkspaceStore();
  });

  it("does not auto-select the first saved run on bootstrap without an active run", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          run_summaries: [
            {
              id: "run_saved_1",
              task_text_preview: "Inspect saved startup run",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:00:00Z",
              has_tool_activity: true,
            },
          ],
        }),
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.activeRunId).toBe("");
    expect(state.selectedRunId).toBe("");

    resetWorkspaceStore();
  });

  it("stores history results, detail payloads, and resets replay artifacts when seeking", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({ active_run_id: "run_history_1" }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          sequence: 1,
          replay: true,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_coder_1",
          text: "preserved transcript",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "approval_request",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          tool_call_id: "call_replay_1",
          tool_name: "write_file",
          input_preview: { path: "README.md" },
          message: "Historical approval",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run.history.result",
        payload: {
          session_id: "session_alpha",
          query: "approval",
          runs: [
            {
              id: "run_history_1",
              generated_title: "Review approval flow",
              task_text_preview: "Audit approval review flow",
              role: "reviewer",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-24T12:00:00Z",
              completed_at: "2026-03-24T12:02:00Z",
              has_tool_activity: true,
              agent_count: 3,
              final_status: "completed",
              has_file_changes: true,
            },
          ],
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run.history.details.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          generated_title: "Review approval flow",
          final_status: "completed",
          agent_count: 3,
          change_records: [
            {
              tool_call_id: "call_replay_1",
              path: "README.md",
              base_content_hash: "sha256:abc",
              approval_state: "applied",
              occurred_at: "2026-03-24T12:00:02Z",
            },
          ],
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent.run.replay.state",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "seeking",
          cursor_ms: 2000,
          duration_ms: 60000,
          speed: 1,
          selected_timestamp: "2026-03-24T12:00:02Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run.history.export.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "completed",
          export_path: "/Users/example/.relay/exports/review-approval-flow.md",
          generated_at: "2026-03-24T12:03:00Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.runHistoryQuery?.query).toBe("approval");
    expect(state.runHistoryResults).toHaveLength(1);
    expect(state.runHistoryDetails.run_history_1?.change_records).toHaveLength(
      1,
    );
    expect(state.replayStateByRunId.run_history_1?.status).toBe("seeking");
    expect(state.exportStateByRunId.run_history_1?.status).toBe("completed");
    expect(state.runEvents.run_history_1 ?? []).toEqual([]);
    expect(state.runTranscripts.run_history_1 ?? "").toBe("");
    expect(state.pendingApprovals.call_replay_1).toBeUndefined();
    expect(state.orchestrationDocuments.run_history_1?.nodes ?? []).toEqual([]);

    resetWorkspaceStore();
  });

  it("tracks replay speed updates and export error state per historical run", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({ active_run_id: "run_history_2" }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent.run.replay.state",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_2",
          status: "playing",
          cursor_ms: 3200,
          duration_ms: 8000,
          speed: 5,
          selected_timestamp: "2026-03-24T12:00:04Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run.history.export.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_2",
          status: "error",
          error: "unable to write export",
          generated_at: "2026-03-24T12:05:00Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.replayStateByRunId.run_history_2).toMatchObject({
      status: "playing",
      speed: 5,
      cursor_ms: 3200,
    });
    expect(state.exportStateByRunId.run_history_2).toMatchObject({
      status: "error",
      error: "unable to write export",
    });

    resetWorkspaceStore();
  });

  it("preserves diff and command approval previews from bootstrap snapshots", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          active_run_id: "run_1",
          pending_approvals: [
            {
              session_id: "session_alpha",
              run_id: "run_1",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              tool_call_id: "call_write",
              tool_name: "write_file",
              request_kind: "file_write",
              status: "proposed",
              repository_root: "/tmp/project",
              input_preview: { path: "README.md" },
              diff_preview: {
                target_path: "README.md",
                original_content: "before\n",
                proposed_content: "after\n",
                base_content_hash: "sha256:abc",
              },
              message:
                "Relay needs approval before it can write files inside the configured project root.",
              occurred_at: "2026-03-23T12:00:01Z",
            },
            {
              session_id: "session_alpha",
              run_id: "run_1",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              tool_call_id: "call_command",
              tool_name: "run_command",
              request_kind: "command",
              status: "proposed",
              repository_root: "/tmp/project",
              input_preview: { command: "go", args: ["test", "./..."] },
              command_preview: {
                command: "go",
                args: ["test", "./..."],
                effective_dir: "/tmp/project",
              },
              message:
                "Relay needs approval before it can run a shell command from the configured project root.",
              occurred_at: "2026-03-23T12:00:02Z",
            },
          ],
        }),
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.pendingApprovals.call_write).toMatchObject({
      requestKind: "file_write",
      repositoryRoot: "/tmp/project",
      diffPreview: {
        targetPath: "README.md",
        originalContent: "before\n",
        proposedContent: "after\n",
        baseContentHash: "sha256:abc",
      },
    });
    expect(state.pendingApprovals.call_command).toMatchObject({
      requestKind: "command",
      commandPreview: {
        command: "go",
        args: ["test", "./..."],
        effectiveDir: "/tmp/project",
      },
    });

    resetWorkspaceStore();
  });

  it("hydrates connected repository state from the bootstrap snapshot", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          preferences: {
            preferred_port: 4747,
            appearance_variant: "midnight",
            has_credentials: true,
            openrouter_configured: true,
            project_root: "/tmp/project",
            project_root_configured: true,
            project_root_valid: true,
            agent_models: {
              planner: "anthropic/claude-opus-4",
              coder: "anthropic/claude-sonnet-4-5",
              reviewer: "anthropic/claude-sonnet-4-5",
              tester: "deepseek/deepseek-chat",
              explainer: "google/gemini-2.0-flash-001",
            },
            open_browser_on_start: true,
          },
        }),
      } as never);
    });

    expect(workspaceStore.getSnapshot().connectedRepository).toMatchObject({
      path: "/tmp/project",
      status: "connected",
    });

    resetWorkspaceStore();
  });

  it("derives repository graph loading state from a connected repository and applies graph status events", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          preferences: {
            preferred_port: 4747,
            appearance_variant: "midnight",
            has_credentials: true,
            openrouter_configured: true,
            project_root: "/tmp/project",
            project_root_configured: true,
            project_root_valid: true,
            agent_models: {
              planner: "anthropic/claude-opus-4",
              coder: "anthropic/claude-sonnet-4-5",
              reviewer: "anthropic/claude-sonnet-4-5",
              tester: "deepseek/deepseek-chat",
              explainer: "google/gemini-2.0-flash-001",
            },
            open_browser_on_start: true,
          },
        }),
      } as never);
    });

    expect(workspaceStore.getSnapshot().repositoryGraph.status).toBe("loading");

    act(() => {
      workspaceStore.handleEnvelope({
        type: "repository_graph_status",
        payload: {
          repository_root: "/tmp/project",
          status: "ready",
          message: "Repository graph ready.",
          nodes: [
            { id: "src/index.ts", label: "src/index.ts", kind: "file" },
            { id: "src/lib/util.ts", label: "src/lib/util.ts", kind: "file" },
          ],
          edges: [
            {
              id: "src/index.ts->src/lib/util.ts",
              source: "src/index.ts",
              target: "src/lib/util.ts",
            },
          ],
        },
      } as never);
    });

    expect(workspaceStore.getSnapshot().repositoryGraph).toMatchObject({
      status: "ready",
      nodes: [
        { id: "src/index.ts", label: "src/index.ts", kind: "file" },
        { id: "src/lib/util.ts", label: "src/lib/util.ts", kind: "file" },
      ],
      edges: [
        {
          id: "src/index.ts->src/lib/util.ts",
          source: "src/index.ts",
          target: "src/lib/util.ts",
        },
      ],
    });
    expect(
      workspaceStore.getSnapshot().repositoryGraph.errorMessage,
    ).toBeUndefined();

    resetWorkspaceStore();
  });

  it("clears pending approvals when approval lifecycle events arrive", () => {
    resetWorkspaceStore();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          active_run_id: "run_1",
          run_summaries: [
            {
              id: "run_1",
              task_text_preview: "Update the README",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              state: "approval_required",
              started_at: "2026-03-23T12:00:00Z",
              has_tool_activity: true,
            },
          ],
          pending_approvals: [
            {
              session_id: "session_alpha",
              run_id: "run_1",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              tool_call_id: "call_1",
              tool_name: "write_file",
              input_preview: { path: "README.md" },
              message:
                "Relay needs approval before it can write files inside the configured project root.",
              occurred_at: "2026-03-23T12:00:01Z",
            },
          ],
        }),
      } as never);
    });

    act(() => {
      workspaceStore.handleEnvelope({
        type: "approval_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          tool_call_id: "call_1",
          status: "approved",
          message: "Tool approved. Relay is resuming the run.",
          occurred_at: "2026-03-23T12:00:02Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.pendingApprovals).toEqual({});
    expect(state.status).toMatchObject({
      phase: "approval-approved",
      message: "Tool approved. Relay is resuming the run.",
    });

    resetWorkspaceStore();
  });

  it("deduplicates replayed run events when the same saved run is opened twice", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        run_summaries: [
          {
            id: "run_15",
            task_text_preview: "Replay the saved run",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: true,
          },
        ],
      }),
    );

    const replayEvent = {
      type: "state_change",
      payload: {
        session_id: "session_alpha",
        run_id: "run_15",
        sequence: 15,
        replay: true,
        role: "coder",
        model: "anthropic/claude-sonnet-4-5",
        state: "thinking",
        message: "Replay restored.",
        occurred_at: "2026-03-23T12:00:15Z",
      },
    } as const;

    act(() => {
      workspaceStore.handleEnvelope(replayEvent as never);
      workspaceStore.handleEnvelope(replayEvent as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.runEvents.run_15).toHaveLength(1);
    expect(state.runEvents.run_15?.[0]?.payload.sequence).toBe(15);

    resetWorkspaceStore();
  });

  it("caches transcript text and does not duplicate replayed token chunks", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        run_summaries: [
          {
            id: "run_tokens",
            task_text_preview: "Stream the transcript",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "thinking",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: false,
          },
        ],
      }),
    );

    const tokenEnvelope = {
      type: "token",
      payload: {
        session_id: "session_alpha",
        run_id: "run_tokens",
        sequence: 2,
        replay: true,
        role: "coder",
        model: "anthropic/claude-sonnet-4-5",
        text: "alpha",
        first_token_latency_ms: 12,
        occurred_at: "2026-03-23T12:00:01Z",
      },
    } as const;

    act(() => {
      workspaceStore.handleEnvelope(tokenEnvelope as never);
      workspaceStore.handleEnvelope(tokenEnvelope as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          ...tokenEnvelope.payload,
          sequence: 3,
          text: "beta",
          occurred_at: "2026-03-23T12:00:02Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.runEvents.run_tokens).toHaveLength(2);
    expect(state.runTranscripts.run_tokens).toBe("alphabeta");

    resetWorkspaceStore();
  });

  it("derives handoff pulse state from live events without backend-owned motion fields", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_1",
        run_summaries: [
          {
            id: "run_1",
            task_text_preview: "Inspect relay startup",
            role: "planner",
            model: "anthropic/claude-opus-4",
            state: "active",
            started_at: "2026-03-24T12:00:00Z",
            has_tool_activity: false,
          },
        ],
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 2,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          label: "Coder",
          spawn_order: 2,
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "handoff_start",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 3,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          from_agent_id: "agent_planner_1",
          to_agent_id: "agent_coder_2",
          reason: "planner_completed",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    let document = workspaceStore.getSnapshot().orchestrationDocuments.run_1;
    expect(document?.edges[0]?.pulseState).toBe("active");

    act(() => {
      workspaceStore.handleEnvelope({
        type: "handoff_complete",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 4,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          from_agent_id: "agent_planner_1",
          to_agent_id: "agent_coder_2",
          reason: "planner_completed",
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    document = workspaceStore.getSnapshot().orchestrationDocuments.run_1;
    expect(document?.edges).toHaveLength(1);
    expect(document?.edges[0]?.pulseState).toBe("settling");

    resetWorkspaceStore();
  });

  it("applies replayed token usage fields to orchestration node details", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({ active_run_id: "run_token_replay" }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_token_replay",
          sequence: 1,
          replay: true,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_reviewer_4",
          label: "Reviewer",
          spawn_order: 4,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);

      workspaceStore.handleEnvelope({
        type: "run_complete",
        payload: {
          session_id: "session_alpha",
          run_id: "run_token_replay",
          sequence: 2,
          replay: true,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_reviewer_4",
          summary: "Reviewer completed the audit.",
          tokens_used: 800,
          context_limit: 1000,
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
    });

    expect(
      workspaceStore.getSnapshot().orchestrationDocuments.run_token_replay
        ?.nodes[0]?.details.tokenUsage,
    ).toMatchObject({
      tokensUsed: 800,
      contextLimit: 1000,
      usagePercent: 0.8,
      tone: "warning",
      summary: "800 / 1,000",
    });

    resetWorkspaceStore();
  });

  it("retains selected agent filtering context after the active run completes", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_tree_1",
        preferences: {
          ...buildWorkspaceSnapshot().preferences,
          project_root: "/tmp/relay",
          project_root_configured: true,
          project_root_valid: true,
        },
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_1",
          sequence: 1,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_coder_1",
          label: "Coder",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "repository.tree.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_1",
          repository_root: "/tmp/relay",
          status: "ready",
          message: "Repository tree is ready.",
          paths: ["README.md"],
          touched_files: [
            {
              run_id: "run_tree_1",
              agent_id: "agent_coder_1",
              file_path: "README.md",
              touch_type: "read",
            },
          ],
        },
      } as never);
      selectWorkspaceCanvasNode("run_tree_1", "agent_coder_1");
      workspaceStore.handleEnvelope({
        type: "run_complete",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_1",
          sequence: 2,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          agent_id: "agent_coder_1",
          summary: "Run complete.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.activeRunId).toBe("");
    expect(state.selectedRunId).toBe("run_tree_1");
    expect(state.orchestrationDocuments.run_tree_1?.selectedNodeId).toBe(
      "agent_coder_1",
    );
    expect(state.repositoryTree.touchedFiles).toEqual([
      {
        run_id: "run_tree_1",
        agent_id: "agent_coder_1",
        file_path: "README.md",
        touch_type: "read",
      },
    ]);

    resetWorkspaceStore();
  });

  it("stores repository tree failures without clearing the last loaded snapshot", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_tree_error",
        preferences: {
          ...buildWorkspaceSnapshot().preferences,
          project_root: "/tmp/relay",
          project_root_configured: true,
          project_root_valid: true,
        },
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "repository.tree.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_error",
          repository_root: "/tmp/relay",
          status: "ready",
          message: "Repository tree is ready.",
          paths: ["README.md"],
          touched_files: [],
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "error",
        payload: {
          code: "repository_tree_failed",
          message: "Relay could not load the connected repository tree.",
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.repositoryTree.status).toBe("error");
    expect(state.repositoryTree.message).toBe(
      "Relay could not load the connected repository tree.",
    );
    expect(state.repositoryTree.paths).toEqual(["README.md"]);

    resetWorkspaceStore();
  });

  it("deduplicates reconnect-hydrated touched files when repeated live events arrive", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_tree_reconnect",
        preferences: {
          ...buildWorkspaceSnapshot().preferences,
          project_root: "/tmp/relay",
          project_root_configured: true,
          project_root_valid: true,
        },
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "repository.tree.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_reconnect",
          repository_root: "/tmp/relay",
          status: "ready",
          message: "Repository tree is ready.",
          paths: ["README.md"],
          touched_files: [
            {
              run_id: "run_tree_reconnect",
              agent_id: "agent_coder_1",
              file_path: "README.md",
              touch_type: "read",
            },
          ],
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "file_touched",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_reconnect",
          agent_id: "agent_coder_1",
          role: "coder",
          file_path: "README.md",
          touch_type: "read",
          replay: false,
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "file_touched",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_reconnect",
          agent_id: "agent_reviewer_1",
          role: "reviewer",
          file_path: "README.md",
          touch_type: "proposed",
          replay: false,
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.repositoryTree.touchedFiles).toEqual([
      {
        run_id: "run_tree_reconnect",
        agent_id: "agent_coder_1",
        file_path: "README.md",
        touch_type: "read",
      },
      {
        run_id: "run_tree_reconnect",
        agent_id: "agent_reviewer_1",
        file_path: "README.md",
        touch_type: "proposed",
      },
    ]);

    resetWorkspaceStore();
  });

  it("preserves expanded repository folders across refreshed tree snapshots", () => {
    resetWorkspaceStore();
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_tree_refresh",
        preferences: {
          ...buildWorkspaceSnapshot().preferences,
          project_root: "/tmp/relay",
          project_root_configured: true,
          project_root_valid: true,
        },
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "repository.tree.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_refresh",
          repository_root: "/tmp/relay",
          status: "ready",
          message: "Repository tree is ready.",
          paths: ["docs", "docs/guides", "docs/guides/setup.md"],
          touched_files: [],
        },
      } as never);
      toggleWorkspaceRepositoryTreePath("docs/guides");
      workspaceStore.handleEnvelope({
        type: "repository.tree.result",
        payload: {
          session_id: "session_alpha",
          run_id: "run_tree_refresh",
          repository_root: "/tmp/relay",
          status: "ready",
          message: "Repository tree is ready.",
          paths: ["docs", "docs/guides", "docs/guides/setup.md"],
          touched_files: [
            {
              run_id: "run_tree_refresh",
              agent_id: "agent_coder_1",
              file_path: "docs/guides/setup.md",
              touch_type: "read",
            },
          ],
        },
      } as never);
    });

    const state = workspaceStore.getSnapshot();
    expect(state.repositoryTree.expandedPaths).toEqual(["docs/guides"]);
    expect(state.repositoryTree.touchedFiles).toEqual([
      {
        run_id: "run_tree_refresh",
        agent_id: "agent_coder_1",
        file_path: "docs/guides/setup.md",
        touch_type: "read",
      },
    ]);

    resetWorkspaceStore();
  });
});