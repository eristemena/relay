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
});