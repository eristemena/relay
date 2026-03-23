# Tasks: Local Relay Workspace

**Input**: Design documents from `/specs/003-local-relay-workspace/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include Go unit and integration coverage plus frontend component coverage for workspace shell behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/app/`, `internal/browser/`, `internal/config/`, `internal/frontend/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/003-local-relay-workspace/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish the repo structure, build tooling, and baseline UI/runtime configuration needed for all stories.

- [x] T001 Create the initial Relay project structure and module entrypoints in `go.mod`, `cmd/relay/main.go`, `web/package.json`, and `web/tsconfig.json`
- [x] T002 Configure frontend export and local dev tooling in `web/next.config.ts`, `web/postcss.config.mjs`, `web/tailwind.config.ts`, and `Makefile`
- [x] T003 [P] Add the shared dark-mode token, font, and global app shell setup in `web/src/app/layout.tsx` and `web/src/app/globals.css`
- [x] T004 [P] Document the approved stack, local data locations, and build commands in `README.md` and `specs/003-local-relay-workspace/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the runtime, persistence, protocol, and transport foundations that block all user stories.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Implement Relay home-directory bootstrap and field-by-field TOML config loading in `internal/config/config.go`
- [x] T006 [P] Define the SQLite schema and generated query inputs for sessions and workspace snapshots in `internal/storage/sqlite/migrations/0001_initial.sql`, `internal/storage/sqlite/queries/sessions.sql`, and `sqlc.yaml`
- [x] T007 [P] Implement the SQLite store and storage models in `internal/storage/sqlite/store.go` and `internal/storage/sqlite/models.go`
- [x] T008 [P] Define shared WebSocket message envelopes and DTOs in `internal/handlers/ws/protocol.go` and `web/src/shared/lib/workspace-protocol.ts`
- [x] T009 Implement the workspace orchestrator service for bootstrap, session switching, and preference saves in `internal/orchestrator/workspace/service.go`
- [x] T010 Implement preferred-port fallback, listener startup, and browser-launch abstraction in `internal/app/server.go` and `internal/browser/open.go`
- [x] T011 Implement Relay-owned HTTP and WebSocket routing plus frontend proxy and static serving in `internal/handlers/http/health.go`, `internal/handlers/ws/workspace.go`, `internal/frontend/proxy.go`, and `internal/frontend/static.go`
- [x] T012 [P] Create shared backend and frontend test helpers for socket bootstrap and persisted state setup in `tests/integration/testutil_test.go` and `web/src/shared/lib/test-helpers.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Launch the workspace from one command (Priority: P1) 🎯 MVP

**Goal**: Let a developer run `relay serve`, have Relay start on an available local port, open the browser when possible, and render the default workspace shell.

**Independent Test**: Run `relay serve` on a clean machine state, verify the browser opens to the actual assigned Relay address, and confirm the top bar, session sidebar, and central canvas render with required loading and error handling.

### Tests for User Story 1

- [x] T013 [P] [US1] Add startup and preferred-port fallback integration coverage in `tests/integration/serve_startup_test.go`
- [x] T014 [P] [US1] Add WebSocket bootstrap and reconnect integration coverage in `tests/integration/websocket_reconnect_test.go`
- [x] T015 [P] [US1] Add component coverage for initial loading, ready, and recoverable error states in `web/src/features/workspace-shell/WorkspaceShell.test.tsx`

### Implementation for User Story 1

- [x] T016 [P] [US1] Implement the Cobra `serve` command and startup logging flow in `cmd/relay/main.go` and `internal/app/server.go`
- [x] T017 [P] [US1] Build the workspace shell layout with the top navigation, session sidebar frame, and canvas frame in `web/src/app/page.tsx` and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [x] T018 [P] [US1] Implement the frontend WebSocket bootstrap client and authoritative workspace state store in `web/src/shared/lib/useWorkspaceSocket.ts` and `web/src/shared/lib/workspace-store.ts`
- [x] T019 [P] [US1] Implement the default canvas shell and first-run empty activity state in `web/src/features/canvas/WorkspaceCanvas.tsx` and `web/src/features/canvas/CanvasEmptyState.tsx`
- [x] T020 [US1] Surface startup, browser-launch fallback, and frontend-unavailable recovery messaging in `internal/app/server.go`, `web/src/features/workspace-shell/WorkspaceStatusBanner.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [x] T021 [US1] Wire bootstrap responses and status events from the Go handlers into the initial shell rendering flow in `internal/handlers/ws/workspace.go` and `internal/orchestrator/workspace/service.go`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Resume past sessions (Priority: P2)

**Goal**: Persist session metadata locally, list saved sessions in the sidebar, and reopen a selected session after restart.

**Independent Test**: Create one or more sessions, restart Relay, verify the sessions appear in the sidebar, and confirm selecting one restores it as the active workspace view.

### Tests for User Story 2

- [x] T022 [P] [US2] Add storage and orchestrator unit coverage for session persistence and reopen behavior in `internal/storage/sqlite/store_test.go` and `internal/orchestrator/workspace/service_test.go`
- [x] T023 [P] [US2] Add integration coverage for session listing and `session.open` events in `tests/integration/workspace_sessions_test.go`
- [x] T024 [P] [US2] Add component coverage for session history, selection, and empty-history states in `web/src/features/history/SessionSidebar.test.tsx`

### Implementation for User Story 2

- [x] T025 [P] [US2] Implement session list, upsert, and active-session persistence queries in `internal/storage/sqlite/queries/sessions.sql` and `internal/storage/sqlite/store.go`
- [x] T026 [P] [US2] Implement session-history bootstrap and `session.open` orchestration in `internal/orchestrator/workspace/service.go` and `internal/handlers/ws/workspace.go`
- [x] T027 [P] [US2] Build the sidebar history list, session list items, and explicit empty state UI in `web/src/features/history/SessionSidebar.tsx` and `web/src/features/history/SessionListItem.tsx`
- [x] T028 [US2] Restore the saved active session and replace local history state on reconnect in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/useWorkspaceSocket.ts`

**Checkpoint**: User Stories 1 and 2 work independently, including restart and reopen flows.

---

## Phase 5: User Story 3 - Start a new session and keep preferences (Priority: P3)

**Goal**: Let the developer create a new session from the workspace and persist supported local preferences, including preferred port and stored credentials, across restarts.

**Independent Test**: Create a new session, update supported preferences, restart Relay, and confirm both the session and persisted preferences remain available with safe fallback behavior for invalid values.

### Tests for User Story 3

- [x] T029 [P] [US3] Add unit coverage for config validation, field-level fallback, and secret redaction in `internal/config/config_test.go` and `internal/orchestrator/workspace/preferences_test.go`
- [x] T030 [P] [US3] Add integration coverage for `session.create` and `preferences.save` flows in `tests/integration/preferences_save_test.go`
- [x] T031 [P] [US3] Add component coverage for new-session actions and preference saving states in `web/src/features/history/NewSessionButton.test.tsx` and `web/src/features/preferences/PreferencesPanel.test.tsx`

### Implementation for User Story 3

- [x] T032 [P] [US3] Implement new-session creation, naming, and active-session switching in `internal/orchestrator/workspace/service.go` and `internal/handlers/ws/workspace.go`
- [x] T033 [P] [US3] Implement persisted preference updates, invalid-value fallback, and secret-safe serialization in `internal/config/config.go` and `internal/orchestrator/workspace/service.go`
- [x] T034 [P] [US3] Build the new-session trigger and preferences editor UI in `web/src/features/history/NewSessionButton.tsx` and `web/src/features/preferences/PreferencesPanel.tsx`
- [x] T035 [US3] Surface saving, invalid-preference, and credential-presence status in `web/src/features/preferences/PreferencesStatus.tsx` and `web/src/shared/lib/workspace-store.ts`

**Checkpoint**: All three user stories work independently, including restart-safe preference persistence.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize build, documentation, performance, and regression coverage across all stories.

- [x] T036 [P] Finalize the embedded frontend build and copy pipeline in `Makefile`, `web/package.json`, and `internal/frontend/embed/.gitkeep`
- [x] T037 [P] Update release, local run, and troubleshooting documentation in `README.md` and `specs/003-local-relay-workspace/quickstart.md`
- [x] T038 [P] Add cross-story regression coverage for launch, resume, and preferences behavior in `tests/integration/serve_startup_test.go`, `tests/integration/workspace_sessions_test.go`, and `tests/integration/preferences_save_test.go`
- [x] T039 Verify human-readable errors, loading states, and offline-safe behavior across shared runtime paths in `internal/app/server.go`, `internal/handlers/ws/workspace.go`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [ ] T040 Measure and tune startup-path performance and WebSocket dispatch latency in `internal/app/server.go` and `internal/orchestrator/workspace/service.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP.
- **Phase 4: US2** depends on Foundational and can proceed after or alongside US1 if staffed, but the recommended path is after the MVP.
- **Phase 5: US3** depends on Foundational and can proceed after or alongside US2 if staffed.
- **Phase 6: Polish** depends on the completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on Foundational storage and WebSocket infrastructure but remains independently testable from US1.
- **US3**: Depends on Foundational config and WebSocket infrastructure but remains independently testable from US1 and US2.

### Within Each User Story

- Write tests first and confirm they fail before implementation.
- Finish persistence and protocol work before UI integration for that story.
- Complete loading, empty, and recoverable error states before marking the story done.

## Parallel Opportunities

- `T003` and `T004` can run in parallel after `T001` and `T002`.
- `T006`, `T007`, `T008`, and `T012` can run in parallel within Foundational work.
- `T013`, `T014`, and `T015` can run in parallel for US1 tests.
- `T016`, `T017`, `T018`, and `T019` can run in parallel after US1 tests are in place.
- `T022`, `T023`, and `T024` can run in parallel for US2 tests.
- `T025`, `T026`, and `T027` can run in parallel after US2 tests are in place.
- `T029`, `T030`, and `T031` can run in parallel for US3 tests.
- `T032`, `T033`, and `T034` can run in parallel after US3 tests are in place.
- `T036`, `T037`, and `T038` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T013 tests/integration/serve_startup_test.go
Task: T014 tests/integration/websocket_reconnect_test.go
Task: T015 web/src/features/workspace-shell/WorkspaceShell.test.tsx

# Parallel US1 implementation work
Task: T016 cmd/relay/main.go + internal/app/server.go
Task: T017 web/src/app/page.tsx + web/src/features/workspace-shell/WorkspaceShell.tsx
Task: T018 web/src/shared/lib/useWorkspaceSocket.ts + web/src/shared/lib/workspace-store.ts
Task: T019 web/src/features/canvas/WorkspaceCanvas.tsx + web/src/features/canvas/CanvasEmptyState.tsx
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T022 internal/storage/sqlite/store_test.go + internal/orchestrator/workspace/service_test.go
Task: T023 tests/integration/workspace_sessions_test.go
Task: T024 web/src/features/history/SessionSidebar.test.tsx

# Parallel US2 implementation work
Task: T025 internal/storage/sqlite/queries/sessions.sql + internal/storage/sqlite/store.go
Task: T026 internal/orchestrator/workspace/service.go + internal/handlers/ws/workspace.go
Task: T027 web/src/features/history/SessionSidebar.tsx + web/src/features/history/SessionListItem.tsx
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T029 internal/config/config_test.go + internal/orchestrator/workspace/preferences_test.go
Task: T030 tests/integration/preferences_save_test.go
Task: T031 web/src/features/history/NewSessionButton.test.tsx + web/src/features/preferences/PreferencesPanel.test.tsx

# Parallel US3 implementation work
Task: T032 internal/orchestrator/workspace/service.go + internal/handlers/ws/workspace.go
Task: T033 internal/config/config.go + internal/orchestrator/workspace/service.go
Task: T034 web/src/features/history/NewSessionButton.tsx + web/src/features/preferences/PreferencesPanel.tsx
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate the `relay serve` flow end to end before expanding scope.

### Incremental Delivery

1. Deliver US1 for the single-command local workspace promise.
2. Add US2 for restart-safe session recovery.
3. Add US3 for new-session creation and persistent preferences.
4. Finish with Polish tasks before packaging the first downloadable binary.

### Parallel Team Strategy

1. One developer owns runtime and persistence foundations in Phase 2.
2. One developer can own frontend shell and socket state work starting in US1.
3. After Foundational work, separate developers can take US2 and US3 in parallel with minimal file overlap.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story is scoped to remain independently testable.
- Do not bypass the Relay architecture boundary of handlers to orchestrator to storage.
- Do not send credentials or raw secrets to the frontend in any task.