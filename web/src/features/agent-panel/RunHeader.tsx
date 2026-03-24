import type { AgentRunSummary } from "@/shared/lib/workspace-protocol";
import { StateBadge } from "@/features/agent-panel/StateBadge";
import { getRunSummaryBadgeState } from "@/features/agent-panel/runStatus";

interface RunHeaderProps {
  run: AgentRunSummary | null;
}

export function RunHeader({ run }: RunHeaderProps) {
  if (!run) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="eyebrow">Live agent panel</p>
          <h2 className="mt-2 font-display text-2xl text-text">No run selected</h2>
        </div>
        <StateBadge state="accepted" />
      </div>
    );
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p className="eyebrow">Live agent panel</p>
        <h2 className="mt-2 font-display text-2xl capitalize text-text">{run.role}</h2>
        <p className="mt-2 text-sm text-text-muted">Model {run.model}</p>
      </div>
      <StateBadge state={getRunSummaryBadgeState(run)} />
    </div>
  );
}