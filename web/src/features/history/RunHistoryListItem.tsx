import clsx from "clsx";
import type { AgentRunSummary } from "@/shared/lib/workspace-protocol";

interface RunHistoryListItemProps {
  isActive: boolean;
  onOpen: (runId: string) => void;
  run: AgentRunSummary;
}

export function RunHistoryListItem({ isActive, onOpen, run }: RunHistoryListItemProps) {
  return (
    <li>
      <button
        className={clsx(
          "w-full rounded-3xl border px-4 py-4 text-left transition duration-300 ease-relay",
          isActive ? "border-brand-mid bg-raised text-text" : "border-border bg-raised/60 text-text-muted hover:border-brand-dim hover:text-text",
        )}
        onClick={() => onOpen(run.id)}
        type="button"
      >
        <span className="eyebrow">{run.role}</span>
        <span className="mt-2 block text-sm leading-6">{run.task_text_preview}</span>
        <span className="mt-3 block text-xs uppercase tracking-[0.18em] text-text-muted">{run.state} • {run.model}</span>
      </button>
    </li>
  );
}