import { act, fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { WorkspaceCanvas } from "@/features/canvas/WorkspaceCanvas";
import {
  buildWorkspaceSnapshot,
  renderWithWorkspace,
} from "@/shared/lib/test-helpers";
import { workspaceStore } from "@/shared/lib/workspace-store";

vi.mock("@xyflow/react", async () => {
  const React = await import("react");

  return {
    Background: () => <div data-testid="react-flow-background" />,
    Controls: () => (
      <div aria-label="Canvas controls">
        <button aria-label="Zoom in" type="button" />
        <button aria-label="Zoom out" type="button" />
        <button aria-label="Fit view" type="button" />
      </div>
    ),
    Handle: ({ position, type }: { position: string; type: string }) => (
      <span data-testid={`${type}-${position}-handle`} />
    ),
    Position: {
      Left: "left",
      Right: "right",
    },
    ReactFlowProvider: ({ children }: { children: React.ReactNode }) => (
      <>{children}</>
    ),
    ReactFlow: ({
      children,
      edges,
      edgeTypes,
      nodeTypes,
      nodes,
      onNodeClick,
      onPaneClick,
    }: {
      children: React.ReactNode;
      edges: Array<Record<string, unknown>>;
      edgeTypes?: Record<
        string,
        (props: Record<string, unknown>) => React.ReactNode
      >;
      nodeTypes: Record<
        string,
        (props: Record<string, unknown>) => React.ReactNode
      >;
      nodes: Array<Record<string, unknown>>;
      onNodeClick?: (event: unknown, node: { id: string }) => void;
      onPaneClick?: () => void;
    }) => (
      <div data-testid="react-flow-mock">
        <button
          aria-label="Canvas background"
          onClick={onPaneClick}
          type="button"
        />
        <div data-testid="react-flow-edge-count">{edges.length}</div>
        {edges.map((edge) => {
          const EdgeComponent = edgeTypes?.[String(edge.type)];

          if (!EdgeComponent) {
            return null;
          }

          return (
            <svg data-testid={`edge-${String(edge.id)}`} key={String(edge.id)}>
              <EdgeComponent
                data={edge.data}
                id={edge.id}
                sourcePosition="right"
                sourceX={0}
                sourceY={0}
                targetPosition="left"
                targetX={100}
                targetY={20}
              />
            </svg>
          );
        })}
        {nodes.map((node) => {
          const NodeComponent = nodeTypes[String(node.type)];

          return (
            <div
              data-testid={`node-position-${String(node.id)}`}
              key={String(node.id)}
              onClick={() => onNodeClick?.({}, { id: String(node.id) })}
              style={{
                left: `${(node.position as { x: number }).x}px`,
                top: `${(node.position as { y: number }).y}px`,
              }}
            >
              <NodeComponent
                data={node.data}
                id={node.id}
                selected={Boolean(node.selected)}
              />
            </div>
          );
        })}
        {children}
      </div>
    ),
    useReactFlow: () => ({
      fitView: () => Promise.resolve(true),
    }),
    getBezierPath: () => ["M0,0 C40,0 60,20 100,20"],
  };
});

describe("AgentCanvas", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("shows the orchestration empty state before any node has spawned", () => {
    const snapshot = buildWorkspaceSnapshot();
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    expect(
      screen.getByRole("heading", {
        name: /submit a goal to start the orchestration graph/i,
      }),
    ).toBeInTheDocument();
  });

  it("does not reserve the selected-node panel until a node is selected", () => {
    const runId = "run_panel_selection_1";
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: runId,
      run_summaries: [
        {
          id: runId,
          task_text_preview: "Inspect relay startup",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });

    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: runId,
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
    });

    expect(
      screen.queryByTestId("agent-canvas-detail-mode-selected"),
    ).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: "Planner, Planner node" }),
    );

    expect(
      screen.getByTestId("agent-canvas-detail-mode-selected"),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Canvas background" }));

    expect(
      screen.getByTestId("agent-canvas-flow").parentElement,
    ).toHaveAttribute("data-detail-open", "false");
  });

  it("appends spawned nodes and patches node state without losing interactivity", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Inspect relay startup",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "task_assigned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 2,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          task_text: "Break the goal into stages.",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 3,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "thinking",
          message: "Planner is breaking the task into stages.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 4,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          text: "Plan the work in stages.",
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 5,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          label: "Coder",
          spawn_order: 2,
          occurred_at: "2026-03-24T12:00:04Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "handoff_complete",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 6,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          from_agent_id: "agent_planner_1",
          to_agent_id: "agent_coder_2",
          reason: "planner_completed",
          occurred_at: "2026-03-24T12:00:05Z",
        },
      } as never);
    });

    expect(
      screen.getByRole("button", { name: /planner, planner node/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /coder, coder node/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("react-flow-edge-count")).toHaveTextContent("1");

    const plannerPosition = screen.getByTestId("node-position-agent_planner_1");
    const leftBefore = plannerPosition.style.left;
    const topBefore = plannerPosition.style.top;

    fireEvent.click(
      screen.getAllByRole("button", { name: /planner, planner node/i })[0],
    );

    expect(screen.getAllByText("Planner").length).toBeGreaterThan(0);
    expect(screen.getByText(/break the goal into stages/i)).toBeInTheDocument();
    expect(
      screen.getAllByText(/plan the work in stages/i).length,
    ).toBeGreaterThan(0);

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 7,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "completed",
          message: "Planner completed its task.",
          occurred_at: "2026-03-24T12:00:06Z",
        },
      } as never);
    });

    expect(within(plannerPosition).getByText("Completed")).toBeInTheDocument();
    expect(plannerPosition.style.left).toBe(leftBefore);
    expect(plannerPosition.style.top).toBe(topBefore);

    fireEvent.click(screen.getByRole("button", { name: /canvas background/i }));

    expect(
      screen.getByTestId("agent-canvas-flow").parentElement,
    ).toHaveAttribute("data-detail-open", "false");
  });

  it("keeps viewport controls available while orchestration nodes stream and fail", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Inspect relay startup",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_tester_3",
          sequence: 1,
          replay: false,
          role: "tester",
          model: "deepseek/deepseek-chat",
          label: "Tester",
          spawn_order: 3,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_tester_3",
          sequence: 2,
          replay: false,
          role: "tester",
          model: "deepseek/deepseek-chat",
          text: "Validate the plan against constraints.",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_error",
        payload: {
          code: "agent_generation_failed",
          message: "Tester could not finish the summary.",
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_tester_3",
          sequence: 3,
          replay: false,
          role: "tester",
          model: "deepseek/deepseek-chat",
          terminal: true,
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    const zoomIn = screen.getByRole("button", { name: /zoom in/i });
    const fitView = screen.getByRole("button", { name: /fit view/i });

    fireEvent.click(
      screen.getAllByRole("button", { name: /tester, tester node/i })[0],
    );

    expect(
      screen.getAllByText(/tester could not finish the summary/i).length,
    ).toBeGreaterThan(0);
    expect(zoomIn).toBeEnabled();
    expect(fitView).toBeEnabled();
  });

  it("renders loading and plain-language error states for the canvas shell", () => {
    const loadingSnapshot = buildWorkspaceSnapshot({
      ui_state: {
        history_state: "loading",
        canvas_state: "empty",
        save_state: "idle",
      },
    });
    const { rerender } = renderWithWorkspace(
      <WorkspaceCanvas activeSession={loadingSnapshot.sessions[0]} />,
      loadingSnapshot,
    );

    expect(
      screen.getByText(/relay is loading the orchestration canvas/i),
    ).toBeInTheDocument();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "error",
        payload: {
          code: "workspace_canvas_unavailable",
          message: "Relay could not load the orchestration canvas.",
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
    });

    rerender(<WorkspaceCanvas activeSession={loadingSnapshot.sessions[0]} />);
    expect(
      screen.getByText(/relay could not load the orchestration canvas/i),
    ).toBeInTheDocument();
  });

  it("keeps node streaming indicators active only while tokens continue arriving", () => {
    vi.useFakeTimers();

    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_streaming",
      run_summaries: [
        {
          id: "run_streaming",
          task_text_preview: "Inspect relay startup",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });

    const { container } = renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_streaming",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_streaming",
          agent_id: "agent_planner_1",
          sequence: 2,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          text: "Planner transcript.",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
    });

    const node = container.querySelector(".agent-canvas-node");
    expect(node).toHaveAttribute("data-streaming-active", "true");

    act(() => {
      vi.advanceTimersByTime(301);
    });

    expect(node).toHaveAttribute("data-streaming-active", "false");
  });

  it("renders active and settling handoff pulse states on the custom edge", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_handoff",
      run_summaries: [
        {
          id: "run_handoff",
          task_text_preview: "Inspect relay startup",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });

    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_handoff",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_handoff",
          agent_id: "agent_coder_2",
          sequence: 2,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          label: "Coder",
          spawn_order: 2,
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "handoff_start",
        payload: {
          session_id: "session_alpha",
          run_id: "run_handoff",
          agent_id: "agent_planner_1",
          sequence: 3,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          from_agent_id: "agent_planner_1",
          to_agent_id: "agent_coder_2",
          reason: "planner_completed",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    expect(
      screen
        .getByTestId("edge-agent_planner_1->agent_coder_2")
        .querySelector('.agent-canvas-edge-pulse[data-pulse-state="active"]'),
    ).not.toBeNull();

    act(() => {
      workspaceStore.handleEnvelope({
        type: "handoff_complete",
        payload: {
          session_id: "session_alpha",
          run_id: "run_handoff",
          agent_id: "agent_planner_1",
          sequence: 4,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          from_agent_id: "agent_planner_1",
          to_agent_id: "agent_coder_2",
          reason: "planner_completed",
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    expect(
      screen
        .getByTestId("edge-agent_planner_1->agent_coder_2")
        .querySelector('.agent-canvas-edge-pulse[data-pulse-state="settling"]'),
    ).not.toBeNull();
  });

  it("switches the detail panel between nodes while transcripts continue updating", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Inspect relay startup",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 2,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          label: "Coder",
          spawn_order: 2,
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 3,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          text: "Planner transcript.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 4,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          text: "Coder transcript.",
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    fireEvent.click(
      screen.getAllByRole("button", { name: /planner, planner node/i })[0],
    );
    expect(screen.getAllByText(/planner transcript\./i).length).toBeGreaterThan(
      0,
    );

    fireEvent.click(
      screen.getAllByRole("button", { name: /coder, coder node/i })[0],
    );
    expect(screen.getAllByText(/coder transcript\./i).length).toBeGreaterThan(
      0,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 5,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          text: " More output.",
          occurred_at: "2026-03-24T12:00:04Z",
        },
      } as never);
    });

    expect(
      screen.getAllByText(/coder transcript\. more output\./i).length,
    ).toBeGreaterThan(0);
  });

  it("renders long task and summary text inside dedicated detail regions", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Review the environment notes",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          started_at: "2026-03-24T12:00:00Z",
          completed_at: "2026-03-24T12:00:03Z",
          has_tool_activity: false,
        },
      ],
    });

    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 1,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          label: "Reviewer",
          spawn_order: 4,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "task_assigned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 2,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          task_text:
            "Original goal: add note to each variable in .env.example\n\nPlanner output: I'll help you add explanatory notes to each variable so the file stays readable even when each line gets very long and includes connection strings, tokens, and URLs.",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 3,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          message:
            "Reviewer condensed the orchestration into a readable handoff with long notes, URLs, and inline examples.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    fireEvent.click(
      screen.getAllByRole("button", { name: /reviewer, reviewer node/i })[0],
    );

    expect(
      screen.getByRole("region", { name: /selected node task/i }),
    ).toHaveClass("agent-canvas-detail-copy");
    expect(
      screen.getByRole("region", { name: /selected node summary/i }),
    ).toHaveClass("agent-canvas-detail-copy");
    expect(
      screen.getByRole("region", { name: /selected node transcript/i }),
    ).toHaveClass("agent-canvas-detail-copy");
  });

  it("renders markdown in the final run result and selected node summary", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Explain the env example",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          started_at: "2026-03-24T12:00:00Z",
          completed_at: "2026-03-24T12:00:03Z",
          has_tool_activity: false,
        },
      ],
    });

    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 1,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          label: "Reviewer",
          spawn_order: 4,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 2,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          message:
            "**Goal:** Explain each setting in `.env.example` with a readable checklist.",
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run_complete",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          sequence: 3,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          summary:
            "**Goal:** Explain the file clearly.\n\n1. Ask for the `.env.example` file.\n2. Add comments above each variable.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    const status = screen
      .getAllByRole("status")
      .find((element) => /Ask for the/i.test(element.textContent ?? ""));

    expect(status).toBeDefined();
    expect(
      within(status as HTMLElement).getByText("Goal:", { selector: "strong" }),
    ).toBeInTheDocument();
    expect(
      within(status as HTMLElement).getByText(".env.example", {
        selector: "code",
      }),
    ).toBeInTheDocument();
    expect(
      within(status as HTMLElement).getByText(/Ask for the/i),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getAllByRole("button", { name: /reviewer, reviewer node/i })[0],
    );

    const summaryRegion = screen.getByRole("region", {
      name: /selected node summary/i,
    });
    expect(
      within(summaryRegion).queryByText(/\*\*Goal:\*\*/i),
    ).not.toBeInTheDocument();
    expect(
      within(summaryRegion).getByText("Goal:", { selector: "strong" }),
    ).toBeInTheDocument();
    expect(
      within(summaryRegion).getByText(".env.example", { selector: "code" }),
    ).toBeInTheDocument();
  });

  it("renders markdown in the selected node transcript with transcript wrapping styles", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Explain the env example",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          started_at: "2026-03-24T12:00:00Z",
          completed_at: "2026-03-24T12:00:03Z",
          has_tool_activity: false,
        },
      ],
    });

    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 1,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          label: "Reviewer",
          spawn_order: 4,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "token",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 2,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          text: [
            "## Transcript summary",
            "",
            "- Includes security reminders for sensitive keys",
            "- Groups related configurations together",
            "",
            "```text",
            "Offers guidance on when to modify defaults",
            "```",
          ].join("\n"),
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_state_changed",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_reviewer_4",
          sequence: 3,
          replay: false,
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          message: "Reviewer finished the transcript.",
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    fireEvent.click(
      screen.getAllByRole("button", { name: /reviewer, reviewer node/i })[0],
    );

    const transcriptRegion = screen.getByRole("region", {
      name: /selected node transcript/i,
    });

    expect(transcriptRegion).toHaveClass("relay-transcript-copy");
    expect(transcriptRegion.querySelector(".relay-markdown")).not.toBeNull();
    expect(
      screen.getByRole("heading", { name: /transcript summary/i }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/includes security reminders/i),
    ).toBeInTheDocument();
    expect(
      transcriptRegion.querySelector(".relay-markdown-pre"),
    ).not.toBeNull();
  });

  it("shows a halted run reason without spawning downstream nodes after planner failure", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Stop after planner failure",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_error",
        payload: {
          code: "agent_generation_failed",
          message: "Planner could not break the goal into stages.",
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 2,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          terminal: true,
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run_error",
        payload: {
          code: "planner_required",
          message:
            "The run stopped because the planner did not complete and downstream work could not continue.",
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 3,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          terminal: true,
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    expect(
      screen.getByRole("alert", {
        name: "",
      }),
    ).toHaveTextContent(
      /planner did not complete and downstream work could not continue/i,
    );
    expect(
      screen.getByRole("button", { name: /planner, planner node/i }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /coder, coder node/i }),
    ).not.toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /planner, planner node/i }),
    );

    expect(
      screen.getAllByText(/planner could not break the goal into stages/i)
        .length,
    ).toBeGreaterThan(0);
    expect(screen.getAllByText(/run halt/i).length).toBeGreaterThan(0);
    expect(
      screen.getAllByText(
        /the run stopped because the planner did not complete and downstream work could not continue/i,
      ).length,
    ).toBeGreaterThan(1);
  });

  it("matches planner halt details when the run_error payload sends a blank agent_id", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Stop after planner failure",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run_error",
        payload: {
          code: "run_stage_failed",
          message:
            "The run stopped because Relay could not finish the planner stage.",
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "",
          sequence: 2,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          terminal: true,
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
    });

    fireEvent.click(
      screen.getByRole("button", { name: /planner, planner node/i }),
    );

    expect(screen.getAllByText(/run halt/i).length).toBeGreaterThan(0);
    expect(
      screen.getAllByText(
        /the run stopped because relay could not finish the planner stage/i,
      ).length,
    ).toBeGreaterThan(1);
  });

  it("surfaces clarification-required halts with a dedicated label", () => {
    const snapshot = buildWorkspaceSnapshot({
      active_run_id: "run_1",
      run_summaries: [
        {
          id: "run_1",
          task_text_preview: "Comment the env example",
          role: "planner",
          model: "anthropic/claude-opus-4",
          state: "active",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        },
      ],
    });
    renderWithWorkspace(
      <WorkspaceCanvas activeSession={snapshot.sessions[0]} />,
      snapshot,
    );

    act(() => {
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_planner_1",
          sequence: 1,
          replay: false,
          role: "planner",
          model: "anthropic/claude-opus-4",
          label: "Planner",
          spawn_order: 1,
          occurred_at: "2026-03-24T12:00:00Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_spawned",
        payload: {
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 2,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          label: "Coder",
          spawn_order: 2,
          occurred_at: "2026-03-24T12:00:01Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "agent_error",
        payload: {
          code: "coder_clarification_required",
          message:
            "The run stopped because the coder asked for user clarification instead of producing actionable output.",
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 3,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          terminal: true,
          occurred_at: "2026-03-24T12:00:02Z",
        },
      } as never);
      workspaceStore.handleEnvelope({
        type: "run_error",
        payload: {
          code: "coder_clarification_required",
          message:
            "The run stopped because the coder asked for user clarification instead of producing actionable output.",
          session_id: "session_alpha",
          run_id: "run_1",
          agent_id: "agent_coder_2",
          sequence: 4,
          replay: false,
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          terminal: true,
          occurred_at: "2026-03-24T12:00:03Z",
        },
      } as never);
    });

    expect(
      screen.getAllByText(/clarification required/i).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getByText(
        /relay stopped before continuing to downstream agents because one stage asked for more input instead of taking action/i,
      ),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /coder, coder node/i }));

    expect(
      screen
        .getByRole("button", { name: /coder, coder node/i })
        .closest("[data-state]"),
    ).toHaveAttribute("data-state", "clarification_required");
    expect(
      screen.getAllByRole("heading", { name: "Coder" }).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByText("Clarification required", { selector: "span" })
        .length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByText(/clarification required/i).length,
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByText(
        /the run stopped because the coder asked for user clarification instead of producing actionable output/i,
      ).length,
    ).toBeGreaterThan(0);
  });
});
