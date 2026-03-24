import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AgentCommandBar } from "@/features/agent-panel/AgentCommandBar";

describe("AgentCommandBar", () => {
  it("submits a trimmed task", () => {
    const onSubmit = vi.fn();

    render(
      <AgentCommandBar
        disabled={false}
        hasActiveRun={false}
        onCancel={() => undefined}
        onSubmit={onSubmit}
      />,
    );

    fireEvent.change(screen.getByLabelText(/agent task/i), {
      target: { value: "  Review the websocket flow  " },
    });
    fireEvent.click(screen.getByRole("button", { name: /run task/i }));

    expect(onSubmit).toHaveBeenCalledWith("Review the websocket flow");
    expect(screen.getByPlaceholderText(/ask relay to code, refactor, or debug/i)).toBeInTheDocument();
  });

  it("shows the cancel action for an active run", () => {
    const onCancel = vi.fn();

    render(
      <AgentCommandBar
        disabled={true}
        hasActiveRun={true}
        onCancel={onCancel}
        onSubmit={() => undefined}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /cancel run/i }));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: /cancel/i })).toBeInTheDocument();
  });

  it("starts collapsed and expands on focus", () => {
    render(
      <AgentCommandBar
        disabled={false}
        hasActiveRun={false}
        onCancel={() => undefined}
        onSubmit={() => undefined}
      />,
    );

    const input = screen.getByLabelText(/agent task/i);

    expect(input).toHaveAttribute("rows", "1");
    expect(input).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText(/agent task/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /run task/i })).toBeInTheDocument();

    fireEvent.focus(input);

    expect(input).toHaveAttribute("rows", "3");
    expect(input).toHaveAttribute("aria-expanded", "true");
  });

  it("collapses again on blur when empty", () => {
    render(
      <AgentCommandBar
        disabled={false}
        hasActiveRun={false}
        onCancel={() => undefined}
        onSubmit={() => undefined}
      />,
    );

    const input = screen.getByLabelText(/agent task/i);

    fireEvent.focus(input);
    fireEvent.blur(input);

    expect(input).toHaveAttribute("rows", "1");
    expect(input).toHaveAttribute("aria-expanded", "false");
  });

  it("stays expanded after text entry", () => {
    render(
      <AgentCommandBar
        disabled={false}
        hasActiveRun={false}
        onCancel={() => undefined}
        onSubmit={() => undefined}
      />,
    );

    const input = screen.getByLabelText(/agent task/i);

    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "Keep this open" } });
    fireEvent.blur(input);

    expect(input).toHaveAttribute("rows", "3");
    expect(input).toHaveAttribute("aria-expanded", "true");
  });
});