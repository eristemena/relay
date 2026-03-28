"use client";

import { useEffect, useEffectEvent, useRef } from "react";
import {
  createAgentRunApprovalRespondRequest,
  createAgentRunCancelRequest,
  createAgentRunOpenRequest,
  createAgentRunReplayControlRequest,
  createAgentRunSubmitRequest,
  createBootstrapRequest,
  createPreferencesSaveRequest,
  createRepositoryBrowseRequest,
  createRunHistoryDetailsRequest,
  createRunHistoryExportRequest,
  createRunHistoryQueryRequest,
  createSessionCreateRequest,
  createSessionOpenRequest,
  type AgentRunReplayControlPayload,
  type Envelope,
  type PreferencesSavePayload,
  type RunHistoryQueryPayload,
  type WorkspaceSnapshotPayload,
} from "@/shared/lib/workspace-protocol";
import {
  startWorkspaceRepositoryBrowse,
  workspaceStore,
} from "@/shared/lib/workspace-store";

function getSocketURL() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws`;
}

export function useWorkspaceSocket() {
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const isMountedRef = useRef(false);
  const hydratedActiveRunRef = useRef<string>("");

  const sendEnvelope = useEffectEvent((message: Envelope<unknown>) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify(message));
    }
  });

  const handleMessage = useEffectEvent((event: MessageEvent<string>) => {
    const parsed = JSON.parse(event.data) as Envelope<unknown>;
    workspaceStore.handleEnvelope(parsed);

    if (
      parsed.type === "workspace.bootstrap" ||
      parsed.type === "session.opened" ||
      parsed.type === "preferences.saved"
    ) {
      const payload = parsed.payload as WorkspaceSnapshotPayload;
      if (!payload.active_run_id) {
        hydratedActiveRunRef.current = "";
        return;
      }

      if (hydratedActiveRunRef.current === payload.active_run_id) {
        return;
      }

      hydratedActiveRunRef.current = payload.active_run_id;
      sendEnvelope(
        createAgentRunOpenRequest(
          payload.active_session_id,
          payload.active_run_id,
        ),
      );
    }
  });

  const connect = useEffectEvent(() => {
    if (!isMountedRef.current) {
      return;
    }

    if (socketRef.current?.readyState === WebSocket.OPEN || socketRef.current?.readyState === WebSocket.CONNECTING) {
      return;
    }

    workspaceStore.setConnectionState("connecting");
    const socket = new WebSocket(getSocketURL());
    socketRef.current = socket;

    socket.addEventListener("open", () => {
      if (!isMountedRef.current || socketRef.current !== socket) {
        return;
      }

      workspaceStore.setConnectionState("connected");
      const lastSessionId = workspaceStore.getSnapshot().activeSessionId || undefined;
      sendEnvelope(createBootstrapRequest(lastSessionId));
    });

    socket.addEventListener("message", handleMessage);

    socket.addEventListener("close", () => {
      if (socketRef.current === socket) {
        socketRef.current = null;
      }

      hydratedActiveRunRef.current = "";

      if (!isMountedRef.current) {
        return;
      }

      workspaceStore.setConnectionState("closed");
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current);
      }
      reconnectTimerRef.current = window.setTimeout(() => connect(), 1200);
    });
  });

  useEffect(() => {
    isMountedRef.current = true;
    connect();

    return () => {
      isMountedRef.current = false;
      if (reconnectTimerRef.current !== null) {
        window.clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
      socketRef.current?.close();
      socketRef.current = null;
    };
  }, []);

  return {
    createSession(displayName?: string) {
      sendEnvelope(createSessionCreateRequest(displayName));
    },
    openSession(sessionId: string) {
      sendEnvelope(createSessionOpenRequest(sessionId));
    },
    openRun(sessionId: string, runId: string) {
      sendEnvelope(createAgentRunOpenRequest(sessionId, runId));
    },
    queryRunHistory(
      sessionId: string,
      payload: Omit<RunHistoryQueryPayload, "session_id"> = {},
    ) {
      sendEnvelope(createRunHistoryQueryRequest(sessionId, payload));
    },
    getRunHistoryDetails(sessionId: string, runId: string) {
      sendEnvelope(createRunHistoryDetailsRequest(sessionId, runId));
    },
    controlReplay(payload: AgentRunReplayControlPayload) {
      sendEnvelope(createAgentRunReplayControlRequest(payload));
    },
    exportRunHistory(sessionId: string, runId: string) {
      sendEnvelope(createRunHistoryExportRequest(sessionId, runId));
    },
    cancelRun(sessionId: string, runId: string) {
      sendEnvelope(createAgentRunCancelRequest(sessionId, runId));
    },
    respondToApproval(
      sessionId: string,
      runId: string,
      toolCallId: string,
      decision: "approved" | "rejected",
    ) {
      sendEnvelope(
        createAgentRunApprovalRespondRequest(
          sessionId,
          runId,
          toolCallId,
          decision,
        ),
      );
    },
    savePreferences(payload: PreferencesSavePayload) {
      sendEnvelope(createPreferencesSaveRequest(payload));
    },
    browseRepository(path?: string, showHidden = false) {
      startWorkspaceRepositoryBrowse(path ?? "", showHidden);
      sendEnvelope(createRepositoryBrowseRequest(path, showHidden));
    },
    submitRun(sessionId: string, task: string) {
      sendEnvelope(createAgentRunSubmitRequest(sessionId, task));
    },
  };
}
