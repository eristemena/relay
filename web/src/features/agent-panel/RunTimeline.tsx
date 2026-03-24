import type { StoredRunEvent } from "@/shared/lib/workspace-store";
import { ToolEventRow } from "@/features/agent-panel/ToolEventRow";
import {
  getRunFailureTitle,
  getTerminalRunError,
  isClarificationRequiredCode,
} from "@/features/agent-panel/runStatus";

interface RunTimelineProps {
  events: StoredRunEvent[];
}

export function RunTimeline({ events }: RunTimelineProps) {
  const timelineEvents = events.filter((event) => event.type !== "token");

  return (
    <section aria-labelledby="run-timeline-heading" className="stream-card rounded-[1.75rem] p-5">
      <div className="flex items-center justify-between gap-3">
        <h3 id="run-timeline-heading" className="font-display text-xl text-text">
          Timeline
        </h3>
        <p className="text-sm text-text-muted">Ordered execution events</p>
      </div>

      {timelineEvents.length === 0 ? (
        <p className="mt-4 text-sm leading-6 text-text-muted">State changes, tool calls, and terminal events will appear here as the run progresses or replays.</p>
      ) : (
        <div className="mt-4 space-y-3">
          {timelineEvents.map((event) => {
            if (event.type === "tool_call" || event.type === "tool_result") {
              return <ToolEventRow key={`${event.payload.run_id}-${event.payload.sequence}`} payload={event.payload as never} type={event.type} />;
            }

            return (
              <article className="rounded-3xl border border-border bg-raised/70 p-4" key={`${event.payload.run_id}-${event.payload.sequence}`}>
                <div className="flex items-center justify-between gap-3">
                  <p className="eyebrow">{event.type.replace("_", " ")}</p>
                  <span className="text-xs text-text-muted">#{event.payload.sequence}</span>
                </div>
                <p className="mt-3 text-sm leading-6 text-text">{describeEvent(event)}</p>
              </article>
            );
          })}
        </div>
      )}
    </section>
  );
}

function describeEvent(event: StoredRunEvent) {
	if (event.type === "state_change" && "message" in event.payload) {
		return event.payload.message;
	}
	if (event.type === "complete" && "finish_reason" in event.payload) {
		return `Run finished: ${event.payload.finish_reason}`;
	}
  if (event.type === "run_complete" && "summary" in event.payload) {
    return event.payload.summary;
  }
  if (
    (event.type === "error" ||
      event.type === "agent_error" ||
      event.type === "run_error") &&
    "message" in event.payload
  ) {
    if (event.type !== "agent_error") {
      const terminalError = getTerminalRunError([event]);
      if (terminalError?.code === "run_cancelled") {
        return "Run cancelled: Relay stopped the active run before it produced more output.";
      }
      if (isClarificationRequiredCode(terminalError?.code)) {
        return `${getRunFailureTitle(terminalError?.code)}: ${event.payload.message}`;
      }
    }
    return event.payload.message;
  }
	return "Run event recorded.";
}