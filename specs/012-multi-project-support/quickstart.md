# Quickstart: Multi-Project Support

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- A fresh local Relay SQLite database created with the multi-project schema
- At least two readable local project roots for manual switching validation

## Development Setup

If you have an older local Relay database from before this feature, delete it before running the validation flow so Relay recreates the schema cleanly.

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
go test ./internal/orchestrator/workspace ./internal/handlers/ws ./internal/storage/sqlite ./internal/app ./cmd/relay
npm --prefix web install
npm --prefix web run typecheck
```

Focused frontend validation after implementation:

```bash
npm --prefix web test -- src/features/workspace-shell/WorkspaceShell.test.tsx src/features/history/RunHistoryPanel.test.tsx src/shared/lib/workspace-store.test.ts
```

Focused backend and integration validation after implementation:

```bash
go test ./internal/orchestrator/workspace ./internal/handlers/ws ./internal/storage/sqlite -run 'Test.*(Project|Bootstrap|History|Switch)'
go test ./tests/integration -run 'Test.*Project'
```

Switch-performance validation target after implementation:

```bash
go test ./tests/integration -run 'Test.*ProjectSwitch'
```

## Run Relay

Start Relay from the current working directory and let it use that directory as the active project root:

```bash
./bin/relay serve --no-browser
```

Start Relay with an explicit project root:

```bash
./bin/relay serve --no-browser --root /absolute/path/to/project
```

## Expected Behavior

- Relay resolves the startup project root using `--root` first, then the current working directory.
- The first time a project root is opened, Relay creates its project-scoped saved context automatically.
- The workspace header shows the active project root at all times.
- A project switcher in the header lists known roots and marks the active one.
- Switching projects rehydrates the selected project's canvas, history, and repository tree without restarting Relay.
- The History tab shows runs for the active project by default.
- Enabling the all-project history mode expands the result set without changing the active project.
- Attempting to switch projects while the current project still has a non-terminal run shows a clear blocking message.

## Manual Validation Flow

1. Start Relay inside one project directory without `--root` and confirm the header shows that directory as the active project root.
2. Confirm Relay creates or reuses persisted project-scoped context for that root without any manual setup action.
3. Start at least one run, then stop or complete it so the project has visible history.
4. Restart Relay with `--root` pointing at a second project and confirm the header now shows the second root.
5. Confirm distinct persisted project-scoped context is created automatically for the second root.
6. Open the project switcher and switch back to the first known root without restarting Relay.
7. Confirm the canvas, run history, and repository tree all update to the first project's scoped state and no stale data from the second project remains visible.
8. Open the History tab and confirm only runs from the active project are shown by default.
9. Enable the all-project history mode and confirm runs from both roots appear with clear project identity labels.
10. Disable the all-project mode and confirm the list returns to the active project's runs only.
11. Start a new run in one project and, while it is still non-terminal, attempt to switch projects. Confirm Relay blocks the switch with a human-readable explanation.
12. Start Relay with an invalid `--root` path and confirm startup fails clearly instead of silently falling back to another directory.
13. Repeat the switch flow at least 20 times and confirm at least 19 switches complete within 2 seconds from initiating the switch to the selected project's header, history, canvas, and repository surfaces rendering.

## Focused Test Commands

Backend project-context scoping, history queries, and switch blocking:

```bash
go test ./internal/orchestrator/workspace ./internal/storage/sqlite ./internal/handlers/ws ./internal/app ./cmd/relay -run 'Test.*(ProjectRoot|ProjectSwitch|HistoryScope|ServeRoot)'
```

Frontend project switcher and store reset behavior:

```bash
npm --prefix web test -- src/features/workspace-shell/WorkspaceShell.test.tsx src/features/history/RunHistoryPanel.test.tsx src/shared/lib/workspace-store.test.ts
```

Type safety validation:

```bash
npm --prefix web run typecheck
```

## Failure Recovery Expectations

- If two launch paths normalize to the same absolute cleaned root, Relay treats them as one known project.
- If a known project path is unavailable, the switcher leaves the current project active and shows a plain-language error.
- If a project switch happens after the previous project had visible run history or canvas state, stale run documents and repository state are cleared before the new project's data is rendered.
- If the all-project history toggle is enabled, switching projects updates the active-project indicator and project-scoped surfaces without silently disabling the chosen history mode.