"use client";

interface PreferencesStatusProps {
  hasCredentials: boolean;
  saveState: "idle" | "saving" | "saved" | "error";
}

export function PreferencesStatus({ hasCredentials, saveState }: PreferencesStatusProps) {
  const message =
    saveState === "saving"
      ? "Saving preferences"
      : saveState === "saved"
        ? "Preferences saved"
        : saveState === "error"
          ? "Preferences need attention"
          : "Preferences ready";

  return (
    <div className="rounded-3xl border border-border bg-raised/80 p-4">
      <p className="eyebrow">Preference status</p>
      <p className="mt-2 text-sm text-text">{message}</p>
      <p className="mt-2 text-sm text-text-muted">Stored credentials: {hasCredentials ? "present" : "not saved"}</p>
    </div>
  );
}
