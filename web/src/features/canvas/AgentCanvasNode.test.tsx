import { act, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentCanvasNode } from "@/features/canvas/AgentCanvasNode";

const defaultTokenUsage = {
  tokensUsed: null,
  contextLimit: null,
  usagePercent: null,
  tone: "unavailable" as const,
  summary: "Usage unavailable",
  detail: "Relay did not receive authoritative token usage for this agent.",
};

vi.mock("@xyflow/react", () => ({
  Handle: () => <span data-testid="canvas-handle" />,
  Position: {
    Left: "left",
    Right: "right",
  },
}));

describe("AgentCanvasNode", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps the streaming ring active only within the silence window", () => {
    vi.useFakeTimers();
    const props = {
      data: {
        label: "Planner",
        role: "planner",
        roleLabel: "Planner",
        state: "streaming",
        stateRevision: 1,
        readCount: 0,
        proposalCount: 0,
        summary: "Planner streamed the first token.",
        tokenUsage: defaultTokenUsage,
      },
      selected: false,
    } as unknown as ComponentProps<typeof AgentCanvasNode>;

    const { container, rerender } = render(<AgentCanvasNode {...props} />);

    const node = container.querySelector(".agent-canvas-node");
    expect(node).toHaveAttribute("data-streaming-active", "true");

    act(() => {
      vi.advanceTimersByTime(299);
    });
    expect(node).toHaveAttribute("data-streaming-active", "true");

    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(node).toHaveAttribute("data-streaming-active", "false");

    rerender(
      <AgentCanvasNode
        {...({
          ...props,
          data: {
            ...props.data,
            stateRevision: 2,
            summary: "Planner streamed the next token.",
          },
        } as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    expect(node).toHaveAttribute("data-streaming-active", "true");
  });

  it("clears the streaming ring immediately when the node leaves streaming state", () => {
    vi.useFakeTimers();
    const props = {
      data: {
        label: "Planner",
        role: "planner",
        roleLabel: "Planner",
        state: "streaming",
        stateRevision: 1,
        readCount: 0,
        proposalCount: 0,
        summary: "Planner streamed the first token.",
        tokenUsage: defaultTokenUsage,
      },
      selected: true,
    } as unknown as ComponentProps<typeof AgentCanvasNode>;

    const { container, rerender, unmount } = render(
      <AgentCanvasNode {...props} />,
    );

    rerender(
      <AgentCanvasNode
        {...({
          ...props,
          data: {
            ...props.data,
            state: "completed",
            stateRevision: 2,
            summary: "Planner finished.",
          },
        } as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    expect(container.querySelector(".agent-canvas-node")).toHaveAttribute(
      "data-streaming-active",
      "false",
    );

    unmount();
    act(() => {
      vi.advanceTimersByTime(400);
    });
  });

  it("renders compact repository activity counts on the node", () => {
    const props = {
      data: {
        label: "Coder",
        role: "coder",
        roleLabel: "Coder",
        state: "thinking",
        stateRevision: 3,
        readCount: 2,
        proposalCount: 1,
        summary: "Coder is validating the patch.",
        tokenUsage: defaultTokenUsage,
      },
      selected: false,
    } as unknown as ComponentProps<typeof AgentCanvasNode>;

    const { getByText } = render(<AgentCanvasNode {...props} />);

    expect(getByText("2 files read · 1 change proposed")).toBeInTheDocument();
  });

  it("renders unavailable token usage copy when authoritative usage is missing", () => {
    render(
      <AgentCanvasNode
        {...({
          data: {
            label: "Coder",
            role: "coder",
            roleLabel: "Coder",
            state: "thinking",
            stateRevision: 1,
            readCount: 0,
            proposalCount: 0,
            summary: "Waiting for the next tool result.",
            tokenUsage: defaultTokenUsage,
          },
          selected: false,
        } as unknown as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    expect(screen.getByText("Usage unavailable")).toBeInTheDocument();
    expect(
      screen.getByText(/did not receive authoritative token usage/i),
    ).toBeInTheDocument();
  });

  it("exposes dialog trigger state for keyboard and assistive technology users", () => {
    const { rerender } = render(
      <AgentCanvasNode
        {...({
          data: {
            label: "Planner",
            role: "planner",
            roleLabel: "Planner",
            state: "thinking",
            stateRevision: 1,
            readCount: 0,
            proposalCount: 0,
            summary: "Planner is waiting.",
            tokenUsage: defaultTokenUsage,
          },
          selected: false,
        } as unknown as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    const button = screen.getByRole("button", {
      name: "Planner, Planner node",
    });
    expect(button).toHaveAttribute("aria-haspopup", "dialog");
    expect(button).toHaveAttribute("aria-expanded", "false");
    expect(button).toHaveAttribute("aria-pressed", "false");

    rerender(
      <AgentCanvasNode
        {...({
          data: {
            label: "Planner",
            role: "planner",
            roleLabel: "Planner",
            state: "thinking",
            stateRevision: 2,
            readCount: 0,
            proposalCount: 0,
            summary: "Planner selected.",
            tokenUsage: defaultTokenUsage,
          },
          selected: true,
        } as unknown as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    expect(button).toHaveAttribute("aria-expanded", "true");
    expect(button).toHaveAttribute("aria-pressed", "true");
  });

  it("renders count-only and critical token usage states", () => {
    const { rerender, container } = render(
      <AgentCanvasNode
        {...({
          data: {
            label: "Reviewer",
            role: "reviewer",
            roleLabel: "Reviewer",
            state: "completed",
            stateRevision: 2,
            readCount: 0,
            proposalCount: 0,
            summary: "Reviewer finished.",
            tokenUsage: {
              tokensUsed: 4812,
              contextLimit: null,
              usagePercent: null,
              tone: "count_only",
              summary: "4,812 used",
              detail: "Context window unavailable for this model.",
            },
          },
          selected: false,
        } as unknown as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    expect(screen.getByText("4,812 used")).toBeInTheDocument();
    expect(screen.getByText(/context window unavailable/i)).toBeInTheDocument();
    expect(container.querySelector(".agent-canvas-token-bar")).toHaveAttribute(
      "data-token-tone",
      "count_only",
    );

    rerender(
      <AgentCanvasNode
        {...({
          data: {
            label: "Reviewer",
            role: "reviewer",
            roleLabel: "Reviewer",
            state: "completed",
            stateRevision: 3,
            readCount: 0,
            proposalCount: 0,
            summary: "Reviewer finished.",
            tokenUsage: {
              tokensUsed: 1200,
              contextLimit: 1000,
              usagePercent: 1,
              tone: "critical",
              summary: "1,200 / 1,000",
              detail:
                "Usage exceeded the known context window and is capped at 100%.",
            },
          },
          selected: false,
        } as unknown as ComponentProps<typeof AgentCanvasNode>)}
      />,
    );

    expect(screen.getByText("1,200 / 1,000")).toBeInTheDocument();
    expect(screen.getByText(/capped at 100%/i)).toBeInTheDocument();
    expect(container.querySelector(".agent-canvas-token-bar")).toHaveAttribute(
      "data-token-tone",
      "critical",
    );
  });
});
