"use client";

import { useEffect, useEffectEvent, useRef } from "react";
import {
  createBootstrapRequest,
  createPreferencesSaveRequest,
  createSessionCreateRequest,
  createSessionOpenRequest,
  type Envelope,
  type PreferencesSavePayload,
} from "@/shared/lib/workspace-protocol";
import { workspaceStore } from "@/shared/lib/workspace-store";

function getSocketURL() {
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  return `${protocol}//${window.location.host}/ws`;
}

export function useWorkspaceSocket() {
  const socketRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<number | null>(null);
  const isMountedRef = useRef(false);

  const sendEnvelope = useEffectEvent((message: Envelope<unknown>) => {
    if (socketRef.current?.readyState === WebSocket.OPEN) {
      socketRef.current.send(JSON.stringify(message));
    }
  });

  const handleMessage = useEffectEvent((event: MessageEvent<string>) => {
    const parsed = JSON.parse(event.data) as Envelope<unknown>;
    workspaceStore.handleEnvelope(parsed);
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
    savePreferences(payload: PreferencesSavePayload) {
      sendEnvelope(createPreferencesSaveRequest(payload));
    },
  };
}
