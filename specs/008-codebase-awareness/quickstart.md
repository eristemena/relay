# Quickstart: Codebase Awareness

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- A local Git repository available on the same machine as Relay
- Existing Relay workspace bootstrapped locally

## Development Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
go mod download
npm --prefix web install
npm --prefix web run typecheck
go test ./internal/tools ./internal/orchestrator/workspace ./internal/handlers/ws ./internal/config
```

This setup resolves the `go-git` backend dependency plus the shipped `monaco-editor` frontend package used by the codebase-awareness surfaces.

Focused frontend validation after implementation:

```bash
npm --prefix web test -- src/features/preferences/PreferencesPanel.test.tsx src/features/approvals/ApprovalReviewPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx src/shared/lib/workspace-store.test.ts
```

Focused integration validation after protocol and persistence changes:

```bash
go test ./tests/integration -run 'TestCodebaseAwareness_|TestToolCallOrdering_|TestWorkspaceSessions_'
```

## Run Relay

Start Relay with a repository path from the command line:

```bash
RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747 --project-root /absolute/path/to/local/repo
```

Or start normally and connect a repository from the UI folder picker or project-root control:

```bash
make dev
```

Optional health check after startup:

```bash
curl -sf http://127.0.0.1:4747/api/healthz
```

## Expected Behavior

- Relay accepts exactly one local Git repository as the connected codebase context.
- Repo-aware tools become available only after the selected path validates as a local Git repository.
- `write_file` requests surface a Monaco side-by-side diff review before any file is written.
- `run_command` requests surface a pending command review before execution.
- Pending approvals survive browser refresh or reconnect and reappear in the review drawer.
- Repository context builds asynchronously in the background and never blocks the main workspace.
- Each agent node detail view can show which repository files it read or proposed to change during the run.

## Manual Validation Flow

1. Start Relay with `--project-root` pointing at a valid local Git repository and confirm the workspace shows the repository as connected.
2. Restart Relay without `--project-root`, use the in-product folder picker or project-root control, and confirm the same repository can be connected from the UI.
3. Confirm the workspace remains responsive while Relay prepares repository context in the background, even for large repositories or repositories with limited import metadata.
4. Ask an agent to inspect code and confirm file reads, listings, searches, Git log access, and Git diff access stay within the connected repository.
5. Ask an agent to modify a file and confirm the request appears as a pending write approval with a side-by-side diff preview.
6. Reject the write request and confirm nothing is written to disk.
7. Create another write request, refresh the browser before acting, reconnect, and confirm the pending approval is restored.
8. Approve the restored write request and confirm the file is written only after approval and revalidation.
9. Ask an agent to run a command, confirm the exact command and arguments are shown for review, and verify the command runs from the repository root only after approval.
10. Change the connected repository while an approval is pending and confirm the stale approval becomes blocked or expired rather than executing against the new repository.
11. Inspect an agent node on the canvas and confirm the node detail view shows the files it read and the files it proposed to change.

## Focused Test Commands

Backend repo-aware tools and approvals:

```bash
go test ./internal/tools ./internal/orchestrator/workspace -run 'Test.*(Approval|ProjectRoot|Git|Search|Read|Write|RunCommand)'
```

Protocol and reconnect coverage:

```bash
go test ./internal/handlers/ws ./tests/integration -run 'Test.*(Approval|Workspace|Reconnect|Codebase)'
```

Frontend repository and approval UI:

```bash
npm --prefix web test -- src/features/preferences/PreferencesPanel.test.tsx src/features/approvals/ApprovalReviewPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx src/shared/lib/workspace-store.test.ts
```

Type safety validation:

```bash
npm --prefix web run typecheck
```

## Failure Recovery Expectations

- If a connected path is not a valid Git repository, Relay keeps repo-aware tools disabled and explains the issue in plain language.
- If the browser reconnects while approvals are still pending, the UI rehydrates from persisted approval records instead of losing the review queue.
- If a file changed after a diff was proposed, the approval must not apply stale content.
- If repository-context construction becomes slow or fails, the workspace remains usable and Relay records the degraded state without blocking the shell or drawer.
- If `run_command` detects the repository root is no longer valid at execution time, it blocks the command and records a plain-language failure state.