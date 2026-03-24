import clsx from "clsx";

export type StateBadgeState =
  | "accepted"
  | "active"
  | "thinking"
  | "tool_running"
  | "approval_required"
  | "clarification_required"
  | "completed"
  | "cancelled"
  | "halted"
  | "errored"
  | "queued"
  | "assigned"
  | "streaming"
  | "blocked";

interface StateBadgeProps {
  state: StateBadgeState;
}

const stateLabels: Record<StateBadgeState, string> = {
  accepted: "Accepted",
  active: "Active",
  thinking: "Thinking",
  tool_running: "Tool running",
  approval_required: "Approval required",
  clarification_required: "Clarification required",
  completed: "Completed",
  cancelled: "Cancelled",
  halted: "Halted",
  errored: "Errored",
  queued: "Queued",
  assigned: "Assigned",
  streaming: "Streaming",
  blocked: "Blocked",
};

export function StateBadge({ state }: StateBadgeProps) {
  return (
    <span
      className={clsx(
        "rounded-full px-3 py-1 text-xs font-medium text-text",
        (state === "thinking" ||
          state === "tool_running" ||
          state === "approval_required" ||
          state === "active" ||
          state === "assigned") &&
          "state-glow-thinking",
        state === "streaming" && "state-glow-executing",
        state === "completed" && "state-glow-complete",
        state === "clarification_required" && "state-glow-clarification",
        (state === "errored" || state === "halted" || state === "cancelled") &&
          "state-glow-error",
        (state === "accepted" || state === "queued" || state === "blocked") &&
          "state-glow-idle",
      )}
    >
      {stateLabels[state]}
    </span>
  );
}