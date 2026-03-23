import dagre from "@dagrejs/dagre";
import type {
  AgentCanvasEdgeModel,
  AgentCanvasNodeModel,
} from "@/features/canvas/canvasModel";

const graphDefaults = {
  marginx: 32,
  marginy: 32,
  nodesep: 72,
  rankdir: "LR",
  ranksep: 132,
};

export function getGraphStructureSignature(
  nodes: AgentCanvasNodeModel[],
  edges: AgentCanvasEdgeModel[],
) {
  const nodeSignature = nodes.map((node) => node.id).join("|");
  const edgeSignature = edges
    .map((edge) => `${edge.sourceNodeId}->${edge.targetNodeId}`)
    .join("|");

  return `${nodeSignature}::${edgeSignature}`;
}

export function layoutAgentGraph(
  nodes: AgentCanvasNodeModel[],
  edges: AgentCanvasEdgeModel[],
) {
  if (nodes.length === 0) {
    return nodes;
  }

  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph(graphDefaults);

  for (const node of nodes) {
    graph.setNode(node.id, {
      height: node.size.height,
      width: node.size.width,
    });
  }

  for (const edge of edges) {
    graph.setEdge(edge.sourceNodeId, edge.targetNodeId);
  }

  dagre.layout(graph);

  return nodes.map((node) => {
    const layoutNode = graph.node(node.id);
    if (!layoutNode) {
      return node;
    }

    return {
      ...node,
      position: {
        x: Math.round(layoutNode.x - node.size.width / 2),
        y: Math.round(layoutNode.y - node.size.height / 2),
      },
    };
  });
}

export function mapNodePositions(nodes: AgentCanvasNodeModel[]) {
  return Object.fromEntries(
    nodes.map((node) => [node.id, { ...node.position }]),
  );
}