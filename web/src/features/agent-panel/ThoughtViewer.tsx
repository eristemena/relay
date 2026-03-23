import type { AgentRunSummary } from "@/shared/lib/workspace-protocol";
import type { PendingApproval, StoredRunEvent } from "@/shared/lib/workspace-store";
import { LiveCursor } from "@/features/agent-panel/LiveCursor";
import {
  describeApprovalState,
  describeRunFailure,
  describeToolRunningState,
  isCancelledRun,
} from "@/features/agent-panel/runStatus";

interface ThoughtViewerProps {
  pendingApproval: PendingApproval | null;
  run: AgentRunSummary | null;
  runEvents: StoredRunEvent[];
  transcript: string;
}

export function ThoughtViewer({ pendingApproval, run, runEvents, transcript }: ThoughtViewerProps) {
  const isStreaming =
    run
      ? run.state === "accepted" ||
        run.state === "thinking" ||
        run.state === "tool_running"
      : false;
  const placeholder = describePlaceholder(
    pendingApproval,
    run,
    transcript,
    runEvents,
  );
  const statusNote = describeStatusNote(pendingApproval, run, runEvents);

  return (
    <section aria-labelledby="thought-viewer-heading" className="stream-card rounded-[1.75rem] p-5">
      <div className="flex items-center justify-between gap-3">
        <h3 id="thought-viewer-heading" className="font-display text-xl text-text">
          Visible output
        </h3>
        {run ? <p className="text-sm text-text-muted">{run.task_text_preview}</p> : null}
      </div>

      {transcript ? (
        <>
          <p className="mt-4 whitespace-pre-wrap text-sm leading-7 text-text">
            {transcript}
            {isStreaming ? <LiveCursor /> : null}
          </p>
          {statusNote ? (
            <p className="mt-4 text-sm leading-6 text-text-muted">{statusNote}</p>
          ) : null}
        </>
      ) : (
        <p className="mt-4 text-sm leading-7 text-text-muted">{placeholder}</p>
      )}
    </section>
  );
}

function describePlaceholder(
  pendingApproval: PendingApproval | null,
  run: AgentRunSummary | null,
  transcript: string,
  runEvents: StoredRunEvent[],
) {
  if (transcript) {
    return transcript;
  }
  if (!run) {
    return "Submit a task to watch one Relay agent stream visible output here. Completed and errored runs stay reviewable from history.";
  }

  const hasTimelineActivity = runEvents.some((event) => event.type !== "token");
  if (run.state === "accepted" || run.state === "thinking") {
    return "Relay accepted this task and is waiting for the first visible provider output.";
  }
  if (run.state === "approval_required") {
    return describeApprovalState(pendingApproval, runEvents);
  }
  if (run.state === "tool_running") {
    return `${describeToolRunningState(runEvents)} The transcript will resume when Relay returns to visible output.`;
  }
  if (run.state === "completed") {
    return hasTimelineActivity
      ? "This run completed without streamed text. Review the timeline for the ordered state changes and tool activity."
      : "This run completed without streamed text.";
  }
  if (run.state === "errored") {
    if (isCancelledRun(runEvents)) {
      return hasTimelineActivity
        ? "This run was cancelled before any visible output arrived. Review the timeline for the cancellation point."
        : "This run was cancelled before any visible output arrived.";
    }
    return hasTimelineActivity
      ? describeRunFailure(runEvents)
      : "This run ended before any visible output arrived.";
  }
  return "Relay is preparing the visible output for this run.";
}

function describeStatusNote(
  pendingApproval: PendingApproval | null,
  run: AgentRunSummary | null,
  runEvents: StoredRunEvent[],
) {
  if (!run) {
    return null;
  }

  if (run.state === "approval_required") {
    return describeApprovalState(pendingApproval, runEvents);
  }
  if (run.state === "tool_running") {
    return describeToolRunningState(runEvents);
  }
  if (run.state === "errored") {
    return describeRunFailure(runEvents);
  }

  return null;
}