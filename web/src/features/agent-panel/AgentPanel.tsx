import type { AgentRunSummary, PreferencesView } from "@/shared/lib/workspace-protocol";
import type { PendingApproval, StoredRunEvent } from "@/shared/lib/workspace-store";
import { AgentCommandBar } from "@/features/agent-panel/AgentCommandBar";
import { RunHeader } from "@/features/agent-panel/RunHeader";
import { RunTimeline } from "@/features/agent-panel/RunTimeline";
import { ThoughtViewer } from "@/features/agent-panel/ThoughtViewer";
import {
  describeApprovalState,
  describeRunFailure,
  describeToolRunningState,
  formatToolName,
  isCancelledRun,
} from "@/features/agent-panel/runStatus";

interface AgentPanelProps {
  activeRunId: string;
  activeSessionId: string;
  preferences: PreferencesView;
  pendingApproval: PendingApproval | null;
  runEvents: StoredRunEvent[];
  runTranscript?: string;
  selectedRunId: string;
  selectedRunSummary: AgentRunSummary | null;
  onApprovalDecision: (toolCallId: string, decision: "approved" | "rejected") => void;
  onCancel: (runId: string) => void;
  onSubmit: (task: string) => void;
}

export function AgentPanel({ activeRunId, activeSessionId, preferences, pendingApproval, runEvents, runTranscript, selectedRunId, selectedRunSummary, onApprovalDecision, onCancel, onSubmit }: AgentPanelProps) {
  const transcript = runTranscript ?? runEvents
    .filter((event) => event.type === "token")
    .map((event) => ("text" in event.payload ? event.payload.text : ""))
    .join("");
  const hasTokenOutput = transcript.length > 0;
  const runSelected = Boolean(selectedRunSummary);
  const showingReplay = Boolean(selectedRunId) && selectedRunId !== activeRunId;
  const submitDisabled = !activeSessionId || Boolean(activeRunId);
  const helpMessage = describeHelpMessage({
    activeRunId,
    activeSessionId,
    hasTokenOutput,
    preferences,
    runSelected,
    runEvents,
    selectedRunId,
    selectedRunSummary,
    showingReplay,
    pendingApproval,
  });
  const rootWarning = !preferences.project_root_valid
    ? preferences.project_root_message || "Repository-reading tools stay blocked until Relay has a valid project root."
    : null;

  return (
    <section
      aria-labelledby="agent-panel-heading"
      className="panel-surface rounded-[2rem] p-5 shadow-idle"
    >
      <div className="space-y-4">
        <div>
          <p className="eyebrow">Live execution</p>
          <h2
            id="agent-panel-heading"
            className="mt-2 font-display text-3xl text-text"
          >
            Watch Relay work in order
          </h2>
          <p className="mt-3 max-w-3xl text-sm leading-6 text-text-muted">
            {helpMessage}
          </p>
        </div>

        {rootWarning ? (
          <div
            className="rounded-[1.5rem] border border-border bg-raised/70 p-4"
            role="status"
          >
            <p className="eyebrow">Project root</p>
            <p className="mt-2 text-sm leading-6 text-text-muted">
              {rootWarning}
            </p>
          </div>
        ) : null}

        {pendingApproval ? (
          <div
            className="rounded-[1.5rem] border border-border bg-raised/70 p-4"
            role="status"
          >
            <p className="eyebrow">Approval required</p>
            <p className="mt-2 text-sm leading-6 text-text">
              {pendingApproval.message}
            </p>
            <p className="mt-2 text-sm leading-6 text-text-muted">
              Tool: {formatToolName(pendingApproval.toolName)}
            </p>
            <div className="mt-3 flex flex-wrap gap-3">
              <button
                className="rounded-full border border-border bg-surface px-4 py-2 text-sm text-text transition duration-200 ease-out hover:border-brand-mid focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-brand)]"
                onClick={() =>
                  onApprovalDecision(pendingApproval.toolCallId, "approved")
                }
                type="button"
              >
                Approve tool
              </button>
              <button
                className="rounded-full border border-border bg-surface px-4 py-2 text-sm text-text transition duration-200 ease-out hover:border-[var(--color-error)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--color-brand)]"
                onClick={() =>
                  onApprovalDecision(pendingApproval.toolCallId, "rejected")
                }
                type="button"
              >
                Reject tool
              </button>
            </div>
          </div>
        ) : null}

        <AgentCommandBar
          disabled={submitDisabled}
          hasActiveRun={Boolean(activeRunId)}
          onCancel={() => onCancel(activeRunId)}
          onSubmit={onSubmit}
        />
        <RunHeader run={selectedRunSummary} />
        <ThoughtViewer
          pendingApproval={pendingApproval}
          run={selectedRunSummary}
          runEvents={runEvents}
          transcript={transcript}
        />
        <RunTimeline events={selectedRunId ? runEvents : []} />
      </div>
    </section>
  );
}

interface HelpMessageInput {
  activeRunId: string;
  activeSessionId: string;
  hasTokenOutput: boolean;
  preferences: PreferencesView;
  runSelected: boolean;
  runEvents: StoredRunEvent[];
  selectedRunId: string;
  selectedRunSummary: AgentRunSummary | null;
  showingReplay: boolean;
  pendingApproval: PendingApproval | null;
}

function describeHelpMessage({
  activeRunId,
  activeSessionId,
  hasTokenOutput,
  preferences,
  runSelected,
  runEvents,
  selectedRunId,
  selectedRunSummary,
  showingReplay,
  pendingApproval,
}: HelpMessageInput) {
  if (!activeSessionId) {
    return "Create or open a session before starting a live run.";
  }
  if (!preferences.openrouter_configured) {
    return "Save an OpenRouter API key in preferences before starting a run.";
  }
  if (selectedRunSummary?.state === "errored" && isCancelledRun(runEvents)) {
    return "This run was cancelled. Review the timeline for the cancellation point, then submit another task when ready.";
  }
  if (pendingApproval) {
    return describeApprovalState(pendingApproval, runEvents);
  }
  if (showingReplay && selectedRunSummary) {
    return `Reviewing saved run ${selectedRunSummary.id} in read-only mode.`;
  }
  if (activeRunId && !selectedRunId) {
    return "A Relay run is already active. Select it from history or wait for it to finish before submitting another task.";
  }
  if (!runSelected) {
    return "Submit a task to watch one Relay agent stream visible output in order.";
  }
  if (
    (selectedRunSummary?.state === "accepted" ||
      selectedRunSummary?.state === "thinking") &&
    !hasTokenOutput
  ) {
    return "Relay accepted the task and is waiting for the first visible provider output.";
  }
  if (selectedRunSummary?.state === "tool_running") {
    return describeToolRunningState(runEvents);
  }
  if (selectedRunSummary?.state === "approval_required") {
    return describeApprovalState(pendingApproval, runEvents);
  }
  if (selectedRunSummary?.state === "completed") {
    return hasTokenOutput
      ? "This run is complete. You can review the transcript and timeline or submit another task."
      : "This run is complete. Review the timeline for the ordered state changes and tool activity, then submit another task when ready.";
  }
  if (selectedRunSummary?.state === "errored") {
    return describeRunFailure(runEvents);
  }
  return "One specialized Relay agent runs at a time.";
}