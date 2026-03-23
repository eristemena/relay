# relay Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-23

## Active Technologies
- SQLite database at `~/.relay/relay.db` for session records and minimal workspace state; TOML config at `~/.relay/config.toml` for preferences and stored credentials (003-local-relay-workspace)

- Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend + Go standard library (`net/http`, `net/http/httputil`, `embed`, `context`, `os`, `os/exec`, `database/sql`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, SQLite via `modernc.org/sqlite` or equivalent driver plus sqlc query generation and goose migrations, `github.com/pelletier/go-toml/v2` for `~/.relay/config.toml`, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion (003-local-relay-workspace)

## Project Structure

```text
backend/
frontend/
tests/
```

## Commands

npm test && npm run lint

## Code Style

Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend: Follow standard conventions

## Recent Changes
- 003-local-relay-workspace: Added Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend + Go standard library (`net/http`, `net/http/httputil`, `embed`, `context`, `os`, `os/exec`, `database/sql`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, SQLite via `modernc.org/sqlite` or equivalent driver plus sqlc query generation and goose migrations, `github.com/pelletier/go-toml/v2` for `~/.relay/config.toml`, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion

- 003-local-relay-workspace: Added Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend + Go standard library (`net/http`, `net/http/httputil`, `embed`, `context`, `os`, `os/exec`, `database/sql`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, SQLite via `modernc.org/sqlite` or equivalent driver plus sqlc query generation and goose migrations, `github.com/pelletier/go-toml/v2` for `~/.relay/config.toml`, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion

<!-- MANUAL ADDITIONS START -->

## Design Rules (always apply these)

**Colors — use CSS tokens, never raw hex:**
```
--color-base: #09090F       /* page background */
--color-surface: #0D0D18    /* cards, panels */
--color-raised: #111120     /* elevated elements */
--color-border: #1A1A2E     /* borders */
--color-brand: #A78BFA      /* primary accent */
--color-brand-mid: #7C3AED  /* interactive */
--color-brand-dim: #5B5BD6  /* trailing/inactive */
--color-text: #FAFAFA
--color-text-muted: #52527A
--color-success: #34D399
--color-error: #F87171
```

**Typography:**
- Display/headings: `Urbanist` (Google Fonts)
- UI labels/body: `DM Sans`
- Code/streams/monospace: `JetBrains Mono`
- Never use Inter, Roboto, or system-ui for branded text

**Animation easing:** `cubic-bezier(0.16, 1, 0.3, 1)` — never linear or ease-in-out for UI transitions

**State = glow:** agent node states are communicated via box-shadow glow, not color alone:
- Thinking: `0 0 0 1px #7C3AED, 0 0 20px rgba(124,58,237,0.35)`
- Complete: `0 0 0 1px #34D399, 0 0 12px rgba(52,211,153,0.2)`
- Error: `0 0 0 1px #F87171`
- Idle: `0 0 0 1px #1A1A2E` (no glow)

---

## Architecture Rules

- Go layered: handlers → orchestrator → agents → tools → storage. No cross-layer imports.
- `Agent` interface is the contract — orchestrator never imports concrete agent types.
- WebSocket events are the only Go↔React communication channel.
- SQLite only — no Postgres, Redis, or external stores.
- Frontend: feature-based folders (`/features/canvas`, `/features/history`) not type-based.
- No file writes or shell commands execute without explicit developer approval — enforced server-side.

---

## What NOT to Do

- No light mode, no light backgrounds
- No Inter, Roboto, Space Grotesk fonts
- No raw hex values in components — always use CSS tokens
- No card drop shadows — use border + background layering
- No bouncing/spring animations on functional UI
- No toast stacks — errors inline, approvals in context
- No `any` in TypeScript
- No `fmt.Println` in Go (use structured logger)
- No goroutines without a `context.Context` cancellation path

<!-- MANUAL ADDITIONS END -->
