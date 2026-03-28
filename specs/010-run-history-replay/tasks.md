# Tasks: Run History and Replay

**Input**: Design documents from `/specs/010-run-history-replay/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/workspace-run-history-replay.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and Relay constitution. Include Go unit coverage for event-log completeness, replay scheduling, checkpoint-backed seek reconstruction, SQLite FTS history queries, transcript-search indexing, run-change extraction, direct-user export enforcement, and markdown export; Go integration coverage for WebSocket replay, history query, replay control, clarification-required replay completion, reconnect restoration, and export rejection; plus Vitest and React Testing Library coverage for the run history panel, replay controls, diff review surface, workspace-store replay state, and canvas synchronization.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/010-run-history-replay/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare replay-specific files, protocol types, and UI shells before shared implementation begins.

- [X] T001 Create replay implementation scaffolding in `internal/orchestrator/workspace/replay_scheduler.go`, `internal/orchestrator/workspace/replay_checkpoint.go`, and `internal/orchestrator/workspace/export.go`
- [X] T002 [P] Add history-query, replay-control, replay-state, detail, and export protocol scaffolding in `internal/handlers/ws/protocol.go`, `web/src/shared/lib/workspace-protocol.ts`, and `web/src/shared/lib/workspace-store.ts`
- [X] T003 [P] Create replay UI shells in `web/src/features/history/replay/ReplayControls.tsx`, `web/src/features/history/replay/ReplayTimeline.tsx`, `web/src/features/history/replay/RunHistoryFilters.tsx`, and `web/src/features/history/replay/RunChangeReviewPanel.tsx`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Complete the shared replay, persistence, and state foundations required before any user story can ship.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Audit and close replay-event persistence gaps across `internal/orchestrator/workspace/runs.go`, `internal/orchestrator/workspace/orchestration.go`, and `internal/orchestrator/workspace/service.go`
- [X] T005 [P] Add failing coverage for event completeness, replay ordering, and clarification-required terminal replay in `internal/orchestrator/workspace/service_test.go`, `internal/orchestrator/workspace/history_test.go`, and `tests/integration/run_history_replay_test.go`
- [X] T006 [P] Implement SQLite schema, queries, models, and store APIs for run history documents, search documents, change records, and export metadata in `internal/storage/sqlite/migrations/0006_run_history_replay.sql`, `internal/storage/sqlite/queries/run_history.sql`, `internal/storage/sqlite/queries/run_change_records.sql`, `internal/storage/sqlite/models.go`, `internal/storage/sqlite/store.go`, and `internal/storage/sqlite/store_test.go`
- [X] T007 [P] Extend shared historical-run selection and replay metadata plumbing in `internal/handlers/ws/workspace.go`, `internal/handlers/ws/workspace_test.go`, and `web/src/shared/lib/workspace-store.ts`
- [X] T008 [P] Add shared replay-state reduction hooks for canvas, transcript, approvals, tool activity, and token playback in `internal/orchestrator/workspace/history.go`, `web/src/shared/lib/workspace-store.ts`, and `web/src/features/canvas/canvasModel.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Replay any past run from history (Priority: P1) 🎯 MVP

**Goal**: Let a developer open a saved run from the toolbar-triggered run history panel and watch it replay faithfully on the canvas from stored events.

**Independent Test**: Open a saved run from the run history panel and confirm Relay reconstructs the recorded timeline in order, including node creation, state changes, transcript content, and terminal status for completed, halted, errored, and clarification-required runs, without creating any new live agent activity.

### Tests for User Story 1

- [X] T009 [P] [US1] Add Go unit coverage for replay scheduler bootstrap, timestamp normalization, and replay-session restoration in `internal/orchestrator/workspace/service_test.go` and `internal/orchestrator/workspace/history_test.go`
- [X] T010 [P] [US1] Add WebSocket integration coverage for deterministic historical replay and reconnect restoration in `tests/integration/run_history_replay_test.go`
- [X] T011 [P] [US1] Add frontend tests for run-history selection and replayed canvas updates in `web/src/features/history/RunHistoryPanel.test.tsx`, `web/src/shared/lib/workspace-store.test.ts`, and `web/src/features/canvas/AgentCanvas.test.tsx`

### Implementation for User Story 1

- [X] T012 [P] [US1] Implement backend replay session bootstrap and scheduled event emission in `internal/orchestrator/workspace/history.go`, `internal/orchestrator/workspace/replay_scheduler.go`, and `internal/orchestrator/workspace/service.go`
- [X] T013 [P] [US1] Add historical open-run and replay-state protocol handling in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, and `internal/orchestrator/workspace/service.go`
- [X] T014 [P] [US1] Extend saved-run summaries with generated titles, agent counts, and terminal status metadata in `internal/storage/sqlite/queries/agent_runs.sql`, `internal/storage/sqlite/store.go`, and `internal/storage/sqlite/models.go`
- [X] T015 [P] [US1] Update workspace replay ingestion and canvas hydration for scheduled historical events in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/canvasModel.ts`
- [X] T016 [US1] Render run history panel loading, empty, error, and replay-state UI in `web/src/features/history/RunHistoryPanel.tsx`, `web/src/features/history/RunHistoryListItem.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Inspect, search, and review recorded run artifacts (Priority: P2)

**Goal**: Let a developer search saved runs, filter by file or date, and inspect preserved before-and-after diffs for recorded file changes.

**Independent Test**: Filter the run history by keyword, touched file, and date range, then open a matching run and verify the diff review surface shows full preserved before and after content for each changed file while keyword results include replay-safe transcript matches.

### Tests for User Story 2

- [X] T017 [P] [US2] Add Go unit coverage for FTS-backed keyword, transcript, file, and date filtering in `internal/storage/sqlite/store_test.go` and `internal/orchestrator/workspace/service_test.go`
- [X] T018 [P] [US2] Add WebSocket and handler coverage for history query and details responses in `tests/integration/run_history_replay_test.go` and `internal/handlers/ws/workspace_test.go`
- [X] T019 [P] [US2] Add frontend tests for history filters and diff review states in `web/src/features/history/RunHistoryPanel.test.tsx`, `web/src/shared/lib/workspace-store.test.ts`, and `web/src/features/history/replay/RunChangeReviewPanel.test.tsx`

### Implementation for User Story 2

- [X] T020 [P] [US2] Build transcript-aware search document indexing and change-record extraction in `internal/storage/sqlite/store.go`, `internal/storage/sqlite/queries/run_history.sql`, and `internal/storage/sqlite/queries/run_change_records.sql`
- [X] T021 [P] [US2] Implement history query and detail services over search documents and normalized change records in `internal/orchestrator/workspace/service.go` and `internal/orchestrator/workspace/history.go`
- [X] T022 [P] [US2] Add `run.history.query` and `run.history.details.request` handler flows in `internal/handlers/ws/protocol.go` and `internal/handlers/ws/workspace.go`
- [X] T023 [P] [US2] Extend workspace-store caches for filtered history results and selected run details in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/workspace-protocol.ts`
- [X] T024 [US2] Implement transcript-aware filters and read-only diff review UI in `web/src/features/history/RunHistoryPanel.tsx`, `web/src/features/history/replay/RunHistoryFilters.tsx`, `web/src/features/history/replay/RunChangeReviewPanel.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`

**Checkpoint**: User Stories 1 and 2 both work independently, including searchable history and preserved diff review.

---

## Phase 5: User Story 3 - Control playback and export a reusable report (Priority: P3)

**Goal**: Let a developer play, pause, seek, change playback speed, and export a full markdown report for a selected saved run.

**Independent Test**: Start replay for a saved run, change playback speed across all supported options, seek to multiple timestamps, and export the run to markdown under `~/.relay/exports/`, while non-user export attempts are rejected server-side and the canvas and supporting panels remain synchronized.

### Tests for User Story 3

- [X] T025 [P] [US3] Add Go unit coverage for checkpoint-backed seek, playback speed control, and direct-user export enforcement in `internal/orchestrator/workspace/service_test.go`, `internal/orchestrator/workspace/history_test.go`, and `internal/handlers/ws/workspace_test.go`
- [X] T026 [P] [US3] Add integration coverage for replay controls, export success, and export rejection in `tests/integration/run_history_replay_test.go`
- [X] T027 [P] [US3] Add frontend tests for scrubber, speed selector, export progress, and export error states in `web/src/features/history/replay/ReplayControls.test.tsx`, `web/src/features/history/replay/ReplayTimeline.test.tsx`, and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 3

- [X] T028 [P] [US3] Implement checkpoint-backed seek reconstruction and playback speed control in `internal/orchestrator/workspace/history.go`, `internal/orchestrator/workspace/replay_checkpoint.go`, and `internal/orchestrator/workspace/replay_scheduler.go`
- [X] T029 [P] [US3] Add `agent.run.replay.control` and `agent.run.replay.state` handling in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, and `internal/orchestrator/workspace/service.go`
- [X] T030 [P] [US3] Implement backend markdown export generation from stored history in `internal/orchestrator/workspace/export.go`, `internal/storage/sqlite/store.go`, and `internal/storage/sqlite/models.go`
- [X] T031 [P] [US3] Enforce direct-developer export approval at the handler boundary in `internal/handlers/ws/workspace.go`, `internal/orchestrator/workspace/export.go`, and `internal/handlers/ws/workspace_test.go`
- [X] T032 [P] [US3] Extend workspace-store replay cursor, speed, selected timestamp, and export state in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/workspace-protocol.ts`
- [X] T033 [US3] Build replay controls, scrubber, and export UI in `web/src/features/history/replay/ReplayControls.tsx`, `web/src/features/history/replay/ReplayTimeline.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [X] T034 [US3] Keep canvas, transcript, token-usage, and diff-review surfaces synchronized on seek in `web/src/features/canvas/AgentCanvas.tsx`, `web/src/features/canvas/AgentNodeDetailPanel.tsx`, `web/src/features/canvas/canvasModel.ts`, and `web/src/features/history/replay/RunChangeReviewPanel.tsx`

**Checkpoint**: All three user stories work independently, including replay controls and governed markdown export.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, accessibility, performance, and regression validation across replay, search, diff review, and export.

- [X] T035 [P] Update release-facing documentation and protocol notes in `README.md`, `specs/010-run-history-replay/quickstart.md`, and `specs/010-run-history-replay/contracts/workspace-run-history-replay.md`
- [X] T036 Verify focused backend, integration, and frontend regression suites for replay, search, and export in `internal/orchestrator/workspace/service_test.go`, `internal/storage/sqlite/store_test.go`, `internal/handlers/ws/workspace_test.go`, `tests/integration/run_history_replay_test.go`, and `web/src/shared/lib/workspace-store.test.ts`
- [X] T037 [P] Validate keyboard access, visible focus, forced-colors behavior, and 320px reflow for replay controls and diff review in `web/src/features/history/replay/ReplayControls.tsx`, `web/src/features/history/replay/RunChangeReviewPanel.tsx`, and `web/src/app/globals.css`
- [X] T038 Run the quickstart validation flow and reconcile any command, performance-target, or edge-case updates in `specs/010-run-history-replay/quickstart.md` and `README.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP historical replay experience.
- **Phase 4: US2** depends on Foundational and the selected-run and replay context introduced by US1.
- **Phase 5: US3** depends on Foundational and builds on the replay session and detail surfaces used by US1 and US2.
- **Phase 6: Polish** depends on completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on US1 because the searchable history and diff review extend the same selected-run panel and replay context introduced for opening saved runs.
- **US3**: Depends on US1 and US2 because playback controls and export act on the selected replay session and its detail surfaces.

### Within Each User Story

- Write the listed tests first and confirm they fail before implementation.
- Complete backend persistence and protocol work before final frontend rendering for that story.
- Preserve WebSocket-only communication and SQLite-only persistence throughout implementation.
- Keep replay read-only and do not re-trigger approvals, tool calls, or agent execution.
- Preserve synchronization between canvas, transcript, token usage, approvals, and diff review before marking the story complete.

## Parallel Opportunities

- `T002` and `T003` can run in parallel after `T001` begins Setup.
- `T005`, `T006`, `T007`, and `T008` can run in parallel once `T004` defines the replay event-completeness matrix.
- `T009`, `T010`, and `T011` can run in parallel for US1 tests.
- `T012`, `T013`, `T014`, and `T015` can run in parallel after US1 tests are in place; `T016` follows once the data reaches the panel and store.
- `T017`, `T018`, and `T019` can run in parallel for US2 tests.
- `T020`, `T021`, `T022`, and `T023` can run in parallel after US2 tests are in place; `T024` follows once the backend payloads are available.
- `T025`, `T026`, and `T027` can run in parallel for US3 tests.
- `T028`, `T029`, `T030`, `T031`, and `T032` can run in parallel after US3 tests are in place; `T033` and `T034` follow once replay-control state is wired.
- `T035` and `T037` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T009 internal/orchestrator/workspace/service_test.go + internal/orchestrator/workspace/history_test.go
Task: T010 tests/integration/run_history_replay_test.go
Task: T011 web/src/features/history/RunHistoryPanel.test.tsx + web/src/shared/lib/workspace-store.test.ts + web/src/features/canvas/AgentCanvas.test.tsx

# Parallel US1 implementation work
Task: T012 internal/orchestrator/workspace/history.go + internal/orchestrator/workspace/replay_scheduler.go + internal/orchestrator/workspace/service.go
Task: T013 internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go
Task: T014 internal/storage/sqlite/queries/agent_runs.sql + internal/storage/sqlite/store.go + internal/storage/sqlite/models.go
Task: T015 web/src/shared/lib/workspace-store.ts + web/src/features/canvas/canvasModel.ts
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T017 internal/storage/sqlite/store_test.go + internal/orchestrator/workspace/service_test.go
Task: T018 tests/integration/run_history_replay_test.go + internal/handlers/ws/workspace_test.go
Task: T019 web/src/features/history/RunHistoryPanel.test.tsx + web/src/shared/lib/workspace-store.test.ts + web/src/features/history/replay/RunChangeReviewPanel.test.tsx

# Parallel US2 implementation work
Task: T020 internal/storage/sqlite/store.go + internal/storage/sqlite/queries/run_history.sql + internal/storage/sqlite/queries/run_change_records.sql
Task: T021 internal/orchestrator/workspace/service.go + internal/orchestrator/workspace/history.go
Task: T022 internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go
Task: T023 web/src/shared/lib/workspace-store.ts + web/src/shared/lib/workspace-protocol.ts
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T025 internal/orchestrator/workspace/service_test.go + internal/orchestrator/workspace/history_test.go + internal/handlers/ws/workspace_test.go
Task: T026 tests/integration/run_history_replay_test.go
Task: T027 web/src/features/history/replay/ReplayControls.test.tsx + web/src/features/history/replay/ReplayTimeline.test.tsx + web/src/shared/lib/workspace-store.test.ts

# Parallel US3 implementation work
Task: T028 internal/orchestrator/workspace/history.go + internal/orchestrator/workspace/replay_checkpoint.go + internal/orchestrator/workspace/replay_scheduler.go
Task: T029 internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go + internal/orchestrator/workspace/service.go
Task: T030 internal/orchestrator/workspace/export.go + internal/storage/sqlite/store.go + internal/storage/sqlite/models.go
Task: T031 internal/handlers/ws/workspace.go + internal/orchestrator/workspace/export.go + internal/handlers/ws/workspace_test.go
Task: T032 web/src/shared/lib/workspace-store.ts + web/src/shared/lib/workspace-protocol.ts
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate deterministic replay from the run history panel before expanding to search, diff review, and export.

### Incremental Delivery

1. Deliver US1 for deterministic historical replay from saved runs.
2. Add US2 for search, file filtering, and preserved diff review.
3. Add US3 for replay controls, seek, and markdown export.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer can own backend replay scheduling and handler plumbing across `internal/orchestrator/workspace` and `internal/handlers/ws`.
2. One developer can own SQLite history documents, transcript-aware search, and run-change extraction across `internal/storage/sqlite`.
3. One developer can own the run history panel, replay controls, and workspace-store integration across `web/src/features/history`, `web/src/features/workspace-shell`, and `web/src/shared/lib`.

## Notes

- [P] tasks touch separate files or proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the limits described in the specification.
- Do not infer missing historical events or read current repository files to reconstruct past changes.
- Keep export writes bounded to `~/.relay/exports/` and preserve handler-level governance for any server-side file output.
- Preserve the canonical term `run history panel` rather than reintroducing `sidebar` terminology in implementation or docs.