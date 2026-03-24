import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RunTimeline } from "./RunTimeline";

describe("RunTimeline", () => {
  it("shows the empty state when only token events are present", () => {
    render(
      <RunTimeline
        events={[
          {
            type: "token",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 1,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              text: "alpha",
              occurred_at: "2026-03-23T12:00:00Z",
            },
          },
        ]}
      />,
    );

    expect(screen.getByText(/state changes, tool calls, and terminal events will appear here/i)).toBeInTheDocument();
  });

  it("renders tool events and hides token-only rows", () => {
    render(
      <RunTimeline
        events={[
          {
            type: "token",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 1,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              text: "alpha",
              occurred_at: "2026-03-23T12:00:00Z",
            },
          },
          {
            type: "tool_call",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 2,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              tool_call_id: "call_1",
              tool_name: "read_file",
              input_preview: { path: "README.md" },
              occurred_at: "2026-03-23T12:00:01Z",
            },
          },
          {
            type: "tool_result",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 3,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              tool_call_id: "call_1",
              tool_name: "read_file",
              status: "completed",
              result_preview: { summary: "Loaded file content." },
              occurred_at: "2026-03-23T12:00:02Z",
            },
          },
        ]}
      />,
    );

    expect(screen.queryByText("alpha")).not.toBeInTheDocument();
    expect(screen.getByText("Tool call")).toBeInTheDocument();
    expect(screen.getByText("Tool result")).toBeInTheDocument();
    expect(screen.getAllByText("read_file")).toHaveLength(2);
  });

  it("renders the explicit cancelled-run timeline copy", () => {
    render(
      <RunTimeline
        events={[
          {
            type: "error",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 4,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              code: "run_cancelled",
              message: "Relay cancelled the active run.",
              terminal: true,
              occurred_at: "2026-03-23T12:00:03Z",
            },
          },
        ]}
      />,
    );

    expect(screen.getByText(/run cancelled: relay stopped the active run before it produced more output/i)).toBeInTheDocument();
  });

  it("renders the planner run_error message instead of a generic placeholder", () => {
    render(
      <RunTimeline
        events={[
          {
            type: "run_error",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              agent_id: "agent_planner_1",
              sequence: 5,
              replay: false,
              role: "planner",
              model: "anthropic/claude-opus-4",
              code: "planner_required",
              message:
                "The run stopped because the planner did not complete and downstream work could not continue.",
              terminal: true,
              occurred_at: "2026-03-23T12:00:04Z",
            },
          },
        ]}
      />,
    );

    expect(
      screen.getByText(
        /the run stopped because the planner did not complete and downstream work could not continue/i,
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText(/run event recorded/i)).not.toBeInTheDocument();
  });

  it("prefixes clarification-required timeline rows with the dedicated label", () => {
    render(
      <RunTimeline
        events={[
          {
            type: "run_error",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              agent_id: "agent_coder_2",
              sequence: 6,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              code: "coder_clarification_required",
              message:
                "The run stopped because the coder asked for user clarification instead of producing actionable output.",
              terminal: true,
              occurred_at: "2026-03-23T12:00:05Z",
            },
          },
        ]}
      />,
    );

    expect(
      screen.getByText(
        /clarification required: the run stopped because the coder asked for user clarification instead of producing actionable output/i,
      ),
    ).toBeInTheDocument();
  });

  it("renders the orchestration agent_error message for a failed node", () => {
    render(
      <RunTimeline
        events={[
          {
            type: "agent_error",
            payload: {
              session_id: "session_alpha",
              run_id: "run_1",
              agent_id: "agent_planner_1",
              sequence: 4,
              replay: false,
              role: "planner",
              model: "anthropic/claude-opus-4",
              code: "provider_error",
              message:
                "OpenRouter returned a planner failure before any visible output arrived.",
              terminal: true,
              occurred_at: "2026-03-23T12:00:03Z",
            },
          },
        ]}
      />,
    );

    expect(
      screen.getByText(
        /openrouter returned a planner failure before any visible output arrived/i,
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText(/run event recorded/i)).not.toBeInTheDocument();
  });
});