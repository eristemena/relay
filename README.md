# Relay

Relay is a local developer workspace that launches a browser-based AI coding shell from a single command.

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

## Local Data

- Config directory: `~/.relay`
- Config file: `~/.relay/config.toml`
- Database: `~/.relay/relay.db`

## Notes

- Relay prefers port `4747` and falls back to a free local port for the current run if needed.
- In development, Relay keeps `/ws` and `/api/healthz` in Go and proxies other browser routes to the discovered Next.js dev server.
- Production builds embed the exported frontend assets into the Go binary.
