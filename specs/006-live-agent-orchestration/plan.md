# Implementation Plan: Live Agent Orchestration

**Branch**: `006-live-agent-orchestration` | **Date**: 2026-03-24 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/006-live-agent-orchestration/spec.md`
**Input**: Feature specification from `/specs/006-live-agent-orchestration/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extend Relay from a single-run live panel plus isolated canvas prototype into a real multi-agent orchestration mode: one submitted goal creates an orchestration run owned by the workspace service, a coordinator goroutine enforces the Planner -> Coder and Tester in parallel -> Reviewer -> Explainer DAG, and each spawned agent executes through a direct `agent.Run(ctx, task)` call inside its own cancellable goroutine with a dedicated output channel. The backend emits per-agent orchestration plus transcript events over the existing WebSocket path for live canvas updates and replay, persists orchestration runs plus per-agent event streams in SQLite, and keeps the feature prompt-only with no repo reads, file writes, or shell execution. All five built-in roles are concrete structs behind the `Agent` interface; there is no separate `Runner` or `Worker` runtime type. On the frontend, the canvas moves from a local reducer-only simulation to live patch-driven updates keyed by `agent_id`, preserving the prior phase's anti-flicker rule: append nodes only on `agent_spawned`, patch existing nodes with `setNodes(prev => prev.map(...))`, and rerun dagre only when a new node appears.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript in strict mode with Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library (`context`, `sync`, `errors`, `time`, `encoding/json`), existing `nhooyr.io/websocket` transport, existing SQLite store and sqlc models, existing Relay agent and OpenRouter integration packages, existing React Flow and dagre frontend stack; no new third-party dependency is required for orchestration itself  
**Storage**: SQLite only for orchestration runs, per-agent executions, and ordered event replay; existing `~/.relay/config.toml` remains the source for provider access and agent model settings  
**Testing**: `go test` for orchestrator, handlers, agents, and storage with table-driven cases where applicable; integration tests for WebSocket orchestration ordering, replay, reconnect, and halt/error behavior; Vitest plus React Testing Library for live canvas patching, node selection, side-panel hydration, and regression coverage for the disappearing-node bug  
**Target Platform**: Local Relay desktop workflow on macOS-first developer machines with browser UI on localhost  
**Project Type**: Relay backend/frontend feature extending the existing live-run and canvas surfaces with real-time multi-agent orchestration  
**Performance Goals**: Planner node visible within 1 second of submission in at least 95% of accepted runs; backend-to-frontend dispatch under 100ms per event; concurrent agent streaming must not block pan, zoom, or node selection; reconnect must resume active-run visibility without duplicate node creation  
**Constraints**: Dark mode only; WebSocket remains the only backend/frontend runtime channel; SQLite remains the only data store; all goroutines require `context.Context` cancellation; prompt/response content must not appear in application logs; this orchestration mode is prompt-only and must not invoke repo reads, file writes, or shell commands; handler-level approval rules remain intact for other product areas but are not exercised by this feature; canvas updates must use patch semantics and run dagre only on node-spawn events  
**Scale/Scope**: Single-user local workstation; one active orchestration run at a time; exactly five built-in roles; one coordinator goroutine plus up to five agent goroutines per run; hundreds to low thousands of persisted runs over time

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: Go changes remain in the existing layered packages, standard library concurrency primitives are sufficient, exported API additions will require godoc comments, and frontend work stays in strict TypeScript with no debug logging.
- [x] Test impact is defined: orchestrator, handler, and storage tests cover concurrent orchestration order, reconnect/replay, and run-level halt semantics; custom canvas node and live patch behavior remain covered by component tests, including the disappearing-node regression.
- [x] Architecture remains compliant: handlers accept requests and broadcast events, orchestrator owns DAG execution and fan-out, concrete agents remain behind the `Agent` interface, no extra runner or worker layer is introduced, tools are not expanded for this feature, storage stays SQLite-only, and WebSocket remains the only browser runtime channel.
- [x] UX and governance impact is defined: the design includes explicit idle, blocked, active, agent-error, run-halted, and replay states; human-readable failure messages; explicit non-selected and empty panel states; and no relaxation of approval enforcement for file writes or shell commands elsewhere.
- [x] Security and performance constraints are covered: orchestration is prompt-only, no repo access is added, no prompt/response content is logged, every goroutine has a cancellation path, event fan-out is ordered per agent and per run, and the frontend patch pattern avoids relayout-driven flicker under concurrent updates.

## Project Structure

### Documentation (this feature)

```text
specs/006-live-agent-orchestration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── websocket-events.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── relay/
    └── main.go

internal/
├── agents/
│   ├── agent.go
│   ├── registry.go
│   ├── planner.go
│   ├── coder.go
│   ├── reviewer.go
│   ├── tester.go
│   ├── explainer.go
│   └── openrouter/
│       └── client.go
├── handlers/
│   └── ws/
│       ├── protocol.go
│       └── workspace.go
├── orchestrator/
│   └── workspace/
│       ├── service.go
│       ├── runs.go
│       ├── history.go
│       └── run_context.go
├── storage/
│   └── sqlite/
│       ├── migrations/
│       ├── queries/
│       ├── models.go
│       └── store.go
└── config/
    └── config.go

tests/
└── integration/
    ├── agent_streaming_test.go
    ├── run_history_replay_test.go
    └── websocket_reconnect_test.go

web/
├── src/
│   ├── features/
│   │   ├── canvas/
│   │   │   ├── WorkspaceCanvas.tsx
│   │   │   ├── AgentCanvas.tsx
│   │   │   ├── AgentCanvasNode.tsx
│   │   │   ├── layoutGraph.ts
│   │   │   └── canvasModel.ts
│   │   ├── agent-panel/
│   │   ├── history/
│   │   └── workspace-shell/
│   └── shared/
│       └── lib/
│           └── workspace-protocol.ts
└── package.json
```

**Structure Decision**: Reuse the existing workspace service, agent package, WebSocket handler, and canvas feature rather than creating a parallel orchestration subsystem. The backend adds orchestration-specific state and persistence under `internal/orchestrator/workspace` and `internal/storage/sqlite`, while `internal/agents` shifts from profile-only selection toward concrete role implementations behind the `Agent` interface. The orchestrator owns concurrency directly by calling `agent.Run(ctx, task)` inside per-agent goroutines rather than introducing a separate runner or worker type. The frontend keeps all live-canvas work under `features/canvas` and related existing feature folders; it does not introduce a second canvas implementation. The existing isolated graph components become the rendering foundation, but live state is driven by WebSocket events and explicit React Flow patch updates instead of the current reducer-only local mutation path.

## Complexity Tracking

No constitution violations are required by this design.
