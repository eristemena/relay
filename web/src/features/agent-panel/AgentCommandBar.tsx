"use client";

import { useState } from "react";

interface AgentCommandBarProps {
  disabled: boolean;
  hasActiveRun: boolean;
  onCancel: () => void;
  onSubmit: (task: string) => void;
}

export function AgentCommandBar({ disabled, hasActiveRun, onCancel, onSubmit }: AgentCommandBarProps) {
  const [task, setTask] = useState("");

  return (
    <form
      className="stream-card rounded-[1.75rem] p-5"
      onSubmit={(event) => {
        event.preventDefault();
        const trimmed = task.trim();
        if (!trimmed) {
          return;
        }
        onSubmit(trimmed);
        setTask("");
      }}
    >
      <label className="flex flex-col gap-3 text-sm text-text" htmlFor="agent-task-input">
        <span className="eyebrow">Agent task</span>
        <textarea
          className="min-h-28 rounded-[1.5rem] border border-border bg-raised px-4 py-4 text-text"
          disabled={disabled}
          id="agent-task-input"
          name="agent-task-input"
          onChange={(event) => setTask(event.target.value)}
          placeholder="Describe the task you want Relay to handle."
          value={task}
        />
      </label>
      <div className="mt-4 flex items-center justify-between gap-3">
        <p className="text-sm text-text-muted">Relay chooses one role automatically and streams its visible output here.</p>
        <div className="flex items-center gap-3">
          {hasActiveRun ? (
            <button
              className="rounded-full border border-border bg-raised px-5 py-3 font-medium text-text disabled:cursor-not-allowed disabled:opacity-60"
              onClick={onCancel}
              type="button"
            >
              Cancel run
            </button>
          ) : null}
          <button className="rounded-full border border-brand-mid bg-brand-mid px-5 py-3 font-medium text-text disabled:cursor-not-allowed disabled:opacity-60" disabled={disabled} type="submit">
            Run task
          </button>
        </div>
      </div>
    </form>
  );
}