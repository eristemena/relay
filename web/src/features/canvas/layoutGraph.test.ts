import { describe, expect, it } from "vitest";
import {
  addNodeToCanvas,
  createEmptyCanvasDocument,
  selectCanvasNode,
  updateSelectedCanvasNodeState,
} from "@/features/canvas/canvasModel";
import {
  getGraphStructureSignature,
  mapNodePositions,
} from "@/features/canvas/layoutGraph";

describe("layoutGraph", () => {
  it("creates a readable left-to-right handoff layout", () => {
    let document = createEmptyCanvasDocument();
    document = addNodeToCanvas(document, "planner");
    document = addNodeToCanvas(document, "coder");

    expect(document.edges).toEqual([
      {
        id: "node_1->node_2",
        kind: "handoff",
        sourceNodeId: "node_1",
        targetNodeId: "node_2",
      },
    ]);
    expect(document.nodes[1].position.x).toBeGreaterThan(document.nodes[0].position.x);
    expect(
      document.nodes[1].position.x - document.nodes[0].position.x,
    ).toBeGreaterThanOrEqual(document.nodes[0].size.width);
    expect(getGraphStructureSignature(document.nodes, document.edges)).toBe(
      "node_1|node_2::node_1->node_2",
    );
  });

  it("preserves node coordinates during state-only updates", () => {
    let document = createEmptyCanvasDocument();
    document = addNodeToCanvas(document, "planner");
    document = addNodeToCanvas(document, "coder");

    const layoutRevisionBefore = document.layoutRevision;
    const positionsBefore = mapNodePositions(document.nodes);

    document = selectCanvasNode(document, "node_2");
    document = updateSelectedCanvasNodeState(document, "thinking");

    expect(document.layoutRevision).toBe(layoutRevisionBefore);
    expect(mapNodePositions(document.nodes)).toEqual(positionsBefore);
    expect(document.nodes[1].state).toBe("thinking");
  });
});