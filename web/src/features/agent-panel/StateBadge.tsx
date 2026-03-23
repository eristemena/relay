import clsx from "clsx";

export type StateBadgeState =
  | "accepted"
  | "thinking"
  | "tool_running"
  | "approval_required"
  | "completed"
  | "errored"
  | "idle"
  | "executing"
  | "complete"
  | "error";

interface StateBadgeProps {
  state: StateBadgeState;
}

const stateLabels: Record<StateBadgeState, string> = {
  accepted: "Accepted",
  thinking: "Thinking",
  tool_running: "Tool running",
  approval_required: "Approval required",
  completed: "Completed",
  errored: "Errored",
  idle: "Idle",
  executing: "Executing",
  complete: "Complete",
  error: "Error",
};

export function StateBadge({ state }: StateBadgeProps) {
  return (
    <span
      className={clsx(
        "rounded-full px-3 py-1 text-xs font-medium text-text",
        (state === "thinking" ||
          state === "tool_running" ||
          state === "approval_required") &&
          "state-glow-thinking",
        state === "executing" && "state-glow-executing",
        (state === "completed" || state === "complete") &&
          "state-glow-complete",
        (state === "errored" || state === "error") && "state-glow-error",
        (state === "accepted" || state === "idle") && "state-glow-idle",
      )}
    >
      {stateLabels[state]}
    </span>
  );
}