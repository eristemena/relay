"use client";

import type { ReactNode } from "react";

interface WorkspaceUtilityDrawerProps {
  children: ReactNode;
  onClose: () => void;
  open: boolean;
}

export function WorkspaceUtilityDrawer({
  children,
  onClose,
  open,
}: WorkspaceUtilityDrawerProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="workspace-utility-shell" data-state="open">
      <div
        aria-hidden="true"
        className="workspace-utility-backdrop"
        onClick={onClose}
      />
      <aside
        aria-labelledby="workspace-utility-heading"
        aria-modal="true"
        className="workspace-utility-drawer panel-surface"
        role="dialog"
      >
        <div className="flex items-start justify-between gap-4 border-b border-border px-5 py-5">
          <div>
            <p className="eyebrow">Workspace menu</p>
            <h2
              className="mt-2 font-display text-2xl text-text"
              id="workspace-utility-heading"
            >
              Sessions, saved runs, and preferences
            </h2>
          </div>
          <button
            aria-label="Close workspace menu"
            className="agent-canvas-detail-close"
            onClick={onClose}
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

        <div className="workspace-utility-scroll">{children}</div>
      </aside>
    </div>
  );
}