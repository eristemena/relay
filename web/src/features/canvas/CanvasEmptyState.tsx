"use client";

import type { ReactNode } from "react";

interface CanvasEmptyStateProps {
  sessionLabel: string;
  eyebrow?: string;
  title?: string;
  description?: string;
  children?: ReactNode;
}

export function CanvasEmptyState({
  sessionLabel,
  eyebrow = "Fresh workspace",
  title = `${sessionLabel} is ready for its first task.`,
  description = "This canvas stays intentionally quiet until the workspace gains activity. Use the workspace toolbar to review project context, inspect saved history, or adjust local preferences before the next run.",
  children,
}: CanvasEmptyStateProps) {
  return (
    <div className="panel-surface relative overflow-hidden rounded-[2rem] p-6 shadow-idle">
      <div className="max-w-xl">
        <p className="eyebrow">{eyebrow}</p>
        <h2 className="mt-3 font-display text-3xl tracking-tight text-text">
          {title}
        </h2>
        <p className="mt-4 text-base leading-7 text-text-muted">
          {description}
        </p>
        {children ? <div className="mt-5">{children}</div> : null}
      </div>
    </div>
  );
}
