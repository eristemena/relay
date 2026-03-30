"use client";

import { useMemo } from "react";
import type { ConnectedRepositoryView, TouchedFilePayload } from "@/shared/lib/workspace-protocol";
import type { RepositoryTreeState } from "@/shared/lib/workspace-store";
import { buildRepositoryTreeView } from "@/features/history/treeModel";

interface RepositoryFileTreePanelProps {
  connectedRepository: ConnectedRepositoryView;
  repositoryTree: RepositoryTreeState;
  selectedAgentId: string | null;
  selectedAgentLabel: string | null;
  onClearSelectedAgent?: () => void;
  onRetry: () => void;
  onTogglePath: (path: string) => void;
}

export function RepositoryFileTreePanel({
  connectedRepository,
  repositoryTree,
  selectedAgentId,
  selectedAgentLabel,
  onClearSelectedAgent,
  onRetry,
  onTogglePath,
}: RepositoryFileTreePanelProps) {
  const treeView = useMemo(
    () =>
      buildRepositoryTreeView({
        paths: repositoryTree.paths,
        touchedFiles: repositoryTree.touchedFiles,
        selectedAgentId,
        expandedPaths: repositoryTree.expandedPaths,
      }),
    [
      repositoryTree.expandedPaths,
      repositoryTree.paths,
      repositoryTree.touchedFiles,
      selectedAgentId,
    ],
  );
  const showFilteredEmptyState =
    Boolean(selectedAgentId) &&
    treeView.entries.length === 0 &&
    treeView.missingTouchedPaths.length === 0;
  const unavailableTitle =
    connectedRepository.status === "invalid"
      ? "Repository needs attention"
      : "Repository not connected";
  const unavailableMessage =
    connectedRepository.status === "invalid"
      ? connectedRepository.message ||
        "Relay could not use the saved project root. Choose a valid local Git repository."
      : connectedRepository.message ||
        "Connect a local Git repository to browse its file tree.";

  return (
    <section
      aria-labelledby="repository-tree-heading"
      className="flex h-full min-h-0 flex-col gap-4 overflow-hidden"
    >
      <div className="panel-surface shrink-0 rounded-[2rem] p-5 shadow-idle">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="eyebrow">Repository tree</p>
            <h2
              className="mt-2 font-display text-2xl text-text"
              id="repository-tree-heading"
            >
              Connected files
            </h2>
          </div>
          {selectedAgentLabel ? (
            <div className="flex flex-wrap items-center justify-end gap-2">
              <p className="rounded-full border border-border px-3 py-1 text-xs uppercase tracking-[0.18em] text-text-muted">
                Filtered to {selectedAgentLabel}
              </p>
              {onClearSelectedAgent ? (
                <button
                  className="rounded-full border border-border px-3 py-1 text-xs uppercase tracking-[0.18em] text-text hover:border-brand-mid hover:bg-raised"
                  onClick={onClearSelectedAgent}
                  type="button"
                >
                  Clear agent filter
                </button>
              ) : null}
            </div>
          ) : (
            <p className="rounded-full border border-border px-3 py-1 text-xs uppercase tracking-[0.18em] text-text-muted">
              All agents
            </p>
          )}
        </div>
        <p className="mt-4 text-sm leading-6 text-text-muted">
          Browse the connected repository in read-only mode and see which files
          the current run has read or proposed to change.
        </p>
        <p className="repository-tree-helper mt-3 text-sm leading-6 text-text-muted">
          Folder rows expand in place. File rows stay read-only and do not open
          editors, apply diffs, or run commands.
        </p>
        {connectedRepository.path ? (
          <p className="mt-3 rounded-3xl border border-border bg-raised/60 px-4 py-3 font-mono text-xs text-text-muted">
            {connectedRepository.path}
          </p>
        ) : null}
        {repositoryTree.syncErrorMessage ? (
          <p className="mt-3 rounded-3xl border border-error px-4 py-3 text-sm text-error">
            {repositoryTree.syncErrorMessage}
          </p>
        ) : null}
      </div>

      {connectedRepository.status !== "connected" ? (
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <UnavailableState
            title={unavailableTitle}
            message={unavailableMessage}
          />
        </div>
      ) : repositoryTree.status === "loading" ? (
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <StateCard
            title="Loading repository tree"
            message="Loading the connected repository tree."
          />
        </div>
      ) : repositoryTree.status === "error" ? (
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <StateCard
            title="Repository tree unavailable"
            actionLabel="Retry"
            message={repositoryTree.message || "Relay could not load the connected repository tree."}
            onAction={onRetry}
          />
        </div>
      ) : repositoryTree.status === "empty" ? (
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <StateCard
            title="No tracked files yet"
            actionLabel="Refresh"
            message={repositoryTree.message || "This repository does not have any tracked files to display yet."}
            onAction={onRetry}
          />
        </div>
      ) : showFilteredEmptyState ? (
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <StateCard
            title="No files for selected agent"
            actionLabel={onClearSelectedAgent ? "Show all files" : undefined}
            message={`${selectedAgentLabel || "This agent"} has not touched any files in the current tree yet.`}
            onAction={onClearSelectedAgent}
          />
        </div>
      ) : (
        <div className="panel-surface flex min-h-0 flex-1 flex-col overflow-hidden rounded-[2rem] p-5 shadow-idle">
          <div className="min-h-0 flex-1 overflow-y-auto pr-1">
            <ul className="space-y-2" role="list">
              {treeView.entries.map((entry) => (
                <li key={entry.path}>
                  {entry.kind === "directory" ? (
                    <button
                      aria-expanded={entry.expanded}
                      className="repository-tree-row repository-tree-row-action flex w-full items-center gap-3 rounded-[1.25rem] border border-transparent px-3 py-2 text-left text-sm text-text hover:border-border hover:bg-raised/70"
                      data-kind="directory"
                      onClick={() => onTogglePath(entry.path)}
                      style={{ paddingLeft: `${entry.depth * 1.25 + 0.75}rem` }}
                      type="button"
                    >
                      <span aria-hidden="true" className="font-mono text-xs text-text-muted">
                        {entry.expanded ? "v" : ">"}
                      </span>
                      <span className="font-medium">{entry.name}</span>
                    </button>
                  ) : (
                    <div
                      className="repository-tree-row repository-tree-row-static flex w-full items-center justify-between gap-3 rounded-[1.25rem] border border-transparent px-3 py-2 text-left text-sm text-text"
                      data-kind="file"
                      style={{ paddingLeft: `${entry.depth * 1.25 + 1.95}rem` }}
                    >
                      <span className="min-w-0 truncate">{entry.name}</span>
                      <TouchBadges touchKinds={entry.touchKinds} />
                    </div>
                  )}
                </li>
              ))}
            </ul>
            {treeView.missingTouchedPaths.length > 0 ? (
              <div className="repository-tree-missing mt-5 rounded-[1.5rem] border border-dashed border-border bg-raised/60 p-4">
                <p className="text-sm text-text">
                  Some historical file activity no longer matches the connected repository tree.
                </p>
                <ul className="mt-3 space-y-2 text-sm text-text-muted">
                  {treeView.missingTouchedPaths.map((path) => (
                    <li key={path} className="flex flex-wrap items-center gap-2 font-mono text-xs">
                      <span className="repository-tree-missing-badge">Missing now</span>
                      <span>{path}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ) : null}
          </div>
        </div>
      )}
    </section>
  );
}

function TouchBadges({ touchKinds }: { touchKinds: TouchedFilePayload["touch_type"][] }) {
  if (touchKinds.length === 0) {
    return <span className="repository-tree-badge repository-tree-badge-muted text-xs">Untouched</span>;
  }

  return (
    <span className="flex flex-wrap justify-end gap-2">
      {touchKinds.map((touchKind) => (
        <span
          className="repository-tree-badge rounded-full border border-border px-2 py-1 text-[0.68rem] font-medium uppercase tracking-[0.16em] text-text"
          data-touch-kind={touchKind}
          key={touchKind}
        >
          {touchKind === "proposed" ? "Proposed" : "Read"}
        </span>
      ))}
    </span>
  );
}

function StateCard({
  title,
  message,
  actionLabel,
  onAction,
}: {
  title: string;
  message: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <div className="panel-surface rounded-[2rem] p-5 shadow-idle">
      <p className="text-sm font-medium text-text">{title}</p>
      <p className="text-sm leading-6 text-text-muted">{message}</p>
      {actionLabel && onAction ? (
        <button
          className="mt-4 rounded-full border border-border px-4 py-2 text-sm font-medium text-text hover:border-brand-mid hover:bg-raised"
          onClick={onAction}
          type="button"
        >
          {actionLabel}
        </button>
      ) : null}
    </div>
  );
}

function UnavailableState({
  title,
  message,
}: {
  title: string;
  message: string;
}) {
  return (
    <div className="panel-surface rounded-[2rem] p-5 shadow-idle">
      <p className="text-sm font-medium text-text">{title}</p>
      <p className="text-sm leading-6 text-text-muted">{message}</p>
    </div>
  );
}