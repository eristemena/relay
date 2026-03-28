# Implementation Plan: Run History and Replay

**Branch**: `010-run-history-replay` | **Date**: 2026-03-28 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/010-run-history-replay/spec.md`
**Input**: Feature specification from `/specs/010-run-history-replay/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extend Relay's current saved-run reopen path into a deterministic historical replay system driven entirely by SQLite-backed event logs and historical approval data. The backend will add a time-based replay scheduler in `internal/orchestrator/workspace` that loads a run's stored events once, emits them over WebSocket at configurable playback speeds, and supports pause, seek, and resume by replaying from lightweight checkpoints rather than re-invoking agents. SQLite will gain indexed run-history documents backed by FTS5 for keyword search across titles, goals, summaries, replay-safe transcript text, and touched file names plus normalized run-change records derived from approval diffs so replay, diff review, and markdown export all consume the same persisted historical source of truth. Markdown export will stay a handler-governed file write that is accepted only for direct developer actions from the workspace client, never from agent-driven replay or orchestration paths. The frontend will extend the existing history and canvas surfaces with a replay scrubber, speed selector, playback status, search and date filters, and a read-only diff review panel while preserving the current workspace-store and React Flow patch model.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library first, especially `context`, `time`, `os`, `path/filepath`, `encoding/json`, and `strings`; existing `modernc.org/sqlite` remains the only database layer and will provide SQLite FTS5 capabilities; existing WebSocket protocol/store layers, React Flow canvas model, and Monaco-based diff review surface from prior repository-aware work remain in use; no new third-party dependency is required for replay scheduling, search, or markdown export  
**Storage**: SQLite only, extending existing `agent_runs`, `agent_run_events`, and `approval_requests` usage with persisted run-history documents, replay-safe transcript search text, FTS5 search index data, and normalized run-change records sourced from stored approval diffs; markdown exports write to `~/.relay/exports/` by default via the backend using standard library file APIs only after a direct developer-initiated export request reaches the handler boundary  
**Testing**: `go test` for `internal/orchestrator/workspace`, `internal/storage/sqlite`, and `internal/handlers/ws`; table-driven Go tests for event-log audits, replay scheduling, seek reconstruction, FTS-backed keyword and file filtering, change-record extraction, direct-user export enforcement, and export generation; Vitest plus React Testing Library for history filters, scrubber controls, diff review, workspace-store replay state, and canvas synchronization; integration coverage under `tests/integration` for restart-safe replay, reconnect restoration, clarification-required replay completion, and full event re-emission  
**Target Platform**: Local Relay development on macOS-first workstations with browser UI on localhost; runtime remains browser-based with WebSocket transport only  
**Project Type**: Full-stack Relay backend/frontend history, replay, search, and export enhancement  
**Performance Goals**: WebSocket replay dispatch remains under 100ms from scheduler tick to emit; opening a saved run starts playback preparation within 1 second; seeking within sessions under 10 minutes completes in under 250ms for at least 95% of local validation attempts by rebuilding from in-memory checkpoints and replaying only the suffix to the target timestamp; history search and filter updates remain responsive while replay is active  
**Constraints**: Dark mode only; WebSocket remains the only backend/frontend runtime channel; SQLite remains the only data store; replay is strictly read-only and must never invoke agents or tools; search uses SQLite FTS5 on generated titles, goals, summaries, replay-safe transcript text, and touched file names; markdown export is the only new file write and must be accepted only for direct developer actions enforced at the handler boundary; historical file review must be reconstructed only from stored approval or event data, never from live repository reads; upstream event emitters in `runs.go` and `orchestration.go` must be audited before implementation because missing historical events cannot be reconstructed  
**Scale/Scope**: Single-user local workstation; one selected replay at a time; sessions typically under 10 minutes and up to a few hundred replayable events; per-run search over local workspace history only; no cloud sync, link sharing, or video export in this phase

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: Go replay services stay in the existing handler -> orchestrator -> storage flow, standard library scheduling and file export cover the new backend behavior, exported Go APIs added for replay services or history queries will require godoc comments, and frontend changes remain strict TypeScript with no banned debug logging.
- [x] Test impact is defined: table-driven Go tests cover event-log completeness audits, deterministic event ordering, seek checkpoint reconstruction, replay-safe transcript and file filtering, clarification-required terminal replay, change-record extraction, direct-user export enforcement, and markdown export; WebSocket integration tests cover new replay-control and history-query protocol paths; React Flow component and store tests cover scrubber state, synchronized canvas updates, and diff review rendering.
- [x] Architecture remains compliant: handlers continue to own WebSocket protocol types, `internal/orchestrator/workspace` owns replay sessions and history coordination, SQLite remains the only persistence layer, no agent code is invoked during replay, and frontend changes stay inside `features/history`, `features/canvas`, and shared workspace protocol/store files.
- [x] UX and governance impact is defined: history query, replay preparation, scrubber movement, diff review, and export all expose visible loading, plain-language errors, and explicit empty states; replay remains read-only and does not alter existing approval enforcement for live file writes and shell commands; markdown export is treated as a direct developer action only when the request enters through the workspace handler and any equivalent non-user path is rejected server-side.
- [x] Security and performance constraints are covered: historical file review uses stored diff data only, export paths stay under `~/.relay/exports`, export requests are accepted only from direct developer actions at the handler boundary, replay workers run with `context.Context` cancellation, SQLite queries avoid per-event N+1 lookups by preloading run data, and backend-managed replay keeps canvas updates responsive while preserving <100ms dispatch expectations.

## Project Structure

### Documentation (this feature)

```text
specs/010-run-history-replay/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── workspace-run-history-replay.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── handlers/
│   └── ws/
│       ├── protocol.go
│       ├── workspace.go
│       └── workspace_test.go
├── orchestrator/
│   └── workspace/
│       ├── history.go
│       ├── runs.go
│       ├── orchestration.go
│       ├── service.go
│       ├── service_test.go
│       └── replay_*.go
└── storage/
    └── sqlite/
        ├── migrations/
        ├── queries/
        ├── models.go
        ├── store.go
        └── store_test.go

tests/
└── integration/
    ├── run_history_replay_test.go
    └── agent_streaming_test.go

web/
└── src/
    ├── features/
    │   ├── canvas/
    │   │   ├── AgentCanvas.tsx
    │   │   ├── AgentNodeDetailPanel.tsx
    │   │   ├── WorkspaceCanvas.tsx
    │   │   └── canvasModel.ts
    │   ├── history/
    │   │   ├── RunHistoryPanel.tsx
    │   │   ├── RunHistoryListItem.tsx
    │   │   └── replay/
    │   └── workspace-shell/
    │       └── WorkspaceShell.tsx
    └── shared/
        └── lib/
            ├── workspace-protocol.ts
            ├── workspace-store.ts
            └── workspace-store.test.ts
```

**Structure Decision**: Keep replay coordination and history search in `internal/orchestrator/workspace` because it already owns `OpenRun`, replay-safe event dispatch, and workspace snapshots. Persist searchable history and change data in `internal/storage/sqlite` rather than adding a sidecar store. On the frontend, extend the existing `features/history` and `features/canvas` surfaces instead of creating a parallel replay app so the scrubber, diff review, and token-usage playback continue using the established workspace store and canvas patch flow.

## Complexity Tracking

No constitution violations are required by this design.
