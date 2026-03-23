# Implementation Plan: Local Relay Workspace

**Branch**: `003-local-relay-workspace` | **Date**: 2026-03-23 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/003-local-relay-workspace/spec.md`
**Input**: Feature specification from `/specs/003-local-relay-workspace/spec.md`

## Summary

Deliver Relay's initial single-command local workspace as a self-contained Go binary that starts a local server on the preferred port or the next free port, opens the browser automatically, restores locally persisted sessions, and keeps developer preferences across restarts. The implementation uses a Go Cobra entrypoint, a SQLite-backed session store with sqlc and goose, a TOML config file for preferences and credentials, a WebSocket-first UI state contract, a Next.js 16.2 App Router frontend reverse-proxied in development against the discovered dev-server port, and statically exported frontend assets embedded into the production binary with `go:embed`.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library (`net/http`, `net/http/httputil`, `embed`, `context`, `os`, `os/exec`, `database/sql`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, SQLite via `modernc.org/sqlite` or equivalent driver plus sqlc query generation and goose migrations, `github.com/pelletier/go-toml/v2` for `~/.relay/config.toml`, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion  
**Storage**: SQLite database at `~/.relay/relay.db` for session records and minimal workspace state; TOML config at `~/.relay/config.toml` for preferences and stored credentials  
**Testing**: `go test ./...` with table-driven unit tests for config, storage, and orchestrator logic; Go integration tests for backend free-port fallback, route ordering, browser-open fallback, dev proxy target discovery, and WebSocket reconnect/bootstrap behavior; Vitest plus React Testing Library for workspace shell, session history, and empty/loading/error states  
**Target Platform**: Local developer machines with macOS as the first target, browser UI on localhost, and portable binary support for Linux/Windows once browser opening is abstracted behind OS-specific adapters  
**Project Type**: Single-binary local web workspace with embedded static frontend assets and a Go-owned WebSocket backend  
**Performance Goals**: Browser-ready startup within 2 seconds in at least 95% of runs, backend-to-frontend WebSocket event dispatch under 100ms, graceful reconnect after browser refresh without losing the active session, and non-blocking React Flow canvas interaction while shell state updates stream  
**Constraints**: Dark mode only; WebSocket is the only backend-to-frontend state channel; SQLite is the only data store; frontend port 3000 and backend port 4747 are preferred defaults rather than fixed requirements; `/ws` and `/api/healthz` must remain owned by Go and registered before any dev proxy catch-all; production assets must be available at a compile-time-stable `go:embed` path; offline core flows must work without internet; handler-level approval enforcement for future file writes and shell commands must remain unchanged; distribution is a single binary with no Docker requirement  
**Scale/Scope**: Single-user local workstation, one active Relay process, one or a few browser tabs, and hundreds to low thousands of stored session rows over time

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: the plan keeps Go code idiomatic, relies on the standard library for serving, proxying, embedding, and browser launch, preserves strict TypeScript, and introduces no banned debug logging requirement.
- [x] Test impact is defined: table-driven Go tests cover config and storage behavior, integration tests cover WebSocket bootstrap/reconnect and local startup behavior, and frontend component tests cover the shell, history, and visible state handling.
- [x] Architecture remains compliant: Go request flow stays in `handlers -> orchestrator -> storage` for this slice, reserves `agents` and `tools` boundaries for later capabilities, keeps SQLite and WebSocket as the only persistence/runtime transport, and uses feature-based frontend folders.
- [x] UX and governance impact is defined: the design includes visible loading, empty, saving, and recoverable error states plus browser-open fallback messaging, with no change to handler-level approval enforcement.
- [x] Security and performance constraints are covered: secrets remain in local config only, credentials are never sent to the frontend, goroutines require cancellation paths, SQLite access is batched through generated queries, and route ordering protects Relay-owned endpoints from proxy leakage.

## Project Structure

### Documentation (this feature)

```text
specs/003-local-relay-workspace/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── http-api.yaml
│   └── websocket-events.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── relay/
    └── main.go

internal/
├── app/
│   └── server.go
├── browser/
│   └── open.go
├── config/
│   └── config.go
├── frontend/
│   ├── embed/
│   │   └── ... compiled Next.js export copied here before `go build`
│   ├── proxy.go
│   └── static.go
├── handlers/
│   ├── http/
│   │   └── health.go
│   └── ws/
│       └── workspace.go
├── orchestrator/
│   └── workspace/
│       └── service.go
└── storage/
    └── sqlite/
        ├── migrations/
        ├── queries/
        ├── sessions.sql.go
        └── store.go

tests/
└── integration/
    ├── serve_startup_test.go
    └── websocket_reconnect_test.go

web/
├── next.config.ts
├── package.json
├── public/
└── src/
    ├── app/
    │   ├── layout.tsx
    │   └── page.tsx
    ├── features/
    │   ├── canvas/
    │   ├── history/
    │   ├── preferences/
    │   └── workspace-shell/
    └── shared/
        ├── lib/
        └── ui/
```

**Structure Decision**: Use a top-level Go application layout with `cmd/` and `internal/` for the local service, keep compiled frontend artifacts under `internal/frontend/embed` so the `go:embed` directive points at a stable compile-time directory, and keep source frontend code under `web/src/features/*` to preserve feature-based organization. The frontend should not call Go HTTP APIs for workspace state; instead, it connects to `/ws` and receives a full bootstrap snapshot plus subsequent state events. A minimal `/api/healthz` endpoint is retained for startup diagnostics and automated checks only. In development, Go must proxy to the actual frontend dev-server port discovered at startup rather than assuming `3000`, and in all modes the browser must open the actual assigned Relay address.

## Complexity Tracking

No constitution violations are required by this design.
