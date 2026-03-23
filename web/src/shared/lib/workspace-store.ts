"use client";

import { useSyncExternalStore } from "react";
import type {
  Envelope,
  ErrorPayload,
  PreferencesView,
  SessionSummary,
  WorkspaceSnapshotPayload,
  WorkspaceStatusPayload,
  WorkspaceUIState,
} from "@/shared/lib/workspace-protocol";

export type ConnectionState = "connecting" | "connected" | "closed";

export interface WorkspaceState {
  connectionState: ConnectionState;
  activeSessionId: string;
  sessions: SessionSummary[];
  preferences: PreferencesView;
  uiState: WorkspaceUIState;
  status: WorkspaceStatusPayload | null;
  error: ErrorPayload | null;
  warnings: string[];
}

const defaultPreferences: PreferencesView = {
  preferred_port: 4747,
  appearance_variant: "midnight",
  has_credentials: false,
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
  sessions: [],
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
        save_state: status?.phase === "preferences-saving" ? "saving" : this.state.uiState.save_state,
      },
    };
    this.emit();
  };

  applySnapshot = (payload: WorkspaceSnapshotPayload) => {
    this.state = {
      ...this.state,
      connectionState: "connected",
      activeSessionId: payload.active_session_id,
      sessions: payload.sessions,
      preferences: payload.preferences,
      uiState: payload.ui_state,
      warnings: payload.warnings ?? [],
      error: null,
    };
    this.emit();
  };

  setError = (payload: ErrorPayload) => {
    this.state = {
      ...this.state,
      error: payload,
      status: null,
      uiState: {
        ...this.state.uiState,
        save_state: payload.code.includes("preferences") ? "error" : this.state.uiState.save_state,
      },
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
          this.setStatus({ phase: "preferences-saved", message: "Preferences saved locally." });
        }
        return;
      case "workspace.status":
        this.setStatus(message.payload as WorkspaceStatusPayload);
        return;
      case "error":
        this.setError(message.payload as ErrorPayload);
        return;
      default:
        return;
    }
  };

  private emit() {
    for (const listener of this.listeners) {
      listener();
    }
  }
}

export const workspaceStore = new WorkspaceStore();

export function useWorkspaceStore<TSelected>(selector: (state: WorkspaceState) => TSelected): TSelected {
  return useSyncExternalStore(workspaceStore.subscribe, () => selector(workspaceStore.getSnapshot()), () => selector(defaultState));
}

export function resetWorkspaceStore() {
  workspaceStore.reset();
}
