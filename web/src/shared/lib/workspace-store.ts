"use client";

import { useSyncExternalStore } from "react";
import {
  addSpawnedNode,
  clearCanvasSelection,
  createEmptyCanvasDocument,
  patchAgentError,
  patchAgentState,
  patchAgentToken,
  patchHandoff,
  patchRunComplete,
  patchRunError,
  patchTaskAssigned,
  selectCanvasNode,
  type AgentCanvasDocument,
} from "@/features/canvas/canvasModel";
import type {
  ApprovalRequestPayload,
  AgentSpawnedPayload,
  AgentStateChangedPayload,
  AgentRunSummary,
  ConnectionMessageType,
  Envelope,
  ErrorPayload,
  HandoffPayload,
  PreferencesView,
  RunEventPayload,
  RunCompletePayload,
  SessionSummary,
  StateChangePayload,
  TaskAssignedPayload,
  TokenPayload,
  WorkspaceSnapshotPayload,
  WorkspaceStatusPayload,
  WorkspaceUIState,
} from "@/shared/lib/workspace-protocol";

export type ConnectionState = "connecting" | "connected" | "closed";

export interface WorkspaceState {
  connectionState: ConnectionState;
  activeSessionId: string;
  activeRunId: string;
  selectedRunId: string;
  sessions: SessionSummary[];
  runSummaries: AgentRunSummary[];
  runEvents: Record<string, StoredRunEvent[]>;
  runTranscripts: Record<string, string>;
  orchestrationDocuments: Record<string, AgentCanvasDocument>;
  pendingApprovals: Record<string, PendingApproval>;
  preferences: PreferencesView;
  uiState: WorkspaceUIState;
  status: WorkspaceStatusPayload | null;
  error: ErrorPayload | null;
  warnings: string[];
}

export interface PendingApproval {
  sessionId: string;
  runId: string;
  toolCallId: string;
  toolName: string;
  inputPreview: Record<string, unknown>;
  message: string;
  occurredAt: string;
}

export interface StoredRunEvent {
  type: Extract<
    ConnectionMessageType,
    | "state_change"
    | "token"
    | "tool_call"
    | "tool_result"
    | "complete"
    | "agent_spawned"
    | "agent_state_changed"
    | "task_assigned"
    | "handoff_start"
    | "handoff_complete"
    | "agent_error"
    | "run_complete"
    | "run_error"
    | "error"
  >;
  payload: RunEventPayload;
}

function isTerminalRunState(
  state: AgentRunSummary["state"],
): state is "completed" | "errored" {
  return state === "completed" || state === "errored";
}

const defaultPreferences: PreferencesView = {
  preferred_port: 4747,
  appearance_variant: "midnight",
  has_credentials: false,
  openrouter_configured: false,
  project_root: "",
  project_root_configured: false,
  project_root_valid: false,
  agent_models: {
    planner: "anthropic/claude-opus-4",
    coder: "anthropic/claude-sonnet-4-5",
    reviewer: "anthropic/claude-sonnet-4-5",
    tester: "deepseek/deepseek-chat",
    explainer: "google/gemini-2.0-flash-001",
  },
  open_browser_on_start: true,
};

const defaultUIState: WorkspaceUIState = {
  history_state: "loading",
  canvas_state: "empty",
  save_state: "idle",
};

const defaultState: WorkspaceState = {
  connectionState: "connecting",
  activeSessionId: "",
  activeRunId: "",
  selectedRunId: "",
  sessions: [],
  runSummaries: [],
  runEvents: {},
  runTranscripts: {},
  orchestrationDocuments: {},
  pendingApprovals: {},
  preferences: defaultPreferences,
  uiState: defaultUIState,
  status: { phase: "startup", message: "Connecting to the Relay workspace." },
  error: null,
  warnings: [],
};

type Listener = () => void;

class WorkspaceStore {
  private state: WorkspaceState = defaultState;

  private listeners = new Set<Listener>();

  getSnapshot = () => this.state;

  subscribe = (listener: Listener) => {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  };

  reset = () => {
    if (this.state === defaultState) {
      return;
    }

    this.state = defaultState;
    this.emit();
  };

  setConnectionState = (connectionState: ConnectionState) => {
    const nextStatus =
      connectionState === "connecting"
        ? { phase: "reconnecting", message: "Reconnecting to Relay." }
        : this.state.status;

    if (
      this.state.connectionState === connectionState &&
      this.state.status?.phase === nextStatus?.phase &&
      this.state.status?.message === nextStatus?.message
    ) {
      return;
    }

    this.state = {
      ...this.state,
      connectionState,
      status: nextStatus,
    };
    this.emit();
  };

  setStatus = (status: WorkspaceStatusPayload | null) => {
    this.state = {
      ...this.state,
      status,
      uiState: {
        ...this.state.uiState,
        save_state:
          status?.phase === "preferences-saving"
            ? "saving"
            : this.state.uiState.save_state,
      },
    };
    this.emit();
  };

  applySnapshot = (payload: WorkspaceSnapshotPayload) => {
    const nextRunSummaries = dedupeRunSummaries(payload.run_summaries ?? []);
    const selectedRunId =
      this.state.selectedRunId &&
      nextRunSummaries.some((run) => run.id === this.state.selectedRunId)
        ? this.state.selectedRunId
        : this.state.selectedRunId &&
            this.state.runEvents[this.state.selectedRunId]
          ? this.state.selectedRunId
          : (payload.active_run_id ?? nextRunSummaries[0]?.id ?? "");

    this.state = {
      ...this.state,
      connectionState: "connected",
      activeSessionId: payload.active_session_id,
      activeRunId: payload.active_run_id ?? "",
      selectedRunId,
      sessions: payload.sessions,
      runSummaries: nextRunSummaries,
      runTranscripts: this.state.runTranscripts,
      orchestrationDocuments: this.state.orchestrationDocuments,
      pendingApprovals: {},
      preferences: payload.preferences,
      uiState: payload.ui_state,
      status: null,
      warnings: payload.warnings ?? [],
      error: null,
    };
    this.emit();
  };

  setError = (payload: ErrorPayload) => {
    const nextRunEvents = { ...this.state.runEvents };
    const nextPendingApprovals = { ...this.state.pendingApprovals };
    if (payload.run_id) {
      const runId = payload.run_id;
      const existing = nextRunEvents[runId] ?? [];
      nextRunEvents[runId] = [...existing, { type: "error", payload }];
      for (const [toolCallId, approval] of Object.entries(
        nextPendingApprovals,
      )) {
        if (approval.runId === runId) {
          delete nextPendingApprovals[toolCallId];
        }
      }
    }

    this.state = {
      ...this.state,
      error: payload,
      runEvents: nextRunEvents,
      pendingApprovals: nextPendingApprovals,
      selectedRunId: payload.run_id ?? this.state.selectedRunId,
      status: null,
      uiState: {
        ...this.state.uiState,
        save_state: payload.code.includes("preferences")
          ? "error"
          : this.state.uiState.save_state,
      },
    };
    this.emit();
  };

  appendRunEvent = (message: Envelope<RunEventPayload>) => {
    const payload = message.payload;
    if (!("run_id" in payload) || typeof payload.run_id !== "string") {
      return;
    }

    const runId = payload.run_id;
    const shouldReset =
      "sequence" in payload &&
      payload.sequence === 1 &&
      (message.type === "state_change" || message.type === "error");

    const existing = shouldReset ? [] : (this.state.runEvents[runId] ?? []);
    const nextEventsForRun = syncRunEvents(existing, {
      type: message.type as StoredRunEvent["type"],
      payload,
    });
    const nextRunEvents = {
      ...this.state.runEvents,
      [runId]: nextEventsForRun,
    };
    const nextRunTranscripts = syncRunTranscripts(
      this.state.runTranscripts,
      runId,
      existing,
      nextEventsForRun,
      { type: message.type as StoredRunEvent["type"], payload },
    );
    const nextPendingApprovals = { ...this.state.pendingApprovals };
    if (message.type === "tool_result" && "tool_call_id" in payload) {
      delete nextPendingApprovals[payload.tool_call_id as string];
    }
    if (message.type === "complete" || message.type === "error") {
      for (const [toolCallId, approval] of Object.entries(
        nextPendingApprovals,
      )) {
        if (approval.runId === runId) {
          delete nextPendingApprovals[toolCallId];
        }
      }
    }

    const nextRunSummaries = syncRunSummaries(this.state.runSummaries, message);
    const nextOrchestrationDocuments = syncOrchestrationDocuments(
      this.state.orchestrationDocuments,
      runId,
      message,
    );
    const clearsActiveRun =
      !payload.replay &&
      (message.type === "complete" ||
        message.type === "run_complete" ||
        message.type === "run_error" ||
        message.type === "error");
    this.state = {
      ...this.state,
      activeRunId: clearsActiveRun
        ? ""
        : payload.replay
          ? this.state.activeRunId
          : runId,
      selectedRunId: runId,
      runEvents: nextRunEvents,
      runTranscripts: nextRunTranscripts,
      orchestrationDocuments: nextOrchestrationDocuments,
      pendingApprovals: nextPendingApprovals,
      runSummaries: nextRunSummaries,
      error:
        message.type === "error" || message.type === "run_error"
          ? (payload as ErrorPayload)
          : this.state.error,
      status:
        message.type === "token"
          ? { phase: "streaming", message: "Relay is streaming agent output." }
          : this.state.status,
    };
    this.emit();
  };

  handleEnvelope = (message: Envelope<unknown>) => {
    switch (message.type) {
      case "workspace.bootstrap":
      case "session.created":
      case "session.opened":
      case "preferences.saved":
        this.applySnapshot(message.payload as WorkspaceSnapshotPayload);
        if (message.type === "preferences.saved") {
          this.setStatus({
            phase: "preferences-saved",
            message: "Preferences saved locally.",
          });
        }
        return;
      case "workspace.status":
        this.setStatus(message.payload as WorkspaceStatusPayload);
        return;
      case "approval_request":
        this.setPendingApproval(message.payload as ApprovalRequestPayload);
        return;
      case "state_change":
      case "token":
      case "tool_call":
      case "tool_result":
      case "complete":
      case "agent_spawned":
      case "agent_state_changed":
      case "task_assigned":
      case "handoff_start":
      case "handoff_complete":
      case "agent_error":
      case "run_complete":
      case "run_error":
        this.appendRunEvent(message as Envelope<RunEventPayload>);
        return;
      case "error":
        if ((message.payload as ErrorPayload).run_id) {
          this.appendRunEvent(message as Envelope<RunEventPayload>);
          return;
        }
        this.setError(message.payload as ErrorPayload);
        return;
      default:
        return;
    }
  };

  setPendingApproval = (payload: ApprovalRequestPayload) => {
    const nextPendingApprovals = {
      ...this.state.pendingApprovals,
      [payload.tool_call_id]: {
        sessionId: payload.session_id,
        runId: payload.run_id,
        toolCallId: payload.tool_call_id,
        toolName: payload.tool_name,
        inputPreview: payload.input_preview,
        message: payload.message,
        occurredAt: payload.occurred_at,
      },
    };

    this.state = {
      ...this.state,
      activeRunId: payload.run_id,
      selectedRunId: payload.run_id,
      pendingApprovals: nextPendingApprovals,
      runSummaries: syncRunSummaries(this.state.runSummaries, {
        type: "approval_request",
        payload,
      } as Envelope<ApprovalRequestPayload>),
      status: {
        phase: "approval-required",
        message: payload.message,
      },
    };
    this.emit();
  };

  private emit() {
    for (const listener of this.listeners) {
      listener();
    }
  }

  selectCanvasNode = (runId: string, agentId: string) => {
    const document = this.state.orchestrationDocuments[runId];
    if (!document) {
      return;
    }

    this.state = {
      ...this.state,
      selectedRunId: runId,
      orchestrationDocuments: {
        ...this.state.orchestrationDocuments,
        [runId]: selectCanvasNode(document, agentId),
      },
    };
    this.emit();
  };

  clearCanvasSelection = (runId: string) => {
    const document = this.state.orchestrationDocuments[runId];
    if (!document) {
      return;
    }

    this.state = {
      ...this.state,
      orchestrationDocuments: {
        ...this.state.orchestrationDocuments,
        [runId]: clearCanvasSelection(document),
      },
    };
    this.emit();
  };
}

function syncRunEvents(runEvents: StoredRunEvent[], nextEvent: StoredRunEvent) {
  const sequence = nextEvent.payload.sequence;
  if (typeof sequence !== "number") {
    return [...runEvents, nextEvent];
  }

  const nextRunEvents = runEvents.filter(
    (event) => event.payload.sequence !== sequence,
  );
  return [...nextRunEvents, nextEvent].sort(
    (left, right) => getRunEventSequence(left) - getRunEventSequence(right),
  );
}

function getRunEventSequence(event: StoredRunEvent) {
  return typeof event.payload.sequence === "number"
    ? event.payload.sequence
    : 0;
}

function syncRunTranscripts(
  runTranscripts: Record<string, string>,
  runID: string,
  previousRunEvents: StoredRunEvent[],
  nextRunEvents: StoredRunEvent[],
  nextEvent: StoredRunEvent,
) {
  if (nextEvent.type !== "token") {
    return runTranscripts;
  }

  const payload = nextEvent.payload as TokenPayload;
  const nextTranscript = shouldRebuildTranscript(
    previousRunEvents,
    payload.sequence,
  )
    ? buildRunTranscript(nextRunEvents)
    : (runTranscripts[runID] ?? "") + payload.text;

  return {
    ...runTranscripts,
    [runID]: nextTranscript,
  };
}

function shouldRebuildTranscript(
  previousRunEvents: StoredRunEvent[],
  sequence: number | undefined,
) {
  if (typeof sequence !== "number") {
    return false;
  }
  return previousRunEvents.some((event) => event.payload.sequence === sequence);
}

function buildRunTranscript(runEvents: StoredRunEvent[]) {
  return runEvents
    .filter((event) => event.type === "token")
    .map((event) => ("text" in event.payload ? event.payload.text : ""))
    .join("");
}

export const workspaceStore = new WorkspaceStore();

export function useWorkspaceStore<TSelected>(selector: (state: WorkspaceState) => TSelected): TSelected {
  return useSyncExternalStore(workspaceStore.subscribe, () => selector(workspaceStore.getSnapshot()), () => selector(defaultState));
}

export function resetWorkspaceStore() {
  workspaceStore.reset();
}

export function selectWorkspaceCanvasNode(runId: string, agentId: string) {
  workspaceStore.selectCanvasNode(runId, agentId);
}

export function clearWorkspaceCanvasSelection(runId: string) {
  workspaceStore.clearCanvasSelection(runId);
}

function syncRunSummaries(
  runSummaries: AgentRunSummary[],
  message: Envelope<RunEventPayload> | Envelope<ApprovalRequestPayload>,
) {
  const payload = message.payload;
  if (!("run_id" in payload) || typeof payload.run_id !== "string") {
    return runSummaries;
  }

  const existing = runSummaries.find((run) => run.id === payload.run_id);
  const nextSummary: AgentRunSummary = existing
    ? { ...existing }
    : {
        id: payload.run_id,
        task_text_preview: "Active task",
        role: payload.role as AgentRunSummary["role"],
        model: payload.model ?? "",
        state: "accepted",
        started_at: payload.occurred_at ?? new Date().toISOString(),
        has_tool_activity: false,
      };

  if (message.type === "state_change") {
    nextSummary.state = (payload as StateChangePayload).state;
  }
  if (message.type === "tool_call" || message.type === "tool_result") {
    nextSummary.has_tool_activity = true;
    if (message.type === "tool_call") {
      nextSummary.state = "tool_running";
    }
    if (
      message.type === "tool_result" &&
      !isTerminalRunState(nextSummary.state)
    ) {
      nextSummary.state = "thinking";
    }
  }
  if (message.type === "approval_request") {
    nextSummary.has_tool_activity = true;
    nextSummary.state = "approval_required";
  }
  if (message.type === "complete") {
    nextSummary.state = "completed";
    nextSummary.completed_at = payload.occurred_at;
  }
  if (message.type === "run_complete") {
    nextSummary.state = "completed";
    nextSummary.completed_at = payload.occurred_at;
  }
  if (message.type === "run_error") {
    nextSummary.state = "halted";
    nextSummary.error_code = (payload as ErrorPayload).code;
    nextSummary.completed_at = (payload as ErrorPayload).occurred_at;
  }
  if (message.type === "error") {
    nextSummary.state = "errored";
    nextSummary.error_code = (payload as ErrorPayload).code;
    nextSummary.completed_at = (payload as ErrorPayload).occurred_at;
  }
  if (
    message.type === "agent_spawned" ||
    message.type === "agent_state_changed"
  ) {
    nextSummary.state = "active";
  }
  if (message.type === "token" && nextSummary.state === "accepted") {
    nextSummary.state = "thinking";
  }

  const nextRunSummaries = runSummaries.filter(
    (run) => run.id !== nextSummary.id,
  );
  return dedupeRunSummaries([nextSummary, ...nextRunSummaries]);
}

function dedupeRunSummaries(runSummaries: AgentRunSummary[]) {
  const seen = new Set<string>();
  const unique: AgentRunSummary[] = [];
  for (const runSummary of runSummaries) {
    if (!runSummary.id || seen.has(runSummary.id)) {
      continue;
    }
    seen.add(runSummary.id);
    unique.push(runSummary);
  }
  return unique;
}

function syncOrchestrationDocuments(
  documents: Record<string, AgentCanvasDocument>,
  runId: string,
  message: Envelope<RunEventPayload>,
) {
  const current = documents[runId] ?? createEmptyCanvasDocument();
  let next = current;

  switch (message.type) {
    case "agent_spawned":
      next = addSpawnedNode(current, message.payload as AgentSpawnedPayload);
      break;
    case "agent_state_changed":
      next = patchAgentState(
        current,
        message.payload as AgentStateChangedPayload,
      );
      break;
    case "task_assigned":
      next = patchTaskAssigned(current, message.payload as TaskAssignedPayload);
      break;
    case "token":
      next = patchAgentToken(current, message.payload as TokenPayload);
      break;
    case "handoff_start":
      next = patchHandoff(
        current,
        message.payload as HandoffPayload,
        "handoff_start",
      );
      break;
    case "handoff_complete":
      next = patchHandoff(
        current,
        message.payload as HandoffPayload,
        "handoff_complete",
      );
      break;
    case "agent_error":
      next = patchAgentError(current, message.payload as ErrorPayload);
      break;
    case "run_complete":
      next = patchRunComplete(current, message.payload as RunCompletePayload);
      break;
    case "run_error":
      next = patchRunError(current, message.payload as ErrorPayload);
      break;
    default:
      return documents;
  }

  return {
    ...documents,
    [runId]: next,
  };
}
