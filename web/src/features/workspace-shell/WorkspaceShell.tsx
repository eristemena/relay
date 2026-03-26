"use client";

import { startTransition, useEffect, useState } from "react";
import { AgentCommandBar } from "@/features/agent-panel/AgentCommandBar";
import { ApprovalReviewPanel } from "@/features/approvals/ApprovalReviewPanel";
import { WorkspaceCanvas } from "@/features/canvas/WorkspaceCanvas";
import { RunHistoryPanel } from "@/features/history/RunHistoryPanel";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { PreferencesPanel } from "@/features/preferences/PreferencesPanel";
import { FormattedMarkdown } from "@/shared/lib/FormattedMarkdown";
import {
  hasWorkspaceStatusBanner,
  WorkspaceStatusBanner,
} from "@/features/workspace-shell/WorkspaceStatusBanner";
import { WorkspaceUtilityDrawer } from "./WorkspaceUtilityDrawer";
import { useWorkspaceSocket } from "@/shared/lib/useWorkspaceSocket";
import { useWorkspaceStore } from "@/shared/lib/workspace-store";

export function WorkspaceShell() {
  const [menuOpen, setMenuOpen] = useState(false);
  const [approvalOpen, setApprovalOpen] = useState(false);
  const {
    browseRepository,
    createSession,
    openRun,
    openSession,
    respondToApproval,
    savePreferences,
    submitRun,
    cancelRun,
  } = useWorkspaceSocket();
  const connectionState = useWorkspaceStore((state) => state.connectionState);
  const activeSessionId = useWorkspaceStore((state) => state.activeSessionId);
  const activeRunId = useWorkspaceStore((state) => state.activeRunId);
  const selectedRunId = useWorkspaceStore((state) => state.selectedRunId);
  const sessions = useWorkspaceStore((state) => state.sessions);
  const runSummaries = useWorkspaceStore((state) => state.runSummaries);
  const pendingApprovals = useWorkspaceStore((state) => state.pendingApprovals);
  const orchestrationDocuments = useWorkspaceStore(
    (state) => state.orchestrationDocuments,
  );
  const repositoryBrowser = useWorkspaceStore(
    (state) => state.repositoryBrowser,
  );
  const preferences = useWorkspaceStore((state) => state.preferences);
  const uiState = useWorkspaceStore((state) => state.uiState);
  const status = useWorkspaceStore((state) => state.status);
  const error = useWorkspaceStore((state) => state.error);
  const warnings = useWorkspaceStore((state) => state.warnings);

  const activeSession =
    sessions.find((session) => session.id === activeSessionId) ?? null;
  const visibleRunId = activeRunId || selectedRunId;
  const visibleRunDocument = visibleRunId
    ? (orchestrationDocuments[visibleRunId] ?? null)
    : null;
  const approvalRunId = selectedRunId || activeRunId;
  const selectedPendingApproval =
    Object.values(pendingApprovals).find(
      (approval) => approval.runId === approvalRunId,
    ) ??
    Object.values(pendingApprovals).find(
      (approval) => approval.status === "proposed",
    ) ??
    Object.values(pendingApprovals)[0] ??
    null;
  const showHeaderStatus = hasWorkspaceStatusBanner({
    connectionState,
    status,
    error,
    projectRootConfigured: preferences.project_root_configured,
    projectRootMessage: preferences.project_root_message,
    projectRootValid: preferences.project_root_valid,
    warnings,
  });
  const repositorySummaryTitle = preferences.project_root_valid
    ? "Repository connected"
    : preferences.project_root_configured
      ? "Repository needs attention"
      : "Repository not connected";
  const repositorySummaryMessage = preferences.project_root_valid
    ? preferences.project_root
    : preferences.project_root_message ||
      "Choose a local Git repository in Local settings to enable repository-aware tools.";
  const pendingApprovalCount = Object.keys(pendingApprovals).length;

  useEffect(() => {
    if (selectedPendingApproval) {
      setApprovalOpen(true);
    }
  }, [selectedPendingApproval]);

  useEffect(() => {
    if (!menuOpen && !approvalOpen) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") {
        return;
      }

      if (approvalOpen) {
        setApprovalOpen(false);
        return;
      }

      setMenuOpen(false);
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [approvalOpen, menuOpen]);

  const commandBarDisabled = !activeSessionId || Boolean(activeRunId);

  return (
    <div className="mx-auto flex h-[100dvh] w-full max-w-[120rem] flex-col overflow-hidden px-4 py-4 md:px-6 md:py-6">
      <header className="panel-surface rounded-[2rem] px-5 py-5 shadow-idle">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="eyebrow">Relay workspace</p>
            <h1
              className="font-display text-4xl text-text 
               -mt-[0.1em] pt-[0.1em] leading-[1.2]"
            >
              Local AI session control, without leaving localhost.
            </h1>
          </div>
          <div className="flex flex-wrap items-start justify-end gap-3">
            {showHeaderStatus ? (
              <div className="workspace-header-cluster rounded-3xl border border-border bg-raised/80 px-4 py-3">
                <WorkspaceStatusBanner
                  compact
                  embedded
                  connectionState={connectionState}
                  error={error}
                  projectRootConfigured={preferences.project_root_configured}
                  projectRootMessage={preferences.project_root_message}
                  projectRootValid={preferences.project_root_valid}
                  status={status}
                  warnings={warnings}
                />
              </div>
            ) : null}
            <button
              aria-controls="workspace-utility-heading"
              aria-expanded={menuOpen}
              className="workspace-menu-trigger"
              onClick={() => setMenuOpen(true)}
              type="button"
            >
              <span className="sr-only">Open workspace menu</span>
              <svg
                aria-hidden="true"
                fill="none"
                height="20"
                viewBox="0 0 24 24"
                width="20"
              >
                <path
                  d="M4 7H20M4 12H20M4 17H14"
                  stroke="currentColor"
                  strokeLinecap="round"
                  strokeWidth="1.8"
                />
              </svg>
              {pendingApprovalCount > 0 ? (
                <span
                  className="workspace-menu-trigger-count"
                  aria-hidden="true"
                >
                  {pendingApprovalCount}
                </span>
              ) : null}
            </button>
          </div>
        </div>
      </header>

      <main
        className="mt-4 grid min-h-0 flex-1 gap-4 overflow-hidden"
        id="maincontent"
        tabIndex={-1}
      >
        <section className="grid min-h-0 min-w-0 gap-4 overflow-hidden">
          <WorkspaceCanvas activeSession={activeSession} />
        </section>
      </main>

      <div className="workspace-task-dock" role="presentation">
        <section
          aria-label="Task composer"
          className="workspace-task-dock-inner"
        >
          <AgentCommandBar
            disabled={commandBarDisabled}
            hasActiveRun={Boolean(activeRunId)}
            onCancel={() =>
              startTransition(() => {
                if (!activeSessionId || !activeRunId) {
                  return;
                }

                cancelRun(activeSessionId, activeRunId);
              })
            }
            onSubmit={(task) =>
              startTransition(() => {
                submitRun(activeSessionId, task);
              })
            }
            panelClassName="workspace-task-dock-card"
          />
        </section>
      </div>

      <WorkspaceUtilityDrawer
        onClose={() => setMenuOpen(false)}
        open={menuOpen}
      >
        <div className="grid gap-4 p-5">
          <nav aria-label="Session history and switching">
            <SessionSidebar
              activeSessionId={activeSessionId}
              onCreate={() =>
                startTransition(() => {
                  createSession();
                })
              }
              onOpen={(sessionId) =>
                startTransition(() => {
                  openSession(sessionId);
                  setMenuOpen(false);
                })
              }
              sessions={sessions}
            />
          </nav>
          <RunHistoryPanel
            activeRunId={activeRunId}
            historyState={uiState.history_state}
            runSummaries={runSummaries}
            selectedRunId={selectedRunId}
            onOpen={(runId: string) =>
              startTransition(() => {
                openRun(activeSessionId, runId);
                setMenuOpen(false);
              })
            }
          />
          <section
            aria-labelledby="workspace-run-summary-heading"
            className="panel-surface rounded-[2rem] p-5 shadow-idle"
          >
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <p className="eyebrow">Run summary</p>
                <h2
                  id="workspace-run-summary-heading"
                  className="mt-2 font-display text-2xl text-text"
                >
                  Latest orchestration recap
                </h2>
              </div>
              {visibleRunId ? (
                <p className="font-mono text-xs uppercase tracking-[0.2em] text-text-muted">
                  {visibleRunId}
                </p>
              ) : null}
            </div>

            {visibleRunDocument?.runSummary ? (
              <div className="mt-4 rounded-[1.25rem] border border-border bg-raised/80 p-4 text-sm leading-6 text-text">
                <FormattedMarkdown content={visibleRunDocument.runSummary} />
              </div>
            ) : (
              <p className="mt-4 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted">
                Replay or complete a run to capture the orchestration summary
                here for quick reference.
              </p>
            )}
          </section>
          <section
            aria-labelledby="workspace-summary-heading"
            className="panel-surface rounded-[2rem] p-5 shadow-idle"
          >
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <p className="eyebrow">Workspace summary</p>
                <h2
                  id="workspace-summary-heading"
                  className="mt-2 font-display text-2xl text-text"
                >
                  Saved workspace defaults
                </h2>
              </div>
              <div className="flex flex-wrap justify-end gap-x-4 gap-y-1 text-sm">
                <p className="font-mono text-text">
                  Port {preferences.preferred_port}
                </p>
                <p className="text-text-muted">
                  Theme {preferences.appearance_variant}
                </p>
                <p className="text-text-muted">{repositorySummaryTitle}</p>
              </div>
            </div>
            <p className="mt-4 break-all text-sm leading-6 text-text-muted">
              {repositorySummaryMessage}
            </p>
          </section>
          <PreferencesPanel
            onBrowseRepository={(path, showHidden) =>
              startTransition(() => {
                browseRepository(path, showHidden);
              })
            }
            onSave={(payload) =>
              startTransition(() => {
                savePreferences(payload);
              })
            }
            preferences={preferences}
            repositoryBrowser={repositoryBrowser}
            saveState={uiState.save_state}
          />
        </div>
      </WorkspaceUtilityDrawer>

      {approvalOpen && pendingApprovalCount > 0 ? (
        <div className="workspace-approval-shell" data-state="open">
          <button
            aria-label="Close approval review"
            className="workspace-approval-backdrop"
            onClick={() => setApprovalOpen(false)}
            type="button"
          />
          <aside
            aria-labelledby="approval-review-heading"
            aria-modal="true"
            className="workspace-approval-dialog"
            role="dialog"
          >
            <div className="workspace-approval-header">
              <div>
                <p className="eyebrow">Action required</p>
                <h2
                  id="approval-review-heading"
                  className="mt-2 font-display text-2xl text-text"
                >
                  Relay is waiting for approval
                </h2>
              </div>
              <button
                className="rounded-full border border-border bg-raised px-4 py-2 text-sm text-text"
                onClick={() => setApprovalOpen(false)}
                type="button"
              >
                Close
              </button>
            </div>
            <div className="workspace-approval-scroll">
              <ApprovalReviewPanel
                approval={selectedPendingApproval}
                pendingCount={pendingApprovalCount}
                selectedApprovalId={selectedPendingApproval?.toolCallId}
                onApprovalDecision={(toolCallId, decision) =>
                  startTransition(() => {
                    respondToApproval(
                      activeSessionId,
                      selectedRunId,
                      toolCallId,
                      decision,
                    );
                    setApprovalOpen(false);
                  })
                }
              />
            </div>
          </aside>
        </div>
      ) : null}
    </div>
  );
}
