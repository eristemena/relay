"use client";

import { useState } from "react";

interface AgentCommandBarProps {
  disabled: boolean;
  hasActiveRun: boolean;
  onCancel: () => void;
  onSubmit: (task: string) => void;
  panelClassName?: string;
}

export function AgentCommandBar({
  disabled,
  hasActiveRun,
  onCancel,
  onSubmit,
  panelClassName,
}: AgentCommandBarProps) {
  const [task, setTask] = useState("");
  const [isExpanded, setIsExpanded] = useState(false);
  const expanded = isExpanded || task.trim().length > 0;

  return (
    <form
      className={`stream-card rounded-[1.75rem] p-4 ${panelClassName ?? ""}`.trim()}
      data-expanded={expanded ? "true" : "false"}
      onSubmit={(event) => {
        event.preventDefault();
        const trimmed = task.trim();
        if (!trimmed) {
          return;
        }
        onSubmit(trimmed);
        setTask("");
        setIsExpanded(false);
      }}
    >
      <label className="agent-command-label" htmlFor="agent-task-input">
        Agent task
      </label>
      <div className="agent-command-row">
        <div aria-hidden="true" className="agent-command-mark">
          <svg fill="none" height="28" viewBox="0 0 28 28" width="28">
            <path
              d="M14 2.75L16.78 10.22L24.25 13L16.78 15.78L14 23.25L11.22 15.78L3.75 13L11.22 10.22L14 2.75Z"
              fill="currentColor"
              opacity="0.95"
            />
            <path
              d="M22.25 4.75L23.26 7.49L26 8.5L23.26 9.51L22.25 12.25L21.24 9.51L18.5 8.5L21.24 7.49L22.25 4.75Z"
              fill="currentColor"
              opacity="0.72"
            />
          </svg>
        </div>
        <textarea
          aria-expanded={expanded}
          className={`agent-command-input ${expanded ? "agent-command-input-expanded" : "agent-command-input-collapsed"}`}
          disabled={disabled}
          id="agent-task-input"
          name="agent-task-input"
          onBlur={() => {
            if (task.trim().length === 0) {
              setIsExpanded(false);
            }
          }}
          onChange={(event) => {
            setTask(event.target.value);
            if (event.target.value.trim().length > 0) {
              setIsExpanded(true);
            }
          }}
          onFocus={() => setIsExpanded(true)}
          placeholder="Ask Relay to code, refactor, or debug..."
          rows={expanded ? 3 : 1}
          value={task}
        />
        <div className="agent-command-actions">
          {hasActiveRun ? (
            <button
              aria-label="Cancel run"
              className="agent-command-cancel"
              onClick={onCancel}
              type="button"
            >
              Cancel
            </button>
          ) : null}
          <button
            aria-label="Run task"
            className="agent-command-submit disabled:cursor-not-allowed disabled:opacity-60"
            disabled={disabled}
            type="submit"
          >
            <span className="agent-command-submit-label">Run</span>
            <svg fill="none" height="18" viewBox="0 0 18 18" width="18">
              <path
                d="M9 14.25V3.75M9 3.75L4.5 8.25M9 3.75L13.5 8.25"
                stroke="currentColor"
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="1.8"
              />
            </svg>
          </button>
        </div>
      </div>
    </form>
  );
}