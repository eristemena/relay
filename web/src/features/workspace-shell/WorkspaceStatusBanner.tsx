"use client";

import clsx from "clsx";
import type { ErrorPayload, WorkspaceStatusPayload } from "@/shared/lib/workspace-protocol";

interface WorkspaceStatusBannerProps {
  connectionState: "connecting" | "connected" | "closed";
  status: WorkspaceStatusPayload | null;
  error: ErrorPayload | null;
  warnings: string[];
}

export function WorkspaceStatusBanner({ connectionState, status, error, warnings }: WorkspaceStatusBannerProps) {
  if (!error && !status && warnings.length === 0 && connectionState === "connected") {
    return null;
  }

  const tone = error ? "error" : connectionState !== "connected" ? "warning" : "info";
  const message = error?.message ?? status?.message ?? warnings[0] ?? "Connecting to the Relay workspace.";
  const title = error
    ? "Recoverable issue"
    : connectionState !== "connected"
      ? "Connection status"
      : "Workspace status";

  return (
    <div
      aria-live="polite"
      className={clsx(
        "panel-surface relative overflow-hidden rounded-2xl px-4 py-3",
        tone === "error" && "shadow-error",
        tone !== "error" && "shadow-idle",
      )}
      role={error ? "alert" : "status"}
    >
      <p className="eyebrow">{title}</p>
      <p className="mt-2 text-sm leading-6 text-text">{message}</p>
      {warnings.length > 1 ? (
        <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-text-muted">
          {warnings.slice(1).map((warning) => (
            <li key={warning}>{warning}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
