import { ReplayControls } from "@/features/history/replay/ReplayControls";
import { ReplayTimeline } from "@/features/history/replay/ReplayTimeline";
import type {
  AgentRunReplayStatePayload,
  AgentRunSummary,
  RunHistoryExportResultPayload,
} from "@/shared/lib/workspace-protocol";

interface ReplayDockProps {
  exportState?: RunHistoryExportResultPayload | null;
  onBrowseRuns: () => void;
  onExport: () => void;
  onReplayControl: (payload: {
    action: "play" | "pause" | "seek" | "set_speed" | "reset";
    cursor_ms?: number;
    speed?: 0.5 | 1 | 2 | 5;
  }) => void;
  replayState?: AgentRunReplayStatePayload | null;
  selectedRun: AgentRunSummary;
}

export function ReplayDock({
  exportState,
  onBrowseRuns,
  onExport,
  onReplayControl,
  replayState,
  selectedRun,
}: ReplayDockProps) {
  const selectedTimestamp = replayState?.selected_timestamp
    ? new Date(replayState.selected_timestamp).toLocaleString()
    : null;
  const replayStatus = replayState?.status ?? "paused";

  return (
    <section
      aria-labelledby="replay-dock-heading"
      className="workspace-replay-dock panel-surface flex h-full min-h-0 flex-col rounded-[2rem] p-5 shadow-idle"
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="eyebrow">Historical replay</p>
          <h2
            className="mt-2 font-display text-2xl text-text"
            id="replay-dock-heading"
          >
            {selectedRun.generated_title || selectedRun.task_text_preview}
          </h2>
        </div>
        <button
          className="replay-control-button"
          onClick={onBrowseRuns}
          type="button"
        >
          Browse runs
        </button>
      </div>

      <div className="mt-4 grid gap-3 text-sm text-text-muted sm:grid-cols-2">
        <p>
          Status: <span className="text-text">{replayStatus}</span>
        </p>
        <p>
          Cursor: <span className="text-text">{selectedTimestamp || "Not selected"}</span>
        </p>
      </div>

      <div className="mt-4 flex min-h-0 flex-1 flex-col gap-4">
        <ReplayControls
          exportStatus={exportState?.status}
          onExport={onExport}
          onPause={() => onReplayControl({ action: "pause" })}
          onPlay={() => onReplayControl({ action: "play" })}
          onReset={() => onReplayControl({ action: "reset" })}
          onSpeedChange={(speed) =>
            onReplayControl({ action: "set_speed", speed })
          }
          speed={replayState?.speed}
          status={replayStatus}
        />

        <ReplayTimeline
          cursorMs={replayState?.cursor_ms}
          durationMs={replayState?.duration_ms}
          onSeek={(cursorMs) =>
            onReplayControl({ action: "seek", cursor_ms: cursorMs })
          }
        />
      </div>
    </section>
  );
}