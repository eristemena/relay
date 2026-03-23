# Relay

Relay is a local developer workspace that launches a browser-based AI coding shell from a single command.

The current workspace includes a live agent panel where Relay selects one built-in role, streams visible output from OpenRouter over WebSocket, and saves completed or errored runs for later replay.

The browser never receives the saved OpenRouter API key. Relay persists configuration and run history locally, gates mutating tools behind explicit approval, and reattaches active runs after reconnects.

## Stack

- Go 1.26
- SQLite for local session persistence at `~/.relay/relay.db`
- TOML config at `~/.relay/config.toml`
- Next.js App Router frontend in `web/`
- Tailwind CSS for styling
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
explainer = "google/gemini-flash-1.5"
```

Relay keeps the key server-side, exposes only configuration status to the browser, and stores run history in SQLite for replay.

## Built-In Roles

- `planner`: planning and sequencing work, read-only tools
- `coder`: implementation tasks, can request mutating tools with approval
- `reviewer`: review and regression analysis, read-only tools
- `tester`: test-oriented work, can request mutating tools with approval
- `explainer`: read-only explanation and walkthrough tasks

Relay keeps the prompts and tool allowlists fixed in code per role. The config file only controls model assignment.

## Isolated Agent Canvas

Relay now includes a frontend-only agent canvas inside the workspace shell.

- Add Planner, Coder, Reviewer, Tester, and Explainer nodes from the local toolbar.
- Inspect a selected node in a side panel without leaving the canvas surface.
- Change the selected node's local state to `idle`, `thinking`, `executing`, `complete`, or `error` without triggering a relayout.
- Pan and zoom the graph while the canvas recalculates layout after structural changes.

The canvas is intentionally isolated from backend runs, WebSocket events, and SQLite persistence. It exists to validate the workflow design surface, not to mirror live execution.

## Frontend Validation

- `npm --prefix web run typecheck`: run the frontend TypeScript checker
- `npm --prefix web test`: run the frontend component and store tests, including the isolated canvas suite
- If `make dev` opens Relay with a "frontend dev server is unavailable" page, start the Relay Next.js app on any free port from `3000` to `3010` and restart the backend so the dev proxy can rediscover it.

## Approval Flow

- `read_file` and `search_codebase` run without approval when `project_root` is valid.
- `write_file` and `run_command` always emit an approval request before execution.
- Approval requests are transient live events. They are not stored in SQLite history after the run finishes.
- If approval is rejected, Relay records the rejected tool result and ends the run with a terminal error while preserving the earlier timeline.

## Run Review And Reconnect

- Relay allows only one active run at a time.
- Completed and errored runs are stored in `~/.relay/relay.db` as ordered events.
- Opening a saved run replays its stored timeline without contacting OpenRouter.
- If the browser reconnects during an active run, Relay can reattach the live stream and continue delivery after replaying stored events.

## Notes

- Relay prefers port `4747` and falls back to a free local port for the current run if needed.
- In development, Relay keeps `/ws` and `/api/healthz` in Go and proxies other browser routes to the discovered Next.js dev server.
- Production builds embed the exported frontend assets into the Go binary.
- The live agent panel supports one active run at a time and replays saved runs from the local SQLite database.
- Repository-aware tools stay disabled until `project_root` is configured as a valid absolute directory.
