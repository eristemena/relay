"use client";

import { useEffect, useState } from "react";
import type { PreferencesView } from "@/shared/lib/workspace-protocol";
import { PreferencesStatus } from "@/features/preferences/PreferencesStatus";

interface PreferencesPanelProps {
  onSave: (payload: {
    preferred_port: number;
    appearance_variant: string;
    open_browser_on_start: boolean;
    openrouter_api_key?: string;
    project_root?: string;
  }) => void;
  preferences: PreferencesView;
  saveState: "idle" | "saving" | "saved" | "error";
}

export function PreferencesPanel({ onSave, preferences, saveState }: PreferencesPanelProps) {
  const [preferredPort, setPreferredPort] = useState(String(preferences.preferred_port));
  const [appearanceVariant, setAppearanceVariant] = useState(preferences.appearance_variant);
  const [openBrowserOnStart, setOpenBrowserOnStart] = useState(preferences.open_browser_on_start);
  const [projectRoot, setProjectRoot] = useState(preferences.project_root);
  const [secret, setSecret] = useState("");

  useEffect(() => {
    setPreferredPort(String(preferences.preferred_port));
  }, [preferences.preferred_port]);

  useEffect(() => {
    setAppearanceVariant(preferences.appearance_variant);
  }, [preferences.appearance_variant]);

  useEffect(() => {
    setOpenBrowserOnStart(preferences.open_browser_on_start);
  }, [preferences.open_browser_on_start]);

  useEffect(() => {
    setProjectRoot(preferences.project_root);
  }, [preferences.project_root]);

  return (
    <section
      aria-labelledby="preferences-heading"
      className="panel-surface rounded-[2rem] p-5 shadow-idle"
    >
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="eyebrow">Preferences</p>
          <h2
            id="preferences-heading"
            className="mt-2 font-display text-2xl text-text"
          >
            Local settings
          </h2>
        </div>
        <PreferencesStatus
          openRouterConfigured={preferences.openrouter_configured}
          projectRootValid={preferences.project_root_valid}
          saveState={saveState}
        />
      </div>

      <form
        className="mt-6 grid gap-4 md:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          onSave({
            preferred_port: Number.parseInt(preferredPort, 10),
            appearance_variant: appearanceVariant,
            open_browser_on_start: openBrowserOnStart,
            openrouter_api_key: secret || undefined,
            project_root: projectRoot,
          });
        }}
      >
        <label
          className="flex flex-col gap-2 text-sm text-text"
          htmlFor="preferred-port"
        >
          Preferred port
          <input
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="preferred-port"
            inputMode="numeric"
            name="preferred-port"
            onChange={(event) => setPreferredPort(event.target.value)}
            value={preferredPort}
          />
        </label>

        <label
          className="flex flex-col gap-2 text-sm text-text"
          htmlFor="appearance-variant"
        >
          Appearance variant
          <select
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="appearance-variant"
            name="appearance-variant"
            onChange={(event) => setAppearanceVariant(event.target.value)}
            value={appearanceVariant}
          >
            <option value="midnight">Midnight</option>
            <option value="graphite">Graphite</option>
          </select>
        </label>

        <label
          className="md:col-span-2 flex flex-col gap-2 text-sm text-text"
          htmlFor="project-root"
        >
          Project root
          <input
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="project-root"
            name="project-root"
            onChange={(event) => setProjectRoot(event.target.value)}
            placeholder="/absolute/path/to/your/repository"
            value={projectRoot}
          />
        </label>

        <label
          className="md:col-span-2 flex flex-col gap-2 text-sm text-text"
          htmlFor="openrouter-api-key"
        >
          OpenRouter API key
          <input
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="openrouter-api-key"
            name="openrouter-api-key"
            onChange={(event) => setSecret(event.target.value)}
            placeholder={
              preferences.openrouter_configured
                ? "Saved key stays hidden until you replace it"
                : "or-your-key-here"
            }
            type="password"
            value={secret}
          />
        </label>

        <p className="md:col-span-2 text-sm leading-6 text-text-muted">
          Relay keeps the API key server-side and only exposes whether it is
          configured. Repository-reading tools stay disabled until the project
          root is valid.
        </p>

        <label
          className="md:col-span-2 inline-flex items-center gap-3 rounded-2xl border border-border bg-raised px-4 py-3 text-sm text-text"
          htmlFor="open-browser-on-start"
        >
          <input
            checked={openBrowserOnStart}
            id="open-browser-on-start"
            name="open-browser-on-start"
            onChange={(event) => setOpenBrowserOnStart(event.target.checked)}
            type="checkbox"
          />
          Open the browser automatically on startup
        </label>

        <div className="md:col-span-2 flex justify-end">
          <button
            className="rounded-full border border-brand-mid bg-brand-mid px-5 py-3 font-medium text-text"
            type="submit"
          >
            Save preferences
          </button>
        </div>
      </form>
    </section>
  );
}
