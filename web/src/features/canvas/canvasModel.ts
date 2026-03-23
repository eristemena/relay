import { layoutAgentGraph } from "@/features/canvas/layoutGraph";

export const agentCanvasRoles = [
  "planner",
  "coder",
  "reviewer",
  "tester",
  "explainer",
] as const;

export const agentCanvasStates = [
  "idle",
  "thinking",
  "executing",
  "complete",
  "error",
] as const;

export type AgentCanvasRole = (typeof agentCanvasRoles)[number];
export type AgentCanvasState = (typeof agentCanvasStates)[number];

export interface AgentNodeDetails {
  summary: string;
  currentStateLabel: string;
  incomingFrom: string[];
  outgoingTo: string[];
}

export interface AgentCanvasNodeModel {
  id: string;
  role: AgentCanvasRole;
  label: string;
  state: AgentCanvasState;
  details: AgentNodeDetails;
  position: {
    x: number;
    y: number;
  };
  size: {
    width: number;
    height: number;
  };
}

export interface AgentCanvasEdgeModel {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  kind: "handoff";
}

export interface AgentCanvasDocument {
  nodes: AgentCanvasNodeModel[];
  edges: AgentCanvasEdgeModel[];
  selectedNodeId: string | null;
  layoutRevision: number;
  validationMessage: string | null;
}

const defaultNodeSize = {
  height: 188,
  width: 336,
};

const roleCopy: Record<AgentCanvasRole, { label: string; summary: string }> = {
  planner: {
    label: "Planner",
    summary: "Frames the task, clarifies scope, and sequences the next agent handoff.",
  },
  coder: {
    label: "Coder",
    summary: "Implements the chosen slice and keeps the local canvas grounded in concrete changes.",
  },
  reviewer: {
    label: "Reviewer",
    summary: "Checks the proposed work for regressions, risks, and missing validation.",
  },
  tester: {
    label: "Tester",
    summary: "Exercises the result and validates the expected behavior under change.",
  },
  explainer: {
    label: "Explainer",
    summary: "Summarizes the workflow and translates the current graph into developer-facing guidance.",
  },
};

const stateLabels: Record<AgentCanvasState, string> = {
  idle: "Idle",
  thinking: "Thinking",
  executing: "Executing",
  complete: "Complete",
  error: "Error",
};

export function createEmptyCanvasDocument(): AgentCanvasDocument {
  return {
    nodes: [],
    edges: [],
    selectedNodeId: null,
    layoutRevision: 0,
    validationMessage: null,
  };
}

export function createAgentCanvasNode(
  role: AgentCanvasRole,
  ordinal: number,
): AgentCanvasNodeModel {
  const roleInfo = roleCopy[role];

  return {
    id: `node_${ordinal}`,
    role,
    label: `${roleInfo.label} ${ordinal}`,
    state: "idle",
    details: {
      summary: roleInfo.summary,
      currentStateLabel: stateLabels.idle,
      incomingFrom: [],
      outgoingTo: [],
    },
    position: { x: 0, y: 0 },
    size: defaultNodeSize,
  };
}

export function addNodeToCanvas(
  document: AgentCanvasDocument,
  role: AgentCanvasRole | null,
) {
  if (!role) {
    return {
      ...document,
      validationMessage: "Choose an agent role before adding a node to the local canvas.",
    };
  }

  const nextNode = createAgentCanvasNode(role, document.nodes.length + 1);
  const nextNodes = [...document.nodes, nextNode];
  const nextEdges = document.nodes.length
    ? [
        ...document.edges,
        createAgentCanvasEdge(document.nodes[document.nodes.length - 1].id, nextNode.id),
      ]
    : document.edges;

  const laidOutNodes = layoutAgentGraph(nextNodes, nextEdges);

  return {
    nodes: syncNodeDetails(laidOutNodes, nextEdges),
    edges: nextEdges,
    selectedNodeId: document.selectedNodeId,
    layoutRevision: document.layoutRevision + 1,
    validationMessage: null,
  } satisfies AgentCanvasDocument;
}

export function selectCanvasNode(
  document: AgentCanvasDocument,
  nodeId: string,
) {
  if (!document.nodes.some((node) => node.id === nodeId)) {
    return document;
  }

  return {
    ...document,
    selectedNodeId: nodeId,
    validationMessage: null,
  };
}

export function clearCanvasSelection(document: AgentCanvasDocument) {
  if (!document.selectedNodeId) {
    return document;
  }

  return {
    ...document,
    selectedNodeId: null,
    validationMessage: null,
  };
}

export function updateSelectedCanvasNodeState(
  document: AgentCanvasDocument,
  state: AgentCanvasState | null,
) {
  if (!document.selectedNodeId) {
    return {
      ...document,
      validationMessage:
        "Select a node before changing its local state.",
    };
  }

  if (!state) {
    return {
      ...document,
      validationMessage: "Choose a state before applying a local state update.",
    };
  }

  const nextNodes = document.nodes.map((node) =>
    node.id === document.selectedNodeId ? { ...node, state } : node,
  );

  return {
    ...document,
    nodes: syncNodeDetails(nextNodes, document.edges),
    validationMessage: null,
  };
}

export function getSelectedCanvasNode(document: AgentCanvasDocument) {
  if (!document.selectedNodeId) {
    return null;
  }

  return (
    document.nodes.find((node) => node.id === document.selectedNodeId) ?? null
  );
}

export function getRoleLabel(role: AgentCanvasRole) {
  return roleCopy[role].label;
}

function createAgentCanvasEdge(
  sourceNodeId: string,
  targetNodeId: string,
): AgentCanvasEdgeModel {
  return {
    id: `${sourceNodeId}->${targetNodeId}`,
    sourceNodeId,
    targetNodeId,
    kind: "handoff",
  };
}

function syncNodeDetails(
  nodes: AgentCanvasNodeModel[],
  edges: AgentCanvasEdgeModel[],
) {
  const nodeLabels = Object.fromEntries(nodes.map((node) => [node.id, node.label]));

  return nodes.map((node) => ({
    ...node,
    details: {
      summary: roleCopy[node.role].summary,
      currentStateLabel: stateLabels[node.state],
      incomingFrom: edges
        .filter((edge) => edge.targetNodeId === node.id)
        .map((edge) => nodeLabels[edge.sourceNodeId])
        .filter((label): label is string => Boolean(label)),
      outgoingTo: edges
        .filter((edge) => edge.sourceNodeId === node.id)
        .map((edge) => nodeLabels[edge.targetNodeId])
        .filter((label): label is string => Boolean(label)),
    },
  }));
}