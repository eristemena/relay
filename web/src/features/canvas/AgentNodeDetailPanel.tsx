"use client";

import { StateBadge } from "@/features/agent-panel/StateBadge";
import type { AgentCanvasNodeModel } from "@/features/canvas/canvasModel";

interface AgentNodeDetailPanelProps {
  selectedNode: AgentCanvasNodeModel | null;
}

export function AgentNodeDetailPanel({
  selectedNode,
}: AgentNodeDetailPanelProps) {
  if (!selectedNode) {
    return null;
  }

  return (
    <aside
      aria-labelledby="agent-canvas-detail-heading"
      className="agent-canvas-detail-panel panel-surface rounded-[1.5rem] p-5 shadow-idle"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="eyebrow">Selected node</p>
          <h3
            className="mt-2 font-display text-2xl text-text"
            id="agent-canvas-detail-heading"
          >
            {selectedNode.label}
          </h3>
        </div>
        <StateBadge state={selectedNode.state} />
      </div>

      <div className="mt-5 grid gap-4 text-sm leading-6 text-text-muted">
        <div>
          <p className="eyebrow">Role</p>
          <p className="mt-2 text-text">{selectedNode.role}</p>
        </div>
        <div>
          <p className="eyebrow">Summary</p>
          <p className="mt-2">{selectedNode.details.summary}</p>
        </div>
        <div>
          <p className="eyebrow">Incoming handoff</p>
          <p className="mt-2 text-text">
            {selectedNode.details.incomingFrom.length
              ? selectedNode.details.incomingFrom.join(", ")
              : "This node currently starts the local workflow."}
          </p>
        </div>
        <div>
          <p className="eyebrow">Outgoing handoff</p>
          <p className="mt-2 text-text">
            {selectedNode.details.outgoingTo.length
              ? selectedNode.details.outgoingTo.join(", ")
              : "No downstream node yet. Add another role to extend the graph."}
          </p>
        </div>
        <p className="rounded-2xl border border-border bg-raised/70 p-4">
          Local-only detail panel. These values simulate the canvas experience and do not reflect a live Relay run.
        </p>
      </div>
    </aside>
  );
}