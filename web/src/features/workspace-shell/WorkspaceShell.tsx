"use client";

import { startTransition, useEffect, useState } from "react";
import { AgentPanel } from "@/features/agent-panel/AgentPanel";
import { WorkspaceCanvas } from "@/features/canvas/WorkspaceCanvas";
import { RunHistoryPanel } from "@/features/history/RunHistoryPanel";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { PreferencesPanel } from "@/features/preferences/PreferencesPanel";
import {
  hasWorkspaceStatusBanner,
  WorkspaceStatusBanner,
} from "@/features/workspace-shell/WorkspaceStatusBanner";
import { WorkspaceUtilityDrawer } from "@/features/workspace-shell/WorkspaceUtilityDrawer";
import { useWorkspaceSocket } from "@/shared/lib/useWorkspaceSocket";
import { useWorkspaceStore } from "@/shared/lib/workspace-store";

export function WorkspaceShell() {
  const [menuOpen, setMenuOpen] = useState(false);
  const {
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
  const runEvents = useWorkspaceStore((state) => state.runEvents);
  const runTranscripts = useWorkspaceStore((state) => state.runTranscripts);
  const pendingApprovals = useWorkspaceStore((state) => state.pendingApprovals);
  const preferences = useWorkspaceStore((state) => state.preferences);
  const uiState = useWorkspaceStore((state) => state.uiState);
  const status = useWorkspaceStore((state) => state.status);
  const error = useWorkspaceStore((state) => state.error);
  const warnings = useWorkspaceStore((state) => state.warnings);

  const activeSession = sessions.find((session) => session.id === activeSessionId) ?? null;
  const selectedRunSummary =
    runSummaries.find((run) => run.id === selectedRunId) ?? null;
  const selectedRunEvents = selectedRunId
    ? (runEvents[selectedRunId] ?? [])
    : [];
  const selectedRunTranscript = selectedRunId
    ? (runTranscripts[selectedRunId] ?? "")
    : "";
  const selectedPendingApproval =
    Object.values(pendingApprovals).find(
      (approval) => approval.runId === selectedRunId,
    ) ?? null;
  const showHeaderStatus = hasWorkspaceStatusBanner({
    connectionState,
    status,
    error,
    projectRootMessage: preferences.project_root_message,
    projectRootValid: preferences.project_root_valid,
    warnings,
  });

  useEffect(() => {
    if (!menuOpen) {
      return;
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setMenuOpen(false);
      }
    }

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [menuOpen]);

  return (
    <div className="mx-auto flex min-h-screen w-full max-w-[112rem] flex-col px-4 py-4 md:px-6 md:py-6">
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
              className="rounded-full border border-brand-mid bg-raised px-5 py-3 text-sm font-medium text-text transition duration-200 hover:border-brand"
              onClick={() => setMenuOpen(true)}
              type="button"
            >
              Open workspace menu
            </button>
          </div>
        </div>
      </header>

      <main
        className="workspace-grid mt-4 grid flex-1 gap-4"
        id="maincontent"
        tabIndex={-1}
      >
        <section className="grid min-w-0 gap-4">
          <AgentPanel
            activeRunId={activeRunId}
            activeSessionId={activeSessionId}
            onCancel={(runId) =>
              startTransition(() => {
                cancelRun(activeSessionId, runId);
              })
            }
            preferences={preferences}
            pendingApproval={selectedPendingApproval}
            runEvents={selectedRunEvents}
            runTranscript={selectedRunTranscript}
            selectedRunId={selectedRunId}
            selectedRunSummary={selectedRunSummary}
            onApprovalDecision={(toolCallId, decision) =>
              startTransition(() => {
                respondToApproval(
                  activeSessionId,
                  selectedRunId,
                  toolCallId,
                  decision,
                );
              })
            }
            onSubmit={(task) =>
              startTransition(() => {
                submitRun(activeSessionId, task);
              })
            }
          />
          <WorkspaceCanvas activeSession={activeSession} />
        </section>
      </main>

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
              </div>
            </div>
          </section>
          <PreferencesPanel
            onSave={(payload) =>
              startTransition(() => {
                savePreferences(payload);
              })
            }
            preferences={preferences}
            saveState={uiState.save_state}
          />
        </div>
      </WorkspaceUtilityDrawer>
    </div>
  );
}
