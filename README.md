# Relay

Relay is a local developer workspace that launches a browser-based AI coding shell from a single command.

The current workspace includes a live multi-agent orchestration surface where Relay coordinates the Planner, Coder, Tester, Reviewer, and Explainer, streams visible output from OpenRouter over WebSocket, and saves completed, halted, or errored runs for later replay.

The browser never receives the saved OpenRouter API key. Relay persists configuration and run history locally, gates mutating tools behind explicit approval, and reattaches active runs after reconnects.

## Stack

- Go 1.26
- SQLite for local session persistence at `~/.relay/relay.db`
- TOML config at `~/.relay/config.toml`
- Next.js App Router frontend in `web/`
- Tailwind CSS for styling
- `go-git` for local repository inspection without shelling out to Git
- `monaco-editor` for side-by-side approval diff review surfaces
- WebSocket-only runtime state delivery between Go and the browser

## Commands

- `make dev`: run the Next.js dev server and Relay together
- `make build`: export the frontend, copy the assets into the embed directory, and build the binary
- `make test`: run Go unit and integration tests
- `make test-web`: run frontend component tests
- `npm --prefix web run typecheck`: run the frontend TypeScript checker

## Local Data

- Config directory: `~/.relay`
- Config file: `~/.relay/config.toml`
- Database: `~/.relay/relay.db`

## Live Agent Setup

Add the OpenRouter API key and a manual project root to `~/.relay/config.toml` before using repository-aware runs:

```toml
project_root = "/absolute/path/to/your/repository"

[openrouter]
api_key = "or-your-key-here"

[agents]
planner = "anthropic/claude-opus-4"
coder = "anthropic/claude-sonnet-4-5"
reviewer = "anthropic/claude-sonnet-4-5"
tester = "deepseek/deepseek-chat"
explainer = "google/gemini-2.0-flash-001"
```

Relay keeps the key server-side, exposes only configuration status to the browser, and stores run history in SQLite for replay.

## Built-In Roles

- `planner`: planning and sequencing work, read-only tools
- `coder`: implementation tasks, can request mutating tools with approval
- `reviewer`: review and regression analysis, read-only tools
- `tester`: test-oriented work, can request mutating tools with approval
- `explainer`: read-only explanation and walkthrough tasks

Relay keeps the prompts and tool allowlists fixed in code per role. The config file only controls model assignment.

## Repository-Aware Setup

- Install backend and frontend dependencies before working on codebase awareness: `go mod download` and `npm --prefix web install`
- Relay only enables repository-aware tools when `project_root` points at a readable local Git repository root.
- Relay uses `monaco-editor` for side-by-side approval diff review and keeps background repository-context state available for backend-driven activity tracking.
- Repository inspection stays inside the Go process through `go-git`, so Relay does not shell out to the system `git` binary for repository validation, history, or working-tree diff reads.

## Live Agent Orchestration

Relay now includes a live orchestration canvas inside the workspace shell.

- Submit one goal and Relay starts a prompt-only run with Planner first, then Coder and Tester in parallel, then Reviewer, then Explainer.
- Nodes are appended only when an agent is spawned, then patched in place as state, transcript, handoff, and failure events arrive.
- The live canvas adds presentation-only motion: spawned nodes fade and scale in, active handoffs pulse on the connecting edge, and streaming borders clear within 300ms after token silence.
- Terminal node updates now include authoritative token usage when OpenRouter returns it, plus the known model context window when Relay can resolve it.
- Each node shows a context-usage bar with explicit unavailable, raw-count-only, neutral, warning, and critical states, and the same data replays for runs recorded after the feature shipped.
- Selecting a node opens that agent's current or preserved output in the side panel without interrupting the run.
- The selected-node panel keeps explicit empty, loading, and plain-language error states while selection changes animate in place.
- Opening a saved run replays the stored orchestration timeline into the same canvas surface.
- The run history panel now supports transcript-aware keyword search, touched-file and date filters, cursor-aware diff review, replay speed control, and markdown export to `~/.relay/exports/`.

This orchestration mode is intentionally prompt-only. It does not read the repository, write files, or run shell commands as part of the orchestration DAG.

## Frontend Validation

- `npm --prefix web run typecheck`: run the frontend TypeScript checker
- `npm --prefix web test`: run the frontend component and store tests, including the live canvas suite
- `npm --prefix web test -- src/features/preferences/PreferencesPanel.test.tsx src/features/approvals/ApprovalReviewPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx src/shared/lib/workspace-store.test.ts`: run the focused repository-awareness UI and store validation suite
- `npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/workspace-shell/WorkspaceShell.test.tsx`: run the focused orchestration canvas coverage
- `npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/canvas/AnimatedHandoffEdge.test.tsx src/features/canvas/AgentNodeDetailPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx src/features/canvas/canvasModel.test.ts src/features/canvas/layoutGraph.test.ts src/shared/lib/workspace-store.test.ts`: run the focused canvas animation and motion-regression suite
- If `make dev` opens Relay with a "frontend dev server is unavailable" page, start the Relay Next.js app on any free port from `3000` to `3010` and restart the backend so the dev proxy can rediscover it.

## Approval Flow

- `read_file` and `search_codebase` run without approval when `project_root` is valid.
- `write_file` and `run_command` always emit an approval request before execution.
- Pending approval requests are persisted in SQLite while they remain actionable, then restored through workspace bootstrap after reconnect or refresh.
- Approval outcomes are also reflected back into the ordered run-event timeline so replay preserves the same visible decision path.
- If approval is rejected, Relay records the rejected tool result and ends the run with a terminal error while preserving the earlier timeline.

## Run Review And Reconnect

- Relay allows only one active run at a time.
- Completed, halted, and errored runs are stored in `~/.relay/relay.db` as ordered events.
- Opening a saved run replays its stored timeline without contacting OpenRouter.
- Historical diff review is reconstructed from persisted approval diffs, not current repository contents.
- Export is accepted only from a direct user action in the workspace UI and writes a markdown report under `~/.relay/exports/`.
- If the browser reconnects during an active run, Relay restores any still-pending approvals, replays stored events, then reattaches the live stream.

## Orchestration Validation

- `go test -cover ./internal/agents ./internal/orchestrator/workspace ./internal/handlers/ws ./internal/storage/sqlite`: verify the current orchestration coverage threshold across the touched core packages
- `go test ./tests/integration -run 'TestAgentStreaming_|TestWorkspaceWebSocket_' -count=1`: run the broader orchestration ordering, replay, and reconnect regressions
- `go test ./tests/integration/run_history_replay_test.go`: run the replay-focused integration file directly
- `go test ./internal/orchestrator/workspace ./internal/handlers/ws -run 'Test.*(Replay|Seek|Export|HistoryQuery)'`: run focused replay-control, export, and history-query coverage
- `go test ./internal/storage/sqlite -run 'Test.*(RunHistory|RunChange|FTS|Export)'`: run focused persistence, search, and export coverage
- `npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/workspace-shell/WorkspaceShell.test.tsx`: run the current canvas and workspace-shell regression suite
- `npm --prefix web test -- src/features/history/replay/RunChangeReviewPanel.test.tsx src/features/canvas/AgentCanvas.test.tsx src/features/history/RunHistoryPanel.test.tsx src/features/history/replay/ReplayControls.test.tsx src/features/history/replay/ReplayTimeline.test.tsx src/shared/lib/workspace-store.test.ts`: run the focused run-history replay UI, diff-review, and store regression suite
- `npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/canvas/AnimatedHandoffEdge.test.tsx src/features/canvas/AgentNodeDetailPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx src/features/canvas/canvasModel.test.ts src/features/canvas/layoutGraph.test.ts src/shared/lib/workspace-store.test.ts`: run the canvas animation layer regressions and store derivation checks
- `go test ./internal/agents/openrouter ./internal/orchestrator/workspace -run 'Test.*(Usage|ContextLimit|Complete)'`: run focused provider-usage and context-limit resolution coverage
- `go test ./internal/storage/sqlite ./internal/handlers/ws ./tests/integration -run 'Test.*(RunEvent|Replay|TokenUsage)'`: run focused persistence, replay, and token-usage regressions
- `npm --prefix web test -- src/features/canvas/layoutGraph.test.ts src/features/canvas/AgentCanvas.test.tsx src/features/workspace-shell/WorkspaceShell.test.tsx`: run the canvas model, layout, and shell regression suite together
- `make build && ./bin/relay serve --help`: verify the packaged frontend assets and built `relay serve` entrypoint
- `env -u RELAY_DEV ./bin/relay serve --no-browser --port 4851`: smoke-test the packaged server path against the embedded frontend; if `RELAY_DEV=true` is still set, Relay will intentionally switch to the dev-proxy frontend mode instead

## Notes

- Relay prefers port `4747` and falls back to a free local port for the current run if needed.
- In development, Relay keeps `/ws` and `/api/healthz` in Go and proxies other browser routes to the discovered Next.js dev server.
- Production builds embed the exported frontend assets into the Go binary.
- The live orchestration surface supports one active run at a time and replays saved runs from the local SQLite database.
- Active-run hydration reuses the existing bootstrap and open-run flow: when the browser reconnects and Relay reports an `active_run_id`, the frontend reopens that run once to replay stored events before live delivery resumes.
- Planner failures emit `agent_error` for the planner node and a terminal `run_error` for the run, while later agent-scoped failures can remain inspectable without forcing an immediate run halt.
- Repository-aware tools stay disabled until `project_root` is configured as a valid absolute directory.
