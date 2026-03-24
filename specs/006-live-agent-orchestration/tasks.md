# Tasks: Live Agent Orchestration

**Input**: Design documents from `/specs/006-live-agent-orchestration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/websocket-events.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include Go unit coverage for agent contract changes, orchestrator DAG execution, SQLite persistence, and WebSocket event dispatch; Go integration coverage for orchestration ordering, reconnect and replay, and agent-error vs run-error behavior; plus frontend component coverage for live canvas patching, selected-node transcript hydration, and the disappearing-node regression.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/agents/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `internal/config/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/006-live-agent-orchestration/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Scaffold the orchestration feature surfaces and update the approved documentation for the new mode.

- [X] T001 Update the orchestration feature docs and runtime notes in `README.md` and `specs/006-live-agent-orchestration/quickstart.md`
- [X] T002 [P] Scaffold orchestration persistence inputs in `internal/storage/sqlite/migrations/0003_orchestration_runs.sql`, `internal/storage/sqlite/queries/orchestration_runs.sql`, `internal/storage/sqlite/queries/agent_executions.sql`, and `internal/storage/sqlite/queries/orchestration_events.sql`
- [X] T003 [P] Scaffold orchestration runtime types in `internal/orchestrator/workspace/run_context.go` and `internal/orchestrator/workspace/orchestration.go`
- [X] T004 [P] Scaffold orchestration-aware frontend state surfaces in `web/src/shared/lib/workspace-protocol.ts`, `web/src/shared/lib/workspace-store.ts`, and `web/src/features/canvas/canvasModel.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the core backend, storage, protocol, and live-canvas patch foundations required by every story.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 Implement SQLite schema, sqlc mappings, and store models for orchestration runs, agent executions, and ordered orchestration events in `internal/storage/sqlite/migrations/0003_orchestration_runs.sql`, `internal/storage/sqlite/models.go`, and `internal/storage/sqlite/store.go`
- [X] T006 [P] Extend WebSocket protocol contracts and TypeScript DTOs for `agent_spawned`, `agent_state_changed`, `task_assigned`, `handoff_start`, `handoff_complete`, `agent_error`, `run_complete`, and `run_error` in `internal/handlers/ws/protocol.go` and `web/src/shared/lib/workspace-protocol.ts`
- [X] T007 [P] Formalize the `Agent` interface and concrete built-in role implementations in `internal/agents/agent.go`, `internal/agents/registry.go`, `internal/agents/planner.go`, `internal/agents/coder.go`, `internal/agents/reviewer.go`, `internal/agents/tester.go`, and `internal/agents/explainer.go`
- [X] T008 [P] Implement coordinator-owned DAG stage types, direct `agent.Run(ctx, task)` goroutine wiring, and cancellable run context primitives in `internal/orchestrator/workspace/run_context.go` and `internal/orchestrator/workspace/orchestration.go`
- [X] T009 Implement orchestration run registration, subscriber fan-out, reconnect-safe live dispatch, and single-active-run guards in `internal/orchestrator/workspace/service.go`, `internal/orchestrator/workspace/runs.go`, and `internal/orchestrator/workspace/history.go`
- [X] T010 [P] Implement orchestration-aware request handling and bootstrap/open-run bridging in `internal/handlers/ws/workspace.go` and `internal/handlers/ws/protocol.go`
- [X] T011 [P] Implement orchestration-aware workspace state, run replay reducers, and `agent_id` keyed event routing in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T012 [P] Refactor the canvas controller to support append-only node spawn and patch-only state updates in `web/src/features/canvas/AgentCanvas.tsx`, `web/src/features/canvas/layoutGraph.ts`, and `web/src/features/canvas/canvasModel.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Watch an orchestration run unfold live (Priority: P1) 🎯 MVP

**Goal**: Let the developer submit one goal and see the Planner, Coder, Tester, Reviewer, and Explainer appear and progress in the correct orchestration order on the live canvas.

**Independent Test**: Submit one goal and verify the Planner appears first, Coder and Tester spawn only after Planner completes, Reviewer starts only after both parallel agents finish, Explainer finishes last, and the canvas remains live throughout.

### Tests for User Story 1

- [X] T013 [P] [US1] Add unit coverage for concrete `Agent` construction, prompt-only role permissions, and registry orchestration selection in `internal/agents/registry_test.go` and `internal/agents/openrouter/client_test.go`
- [X] T014 [P] [US1] Add orchestrator unit coverage for DAG stage transitions, direct per-agent goroutine fan-out, and single-active-run enforcement in `internal/orchestrator/workspace/service_test.go` and `internal/orchestrator/workspace/tool_executor_test.go`
- [X] T015 [P] [US1] Add integration coverage for orchestration event ordering and concurrent Coder or Tester spawn behavior in `tests/integration/agent_streaming_test.go` and `tests/integration/websocket_reconnect_test.go`
- [X] T016 [P] [US1] Add frontend component coverage for `agent_spawned` append behavior, spawn-only dagre execution, and live node-state patching in `web/src/features/canvas/AgentCanvas.test.tsx` and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`

### Implementation for User Story 1

- [X] T017 [P] [US1] Implement orchestration run submission, Planner-first execution, and DAG-driven downstream spawning in `internal/orchestrator/workspace/runs.go` and `internal/orchestrator/workspace/orchestration.go`
- [X] T018 [P] [US1] Implement per-agent event normalization and WebSocket emission for spawn, assignment, handoff, state, token, and completion events in `internal/orchestrator/workspace/service.go` and `internal/handlers/ws/workspace.go`
- [X] T019 [P] [US1] Implement concrete role `Agent` types and prompt-only task assignment flow in `internal/agents/registry.go`, `internal/agents/planner.go`, `internal/agents/coder.go`, `internal/agents/reviewer.go`, `internal/agents/tester.go`, and `internal/agents/explainer.go`
- [X] T020 [P] [US1] Wire orchestration submit, active-run bootstrap hydration, and live event subscription into the frontend socket layer in `web/src/shared/lib/useWorkspaceSocket.ts` and `web/src/shared/lib/workspace-store.ts`
- [X] T021 [P] [US1] Implement live canvas node creation and per-agent state patching driven by `agent_spawned` and `agent_state_changed` events in `web/src/features/canvas/AgentCanvas.tsx` and `web/src/features/canvas/canvasModel.ts`
- [X] T022 [US1] Surface orchestration idle, submitting, active, and completed states plus run summary messaging in `web/src/features/canvas/WorkspaceCanvas.tsx`, `web/src/features/canvas/AgentCanvas.tsx`, and `web/src/features/agent-panel/AgentPanel.tsx`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Inspect live and completed agent output without losing context (Priority: P2)

**Goal**: Let the developer select any live or completed agent node and inspect its current or preserved transcript in the side panel while keeping the canvas interactive.

**Independent Test**: Open a live run, click the active Planner node while it is streaming, then click completed nodes later in the run and verify the side panel always shows the correct selected agent transcript and state without freezing the canvas.

### Tests for User Story 2

- [X] T023 [P] [US2] Add storage and replay coverage for per-agent transcript slices and selected-node hydration in `internal/storage/sqlite/store_test.go` and `internal/orchestrator/workspace/service_test.go`
- [X] T024 [P] [US2] Add integration coverage for run replay, selected-node transcript reconstruction, and reconnect without duplicate nodes in `tests/integration/run_history_replay_test.go` and `tests/integration/websocket_reconnect_test.go`
- [X] T025 [P] [US2] Add frontend component coverage for node selection, side-panel switching, and transcript updates while streaming in `web/src/features/canvas/AgentCanvas.test.tsx` and `web/src/features/history/RunHistoryPanel.test.tsx`

### Implementation for User Story 2

- [X] T026 [P] [US2] Persist per-agent assignment text, transcript slices, and replay-safe selected-node data in `internal/storage/sqlite/store.go`, `internal/orchestrator/workspace/history.go`, and `internal/orchestrator/workspace/runs.go`
- [X] T027 [P] [US2] Implement selected-node transcript projection and live side-panel synchronization in `web/src/features/canvas/canvasModel.ts` and `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [X] T028 [P] [US2] Route `task_assigned`, `token`, and `agent_state_changed` events into the currently selected node without interrupting pan or zoom in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/AgentCanvas.tsx`
- [X] T029 [US2] Integrate completed-run open and replay into the canvas-oriented inspection flow in `internal/handlers/ws/workspace.go`, `web/src/shared/lib/useWorkspaceSocket.ts`, `web/src/features/history/RunHistoryPanel.tsx`, and `web/src/features/canvas/WorkspaceCanvas.tsx`

**Checkpoint**: User Stories 1 and 2 work independently, including live and replayed node inspection.

---

## Phase 5: User Story 3 - Understand partial failures and run-level halts clearly (Priority: P3)

**Goal**: Distinguish agent-scoped failures from unrecoverable run-level failures while preserving inspectable node output and preventing invalid downstream progress.

**Independent Test**: Observe one run where an individual agent errors and the run remains inspectable, then another where a run-level failure halts remaining work, preserves started nodes, and shows a clear halt reason.

### Tests for User Story 3

- [X] T030 [P] [US3] Add orchestrator unit coverage for `agent_error` continuation rules, `run_error` halts, and downstream blocking after planner failure in `internal/orchestrator/workspace/service_test.go` and `internal/orchestrator/workspace/runs.go`
- [X] T031 [P] [US3] Add integration coverage for agent-error versus run-error event sequences and halted replay behavior in `tests/integration/agent_streaming_test.go` and `tests/integration/run_history_replay_test.go`
- [X] T032 [P] [US3] Add frontend component coverage for errored nodes, halted-run messaging, and no-spawn-after-halt behavior in `web/src/features/canvas/AgentCanvas.test.tsx` and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`

### Implementation for User Story 3

- [X] T033 [P] [US3] Implement agent-error versus run-error branching, downstream eligibility checks, and halted-run state transitions in `internal/orchestrator/workspace/orchestration.go` and `internal/orchestrator/workspace/runs.go`
- [X] T034 [P] [US3] Persist and replay run halt reasons plus agent terminal failures in `internal/storage/sqlite/models.go`, `internal/storage/sqlite/store.go`, and `internal/orchestrator/workspace/history.go`
- [X] T035 [P] [US3] Emit and hydrate `agent_error` and `run_error` payloads with clear human-readable messaging in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, and `web/src/shared/lib/workspace-protocol.ts`
- [X] T036 [US3] Render errored nodes, halted-run banners, and preserved failure transcripts in `web/src/features/canvas/AgentCanvasNode.tsx`, `web/src/features/canvas/AgentCanvas.tsx`, and `web/src/features/canvas/AgentNodeDetailPanel.tsx`

**Checkpoint**: All three user stories work independently, including partial failure visibility and run-level halt behavior.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize docs, regression protection, accessibility, and quickstart validation across the orchestration feature.

- [X] T037 [P] Update the orchestration docs, event contract notes, and implementation decisions in `README.md`, `specs/006-live-agent-orchestration/research.md`, and `specs/006-live-agent-orchestration/contracts/websocket-events.md`
- [X] T038 Verify `internal/agents`, `internal/orchestrator/workspace`, and any touched core packages remain at or above 75% coverage in `internal/agents/registry_test.go`, `internal/orchestrator/workspace/service_test.go`, and `tests/integration/agent_streaming_test.go`
- [X] T039 [P] Run and fix the focused backend and frontend regression suites for orchestration ordering, replay, and canvas patch stability in `tests/integration/agent_streaming_test.go`, `tests/integration/websocket_reconnect_test.go`, `web/src/features/canvas/AgentCanvas.test.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`
- [X] T040 [P] Verify keyboard access, visible focus, explicit empty and halted states, forced-colors behavior, and 320px reflow in `web/src/features/canvas/AgentCanvas.tsx`, `web/src/features/canvas/AgentNodeDetailPanel.tsx`, and `web/src/app/globals.css`
- [X] T041 Run the full quickstart validation flow and capture any follow-up fixes in `specs/006-live-agent-orchestration/quickstart.md`, `README.md`, and `tests/integration/run_history_replay_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP.
- **Phase 4: US2** depends on Foundational and on the live orchestration surface delivered in US1.
- **Phase 5: US3** depends on Foundational and on the orchestration lifecycle delivered in US1; it reuses US2 inspection surfaces for failure review.
- **Phase 6: Polish** depends on the completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on US1 because node inspection requires live node creation and replayable per-agent state.
- **US3**: Depends on US1 for orchestration execution and benefits from US2 for transcript inspection, but its backend halt semantics can be developed in parallel after Foundational work if coordinated carefully.

### Within Each User Story

- Write tests first and confirm they fail before implementation.
- Complete backend orchestration and persistence work before final UI wiring for that story.
- Preserve prompt-only execution boundaries and do not introduce repo access, file writes, or shell commands.
- Finish empty, loading, blocked, and recoverable error states before marking the story complete.
- Keep canvas node updates patch-based; do not rerun dagre on non-spawn events.

## Parallel Opportunities

- `T002`, `T003`, and `T004` can run in parallel during Setup.
- `T006`, `T007`, `T008`, `T010`, `T011`, and `T012` can run in parallel within Foundational work after storage direction in `T005` is set.
- `T013`, `T014`, `T015`, and `T016` can run in parallel for US1 tests.
- `T017`, `T018`, `T019`, `T020`, and `T021` can run in parallel after US1 tests are in place.
- `T023`, `T024`, and `T025` can run in parallel for US2 tests.
- `T026`, `T027`, and `T028` can run in parallel after US2 tests are in place.
- `T030`, `T031`, and `T032` can run in parallel for US3 tests.
- `T033`, `T034`, and `T035` can run in parallel after US3 tests are in place.
- `T037`, `T039`, and `T040` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T013 internal/agents/registry_test.go + internal/agents/openrouter/client_test.go
Task: T014 internal/orchestrator/workspace/service_test.go + internal/orchestrator/workspace/tool_executor_test.go
Task: T015 tests/integration/agent_streaming_test.go + tests/integration/websocket_reconnect_test.go
Task: T016 web/src/features/canvas/AgentCanvas.test.tsx + web/src/features/workspace-shell/WorkspaceShell.test.tsx

# Parallel US1 implementation work
Task: T017 internal/orchestrator/workspace/runs.go + internal/orchestrator/workspace/orchestration.go
Task: T018 internal/orchestrator/workspace/service.go + internal/handlers/ws/workspace.go
Task: T019 internal/agents/registry.go + internal/agents/planner.go + internal/agents/coder.go + internal/agents/reviewer.go + internal/agents/tester.go + internal/agents/explainer.go
Task: T020 web/src/shared/lib/useWorkspaceSocket.ts + web/src/shared/lib/workspace-store.ts
Task: T021 web/src/features/canvas/AgentCanvas.tsx + web/src/features/canvas/canvasModel.ts
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T023 internal/storage/sqlite/store_test.go + internal/orchestrator/workspace/service_test.go
Task: T024 tests/integration/run_history_replay_test.go + tests/integration/websocket_reconnect_test.go
Task: T025 web/src/features/canvas/AgentCanvas.test.tsx + web/src/features/history/RunHistoryPanel.test.tsx

# Parallel US2 implementation work
Task: T026 internal/storage/sqlite/store.go + internal/orchestrator/workspace/history.go + internal/orchestrator/workspace/runs.go
Task: T027 web/src/features/canvas/canvasModel.ts + web/src/features/canvas/AgentNodeDetailPanel.tsx
Task: T028 web/src/shared/lib/workspace-store.ts + web/src/features/canvas/AgentCanvas.tsx
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T030 internal/orchestrator/workspace/service_test.go + internal/orchestrator/workspace/runs.go
Task: T031 tests/integration/agent_streaming_test.go + tests/integration/run_history_replay_test.go
Task: T032 web/src/features/canvas/AgentCanvas.test.tsx + web/src/features/workspace-shell/WorkspaceShell.test.tsx

# Parallel US3 implementation work
Task: T033 internal/orchestrator/workspace/orchestration.go + internal/orchestrator/workspace/runs.go
Task: T034 internal/storage/sqlite/models.go + internal/storage/sqlite/store.go + internal/orchestrator/workspace/history.go
Task: T035 internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go + web/src/shared/lib/workspace-protocol.ts
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate orchestration ordering and spawn-only canvas patching before expanding scope.

### Incremental Delivery

1. Deliver US1 for the live orchestration graph and correct DAG sequencing.
2. Add US2 for selected-node live and replayed transcript inspection.
3. Add US3 for agent-error versus run-error clarity and halt semantics.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer owns backend orchestration and persistence during Foundational work.
2. One developer owns WebSocket DTOs and frontend workspace-store wiring during Foundational and US1.
3. After the foundation is stable, one developer can focus on transcript inspection while another focuses on failure semantics and halted-run behavior.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the limits described in the specification.
- Do not bypass the Relay architecture boundary of handlers to orchestrator to agents to tools to storage.
- Do not introduce repo reads, file writes, shell commands, or tool transcript events into this prompt-only orchestration mode.
- The live canvas regression guard is mandatory: node creation uses append semantics, state changes use patch semantics, and dagre runs only on `agent_spawned`.