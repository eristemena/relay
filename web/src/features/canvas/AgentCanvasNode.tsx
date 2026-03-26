"use client";

import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { motion, useReducedMotion } from "framer-motion";
import { useEffect, useRef, useState } from "react";
import { StateBadge } from "@/features/agent-panel/StateBadge";
import {
  CANVAS_STREAMING_SILENCE_MS,
  canvasNodeEnterVariants,
  getCanvasTransition,
  getNodeMotionTarget,
} from "@/features/canvas/canvasMotion";
import type {
  AgentCanvasRole,
  AgentCanvasState,
} from "@/features/canvas/canvasModel";

export interface AgentCanvasNodeData extends Record<string, unknown> {
  label: string;
  roleLabel: string;
  role: AgentCanvasRole;
  state: AgentCanvasState;
  stateRevision: number;
  readCount: number;
  proposalCount: number;
  summary: string;
}

export type AgentCanvasFlowNode = Node<AgentCanvasNodeData, "agentCanvasNode">;

export function AgentCanvasNode({
  data,
  selected,
}: NodeProps<AgentCanvasFlowNode>) {
  const prefersReducedMotion = useReducedMotion() ?? false;
  const timeoutRef = useRef<number | null>(null);
  const [streamingActive, setStreamingActive] = useState(
    data.state === "streaming",
  );

  useEffect(() => {
    return () => {
      if (timeoutRef.current !== null) {
        window.clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (timeoutRef.current !== null) {
      window.clearTimeout(timeoutRef.current);
      timeoutRef.current = null;
    }

    if (data.state !== "streaming") {
      setStreamingActive(false);
      return;
    }

    setStreamingActive(true);
    timeoutRef.current = window.setTimeout(() => {
      setStreamingActive(false);
      timeoutRef.current = null;
    }, CANVAS_STREAMING_SILENCE_MS);
  }, [data.state, data.stateRevision]);

  return (
    <motion.div
      className="agent-canvas-node"
      data-selected={selected}
      data-state={data.state}
      data-state-revision={data.stateRevision}
      data-streaming-active={streamingActive}
      initial="hidden"
      animate={getNodeMotionTarget({
        reducedMotion: prefersReducedMotion,
        selected,
        state: data.state,
        streamingActive,
      })}
      transition={getCanvasTransition(prefersReducedMotion)}
      variants={canvasNodeEnterVariants}
    >
      <Handle
        className="agent-canvas-node-handle agent-canvas-node-handle-target"
        isConnectable={false}
        position={Position.Left}
        type="target"
      />
      <button
        aria-label={`${data.label}, ${data.roleLabel} node`}
        className="agent-canvas-node-button"
        type="button"
      >
        <span aria-hidden="true" className="agent-canvas-node-streaming-ring" />
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="eyebrow">{data.roleLabel}</p>
            <h3 className="mt-2 font-display text-xl text-text">
              {data.label}
            </h3>
          </div>
          <StateBadge state={data.state} />
        </div>
        <p className="mt-4 text-sm leading-6 text-text-muted">{data.summary}</p>
        <p className="mt-4 text-xs text-text-muted">
          {formatActivitySummary(data.readCount, data.proposalCount)}
        </p>
        <p className="mt-4 text-xs uppercase tracking-[0.22em] text-text-muted">
          {selected ? "Selected" : "Inspect node"}
        </p>
      </button>
      <Handle
        className="agent-canvas-node-handle agent-canvas-node-handle-source"
        isConnectable={false}
        position={Position.Right}
        type="source"
      />
    </motion.div>
  );
}

function formatActivitySummary(readCount: number, proposalCount: number) {
  const segments: string[] = [];

  if (readCount > 0) {
    segments.push(`${readCount} file${readCount === 1 ? "" : "s"} read`);
  }
  if (proposalCount > 0) {
    segments.push(
      `${proposalCount} change${proposalCount === 1 ? "" : "s"} proposed`,
    );
  }

  return segments.length > 0
    ? segments.join(" · ")
    : "No repository file activity recorded yet.";
}