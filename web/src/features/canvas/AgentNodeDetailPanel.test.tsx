import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AgentNodeDetailPanel } from "@/features/canvas/AgentNodeDetailPanel";
import type { SelectedCanvasNodeView } from "@/features/canvas/canvasModel";

function buildSelectedNode(overrides: Partial<SelectedCanvasNodeView> = {}): SelectedCanvasNodeView {
  return {
    id: "agent_planner_1",
    label: "Planner",
    role: "planner",
    state: "thinking",
    stateRevision: 1,
    details: {
      currentStateLabel: "Thinking",
      incomingFrom: [],
      outgoingTo: ["Coder"],
      proposedChanges: [],
      readPaths: [],
      summary: "Planner is sequencing the work.",
      taskText: "Break the goal into stages.",
      transcript: "Planner transcript.",
    },
    ...overrides,
  };
}

describe("AgentNodeDetailPanel", () => {
  it("renders explicit empty, loading, and error states in plain language", () => {
    const { rerender } = render(
      <AgentNodeDetailPanel
        haltAgentId={null}
        haltCode={null}
        haltMessage=""
        haltRole={null}
        selectedNode={null}
      />,
    );

    expect(screen.getByText(/inspect an agent/i)).toBeInTheDocument();
    expect(screen.getByTestId("agent-canvas-detail-mode-empty")).toBeInTheDocument();

    rerender(
      <AgentNodeDetailPanel
        errorMessage="Relay could not load this canvas detail view."
        haltAgentId={null}
        haltCode={null}
        haltMessage=""
        haltRole={null}
        selectedNode={null}
      />,
    );

    expect(screen.getByText(/canvas details unavailable/i)).toBeInTheDocument();
    expect(screen.getByText(/could not load this canvas detail view/i)).toBeInTheDocument();

    rerender(
      <AgentNodeDetailPanel
        haltAgentId={null}
        haltCode={null}
        haltMessage=""
        haltRole={null}
        isLoading
        selectedNode={null}
      />,
    );

    expect(screen.getByText(/loading agent details/i)).toBeInTheDocument();
  });

  it("switches to the latest selected node without mixing content", () => {
    const { rerender } = render(
      <AgentNodeDetailPanel
        haltAgentId={null}
        haltCode={null}
        haltMessage=""
        haltRole={null}
        selectedNode={buildSelectedNode()}
      />,
    );

    expect(screen.getByText("Planner")).toBeInTheDocument();
    expect(screen.getByText(/planner transcript\./i)).toBeInTheDocument();

    rerender(
      <AgentNodeDetailPanel
        haltAgentId={null}
        haltCode={null}
        haltMessage=""
        haltRole={null}
        selectedNode={buildSelectedNode({
          id: "agent_coder_2",
          label: "Coder",
          role: "coder",
          state: "streaming",
          stateRevision: 2,
          details: {
            currentStateLabel: "Streaming",
            errorMessage: "Coder needs a missing file path.",
            incomingFrom: ["Planner"],
            outgoingTo: ["Reviewer"],
            proposedChanges: [],
            readPaths: [],
            summary: "Coder is writing the change.",
            taskText: "Implement the requested patch.",
            transcript: "Coder transcript.",
          },
        })}
      />,
    );

    expect(
      screen.getByRole("heading", { name: "Coder" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(
      screen.getAllByRole("button", { name: /close agent details/i }).length,
    ).toBeGreaterThan(0);
    expect(screen.getByText(/coder transcript\./i)).toBeInTheDocument();
    expect(screen.getByText(/coder needs a missing file path/i)).toBeInTheDocument();
  });

  it("renders read files and proposal approval outcomes for the selected node", () => {
    render(
      <AgentNodeDetailPanel
        haltAgentId={null}
        haltCode={null}
        haltMessage=""
        haltRole={null}
        selectedNode={buildSelectedNode({
          details: {
            currentStateLabel: "Thinking",
            incomingFrom: ["Planner"],
            outgoingTo: ["Reviewer"],
            proposedChanges: [
              {
                path: "web/src/features/canvas/canvasModel.ts",
                toolCallId: "call_write",
                approvalState: "applied",
              },
            ],
            readPaths: ["internal/agents/coder.go", "web/src/app/page.tsx"],
            summary: "Coder finished the change.",
            taskText: "Implement the requested patch.",
            transcript: "Coder transcript.",
          },
        })}
      />,
    );

    expect(screen.getByText("internal/agents/coder.go")).toBeInTheDocument();
    expect(screen.getByText("web/src/app/page.tsx")).toBeInTheDocument();
    expect(
      screen.getByText("web/src/features/canvas/canvasModel.ts"),
    ).toBeInTheDocument();
    expect(screen.getByText(/applied to the repository/i)).toBeInTheDocument();
  });
});