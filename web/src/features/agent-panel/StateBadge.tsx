import clsx from "clsx";

interface StateBadgeProps {
  state:
    | "accepted"
    | "thinking"
    | "tool_running"
    | "approval_required"
    | "completed"
    | "errored";
}

const stateLabels: Record<StateBadgeProps["state"], string> = {
  accepted: "Accepted",
  thinking: "Thinking",
  tool_running: "Tool running",
  approval_required: "Approval required",
  completed: "Completed",
  errored: "Errored",
};

export function StateBadge({ state }: StateBadgeProps) {
  return (
    <span
      className={clsx(
        "rounded-full px-3 py-1 text-xs font-medium text-text",
        state === "thinking" && "state-glow-thinking",
        state === "tool_running" && "state-glow-thinking",
        state === "approval_required" && "state-glow-thinking",
        state === "completed" && "state-glow-complete",
        state === "errored" && "state-glow-error",
        state === "accepted" && "state-glow-idle",
      )}
    >
      {stateLabels[state]}
    </span>
  );
}