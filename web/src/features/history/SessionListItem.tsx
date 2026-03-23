"use client";

import clsx from "clsx";
import type { SessionSummary } from "@/shared/lib/workspace-protocol";

interface SessionListItemProps {
  isActive: boolean;
  onOpen: (sessionId: string) => void;
  session: SessionSummary;
}

export function SessionListItem({ isActive, onOpen, session }: SessionListItemProps) {
  return (
    <li>
      <button
        aria-current={isActive ? "page" : undefined}
        aria-label={`Open session: ${session.display_name}`}
        className={clsx(
          "w-full rounded-2xl border px-4 py-3 text-left transition duration-300 ease-relay",
          isActive
            ? "border-brand-mid bg-raised shadow-thinking"
            : "border-border bg-surface hover:border-brand-dim hover:bg-raised",
        )}
        onClick={() => onOpen(session.id)}
        type="button"
      >
        <span className="block font-medium text-text">{session.display_name}</span>
        <span className="mt-2 block font-mono text-xs text-text-muted">Last opened {new Date(session.last_opened_at).toLocaleString()}</span>
      </button>
    </li>
  );
}
