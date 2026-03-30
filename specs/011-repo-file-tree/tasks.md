# Tasks: Repository File Tree Sidebar

**Input**: Design documents from `/specs/011-repo-file-tree/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/workspace-repository-file-tree.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the Relay constitution and feature scope. Include Go unit coverage for repository tree caching, touched-file persistence, deduplication, reconnect restoration, and tool-layer touch emission; Go integration coverage for WebSocket protocol additions and workspace bootstrap hydration; plus Vitest and React Testing Library coverage for right-rail panel switching, repository tree rendering, expansion state, agent filtering, and read-only behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `internal/tools/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/011-repo-file-tree/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the repository tree feature files, sidebar composition points, and documentation scaffolding.

- [X] T001 Create repository tree feature scaffolding in `web/src/features/history/RepositoryFileTreePanel.tsx`, `web/src/features/history/treeModel.ts`, and `web/src/features/history/treeModel.test.ts`
- [X] T002 [P] Create right-rail panel scaffolding for Historical Replay and File Tree in `web/src/features/history/SidebarTabs.tsx`, `web/src/features/history/replay/ReplayDock.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [X] T003 [P] Create backend repository tree scaffolding in `internal/orchestrator/workspace/repository_tree_cache.go`, `internal/orchestrator/workspace/repository_tree_service.go`, and `internal/handlers/ws/workspace_test.go`
- [X] T004 [P] Update feature notes and developer validation commands in `README.md` and `specs/011-repo-file-tree/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Complete the shared persistence, protocol, cache, and store infrastructure required before any user story can ship.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 Add the `touched_files` schema and store models in `internal/storage/sqlite/migrations/0007_repository_file_tree.sql`, `internal/storage/sqlite/models.go`, and `internal/storage/sqlite/store.go`
- [X] T006 [P] Add failing backend coverage for touched-file persistence, deduplication, and reconnect hydration in `internal/storage/sqlite/store_test.go`, `internal/orchestrator/workspace/service_test.go`, and `internal/orchestrator/workspace/tool_executor_test.go`
- [X] T007 [P] Extend workspace protocol types for `repository.tree.request`, `repository.tree.result`, and `file_touched` in `internal/handlers/ws/protocol.go`, `web/src/shared/lib/workspace-protocol.ts`, and `web/src/shared/lib/workspace-store.ts`
- [X] T008 [P] Implement workspace service plumbing for repository tree caching and touched-file snapshot loading in `internal/orchestrator/workspace/service.go`, `internal/orchestrator/workspace/repository_browser.go`, and `internal/orchestrator/workspace/repository_tree_service.go`
- [X] T009 [P] Add handler request and bootstrap plumbing for repository tree hydration in `internal/handlers/ws/workspace.go` and `internal/handlers/ws/workspace_test.go`
- [X] T010 [P] Extend shared frontend store state for replay-dock tabs, tree snapshots, touched-file maps, and agent-filter state in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/useWorkspaceSocket.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Browse the connected repository structure (Priority: P1) 🎯 MVP

**Goal**: Let the developer browse the connected repository in the right-side File Tree panel beside the graph, while reopened saved runs expose top-level Historical Replay and File Tree tabs.

**Independent Test**: Connect Relay to a valid repository, start a run and confirm the right rail shows File Tree directly, then reopen a saved run and confirm the same right rail exposes top-level Historical Replay and File Tree tabs while the File Tree still renders the repo-rooted tree with top-level entries plus one nested level visible by default, expandable folders, `.gitignore`d paths excluded, and no file-opening behavior.

### Tests for User Story 1

- [X] T011 [P] [US1] Add Go unit coverage for repository tree cache construction and path normalization in `internal/orchestrator/workspace/repository_browser_test.go` and `internal/orchestrator/workspace/service_test.go`
- [X] T012 [P] [US1] Add WebSocket integration coverage for repository tree request/result hydration in `tests/integration/repository_file_tree_test.go`
- [X] T013 [P] [US1] Add frontend component and store coverage for right-rail panel switching, initial one-level-deep rendering, folder expansion, read-only clicks, `.gitignore`-aware repository tree loading, and invalid-or-disconnected repository tree requests in `web/src/features/history/treeModel.test.ts`, `web/src/features/workspace-shell/WorkspaceShell.test.tsx`, and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 1

- [X] T014 [P] [US1] Implement in-memory repository tree cache construction from connected repositories in `internal/orchestrator/workspace/repository_tree_cache.go`, `internal/orchestrator/workspace/repository_browser.go`, and `internal/orchestrator/workspace/service.go`
- [X] T015 [P] [US1] Implement `repository.tree.request` and `repository.tree.result` handling in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, and `internal/handlers/ws/workspace_test.go`
- [X] T016 [P] [US1] Implement client tree projection and shallow initial visibility logic in `web/src/features/history/treeModel.ts` and `web/src/shared/lib/workspace-store.ts`
- [X] T017 [P] [US1] Build the right-rail Historical Replay and File Tree panel shell in `web/src/features/history/SidebarTabs.tsx`, `web/src/features/history/replay/ReplayDock.tsx`, `web/src/features/history/RepositoryFileTreePanel.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [X] T018 [US1] Add loading, invalid-repository, out-of-scope, empty, focus, read-only, and non-color-only touched-indicator states for the right-rail File Tree panel in `web/src/features/history/RepositoryFileTreePanel.tsx`, `web/src/features/workspace-shell/WorkspaceShell.tsx`, and `web/src/app/globals.css`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - See which files the run has touched (Priority: P2)

**Goal**: Let the developer see live touched indicators for files that agents read or proposed to change during the current run.

**Independent Test**: Start a run that uses `read_file` and `write_file`, then confirm matching repository tree rows gain touched indicators immediately and remain deduplicated across repeated touches.

### Tests for User Story 2

- [X] T019 [P] [US2] Add Go unit coverage for `touched_files` writes, deduplication, and current-run snapshot queries in `internal/storage/sqlite/store_test.go` and `internal/orchestrator/workspace/tool_executor_test.go`
- [X] T020 [P] [US2] Add tool and executor-path tests for `read_file` and `write_file` metadata propagation plus executor-triggered touch recording in `internal/tools/read_file_test.go`, `internal/tools/write_file_test.go`, `internal/tools/catalog_test.go`, and `internal/orchestrator/workspace/tool_executor_test.go`
- [X] T021 [P] [US2] Add WebSocket integration coverage for live `file_touched` streaming and reconnect restoration in `tests/integration/repository_file_tree_test.go`
- [X] T022 [P] [US2] Add frontend tests for touched indicator rendering, deduped workspace-wide paths, and reconnect hydration in `web/src/features/history/treeModel.test.ts` and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 2

- [X] T023 [P] [US2] Implement touched-file persistence APIs in `internal/storage/sqlite/store.go`, `internal/storage/sqlite/models.go`, and `internal/storage/sqlite/migrations/0007_repository_file_tree.sql`
- [X] T024 [P] [US2] Record `read` and `proposed` touches in `internal/orchestrator/workspace/tool_executor.go` and `internal/storage/sqlite/store.go`, using `internal/tools/read_file.go` and `internal/tools/write_file.go` only for path metadata already returned by tool execution
- [X] T025 [P] [US2] Emit `file_touched` events and bootstrap touched snapshots in `internal/orchestrator/workspace/service.go`, `internal/handlers/ws/protocol.go`, and `internal/handlers/ws/workspace.go`
- [X] T026 [P] [US2] Extend client store reduction for touched-file snapshots, live `file_touched` events, and touched-sync error state preservation in `web/src/shared/lib/workspace-protocol.ts` and `web/src/shared/lib/workspace-store.ts`
- [X] T027 [US2] Render workspace-wide touched indicators, no-touch default state, and live synchronization error feedback without clearing the loaded tree in `web/src/features/history/RepositoryFileTreePanel.tsx` and `web/src/features/history/treeModel.ts`

**Checkpoint**: User Stories 1 and 2 work independently, including live touched-file visibility.

---

## Phase 5: User Story 3 - Filter the tree by selected agent activity (Priority: P3)

**Goal**: Let the developer click an agent node and narrow the repository tree to only the files touched by that agent.

**Independent Test**: Run a multi-agent workflow, click one agent node, and confirm the sidebar narrows to that agent's touched files, updates live when that agent touches more files, and restores the full tree when the selection is cleared.

### Tests for User Story 3

- [X] T028 [P] [US3] Add backend and store coverage for reconnect restoration of touched snapshots, selected-agent filtering, expanded folder state, and filtered-state retention when the active run ends in `internal/orchestrator/workspace/service_test.go`, `internal/handlers/ws/workspace_test.go`, and `web/src/shared/lib/workspace-store.test.ts`
- [X] T029 [P] [US3] Add integration coverage for selected-agent filtering and live filtered updates in `tests/integration/repository_file_tree_test.go`
- [X] T030 [P] [US3] Add frontend tests for selected-agent filtering, no-files-touched empty state, clearing back to workspace-wide view, and missing historical touched paths after repository drift in `web/src/features/history/treeModel.test.ts`, `web/src/features/workspace-shell/WorkspaceShell.test.tsx`, and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 3

- [X] T031 [P] [US3] Expose agent identity and touched snapshot data needed for filtering in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, and `internal/orchestrator/workspace/service.go`
- [X] T032 [P] [US3] Implement client-side selected-agent filtering plus reconnect-safe expanded-folder restoration and end-of-run touched-state retention in `web/src/features/history/treeModel.ts`, `web/src/shared/lib/workspace-store.ts`, and `web/src/features/history/RepositoryFileTreePanel.tsx`
- [X] T033 [P] [US3] Wire canvas selection into repository tree filtering in `web/src/features/canvas/canvasModel.ts`, `web/src/features/canvas/AgentCanvas.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [X] T034 [US3] Render selected-agent badges, filtered empty states, clear-filter affordances, and missing-path historical touched markers or explanations in `web/src/features/history/RepositoryFileTreePanel.tsx`, `web/src/features/history/SidebarTabs.tsx`, and `web/src/app/globals.css`

**Checkpoint**: All three user stories work independently, including per-agent tree narrowing.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, performance, accessibility, and regression validation across the right-rail panel switching and live touched-file flows.

- [X] T035 [P] Update repository-tree protocol and validation documentation in `README.md`, `specs/011-repo-file-tree/contracts/workspace-repository-file-tree.md`, and `specs/011-repo-file-tree/quickstart.md`
- [X] T036 Verify focused backend, integration, and frontend regression suites for repository tree hydration, touched-file streaming, and agent filtering in `internal/orchestrator/workspace/service_test.go`, `internal/storage/sqlite/store_test.go`, `internal/handlers/ws/workspace_test.go`, `tests/integration/repository_file_tree_test.go`, and `web/src/shared/lib/workspace-store.test.ts`
- [X] T037 [P] Validate keyboard access, visible focus, plain-language errors, forced-colors behavior, and 320px reflow for the right-rail Historical Replay and File Tree panels in `web/src/features/history/RepositoryFileTreePanel.tsx`, `web/src/features/history/SidebarTabs.tsx`, `web/src/features/history/replay/ReplayDock.tsx`, and `web/src/app/globals.css`
- [X] T038 [P] Optimize large-repository rendering and verify the one-level-deep initial render strategy plus ignored-path exclusion in `web/src/features/history/treeModel.ts`, `web/src/features/history/RepositoryFileTreePanel.tsx`, `web/src/features/history/treeModel.test.ts`, and `internal/orchestrator/workspace/repository_tree_cache.go`
- [X] T039 Run the full quickstart validation flow and reconcile any follow-up fixes in `specs/011-repo-file-tree/quickstart.md`, `README.md`, and `tests/integration/repository_file_tree_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP repository tree browsing experience.
- **Phase 4: US2** depends on Foundational and on the repository tree surface from US1.
- **Phase 5: US3** depends on Foundational and on the touched-file state introduced in US2.
- **Phase 6: Polish** depends on completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on US1 because touched indicators need the repository tree surface to exist.
- **US3**: Depends on US1 and US2 because selected-agent narrowing builds on the visible tree and current-run touched-file state.

### Within Each User Story

- Write the listed tests first and confirm they fail before implementation.
- Complete backend persistence and protocol work before final frontend rendering for that story.
- Preserve WebSocket-only communication and SQLite-only persistence throughout implementation.
- Keep the repository tree read-only and avoid adding actions that open files, apply diffs, or execute commands.
- Finish loading, empty, and plain-language error states before marking the story complete.

## Parallel Opportunities

- `T002`, `T003`, and `T004` can run in parallel during Setup.
- `T006`, `T007`, `T008`, `T009`, and `T010` can run in parallel once `T005` establishes the shared storage direction.
- `T011`, `T012`, and `T013` can run in parallel for US1 tests.
- `T014`, `T015`, `T016`, and `T017` can run in parallel after US1 tests are in place; `T018` follows once the UI surfaces exist.
- `T019`, `T020`, `T021`, and `T022` can run in parallel for US2 tests.
- `T023`, `T024`, `T025`, and `T026` can run in parallel after US2 tests are in place; `T027` follows once live touched-file state reaches the client.
- `T028`, `T029`, and `T030` can run in parallel for US3 tests.
- `T031`, `T032`, and `T033` can run in parallel after US3 tests are in place; `T034` follows once filtering state is available.
- `T035`, `T037`, and `T038` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T011 internal/orchestrator/workspace/repository_browser_test.go + internal/orchestrator/workspace/service_test.go
Task: T012 tests/integration/repository_file_tree_test.go
Task: T013 web/src/features/history/treeModel.test.ts + web/src/features/workspace-shell/WorkspaceShell.test.tsx + web/src/shared/lib/workspace-store.test.ts

# Parallel US1 implementation work
Task: T014 internal/orchestrator/workspace/repository_tree_cache.go + internal/orchestrator/workspace/repository_browser.go + internal/orchestrator/workspace/service.go
Task: T015 internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go + internal/handlers/ws/workspace_test.go
Task: T016 web/src/features/history/treeModel.ts + web/src/shared/lib/workspace-store.ts
Task: T017 web/src/features/history/SidebarTabs.tsx + web/src/features/history/RepositoryFileTreePanel.tsx + web/src/features/workspace-shell/WorkspaceShell.tsx
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T019 internal/storage/sqlite/store_test.go + internal/orchestrator/workspace/tool_executor_test.go
Task: T020 internal/tools/read_file_test.go + internal/tools/write_file_test.go + internal/tools/catalog_test.go + internal/orchestrator/workspace/tool_executor_test.go
Task: T021 tests/integration/repository_file_tree_test.go
Task: T022 web/src/features/history/treeModel.test.ts + web/src/shared/lib/workspace-store.test.ts

# Parallel US2 implementation work
Task: T023 internal/storage/sqlite/migrations/0007_repository_file_tree.sql + internal/storage/sqlite/store.go + internal/storage/sqlite/models.go
Task: T024 internal/orchestrator/workspace/tool_executor.go + internal/storage/sqlite/store.go
Task: T025 internal/orchestrator/workspace/service.go + internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go
Task: T026 web/src/shared/lib/workspace-protocol.ts + web/src/shared/lib/workspace-store.ts
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T028 internal/orchestrator/workspace/service_test.go + internal/handlers/ws/workspace_test.go
Task: T029 tests/integration/repository_file_tree_test.go
Task: T030 web/src/features/history/treeModel.test.ts + web/src/features/workspace-shell/WorkspaceShell.test.tsx + web/src/shared/lib/workspace-store.test.ts

# Parallel US3 implementation work
Task: T031 internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go + internal/orchestrator/workspace/service.go
Task: T032 web/src/features/history/treeModel.ts + web/src/shared/lib/workspace-store.ts
Task: T033 web/src/features/canvas/canvasModel.ts + web/src/features/canvas/AgentCanvas.tsx + web/src/features/workspace-shell/WorkspaceShell.tsx
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate repository tree browsing and read-only behavior before expanding to touched indicators and agent filtering.

### Incremental Delivery

1. Deliver US1 for right-rail repository browsing.
2. Add US2 for live touched-file visibility from read and proposed-change activity.
3. Add US3 for selected-agent filtering and filtered empty states.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer owns backend SQLite, workspace service, and handler protocol work across `internal/storage/sqlite`, `internal/orchestrator/workspace`, and `internal/handlers/ws`.
2. One developer owns frontend store, right-rail panel switching, and repository tree rendering across `web/src/shared/lib` and `web/src/features/history`.
3. Once touched-file state is stable, another developer can wire canvas selection into filtering in parallel.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the limits described in the specification.
- Do not add an HTTP tree endpoint; keep tree hydration and live updates on the workspace WebSocket protocol.
- Do not bypass the Relay architecture boundary of handlers to orchestrator to agents to tools to storage.
- Do not record touches at approval time; `proposed` touches must be captured when the proposal is created.
- The repository tree must remain read-only throughout the implementation.