import { describe, expect, it } from "vitest";
import {
  addSpawnedNode,
  createEmptyCanvasDocument,
  patchAgentState,
  patchHandoff,
  selectCanvasNode,
} from "@/features/canvas/canvasModel";
import {
  getGraphStructureSignature,
  layoutAgentGraph,
  mapNodePositions,
} from "@/features/canvas/layoutGraph";

describe("layoutGraph", () => {
  it("creates a readable left-to-right handoff layout", () => {
    let document = createEmptyCanvasDocument();
    document = addSpawnedNode(document, {
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
    });
    document = addSpawnedNode(document, {
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
    });
    document = patchHandoff(
      document,
      {
        session_id: "session_alpha",
        run_id: "run_1",
        agent_id: "agent_planner_1",
        sequence: 3,
        replay: false,
        from_agent_id: "agent_planner_1",
        to_agent_id: "agent_coder_2",
        reason: "planner_completed",
        occurred_at: "2026-03-24T12:00:02Z",
        role: "planner",
        model: "anthropic/claude-opus-4",
      },
      "handoff_start",
    );

    expect(document.edges).toEqual([
      {
        id: "agent_planner_1->agent_coder_2",
        kind: "handoff",
        lastHandoffAt: "2026-03-24T12:00:02Z",
        pulseState: "active",
        sourceNodeId: "agent_planner_1",
        targetNodeId: "agent_coder_2",
      },
    ]);
    const laidOutNodes = layoutAgentGraph(document.nodes, document.edges);
    expect(laidOutNodes[1].position.x).toBeGreaterThan(
      laidOutNodes[0].position.x,
    );
    expect(
      laidOutNodes[1].position.x - laidOutNodes[0].position.x,
    ).toBeGreaterThanOrEqual(laidOutNodes[0].size.width);
    expect(getGraphStructureSignature(document.nodes, document.edges)).toBe(
      "agent_planner_1|agent_coder_2::agent_planner_1->agent_coder_2",
    );
  });

  it("preserves node coordinates during state-only updates", () => {
    let document = createEmptyCanvasDocument();
    document = addSpawnedNode(document, {
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
    });
    document = addSpawnedNode(document, {
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
    });

    const layoutRevisionBefore = document.layoutRevision;
    const positionsBefore = mapNodePositions(document.nodes);

    document = selectCanvasNode(document, "agent_coder_2");
    document = patchAgentState(document, {
      session_id: "session_alpha",
      run_id: "run_1",
      agent_id: "agent_coder_2",
      sequence: 3,
      replay: false,
      role: "coder",
      model: "anthropic/claude-sonnet-4-5",
      state: "thinking",
      message: "Coder is drafting the implementation.",
      occurred_at: "2026-03-24T12:00:02Z",
    });

    expect(document.layoutRevision).toBe(layoutRevisionBefore);
    expect(mapNodePositions(document.nodes)).toEqual(positionsBefore);
    expect(document.nodes[1].state).toBe("thinking");
  });

  it("keeps branched nodes from overlapping after token usage increased card height", () => {
    let document = createEmptyCanvasDocument();
    document = addSpawnedNode(document, {
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
    });
    document = addSpawnedNode(document, {
      session_id: "session_alpha",
      run_id: "run_1",
      agent_id: "agent_tester_2",
      sequence: 2,
      replay: false,
      role: "tester",
      model: "anthropic/claude-sonnet-4-5",
      label: "Tester",
      spawn_order: 2,
      occurred_at: "2026-03-24T12:00:01Z",
    });
    document = addSpawnedNode(document, {
      session_id: "session_alpha",
      run_id: "run_1",
      agent_id: "agent_coder_3",
      sequence: 3,
      replay: false,
      role: "coder",
      model: "anthropic/claude-sonnet-4-5",
      label: "Coder",
      spawn_order: 3,
      occurred_at: "2026-03-24T12:00:02Z",
    });
    document = addSpawnedNode(document, {
      session_id: "session_alpha",
      run_id: "run_1",
      agent_id: "agent_reviewer_4",
      sequence: 4,
      replay: false,
      role: "reviewer",
      model: "anthropic/claude-sonnet-4-5",
      label: "Reviewer",
      spawn_order: 4,
      occurred_at: "2026-03-24T12:00:03Z",
    });

    document = patchHandoff(
      document,
      {
        session_id: "session_alpha",
        run_id: "run_1",
        agent_id: "agent_planner_1",
        sequence: 5,
        replay: false,
        from_agent_id: "agent_planner_1",
        to_agent_id: "agent_tester_2",
        reason: "planner_completed",
        occurred_at: "2026-03-24T12:00:04Z",
        role: "planner",
        model: "anthropic/claude-opus-4",
      },
      "handoff_start",
    );
    document = patchHandoff(
      document,
      {
        session_id: "session_alpha",
        run_id: "run_1",
        agent_id: "agent_planner_1",
        sequence: 6,
        replay: false,
        from_agent_id: "agent_planner_1",
        to_agent_id: "agent_coder_3",
        reason: "planner_completed",
        occurred_at: "2026-03-24T12:00:05Z",
        role: "planner",
        model: "anthropic/claude-opus-4",
      },
      "handoff_start",
    );
    document = patchHandoff(
      document,
      {
        session_id: "session_alpha",
        run_id: "run_1",
        agent_id: "agent_tester_2",
        sequence: 7,
        replay: false,
        from_agent_id: "agent_tester_2",
        to_agent_id: "agent_reviewer_4",
        reason: "tester_completed",
        occurred_at: "2026-03-24T12:00:06Z",
        role: "tester",
        model: "anthropic/claude-sonnet-4-5",
      },
      "handoff_start",
    );
    document = patchHandoff(
      document,
      {
        session_id: "session_alpha",
        run_id: "run_1",
        agent_id: "agent_coder_3",
        sequence: 8,
        replay: false,
        from_agent_id: "agent_coder_3",
        to_agent_id: "agent_reviewer_4",
        reason: "coder_completed",
        occurred_at: "2026-03-24T12:00:07Z",
        role: "coder",
        model: "anthropic/claude-sonnet-4-5",
      },
      "handoff_start",
    );

    const testerNode = document.nodes.find(
      (node) => node.id === "agent_tester_2",
    );
    const coderNode = document.nodes.find(
      (node) => node.id === "agent_coder_3",
    );

    expect(testerNode).toBeDefined();
    expect(coderNode).toBeDefined();
    expect(
      Math.abs((testerNode?.position.y ?? 0) - (coderNode?.position.y ?? 0)),
    ).toBeGreaterThanOrEqual(testerNode?.size.height ?? 0);
  });
});