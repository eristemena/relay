# Implementation Plan: Agent Token Usage

**Branch**: `009-agent-token-usage` | **Date**: 2026-03-27 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/009-agent-token-usage/spec.md`
**Input**: Feature specification from `/specs/009-agent-token-usage/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extend Relay's completion telemetry so each agent stage can report provider token usage and known context-window limits through the existing event pipeline, persist those values in SQLite, and render a live and replayable token-usage bar on each canvas node. The backend will capture `usage.total_tokens` from the final OpenRouter streaming chunk, resolve `context_limit` through a startup-loaded and TTL-cached model metadata registry with a local fallback path for non-OpenRouter models, and store both values in new nullable columns on `agent_run_events` while preserving JSON payload replay. The frontend will consume the additional payload fields through the existing workspace store and canvas patch flow, deriving a neutral, warning, or critical fill state without introducing aggregate usage dashboards or cost reporting.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library first, including `net/http`, `context`, `sync`, and `time`; existing `github.com/sashabaranov/go-openai` OpenRouter client remains in use for streaming responses; existing `modernc.org/sqlite`, React Flow, Framer Motion, Tailwind, and workspace protocol/store layers remain in use; no new third-party dependency is required if model metadata fetching uses the standard library  
**Storage**: SQLite only, using the existing `agent_run_events` table plus two new nullable integer columns for `tokens_used` and `context_limit`; payload JSON remains stored for replay compatibility  
**Testing**: `go test` for `internal/agents/openrouter`, `internal/orchestrator/workspace`, `internal/storage/sqlite`, and `internal/handlers/ws`; table-driven Go tests for final-chunk usage capture, model-limit fallback, migration-backed event reads, and replay hydration; Vitest plus React Testing Library for workspace-store protocol handling and canvas node rendering; integration coverage under `tests/integration` for live stream plus replay behavior  
**Target Platform**: Local Relay development on macOS-first workstations with browser UI on localhost; runtime remains browser-based with WebSocket transport only  
**Project Type**: Full-stack Relay backend/frontend protocol and visualization enhancement  
**Performance Goals**: WebSocket dispatch remains under 100ms per event; startup model metadata refresh does not block workspace bootstrap beyond a bounded initial fetch path; canvas token-bar updates remain non-blocking during active streaming and replay  
**Constraints**: Dark mode only; WebSocket remains the only backend/frontend runtime channel; SQLite remains the only data store; current provider runtime is OpenRouter-first, but context-limit resolution must degrade safely for local or non-OpenRouter model names; OpenRouter usage is only reliable in the final streaming chunk, so missing or zero values must not be treated as authoritative; Relay must keep older runs replayable without backfill  
**Scale/Scope**: Single-user local workstation; one run at a time; up to five orchestration agent nodes per run plus single-agent runs; per-node token telemetry only, with no multi-run aggregates or cost accounting in this phase

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: Go changes stay inside the existing agents -> orchestrator -> storage flow, standard library networking can cover the model metadata cache, exported APIs added for usage or model-limit services will require godoc comments, and frontend changes remain strict TypeScript with no banned debug logging.
- [x] Test impact is defined: table-driven Go tests cover final-chunk usage extraction, model-limit cache refresh and fallback behavior, store persistence and replay hydration, and handler payload serialization; WebSocket integration tests cover extended completion payloads; React Flow node and workspace-store tests cover token-bar state derivation and replay behavior.
- [x] Architecture remains compliant: handlers continue to own protocol types, orchestrator/workspace owns event emission and replay hydration, agents/openrouter owns provider stream parsing, storage remains SQLite-only, and frontend changes stay within `features/canvas` plus shared protocol/store files.
- [x] UX and governance impact is defined: live token bars update visibly, missing usage or limit data surfaces a plain fallback state, replay preserves historical values when available, and no approval or file-system behavior changes are introduced.
- [x] Security and performance constraints are covered: no prompt or response logging changes are introduced, metadata cache refresh runs with context cancellation and TTL bounds, SQLite access continues to avoid extra tables or N+1 fetches, and canvas interaction remains responsive while token usage updates stream.

## Project Structure

### Documentation (this feature)

```text
specs/009-agent-token-usage/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── workspace-token-usage.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── agents/
│   ├── agent.go
│   ├── registry.go
│   └── openrouter/
│       ├── client.go
│       └── client_test.go
├── config/
│   └── config.go
├── handlers/
│   └── ws/
│       └── protocol.go
├── orchestrator/
│   └── workspace/
│       ├── history.go
│       ├── orchestration.go
│       ├── runs.go
│       ├── service.go
│       └── service_test.go
└── storage/
    └── sqlite/
        ├── migrations/
        │   └── 0005_agent_run_event_token_usage.sql
        ├── queries/
        │   └── agent_run_events.sql
        ├── models.go
        ├── store.go
        └── store_test.go

tests/
└── integration/
    ├── agent_streaming_test.go
    ├── run_history_replay_test.go
    └── tool_call_ordering_test.go

web/
└── src/
    ├── features/
    │   └── canvas/
    │       ├── AgentCanvasNode.tsx
    │       ├── AgentCanvasNode.test.tsx
    │       ├── AgentNodeDetailPanel.tsx
    │       ├── AgentCanvas.test.tsx
    │       └── canvasModel.ts
    └── shared/
        └── lib/
            ├── workspace-protocol.ts
            ├── workspace-store.ts
            └── workspace-store.test.ts
```

**Structure Decision**: Keep provider-specific usage extraction inside `internal/agents/openrouter`, thread normalized usage data through existing `agents.StreamEventHandlers`, and let `internal/orchestrator/workspace` attach per-agent token usage to terminal events and replay payloads. Persist the new telemetry in the existing `agent_run_events` table rather than introducing a parallel usage table, and keep frontend state derivation in `workspace-store` plus `features/canvas` so the canvas still updates through the established patch model.

## Complexity Tracking

No constitution violations are required by this design.
