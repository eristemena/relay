"use client";

import { useState } from "react";
import type { PreferencesView } from "@/shared/lib/workspace-protocol";
import { PreferencesStatus } from "@/features/preferences/PreferencesStatus";

interface PreferencesPanelProps {
  onSave: (payload: {
    preferred_port: number;
    appearance_variant: string;
    open_browser_on_start: boolean;
    credentials?: Array<{ provider: string; label?: string; secret: string }>;
  }) => void;
  preferences: PreferencesView;
  saveState: "idle" | "saving" | "saved" | "error";
}

export function PreferencesPanel({ onSave, preferences, saveState }: PreferencesPanelProps) {
  const [preferredPort, setPreferredPort] = useState(String(preferences.preferred_port));
  const [appearanceVariant, setAppearanceVariant] = useState(preferences.appearance_variant);
  const [openBrowserOnStart, setOpenBrowserOnStart] = useState(preferences.open_browser_on_start);
  const [provider, setProvider] = useState("openai");
  const [label, setLabel] = useState("");
  const [secret, setSecret] = useState("");

  return (
    <section aria-labelledby="preferences-heading" className="panel-surface rounded-[2rem] p-5 shadow-idle">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="eyebrow">Preferences</p>
          <h2 id="preferences-heading" className="mt-2 font-display text-2xl text-text">
            Local settings
          </h2>
        </div>
        <PreferencesStatus hasCredentials={preferences.has_credentials} saveState={saveState} />
      </div>

      <form
        className="mt-6 grid gap-4 md:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          onSave({
            preferred_port: Number.parseInt(preferredPort, 10),
            appearance_variant: appearanceVariant,
            open_browser_on_start: openBrowserOnStart,
            credentials: secret
              ? [
                  {
                    provider,
                    label,
                    secret,
                  },
                ]
              : undefined,
          });
        }}
      >
        <label className="flex flex-col gap-2 text-sm text-text" htmlFor="preferred-port">
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

        <label className="flex flex-col gap-2 text-sm text-text" htmlFor="appearance-variant">
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

        <label className="flex flex-col gap-2 text-sm text-text" htmlFor="provider">
          Credential provider
          <input
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="provider"
            name="provider"
            onChange={(event) => setProvider(event.target.value)}
            value={provider}
          />
        </label>

        <label className="flex flex-col gap-2 text-sm text-text" htmlFor="credential-label">
          Credential label
          <input
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="credential-label"
            name="credential-label"
            onChange={(event) => setLabel(event.target.value)}
            value={label}
          />
        </label>

        <label className="md:col-span-2 flex flex-col gap-2 text-sm text-text" htmlFor="credential-secret">
          API credential secret
          <input
            className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
            id="credential-secret"
            name="credential-secret"
            onChange={(event) => setSecret(event.target.value)}
            type="password"
            value={secret}
          />
        </label>

        <label className="md:col-span-2 inline-flex items-center gap-3 rounded-2xl border border-border bg-raised px-4 py-3 text-sm text-text" htmlFor="open-browser-on-start">
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
          <button className="rounded-full border border-brand-mid bg-brand-mid px-5 py-3 font-medium text-text" type="submit">
            Save preferences
          </button>
        </div>
      </form>
    </section>
  );
}
