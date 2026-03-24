import type {
  ApprovalRequestPayload,
  AgentStateChangedPayload,
  AgentSpawnedPayload,
  ErrorPayload,
  HandoffPayload,
  RunCompletePayload,
  TaskAssignedPayload,
  ToolCallPayload,
  ToolResultPayload,
  TokenPayload,
} from "@/shared/lib/workspace-protocol";
import { layoutAgentGraph } from "@/features/canvas/layoutGraph";

export const agentCanvasRoles = [
  "planner",
  "coder",
  "reviewer",
  "tester",
  "explainer",
] as const;
export const agentCanvasStates = [
  "queued",
  "assigned",
  "thinking",
  "tool_running",
  "approval_required",
  "streaming",
  "completed",
  "clarification_required",
  "errored",
  "cancelled",
  "blocked",
] as const;

export type AgentCanvasRole = (typeof agentCanvasRoles)[number];
export type AgentCanvasState = (typeof agentCanvasStates)[number];
export type AgentCanvasEdgePulseState = "idle" | "active" | "settling";
export type CanvasRunState =
  | "idle"
  | "submitting"
  | "active"
  | "completed"
  | "halted";

export interface AgentNodeDetails {
  summary: string;
  currentStateLabel: string;
  incomingFrom: string[];
  outgoingTo: string[];
  taskText: string;
  transcript: string;
  errorMessage?: string;
}

export interface AgentCanvasNodeModel {
  id: string;
  role: AgentCanvasRole;
  label: string;
  state: AgentCanvasState;
  stateRevision: number;
  lastUpdatedAt: string | null;
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
  pulseState: AgentCanvasEdgePulseState;
  lastHandoffAt: string | null;
}

export interface AgentCanvasDocument {
  nodes: AgentCanvasNodeModel[];
  edges: AgentCanvasEdgeModel[];
  selectedNodeId: string | null;
  layoutRevision: number;
  validationMessage: string | null;
  runState: CanvasRunState;
  runSummary: string;
  haltCode: string | null;
  haltMessage: string;
  haltAgentId: string | null;
  haltRole: AgentCanvasRole | null;
}

export interface SelectedCanvasNodeView {
  id: string;
  role: AgentCanvasRole;
  label: string;
  state: AgentCanvasState;
  stateRevision: number;
  details: AgentNodeDetails;
}

const defaultNodeSize = { height: 188, width: 336 };

const roleCopy: Record<AgentCanvasRole, { label: string; summary: string }> = {
  planner: {
    label: "Planner",
    summary: "Frames the task, clarifies scope, and sequences downstream work.",
  },
  coder: {
    label: "Coder",
    summary: "Drafts the implementation approach from the planner handoff.",
  },
  reviewer: {
    label: "Reviewer",
    summary:
      "Checks the combined work for regressions, risks, and missing validation.",
  },
  tester: {
    label: "Tester",
    summary: "Builds a validation strategy and failure-focused checks.",
  },
  explainer: {
    label: "Explainer",
    summary: "Closes the run with a plain-language summary for the developer.",
  },
};

const stateLabels: Record<AgentCanvasState, string> = {
  queued: "Queued",
  assigned: "Assigned",
  thinking: "Thinking",
  tool_running: "Tool running",
  approval_required: "Approval required",
  streaming: "Streaming",
  completed: "Completed",
  clarification_required: "Clarification required",
  errored: "Errored",
  cancelled: "Cancelled",
  blocked: "Blocked",
};

export function createEmptyCanvasDocument(): AgentCanvasDocument {
  return {
    nodes: [],
    edges: [],
    selectedNodeId: null,
    layoutRevision: 0,
    validationMessage: null,
    runState: "idle",
    runSummary: "",
    haltCode: null,
    haltMessage: "",
    haltAgentId: null,
    haltRole: null,
  };
}

export function addSpawnedNode(
  document: AgentCanvasDocument,
  payload: AgentSpawnedPayload,
): AgentCanvasDocument {
  if (document.nodes.some((node) => node.id === payload.agent_id)) {
    return document;
  }

  const role = payload.role as AgentCanvasRole;
  const nextNode: AgentCanvasNodeModel = {
    id: payload.agent_id,
    role,
    label: payload.label || roleCopy[role].label,
    state: "queued",
    stateRevision: 0,
    lastUpdatedAt: payload.occurred_at,
    details: {
      summary: roleCopy[role].summary,
      currentStateLabel: stateLabels.queued,
      incomingFrom: [],
      outgoingTo: [],
      taskText: "",
      transcript: "",
    },
    position: { x: 0, y: 0 },
    size: defaultNodeSize,
  };

  const laidOutNodes = layoutAgentGraph(
    [...document.nodes, nextNode],
    document.edges,
  );
  return {
    ...document,
    nodes: syncNodeDetails(laidOutNodes, document.edges),
    layoutRevision: document.layoutRevision + 1,
    runState: "active",
  };
}

export function patchAgentState(
  document: AgentCanvasDocument,
  payload: AgentStateChangedPayload,
): AgentCanvasDocument {
  return {
    ...document,
    nodes: syncNodeDetails(
      document.nodes.map((node) =>
        node.id === payload.agent_id
          ? updateNodePresentation(node, payload.occurred_at, {
              state: payload.state,
              details: {
                ...node.details,
                summary: payload.message || node.details.summary,
                currentStateLabel: stateLabels[payload.state],
              },
            })
          : node,
      ),
      document.edges,
    ),
    runState:
      payload.state === "completed" || payload.state === "errored"
        ? document.runState
        : "active",
  };
}

export function patchTaskAssigned(
  document: AgentCanvasDocument,
  payload: TaskAssignedPayload,
): AgentCanvasDocument {
  return {
    ...document,
    nodes: syncNodeDetails(
      document.nodes.map((node) =>
        node.id === payload.agent_id
          ? updateNodePresentation(node, payload.occurred_at, {
              state: "assigned",
              details: {
                ...node.details,
                taskText: payload.task_text,
                currentStateLabel: stateLabels.assigned,
              },
            })
          : node,
      ),
      document.edges,
    ),
    runState: "active",
  };
}

export function patchAgentToken(
  document: AgentCanvasDocument,
  payload: TokenPayload,
): AgentCanvasDocument {
  if (!payload.agent_id) {
    return document;
  }

  return {
    ...document,
    nodes: syncNodeDetails(
      document.nodes.map((node) =>
        node.id === payload.agent_id
          ? updateNodePresentation(node, payload.occurred_at, {
              state: node.state === "completed" ? node.state : "streaming",
              details: {
                ...node.details,
                currentStateLabel: stateLabels.streaming,
                transcript: `${node.details.transcript}${payload.text}`,
                summary: summarizeTranscript(
                  `${node.details.transcript}${payload.text}`,
                ),
              },
            })
          : node,
      ),
      document.edges,
    ),
    runState: "active",
  };
}

export function patchApprovalRequest(
  document: AgentCanvasDocument,
  payload: ApprovalRequestPayload,
): AgentCanvasDocument {
  const role = payload.role as AgentCanvasRole | undefined;
  if (!role) {
    return document;
  }

  return patchRoleState(document, role, payload.occurred_at, {
    state: "approval_required",
    details: {
      currentStateLabel: stateLabels.approval_required,
      summary: payload.message,
    },
  });
}

export function patchToolCall(
  document: AgentCanvasDocument,
  payload: ToolCallPayload,
): AgentCanvasDocument {
  return patchRoleState(
    document,
    payload.role as AgentCanvasRole,
    payload.occurred_at,
    {
      state: "tool_running",
      details: {
        currentStateLabel: stateLabels.tool_running,
        summary: `Running ${formatToolName(payload.tool_name)}.`,
      },
    },
  );
}

export function patchToolResult(
  document: AgentCanvasDocument,
  payload: ToolResultPayload,
): AgentCanvasDocument {
  const nextState =
    payload.status === "completed"
      ? "thinking"
      : payload.status === "rejected"
        ? "blocked"
        : "errored";
  const previewSummary =
    typeof payload.result_preview.summary === "string"
      ? payload.result_preview.summary
      : typeof payload.result_preview.message === "string"
        ? payload.result_preview.message
        : undefined;

  return patchRoleState(
    document,
    payload.role as AgentCanvasRole,
    payload.occurred_at,
    {
      state: nextState,
      details: {
        currentStateLabel: stateLabels[nextState],
        summary: previewSummary,
        errorMessage:
          nextState === "blocked" || nextState === "errored"
            ? previewSummary
            : undefined,
      },
    },
  );
}

export function patchHandoff(
  document: AgentCanvasDocument,
  payload: HandoffPayload,
  eventType: "handoff_start" | "handoff_complete",
): AgentCanvasDocument {
  const nextEdge: AgentCanvasEdgeModel = {
    id: `${payload.from_agent_id}->${payload.to_agent_id}`,
    sourceNodeId: payload.from_agent_id,
    targetNodeId: payload.to_agent_id,
    kind: "handoff",
    pulseState: eventType === "handoff_start" ? "active" : "settling",
    lastHandoffAt: eventType === "handoff_start" ? payload.occurred_at : null,
  };
  const edges = document.edges.some((edge) => edge.id === nextEdge.id)
    ? document.edges.map((edge) =>
        edge.id === nextEdge.id
          ? {
              ...edge,
              pulseState: nextEdge.pulseState,
              lastHandoffAt:
                eventType === "handoff_start"
                  ? payload.occurred_at
                  : edge.lastHandoffAt,
            }
          : edge,
      )
    : [...document.edges, nextEdge];
  return {
    ...document,
    edges,
    nodes: syncNodeDetails(document.nodes, edges),
  };
}

export function patchAgentError(
  document: AgentCanvasDocument,
  payload: ErrorPayload,
): AgentCanvasDocument {
  if (!payload.agent_id) {
    return document;
  }

  const nextState = isClarificationRequiredError(payload.code)
    ? "clarification_required"
    : "errored";

  return {
    ...document,
    nodes: syncNodeDetails(
      document.nodes.map((node) =>
        node.id === payload.agent_id
          ? updateNodePresentation(node, payload.occurred_at ?? null, {
              state: nextState,
              details: {
                ...node.details,
                currentStateLabel: stateLabels[nextState],
                errorMessage: payload.message,
                summary: payload.message || node.details.summary,
              },
            })
          : node,
      ),
      document.edges,
    ),
  };
}

export function patchRunComplete(
  document: AgentCanvasDocument,
  payload: RunCompletePayload,
): AgentCanvasDocument {
  return {
    ...document,
    edges: settleCanvasEdges(document.edges),
    runState: "completed",
    runSummary: payload.summary,
  };
}

export function patchRunError(
  document: AgentCanvasDocument,
  payload: ErrorPayload,
): AgentCanvasDocument {
  const haltAgentId =
    typeof payload.agent_id === "string" && payload.agent_id.trim().length > 0
      ? payload.agent_id
      : null;
  const haltRole = payload.role ? (payload.role as AgentCanvasRole) : null;
  const nodes = isClarificationRequiredError(payload.code)
    ? syncNodeDetails(
        document.nodes.map((node) => {
          const matchesHaltNode = haltAgentId
            ? node.id === haltAgentId
            : haltRole
              ? node.role === haltRole
              : false;

          if (!matchesHaltNode) {
            return node;
          }

          return updateNodePresentation(node, payload.occurred_at ?? null, {
            state: "clarification_required",
            details: {
              ...node.details,
              currentStateLabel: stateLabels.clarification_required,
              errorMessage: payload.message,
              summary: payload.message || node.details.summary,
            },
          });
        }),
        document.edges,
      )
    : document.nodes;

  return {
    ...document,
    edges: settleCanvasEdges(document.edges),
    nodes,
    runState: "halted",
    haltCode: payload.code,
    haltMessage: payload.message,
    haltAgentId,
    haltRole,
  };
}

export function selectCanvasNode(
  document: AgentCanvasDocument,
  nodeId: string,
): AgentCanvasDocument {
  if (!document.nodes.some((node) => node.id === nodeId)) {
    return document;
  }

  return {
    ...document,
    selectedNodeId: nodeId,
  };
}

export function clearCanvasSelection(
  document: AgentCanvasDocument,
): AgentCanvasDocument {
  return {
    ...document,
    selectedNodeId: null,
  };
}

export function getSelectedCanvasNode(
  document: AgentCanvasDocument,
): SelectedCanvasNodeView | null {
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

function updateNodePresentation(
  node: AgentCanvasNodeModel,
  occurredAt: string | null,
  patch: Partial<AgentCanvasNodeModel>,
): AgentCanvasNodeModel {
  return {
    ...node,
    ...patch,
    stateRevision: node.stateRevision + 1,
    lastUpdatedAt: occurredAt,
  };
}

function patchRoleState(
  document: AgentCanvasDocument,
  role: AgentCanvasRole,
  occurredAt: string | null,
  patch: {
    state: AgentCanvasState;
    details?: Partial<AgentNodeDetails>;
  },
): AgentCanvasDocument {
  return {
    ...document,
    nodes: syncNodeDetails(
      document.nodes.map((node) =>
        node.role === role
          ? updateNodePresentation(node, occurredAt, {
              state: patch.state,
              details: {
                ...node.details,
                ...patch.details,
              },
            })
          : node,
      ),
      document.edges,
    ),
    runState: "active",
  };
}

function formatToolName(toolName: string) {
  return toolName.replaceAll("_", " ");
}

function settleCanvasEdges(edges: AgentCanvasEdgeModel[]) {
  return edges.map((edge) => ({
    ...edge,
    pulseState: "idle" as const,
  }));
}

function syncNodeDetails(
  nodes: AgentCanvasNodeModel[],
  edges: AgentCanvasEdgeModel[],
) {
  const nodeLabels = Object.fromEntries(
    nodes.map((node) => [node.id, node.label]),
  );
  return nodes.map((node) => ({
    ...node,
    details: {
      ...node.details,
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

function summarizeTranscript(transcript: string) {
  const trimmed = transcript.trim();
  if (trimmed.length <= 140) {
    return trimmed;
  }
  return `${trimmed.slice(0, 137)}...`;
}

function isClarificationRequiredError(code?: string | null) {
  return Boolean(code && code.endsWith("_clarification_required"));
}
