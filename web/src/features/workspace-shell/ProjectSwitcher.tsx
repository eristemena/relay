"use client";

import { useDeferredValue, useId } from "react";
import type { KnownProjectPayload } from "@/shared/lib/workspace-protocol";

interface ProjectSwitcherProps {
  activeProjectRoot: string;
  knownProjects: KnownProjectPayload[];
  onSwitch: (projectRoot: string) => void;
}

export function ProjectSwitcher({
  activeProjectRoot,
  knownProjects,
  onSwitch,
}: ProjectSwitcherProps) {
  const deferredProjects = useDeferredValue(knownProjects);
  const labelId = useId();

  if (deferredProjects.length <= 1) {
    const hasActiveProject = Boolean(activeProjectRoot);

    return (
      <section
        aria-labelledby={labelId}
        className="panel-surface rounded-[1.5rem] p-4 shadow-idle"
      >
        <p id={labelId} className="eyebrow">
          Project
        </p>
        <p className="mt-2 text-sm leading-6 text-text">
          {activeProjectRoot || "No active project selected yet."}
        </p>
        <p className="mt-3 text-sm leading-6 text-text-muted">
          {hasActiveProject
            ? "Relay only knows the current project so far."
            : "Open Local settings to choose the first project root for this workspace."}
        </p>
      </section>
    );
  }

  return (
    <section
      aria-labelledby={labelId}
      className="panel-surface rounded-[1.5rem] p-4 shadow-idle"
    >
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="eyebrow">Project</p>
          <h2 id={labelId} className="mt-2 font-display text-xl text-text">
            Switch active root
          </h2>
        </div>
      </div>
      <ul className="mt-4 space-y-2">
        {deferredProjects.map((project) => {
          const isCurrent = project.project_root === activeProjectRoot;
          const isBlocked = !isCurrent && Boolean(project.blocked_reason);
          const statusLabel = isCurrent
            ? "Active"
            : !project.is_available
              ? "Unavailable"
              : isBlocked
                ? "Blocked"
                : "Switch";
          const detailMessage = isCurrent
            ? "Currently active in this Relay workspace."
            : project.blocked_reason
              ? project.blocked_reason
              : project.is_available
                ? "Switch to load this project's canvas, history, and repository context."
                : "Relay cannot switch to this project until the path becomes available again.";
          return (
            <li key={project.project_root}>
              <button
                aria-current={isCurrent ? "true" : undefined}
                className="flex w-full items-start justify-between rounded-2xl border border-border bg-raised/70 px-4 py-3 text-left text-text transition-colors duration-200 hover:border-brand-mid focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand"
                disabled={!project.is_available || isCurrent || isBlocked}
                onClick={() => onSwitch(project.project_root)}
                type="button"
              >
                <span className="min-w-0">
                  <span className="block text-sm font-medium text-text">
                    {project.label}
                  </span>
                  <span className="mt-1 block overflow-wrap-anywhere text-xs text-text-muted">
                    {project.project_root}
                  </span>
                  <span className="mt-2 block text-xs leading-5 text-text-muted">
                    {detailMessage}
                  </span>
                </span>
                <span className="ml-4 shrink-0 text-xs text-text-muted">
                  {statusLabel}
                </span>
              </button>
            </li>
          );
        })}
      </ul>
    </section>
  );
}