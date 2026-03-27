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
import { AnimatePresence } from "framer-motion";
import { useEffect, useMemo, type ReactNode } from "react";
import {
  AnimatedHandoffEdge,
  type AnimatedHandoffEdgeData,
} from "@/features/canvas/AnimatedHandoffEdge";
import { CanvasEmptyState } from "@/features/canvas/CanvasEmptyState";
import {
  AgentCanvasNode,
  type AgentCanvasFlowNode,
} from "@/features/canvas/AgentCanvasNode";
import { AgentNodeDetailPanel } from "@/features/canvas/AgentNodeDetailPanel";
import {
  type AgentCanvasEdgePulseState,
  getRoleLabel,
  getSelectedCanvasNode,
  type AgentCanvasDocument,
} from "@/features/canvas/canvasModel";
import {
  clearWorkspaceCanvasSelection,
  selectWorkspaceCanvasNode,
} from "@/shared/lib/workspace-store";
import type { WorkspaceUIState } from "@/shared/lib/workspace-protocol";
import {
  getRunFailureTitle,
  isClarificationRequiredCode,
} from "@/features/agent-panel/runStatus";

interface AgentCanvasProps {
  canvasState?: WorkspaceUIState["canvas_state"];
  sessionLabel: string;
  historyState?: WorkspaceUIState["history_state"];
  errorMessage?: string | null;
  runId: string;
  document: AgentCanvasDocument | null;
  workspaceToolbar?: ReactNode;
}

const nodeTypes = { agentCanvasNode: AgentCanvasNode };
const edgeTypes = { animatedHandoff: AnimatedHandoffEdge };
type AgentCanvasFlowEdge = Edge<AnimatedHandoffEdgeData>;

export function AgentCanvas({
  canvasState,
  sessionLabel,
  historyState,
  errorMessage,
  runId,
  document,
  workspaceToolbar,
}: AgentCanvasProps) {
  return (
    <ReactFlowProvider>
      <AgentCanvasSurface
        canvasState={canvasState}
        document={document}
        errorMessage={errorMessage}
        historyState={historyState}
        runId={runId}
        sessionLabel={sessionLabel}
        workspaceToolbar={workspaceToolbar}
      />
    </ReactFlowProvider>
  );
}

function AgentCanvasSurface({
  canvasState,
  sessionLabel,
  historyState,
  errorMessage,
  runId,
  document,
  workspaceToolbar,
}: AgentCanvasProps) {
  const canvasDocument = document;
  const canvasErrorMessage =
    !canvasDocument && errorMessage
      ? errorMessage
      : canvasState === "error"
        ? errorMessage || "Relay could not load the orchestration canvas."
        : null;
  const isCanvasLoading =
    !canvasDocument &&
    !canvasErrorMessage &&
    (historyState === "loading" || Boolean(runId));
  const selectedNode = canvasDocument
    ? getSelectedCanvasNode(canvasDocument)
    : null;
  const haltTitle = getRunFailureTitle(canvasDocument?.haltCode);
  const haltNote = isClarificationRequiredCode(canvasDocument?.haltCode)
    ? "Relay stopped before continuing to downstream agents because one stage asked for more input instead of taking action."
    : null;

  useEffect(() => {
    if (!selectedNode) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") {
        return;
      }

      clearWorkspaceCanvasSelection(runId);
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [runId, selectedNode]);

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
          stateRevision: node.stateRevision,
          readCount: node.details.readPaths.length,
          proposalCount: node.details.proposedChanges.length,
          summary: node.details.summary,
        },
        selected: node.id === canvasDocument?.selectedNodeId,
        draggable: false,
        selectable: true,
      })),
    [canvasDocument],
  );
  const flowEdges = useMemo<AgentCanvasFlowEdge[]>(
    () =>
      (canvasDocument?.edges ?? []).map((edge) => ({
        id: edge.id,
        source: edge.sourceNodeId,
        target: edge.targetNodeId,
        type: "animatedHandoff",
        animated: false,
        data: {
          pulseState: edge.pulseState as AgentCanvasEdgePulseState,
        },
        selectable: false,
      })),
    [canvasDocument],
  );

  return (
    <section
      aria-describedby="agent-canvas-description agent-canvas-status"
      aria-labelledby="agent-canvas-heading"
      className="panel-surface noise-overlay relative flex h-full min-h-0 flex-col overflow-hidden rounded-[2rem] p-5 shadow-idle"
    >
      <div className="relative z-10 flex h-full min-h-0 flex-col gap-5">
        <h2 className="sr-only" id="agent-canvas-heading">
          {sessionLabel} agent graph
        </h2>
        <p className="sr-only" id="agent-canvas-description">
          Inspect any node and reopen saved runs without losing per-agent
          context.
        </p>

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

          {workspaceToolbar ? workspaceToolbar : null}

          {isCanvasLoading ? (
            <div
              className="agent-canvas-inline-state rounded-[1.25rem] border border-border bg-raised/80 p-4 text-sm leading-6 text-text"
              role="status"
            >
              Relay is loading the orchestration canvas.
            </div>
          ) : canvasErrorMessage && !canvasDocument ? (
            <div
              className="agent-canvas-inline-state rounded-[1.25rem] border border-[var(--color-error)] bg-raised/80 p-4 text-sm leading-6 text-text"
              role="alert"
            >
              {canvasErrorMessage}
            </div>
          ) : !canvasDocument || canvasDocument.nodes.length === 0 ? (
            <CanvasEmptyState
              description="Relay will append nodes only when agents spawn, then patch them in place as state, transcript, and handoff events arrive."
              eyebrow="Live agent graph"
              sessionLabel={sessionLabel}
              title="Submit a goal to start the orchestration graph."
            />
          ) : (
            <div
              className="agent-canvas-detail-grid"
              data-detail-open={selectedNode ? "true" : "false"}
            >
              <div className="agent-canvas-stage-shell">
                <div
                  className="agent-canvas-stage agent-canvas-flow"
                  data-testid="agent-canvas-flow"
                >
                  <ReactFlow<AgentCanvasFlowNode, AgentCanvasFlowEdge>
                    aria-label="Agent canvas graph"
                    edgeTypes={edgeTypes}
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
                <AnimatePresence initial={false} mode="sync">
                  {selectedNode ? (
                    <AgentNodeDetailPanel
                      haltAgentId={canvasDocument.haltAgentId}
                      haltCode={canvasDocument.haltCode}
                      haltMessage={canvasDocument.haltMessage}
                      haltRole={canvasDocument.haltRole}
                      key={selectedNode.id}
                      onClose={() => clearWorkspaceCanvasSelection(runId)}
                      selectedNode={selectedNode}
                    />
                  ) : null}
                </AnimatePresence>
              </div>
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