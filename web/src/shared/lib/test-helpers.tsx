import { render } from "@testing-library/react";
import type { ReactElement } from "react";
import {
  buildConnectedRepositoryView,
  type WorkspaceSnapshotPayload,
} from "@/shared/lib/workspace-protocol";
import { resetWorkspaceStore, workspaceStore } from "@/shared/lib/workspace-store";

export function buildWorkspaceSnapshot(overrides: Partial<WorkspaceSnapshotPayload> = {}): WorkspaceSnapshotPayload {
  const baseSnapshot: WorkspaceSnapshotPayload = {
    active_session_id: "session_alpha",
    active_project_root: "/tmp/relay",
    known_projects: [
      {
        project_root: "/tmp/relay",
        label: "relay",
        is_active: true,
        is_available: true,
        last_opened_at: "2026-03-23T12:15:00Z",
      },
    ],
    sessions: [
      {
        id: "session_alpha",
        display_name: "Inspect relay startup",
        created_at: "2026-03-23T12:00:00Z",
        last_opened_at: "2026-03-23T12:15:00Z",
        status: "active",
        has_activity: false,
      },
    ],
    preferences: {
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
    },
    ui_state: {
      history_state: "ready",
      canvas_state: "empty",
      save_state: "idle",
    },
    connected_repository: buildConnectedRepositoryView({
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
    }),
    credential_status: {
      configured: false,
    },
    run_summaries: [],
    pending_approvals: [],
    warnings: [],
  };

  const snapshot = {
    ...baseSnapshot,
    ...overrides,
  };

  return {
    ...snapshot,
    connected_repository:
      overrides.connected_repository ??
      buildConnectedRepositoryView(snapshot.preferences),
  };
}

export function primeWorkspaceStore(snapshot: WorkspaceSnapshotPayload) {
  resetWorkspaceStore();
  workspaceStore.applySnapshot(snapshot);
}

export function renderWithWorkspace(ui: ReactElement, snapshot = buildWorkspaceSnapshot()) {
  primeWorkspaceStore(snapshot);
  return render(ui);
}

export function renderIsolatedCanvas(ui: ReactElement) {
  return render(ui);
}