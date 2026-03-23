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
});