"use client";

import { startTransition, useEffect, useMemo, useState } from "react";
import { AgentCommandBar } from "@/features/agent-panel/AgentCommandBar";
import { ApprovalReviewPanel } from "@/features/approvals/ApprovalReviewPanel";
import { WorkspaceCanvas } from "@/features/canvas/WorkspaceCanvas";
import { getSelectedCanvasNode } from "@/features/canvas/canvasModel";
import {
  WorkspaceCanvasToolbar,
  type WorkspaceCanvasPanelId,
} from "@/features/canvas/WorkspaceCanvasToolbar";
import { RepositoryFileTreePanel } from "@/features/history/RepositoryFileTreePanel";
import { RunHistoryPanel } from "@/features/history/RunHistoryPanel";
import { SidebarTabs } from "@/features/history/SidebarTabs";
import { ReplayDock } from "@/features/history/replay/ReplayDock";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { PreferencesPanel } from "@/features/preferences/PreferencesPanel";
import { FormattedMarkdown } from "@/shared/lib/FormattedMarkdown";
import type { AgentRunSummary } from "@/shared/lib/workspace-protocol";
import {
  hasWorkspaceStatusBanner,
  WorkspaceStatusBanner,
} from "@/features/workspace-shell/WorkspaceStatusBanner";
import { useWorkspaceSocket } from "@/shared/lib/useWorkspaceSocket";
import {
  clearWorkspaceCanvasSelection,
  setWorkspaceRepositoryTreeActiveTab,
  toggleWorkspaceRepositoryTreePath,
  useWorkspaceStore,
} from "@/shared/lib/workspace-store";

export function WorkspaceShell() {
  const [activePanel, setActivePanel] = useState<WorkspaceCanvasPanelId | null>(
    null,
  );
  const {
    browseRepository,
    controlReplay,
    exportRunHistory,
    getRunHistoryDetails,
    openRun,
    queryRunHistory,
    requestRepositoryTree,
    respondToApproval,
    savePreferences,
    switchProject,
    submitRun,
    cancelRun,
  } = useWorkspaceSocket();
  const connectionState = useWorkspaceStore((state) => state.connectionState);
  const activeProjectRoot = useWorkspaceStore(
    (state) => state.activeProjectRoot,
  );
  const activeSessionId = useWorkspaceStore((state) => state.activeSessionId);
  const activeRunId = useWorkspaceStore((state) => state.activeRunId);
  const selectedRunId = useWorkspaceStore((state) => state.selectedRunId);
  const knownProjects = useWorkspaceStore((state) => state.knownProjects);
  const sessions = useWorkspaceStore((state) => state.sessions);
  const runSummaries = useWorkspaceStore((state) => state.runSummaries);
  const pendingApprovals = useWorkspaceStore((state) => state.pendingApprovals);
  const runHistoryQuery = useWorkspaceStore((state) => state.runHistoryQuery);
  const runHistoryResults = useWorkspaceStore(
    (state) => state.runHistoryResults,
  );
  const runHistoryDetails = useWorkspaceStore(
    (state) => state.runHistoryDetails,
  );
  const replayStateByRunId = useWorkspaceStore(
    (state) => state.replayStateByRunId,
  );
  const exportStateByRunId = useWorkspaceStore(
    (state) => state.exportStateByRunId,
  );
  const orchestrationDocuments = useWorkspaceStore(
    (state) => state.orchestrationDocuments,
  );
  const repositoryBrowser = useWorkspaceStore(
    (state) => state.repositoryBrowser,
  );
  const connectedRepository = useWorkspaceStore(
    (state) => state.connectedRepository,
  );
  const repositoryTree = useWorkspaceStore((state) => state.repositoryTree);
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
  const historyRuns =
    runHistoryQuery || runHistoryResults.length > 0
      ? runHistoryResults
      : runSummaries;
  const selectedHistoryRun =
    historyRuns.find((run) => run.id === selectedRunId) ??
    historyRuns.find((run) => run.id === activeRunId) ??
    null;
  const selectedHistoryRunId = selectedHistoryRun?.id ?? "";
  const selectedHistoryRunDetails = selectedHistoryRunId
    ? (runHistoryDetails[selectedHistoryRunId] ?? null)
    : null;
  const selectedReplayState = selectedHistoryRunId
    ? (replayStateByRunId[selectedHistoryRunId] ?? null)
    : null;
  const selectedExportState = selectedHistoryRunId
    ? (exportStateByRunId[selectedHistoryRunId] ?? null)
    : null;
  const selectedCanvasNode = visibleRunDocument
    ? getSelectedCanvasNode(visibleRunDocument)
    : null;
  const repositoryTreeRunId = selectedHistoryRunId || activeRunId;
  const selectedRunIsLive = isLiveRunState(selectedHistoryRun?.state);
  const showRightRail = Boolean(activeSessionId);
  const showRepositoryTreeOnly = showRightRail && selectedRunIsLive;
  const showRightRailTabs = showRightRail && !selectedRunIsLive;
  const showInspectorPlaceholder = Boolean(
    activeSessionId && !selectedHistoryRun,
  );

  useEffect(() => {
    if (selectedPendingApproval) {
      setActivePanel("approvals");
    }
  }, [selectedPendingApproval]);

  useEffect(() => {
    if (activePanel !== "history" || !activeSessionId) {
      return;
    }

    startTransition(() => {
      queryRunHistory(activeSessionId, {
        all_projects: runHistoryQuery?.all_projects,
        query: runHistoryQuery?.query,
        file_path: runHistoryQuery?.file_path,
        date_from: runHistoryQuery?.date_from,
        date_to: runHistoryQuery?.date_to,
      });
    });
  }, [
    activePanel,
    activeSessionId,
    runHistoryQuery?.date_from,
    runHistoryQuery?.date_to,
    runHistoryQuery?.file_path,
    runHistoryQuery?.all_projects,
    runHistoryQuery?.query,
  ]);

  useEffect(() => {
    if (
      activePanel !== "history" ||
      !activeSessionId ||
      !selectedHistoryRunId ||
      selectedHistoryRunDetails
    ) {
      return;
    }

    startTransition(() => {
      getRunHistoryDetails(activeSessionId, selectedHistoryRunId);
    });
  }, [
    activePanel,
    activeSessionId,
    selectedHistoryRunDetails,
    selectedHistoryRunId,
  ]);

  useEffect(() => {
    if (
      connectionState !== "connected" ||
      !activeSessionId ||
      !repositoryTreeRunId ||
      connectedRepository.status !== "connected"
    ) {
      return;
    }

    if (
      !showRepositoryTreeOnly &&
      repositoryTree.activeTab !== "repository_tree"
    ) {
      return;
    }

    if (
      repositoryTree.requestRunId === repositoryTreeRunId &&
      (repositoryTree.status === "loading" ||
        repositoryTree.status === "ready" ||
        repositoryTree.status === "empty")
    ) {
      return;
    }

    startTransition(() => {
      requestRepositoryTree(activeSessionId, repositoryTreeRunId);
    });
  }, [
    activeSessionId,
    connectedRepository.status,
    connectionState,
    repositoryTreeRunId,
    repositoryTree.activeTab,
    repositoryTree.requestRunId,
    repositoryTree.status,
    requestRepositoryTree,
    showRepositoryTreeOnly,
  ]);

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
          <nav aria-label="Project context and switching">
            <SessionSidebar
              activeProjectRoot={activeProjectRoot}
              knownProjects={knownProjects}
              onOpenPreferences={() => setActivePanel("preferences")}
              onSwitch={(projectRoot) =>
                startTransition(() => {
                  switchProject(projectRoot);
                })
              }
            />
          </nav>
        );
      case "history":
        return (
          <RunHistoryPanel
            activeRunId={activeRunId}
            historyState={uiState.history_state}
            exportState={selectedExportState}
            onExport={() => {
              if (!activeSessionId || !selectedHistoryRunId) {
                return;
              }
              startTransition(() => {
                exportRunHistory(activeSessionId, selectedHistoryRunId);
              });
            }}
            selectedRunId={selectedRunId}
            onQuery={(payload) => {
              if (!activeSessionId) {
                return;
              }
              startTransition(() => {
                queryRunHistory(activeSessionId, payload);
              });
            }}
            onReplayControl={(payload) => {
              if (!activeSessionId || !selectedHistoryRunId) {
                return;
              }
              startTransition(() => {
                controlReplay({
                  session_id: activeSessionId,
                  run_id: selectedHistoryRunId,
                  action: payload.action,
                  cursor_ms: payload.cursor_ms,
                  speed: payload.speed,
                });
              });
            }}
            replayState={selectedReplayState}
            runHistoryQuery={runHistoryQuery}
            runSummaries={historyRuns}
            selectedRun={selectedHistoryRun}
            selectedRunDetails={selectedHistoryRunDetails}
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
    activeProjectRoot,
    activeRunId,
    activeSessionId,
    browseRepository,
    openRun,
    connectedRepository,
    knownProjects,
    pendingApprovalCount,
    preferences,
    repositoryBrowser,
    repositoryTree,
    repositorySummaryMessage,
    repositorySummaryTitle,
    respondToApproval,
    queryRunHistory,
    requestRepositoryTree,
    getRunHistoryDetails,
    controlReplay,
    exportRunHistory,
    runSummaries,
    runHistoryDetails,
    runHistoryQuery,
    runHistoryResults,
    savePreferences,
    selectedCanvasNode,
    selectedExportState,
    selectedPendingApproval,
    selectedReplayState,
    selectedHistoryRun,
    selectedHistoryRunDetails,
    selectedHistoryRunId,
    selectedRunId,
    toggleWorkspaceRepositoryTreePath,
    uiState.history_state,
    uiState.save_state,
    visibleRunDocument,
    visibleRunId,
  ]);
  const workspaceToolbar = (
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
  );
  const commandBarDisabled = !activeSessionId || Boolean(activeRunId);
  const repositoryTreePanel = (
    <RepositoryFileTreePanel
      connectedRepository={connectedRepository}
      onClearSelectedAgent={() => {
        if (!repositoryTreeRunId) {
          return;
        }
        clearWorkspaceCanvasSelection(repositoryTreeRunId);
      }}
      onRetry={() => {
        if (
          !activeSessionId ||
          !repositoryTreeRunId ||
          connectedRepository.status !== "connected"
        ) {
          return;
        }
        startTransition(() => {
          requestRepositoryTree(activeSessionId, repositoryTreeRunId);
        });
      }}
      onTogglePath={(path) => toggleWorkspaceRepositoryTreePath(path)}
      repositoryTree={repositoryTree}
      selectedAgentId={selectedCanvasNode?.id ?? null}
      selectedAgentLabel={selectedCanvasNode?.label ?? null}
    />
  );

  return (
    <div className="mx-auto flex h-[100dvh] w-full max-w-[120rem] flex-col overflow-hidden px-4 py-4 md:px-6 md:py-6">
      <header className="panel-surface rounded-[1.5rem] px-4 py-3 shadow-idle md:px-5 md:py-3.5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="min-w-0 flex-1">
            <p className="eyebrow">Relay workspace</p>
            <h1 className="mt-1 font-display text-xl text-text leading-[1.15] md:text-[1.65rem]">
              Local AI session control, without leaving localhost.
            </h1>
          </div>
          <div className="flex min-w-0 flex-wrap items-center justify-end gap-2.5 md:gap-3">
            <section
              aria-label="Active project"
              className="workspace-header-cluster rounded-[1.25rem] border border-border bg-raised/80 px-3 py-2.5"
            >
              <p className="eyebrow">Active project</p>
              <p
                className="mt-1 max-w-[24rem] overflow-hidden text-sm leading-5 text-text md:max-w-[28rem]"
                title={activeProjectRoot || "No active project selected yet."}
              >
                {activeProjectRoot || "No active project selected yet."}
              </p>
              <p className="mt-1 text-[0.68rem] uppercase tracking-[0.18em] text-text-muted">
                {knownProjects.length <= 1
                  ? "Single known project"
                  : `${knownProjects.length} known projects`}
              </p>
            </section>
            {showHeaderStatus ? (
              <div className="workspace-header-cluster rounded-[1.25rem] border border-border bg-raised/80 px-3 py-2.5">
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
        <section
          className={
            showRightRail
              ? "grid min-h-0 min-w-0 gap-4 overflow-hidden xl:grid-cols-[minmax(0,1.65fr)_minmax(24rem,0.75fr)]"
              : "grid min-h-0 min-w-0 gap-4 overflow-hidden"
          }
        >
          <div className="flex min-h-0 min-w-0 flex-col gap-4 overflow-hidden">
            <div className="flex-1 min-h-0 overflow-hidden">
              <WorkspaceCanvas
                activeSession={activeSession}
                workspaceToolbar={workspaceToolbar}
              />
            </div>

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
          {showRepositoryTreeOnly && selectedHistoryRun ? (
            repositoryTreePanel
          ) : showRightRailTabs && selectedHistoryRun ? (
            <section className="flex min-h-0 flex-col gap-4">
              <SidebarTabs
                activeTab={repositoryTree.activeTab}
                onChange={(tab) => setWorkspaceRepositoryTreeActiveTab(tab)}
              />
              {repositoryTree.activeTab === "replay" ? (
                <div
                  aria-labelledby="replay-tab"
                  className="min-h-0 flex-1"
                  id="replay-tabpanel"
                  role="tabpanel"
                >
                  <ReplayDock
                    exportState={selectedExportState}
                    onBrowseRuns={() => setActivePanel("history")}
                    onExport={() => {
                      if (!selectedHistoryRunId) {
                        return;
                      }
                      startTransition(() => {
                        exportRunHistory(activeSessionId, selectedHistoryRunId);
                      });
                    }}
                    onReplayControl={(payload) => {
                      if (!selectedHistoryRunId) {
                        return;
                      }
                      startTransition(() => {
                        controlReplay({
                          session_id: activeSessionId,
                          run_id: selectedHistoryRunId,
                          action: payload.action,
                          cursor_ms: payload.cursor_ms,
                          speed: payload.speed,
                        });
                      });
                    }}
                    replayState={selectedReplayState}
                    selectedRun={selectedHistoryRun}
                  />
                </div>
              ) : (
                <div
                  aria-labelledby="repository-tree-tab"
                  className="min-h-0 flex-1 overflow-hidden"
                  id="repository-tree-tabpanel"
                  role="tabpanel"
                >
                  {repositoryTreePanel}
                </div>
              )}
            </section>
          ) : showInspectorPlaceholder ? (
            <section className="flex min-h-0 flex-col gap-4">
              <SidebarTabs
                activeTab={repositoryTree.activeTab}
                disabled
                onChange={() => {}}
              />
              <div
                aria-labelledby={
                  repositoryTree.activeTab === "replay"
                    ? "replay-tab"
                    : "repository-tree-tab"
                }
                className="min-h-0 flex-1"
                id={
                  repositoryTree.activeTab === "replay"
                    ? "replay-tabpanel"
                    : "repository-tree-tabpanel"
                }
                role="tabpanel"
              >
                <WorkspaceInspectorPlaceholder
                  activeTab={repositoryTree.activeTab}
                  hasSavedRuns={runSummaries.length > 0}
                  onBrowseRuns={() => setActivePanel("history")}
                />
              </div>
            </section>
          ) : null}
        </section>
      </main>
    </div>
  );
}

function isLiveRunState(state: AgentRunSummary["state"] | undefined) {
  return (
    state === "accepted" ||
    state === "active" ||
    state === "thinking" ||
    state === "tool_running" ||
    state === "approval_required"
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

function WorkspaceInspectorPlaceholder({
  activeTab,
  hasSavedRuns,
  onBrowseRuns,
}: {
  activeTab: "replay" | "repository_tree";
  hasSavedRuns: boolean;
  onBrowseRuns: () => void;
}) {
  const isReplayTab = activeTab === "replay";

  return (
    <aside className="panel-surface flex min-h-0 flex-col rounded-[2rem] p-5 shadow-idle">
      <div>
        <p className="eyebrow">Workspace detail rail</p>
        <h2
          className="mt-2 font-display text-2xl text-text"
          id="workspace-inspector-placeholder-heading"
        >
          {isReplayTab ? "Historical replay" : "Repository tree"}
        </h2>
      </div>
      <div className="mt-4 space-y-4 text-sm leading-6 text-text-muted">
        {isReplayTab ? (
          <>
            <p>
              Historical replay appears here after you reopen a saved run from
              Run history.
            </p>
            <p>
              {hasSavedRuns
                ? "Choose a saved run to inspect its recorded timeline, replay controls, and export actions."
                : "Submit a goal first, then reopen the saved run from Run history when you want replay controls."}
            </p>
          </>
        ) : (
          <>
            <p>
              The File Tree browser appears here while a live run is active.
            </p>
            <p>
              Reopened saved runs can also show repository state in this rail
              after you select them from Run history.
            </p>
          </>
        )}
      </div>
      {hasSavedRuns ? (
        <div className="mt-6">
          <button
            className="rounded-full border border-border px-4 py-2 text-sm font-medium text-text hover:border-brand-mid hover:bg-raised"
            onClick={onBrowseRuns}
            type="button"
          >
            Browse saved runs
          </button>
        </div>
      ) : null}
    </aside>
  );
}
