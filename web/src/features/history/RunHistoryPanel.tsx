import type { AgentRunSummary, WorkspaceUIState } from "@/shared/lib/workspace-protocol";
import { RunHistoryListItem } from "@/features/history/RunHistoryListItem";

interface RunHistoryPanelProps {
  activeRunId: string;
  historyState: WorkspaceUIState["history_state"];
  runSummaries: AgentRunSummary[];
  selectedRunId: string;
  onOpen: (runId: string) => void;
}

export function RunHistoryPanel({ activeRunId, historyState, runSummaries, selectedRunId, onOpen }: RunHistoryPanelProps) {
  return (
    <section aria-labelledby="run-history-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
      <div>
        <p className="eyebrow">Run history</p>
        <h2 id="run-history-heading" className="mt-2 font-display text-2xl text-text">
          Saved runs
        </h2>
      </div>

      {historyState === "loading" ? <p className="mt-6 text-sm text-text-muted">Loading saved runs.</p> : null}
      {historyState === "error" ? <p className="mt-6 text-sm text-error">Relay could not load the saved run history.</p> : null}
      {historyState === "ready" && runSummaries.length === 0 ? (
        <p className="mt-6 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted">
          No saved runs yet. Completed, clarification-required, and errored agent tasks will appear here for replay.
        </p>
      ) : null}

      {runSummaries.length > 0 ? (
        <ul className="mt-6 space-y-3">
          {runSummaries.map((run) => (
            <RunHistoryListItem
              isActive={run.id === selectedRunId || run.id === activeRunId}
              key={run.id}
              onOpen={onOpen}
              run={run}
            />
          ))}
        </ul>
      ) : null}
    </section>
  );
}