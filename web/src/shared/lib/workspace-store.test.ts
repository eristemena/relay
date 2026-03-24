import { act } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import {
  resetWorkspaceStore,
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
});