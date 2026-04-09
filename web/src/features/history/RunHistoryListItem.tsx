import clsx from "clsx";
import type { AgentRunSummary } from "@/shared/lib/workspace-protocol";
import { StateBadge } from "@/features/agent-panel/StateBadge";
import {
  getRunSummaryBadgeState,
  getRunSummaryStateLabel,
} from "@/features/agent-panel/runStatus";

interface RunHistoryListItemProps {
  isActive: boolean;
  onOpen: (runId: string) => void;
  run: AgentRunSummary;
}

export function RunHistoryListItem({ isActive, onOpen, run }: RunHistoryListItemProps) {
  const projectLabel = run.project_label || run.project_root;

  return (
    <li>
      <button
        className={clsx(
          "w-full rounded-3xl border px-4 py-4 text-left transition duration-300 ease-relay",
          isActive
            ? "border-brand-mid bg-raised text-text"
            : "border-border bg-raised/60 text-text-muted hover:border-brand-dim hover:text-text",
        )}
        onClick={() => onOpen(run.id)}
        type="button"
      >
        <div className="flex items-start justify-between gap-3">
          <span className="eyebrow">{run.role}</span>
          <StateBadge state={getRunSummaryBadgeState(run)} />
        </div>
        <span className="mt-2 block text-sm leading-6">
          {run.task_text_preview}
        </span>
        {projectLabel ? (
          <span className="mt-3 block text-xs uppercase tracking-[0.18em] text-text-muted">
            Project {projectLabel}
          </span>
        ) : null}
        <span className="mt-3 block text-xs uppercase tracking-[0.18em] text-text-muted">
          {getRunSummaryStateLabel(run)} • {run.model}
        </span>
      </button>
    </li>
  );
}