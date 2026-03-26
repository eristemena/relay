# relay Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-25

## Active Technologies
- SQLite database at `~/.relay/relay.db` for session records and minimal workspace state; TOML config at `~/.relay/config.toml` for preferences and stored credentials (003-local-relay-workspace)
- Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend + Go standard library (`context`, `net/http`, `encoding/json`, `sync`, `time`, `os/exec`, `database/sql`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, `github.com/pelletier/go-toml/v2` for local config, SQLite with sqlc-generated queries and goose migrations for persistence, `github.com/sashabaranov/go-openai` for OpenAI-compatible chat completion streaming against OpenRouter, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion (004-live-agent-panel)
- SQLite for sessions, agent runs, and append-only run events; TOML config at `~/.relay/config.toml` for `[openrouter]` credentials and `[agents]` role-to-model assignments (004-live-agent-panel)
- Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend + Go standard library (`context`, `net/http`, `encoding/json`, `sync`, `time`, `os/exec`, `database/sql`, `errors`, `path/filepath`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, `github.com/pelletier/go-toml/v2` for local config, SQLite with sqlc-generated queries and repository migrations, `github.com/sashabaranov/go-openai` for OpenAI-compatible chat completion streaming against OpenRouter, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion (004-live-agent-panel)
- SQLite for sessions, agent runs, and append-only run events; TOML config at `~/.relay/config.toml` for `project_root`, `[openrouter]` credentials, and `[agents]` role-to-model assignments (004-live-agent-panel)
- TypeScript 5.8.x in strict mode, React 19.1, Next.js 16.2 App Router frontend + Existing frontend stack (`next`, `react`, `react-dom`, `framer-motion`, `tailwindcss`, `vitest`, `@testing-library/react`) plus planned additions `@xyflow/react` for the controlled graph canvas and `@dagrejs/dagre` for directed graph layou (005-agent-canvas)
- No persistent storage; all node, edge, selection, and detail state remains in local client memory for this isolated experience (005-agent-canvas)
- Go 1.26 backend; TypeScript in strict mode with Next.js 16.2 App Router frontend + Go standard library (`context`, `sync`, `errors`, `time`, `encoding/json`), existing `nhooyr.io/websocket` transport, existing SQLite store and sqlc models, existing Relay agent and OpenRouter integration packages, existing React Flow and dagre frontend stack; no new third-party dependency is required for orchestration itself (006-live-agent-orchestration)
- SQLite only for orchestration runs, per-agent executions, and ordered event replay; existing `~/.relay/config.toml` remains the source for provider access and agent model settings (006-live-agent-orchestration)
- Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend + Go standard library and existing backend packages remain unchanged by default; existing frontend dependencies `@xyflow/react` for the canvas and existing `framer-motion` for node and panel animation; existing global CSS for state glow styling; no new third-party dependency is required (007-canvas-animation-layer)
- SQLite only for existing run persistence; no new persisted animation state (007-canvas-animation-layer)
- Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend + Go standard library first, including `os/exec`, `context`, `filepath`, and `sync`; new backend dependency `github.com/go-git/go-git/v5` for repository validation, tree traversal, commit log access, and working-tree diffs without a system Git dependency; new frontend dependency `monaco-editor` for side-by-side diff review; existing `@xyflow/react`, `framer-motion`, and workspace protocol/store layers remain in use (008-codebase-awareness)
- SQLite only, with new persisted approval-request state and possible repository-analysis metadata; existing config file remains the source of truth for `project_root` (008-codebase-awareness)

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
- 008-codebase-awareness: Added Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend + Go standard library first, including `os/exec`, `context`, `filepath`, and `sync`; new backend dependency `github.com/go-git/go-git/v5` for repository validation, tree traversal, commit log access, and working-tree diffs without a system Git dependency; new frontend dependency `monaco-editor` for side-by-side diff review; existing `@xyflow/react`, `framer-motion`, and workspace protocol/store layers remain in use
- 007-canvas-animation-layer: Added Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend + Go standard library and existing backend packages remain unchanged by default; existing frontend dependencies `@xyflow/react` for the canvas and existing `framer-motion` for node and panel animation; existing global CSS for state glow styling; no new third-party dependency is required
- 006-live-agent-orchestration: Added Go 1.26 backend; TypeScript in strict mode with Next.js 16.2 App Router frontend + Go standard library (`context`, `sync`, `errors`, `time`, `encoding/json`), existing `nhooyr.io/websocket` transport, existing SQLite store and sqlc models, existing Relay agent and OpenRouter integration packages, existing React Flow and dagre frontend stack; no new third-party dependency is required for orchestration itself


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
