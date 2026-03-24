import type {
  AgentRunSummary,
  ErrorPayload,
  ToolCallPayload,
  ToolResultPayload,
} from "@/shared/lib/workspace-protocol";
import type {
  PendingApproval,
  StoredRunEvent,
} from "@/shared/lib/workspace-store";

export function getTerminalRunError(events: StoredRunEvent[]): ErrorPayload | null {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.type !== "error" && event.type !== "run_error") {
      continue;
    }

    const payload = event.payload as ErrorPayload;
    if (payload.terminal) {
      return payload;
    }
  }

  return null;
}

export function isClarificationRequiredCode(code?: string | null) {
  return Boolean(code && code.endsWith("_clarification_required"));
}

export function getRunFailureTitle(code?: string | null) {
  if (isClarificationRequiredCode(code)) {
    return "Clarification required";
  }
  if (code === "run_cancelled") {
    return "Run cancelled";
  }
  return "Run halted";
}

export function getRunSummaryBadgeState(
  run: AgentRunSummary | null,
): import("@/features/agent-panel/StateBadge").StateBadgeState {
  if (!run) {
    return "accepted";
  }
  if (isClarificationRequiredCode(run.error_code)) {
    return "clarification_required";
  }
  return run.state;
}

export function getRunSummaryStateLabel(run: AgentRunSummary) {
  return isClarificationRequiredCode(run.error_code)
    ? "Clarification required"
    : run.state;
}

export function describeRunSummaryReplayBanner(run: AgentRunSummary) {
  if (isClarificationRequiredCode(run.error_code)) {
    return `Reviewing saved run ${run.id} in read-only mode. Clarification was required before Relay could continue this run.`;
  }

  return `Reviewing saved run ${run.id} in read-only mode.`;
}

export function describeRunSummaryPlaceholder(run: AgentRunSummary) {
  if (isClarificationRequiredCode(run.error_code)) {
    return "Clarification required before Relay could continue this run. Update the task or missing context, then rerun when ready.";
  }
  if (run.state === "halted") {
    return "This run halted before any visible output arrived.";
  }
  if (run.state === "errored") {
    return "This run ended before any visible output arrived.";
  }

  return null;
}

export function isCancelledRun(events: StoredRunEvent[]) {
  return getTerminalRunError(events)?.code === "run_cancelled";
}

export function getLatestToolCall(
  events: StoredRunEvent[],
): ToolCallPayload | null {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.type === "tool_call") {
      return event.payload as ToolCallPayload;
    }
  }

  return null;
}

export function getLatestToolResult(
  events: StoredRunEvent[],
): ToolResultPayload | null {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const event = events[index];
    if (event.type === "tool_result") {
      return event.payload as ToolResultPayload;
    }
  }

  return null;
}

export function formatToolName(toolName: string) {
  return toolName.replaceAll("_", " ");
}

export function describeToolRunningState(events: StoredRunEvent[]) {
  const latestToolCall = getLatestToolCall(events);
  if (!latestToolCall) {
    return "Relay is inside a tool step now. Review the timeline for the current action.";
  }

  return `Relay is running ${formatToolName(latestToolCall.tool_name)} now. Review the timeline for the active input and result.`;
}

export function describeApprovalState(
  pendingApproval: PendingApproval | null,
  events: StoredRunEvent[],
) {
  if (pendingApproval) {
    return `${pendingApproval.message} Approve or reject ${formatToolName(pendingApproval.toolName)} to continue the run.`;
  }

  const latestToolCall = getLatestToolCall(events);
  if (!latestToolCall) {
    return "Relay is waiting for approval before it can continue the current tool step.";
  }

  return `Relay is waiting for approval before it can continue ${formatToolName(latestToolCall.tool_name)}.`;
}

export function describeRunFailure(events: StoredRunEvent[]) {
  const terminalError = getTerminalRunError(events);
  if (!terminalError) {
    return "This run halted before Relay could finish the orchestration. Review the timeline for the failure details.";
  }

  if (isClarificationRequiredCode(terminalError.code)) {
    return `Clarification required. ${terminalError.message} Update the task or supply the missing context, then rerun.`;
  }

  if (terminalError.code === "run_cancelled") {
    return "This run was cancelled. Review the timeline for the cancellation point, then submit another task when ready.";
  }

  const latestToolResult = getLatestToolResult(events);
  if (latestToolResult?.status === "rejected") {
    return `This run hit a blocked ${formatToolName(latestToolResult.tool_name)} step after approval was rejected. Review the timeline for the protected action and decide whether to retry with a different task.`;
  }
  if (latestToolResult?.status === "error") {
    return `This run ended after ${formatToolName(latestToolResult.tool_name)} failed. Review the timeline for the tool failure details.`;
  }

  return (
    terminalError.message ||
    "This run halted before Relay could finish the orchestration. Review the timeline for the failure details."
  );
}