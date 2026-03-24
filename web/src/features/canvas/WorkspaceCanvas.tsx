"use client";

import { motion } from "framer-motion";
import type { SessionSummary } from "@/shared/lib/workspace-protocol";
import { AgentCanvas } from "@/features/canvas/AgentCanvas";
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

  if (!activeSession) {
    return <CanvasEmptyState sessionLabel="Relay" />;
  }

  const runId = activeRunId || selectedRunId;
  const document = runId ? (orchestrationDocuments[runId] ?? null) : null;

  return (
    <motion.section
      animate={{ opacity: 1, y: 0 }}
      className="relative"
      initial={{ opacity: 0, y: 12 }}
      transition={{ duration: 0.45, ease: [0.16, 1, 0.3, 1] }}
    >
      <AgentCanvas
        document={document}
        runId={runId}
        sessionLabel={activeSession.display_name}
      />
    </motion.section>
  );
}
