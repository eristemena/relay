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
import { useEffect, useMemo } from "react";
import { CanvasEmptyState } from "@/features/canvas/CanvasEmptyState";
import {
  AgentCanvasNode,
  type AgentCanvasFlowNode,
} from "@/features/canvas/AgentCanvasNode";
import { AgentNodeDetailPanel } from "@/features/canvas/AgentNodeDetailPanel";
import {
  getRoleLabel,
  getSelectedCanvasNode,
  type AgentCanvasDocument,
} from "@/features/canvas/canvasModel";
import {
  clearWorkspaceCanvasSelection,
  selectWorkspaceCanvasNode,
} from "@/shared/lib/workspace-store";
import { FormattedMarkdown } from "@/shared/lib/FormattedMarkdown";
import {
  getRunFailureTitle,
  isClarificationRequiredCode,
} from "@/features/agent-panel/runStatus";

interface AgentCanvasProps {
  sessionLabel: string;
  runId: string;
  document: AgentCanvasDocument | null;
}

const nodeTypes = { agentCanvasNode: AgentCanvasNode };

export function AgentCanvas({
  sessionLabel,
  runId,
  document,
}: AgentCanvasProps) {
  return (
    <ReactFlowProvider>
      <AgentCanvasSurface
        document={document}
        runId={runId}
        sessionLabel={sessionLabel}
      />
    </ReactFlowProvider>
  );
}

function AgentCanvasSurface({
  sessionLabel,
  runId,
  document,
}: AgentCanvasProps) {
  const canvasDocument = document;
  const selectedNode = canvasDocument
    ? getSelectedCanvasNode(canvasDocument)
    : null;
  const haltTitle = getRunFailureTitle(canvasDocument?.haltCode);
  const haltNote = isClarificationRequiredCode(canvasDocument?.haltCode)
    ? "Relay stopped before continuing to downstream agents because one stage asked for more input instead of taking action."
    : null;
  const flowNodes = useMemo<AgentCanvasFlowNode[]>(
    () =>
      (canvasDocument?.nodes ?? []).map((node) => ({
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
        selected: node.id === canvasDocument?.selectedNodeId,
        draggable: false,
        selectable: true,
      })),
    [canvasDocument],
  );
  const flowEdges = useMemo<Edge[]>(
    () =>
      (canvasDocument?.edges ?? []).map((edge) => ({
        id: edge.id,
        source: edge.sourceNodeId,
        target: edge.targetNodeId,
        animated: false,
        selectable: false,
      })),
    [canvasDocument],
  );

  return (
    <section
      aria-describedby="agent-canvas-description agent-canvas-status"
      aria-labelledby="agent-canvas-heading"
      className="panel-surface noise-overlay relative overflow-hidden rounded-[2rem] p-5 shadow-idle"
    >
      <div className="relative z-10 space-y-5">
        <div className="max-w-3xl">
          <p className="eyebrow">Live orchestration</p>
          <h2
            className="mt-2 font-display text-3xl text-text"
            id="agent-canvas-heading"
          >
            {sessionLabel} agent graph
          </h2>
          <p
            className="mt-3 text-sm leading-6 text-text-muted"
            id="agent-canvas-description"
          >
            Watch the live orchestration unfold, inspect any node, and reopen
            saved runs without losing per-agent context.
          </p>
        </div>

        {canvasDocument?.validationMessage ? (
          <div
            className="rounded-[1.25rem] border border-[var(--color-error)] bg-raised/80 p-4 text-sm leading-6 text-text"
            role="alert"
          >
            {canvasDocument.validationMessage}
          </div>
        ) : null}

        {canvasDocument?.haltMessage ? (
          <div
            className="rounded-[1.25rem] border border-[var(--color-error)] bg-raised/80 p-4 text-sm leading-6 text-text"
            role="alert"
          >
            <p className="eyebrow">{haltTitle}</p>
            <p className="mt-2">{canvasDocument.haltMessage}</p>
            {haltNote ? (
              <p className="mt-2 text-text-muted">{haltNote}</p>
            ) : null}
          </div>
        ) : null}

        {canvasDocument?.runSummary ? (
          <div
            aria-live="polite"
            className="rounded-[1.25rem] border border-border bg-raised/80 p-4 text-sm leading-6 text-text"
            role="status"
          >
            <FormattedMarkdown content={canvasDocument.runSummary} />
          </div>
        ) : null}

        <div className="agent-canvas-shell">
          <p
            aria-live="polite"
            className="text-sm leading-6 text-text-muted"
            id="agent-canvas-status"
            role="status"
          >
            {canvasDocument
              ? `Live graph with ${canvasDocument.nodes.length} ${canvasDocument.nodes.length === 1 ? "node" : "nodes"} and ${canvasDocument.edges.length} ${canvasDocument.edges.length === 1 ? "handoff" : "handoffs"}.`
              : "Submit a goal or reopen a saved run to populate the orchestration canvas."}
          </p>

          {!canvasDocument || canvasDocument.nodes.length === 0 ? (
            <CanvasEmptyState
              description="Relay will append nodes only when agents spawn, then patch them in place as state, transcript, and handoff events arrive."
              eyebrow="Live agent graph"
              sessionLabel={sessionLabel}
              title="Submit a goal to start the orchestration graph."
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
                    selectWorkspaceCanvasNode(runId, node.id)
                  }
                  onPaneClick={() => clearWorkspaceCanvasSelection(runId)}
                  panOnDrag
                  proOptions={{ hideAttribution: true }}
                  zoomOnPinch
                  zoomOnScroll
                >
                  <Background />
                  <Controls showInteractive={false} />
                  <ViewportSync
                    layoutRevision={canvasDocument.layoutRevision}
                  />
                </ReactFlow>
              </div>
              <AgentNodeDetailPanel
                haltAgentId={canvasDocument.haltAgentId}
                haltCode={canvasDocument.haltCode}
                haltMessage={canvasDocument.haltMessage}
                haltRole={canvasDocument.haltRole}
                selectedNode={selectedNode}
              />
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