"use client";

import { useEffect, useState } from "react";
import type { PreferencesView } from "@/shared/lib/workspace-protocol";
import { PreferencesStatus } from "@/features/preferences/PreferencesStatus";
import type { RepositoryBrowserState } from "@/shared/lib/workspace-store";

interface PreferencesPanelProps {
  onSave: (payload: {
    preferred_port: number;
    appearance_variant: string;
    open_browser_on_start: boolean;
    openrouter_api_key?: string;
    project_root?: string;
  }) => void;
  onBrowseRepository: (path?: string, showHidden?: boolean) => void;
  preferences: PreferencesView;
  repositoryBrowser: RepositoryBrowserState;
  saveState: "idle" | "saving" | "saved" | "error";
}

export function PreferencesPanel({
  onSave,
  onBrowseRepository,
  preferences,
  repositoryBrowser,
  saveState,
}: PreferencesPanelProps) {
  const [preferredPort, setPreferredPort] = useState(
    String(preferences.preferred_port),
  );
  const [appearanceVariant, setAppearanceVariant] = useState(
    preferences.appearance_variant,
  );
  const [openBrowserOnStart, setOpenBrowserOnStart] = useState(
    preferences.open_browser_on_start,
  );
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

  const hasConnectedRepository =
    preferences.project_root_valid && Boolean(preferences.project_root);
  const repositoryStatusTitle = hasConnectedRepository
    ? "Connected to a local Git repository"
    : preferences.project_root_configured
      ? "Saved repository needs attention"
      : "No repository connected";
  const repositoryStatusMessage = hasConnectedRepository
    ? preferences.project_root
    : preferences.project_root_message ||
      "Choose a local Git repository to enable repository-aware tools.";
  const repositoryBrowserEmpty =
    Boolean(repositoryBrowser.path) &&
    !repositoryBrowser.isLoading &&
    !repositoryBrowser.errorMessage &&
    repositoryBrowser.directories.length === 0;

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
          <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto]">
            <input
              className="rounded-2xl border border-border bg-raised px-4 py-3 text-text"
              id="project-root"
              name="project-root"
              onChange={(event) => setProjectRoot(event.target.value)}
              placeholder="/absolute/path/to/your/repository"
              value={projectRoot}
            />
            <button
              className="rounded-full border border-border bg-raised px-4 py-3 text-sm font-medium text-text transition duration-200 hover:border-brand"
              onClick={() =>
                onBrowseRepository(
                  projectRoot || undefined,
                  repositoryBrowser.showHidden,
                )
              }
              type="button"
            >
              Browse folders
            </button>
          </div>
        </label>

        <section
          aria-labelledby="repository-browser-heading"
          className="md:col-span-2 rounded-[1.5rem] border border-border bg-raised px-4 py-4"
        >
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="eyebrow">Repository browser</p>
              <h3
                id="repository-browser-heading"
                className="mt-2 font-display text-xl text-text"
              >
                Choose a local Git repository
              </h3>
              <p className="mt-2 text-sm leading-6 text-text-muted">
                Browse one directory level at a time, then choose a Git
                repository to fill the project root field.
              </p>
            </div>
            <label
              className="inline-flex items-center gap-2 text-sm text-text"
              htmlFor="show-hidden-folders"
            >
              <input
                checked={repositoryBrowser.showHidden}
                id="show-hidden-folders"
                onChange={(event) =>
                  onBrowseRepository(
                    projectRoot || repositoryBrowser.path || undefined,
                    event.target.checked,
                  )
                }
                type="checkbox"
              />
              Show hidden folders
            </label>
          </div>

          <div className="mt-4 rounded-2xl border border-border/80 bg-base/60 px-4 py-3">
            <p className="text-xs uppercase tracking-[0.28em] text-text-muted">
              Browsing path
            </p>
            <p className="mt-2 break-all font-mono text-sm text-text">
              {repositoryBrowser.path ||
                projectRoot ||
                "Choose a folder path to begin browsing."}
            </p>
          </div>

          <div className="mt-4 rounded-2xl border border-border bg-surface px-4 py-4">
            <p className="text-xs uppercase tracking-[0.28em] text-text-muted">
              Repository connection
            </p>
            <p className="mt-2 text-sm font-medium text-text">
              {repositoryStatusTitle}
            </p>
            <p className="mt-2 break-all text-sm leading-6 text-text-muted">
              {repositoryStatusMessage}
            </p>
          </div>

          {repositoryBrowser.isLoading ? (
            <p
              aria-live="polite"
              className="mt-4 text-sm text-text-muted"
              role="status"
            >
              Relay is loading folders from the selected path.
            </p>
          ) : null}

          {repositoryBrowser.errorMessage ? (
            <p className="mt-4 text-sm text-error" role="alert">
              {repositoryBrowser.errorMessage}
            </p>
          ) : null}

          {repositoryBrowserEmpty ? (
            <p
              aria-live="polite"
              className="mt-4 text-sm leading-6 text-text-muted"
              role="status"
            >
              Relay did not find any child folders here. Open a parent directory
              or show hidden folders to keep browsing.
            </p>
          ) : null}

          {repositoryBrowser.directories.length > 0 ? (
            <ul className="mt-4 grid gap-3" role="list">
              {repositoryBrowser.directories.map((directory) => (
                <li
                  className="repository-browser-entry flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-border bg-surface px-4 py-3"
                  key={directory.path}
                >
                  <div className="repository-browser-meta min-w-0">
                    <p className="truncate font-medium text-text">
                      {directory.name}
                    </p>
                    <p className="mt-1 break-all text-sm text-text-muted">
                      {directory.path}
                    </p>
                    <p className="mt-2 text-xs uppercase tracking-[0.24em] text-text-muted">
                      {directory.isGitRepository
                        ? "Git repository"
                        : "Directory"}
                    </p>
                  </div>
                  <div className="repository-browser-actions flex flex-wrap gap-2">
                    <button
                      className="rounded-full border border-border bg-raised px-4 py-2 text-sm text-text transition duration-200 hover:border-brand"
                      onClick={() =>
                        onBrowseRepository(
                          directory.path,
                          repositoryBrowser.showHidden,
                        )
                      }
                      type="button"
                    >
                      Open
                    </button>
                    {directory.isGitRepository ? (
                      <button
                        className="rounded-full border border-brand-mid bg-brand-mid px-4 py-2 text-sm font-medium text-text"
                        onClick={() => setProjectRoot(directory.path)}
                        type="button"
                      >
                        Use repository
                      </button>
                    ) : null}
                  </div>
                </li>
              ))}
            </ul>
          ) : null}
        </section>

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
