import type {
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
    if (event.type !== "error") {
      continue;
    }

    const payload = event.payload as ErrorPayload;
    if (payload.terminal) {
      return payload;
    }
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
    return "This run ended with an error. Review the timeline for the failure details.";
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

  return terminalError.message || "This run ended with an error. Review the timeline for the failure details.";
}