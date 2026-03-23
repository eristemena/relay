# Quickstart: Local Relay Workspace

## Prerequisites

- Go 1.26
- Node.js LTS version compatible with Next.js 16.2
- npm
- A writable home directory for `~/.relay/config.toml` and `~/.relay/relay.db`

## Repository Layout Assumption

- Go application code lives under `cmd/` and `internal/`.
- Next.js source lives under `web/`.
- Production frontend assets are copied into `internal/frontend/embed/` before compiling the Go binary.

## First-Time Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
npm --prefix web install
go mod tidy
```

## Development Workflow

Run the frontend and backend together so frontend changes do not require recompiling the binary.

```bash
make dev
```

Expected behavior:

- `next dev` prefers port `3000` and falls back to a free port when `3000` is unavailable
- `RELAY_DEV=true air` prefers port `4747` and falls back to a free port when `4747` is unavailable
- Go handles `/ws` and `/api/healthz` directly
- All other browser routes are reverse-proxied to the discovered frontend dev-server port

If a Procfile and overmind are preferred, the equivalent process model is:

```bash
overmind start
```

## Running Relay Manually

Development mode with proxy enabled:

```bash
RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747
```

The `--port 4747` value is the preferred backend port. If it is already in use, Relay should choose a free port automatically for that run and print the assigned address.

Production-style local run from a built binary:

```bash
./bin/relay serve --port 4747
```

The built binary follows the same preferred-port behavior: it should bind to `4747` when available and otherwise continue on a free port.

If automatic browser launch is not desired for a given run, use:

```bash
./bin/relay serve --port 4747 --no-browser
```

## Production Build

The build must export the Next.js frontend first, copy it into the embed directory, then compile the Go binary with build metadata.

```bash
make build
```

Expected build sequence:

1. `next build` runs in `web/` with `output: 'export'`
2. `web/out/` is copied into `internal/frontend/embed/`
3. `go build` compiles the final binary with `-ldflags` for version, commit, and date
4. The final executable is written to `bin/relay`

## Test Commands

Backend and integration coverage:

```bash
go test ./...
```

Frontend component coverage:

```bash
npm --prefix web test
```

## Manual Verification Flow

1. Run `relay serve` or the development command.
2. Confirm the browser opens automatically to the actual Relay address printed at startup, using `4747` when available or a discovered free port when it is not.
3. Confirm the workspace shows the top navigation, session sidebar, and central canvas shell.
4. Create a new session and verify it becomes active immediately.
5. Restart Relay and confirm the previous session still appears in the sidebar.
6. Change a supported preference, restart Relay, and confirm the preference remains applied.
7. Stop the frontend dev server while Go is still running in dev mode and confirm Relay shows a recoverable error instead of proxying a broken blank page silently.

## Failure Recovery Expectations

- If the preferred Relay port is unavailable, Relay must choose a free port, print the actual URL, and avoid opening a broken browser tab. It should fail only if no usable local port can be allocated.
- If the preferred frontend dev port `3000` is unavailable, the development workflow must still start and the Go proxy must target the discovered frontend port automatically.
- If browser launch is blocked, Relay must keep running and print the local URL.
- If config parsing fails partially, Relay must keep valid settings and explain which values were ignored.