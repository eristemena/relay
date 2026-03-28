import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
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

vi.mock("@xyflow/react", async () => {
  const React = await import("react");

  return {
    Background: () => <div data-testid="react-flow-background" />,
    Controls: () => <div aria-label="Canvas controls" />,
    Handle: () => <span />,
    Position: {
      Left: "left",
      Right: "right",
    },
    ReactFlowProvider: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
    ReactFlow: ({ children }: { children: React.ReactNode }) => (
      <div data-testid="react-flow-mock">{children}</div>
    ),
    useReactFlow: () => ({
      fitView: () => Promise.resolve(true),
    }),
  };
});

vi.mock("@/features/approvals/MonacoDiffViewer", () => ({
  MonacoDiffViewer: ({ targetPath }: { targetPath: string }) => (
    <div data-testid="monaco-diff-viewer">Diff viewer for {targetPath}</div>
  ),
}));

const socketActions = {
  browseRepository: vi.fn(),
  cancelRun: vi.fn(),
  controlReplay: vi.fn(),
  createSession: vi.fn(),
  exportRunHistory: vi.fn(),
  getRunHistoryDetails: vi.fn(),
  openRun: vi.fn(),
  openSession: vi.fn(),
  queryRunHistory: vi.fn(),
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
    expect(screen.getByRole("main")).toHaveAttribute("id", "maincontent");
    expect(
      screen.getByRole("heading", {
        name: /inspect relay startup agent graph/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /open sessions/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /open run history/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /open preferences/i }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/agent task/i)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /run task/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("navigation", {
        name: /session history and switching/i,
      }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/saved workspace defaults/i),
    ).not.toBeInTheDocument();
  });

  it("does not show the historical replay rail on first load without an active run", () => {
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

    expect(
      screen.queryByRole("heading", { name: /historical replay/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByText(
        /submit a goal or reopen a saved run to populate the orchestration canvas/i,
      ),
    ).toBeInTheDocument();
  });

  it("keeps session creation reachable on a clean install with no saved sessions", async () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_session_id: "",
        sessions: [],
      }),
    );
    render(<WorkspaceShell />);

    expect(
      screen.getByRole("button", { name: /open sessions/i }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /open sessions/i }));

    expect(
      await screen.findByRole("button", { name: /start new session/i }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /start new session/i }));

    expect(socketActions.createSession).toHaveBeenCalledTimes(1);
  });

  it("switches workspace panels from the graph toolbar", async () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    fireEvent.click(screen.getByRole("button", { name: /open sessions/i }));

    expect(
      await screen.findByRole("navigation", {
        name: /session history and switching/i,
      }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /open run summary/i }));

    expect(
      await screen.findByRole("heading", {
        name: /latest orchestration recap/i,
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /replay or complete a run to capture the orchestration summary here/i,
      ),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /open workspace summary/i }),
    );

    expect(
      await screen.findByText(/saved workspace defaults/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/port 4747/i)).toBeInTheDocument();
    expect(screen.getByText(/theme midnight/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /open preferences/i }));

    expect(
      await screen.findByRole("heading", { name: /local settings/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: /choose a local git repository/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /close workspace panel/i }),
    ).toBeInTheDocument();
  });

  it("queries run history and requests run details when the history panel opens", async () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_history_1",
        run_summaries: [
          {
            id: "run_history_1",
            task_text_preview: "Audit approval review flow",
            role: "reviewer",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-24T12:00:00Z",
            has_tool_activity: true,
          },
        ],
      }),
    );

    render(<WorkspaceShell />);

    fireEvent.click(screen.getByRole("button", { name: /open run history/i }));

    await waitFor(() => {
      expect(socketActions.queryRunHistory).toHaveBeenCalledWith(
        "session_alpha",
        {
          query: undefined,
          file_path: undefined,
          date_from: undefined,
          date_to: undefined,
        },
      );
    });
    await waitFor(() => {
      expect(socketActions.getRunHistoryDetails).toHaveBeenCalledWith(
        "session_alpha",
        "run_history_1",
      );
    });

    const historyDialog = screen.getByRole("dialog", {
      name: /close run history/i,
    });
    expect(
      within(historyDialog).getByRole("heading", {
        name: /audit approval review flow/i,
      }),
    ).toBeInTheDocument();
  });

  it("closes the history dialog after opening a saved run and keeps replay controls visible", async () => {
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

    fireEvent.click(screen.getByRole("button", { name: /open run history/i }));

    const panel = await screen.findByRole("dialog", {
      name: /close run history/i,
    });

    fireEvent.click(
      within(panel).getByRole("button", { name: /inspect saved startup run/i }),
    );

    expect(socketActions.openRun).toHaveBeenCalledWith(
      "session_alpha",
      "run_saved_1",
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "workspace.bootstrap",
        payload: buildWorkspaceSnapshot({
          active_run_id: "run_saved_1",
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
      } as never);
    });

    await waitFor(() => {
      expect(
        screen.queryByRole("dialog", { name: /run history/i }),
      ).not.toBeInTheDocument();
    });

    expect(
      screen.getByRole("heading", { name: /inspect saved startup run/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /browse runs/i }),
    ).toBeInTheDocument();
  });

  it("updates the replay scrubber when backend replay cursor states change", () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_history_1",
        run_summaries: [
          {
            id: "run_history_1",
            task_text_preview: "Replay the long historical run",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-24T12:00:00Z",
            completed_at: "2026-03-24T12:01:23Z",
            has_tool_activity: true,
          },
        ],
      }),
    );

    render(<WorkspaceShell />);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent.run.replay.state",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "completed",
          cursor_ms: 83000,
          duration_ms: 83000,
          speed: 1,
          selected_timestamp: "2026-03-24T12:01:23Z",
        },
      } as never);
    });

    const slider = screen.getByLabelText(/replay position/i);
    expect(slider).toHaveValue("83000");
    expect(screen.getAllByText("83000 ms")).toHaveLength(2);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent.run.replay.state",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "seeking",
          cursor_ms: 0,
          duration_ms: 83000,
          speed: 1,
          selected_timestamp: "2026-03-24T12:00:00Z",
        },
      } as never);
    });

    expect(slider).toHaveValue("0");
    expect(screen.getByText("0 ms")).toBeInTheDocument();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent.run.replay.state",
        payload: {
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "playing",
          cursor_ms: 1200,
          duration_ms: 83000,
          speed: 1,
          selected_timestamp: "2026-03-24T12:00:01Z",
        },
      } as never);
    });

    expect(slider).toHaveValue("1200");
    expect(screen.getByText("1200 ms")).toBeInTheDocument();
  });

  it("forwards repository browse requests from the preferences panel", async () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    fireEvent.click(screen.getByRole("button", { name: /open preferences/i }));

    fireEvent.click(
      await screen.findByRole("button", { name: /browse folders/i }),
    );

    expect(socketActions.browseRepository).toHaveBeenCalledWith(
      undefined,
      false,
    );
  });

  it("submits tasks from the floating bottom composer", () => {
    primeWorkspaceStore(buildWorkspaceSnapshot());
    render(<WorkspaceShell />);

    fireEvent.change(screen.getByLabelText(/agent task/i), {
      target: { value: "Trace the orchestration history flow" },
    });
    fireEvent.click(screen.getByRole("button", { name: /run task/i }));

    expect(socketActions.submitRun).toHaveBeenCalledWith(
      "session_alpha",
      "Trace the orchestration history flow",
    );
  });

  it("cancels the active run with the session and run ids", () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        active_run_id: "run_active",
        run_summaries: [
          {
            id: "run_active",
            task_text_preview: "Inspect relay startup",
            role: "planner",
            model: "anthropic/claude-opus-4",
            state: "active",
            started_at: "2026-03-24T12:00:00Z",
            has_tool_activity: false,
          },
        ],
      }),
    );

    render(<WorkspaceShell />);

    fireEvent.click(screen.getByRole("button", { name: /cancel run/i }));

    expect(socketActions.cancelRun).toHaveBeenCalledWith(
      "session_alpha",
      "run_active",
    );
  });

  it("keeps the floating composer available when project root warnings are shown", () => {
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
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        },
      }),
    );

    render(<WorkspaceShell />);

    expect(screen.getByLabelText(/agent task/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run task/i })).toBeEnabled();
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

    expect(
      screen
        .getAllByRole("alert")
        .some((element) =>
          /that session is no longer available/i.test(
            element.textContent ?? "",
          ),
        ),
    ).toBe(true);
  });

  it("describes halted orchestration runs with the preserved halt reason", () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
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
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        },
        run_summaries: [
          {
            id: "run_halted",
            task_text_preview: "Replay the halted orchestration",
            role: "planner",
            model: "anthropic/claude-opus-4",
            state: "halted",
            started_at: "2026-03-24T12:00:00Z",
            completed_at: "2026-03-24T12:00:03Z",
            has_tool_activity: false,
          },
        ],
      }),
    );

    render(<WorkspaceShell />);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "run_error",
        payload: {
          code: "planner_required",
          message:
            "The run stopped because the planner did not complete and downstream work could not continue.",
          session_id: "session_alpha",
          run_id: "run_halted",
          agent_id: "agent_planner_1",
          sequence: 3,
          replay: true,
          role: "planner",
          model: "anthropic/claude-opus-4",
          terminal: true,
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    expect(
      screen.getAllByText(
        /the run stopped because the planner did not complete and downstream work could not continue/i,
      ).length,
    ).toBeGreaterThan(0);
  });

  it("labels clarification-required halts explicitly", () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
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
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        },
        run_summaries: [
          {
            id: "run_clarification",
            task_text_preview: "Comment the env example",
            role: "coder",
            model: "anthropic/claude-sonnet-4-5",
            state: "halted",
            started_at: "2026-03-24T12:00:00Z",
            completed_at: "2026-03-24T12:00:03Z",
            has_tool_activity: false,
          },
        ],
      }),
    );

    render(<WorkspaceShell />);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "run_error",
        payload: {
          code: "coder_clarification_required",
          message:
            "The run stopped because the coder asked for user clarification instead of producing actionable output.",
          session_id: "session_alpha",
          run_id: "run_clarification",
          agent_id: "agent_run_clarification_coder_2",
          sequence: 3,
          replay: true,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          terminal: true,
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    expect(
      screen.getAllByText(/clarification required/i).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByText(
        /the run stopped because the coder asked for user clarification instead of producing actionable output/i,
      ).length,
    ).toBeGreaterThan(0);
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
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        },
      }),
    );

    render(<WorkspaceShell />);

    const header = screen.getByRole("banner");
    const workspaceStatusBanner = within(header).getByRole("status");
    expect(
      within(workspaceStatusBanner).getByText(/repository not connected/i),
    ).toBeInTheDocument();
    expect(
      within(workspaceStatusBanner).getByText(
        /repository-reading tools stay disabled/i,
      ),
    ).toBeInTheDocument();
  });

  it("shows the repository summary in the workspace summary panel", async () => {
    primeWorkspaceStore(
      buildWorkspaceSnapshot({
        preferences: {
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: true,
          project_root: "/tmp/relay",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        },
      }),
    );

    render(<WorkspaceShell />);

    fireEvent.click(
      screen.getByRole("button", { name: /open workspace summary/i }),
    );

    const panel = await screen.findByRole("dialog", {
      name: /close workspace summary/i,
    });

    expect(
      within(panel).getByText(/repository connected/i),
    ).toBeInTheDocument();
    expect(within(panel).getAllByText("/tmp/relay").length).toBeGreaterThan(0);
  });

  it("opens the approval panel automatically and forwards approval decisions", async () => {
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
            explainer: "google/gemini-2.0-flash-001",
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

    expect(
      await screen.findByRole("button", { name: /close approval review/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /approve request/i }),
    ).toBeInTheDocument();

    act(() => {
      screen.getByRole("button", { name: /approve request/i }).click();
    });

    expect(socketActions.respondToApproval).toHaveBeenCalledWith(
      "session_alpha",
      "run_1",
      "call_1",
      "approved",
    );
  });

  it("does not duplicate saved runs after opening the same run twice", async () => {
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

    fireEvent.click(screen.getByRole("button", { name: /open run history/i }));

    const panel = await screen.findByRole("dialog", {
      name: /close run history/i,
    });
    const savedRunButton = within(panel)
      .getAllByRole("button")
      .find((button) =>
        /inspect saved startup run/i.test(button.textContent ?? ""),
      );

    expect(savedRunButton).not.toBeNull();

    act(() => {
      savedRunButton?.click();
      savedRunButton?.click();
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

    const reopenedPanel = screen.getByRole("dialog", {
      name: /run history/i,
    });

    expect(
      within(reopenedPanel)
        .getAllByRole("button")
        .filter((button) =>
          /inspect saved startup run/i.test(button.textContent ?? ""),
        ),
    ).toHaveLength(1);
    expect(
      consoleErrorSpy.mock.calls.some(([message]) =>
        String(message).includes("Encountered two children with the same key"),
      ),
    ).toBe(false);

    consoleErrorSpy.mockRestore();
  });
});
