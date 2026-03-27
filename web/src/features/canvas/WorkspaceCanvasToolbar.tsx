"use client";

import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

export type WorkspaceCanvasPanelId =
  | "sessions"
  | "history"
  | "run-summary"
  | "workspace-summary"
  | "preferences"
  | "approvals";

interface WorkspaceCanvasToolbarProps {
  activePanel: WorkspaceCanvasPanelId | null;
  expandedPanel: boolean;
  onClose: () => void;
  onToggle: (panelId: WorkspaceCanvasPanelId) => void;
  panelContent: ReactNode;
  pendingApprovalCount: number;
}

interface ToolbarButtonDefinition {
  id: WorkspaceCanvasPanelId;
  label: string;
  icon: ReactNode;
}

const toolbarButtons: ToolbarButtonDefinition[] = [
  {
    id: "sessions",
    label: "Sessions",
    icon: (
      <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
        <path
          d="M7 5.5H17M7 12H17M7 18.5H13"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="1.8"
        />
      </svg>
    ),
  },
  {
    id: "history",
    label: "Run history",
    icon: (
      <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
        <path
          d="M4 12A8 8 0 1 0 7 5.8"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="1.8"
        />
        <path
          d="M4 4V8H8M12 8V12L14.75 14.75"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.8"
        />
      </svg>
    ),
  },
  {
    id: "run-summary",
    label: "Run summary",
    icon: (
      <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
        <path
          d="M7 5.5H17M7 10.5H17M7 15.5H13"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="1.8"
        />
        <path
          d="M15.5 15.5H19.5M17.5 13.5V17.5"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="1.8"
        />
      </svg>
    ),
  },
  {
    id: "workspace-summary",
    label: "Workspace summary",
    icon: (
      <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
        <path
          d="M4.5 8.5L12 4L19.5 8.5V17.5L12 20L4.5 17.5V8.5Z"
          stroke="currentColor"
          strokeLinejoin="round"
          strokeWidth="1.8"
        />
        <path
          d="M9 11.5H15M9 15H13"
          stroke="currentColor"
          strokeLinecap="round"
          strokeWidth="1.8"
        />
      </svg>
    ),
  },
  {
    id: "preferences",
    label: "Preferences",
    icon: (
      <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
        <path
          d="M12 8.25A3.75 3.75 0 1 0 12 15.75A3.75 3.75 0 1 0 12 8.25Z"
          stroke="currentColor"
          strokeWidth="1.8"
        />
        <path
          d="M19 12C19 11.5 18.95 11.02 18.86 10.55L21 8.86L18.86 5.14L16.25 6.1C15.53 5.53 14.7 5.11 13.8 4.89L13.38 2H10.62L10.2 4.89C9.3 5.11 8.47 5.53 7.75 6.1L5.14 5.14L3 8.86L5.14 10.55C5.05 11.02 5 11.5 5 12C5 12.5 5.05 12.98 5.14 13.45L3 15.14L5.14 18.86L7.75 17.9C8.47 18.47 9.3 18.89 10.2 19.11L10.62 22H13.38L13.8 19.11C14.7 18.89 15.53 18.47 16.25 17.9L18.86 18.86L21 15.14L18.86 13.45C18.95 12.98 19 12.5 19 12Z"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.4"
        />
      </svg>
    ),
  },
  {
    id: "approvals",
    label: "Approval review",
    icon: (
      <svg aria-hidden="true" fill="none" viewBox="0 0 24 24">
        <path
          d="M12 3L20 7.2V12.8C20 16.93 16.86 20.62 12 21C7.14 20.62 4 16.93 4 12.8V7.2L12 3Z"
          stroke="currentColor"
          strokeLinejoin="round"
          strokeWidth="1.8"
        />
        <path
          d="M9.5 12.25L11.15 13.9L14.8 10.25"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="1.8"
        />
      </svg>
    ),
  },
];

export function WorkspaceCanvasToolbar({
  activePanel,
  expandedPanel,
  onClose,
  onToggle,
  panelContent,
  pendingApprovalCount,
}: WorkspaceCanvasToolbarProps) {
  const buttonRefs = useRef<
    Partial<Record<WorkspaceCanvasPanelId, HTMLButtonElement | null>>
  >({});
  const previousActivePanelRef = useRef<WorkspaceCanvasPanelId | null>(null);
  const [portalReady, setPortalReady] = useState(false);

  useEffect(() => {
    const previousPanel = previousActivePanelRef.current;

    if (previousPanel && !activePanel) {
      buttonRefs.current[previousPanel]?.focus();
    }

    previousActivePanelRef.current = activePanel;
  }, [activePanel]);

  useEffect(() => {
    setPortalReady(true);
    return () => setPortalReady(false);
  }, []);

  const lightbox = (
    <AnimatePresence initial={false} mode="wait">
      {activePanel ? (
        <motion.div
          animate={{ opacity: 1 }}
          className="workspace-canvas-lightbox-shell"
          data-testid="workspace-canvas-lightbox"
          exit={{ opacity: 0 }}
          initial={{ opacity: 0 }}
          key={activePanel}
          transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
        >
          <button
            aria-label="Dismiss workspace panel overlay"
            className="workspace-canvas-lightbox-backdrop"
            onClick={onClose}
            type="button"
          />
          <motion.aside
            animate={{ opacity: 1, scale: 1, y: 0 }}
            aria-labelledby={`workspace-canvas-toolbar-trigger-${activePanel}`}
            aria-modal="true"
            className="workspace-canvas-panel"
            data-size={expandedPanel ? "expanded" : "standard"}
            data-testid="workspace-canvas-panel"
            exit={{ opacity: 0, scale: 0.98, y: 10 }}
            id="workspace-canvas-panel"
            initial={{ opacity: 0, scale: 0.98, y: 10 }}
            role="dialog"
            transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
          >
            <div className="workspace-canvas-panel-header">
              <div>
                <p className="eyebrow">Workspace panel</p>
                <p className="mt-2 text-sm leading-6 text-text-muted">
                  Review workspace details here without leaving the graph.
                </p>
              </div>
              <button
                aria-label="Close workspace panel"
                className="workspace-canvas-panel-close"
                onClick={onClose}
                type="button"
              >
                <svg aria-hidden="true" fill="none" viewBox="0 0 18 18">
                  <path
                    d="M4.5 4.5L13.5 13.5M13.5 4.5L4.5 13.5"
                    stroke="currentColor"
                    strokeLinecap="round"
                    strokeWidth="1.8"
                  />
                </svg>
              </button>
            </div>
            <div className="workspace-canvas-panel-body">{panelContent}</div>
          </motion.aside>
        </motion.div>
      ) : null}
    </AnimatePresence>
  );

  return (
    <div className="workspace-canvas-toolbar-shell">
      <div
        aria-label="Workspace panels"
        className="workspace-canvas-toolbar"
        role="toolbar"
      >
        {toolbarButtons.map((panel) => {
          const isActive = activePanel === panel.id;
          const approvalSuffix =
            panel.id === "approvals" && pendingApprovalCount > 0
              ? `, ${pendingApprovalCount} pending ${pendingApprovalCount === 1 ? "request" : "requests"}`
              : "";

          return (
            <button
              aria-controls="workspace-canvas-panel"
              aria-expanded={isActive}
              aria-label={`${isActive ? "Close" : "Open"} ${panel.label}${approvalSuffix}`}
              aria-pressed={isActive}
              className="workspace-canvas-toolbar-button"
              data-active={isActive ? "true" : "false"}
              id={`workspace-canvas-toolbar-trigger-${panel.id}`}
              key={panel.id}
              onClick={() => onToggle(panel.id)}
              ref={(element) => {
                buttonRefs.current[panel.id] = element;
              }}
              title={panel.label}
              type="button"
            >
              <span aria-hidden="true" className="workspace-canvas-toolbar-icon">
                {panel.icon}
              </span>
              <span className="sr-only">{panel.label}</span>
              {panel.id === "approvals" && pendingApprovalCount > 0 ? (
                <span
                  aria-hidden="true"
                  className="workspace-canvas-toolbar-badge"
                >
                  {pendingApprovalCount}
                </span>
              ) : null}
            </button>
          );
        })}
      </div>

      {portalReady ? createPortal(lightbox, document.body) : null}
    </div>
  );
}