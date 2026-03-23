"use client";

import { startTransition } from "react";
import { WorkspaceCanvas } from "@/features/canvas/WorkspaceCanvas";
import { SessionSidebar } from "@/features/history/SessionSidebar";
import { PreferencesPanel } from "@/features/preferences/PreferencesPanel";
import { WorkspaceStatusBanner } from "@/features/workspace-shell/WorkspaceStatusBanner";
import { useWorkspaceSocket } from "@/shared/lib/useWorkspaceSocket";
import { useWorkspaceStore } from "@/shared/lib/workspace-store";

export function WorkspaceShell() {
  const { createSession, openSession, savePreferences } = useWorkspaceSocket();
  const connectionState = useWorkspaceStore((state) => state.connectionState);
  const activeSessionId = useWorkspaceStore((state) => state.activeSessionId);
  const sessions = useWorkspaceStore((state) => state.sessions);
  const preferences = useWorkspaceStore((state) => state.preferences);
  const uiState = useWorkspaceStore((state) => state.uiState);
  const status = useWorkspaceStore((state) => state.status);
  const error = useWorkspaceStore((state) => state.error);
  const warnings = useWorkspaceStore((state) => state.warnings);

  const activeSession = sessions.find((session) => session.id === activeSessionId) ?? null;

  return (
    <div className="mx-auto flex min-h-screen w-full max-w-[96rem] flex-col px-4 py-4 md:px-6 md:py-6">
      <header className="panel-surface rounded-[2rem] px-5 py-5 shadow-idle">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="eyebrow">Relay workspace</p>
            <h1 className="font-display text-4xl text-text 
               -mt-[0.1em] pt-[0.1em] leading-[1.2]">Local AI session control, without leaving localhost.</h1>
          </div>
          <div className="rounded-3xl border border-border bg-raised/80 px-4 py-3 text-right">
            <p className="eyebrow">Saved preference</p>
            <p className="mt-2 font-mono text-sm text-text">Port {preferences.preferred_port}</p>
            <p className="mt-1 text-sm text-text-muted">Theme {preferences.appearance_variant}</p>
          </div>
        </div>
      </header>

      <div className="workspace-grid mt-4 grid flex-1 gap-4 lg:grid-cols-[22rem_minmax(0,1fr)]">
        <aside>
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
              })
            }
            sessions={sessions}
          />
        </aside>

        <section className="grid gap-4">
          <WorkspaceStatusBanner connectionState={connectionState} error={error} status={status} warnings={warnings} />
          <WorkspaceCanvas activeSession={activeSession} />
          <PreferencesPanel
            onSave={(payload) =>
              startTransition(() => {
                savePreferences(payload);
              })
            }
            preferences={preferences}
            saveState={uiState.save_state}
          />
        </section>
      </div>
    </div>
  );
}
