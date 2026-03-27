"use client";

import { startTransition, useEffect, useMemo, useState } from "react";
import { AgentCommandBar } from "@/features/agent-panel/AgentCommandBar";
import { ApprovalReviewPanel } from "@/features/approvals/ApprovalReviewPanel";
import { WorkspaceCanvas } from "@/features/canvas/WorkspaceCanvas";
import {
  WorkspaceCanvasToolbar,
  type WorkspaceCanvasPanelId,
} from "@/features/canvas/WorkspaceCanvasToolbar";
import { RunHistoryPanel } from "@/features/history/RunHistoryPanel";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { PreferencesPanel } from "@/features/preferences/PreferencesPanel";
import { FormattedMarkdown } from "@/shared/lib/FormattedMarkdown";
import {
  hasWorkspaceStatusBanner,
  WorkspaceStatusBanner,
} from "@/features/workspace-shell/WorkspaceStatusBanner";
import { useWorkspaceSocket } from "@/shared/lib/useWorkspaceSocket";
import { useWorkspaceStore } from "@/shared/lib/workspace-store";

export function WorkspaceShell() {
  const [activePanel, setActivePanel] = useState<WorkspaceCanvasPanelId | null>(
    null,
  );
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
      setActivePanel("approvals");
    }
  }, [selectedPendingApproval]);

  useEffect(() => {
    if (!activePanel) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape") {
        return;
      }

      setActivePanel(null);
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [activePanel]);

  const expandedPanel =
    activePanel === "preferences" || activePanel === "approvals";
  const panelContent = useMemo(() => {
    switch (activePanel) {
      case "sessions":
        return (
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
                  setActivePanel(null);
                })
              }
              sessions={sessions}
            />
          </nav>
        );
      case "history":
        return (
          <RunHistoryPanel
            activeRunId={activeRunId}
            historyState={uiState.history_state}
            runSummaries={runSummaries}
            selectedRunId={selectedRunId}
            onOpen={(runId: string) =>
              startTransition(() => {
                openRun(activeSessionId, runId);
                setActivePanel(null);
              })
            }
          />
        );
      case "run-summary":
        return (
          <WorkspaceRunSummaryPanel
            runSummary={visibleRunDocument?.runSummary ?? null}
            visibleRunId={visibleRunId}
          />
        );
      case "workspace-summary":
        return (
          <WorkspaceSummaryPanel
            preferredPort={preferences.preferred_port}
            repositorySummaryMessage={repositorySummaryMessage}
            repositorySummaryTitle={repositorySummaryTitle}
            theme={preferences.appearance_variant}
          />
        );
      case "preferences":
        return (
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
        );
      case "approvals":
        return (
          <ApprovalReviewPanel
            approval={selectedPendingApproval}
            pendingCount={pendingApprovalCount}
            selectedApprovalId={selectedPendingApproval?.toolCallId}
            onApprovalDecision={(toolCallId, decision) =>
              startTransition(() => {
                respondToApproval(
                  activeSessionId,
                  selectedPendingApproval?.runId ||
                    selectedRunId ||
                    activeRunId,
                  toolCallId,
                  decision,
                );
                setActivePanel(null);
              })
            }
          />
        );
      default:
        return null;
    }
  }, [
    activePanel,
    activeRunId,
    activeSessionId,
    browseRepository,
    createSession,
    openRun,
    openSession,
    pendingApprovalCount,
    preferences,
    repositoryBrowser,
    repositorySummaryMessage,
    repositorySummaryTitle,
    respondToApproval,
    runSummaries,
    savePreferences,
    selectedPendingApproval,
    selectedRunId,
    sessions,
    uiState.history_state,
    uiState.save_state,
    visibleRunDocument,
    visibleRunId,
  ]);
  const workspaceToolbar = activeSession ? (
    <WorkspaceCanvasToolbar
      activePanel={activePanel}
      expandedPanel={expandedPanel}
      onClose={() => setActivePanel(null)}
      onToggle={(panelId) =>
        setActivePanel((currentPanel) =>
          currentPanel === panelId ? null : panelId,
        )
      }
      panelContent={panelContent}
      pendingApprovalCount={pendingApprovalCount}
    />
  ) : null;
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
          </div>
        </div>
      </header>

      <main
        className="mt-4 grid min-h-0 flex-1 gap-4 overflow-hidden"
        id="maincontent"
        tabIndex={-1}
      >
        <section className="grid min-h-0 min-w-0 gap-4 overflow-hidden">
          <WorkspaceCanvas
            activeSession={activeSession}
            workspaceToolbar={workspaceToolbar}
          />
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
    </div>
  );
}

function WorkspaceRunSummaryPanel({
  runSummary,
  visibleRunId,
}: {
  runSummary: string | null;
  visibleRunId: string;
}) {
  return (
    <section
      aria-labelledby="workspace-run-summary-heading"
      className="panel-surface rounded-[2rem] p-5 shadow-idle"
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="eyebrow">Run summary</p>
          <h2
            className="mt-2 font-display text-2xl text-text"
            id="workspace-run-summary-heading"
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

      {runSummary ? (
        <div className="mt-4 rounded-[1.25rem] border border-border bg-raised/80 p-4 text-sm leading-6 text-text">
          <FormattedMarkdown content={runSummary} />
        </div>
      ) : (
        <p className="mt-4 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted">
          Replay or complete a run to capture the orchestration summary here for
          quick reference.
        </p>
      )}
    </section>
  );
}

function WorkspaceSummaryPanel({
  preferredPort,
  repositorySummaryMessage,
  repositorySummaryTitle,
  theme,
}: {
  preferredPort: number;
  repositorySummaryMessage: string;
  repositorySummaryTitle: string;
  theme: string;
}) {
  return (
    <section
      aria-labelledby="workspace-summary-heading"
      className="panel-surface rounded-[2rem] p-5 shadow-idle"
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="eyebrow">Workspace summary</p>
          <h2
            className="mt-2 font-display text-2xl text-text"
            id="workspace-summary-heading"
          >
            Saved workspace defaults
          </h2>
        </div>
        <div className="flex flex-wrap justify-end gap-x-4 gap-y-1 text-sm">
          <p className="font-mono text-text">Port {preferredPort}</p>
          <p className="text-text-muted">Theme {theme}</p>
          <p className="text-text-muted">{repositorySummaryTitle}</p>
        </div>
      </div>
      <p className="mt-4 break-all text-sm leading-6 text-text-muted">
        {repositorySummaryMessage}
      </p>
    </section>
  );
}
