"use client";

import { motion } from "framer-motion";
import type { SessionSummary } from "@/shared/lib/workspace-protocol";
import { AgentCanvas } from "@/features/canvas/AgentCanvas";
import { CanvasEmptyState } from "@/features/canvas/CanvasEmptyState";

interface WorkspaceCanvasProps {
  activeSession: SessionSummary | null;
}

export function WorkspaceCanvas({ activeSession }: WorkspaceCanvasProps) {
  if (!activeSession) {
    return <CanvasEmptyState sessionLabel="Relay" />;
  }

  return (
    <motion.section
      animate={{ opacity: 1, y: 0 }}
      className="relative"
      initial={{ opacity: 0, y: 12 }}
      transition={{ duration: 0.45, ease: [0.16, 1, 0.3, 1] }}
    >
      <AgentCanvas sessionLabel={activeSession.display_name} />
    </motion.section>
  );
}
