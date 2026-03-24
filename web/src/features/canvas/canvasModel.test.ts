import { describe, expect, it } from "vitest";
import {
  addSpawnedNode,
  createEmptyCanvasDocument,
  patchApprovalRequest,
  patchAgentState,
  patchHandoff,
  patchRunComplete,
  patchToolResult,
} from "@/features/canvas/canvasModel";

describe("canvasModel", () => {
  it("reuses the same edge record and updates pulse state across handoffs", () => {
    let document = createEmptyCanvasDocument();

    document = addSpawnedNode(document, {
      agent_id: "agent_planner_1",
      label: "Planner",
      model: "anthropic/claude-opus-4",
      occurred_at: "2026-03-24T12:00:00Z",
      replay: false,
      role: "planner",
      run_id: "run_1",
      sequence: 1,
      session_id: "session_alpha",
      spawn_order: 1,
    });
    document = addSpawnedNode(document, {
      agent_id: "agent_coder_2",
      label: "Coder",
      model: "anthropic/claude-sonnet-4-5",
      occurred_at: "2026-03-24T12:00:01Z",
      replay: false,
      role: "coder",
      run_id: "run_1",
      sequence: 2,
      session_id: "session_alpha",
      spawn_order: 2,
    });

    const started = patchHandoff(
      document,
      {
        agent_id: "agent_planner_1",
        from_agent_id: "agent_planner_1",
        model: "anthropic/claude-opus-4",
        occurred_at: "2026-03-24T12:00:02Z",
        reason: "planner_completed",
        replay: false,
        role: "planner",
        run_id: "run_1",
        sequence: 3,
        session_id: "session_alpha",
        to_agent_id: "agent_coder_2",
      },
      "handoff_start",
    );

    expect(started.edges).toHaveLength(1);
    expect(started.edges[0]).toMatchObject({
      id: "agent_planner_1->agent_coder_2",
      pulseState: "active",
    });

    const completed = patchHandoff(
      started,
      {
        agent_id: "agent_planner_1",
        from_agent_id: "agent_planner_1",
        model: "anthropic/claude-opus-4",
        occurred_at: "2026-03-24T12:00:03Z",
        reason: "planner_completed",
        replay: false,
        role: "planner",
        run_id: "run_1",
        sequence: 4,
        session_id: "session_alpha",
        to_agent_id: "agent_coder_2",
      },
      "handoff_complete",
    );

    expect(completed.edges).toHaveLength(1);
    expect(completed.edges[0]?.pulseState).toBe("settling");
  });

  it("tracks node presentation revisions and settles edges when the run completes", () => {
    let document = createEmptyCanvasDocument();

    document = addSpawnedNode(document, {
      agent_id: "agent_planner_1",
      label: "Planner",
      model: "anthropic/claude-opus-4",
      occurred_at: "2026-03-24T12:00:00Z",
      replay: false,
      role: "planner",
      run_id: "run_1",
      sequence: 1,
      session_id: "session_alpha",
      spawn_order: 1,
    });

    document = patchAgentState(document, {
      agent_id: "agent_planner_1",
      message: "Planner is thinking.",
      model: "anthropic/claude-opus-4",
      occurred_at: "2026-03-24T12:00:01Z",
      replay: false,
      role: "planner",
      run_id: "run_1",
      sequence: 2,
      session_id: "session_alpha",
      state: "thinking",
    });

    document = patchHandoff(
      document,
      {
        agent_id: "agent_planner_1",
        from_agent_id: "agent_planner_1",
        model: "anthropic/claude-opus-4",
        occurred_at: "2026-03-24T12:00:02Z",
        reason: "planner_completed",
        replay: false,
        role: "planner",
        run_id: "run_1",
        sequence: 3,
        session_id: "session_alpha",
        to_agent_id: "agent_planner_1",
      },
      "handoff_start",
    );

    const completed = patchRunComplete(document, {
      model: "anthropic/claude-opus-4",
      occurred_at: "2026-03-24T12:00:03Z",
      replay: false,
      role: "planner",
      run_id: "run_1",
      sequence: 4,
      session_id: "session_alpha",
      summary: "Planner completed the work.",
    });

    expect(completed.nodes[0]?.stateRevision).toBe(1);
    expect(completed.edges[0]?.pulseState).toBe("idle");
  });

  it("marks a role as approval required and returns it to thinking after a completed tool result", () => {
    let document = createEmptyCanvasDocument();

    document = addSpawnedNode(document, {
      agent_id: "agent_tester_3",
      label: "Tester",
      model: "anthropic/claude-sonnet-4-5",
      occurred_at: "2026-03-24T12:00:00Z",
      replay: false,
      role: "tester",
      run_id: "run_1",
      sequence: 1,
      session_id: "session_alpha",
      spawn_order: 3,
    });

    const awaitingApproval = patchApprovalRequest(document, {
      session_id: "session_alpha",
      run_id: "run_1",
      role: "tester",
      model: "anthropic/claude-sonnet-4-5",
      tool_call_id: "call_1",
      tool_name: "write_file",
      input_preview: { path: "tests/generated/smoke_test.sh" },
      message:
        "Relay needs approval before it can write files inside the configured project root.",
      occurred_at: "2026-03-24T12:00:01Z",
    });

    expect(awaitingApproval.nodes[0]?.state).toBe("approval_required");
    expect(awaitingApproval.nodes[0]?.details.currentStateLabel).toBe(
      "Approval required",
    );

    const resumed = patchToolResult(awaitingApproval, {
      session_id: "session_alpha",
      run_id: "run_1",
      sequence: 3,
      replay: false,
      role: "tester",
      model: "anthropic/claude-sonnet-4-5",
      tool_call_id: "call_1",
      tool_name: "write_file",
      status: "completed",
      result_preview: { summary: "Wrote file content." },
      occurred_at: "2026-03-24T12:00:03Z",
    });

    expect(resumed.nodes[0]?.state).toBe("thinking");
    expect(resumed.nodes[0]?.details.summary).toBe("Wrote file content.");
  });
});