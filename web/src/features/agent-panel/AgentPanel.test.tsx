import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AgentPanel } from "@/features/agent-panel/AgentPanel";

describe("AgentPanel", () => {
  it("renders the empty state before the first task", () => {
    render(
      <AgentPanel
        activeRunId=""
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        selectedRunId=""
        selectedRunSummary={null}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(screen.getAllByText(/submit a task to watch one relay agent/i)).toHaveLength(2);
  });

  it("shows the waiting-for-output message after a run is accepted", () => {
    render(
      <AgentPanel
        activeRunId="run_1"
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Review the websocket flow",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "accepted",
          started_at: "2026-03-23T12:00:00Z",
          has_tool_activity: false,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(screen.getByText(/relay accepted the task and is waiting for the first visible provider output/i)).toBeInTheDocument();
    expect(screen.getByText(/relay accepted this task and is waiting for the first visible provider output/i)).toBeInTheDocument();
  });

  it("surfaces the project root warning", () => {
    render(
      <AgentPanel
        activeRunId=""
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "",
          project_root_configured: false,
          project_root_valid: false,
          project_root_message:
            "Repository-reading tools stay disabled until Relay has a valid project_root in config.toml.",
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        selectedRunId=""
        selectedRunSummary={null}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(
      screen.getByText(/repository-reading tools stay disabled/i),
    ).toBeInTheDocument();
  });

  it("shows the missing-key guidance when the OpenRouter key is missing", () => {
    render(
      <AgentPanel
        activeRunId=""
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: false,
          openrouter_configured: false,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        selectedRunId=""
        selectedRunSummary={null}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(
      screen.getByText(
        /save an openrouter api key in preferences before starting a run/i,
      ),
    ).toBeInTheDocument();
  });

  it("shows explicit cancelled-run messaging instead of the generic error copy", () => {
    render(
      <AgentPanel
        activeRunId=""
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[
          {
            type: "error",
            payload: {
              code: "run_cancelled",
              message: "Relay cancelled the active run.",
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 2,
              replay: false,
              role: "reviewer",
              model: "anthropic/claude-sonnet-4-5",
              terminal: true,
              occurred_at: "2026-03-23T12:00:02Z",
            },
          },
        ]}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Review the websocket flow",
          role: "reviewer",
          model: "anthropic/claude-sonnet-4-5",
          state: "errored",
          started_at: "2026-03-23T12:00:00Z",
          completed_at: "2026-03-23T12:00:02Z",
          has_tool_activity: false,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(screen.getByText(/this run was cancelled\. review the timeline for the cancellation point/i)).toBeInTheDocument();
    expect(screen.getByText(/this run was cancelled before any visible output arrived/i)).toBeInTheDocument();
    expect(screen.getByText(/run cancelled: relay stopped the active run before it produced more output/i)).toBeInTheDocument();
  });

  it("describes clarification-required replay runs in the banner and placeholder copy", () => {
    render(
      <AgentPanel
        activeRunId="run_live"
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        selectedRunId="run_saved"
        selectedRunSummary={{
          id: "run_saved",
          task_text_preview: "Explain why the orchestration halted",
          role: "explainer",
          model: "google/gemini-2.0-flash-001",
          state: "halted",
          error_code: "coder_clarification_required",
          started_at: "2026-03-23T12:00:00Z",
          completed_at: "2026-03-23T12:00:03Z",
          has_tool_activity: false,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(
      screen.getByText(
        /reviewing saved run run_saved in read-only mode\. clarification was required before relay could continue this run/i,
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /clarification required before relay could continue this run\. update the task or missing context, then rerun when ready/i,
      ),
    ).toBeInTheDocument();
  });

  it("shows the approval prompt and forwards approval decisions", () => {
    const onApprovalDecision = vi.fn();

    render(
      <AgentPanel
        activeRunId="run_1"
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={{
          sessionId: "session_alpha",
          runId: "run_1",
          toolCallId: "call_1",
          toolName: "write_file",
          inputPreview: { path: "README.md" },
          message:
            "Relay needs approval before it can write files inside the configured project root.",
          occurredAt: "2026-03-23T12:00:01Z",
        }}
        runEvents={[]}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Update the README",
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          state: "approval_required",
          started_at: "2026-03-23T12:00:00Z",
          has_tool_activity: true,
        }}
        onApprovalDecision={onApprovalDecision}
      />,
    );

    expect(screen.getAllByText(/approval required/i)).toHaveLength(2);
    expect(screen.getAllByText(/relay needs approval before it can write files/i)).toHaveLength(3);

    fireEvent.click(screen.getByRole("button", { name: /approve tool/i }));
    fireEvent.click(screen.getByRole("button", { name: /reject tool/i }));

    expect(onApprovalDecision).toHaveBeenNthCalledWith(1, "call_1", "approved");
    expect(onApprovalDecision).toHaveBeenNthCalledWith(2, "call_1", "rejected");
  });

  it("explains the active tool step while the transcript is waiting", () => {
    render(
      <AgentPanel
        activeRunId="run_1"
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[
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
              tool_name: "search_codebase",
              input_preview: { query: "workspace store" },
              occurred_at: "2026-03-23T12:00:02Z",
            },
          },
        ]}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Trace the workspace store",
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          state: "tool_running",
          started_at: "2026-03-23T12:00:00Z",
          has_tool_activity: true,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(
      screen.getAllByText(/relay is running search codebase now/i),
    ).toHaveLength(2);
  });

  it("renders visible output as markdown inside a bounded transcript region", () => {
    const transcript = [
      "## Server Configuration",
      "",
      "I'll help you update `.env.example` with clearer notes.",
      "",
      "```env",
      "PORT=3000",
      "```",
    ].join("\n");

    const { container } = render(
      <AgentPanel
        activeRunId="run_1"
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        runTranscript={transcript}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Document the environment file",
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          state: "thinking",
          started_at: "2026-03-24T12:00:00Z",
          has_tool_activity: false,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    const transcriptRegion = screen.getByRole("region", {
      name: /visible output transcript/i,
    });

    expect(transcriptRegion).toHaveClass("relay-transcript-copy");
    expect(transcriptRegion.querySelector(".relay-markdown")).not.toBeNull();
    expect(
      screen.getByRole("heading", { name: /server configuration/i }),
    ).toBeInTheDocument();
    expect(screen.getByText(".env.example")).toBeInTheDocument();
    expect(screen.getByText("PORT=3000")).toBeInTheDocument();
    expect(container.querySelector(".live-cursor")).not.toBeNull();
  });

  it("applies transcript-specific wrapping styles to markdown preformatted content", () => {
    const transcript = [
      "Summary:",
      "",
      "    - Includes security reminders for sensitive keys",
      "    - Groups related configurations together",
      "    - Offers guidance on when to modify defaults",
    ].join("\n");

    const { container } = render(
      <AgentPanel
        activeRunId="run_1"
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[]}
        runTranscript={transcript}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Document the environment file",
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          state: "completed",
          started_at: "2026-03-24T12:00:00Z",
          completed_at: "2026-03-24T12:00:10Z",
          has_tool_activity: false,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(
      screen.getByText(/includes security reminders/i),
    ).toBeInTheDocument();
    expect(
      container.querySelector(".relay-transcript-markdown .relay-markdown-pre"),
    ).not.toBeNull();
  });

  it("explains approval-rejection failures in plain language", () => {
    render(
      <AgentPanel
        activeRunId=""
        activeSessionId="session_alpha"
        preferences={{
          preferred_port: 4747,
          appearance_variant: "midnight",
          has_credentials: true,
          openrouter_configured: true,
          project_root: "/tmp/project",
          project_root_configured: true,
          project_root_valid: true,
          agent_models: {
            planner: "anthropic/claude-opus-4",
            coder: "anthropic/claude-sonnet-4-5",
            reviewer: "anthropic/claude-sonnet-4-5",
            tester: "deepseek/deepseek-chat",
            explainer: "google/gemini-2.0-flash-001",
          },
          open_browser_on_start: true,
        }}
        pendingApproval={null}
        runEvents={[
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
              tool_name: "write_file",
              status: "rejected",
              result_preview: {
                message:
                  "Relay blocked the tool call because approval was rejected",
              },
              occurred_at: "2026-03-23T12:00:03Z",
            },
          },
          {
            type: "error",
            payload: {
              code: "run_failed",
              message: "Relay could not complete the agent run.",
              session_id: "session_alpha",
              run_id: "run_1",
              sequence: 4,
              replay: false,
              role: "coder",
              model: "anthropic/claude-sonnet-4-5",
              terminal: true,
              occurred_at: "2026-03-23T12:00:04Z",
            },
          },
        ]}
        selectedRunId="run_1"
        selectedRunSummary={{
          id: "run_1",
          task_text_preview: "Update the README",
          role: "coder",
          model: "anthropic/claude-sonnet-4-5",
          state: "errored",
          started_at: "2026-03-23T12:00:00Z",
          completed_at: "2026-03-23T12:00:04Z",
          has_tool_activity: true,
        }}
        onApprovalDecision={() => undefined}
      />,
    );

    expect(
      screen.getByText(
        /this run hit a blocked write file step after approval was rejected/i,
      ),
    ).toBeInTheDocument();
  });
});