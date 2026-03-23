"use client";

import { useDeferredValue } from "react";
import type { SessionSummary } from "@/shared/lib/workspace-protocol";
import { NewSessionButton } from "@/features/history/NewSessionButton";
import { SessionListItem } from "@/features/history/SessionListItem";

interface SessionSidebarProps {
  activeSessionId: string;
  onCreate: () => void;
  onOpen: (sessionId: string) => void;
  sessions: SessionSummary[];
}

export function SessionSidebar({ activeSessionId, onCreate, onOpen, sessions }: SessionSidebarProps) {
  const deferredSessions = useDeferredValue(sessions);

  return (
    <section aria-labelledby="session-history-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="eyebrow">History</p>
          <h2 id="session-history-heading" className="mt-2 font-display text-2xl text-text">
            Sessions
          </h2>
        </div>
        <NewSessionButton onCreate={onCreate} />
      </div>

      {deferredSessions.length === 0 ? (
        <div className="mt-6 rounded-3xl border border-dashed border-border bg-raised/60 p-5">
          <p className="text-sm leading-6 text-text-muted">
            No saved sessions yet. Start a new session to seed the sidebar and keep it available after the next restart.
          </p>
        </div>
      ) : (
        <ul className="mt-6 space-y-3">
          {deferredSessions.map((session) => (
            <SessionListItem
              isActive={session.id === activeSessionId}
              key={session.id}
              onOpen={onOpen}
              session={session}
            />
          ))}
        </ul>
      )}
    </section>
  );
}
