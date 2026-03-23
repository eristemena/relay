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
      <button
        aria-label="Close workspace menu"
        className="workspace-utility-backdrop"
        onClick={onClose}
        type="button"
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
            className="rounded-full border border-border bg-raised px-4 py-2 text-sm text-text"
            onClick={onClose}
            type="button"
          >
            Close
          </button>
        </div>

        <div className="workspace-utility-scroll">{children}</div>
      </aside>
    </div>
  );
}