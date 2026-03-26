"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useEffect, useRef } from "react";
import { StateBadge } from "@/features/agent-panel/StateBadge";
import {
  canvasPanelPresenceVariants,
  getCanvasTransition,
} from "@/features/canvas/canvasMotion";
import { getRunFailureTitle } from "@/features/agent-panel/runStatus";
import type {
  AgentNodeProposedChange,
  AgentCanvasRole,
  SelectedCanvasNodeView,
} from "@/features/canvas/canvasModel";
import { FormattedMarkdown } from "@/shared/lib/FormattedMarkdown";

interface AgentNodeDetailPanelProps {
  haltAgentId: string | null;
  haltCode: string | null;
  haltMessage: string;
  haltRole: AgentCanvasRole | null;
  isLoading?: boolean;
  onClose?: () => void;
  errorMessage?: string | null;
  selectedNode: SelectedCanvasNodeView | null;
}

export function AgentNodeDetailPanel({
  haltAgentId,
  haltCode,
  haltMessage,
  haltRole,
  isLoading = false,
  onClose,
  errorMessage = null,
  selectedNode,
}: AgentNodeDetailPanelProps) {
  const prefersReducedMotion = useReducedMotion() ?? false;
  const closeButtonRef = useRef<HTMLButtonElement | null>(null);
  const panelMode = selectedNode
    ? "selected"
    : isLoading
      ? "loading"
      : errorMessage
        ? "error"
        : "empty";
  const panelKey = selectedNode ? selectedNode.id : panelMode;

  if (!selectedNode) {
    return (
      <aside
        aria-labelledby="agent-canvas-detail-heading"
        className="agent-canvas-detail-panel panel-surface rounded-[1.5rem] p-5 shadow-idle"
      >
        <AnimatePresence initial={false} mode="sync">
          <motion.div
            key={panelKey}
            className="agent-canvas-panel-motion"
            data-panel-mode={panelMode}
            data-testid={`agent-canvas-detail-mode-${panelMode}`}
            initial="hidden"
            animate="visible"
            exit="exit"
            transition={getCanvasTransition(prefersReducedMotion)}
            variants={canvasPanelPresenceVariants}
          >
            {panelMode === "loading" ? (
              <div>
                <p className="eyebrow">Selected node</p>
                <h3
                  className="mt-2 font-display text-2xl text-text"
                  id="agent-canvas-detail-heading"
                >
                  Loading agent details
                </h3>
                <p className="mt-4 text-sm leading-6 text-text-muted">
                  Relay is loading the selected run details for this canvas.
                </p>
              </div>
            ) : panelMode === "error" ? (
              <div>
                <p className="eyebrow">Selected node</p>
                <h3
                  className="mt-2 font-display text-2xl text-text"
                  id="agent-canvas-detail-heading"
                >
                  Canvas details unavailable
                </h3>
                <p className="mt-4 text-sm leading-6 text-text-muted">
                  {errorMessage}
                </p>
              </div>
            ) : (
              <div>
                <p className="eyebrow">Selected node</p>
                <h3
                  className="mt-2 font-display text-2xl text-text"
                  id="agent-canvas-detail-heading"
                >
                  Inspect an agent
                </h3>
                <p className="mt-4 text-sm leading-6 text-text-muted">
                  Select a node on the canvas to review its task handoff, live
                  transcript, and any preserved failure details.
                </p>
              </div>
            )}
          </motion.div>
        </AnimatePresence>
      </aside>
    );
  }

  const showRunHalt =
    Boolean(haltMessage) &&
    (haltAgentId === selectedNode.id ||
      (haltAgentId === null && haltRole === selectedNode.role)) &&
    haltMessage !== selectedNode.details.errorMessage;
  const nodeFailureTitle =
    selectedNode.state === "clarification_required"
      ? "Clarification required"
      : "Failure";

  useEffect(() => {
    closeButtonRef.current?.focus();
  }, [selectedNode]);

  return (
    <aside
      aria-labelledby="agent-canvas-detail-heading"
      aria-modal="false"
      className="agent-canvas-detail-panel agent-canvas-detail-popup panel-surface rounded-[1.5rem] p-5 shadow-idle"
      role="dialog"
    >
      <AnimatePresence initial={false} mode="sync">
        <motion.div
          key={panelKey}
          className="agent-canvas-panel-motion agent-canvas-detail-layout h-full"
          data-panel-mode={panelMode}
          data-testid={`agent-canvas-detail-mode-${panelMode}`}
          initial="hidden"
          animate="visible"
          exit="exit"
          transition={getCanvasTransition(prefersReducedMotion)}
          variants={canvasPanelPresenceVariants}
        >
          <div className="agent-canvas-detail-header flex items-start justify-between gap-3">
            <div>
              <p className="eyebrow">Selected node</p>
              <h3
                className="mt-2 font-display text-2xl text-text"
                id="agent-canvas-detail-heading"
              >
                {selectedNode.label}
              </h3>
            </div>
            <div className="flex items-start gap-3">
              <StateBadge state={selectedNode.state} />
              <button
                aria-label="Close agent details"
                className="agent-canvas-detail-close"
                onClick={onClose}
                ref={closeButtonRef}
                type="button"
              >
                <svg
                  aria-hidden="true"
                  fill="none"
                  height="18"
                  viewBox="0 0 18 18"
                  width="18"
                >
                  <path
                    d="M4.5 4.5L13.5 13.5M13.5 4.5L4.5 13.5"
                    stroke="currentColor"
                    strokeLinecap="round"
                    strokeWidth="1.8"
                  />
                </svg>
              </button>
            </div>
          </div>

          <div className="agent-canvas-detail-scroll mt-5 grid gap-4 text-sm leading-6 text-text-muted">
            <div>
              <p className="eyebrow">Role</p>
              <p className="mt-2 text-text">{selectedNode.role}</p>
            </div>
            <div>
              <p className="eyebrow">Task</p>
              <div
                aria-label="Selected node task"
                className="agent-canvas-detail-copy mt-2"
                role="region"
              >
                <p className="whitespace-pre-wrap text-text">
                  {selectedNode.details.taskText ||
                    "This agent has not received a visible task assignment yet."}
                </p>
              </div>
            </div>
            <div>
              <p className="eyebrow">Summary</p>
              <div
                aria-label="Selected node summary"
                className="agent-canvas-detail-copy mt-2"
                role="region"
              >
                <FormattedMarkdown content={selectedNode.details.summary} />
              </div>
            </div>
            <div>
              <p className="eyebrow">Incoming handoff</p>
              <p className="mt-2 text-text">
                {selectedNode.details.incomingFrom.length
                  ? selectedNode.details.incomingFrom.join(", ")
                  : "This node currently starts the orchestration flow."}
              </p>
            </div>
            <div>
              <p className="eyebrow">Outgoing handoff</p>
              <p className="mt-2 text-text">
                {selectedNode.details.outgoingTo.length
                  ? selectedNode.details.outgoingTo.join(", ")
                  : "No downstream handoff has been recorded for this node yet."}
              </p>
            </div>
            <div>
              <p className="eyebrow">Files read</p>
              {selectedNode.details.readPaths.length ? (
                <ul className="mt-2 space-y-2 text-sm leading-6 text-text">
                  {selectedNode.details.readPaths.map((path) => (
                    <li key={path} className="break-all">
                      {path}
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-text">
                  This agent has not read a repository file yet.
                </p>
              )}
            </div>
            <div>
              <p className="eyebrow">Proposed changes</p>
              {selectedNode.details.proposedChanges.length ? (
                <ul className="mt-2 space-y-3 text-sm leading-6 text-text">
                  {selectedNode.details.proposedChanges.map((change) => (
                    <li key={change.toolCallId}>
                      <p className="break-all">{change.path}</p>
                      <p className="text-text-muted">
                        {formatProposalStatus(change)}
                      </p>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-text">
                  This agent has not proposed a file change yet.
                </p>
              )}
            </div>
            {selectedNode.details.errorMessage ? (
              <div className="rounded-2xl border border-[var(--color-error)] bg-raised/80 p-4 text-text">
                <p className="eyebrow">{nodeFailureTitle}</p>
                <p className="mt-2 text-sm leading-6">
                  {selectedNode.details.errorMessage}
                </p>
              </div>
            ) : null}
            {showRunHalt ? (
              <div className="rounded-2xl border border-[var(--color-error)] bg-raised/80 p-4 text-text">
                <p className="eyebrow">{getRunFailureTitle(haltCode)}</p>
                <p className="mt-2 text-sm leading-6">{haltMessage}</p>
              </div>
            ) : null}
            <div>
              <p className="eyebrow">Transcript</p>
              <div
                aria-live="polite"
                aria-label="Selected node transcript"
                className="agent-canvas-detail-copy relay-transcript-copy mt-2 max-h-[24rem]"
                role="region"
              >
                <FormattedMarkdown
                  className="relay-transcript-markdown text-sm leading-6 text-text"
                  content={
                    selectedNode.details.transcript ||
                    "Visible output will appear here as this agent streams or after replay restores its saved transcript."
                  }
                />
              </div>
            </div>
          </div>
        </motion.div>
      </AnimatePresence>
    </aside>
  );
}

function formatProposalStatus(change: AgentNodeProposedChange) {
  switch (change.approvalState) {
    case "proposed":
      return "Awaiting approval.";
    case "approved":
      return "Approved and waiting to apply.";
    case "applied":
      return "Applied to the repository.";
    case "rejected":
      return "Rejected before Relay wrote the change.";
    case "blocked":
      return "Blocked before Relay could apply the change.";
    case "expired":
      return "Expired before Relay could apply the change.";
  }
}