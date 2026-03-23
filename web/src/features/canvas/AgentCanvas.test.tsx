import { fireEvent, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AgentCanvas } from "@/features/canvas/AgentCanvas";
import { renderIsolatedCanvas } from "@/shared/lib/test-helpers";

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
    ReactFlowProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
    ReactFlow: ({
      children,
      edges,
      nodeTypes,
      nodes,
      onNodeClick,
      onPaneClick,
    }: {
      children: React.ReactNode;
      edges: Array<{ id: string }>;
      nodeTypes: Record<string, (props: Record<string, unknown>) => React.ReactNode>;
      nodes: Array<Record<string, unknown>>;
      onNodeClick?: (event: unknown, node: { id: string }) => void;
      onPaneClick?: () => void;
    }) => (
      <div data-testid="react-flow-mock">
        <button aria-label="Canvas background" onClick={onPaneClick} type="button" />
        <div data-testid="react-flow-edge-count">{edges.length}</div>
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
  };
});

describe("AgentCanvas", () => {
  it("replaces the empty state after adding nodes from the toolbar", () => {
    renderIsolatedCanvas(<AgentCanvas sessionLabel="Inspect relay startup" />);

    expect(
      screen.getByRole("heading", {
        name: /start by placing the first agent on the canvas/i,
      }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /add node/i }));

    expect(screen.getByRole("alert")).toHaveTextContent(
      /choose an agent role before adding a node/i,
    );

    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "planner" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));

    expect(
      screen.getByRole("button", { name: /planner 1, planner node/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(/1 node and 0 handoffs/i);

    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "coder" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));

    expect(
      screen.getByRole("button", { name: /coder 2, coder node/i }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("react-flow-edge-count")).toHaveTextContent("1");
  });

  it("opens details, preserves coordinates on state changes, and clears selection from the canvas background", () => {
    renderIsolatedCanvas(<AgentCanvas sessionLabel="Inspect relay startup" />);

    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "planner" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));
    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "coder" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));

    const coderNode = screen.getByRole("button", {
      name: /coder 2, coder node/i,
    });
    const coderPosition = screen.getByTestId("node-position-node_2");
    const leftBefore = coderPosition.style.left;
    const topBefore = coderPosition.style.top;

    fireEvent.click(coderNode);

    expect(screen.getAllByText("Coder 2")).toHaveLength(2);
    expect(screen.getByText(/local-only detail panel/i)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/local node state/i), {
      target: { value: "thinking" },
    });
    fireEvent.click(screen.getByRole("button", { name: /apply state/i }));

    expect(within(coderPosition).getByText("Thinking")).toBeInTheDocument();
    expect(coderPosition.style.left).toBe(leftBefore);
    expect(coderPosition.style.top).toBe(topBefore);

    fireEvent.click(screen.getByRole("button", { name: /canvas background/i }));

    expect(
      screen.queryByText(/local-only detail panel/i),
    ).not.toBeInTheDocument();
  });

  it("keeps viewport controls available while nodes and local states change", () => {
    renderIsolatedCanvas(<AgentCanvas sessionLabel="Inspect relay startup" />);

    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "planner" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));
    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "coder" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));

    const zoomIn = screen.getByRole("button", { name: /zoom in/i });
    const fitView = screen.getByRole("button", { name: /fit view/i });

    fireEvent.click(
      screen.getByRole("button", { name: /coder 2, coder node/i }),
    );
    fireEvent.change(screen.getByLabelText(/agent role/i), {
      target: { value: "tester" },
    });
    fireEvent.click(screen.getByRole("button", { name: /add node/i }));

    expect(screen.getByText(/local-only detail panel/i)).toBeInTheDocument();
    expect(zoomIn).toBeEnabled();
    expect(fitView).toBeEnabled();
  });
});