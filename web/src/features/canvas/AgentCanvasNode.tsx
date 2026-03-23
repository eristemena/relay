"use client";

import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { StateBadge } from "@/features/agent-panel/StateBadge";
import type {
  AgentCanvasRole,
  AgentCanvasState,
} from "@/features/canvas/canvasModel";

export interface AgentCanvasNodeData extends Record<string, unknown> {
  label: string;
  roleLabel: string;
  role: AgentCanvasRole;
  state: AgentCanvasState;
  summary: string;
}

export type AgentCanvasFlowNode = Node<AgentCanvasNodeData, "agentCanvasNode">;

export function AgentCanvasNode({ data, selected }: NodeProps<AgentCanvasFlowNode>) {
  return (
    <div
      className="agent-canvas-node"
      data-selected={selected}
      data-state={data.state}
    >
      <Handle
        className="!h-3 !w-3 !border-border !bg-raised"
        isConnectable={false}
        position={Position.Left}
        type="target"
      />
      <button
        aria-label={`${data.label}, ${data.roleLabel} node`}
        className="agent-canvas-node-button"
        type="button"
      >
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="eyebrow">{data.roleLabel}</p>
            <h3 className="mt-2 font-display text-xl text-text">{data.label}</h3>
          </div>
          <StateBadge state={data.state} />
        </div>
        <p className="mt-4 text-sm leading-6 text-text-muted">{data.summary}</p>
        <p className="mt-4 text-xs uppercase tracking-[0.22em] text-text-muted">
          {selected ? "Selected" : "Inspect node"}
        </p>
      </button>
      <Handle
        className="!h-3 !w-3 !border-border !bg-raised"
        isConnectable={false}
        position={Position.Right}
        type="source"
      />
    </div>
  );
}