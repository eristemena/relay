import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AnimatedHandoffEdge } from "@/features/canvas/AnimatedHandoffEdge";

vi.mock("@xyflow/react", () => ({
  getBezierPath: () => ["M0,0 C20,0 20,40 40,40"],
}));

describe("AnimatedHandoffEdge", () => {
  it("renders an active pulse overlay when a handoff is in progress", () => {
    const { container } = render(
      <svg>
        <AnimatedHandoffEdge
          data={{ pulseState: "active" }}
          id="planner->coder"
          source="planner"
          sourcePosition={"right" as never}
          sourceX={0}
          sourceY={0}
          target="coder"
          targetPosition={"left" as never}
          targetX={40}
          targetY={40}
        />
      </svg>,
    );

    expect(container.querySelector(".agent-canvas-edge-pulse")).not.toBeNull();
    expect(container.querySelector('[data-pulse-state="active"]')).not.toBeNull();
  });

  it("omits the pulse overlay when the edge is idle", () => {
    const { container } = render(
      <svg>
        <AnimatedHandoffEdge
          data={{ pulseState: "idle" }}
          id="planner->coder"
          source="planner"
          sourcePosition={"right" as never}
          sourceX={0}
          sourceY={0}
          target="coder"
          targetPosition={"left" as never}
          targetX={40}
          targetY={40}
        />
      </svg>,
    );

    expect(container.querySelector(".agent-canvas-edge-pulse")).toBeNull();
  });
});