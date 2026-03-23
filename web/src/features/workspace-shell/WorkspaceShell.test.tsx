import { act, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WorkspaceShell } from "@/features/workspace-shell/WorkspaceShell";
import {
  buildWorkspaceSnapshot,
  primeWorkspaceStore,
} from "@/shared/lib/test-helpers";
import {
  resetWorkspaceStore,
  workspaceStore,
} from "@/shared/lib/workspace-store";

const socketActions = {
  cancelRun: vi.fn(),
  createSession: vi.fn(),
  openRun: vi.fn(),
  openSession: vi.fn(),
  respondToApproval: vi.fn(),
  savePreferences: vi.fn(),
  submitRun: vi.fn(),
};

vi.mock("@/shared/lib/useWorkspaceSocket", () => ({
  useWorkspaceSocket: () => socketActions,
}));

describe("WorkspaceShell", () => {
  beforeEach(() => {
    resetWorkspaceStore();
    vi.clearAllMocks();
  });

  it("renders the initial loading state", () => {
    render(<WorkspaceShell />);

    expect(
      screen.getByText(/connecting to the relay workspace/i),
    ).toBeInTheDocument();
  });

  it("renders the ready workspace state", () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    expect(
      screen.getByRole("heading", { name: /local ai session control/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: /open session: inspect relay startup/i,
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole("main")).toHaveAttribute("id", "maincontent");
    expect(
      screen.getByRole("navigation", {
        name: /session history and switching/i,
      }),
    ).toBeInTheDocument();
  });

  it("renders a recoverable error state", () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "error",
        payload: {
          code: "session_not_found",
          message:
            "That session is no longer available. Choose another session or start a new one.",
        },
      } as never);
    });

    expect(screen.getByRole("alert")).toHaveTextContent(
      /that session is no longer available/i,
    );
  });

  it("surfaces the saved project root warning in the status banner", () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        preferences: {
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: true,
          project_root: "",
          project_root_configured: false,
          project_root_valid: false,
          project_root_message:
            "Repository-reading tools stay disabled until Relay has a valid project_root in config.toml.",
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-flash-1.5",
          },
          open_browser_on_start: true,
        },
      }),
    );

    render(<WorkspaceShell />);

    const [workspaceStatusBanner] = screen.getAllByRole("status");
    expect(
      within(workspaceStatusBanner).getByText(/project root needs attention/i),
    ).toBeInTheDocument();
    expect(
      within(workspaceStatusBanner).getByText(
        /repository-reading tools stay disabled/i,
      ),
    ).toBeInTheDocument();
  });

  it("forwards approval decisions from the inline approval prompt", () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_1",
        run_summaries: [
          {
            id: "run_1",
            task_text_preview: "Update the README",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "tool_running",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: true,
          },
        ],
        preferences: {
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-flash-1.5",
          },
          open_browser_on_start: true,
        },
      }),
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "approval_request",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          tool_call_id: "call_1",
          tool_name: "write_file",
          input_preview: { path: "README.md" },
          message:
            "Relay needs approval before it can write files inside the configured project root.",
          occurred_at: "2026-03-23T12:00:01Z",
        },
      } as never);
    });

    render(<WorkspaceShell />);

    act(() => {
      screen.getByRole("button", { name: /approve tool/i }).click();
    });

    expect(socketActions.respondToApproval).toHaveBeenCalledWith(
      "session_alpha",
      "run_1",
      "call_1",
      "approved",
    );
  });

  it("does not duplicate saved runs after opening the same run twice", () => {
    const consoleErrorSpy = vi
      .spyOn(console, "error")
      .mockImplementation(() => undefined);

    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        run_summaries: [
          {
            id: "run_saved_1",
            task_text_preview: "Inspect saved startup run",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: true,
          },
        ],
      }),
    );

    render(<WorkspaceShell />);

    const savedRunButton = screen.getByRole("button", {
      name: /inspect saved startup run/i,
    });

    act(() => {
      savedRunButton.click();
      savedRunButton.click();
    });

    expect(socketActions.openRun).toHaveBeenNthCalledWith(
      1,
      "session_alpha",
      "run_saved_1",
    );
    expect(socketActions.openRun).toHaveBeenNthCalledWith(
      2,
      "session_alpha",
      "run_saved_1",
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          run_summaries: [
            {
              id: "run_saved_1",
              task_text_preview: "Inspect saved startup run",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:00:00Z",
              has_tool_activity: true,
            },
            {
              id: "run_saved_1",
              task_text_preview: "Inspect saved startup run",
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              state: "completed",
              started_at: "2026-03-23T12:00:00Z",
              has_tool_activity: true,
            },
          ],
        }),
      } as never);
    });

    expect(
      screen.getAllByRole("button", { name: /inspect saved startup run/i }),
    ).toHaveLength(1);
    expect(
      consoleErrorSpy.mock.calls.some(([message]) =>
        String(message).includes("Encountered two children with the same key"),
      ),
    ).toBe(false);

    consoleErrorSpy.mockRestore();
  });
});
