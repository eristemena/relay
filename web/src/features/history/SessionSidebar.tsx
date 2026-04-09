"use client";

import { useDeferredValue } from "react";
import type { KnownProjectPayload } from "@/shared/lib/workspace-protocol";
import { NewSessionButton } from "@/features/history/NewSessionButton";
import { ProjectSwitcher } from "@/features/workspace-shell/ProjectSwitcher";

interface SessionSidebarProps {
  activeProjectRoot: string;
  knownProjects: KnownProjectPayload[];
  onOpenPreferences: () => void;
  onSwitch: (projectRoot: string) => void;
}

export function SessionSidebar({
  activeProjectRoot,
  knownProjects,
  onOpenPreferences,
  onSwitch,
}: SessionSidebarProps) {
  const deferredProjects = useDeferredValue(knownProjects);
  const activeProject =
    deferredProjects.find(
      (project) => project.project_root === activeProjectRoot,
    ) ??
    deferredProjects.find((project) => project.is_active) ??
    null;

  return (
    <section
      aria-labelledby="session-history-heading"
      className="panel-surface rounded-[2rem] p-5 shadow-idle"
    >
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="eyebrow">Project context</p>
          <h2
            id="session-history-heading"
            className="mt-2 font-display text-2xl text-text"
          >
            Projects
          </h2>
        </div>
        <NewSessionButton onClick={onOpenPreferences} />
      </div>

      {deferredProjects.length === 0 ? (
        <div className="mt-6 rounded-3xl border border-dashed border-border bg-raised/60 p-5">
          <p className="text-sm leading-6 text-text-muted">
            Relay has not connected a project root yet. Open Local settings to
            choose a repository for this workspace.
          </p>
        </div>
      ) : (
        <div className="mt-6 space-y-4">
          <div className="rounded-3xl border border-border bg-raised/60 p-4">
            <p className="text-xs uppercase tracking-[0.18em] text-text-muted">
              Active root
            </p>
            <p className="mt-2 overflow-wrap-anywhere text-sm leading-6 text-text">
              {activeProject?.project_root || activeProjectRoot}
            </p>
            <p className="mt-2 text-sm leading-6 text-text-muted">
              {deferredProjects.length > 1
                ? "Switch roots here without opening sessions manually."
                : "Relay is already scoped to the current project root."}
            </p>
          </div>
          {deferredProjects.length > 1 ? (
            <ProjectSwitcher
              activeProjectRoot={activeProjectRoot}
              knownProjects={deferredProjects}
              onSwitch={onSwitch}
            />
          ) : null}
          <ul className="space-y-3">
            {deferredProjects.map((project) => {
              const status =
                project.project_root === activeProjectRoot
                  ? "Active"
                  : !project.is_available
                    ? "Unavailable"
                    : project.blocked_reason
                      ? "Blocked"
                      : "Known";

              return (
                <li
                  className="rounded-3xl border border-border bg-raised/60 px-4 py-4"
                  key={project.project_root}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-text">
                        {project.label}
                      </p>
                      <p className="mt-1 overflow-wrap-anywhere text-xs leading-5 text-text-muted">
                        {project.project_root}
                      </p>
                    </div>
                    <p className="text-xs uppercase tracking-[0.18em] text-text-muted">
                      {status}
                    </p>
                  </div>
                  {project.blocked_reason ? (
                    <p className="mt-3 text-sm leading-6 text-text-muted">
                      {project.blocked_reason}
                    </p>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </section>
  );
}
