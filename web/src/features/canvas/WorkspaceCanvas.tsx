"use client";

import { motion } from "framer-motion";
import type { SessionSummary } from "@/shared/lib/workspace-protocol";
import { AgentCanvas } from "@/features/canvas/AgentCanvas";
import {
  CANVAS_MOTION_EASE,
  CANVAS_MOTION_SECONDS,
} from "@/features/canvas/canvasMotion";
import { CanvasEmptyState } from "@/features/canvas/CanvasEmptyState";
import { useWorkspaceStore } from "@/shared/lib/workspace-store";

interface WorkspaceCanvasProps {
  activeSession: SessionSummary | null;
}

export function WorkspaceCanvas({ activeSession }: WorkspaceCanvasProps) {
  const activeRunId = useWorkspaceStore((state) => state.activeRunId);
  const selectedRunId = useWorkspaceStore((state) => state.selectedRunId);
  const orchestrationDocuments = useWorkspaceStore(
    (state) => state.orchestrationDocuments,
  );
  const uiState = useWorkspaceStore((state) => state.uiState);
  const error = useWorkspaceStore((state) => state.error);

  if (!activeSession) {
    return <CanvasEmptyState sessionLabel="Relay" />;
  }

  const runId = activeRunId || selectedRunId;
  const document = runId ? (orchestrationDocuments[runId] ?? null) : null;
  const canvasErrorMessage =
    uiState.canvas_state === "error" || (!document && error)
      ? error?.message || "Relay could not load the orchestration canvas."
      : null;

  return (
    <motion.section
      animate={{ opacity: 1, y: 0 }}
      className="relative"
      initial={{ opacity: 0, y: 12 }}
      transition={{ duration: CANVAS_MOTION_SECONDS, ease: CANVAS_MOTION_EASE }}
    >
      <AgentCanvas
        canvasState={uiState.canvas_state}
        document={document}
        errorMessage={canvasErrorMessage}
        historyState={uiState.history_state}
        runId={runId}
        sessionLabel={activeSession.display_name}
      />
    </motion.section>
  );
}
