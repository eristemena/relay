"use client";

import clsx from "clsx";
import type { ErrorPayload, WorkspaceStatusPayload } from "@/shared/lib/workspace-protocol";

interface WorkspaceStatusBannerProps {
  connectionState: "connecting" | "connected" | "closed";
  status: WorkspaceStatusPayload | null;
  error: ErrorPayload | null;
  projectRootMessage?: string;
  projectRootValid?: boolean;
  warnings: string[];
  compact?: boolean;
  embedded?: boolean;
}

export function hasWorkspaceStatusBanner(params: {
  connectionState: "connecting" | "connected" | "closed";
  status: WorkspaceStatusPayload | null;
  error: ErrorPayload | null;
  projectRootMessage?: string;
  projectRootValid?: boolean;
  warnings: string[];
}) {
  return !(
    !params.error &&
    !params.status &&
    params.warnings.length === 0 &&
    params.connectionState === "connected" &&
    (params.projectRootValid ?? true)
  );
}

export function WorkspaceStatusBanner({
  connectionState,
  status,
  error,
  projectRootMessage,
  projectRootValid = true,
  warnings,
  compact = false,
  embedded = false,
}: WorkspaceStatusBannerProps) {
  if (
    !hasWorkspaceStatusBanner({
      connectionState,
      status,
      error,
      projectRootMessage,
      projectRootValid,
      warnings,
    })
  ) {
    return null;
  }

  const hasProjectRootWarning =
    !projectRootValid && Boolean(projectRootMessage);
  const tone = error
    ? "error"
    : connectionState !== "connected" || hasProjectRootWarning
      ? "warning"
      : "info";
  const message =
    error?.message ??
    status?.message ??
    projectRootMessage ??
    warnings[0] ??
    "Connecting to the Relay workspace.";
  const title = error
    ? "Recoverable issue"
    : connectionState !== "connected"
      ? "Connection status"
      : hasProjectRootWarning
        ? "Project root needs attention"
        : "Workspace status";

  return (
    <div
      aria-live="polite"
      className={clsx(
        embedded
          ? "workspace-status-embedded"
          : compact
            ? "workspace-status-inline rounded-2xl px-4 py-3"
            : "panel-surface relative overflow-hidden rounded-2xl px-4 py-3",
        !embedded && tone === "error" && "shadow-error",
        !embedded && tone !== "error" && "shadow-idle",
      )}
      role={error ? "alert" : "status"}
    >
      <div
        className={clsx(
          "min-w-0",
          compact && "flex flex-wrap items-start gap-x-3 gap-y-2",
        )}
      >
        <p className={clsx("eyebrow", compact && "mt-1")}>{title}</p>
        <p className={clsx("text-sm leading-6 text-text", !compact && "mt-2")}>
          {message}
        </p>
      </div>
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
