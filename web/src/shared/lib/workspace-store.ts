"use client";

import { useSyncExternalStore } from "react";
import {
  emptyRepositoryGraph,
  type RepositoryGraphSnapshot,
} from "@/features/codebase/graphModel";
import {
  addSpawnedNode,
  clearCanvasSelection,
  createEmptyCanvasDocument,
  patchApprovalRequest,
  patchApprovalStateChanged,
  patchAgentError,
  patchAgentState,
  patchAgentToken,
  patchHandoff,
  patchRunComplete,
  patchRunError,
  patchTaskAssigned,
  patchToolCall,
  patchToolResult,
  resetCanvasDocumentForReplay,
  selectCanvasNode,
  type AgentCanvasDocument,
} from "@/features/canvas/canvasModel";
import type {
  ApprovalStateChangedPayload,
  ApprovalRequestPayload,
  AgentSpawnedPayload,
  AgentRunReplayStatePayload,
  AgentStateChangedPayload,
  AgentRunSummary,
  ConnectedRepositoryView,
  ConnectionMessageType,
  Envelope,
  ErrorPayload,
  FileTouchedPayload,
  HandoffPayload,
  KnownProjectPayload,
  PreferencesView,
  RepositoryBrowseResultPayload,
  RepositoryDirectoryPayload,
  RepositoryGraphStatusPayload,
  RepositoryTreeResultPayload,
  RealtimeRunMessage,
  RunChangeRecordPayload,
  RunEventPayload,
  RunHistoryDetailsResultPayload,
  RunHistoryResultPayload,
  RunHistoryQueryPayload,
  RunHistoryExportResultPayload,
  RunCompletePayload,
  SessionSummary,
  StateChangePayload,
  TaskAssignedPayload,
  TouchedFilePayload,
  ToolCallPayload,
  ToolResultPayload,
  TokenPayload,
  WorkspaceSnapshotPayload,
  WorkspaceStatusPayload,
  WorkspaceUIState,
} from "@/shared/lib/workspace-protocol";

export type ConnectionState = "connecting" | "connected" | "closed";

export interface WorkspaceState {
  connectionState: ConnectionState;
  activeSessionId: string;
  activeProjectRoot: string;
  activeRunId: string;
  selectedRunId: string;
  knownProjects: KnownProjectPayload[];
  sessions: SessionSummary[];
  runSummaries: AgentRunSummary[];
  runEvents: Record<string, StoredRunEvent[]>;
  runTranscripts: Record<string, string>;
  runHistoryQuery: RunHistoryQueryPayload | null;
  runHistoryResults: AgentRunSummary[];
  runHistoryDetails: Record<string, RunHistoryDetailsResultPayload>;
  replayStateByRunId: Record<string, AgentRunReplayStatePayload>;
  exportStateByRunId: Record<string, RunHistoryExportResultPayload>;
  orchestrationDocuments: Record<string, AgentCanvasDocument>;
  pendingApprovals: Record<string, PendingApproval>;
  connectedRepository: ConnectedRepositoryView;
  repositoryGraph: RepositoryGraphSnapshot;
  repositoryBrowser: RepositoryBrowserState;
  repositoryTree: RepositoryTreeState;
  preferences: PreferencesView;
  uiState: WorkspaceUIState;
  status: WorkspaceStatusPayload | null;
  error: ErrorPayload | null;
  warnings: string[];
}

export interface RepositoryBrowserDirectory {
  name: string;
  path: string;
  isGitRepository: boolean;
}

export interface RepositoryBrowserState {
  path: string;
  directories: RepositoryBrowserDirectory[];
  isLoading: boolean;
  showHidden: boolean;
  errorMessage: string;
}

export interface RepositoryTreeState {
  activeTab: "replay" | "repository_tree";
  status: "idle" | "loading" | "ready" | "empty" | "error";
  repositoryRoot: string;
  requestRunId: string;
  paths: string[];
  touchedFiles: TouchedFilePayload[];
  expandedPaths: string[];
  message: string;
  syncErrorMessage: string;
}

export interface PendingApproval {
  sessionId: string;
  runId: string;
  toolCallId: string;
  toolName: string;
  requestKind?: "file_write" | "command";
  status?: "proposed";
  repositoryRoot?: string;
  inputPreview: Record<string, unknown>;
  diffPreview?: {
    targetPath: string;
    originalContent: string;
    proposedContent: string;
    baseContentHash: string;
  };
  commandPreview?: {
    command: string;
    args: string[];
    effectiveDir: string;
  };
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
    | "approval_state_changed"
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
  activeProjectRoot: "",
  activeRunId: "",
  selectedRunId: "",
  knownProjects: [],
  sessions: [],
  runSummaries: [],
  runEvents: {},
  runTranscripts: {},
  runHistoryQuery: null,
  runHistoryResults: [],
  runHistoryDetails: {},
  replayStateByRunId: {},
  exportStateByRunId: {},
  orchestrationDocuments: {},
  pendingApprovals: {},
  connectedRepository: deriveConnectedRepositoryView(defaultPreferences),
  repositoryGraph: { ...emptyRepositoryGraph },
  repositoryBrowser: {
    path: "",
    directories: [],
    isLoading: false,
    showHidden: false,
    errorMessage: "",
  },
  repositoryTree: {
    activeTab: "replay",
    status: "idle",
    repositoryRoot: "",
    requestRunId: "",
    paths: [],
    touchedFiles: [],
    expandedPaths: [],
    message: "",
    syncErrorMessage: "",
  },
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
    const nextActiveProjectRoot =
      payload.active_project_root ?? payload.preferences.project_root ?? "";
    const projectChanged =
      this.state.activeProjectRoot !== "" &&
      nextActiveProjectRoot !== "" &&
      this.state.activeProjectRoot !== nextActiveProjectRoot;
    const nextRunSummaries = dedupeRunSummaries(payload.run_summaries ?? []);
    const nextPendingApprovals = buildPendingApprovalMap(
      payload.pending_approvals ?? [],
    );
    const selectedRunId =
      this.state.selectedRunId &&
      nextRunSummaries.some((run) => run.id === this.state.selectedRunId)
        ? this.state.selectedRunId
        : this.state.selectedRunId &&
            this.state.runEvents[this.state.selectedRunId]
          ? this.state.selectedRunId
          : (payload.active_run_id ?? "");

    const nextConnectedRepository =
      payload.connected_repository ??
      deriveConnectedRepositoryView(payload.preferences);
    const repositoryChanged =
      nextConnectedRepository.path !== this.state.connectedRepository.path ||
      nextConnectedRepository.status !== this.state.connectedRepository.status;

    this.state = {
      ...this.state,
      connectionState: "connected",
      activeSessionId: payload.active_session_id,
      activeProjectRoot: nextActiveProjectRoot,
      activeRunId: payload.active_run_id ?? "",
      selectedRunId,
      knownProjects: payload.known_projects ?? [],
      sessions: payload.sessions,
      runSummaries: nextRunSummaries,
      runEvents: projectChanged ? {} : this.state.runEvents,
      runTranscripts: projectChanged ? {} : this.state.runTranscripts,
      runHistoryResults: projectChanged ? [] : this.state.runHistoryResults,
      runHistoryQuery: this.state.runHistoryQuery,
      runHistoryDetails: projectChanged ? {} : this.state.runHistoryDetails,
      replayStateByRunId: projectChanged ? {} : this.state.replayStateByRunId,
      exportStateByRunId: projectChanged ? {} : this.state.exportStateByRunId,
      orchestrationDocuments: projectChanged
        ? {}
        : this.state.orchestrationDocuments,
      pendingApprovals: nextPendingApprovals,
      connectedRepository: nextConnectedRepository,
      repositoryGraph: repositoryChanged
        ? graphSnapshotFromRepository(nextConnectedRepository)
        : this.state.repositoryGraph,
      repositoryBrowser:
        payload.preferences.project_root &&
        payload.preferences.project_root !== this.state.repositoryBrowser.path
          ? {
              ...this.state.repositoryBrowser,
              errorMessage: "",
            }
          : this.state.repositoryBrowser,
      repositoryTree: repositoryChanged
        ? {
            ...this.state.repositoryTree,
            status: "idle",
            repositoryRoot: nextConnectedRepository.path,
            requestRunId: "",
            paths: [],
            touchedFiles: [],
            message: "",
            syncErrorMessage: "",
          }
        : projectChanged
          ? {
              ...this.state.repositoryTree,
              status: "idle",
              repositoryRoot: nextConnectedRepository.path,
              requestRunId: "",
              paths: [],
              touchedFiles: [],
              expandedPaths: [],
              message: "",
              syncErrorMessage: "",
            }
          : {
              ...this.state.repositoryTree,
              repositoryRoot: nextConnectedRepository.path,
            },
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
      repositoryTree:
        payload.code === "repository_tree_failed"
          ? {
              ...this.state.repositoryTree,
              status: "error",
              message: payload.message,
              requestRunId:
                payload.run_id ?? this.state.repositoryTree.requestRunId,
            }
          : payload.code === "repository_tree_sync_failed"
            ? {
                ...this.state.repositoryTree,
                syncErrorMessage: payload.message,
              }
            : this.state.repositoryTree,
      repositoryBrowser:
        payload.code === "repository_browse_failed"
          ? {
              ...this.state.repositoryBrowser,
              isLoading: false,
              errorMessage: payload.message,
            }
          : this.state.repositoryBrowser,
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
    if (
      message.type === "approval_state_changed" &&
      "tool_call_id" in payload
    ) {
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
      status: nextWorkspaceStatus(this.state.status, message),
    };
    this.emit();
  };

  setRunHistoryResult = (payload: RunHistoryResultPayload) => {
    this.state = {
      ...this.state,
      runHistoryQuery: {
        session_id: payload.session_id,
        all_projects: payload.all_projects,
        query: payload.query,
        file_path: payload.file_path,
        date_from: payload.date_from,
        date_to: payload.date_to,
      },
      runHistoryResults: payload.runs,
    };
    this.emit();
  };

  setRunHistoryDetails = (payload: RunHistoryDetailsResultPayload) => {
    this.state = {
      ...this.state,
      runHistoryDetails: {
        ...this.state.runHistoryDetails,
        [payload.run_id]: payload,
      },
    };
    this.emit();
  };

  setReplayState = (payload: AgentRunReplayStatePayload) => {
    const shouldResetReplayArtifacts =
      payload.status === "preparing" || payload.status === "seeking";

    this.state = {
      ...this.state,
      runEvents: shouldResetReplayArtifacts
        ? { ...this.state.runEvents, [payload.run_id]: [] }
        : this.state.runEvents,
      runTranscripts: shouldResetReplayArtifacts
        ? { ...this.state.runTranscripts, [payload.run_id]: "" }
        : this.state.runTranscripts,
      pendingApprovals: shouldResetReplayArtifacts
        ? Object.fromEntries(
            Object.entries(this.state.pendingApprovals).filter(
              ([, approval]) => approval.runId !== payload.run_id,
            ),
          )
        : this.state.pendingApprovals,
      orchestrationDocuments: shouldResetReplayArtifacts
        ? {
            ...this.state.orchestrationDocuments,
            [payload.run_id]: resetCanvasDocumentForReplay(),
          }
        : this.state.orchestrationDocuments,
      replayStateByRunId: {
        ...this.state.replayStateByRunId,
        [payload.run_id]: payload,
      },
    };
    this.emit();
  };

  setRunHistoryExportState = (payload: RunHistoryExportResultPayload) => {
    this.state = {
      ...this.state,
      exportStateByRunId: {
        ...this.state.exportStateByRunId,
        [payload.run_id]: payload,
      },
    };
    this.emit();
  };

  setRepositoryTreeActiveTab = (
    activeTab: RepositoryTreeState["activeTab"],
  ) => {
    if (this.state.repositoryTree.activeTab === activeTab) {
      return;
    }

    this.state = {
      ...this.state,
      repositoryTree: {
        ...this.state.repositoryTree,
        activeTab,
      },
    };
    this.emit();
  };

  startRepositoryTreeLoad = (runId: string) => {
    if (
      this.state.repositoryTree.requestRunId === runId &&
      this.state.repositoryTree.status === "loading"
    ) {
      return;
    }

    this.state = {
      ...this.state,
      repositoryTree: {
        ...this.state.repositoryTree,
        status: "loading",
        requestRunId: runId,
        message: "Loading the connected repository tree.",
        syncErrorMessage: "",
      },
    };
    this.emit();
  };

  toggleRepositoryTreePath = (path: string) => {
    const normalizedPath = path.trim();
    if (!normalizedPath) {
      return;
    }

    const expandedPaths = new Set(this.state.repositoryTree.expandedPaths);
    if (expandedPaths.has(normalizedPath)) {
      expandedPaths.delete(normalizedPath);
    } else {
      expandedPaths.add(normalizedPath);
    }

    this.state = {
      ...this.state,
      repositoryTree: {
        ...this.state.repositoryTree,
        expandedPaths: Array.from(expandedPaths).sort(),
      },
    };
    this.emit();
  };

  private setRepositoryTreeResult(payload: RepositoryTreeResultPayload) {
    const paths = payload.paths ?? [];
    this.state = {
      ...this.state,
      repositoryTree: {
        ...this.state.repositoryTree,
        status: paths.length > 0 ? "ready" : "empty",
        repositoryRoot:
          payload.repository_root ?? this.state.repositoryTree.repositoryRoot,
        requestRunId: payload.run_id ?? this.state.repositoryTree.requestRunId,
        paths,
        touchedFiles: payload.touched_files ?? [],
        message:
          payload.message ??
          (paths.length > 0
            ? "Repository tree is ready."
            : "This repository does not have any tracked files to display yet."),
        syncErrorMessage: "",
      },
      error:
        this.state.error?.code === "repository_tree_failed"
          ? null
          : this.state.error,
      status: null,
    };
    this.emit();
  }

  private recordTouchedFile(payload: FileTouchedPayload) {
    this.state = {
      ...this.state,
      repositoryTree: {
        ...this.state.repositoryTree,
        requestRunId: payload.run_id,
        touchedFiles: upsertTouchedFile(
          this.state.repositoryTree.touchedFiles,
          payload,
        ),
        syncErrorMessage: "",
      },
    };
    this.emit();
  }

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
      case "approval_state_changed":
        this.appendRunEvent(message as Envelope<RunEventPayload>);
        return;
      case "repository.browse.result":
        this.setRepositoryBrowseResult(
          message.payload as RepositoryBrowseResultPayload,
        );
        return;
      case "repository.tree.result":
        this.setRepositoryTreeResult(
          message.payload as RepositoryTreeResultPayload,
        );
        return;
      case "file_touched":
        this.recordTouchedFile(message.payload as FileTouchedPayload);
        return;
      case "run.history.result":
        this.setRunHistoryResult(message.payload as RunHistoryResultPayload);
        return;
      case "run.history.details.result":
        this.setRunHistoryDetails(
          message.payload as RunHistoryDetailsResultPayload,
        );
        return;
      case "agent.run.replay.state":
        this.setReplayState(message.payload as AgentRunReplayStatePayload);
        return;
      case "run.history.export.result":
        this.setRunHistoryExportState(
          message.payload as RunHistoryExportResultPayload,
        );
        return;
      case "repository_graph_status":
        this.setRepositoryGraphStatus(
          message.payload as RepositoryGraphStatusPayload,
        );
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
        requestKind: payload.request_kind,
        status: payload.status,
        repositoryRoot: payload.repository_root,
        inputPreview: payload.input_preview,
        diffPreview: payload.diff_preview
          ? {
              targetPath: payload.diff_preview.target_path,
              originalContent: payload.diff_preview.original_content,
              proposedContent: payload.diff_preview.proposed_content,
              baseContentHash: payload.diff_preview.base_content_hash,
            }
          : undefined,
        commandPreview: payload.command_preview
          ? {
              command: payload.command_preview.command,
              args: payload.command_preview.args,
              effectiveDir: payload.command_preview.effective_dir,
            }
          : undefined,
        message: payload.message,
        occurredAt: payload.occurred_at,
      },
    };

    this.state = {
      ...this.state,
      activeRunId: payload.run_id,
      selectedRunId: payload.run_id,
      pendingApprovals: nextPendingApprovals,
      orchestrationDocuments: syncOrchestrationDocuments(
        this.state.orchestrationDocuments,
        payload.run_id,
        {
          type: "approval_request",
          payload,
        } as Envelope<ApprovalRequestPayload>,
      ),
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

  startRepositoryBrowse = (path: string, showHidden: boolean) => {
    this.state = {
      ...this.state,
      repositoryBrowser: {
        ...this.state.repositoryBrowser,
        path,
        showHidden,
        isLoading: true,
        errorMessage: "",
      },
      status: {
        phase: "repository-browse",
        message: "Browsing local folders for a repository.",
      },
    };
    this.emit();
  };

  private setRepositoryBrowseResult(payload: RepositoryBrowseResultPayload) {
    this.state = {
      ...this.state,
      repositoryBrowser: {
        path: payload.path,
        directories: payload.directories.map(mapRepositoryDirectory),
        isLoading: false,
        showHidden: this.state.repositoryBrowser.showHidden,
        errorMessage: "",
      },
      status: null,
      error:
        this.state.error?.code === "repository_browse_failed"
          ? null
          : this.state.error,
    };
    this.emit();
  }

  private setRepositoryGraphStatus(payload: RepositoryGraphStatusPayload) {
    this.state = {
      ...this.state,
      repositoryGraph: {
        ...this.state.repositoryGraph,
        status: payload.status,
        errorMessage: payload.status === "error" ? payload.message : undefined,
        nodes:
          payload.status === "ready"
            ? (payload.nodes ?? []).map((node) => ({
                id: node.id,
                label: node.label,
                kind: node.kind,
              }))
            : payload.status === "idle" || payload.status === "loading"
              ? []
              : this.state.repositoryGraph.nodes,
        edges:
          payload.status === "ready"
            ? (payload.edges ?? []).map((edge) => ({
                id: edge.id,
                source: edge.source,
                target: edge.target,
              }))
            : payload.status === "idle" || payload.status === "loading"
              ? []
              : this.state.repositoryGraph.edges,
      },
    };
    this.emit();
  }

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

export function startWorkspaceRepositoryBrowse(
  path: string,
  showHidden: boolean,
) {
  workspaceStore.startRepositoryBrowse(path, showHidden);
}

export function setWorkspaceRepositoryTreeActiveTab(
  activeTab: RepositoryTreeState["activeTab"],
) {
  workspaceStore.setRepositoryTreeActiveTab(activeTab);
}

export function startWorkspaceRepositoryTreeLoad(runId: string) {
  workspaceStore.startRepositoryTreeLoad(runId);
}

export function toggleWorkspaceRepositoryTreePath(path: string) {
  workspaceStore.toggleRepositoryTreePath(path);
}

function syncRunSummaries(
  runSummaries: AgentRunSummary[],
  message: RealtimeRunMessage,
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
  if (message.type === "approval_state_changed") {
    nextSummary.has_tool_activity = true;
    const approvalPayload = payload as ApprovalStateChangedPayload;
    if (approvalPayload.status === "approved") {
      nextSummary.state = "tool_running";
    }
    if (
      approvalPayload.status === "applied" ||
      approvalPayload.status === "rejected" ||
      approvalPayload.status === "blocked" ||
      approvalPayload.status === "expired"
    ) {
      nextSummary.state = "thinking";
    }
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

function upsertTouchedFile(
  touchedFiles: TouchedFilePayload[],
  payload: FileTouchedPayload,
) {
  const nextTouchedFiles = touchedFiles.filter(
    (item) =>
      !(
        item.run_id === payload.run_id &&
        item.agent_id === payload.agent_id &&
        item.file_path === payload.file_path &&
        item.touch_type === payload.touch_type
      ),
  );
  nextTouchedFiles.push({
    run_id: payload.run_id,
    agent_id: payload.agent_id,
    file_path: payload.file_path,
    touch_type: payload.touch_type,
  });

  return nextTouchedFiles.sort((left, right) =>
    `${left.agent_id}:${left.file_path}:${left.touch_type}`.localeCompare(
      `${right.agent_id}:${right.file_path}:${right.touch_type}`,
    ),
  );
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
  message: RealtimeRunMessage,
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
    case "approval_request":
      next = patchApprovalRequest(
        current,
        message.payload as ApprovalRequestPayload,
      );
      break;
    case "approval_state_changed": {
      const payload = message.payload as ApprovalStateChangedPayload;
      next = patchApprovalStateChanged(current, payload);
      break;
    }
    case "tool_call":
      next = patchToolCall(current, message.payload as ToolCallPayload);
      break;
    case "tool_result":
      next = patchToolResult(current, message.payload as ToolResultPayload);
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

function buildPendingApprovalMap(approvals: ApprovalRequestPayload[]) {
  return approvals.reduce<Record<string, PendingApproval>>(
    (pending, approval) => {
      pending[approval.tool_call_id] = {
        sessionId: approval.session_id,
        runId: approval.run_id,
        toolCallId: approval.tool_call_id,
        toolName: approval.tool_name,
        requestKind: approval.request_kind,
        status: approval.status,
        repositoryRoot: approval.repository_root,
        inputPreview: approval.input_preview,
        diffPreview: approval.diff_preview
          ? {
              targetPath: approval.diff_preview.target_path,
              originalContent: approval.diff_preview.original_content,
              proposedContent: approval.diff_preview.proposed_content,
              baseContentHash: approval.diff_preview.base_content_hash,
            }
          : undefined,
        commandPreview: approval.command_preview
          ? {
              command: approval.command_preview.command,
              args: approval.command_preview.args,
              effectiveDir: approval.command_preview.effective_dir,
            }
          : undefined,
        message: approval.message,
        occurredAt: approval.occurred_at,
      };
      return pending;
    },
    {},
  );
}

function mapRepositoryDirectory(
  directory: RepositoryDirectoryPayload,
): RepositoryBrowserDirectory {
  return {
    name: directory.name,
    path: directory.path,
    isGitRepository: directory.is_git_repository,
  };
}

function graphSnapshotFromRepository(
  repository: ConnectedRepositoryView,
): RepositoryGraphSnapshot {
  switch (repository.status) {
    case "connected":
      return { ...emptyRepositoryGraph, status: "loading" };
    case "invalid":
      return {
        ...emptyRepositoryGraph,
        status: "error",
        errorMessage: repository.message,
      };
    default:
      return emptyRepositoryGraph;
  }
}

function deriveConnectedRepositoryView(
  preferences: PreferencesView,
): ConnectedRepositoryView {
  if (preferences.project_root_valid && preferences.project_root) {
    return {
      path: preferences.project_root,
      status: "connected",
      message: "Repository-aware reads stay inside this local Git worktree.",
    };
  }

  if (preferences.project_root_configured) {
    return {
      path: preferences.project_root,
      status: "invalid",
      message:
        preferences.project_root_message ||
        "Relay could not use the saved project root. Choose a valid local Git repository.",
    };
  }

  return {
    path: "",
    status: "not_configured",
    message: "Choose a local Git repository to enable repository-aware tools.",
  };
}

function nextWorkspaceStatus(
  current: WorkspaceStatusPayload | null,
  message: Envelope<RunEventPayload>,
) {
  if (message.type === "token") {
    return { phase: "streaming", message: "Relay is streaming agent output." };
  }
  if (message.type === "approval_state_changed") {
    const payload = message.payload as ApprovalStateChangedPayload;
    return { phase: `approval-${payload.status}`, message: payload.message };
  }
  return current;
}
