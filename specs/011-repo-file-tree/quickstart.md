# Quickstart: Repository File Tree Sidebar

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- A valid local Git repository configured in Relay as the connected project root

## Development Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
go test ./internal/orchestrator/workspace ./internal/storage/sqlite ./internal/handlers/ws ./internal/tools
npm --prefix web install
npm --prefix web run typecheck
```

Focused frontend validation after implementation:

```bash
npm --prefix web test -- workspace-store.test.ts WorkspaceShell.test.tsx treeModel.test.ts
```

Focused integration validation after protocol and touch-tracking changes:

```bash
go test ./tests/integration -run 'TestRepositoryFileTree_'
```

## Run Relay

Start Relay in development mode:

```bash
make dev
```

Optional health check after startup:

```bash
curl -sf http://127.0.0.1:4747/api/healthz
```

## Expected Behavior

- The existing left sidebar continues to handle run browsing, while the right-side workspace detail rail beside the graph shows File Tree directly during live runs.
- When the developer reopens a saved run, the same right rail exposes top-level tabs for Historical Replay and File Tree.
- When a valid repository is connected, the File Tree panel loads the recursive structure from the repo root after `.gitignore` filtering, while initially rendering only top-level entries and one nested level.
- Expanding a folder reveals deeper descendants from the already-loaded flat path list without opening files.
- Any successful `read_file` tool call marks the corresponding file as touched immediately.
- Any `write_file` proposal marks the corresponding file as touched with `proposed` state before approval is granted or denied.
- Clicking an agent node narrows the tree to that agent's touched files, and clearing the selection restores the workspace-wide view.
- The tree remains read-only and never opens editors, applies diffs, or runs commands.

## Manual Validation Flow

1. Start Relay with a valid connected repository and confirm the workspace opens normally.
2. Start a live run and confirm the right-side detail rail shows File Tree directly beside the graph.
3. Confirm the File Tree panel loads with top-level entries and one nested level visible, while deeper folders remain collapsed until expanded.
4. Open run history from the left toolbar, reopen a saved run, and confirm the same right rail now exposes top-level tabs for Historical Replay and File Tree.
5. Switch the right rail from the Historical Replay tab to the File Tree tab.
6. Start an orchestration run that reads at least one file and confirm the matching file gains a touched indicator without reloading the page.
7. Trigger a `write_file` proposal and confirm the matching file gains a touched indicator before approval is decided.
8. Click an agent node that touched files and confirm the File Tree panel narrows to only that agent's touched paths.
9. Click an agent node that has not touched any files and confirm the panel shows an explicit no-files-touched empty state.
10. Clear the selected agent and confirm the full tree returns with all current-run touched markers preserved.
11. Reconnect the workspace socket during an active run and confirm the tree, expanded folders, and touched markers are restored without duplicate rows.
12. Click file and folder rows in the tree and confirm no file editor, diff application, or command execution occurs.

## Focused Test Commands

Backend tree caching, touched-file persistence, and protocol behavior:

```bash
go test ./internal/orchestrator/workspace ./internal/storage/sqlite ./internal/handlers/ws ./internal/tools -run 'Test.*(RepositoryTree|TouchedFile|FileTouched|Bootstrap)'
```

Frontend right-rail panel switching, tree rendering, and store behavior:

```bash
npm --prefix web test -- workspace-store.test.ts WorkspaceShell.test.tsx treeModel.test.ts
```

Type safety validation:

```bash
npm --prefix web run typecheck
```

## Failure Recovery Expectations

- If the repository tree cannot be built for the connected repository, the repository tab shows a plain-language error instead of a blank panel.
- If the active agent has not touched any files, the filtered tree shows an explicit empty state rather than silently reverting to the full tree.
- If the same file is touched multiple times by the same agent and touch type, the sidebar continues showing one file entry rather than duplicate rows.
- If the workspace reconnects while the repository tab is open, Relay restores the touched-file snapshot for the active run and then resumes live updates.
- If the repository contains very large or deep directory structures, Relay still renders an initial shallow tree rather than blocking on full nested DOM creation, and ignored paths such as `node_modules` stay out of the tree when `.gitignore` excludes them.