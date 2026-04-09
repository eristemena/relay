import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { RunHistoryPanel } from "@/features/history/RunHistoryPanel";

describe("RunHistoryPanel", () => {
  it("renders an empty state when no runs are saved", () => {
    render(
      <RunHistoryPanel
        activeRunId=""
        historyState="ready"
        onExport={() => undefined}
        runSummaries={[]}
        onQuery={() => undefined}
        onReplayControl={() => undefined}
        replayState={null}
        runHistoryQuery={null}
        selectedRun={null}
        selectedRunDetails={null}
        selectedRunId=""
        onOpen={() => undefined}
      />,
    );

    expect(
      screen.getByText(/no saved runs found for the active project/i),
    ).toBeInTheDocument();
  });

  it("toggles all-project mode and renders project labels", () => {
    const onQuery = vi.fn();

    render(
      <RunHistoryPanel
        activeRunId=""
        historyState="ready"
        onExport={() => undefined}
        onOpen={() => undefined}
        onQuery={onQuery}
        onReplayControl={() => undefined}
        replayState={null}
        runHistoryQuery={{ all_projects: true }}
        runSummaries={[
          {
            id: "run_1",
            task_text_preview: "Review project history",
            role: "reviewer",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-23T12:00:00Z",
            has_tool_activity: false,
            project_label: "relay",
            project_root: "/tmp/relay",
          },
        ]}
        selectedRun={null}
        selectedRunDetails={null}
        selectedRunId=""
      />,
    );

    const toggle = screen.getByRole("checkbox", {
      name: /include runs from all known projects/i,
    });
    expect(toggle).toBeChecked();
    expect(screen.getByText(/project relay/i)).toBeInTheDocument();

    fireEvent.click(toggle);

    expect(onQuery).toHaveBeenCalledWith({
      all_projects: false,
      query: undefined,
      file_path: undefined,
      date_from: undefined,
      date_to: undefined,
    });
  });

  it("opens a saved run when selected", () => {
    const onOpen = vi.fn();
    render(
      <RunHistoryPanel
        activeRunId=""
        historyState="ready"
        onExport={() => undefined}
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
        onQuery={() => undefined}
        onReplayControl={() => undefined}
        replayState={null}
        runHistoryQuery={null}
        selectedRun={null}
        selectedRunDetails={null}
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
        exportState={null}
        historyState="ready"
        onExport={() => undefined}
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
        onQuery={() => undefined}
        onReplayControl={() => undefined}
        replayState={null}
        runHistoryQuery={null}
        selectedRun={null}
        selectedRunDetails={null}
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

  it("applies filters and shows filtered diff review for the selected run", () => {
    const onQuery = vi.fn();
    render(
      <RunHistoryPanel
        activeRunId="run_history_1"
        exportState={{
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "completed",
          export_path: "/Users/example/.relay/exports/review.md",
          generated_at: "2026-03-24T12:03:00Z",
        }}
        historyState="ready"
        onExport={() => undefined}
        onOpen={() => undefined}
        onQuery={onQuery}
        onReplayControl={() => undefined}
        replayState={{
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "paused",
          cursor_ms: 1200,
          duration_ms: 5000,
          speed: 1,
          selected_timestamp: "2026-03-24T12:00:02Z",
        }}
        runHistoryQuery={null}
        runSummaries={[
          {
            id: "run_history_1",
            generated_title: "Review approval flow",
            task_text_preview: "Audit approval review flow",
            role: "reviewer",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-24T12:00:00Z",
            has_tool_activity: true,
            agent_count: 3,
            final_status: "completed",
            has_file_changes: true,
          },
        ]}
        selectedRun={{
          id: "run_history_1",
          generated_title: "Review approval flow",
          task_text_preview: "Audit approval review flow",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: true,
          agent_count: 3,
          final_status: "completed",
          has_file_changes: true,
        }}
        selectedRunDetails={{
          session_id: "session_alpha",
          run_id: "run_history_1",
          generated_title: "Review approval flow",
          final_status: "completed",
          agent_count: 3,
          change_records: [
            {
              tool_call_id: "call_1",
              path: "README.md",
              original_content: "before\n",
              proposed_content: "after\n",
              approval_state: "applied",
              occurred_at: "2026-03-24T12:00:01Z",
            },
            {
              tool_call_id: "call_2",
              path: "docs/future.md",
              original_content: "later before\n",
              proposed_content: "later after\n",
              approval_state: "applied",
              occurred_at: "2026-03-24T12:00:03Z",
            },
          ],
        }}
        selectedRunId="run_history_1"
      />,
    );

    fireEvent.change(screen.getByLabelText(/keyword/i), {
      target: { value: "approval" },
    });
    fireEvent.click(screen.getByRole("button", { name: /apply filters/i }));
    expect(onQuery).toHaveBeenCalledWith({
      all_projects: false,
      query: "approval",
      file_path: undefined,
      date_from: undefined,
      date_to: undefined,
    });
    expect(
      screen.getByText(/showing 1 of 2 recorded changes through/i),
    ).toBeInTheDocument();
    expect(screen.getByText(/readme\.md/i)).toBeInTheDocument();
    expect(screen.getByText(/^applied$/i)).toBeInTheDocument();
    expect(screen.queryByText(/docs\/future\.md/i)).not.toBeInTheDocument();
  });

  it("treats missing change records in selected run details as empty", () => {
    render(
      <RunHistoryPanel
        activeRunId="run_history_1"
        exportState={null}
        historyState="ready"
        onExport={() => undefined}
        onOpen={() => undefined}
        onQuery={() => undefined}
        onReplayControl={() => undefined}
        replayState={{
          session_id: "session_alpha",
          run_id: "run_history_1",
          status: "paused",
          cursor_ms: 1200,
          duration_ms: 5000,
          speed: 1,
          selected_timestamp: "2026-03-24T12:00:02Z",
        }}
        runHistoryQuery={null}
        runSummaries={[
          {
            id: "run_history_1",
            generated_title: "Review approval flow",
            task_text_preview: "Audit approval review flow",
            role: "reviewer",
            model: "anthropic/claude-sonnet-4-5",
            state: "completed",
            started_at: "2026-03-24T12:00:00Z",
            has_tool_activity: true,
            agent_count: 3,
            final_status: "completed",
            has_file_changes: true,
          },
        ]}
        selectedRun={{
          id: "run_history_1",
          generated_title: "Review approval flow",
          task_text_preview: "Audit approval review flow",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: true,
          agent_count: 3,
          final_status: "completed",
          has_file_changes: true,
        }}
        selectedRunDetails={
          {
            session_id: "session_alpha",
            run_id: "run_history_1",
            generated_title: "Review approval flow",
            final_status: "completed",
            agent_count: 3,
          } as never
        }
        selectedRunId="run_history_1"
      />,
    );

    expect(
      screen.getByText(/does not include recorded file changes/i),
    ).toBeInTheDocument();
  });
});