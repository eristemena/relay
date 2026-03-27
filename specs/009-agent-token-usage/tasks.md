# Tasks: Agent Token Usage

**Input**: Design documents from `/specs/009-agent-token-usage/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/workspace-token-usage.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include Go unit coverage for final-chunk usage capture, model-limit cache resolution, SQLite event persistence, and replay hydration; Go integration coverage for live streaming and replayed token telemetry; plus Vitest and React Testing Library coverage for workspace-store updates, canvas state derivation, and token-bar rendering and fallback states.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/agents/`, `internal/config/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/009-agent-token-usage/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the token-usage feature entry points and implementation scaffolding before shared infrastructure work begins.

- [x] T001 Create the event-storage migration scaffold in `internal/storage/sqlite/migrations/0005_agent_run_event_token_usage.sql` and align query placeholders in `internal/storage/sqlite/queries/agent_run_events.sql`
- [x] T002 [P] Add token-usage payload scaffolding to `internal/agents/agent.go`, `internal/handlers/ws/protocol.go`, and `web/src/shared/lib/workspace-protocol.ts`
- [x] T003 [P] Stage quickstart and contract validation targets for token usage in `specs/009-agent-token-usage/quickstart.md` and `specs/009-agent-token-usage/contracts/workspace-token-usage.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the shared provider, persistence, replay, and state-derivation infrastructure required before any user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Extend completion callback contracts for normalized usage metadata in `internal/agents/agent.go`, `internal/agents/registry.go`, and `internal/agents/openrouter/client.go`
- [x] T005 [P] Add startup-loaded model metadata cache and context-limit resolver scaffolding in `internal/orchestrator/workspace/service.go`, `internal/orchestrator/workspace/service_test.go`, and `internal/config/config.go`
- [x] T006 [P] Extend SQLite run-event models and store APIs for nullable `tokens_used` and `context_limit` columns in `internal/storage/sqlite/models.go`, `internal/storage/sqlite/store.go`, and `internal/storage/sqlite/store_test.go`
- [x] T007 [P] Extend replay and WebSocket protocol plumbing for optional token telemetry in `internal/orchestrator/workspace/history.go`, `internal/handlers/ws/protocol.go`, and `web/src/shared/lib/workspace-protocol.ts`
- [x] T008 [P] Add shared token-usage state plumbing to the frontend workspace store and canvas model in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/canvasModel.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Monitor live token usage per agent (Priority: P1) 🎯 MVP

**Goal**: Show live per-agent token usage on the canvas as runs complete, including neutral, warning, and critical threshold states.

**Independent Test**: Start a live run that yields authoritative usage data in the terminal provider chunk and confirm the relevant canvas node updates without reload, using the correct fill width and threshold color state.

### Tests for User Story 1

- [x] T009 [P] [US1] Add unit coverage for final-chunk usage capture and completion metadata propagation in `internal/agents/openrouter/client_test.go` and `internal/orchestrator/workspace/service_test.go`
- [x] T010 [P] [US1] Add integration coverage for live token-usage event delivery in `tests/integration/agent_streaming_test.go`
- [x] T011 [P] [US1] Add frontend store and canvas tests for live token-bar updates in `web/src/shared/lib/workspace-store.test.ts`, `web/src/features/canvas/canvasModel.test.ts`, and `web/src/features/canvas/AgentCanvasNode.test.tsx`

### Implementation for User Story 1

- [x] T012 [P] [US1] Capture `usage.total_tokens` from the final OpenRouter stream chunk and pass normalized completion metadata through `internal/agents/openrouter/client.go` and `internal/agents/registry.go`
- [x] T013 [P] [US1] Emit live `tokens_used` and `context_limit` values on single-agent and orchestration terminal events in `internal/orchestrator/workspace/runs.go` and `internal/orchestrator/workspace/orchestration.go`
- [x] T014 [P] [US1] Apply live token-usage event data to frontend run and node state in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/canvasModel.ts`
- [x] T015 [US1] Render the live token usage fill bar and threshold styling in `web/src/features/canvas/AgentCanvasNode.tsx` and `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [x] T016 [US1] Add visible live fallback copy and non-blocking update behavior for the token bar in `web/src/features/canvas/AgentCanvas.tsx` and `web/src/features/canvas/AgentCanvasNode.tsx`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Revisit token usage in prior runs (Priority: P2)

**Goal**: Persist token telemetry with run events and replay the same values on historical runs recorded after the feature ships.

**Independent Test**: Open a saved run created after the change and confirm replay emits the stored token values, updates the canvas bar state correctly, and leaves pre-change runs stable with no fabricated values.

### Tests for User Story 2

- [x] T017 [P] [US2] Add unit coverage for token-usage column persistence and replay hydration in `internal/storage/sqlite/store_test.go` and `internal/orchestrator/workspace/service_test.go`
- [x] T018 [P] [US2] Add integration coverage for replayed token telemetry and mixed old or new event histories in `tests/integration/run_history_replay_test.go`
- [x] T019 [P] [US2] Add frontend replay tests for stored token usage application in `web/src/shared/lib/workspace-store.test.ts` and `web/src/features/canvas/canvasModel.test.ts`

### Implementation for User Story 2

- [x] T020 [P] [US2] Add nullable token-usage columns and query writes to `internal/storage/sqlite/migrations/0005_agent_run_event_token_usage.sql`, `internal/storage/sqlite/queries/agent_run_events.sql`, and `internal/storage/sqlite/store.go`
- [x] T021 [P] [US2] Merge stored token-usage columns into replayed event payloads in `internal/orchestrator/workspace/history.go` and `internal/handlers/ws/protocol.go`
- [x] T022 [P] [US2] Hydrate replayed token usage into workspace state and node models in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/canvasModel.ts`
- [x] T023 [US2] Preserve history-view behavior and plain-language empty states for older events in `web/src/features/canvas/AgentCanvas.tsx` and `web/src/features/canvas/AgentNodeDetailPanel.tsx`

**Checkpoint**: User Stories 1 and 2 both work independently, including persisted replay for new runs and safe handling for older runs.

---

## Phase 5: User Story 3 - Trust incomplete or unavailable usage states (Priority: P3)

**Goal**: Degrade gracefully when provider usage is missing or model limits are unavailable or inconsistent, without misleading the developer or crashing the canvas.

**Independent Test**: Stream or replay runs with missing usage, missing context limits, zero or invalid limits, and over-limit values, then confirm Relay shows the correct unavailable or capped-critical state instead of guessing or failing.

### Tests for User Story 3

- [x] T024 [P] [US3] Add unit coverage for model-limit cache refresh and fallback resolution in `internal/orchestrator/workspace/service_test.go` and `internal/config/config_test.go`
- [x] T025 [P] [US3] Add integration coverage for missing usage and invalid-limit handling in `tests/integration/agent_streaming_test.go` and `tests/integration/run_history_replay_test.go`
- [x] T026 [P] [US3] Add frontend tests for unavailable, raw-count-only, and capped-critical token states in `web/src/features/canvas/AgentCanvasNode.test.tsx` and `web/src/features/canvas/canvasModel.test.ts`

### Implementation for User Story 3

- [x] T027 [P] [US3] Implement the startup-loaded TTL model metadata cache and fallback limit resolution in `internal/orchestrator/workspace/service.go` and `internal/config/config.go`
- [x] T028 [P] [US3] Handle missing, zero, and over-limit token values during completion assembly in `internal/agents/openrouter/client.go`, `internal/orchestrator/workspace/runs.go`, and `internal/orchestrator/workspace/orchestration.go`
- [x] T029 [P] [US3] Derive unavailable, raw-count-only, and capped-critical node states in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/canvasModel.ts`
- [x] T030 [US3] Render the plain-language unavailable and raw-count fallback states in `web/src/features/canvas/AgentCanvasNode.tsx` and `web/src/features/canvas/AgentNodeDetailPanel.tsx`

**Checkpoint**: All three user stories work independently, including graceful degradation for incomplete telemetry.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, performance, accessibility, and regression validation across live, replayed, and degraded token-usage states.

- [x] T031 [P] Update release-facing documentation and feature notes in `README.md`, `specs/009-agent-token-usage/quickstart.md`, and `specs/009-agent-token-usage/research.md`
- [x] T032 Verify focused backend and frontend coverage plus regression suites for token usage in `internal/agents/openrouter/client_test.go`, `internal/orchestrator/workspace/service_test.go`, `internal/storage/sqlite/store_test.go`, `tests/integration/run_history_replay_test.go`, and `web/src/shared/lib/workspace-store.test.ts`
- [x] T033 [P] Validate keyboard access, visible focus, forced-colors behavior, and 320px reflow for the token bar UI in `web/src/features/canvas/AgentCanvasNode.tsx`, `web/src/features/canvas/AgentNodeDetailPanel.tsx`, and `web/src/app/globals.css`
- [x] T034 Run the quickstart validation flow and reconcile any command or edge-case updates in `specs/009-agent-token-usage/quickstart.md` and `README.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP live token-usage experience.
- **Phase 4: US2** depends on Foundational and the event or state plumbing established for US1.
- **Phase 5: US3** depends on Foundational and integrates with the live and replay paths established by US1 and US2.
- **Phase 6: Polish** depends on completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on US1 because replay uses the same token event fields and node state model introduced for live updates.
- **US3**: Depends on US1 and US2 because graceful degradation spans both live and replayed token-usage paths.

### Within Each User Story

- Write the listed tests first and confirm they fail before implementation.
- Complete backend event and persistence work before final frontend rendering for that story.
- Keep replay behavior backward-compatible with older events before marking US2 complete.
- Implement unavailable and invalid-data states before marking US3 complete.
- Preserve Relay’s handler -> orchestrator -> agent -> tool -> storage boundaries throughout the work.

## Parallel Opportunities

- `T002` and `T003` can run in parallel after `T001` begins Setup.
- `T005`, `T006`, `T007`, and `T008` can run in parallel once `T004` defines the normalized completion metadata contract.
- `T009`, `T010`, and `T011` can run in parallel for US1 tests.
- `T012`, `T013`, and `T014` can run in parallel after US1 tests are in place; `T015` and `T016` follow once data reaches the node model.
- `T017`, `T018`, and `T019` can run in parallel for US2 tests.
- `T020`, `T021`, and `T022` can run in parallel after US2 tests are in place; `T023` follows once replay state is wired.
- `T024`, `T025`, and `T026` can run in parallel for US3 tests.
- `T027`, `T028`, and `T029` can run in parallel after US3 tests are in place; `T030` follows once the fallback states are in the model.
- `T031` and `T033` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T009 internal/agents/openrouter/client_test.go + internal/orchestrator/workspace/service_test.go
Task: T010 tests/integration/agent_streaming_test.go
Task: T011 web/src/shared/lib/workspace-store.test.ts + web/src/features/canvas/canvasModel.test.ts + web/src/features/canvas/AgentCanvasNode.test.tsx

# Parallel US1 implementation work
Task: T012 internal/agents/openrouter/client.go + internal/agents/registry.go
Task: T013 internal/orchestrator/workspace/runs.go + internal/orchestrator/workspace/orchestration.go
Task: T014 web/src/shared/lib/workspace-store.ts + web/src/features/canvas/canvasModel.ts
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T017 internal/storage/sqlite/store_test.go + internal/orchestrator/workspace/service_test.go
Task: T018 tests/integration/run_history_replay_test.go
Task: T019 web/src/shared/lib/workspace-store.test.ts + web/src/features/canvas/canvasModel.test.ts

# Parallel US2 implementation work
Task: T020 internal/storage/sqlite/migrations/0005_agent_run_event_token_usage.sql + internal/storage/sqlite/queries/agent_run_events.sql + internal/storage/sqlite/store.go
Task: T021 internal/orchestrator/workspace/history.go + internal/handlers/ws/protocol.go
Task: T022 web/src/shared/lib/workspace-store.ts + web/src/features/canvas/canvasModel.ts
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T024 internal/orchestrator/workspace/service_test.go + internal/config/config_test.go
Task: T025 tests/integration/agent_streaming_test.go + tests/integration/run_history_replay_test.go
Task: T026 web/src/features/canvas/AgentCanvasNode.test.tsx + web/src/features/canvas/canvasModel.test.ts

# Parallel US3 implementation work
Task: T027 internal/orchestrator/workspace/service.go + internal/config/config.go
Task: T028 internal/agents/openrouter/client.go + internal/orchestrator/workspace/runs.go + internal/orchestrator/workspace/orchestration.go
Task: T029 web/src/shared/lib/workspace-store.ts + web/src/features/canvas/canvasModel.ts
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate live token-usage streaming on the canvas before expanding to replay and fallback cases.

### Incremental Delivery

1. Deliver US1 for live per-agent token usage on active runs.
2. Add US2 for persisted replay of token usage in historical runs.
3. Add US3 for graceful fallback and unavailable-state handling.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer can own provider and persistence plumbing across `internal/agents/openrouter` and `internal/storage/sqlite`.
2. One developer can own protocol, store, and replay wiring across `internal/handlers/ws`, `internal/orchestrator/workspace/history.go`, and `web/src/shared/lib`.
3. One developer can own canvas rendering and accessibility work across `web/src/features/canvas` once the shared state model stabilizes.

## Notes

- [P] tasks touch separate files or proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the limits described in the specification.
- Do not fabricate token estimates when provider usage is absent.
- Do not break older run replay while introducing the new event-table columns.
- Keep model metadata refresh cancellable, bounded by TTL, and isolated from prompt or response logging.