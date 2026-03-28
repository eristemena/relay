"use client";

import { useEffect, useState } from "react";
import { describeRunSummaryReplayBanner } from "@/features/agent-panel/runStatus";
import { RunHistoryListItem } from "@/features/history/RunHistoryListItem";
import { RunChangeReviewPanel } from "@/features/history/replay/RunChangeReviewPanel";
import { RunHistoryFilters } from "@/features/history/replay/RunHistoryFilters";
import type {
  AgentRunReplayStatePayload,
  AgentRunSummary,
  RunHistoryDetailsResultPayload,
  RunHistoryExportResultPayload,
  RunHistoryQueryPayload,
  WorkspaceUIState,
} from "@/shared/lib/workspace-protocol";

interface RunHistoryPanelProps {
  activeRunId: string;
  historyState: WorkspaceUIState["history_state"];
  runSummaries: AgentRunSummary[];
  selectedRunId: string;
  selectedRun?: AgentRunSummary | null;
  runHistoryQuery?: RunHistoryQueryPayload | null;
  selectedRunDetails?: RunHistoryDetailsResultPayload | null;
  replayState?: AgentRunReplayStatePayload | null;
  exportState?: RunHistoryExportResultPayload | null;
  onOpen: (runId: string) => void;
  onQuery: (payload: Omit<RunHistoryQueryPayload, "session_id">) => void;
  onReplayControl: (payload: {
    action: "play" | "pause" | "seek" | "set_speed" | "reset";
    cursor_ms?: number;
    speed?: 0.5 | 1 | 2 | 5;
  }) => void;
  onExport: () => void;
}

export function RunHistoryPanel({
  activeRunId,
  historyState,
  runSummaries,
  selectedRunId,
  selectedRun,
  runHistoryQuery,
  selectedRunDetails,
  replayState,
  onOpen,
  onQuery,
}: RunHistoryPanelProps) {
  const [query, setQuery] = useState(runHistoryQuery?.query ?? "");
  const [filePath, setFilePath] = useState(runHistoryQuery?.file_path ?? "");
  const [dateFrom, setDateFrom] = useState(runHistoryQuery?.date_from ?? "");
  const [dateTo, setDateTo] = useState(runHistoryQuery?.date_to ?? "");

  useEffect(() => {
    setQuery(runHistoryQuery?.query ?? "");
    setFilePath(runHistoryQuery?.file_path ?? "");
    setDateFrom(runHistoryQuery?.date_from ?? "");
    setDateTo(runHistoryQuery?.date_to ?? "");
  }, [
    runHistoryQuery?.date_from,
    runHistoryQuery?.date_to,
    runHistoryQuery?.file_path,
    runHistoryQuery?.query,
  ]);

  const changeRecords = selectedRunDetails?.change_records ?? [];

  const filteredChangeRecords = selectedRunDetails
    ? filterChangeRecordsForReplay(
        changeRecords,
        replayState?.selected_timestamp,
      )
    : [];

  const changeReviewState = !selectedRun
    ? "empty"
    : selectedRunDetails
      ? changeRecords.length > 0
        ? "ready"
        : "empty"
      : "loading";
  const selectedTimestamp = replayState?.selected_timestamp
    ? new Date(replayState.selected_timestamp).toLocaleString()
    : null;

  return (
    <section aria-labelledby="run-history-heading" className="space-y-4">
      <div className="panel-surface rounded-[2rem] p-5 shadow-idle">
        <div>
          <p className="eyebrow">Run history</p>
          <h2
            id="run-history-heading"
            className="mt-2 font-display text-2xl text-text"
          >
            Saved runs
          </h2>
        </div>
        <p className="mt-4 text-sm leading-6 text-text-muted">
          Review preserved runs in read-only mode, search by transcript or path,
          and drive replay from the recorded timeline.
        </p>
      </div>

      <RunHistoryFilters
        dateFrom={dateFrom}
        dateTo={dateTo}
        filePath={filePath}
        onApply={() =>
          onQuery({
            query: query || undefined,
            file_path: filePath || undefined,
            date_from: dateFrom || undefined,
            date_to: dateTo || undefined,
          })
        }
        onDateFromChange={setDateFrom}
        onDateToChange={setDateTo}
        onFilePathChange={setFilePath}
        onQueryChange={setQuery}
        onReset={() => {
          setQuery("");
          setFilePath("");
          setDateFrom("");
          setDateTo("");
          onQuery({});
        }}
        query={query}
      />

      <div className="grid gap-4 xl:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
        <section className="panel-surface rounded-[2rem] p-5 shadow-idle">
          {historyState === "loading" ? (
            <p className="mt-2 text-sm text-text-muted">Loading saved runs.</p>
          ) : null}
          {historyState === "error" ? (
            <p className="mt-2 text-sm text-error">
              Relay could not load the saved run history.
            </p>
          ) : null}
          {historyState === "ready" && runSummaries.length === 0 ? (
            <p className="mt-4 rounded-3xl border border-dashed border-border bg-raised/60 p-5 text-sm leading-6 text-text-muted">
              No saved runs yet. Completed, clarification-required, and errored
              agent tasks will appear here for replay.
            </p>
          ) : null}
          {runSummaries.length > 0 ? (
            <ul className="mt-2 space-y-3">
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

        <div className="space-y-4">
          <section className="panel-surface rounded-[2rem] p-5 shadow-idle">
            <div className="flex flex-wrap items-start justify-between gap-4">
              <div>
                <p className="eyebrow">Selected run</p>
                <h3 className="mt-2 font-display text-2xl text-text">
                  {selectedRun?.generated_title ||
                    selectedRun?.task_text_preview ||
                    "Choose a saved run"}
                </h3>
              </div>
              {selectedRun ? (
                <p className="rounded-full border border-border px-3 py-1 text-xs uppercase tracking-[0.18em] text-text-muted">
                  {selectedRun.final_status || selectedRun.state}
                </p>
              ) : null}
            </div>
            {selectedRun ? (
              <>
                <p className="mt-4 text-sm leading-6 text-text-muted">
                  {describeRunSummaryReplayBanner(selectedRun)}
                </p>
                <div className="mt-4 grid gap-3 text-sm text-text-muted md:grid-cols-1">
                  <p>
                    Timestamp:{" "}
                    <span className="text-text">
                      {selectedTimestamp || "Not selected"}
                    </span>
                  </p>
                </div>
              </>
            ) : (
              <p className="mt-4 text-sm leading-6 text-text-muted">
                Choose a saved run to inspect its recorded timeline, metadata,
                and preserved file changes.
              </p>
            )}
          </section>

          <RunChangeReviewPanel
            changeRecords={filteredChangeRecords}
            state={changeReviewState}
            selectedTimestampLabel={selectedTimestamp}
            totalChangeCount={changeRecords.length}
          />
        </div>
      </div>
    </section>
  );
}

function filterChangeRecordsForReplay(
  changeRecords: RunHistoryDetailsResultPayload["change_records"] | undefined,
  selectedTimestamp?: string,
) {
  if (!changeRecords || changeRecords.length === 0) {
    return [];
  }

  if (!selectedTimestamp) {
    return changeRecords;
  }

  const selectedTime = Date.parse(selectedTimestamp);
  if (Number.isNaN(selectedTime)) {
    return changeRecords;
  }

  return changeRecords.filter((record) => {
    if (!record.occurred_at) {
      return true;
    }

    const occurredAt = Date.parse(record.occurred_at);
    return Number.isNaN(occurredAt) || occurredAt <= selectedTime;
  });
}
