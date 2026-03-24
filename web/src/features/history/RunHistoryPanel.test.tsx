import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RunHistoryPanel } from "@/features/history/RunHistoryPanel";

describe("RunHistoryPanel", () => {
  it("renders an empty state when no runs are saved", () => {
    render(
      <RunHistoryPanel
        activeRunId=""
        historyState="ready"
        runSummaries={[]}
        selectedRunId=""
        onOpen={() => undefined}
      />,
    );

    expect(screen.getByText(/no saved runs yet/i)).toBeInTheDocument();
    expect(
      screen.getByText(
        /completed, clarification-required, and errored agent tasks will appear here for replay/i,
      ),
    ).toBeInTheDocument();
  });

  it("opens a saved run when selected", () => {
    const onOpen = vi.fn();
    render(
      <RunHistoryPanel
        activeRunId=""
        historyState="ready"
        runSummaries={[
          {
            id: "run_1",
            task_text_preview: "Plan the next steps",
            role: "planner",
            model: "anthropic/claude-opus-4",
            state: "completed",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: false,
          },
        ]}
        selectedRunId=""
        onOpen={onOpen}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /plan the next steps/i }));
    expect(onOpen).toHaveBeenCalledWith("run_1");
  });

  it("renders orchestration replay states and keeps the selected run openable", () => {
    const onOpen = vi.fn();
    render(
      <RunHistoryPanel
        activeRunId="run_live"
        historyState="ready"
        runSummaries={[
          {
            id: "run_halted",
            task_text_preview: "Replay the halted orchestration",
            role: "reviewer",
            model: "anthropic/claude-sonnet-4-5",
            state: "halted",
            error_code: "reviewer_clarification_required",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: false,
          },
          {
            id: "run_live",
            task_text_preview: "Inspect the active orchestration",
            role: "planner",
            model: "anthropic/claude-opus-4",
            state: "active",
            started_at: "2026-03-23T12:05:00Z",
            has_tool_activity: false,
          },
        ]}
        selectedRunId="run_halted"
        onOpen={onOpen}
      />,
    );

    expect(
      screen.getByText(/clarification required • anthropic\/claude-sonnet-4-5/i),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Clarification required", { selector: "span" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/active • anthropic\/claude-opus-4/i),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole("button", { name: /replay the halted orchestration/i }),
    );
    expect(onOpen).toHaveBeenCalledWith("run_halted");
  });
});