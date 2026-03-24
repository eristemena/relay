"use client";

import { getBezierPath, type EdgeProps } from "@xyflow/react";
import type { AgentCanvasEdgePulseState } from "@/features/canvas/canvasModel";

export interface AnimatedHandoffEdgeData extends Record<string, unknown> {
  pulseState: AgentCanvasEdgePulseState;
}

export function AnimatedHandoffEdge({
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps) {
  const [edgePath] = getBezierPath({
    sourceX,
    sourceY,
    targetX,
    targetY,
    sourcePosition,
    targetPosition,
  });
  const pulseState =
    (data as AnimatedHandoffEdgeData | undefined)?.pulseState ?? "idle";

  return (
    <g aria-hidden="true" className="agent-canvas-edge" data-pulse-state={pulseState}>
      <path className="react-flow__edge-path agent-canvas-edge-path" d={edgePath} fill="none" />
      {pulseState !== "idle" ? (
        <path
          className="agent-canvas-edge-pulse"
          data-pulse-state={pulseState}
          d={edgePath}
          fill="none"
        />
      ) : null}
    </g>
  );
}