# Quickstart: Live Agent Panel

## Historical Note

This phase introduced the original live execution drawer panel. In the current product, that dedicated panel has been removed; task submission remains in the command bar, while live and replayed inspection happens through canvas node detail, saved runs, and approval review surfaces.

## Prerequisites

- Go 1.26
- Node.js LTS version compatible with Next.js 16.2
- npm
- A writable home directory for `~/.relay/config.toml` and `~/.relay/relay.db`
- A valid OpenRouter API key

## Config Setup

Add or update the Relay config so it contains the OpenRouter credential, manual project root, and per-role model assignments.

```toml
port = 4747
open_browser_on_start = true
appearance_variant = "midnight"
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

Expected behavior:

- The API key remains on the Go side only.
- `project_root` is configured manually in `config.toml` for this phase; there is no folder-picker UI yet.
- Invalid or missing model strings fall back to Relay defaults.
- The recommended default role-to-model assignments are Planner -> `anthropic/claude-opus-4`, Coder -> `anthropic/claude-sonnet-4-5`, Reviewer -> `anthropic/claude-sonnet-4-5`, Tester -> `deepseek/deepseek-chat`, and Explainer -> `google/gemini-2.0-flash-001`.
- The frontend sees only whether credentials are configured and which model was used for each run.
- Read-only repo tools require a valid absolute `project_root`.
- Mutating tools (`write_file`, `run_command`) stop and wait for approval before they execute.

## Role Summary

- `planner`: plan and sequence work, read-only repo access
- `coder`: implement code changes, may request file writes and commands with approval
- `reviewer`: analyze code and regressions, read-only repo access
- `tester`: inspect and run test-oriented workflows, may request file writes and commands with approval
- `explainer`: explain code or runtime behavior, read-only repo access

## First-Time Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
npm --prefix web install
npm --prefix web run typecheck
go mod tidy
```

## Development Workflow

```bash
make dev
```

Expected behavior:

- The existing local workspace still boots through Relay.
- Live and replayed execution details are inspected through the canvas, saved runs, and approval review surfaces rather than a dedicated drawer panel.
- The command bar is available once the WebSocket bootstrap completes.

## Running Relay Manually

Development mode with the frontend dev server:

```bash
RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747
```

Production-style local run from a built binary:

```bash
./bin/relay serve --port 4747
```

## Test Commands

Backend and integration tests:

```bash
go test ./...
```

Frontend component tests:

```bash
npm --prefix web test
npm --prefix web run typecheck
```

## Manual Validation Flow

1. Start Relay and confirm the workspace bootstrap succeeds.
2. Before saving credentials, submit a task and confirm the command bar stays available while Relay returns a clear OpenRouter configuration error.
3. Open settings and save a valid OpenRouter API key.
4. Submit a planning-style task such as `Plan the steps to add a JWT parser to this Go service` and confirm the panel shows a live stream, a model badge, and state changes.
5. Submit a task that triggers at least one tool call after configuring repository access for the backend and confirm `tool_call` and `tool_result` entries appear inline in order.
6. Reject a mutating tool approval request and confirm the run records a rejected tool result plus a final error while preserving earlier timeline entries.
7. Reopen the saved errored or completed run from history and confirm the timeline replays without contacting OpenRouter.
8. Reload the browser during an active run and confirm Relay restores the workspace bootstrap and can continue the active stream after replay.
9. Confirm the active stream shows a live cursor while output is arriving.
10. Validate that each default model behaves acceptably with its built-in role prompt and that unsupported tool-calling behavior fails with a clear inline error rather than silent corruption.
11. Restart Relay and verify the saved run remains reviewable from run history without re-running the model.
12. Replace the API key with an invalid value and confirm the next run fails gracefully with a plain-language configuration or provider error.

## Failure Recovery Expectations

- If the OpenRouter API key is missing, the command bar remains available but run submission returns a clear configuration error instead of silently failing.
- If `project_root` is missing or invalid, Relay blocks repo-scoped tool activity and explains that the path must be corrected in `config.toml`.
- If a mutating tool requires approval and the developer rejects it, Relay records the rejection inline and terminates the run without hiding previous events.
- If the chosen model does not support the requested tool behavior, Relay records the partial run, emits an error event, and preserves the visible history.
- If OpenRouter returns a mid-stream error, Relay must preserve all prior ordered events and terminate the run with a final error event.
- If the browser reconnects during or after a run, the server must deliver a fresh bootstrap snapshot and replay the selected run in stored order when needed, then resume live delivery for active runs.
