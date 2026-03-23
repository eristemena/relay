import { render } from "@testing-library/react";
import type { ReactElement } from "react";
import type { WorkspaceSnapshotPayload } from "@/shared/lib/workspace-protocol";
import { resetWorkspaceStore, workspaceStore } from "@/shared/lib/workspace-store";

export function buildWorkspaceSnapshot(overrides: Partial<WorkspaceSnapshotPayload> = {}): WorkspaceSnapshotPayload {
  return {
    active_session_id: "session_alpha",
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
        explainer: "google/gemini-flash-1.5",
      },
      open_browser_on_start: true,
    },
    ui_state: {
      history_state: "ready",
      canvas_state: "empty",
      save_state: "idle",
    },
    credential_status: {
      configured: false,
    },
    run_summaries: [],
    warnings: [],
    ...overrides,
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
