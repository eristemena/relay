import { act, render } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentCanvasNode } from "@/features/canvas/AgentCanvasNode";

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
        summary: "Planner streamed the first token.",
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
        summary: "Planner streamed the first token.",
      },
      selected: true,
    } as unknown as ComponentProps<typeof AgentCanvasNode>;

    const { container, rerender, unmount } = render(<AgentCanvasNode {...props} />);

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
});