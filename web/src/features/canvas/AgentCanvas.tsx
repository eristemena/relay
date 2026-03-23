"use client";

import {
  Background,
  Controls,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
  type Edge,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { useEffect, useMemo, useReducer } from "react";
import { CanvasEmptyState } from "@/features/canvas/CanvasEmptyState";
import {
  AgentCanvasNode,
  type AgentCanvasFlowNode,
  type AgentCanvasNodeData,
} from "@/features/canvas/AgentCanvasNode";
import { AgentCanvasToolbar } from "@/features/canvas/AgentCanvasToolbar";
import { AgentNodeDetailPanel } from "@/features/canvas/AgentNodeDetailPanel";
import {
  addNodeToCanvas,
  clearCanvasSelection,
  createEmptyCanvasDocument,
  getRoleLabel,
  getSelectedCanvasNode,
  selectCanvasNode,
  updateSelectedCanvasNodeState,
  type AgentCanvasDocument,
} from "@/features/canvas/canvasModel";

interface AgentCanvasProps {
  sessionLabel: string;
}

type AgentCanvasAction =
  | { type: "add-node"; role: Parameters<typeof addNodeToCanvas>[1] }
  | { type: "apply-state"; state: Parameters<typeof updateSelectedCanvasNodeState>[1] }
  | { type: "select-node"; nodeId: string }
  | { type: "clear-selection" };

const nodeTypes = {
  agentCanvasNode: AgentCanvasNode,
};

function reducer(
  document: AgentCanvasDocument,
  action: AgentCanvasAction,
): AgentCanvasDocument {
  switch (action.type) {
    case "add-node":
      return addNodeToCanvas(document, action.role);
    case "apply-state":
      return updateSelectedCanvasNodeState(document, action.state);
    case "select-node":
      return selectCanvasNode(document, action.nodeId);
    case "clear-selection":
      return clearCanvasSelection(document);
    default:
      return document;
  }
}

export function AgentCanvas({ sessionLabel }: AgentCanvasProps) {
  return (
    <ReactFlowProvider>
      <AgentCanvasSurface sessionLabel={sessionLabel} />
    </ReactFlowProvider>
  );
}

function AgentCanvasSurface({ sessionLabel }: AgentCanvasProps) {
  const [document, dispatch] = useReducer(reducer, undefined, createEmptyCanvasDocument);
  const selectedNode = getSelectedCanvasNode(document);
  const flowNodes = useMemo<AgentCanvasFlowNode[]>(
    () =>
      document.nodes.map((node) => ({
        id: node.id,
        type: "agentCanvasNode",
        position: node.position,
        style: {
          width: `${node.size.width}px`,
        },
        data: {
          label: node.label,
          role: node.role,
          roleLabel: getRoleLabel(node.role),
          state: node.state,
          summary: node.details.summary,
        },
        selected: node.id === document.selectedNodeId,
        draggable: false,
        selectable: true,
      })),
    [document.nodes, document.selectedNodeId],
  );
  const flowEdges = useMemo<Edge[]>(
    () =>
      document.edges.map((edge) => ({
        id: edge.id,
        source: edge.sourceNodeId,
        target: edge.targetNodeId,
        animated: false,
        selectable: false,
      })),
    [document.edges],
  );

  return (
    <section
      aria-labelledby="agent-canvas-heading"
      className="panel-surface noise-overlay relative overflow-hidden rounded-[2rem] p-5 shadow-idle"
    >
      <div className="relative z-10 space-y-5">
        <div className="max-w-3xl">
          <p className="eyebrow">Isolated canvas</p>
          <h2
            className="mt-2 font-display text-3xl text-text"
            id="agent-canvas-heading"
          >
            {sessionLabel} agent graph
          </h2>
          <p className="mt-3 text-sm leading-6 text-text-muted">
            Build a local-only agent workflow, inspect each node, and tune states without waiting for backend events or live execution.
          </p>
        </div>

        <AgentCanvasToolbar
          onAddNode={(role) => dispatch({ type: "add-node", role })}
          onApplyState={(state) => dispatch({ type: "apply-state", state })}
          selectedNode={selectedNode}
        />

        {document.validationMessage ? (
          <div
            className="rounded-[1.25rem] border border-[var(--color-error)] bg-raised/80 p-4 text-sm leading-6 text-text"
            role="alert"
          >
            {document.validationMessage}
          </div>
        ) : null}

        <div className="agent-canvas-shell">
          <p className="text-sm leading-6 text-text-muted" role="status">
            Local graph with {document.nodes.length} {document.nodes.length === 1 ? "node" : "nodes"} and {document.edges.length} {document.edges.length === 1 ? "handoff" : "handoffs"}.
          </p>

          {document.nodes.length === 0 ? (
            <CanvasEmptyState
              description="Use the toolbar above to add the first role. Every node and state mutation stays local to this isolated canvas experience."
              eyebrow="Empty agent graph"
              sessionLabel={sessionLabel}
              title="Start by placing the first agent on the canvas."
            />
          ) : (
            <div className="agent-canvas-detail-grid">
              <div
                className="agent-canvas-stage agent-canvas-flow"
                data-testid="agent-canvas-flow"
              >
                <ReactFlow<AgentCanvasFlowNode, Edge>
                  aria-label="Agent canvas graph"
                  edges={flowEdges}
                  elementsSelectable
                  fitView
                  maxZoom={1.6}
                  minZoom={0.5}
                  nodeTypes={nodeTypes}
                  nodes={flowNodes}
                  nodesConnectable={false}
                  nodesDraggable={false}
                  onNodeClick={(_, node) =>
                    dispatch({ type: "select-node", nodeId: node.id })
                  }
                  onPaneClick={() => dispatch({ type: "clear-selection" })}
                  panOnDrag
                  proOptions={{ hideAttribution: true }}
                  zoomOnPinch
                  zoomOnScroll
                >
                  <Background />
                  <Controls showInteractive={false} />
                  <ViewportSync layoutRevision={document.layoutRevision} />
                </ReactFlow>
              </div>

              <AgentNodeDetailPanel selectedNode={selectedNode} />
            </div>
          )}
        </div>
      </div>
    </section>
  );
}

function ViewportSync({ layoutRevision }: { layoutRevision: number }) {
  const reactFlow = useReactFlow();

  useEffect(() => {
    if (layoutRevision === 0) {
      return;
    }

    const frame = window.requestAnimationFrame(() => {
      void reactFlow.fitView({
        duration: 250,
        maxZoom: 1.1,
        padding: 0.2,
      });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [layoutRevision, reactFlow]);

  return null;
}