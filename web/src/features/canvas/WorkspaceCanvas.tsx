"use client";

import { motion } from "framer-motion";
import type { SessionSummary } from "@/shared/lib/workspace-protocol";
import { CanvasEmptyState } from "@/features/canvas/CanvasEmptyState";

interface WorkspaceCanvasProps {
  activeSession: SessionSummary | null;
}

export function WorkspaceCanvas({ activeSession }: WorkspaceCanvasProps) {
  if (!activeSession) {
    return <CanvasEmptyState sessionLabel="Relay" />;
  }

  if (!activeSession.has_activity) {
    return <CanvasEmptyState sessionLabel={activeSession.display_name} />;
  }

  return (
    <motion.section
      animate={{ opacity: 1, y: 0 }}
      className="panel-surface canvas-grid noise-overlay relative min-h-[28rem] overflow-hidden rounded-[2rem] p-6 shadow-complete"
      initial={{ opacity: 0, y: 12 }}
      transition={{ duration: 0.45, ease: [0.16, 1, 0.3, 1] }}
    >
      <div className="relative z-10 max-w-3xl">
        <p className="eyebrow">Active canvas</p>
        <h2 className="mt-3 font-display text-3xl text-text">{activeSession.display_name}</h2>
        <div className="mt-8 grid gap-4 md:grid-cols-2">
          <article className="rounded-3xl border border-border bg-raised/80 p-5">
            <p className="eyebrow">Agent flow</p>
            <p className="mt-3 text-sm leading-6 text-text-muted">
              Relay will project future agent nodes here as the session accumulates activity.
            </p>
          </article>
          <article className="rounded-3xl border border-border bg-raised/80 p-5">
            <p className="eyebrow">Workspace memory</p>
            <p className="mt-3 text-sm leading-6 text-text-muted">
              This run already has activity, so the canvas stays ready instead of dropping back to the empty-state guide.
            </p>
          </article>
        </div>
      </div>
    </motion.section>
  );
}
