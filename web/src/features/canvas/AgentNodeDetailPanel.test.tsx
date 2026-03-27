import { act, fireEvent, render, screen } from "@testing-library/react";
import { useState, type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { AgentNodeDetailPanel } from "@/features/canvas/AgentNodeDetailPanel";
import type { SelectedCanvasNodeView } from "@/features/canvas/canvasModel";

const defaultTokenUsage = {
  tokensUsed: null,
  contextLimit: null,
  usagePercent: null,
  tone: "unavailable" as const,
  summary: "Usage unavailable",
  detail: "Relay did not receive authoritative token usage for this agent.",
};

function buildSelectedNode(
  overrides: Partial<SelectedCanvasNodeView> = {},
): SelectedCanvasNodeView {
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
      tokenUsage: defaultTokenUsage,
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
    expect(
      screen.getByTestId("agent-canvas-detail-mode-empty"),
    ).toBeInTheDocument();

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
    expect(
      screen.getByText(/could not load this canvas detail view/i),
    ).toBeInTheDocument();

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
            tokenUsage: defaultTokenUsage,
            transcript: "Coder transcript.",
          },
        })}
      />,
    );

    expect(screen.getByRole("heading", { name: "Coder" })).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(
      screen.getAllByRole("button", { name: /close agent details/i }).length,
    ).toBeGreaterThan(0);
    expect(screen.getByText(/coder transcript\./i)).toBeInTheDocument();
    expect(
      screen.getByText(/coder needs a missing file path/i),
    ).toBeInTheDocument();
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
            tokenUsage: defaultTokenUsage,
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

  it("moves focus into the dialog, traps tab navigation, and restores focus on close", () => {
    function Harness(): ReactElement {
      const [selectedNode, setSelectedNode] =
        useState<SelectedCanvasNodeView | null>(null);

      return (
        <>
          <button onClick={() => setSelectedNode(null)} type="button">
            Return focus target
          </button>
          <AgentNodeDetailPanel
            haltAgentId={null}
            haltCode={null}
            haltMessage=""
            haltRole={null}
            onClose={() => setSelectedNode(null)}
            selectedNode={selectedNode}
          />
          <button
            onClick={() => setSelectedNode(buildSelectedNode())}
            type="button"
          >
            Open detail panel
          </button>
        </>
      );
    }

    render(<Harness />);

    const returnFocusTarget = screen.getByRole("button", {
      name: "Return focus target",
    });
    returnFocusTarget.focus();
    expect(returnFocusTarget).toHaveFocus();

    fireEvent.click(screen.getByRole("button", { name: "Open detail panel" }));

    const closeButton = screen.getByRole("button", {
      name: /close agent details/i,
    });
    expect(closeButton).toHaveFocus();

    act(() => {
      fireEvent.keyDown(document, { key: "Tab" });
    });
    expect(closeButton).toHaveFocus();

    act(() => {
      fireEvent.keyDown(document, { key: "Escape" });
    });

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(returnFocusTarget).toHaveFocus();
  });
});
