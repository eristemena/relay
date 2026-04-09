# Tasks: Multi-Project Support

**Input**: Design documents from `/specs/012-multi-project-support/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/workspace-project-context.md, quickstart.md

**Tests**: Tests are required for this feature because it changes WebSocket protocol payloads, SQLite session persistence, workspace bootstrap behavior, CLI root selection, and frontend workspace state reset behavior.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated independently.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the project-root scaffolding, test surfaces, and protocol placeholders required before deeper implementation.

- [X] T001 Add the `sessions.project_root` schema change in `internal/storage/sqlite/migrations/0008_multi_project_support.sql` for clean multi-project database initialization
- [X] T002 [P] Add project-aware session query placeholders in `internal/storage/sqlite/queries/sessions.sql` and `internal/storage/sqlite/models.go`
- [X] T003 [P] Add project-context protocol placeholders in `internal/handlers/ws/protocol.go` and `web/src/shared/lib/workspace-protocol.ts`
- [X] T004 [P] Add project-switcher component and store test scaffolding in `web/src/features/workspace-shell/ProjectSwitcher.tsx`, `web/src/features/workspace-shell/ProjectSwitcher.test.tsx`, and `web/src/shared/lib/test-helpers.tsx`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement the root identity, session lookup, protocol, and store-reset infrastructure that every user story depends on.

**⚠️ CRITICAL**: No user story work should start until this phase is complete.

- [X] T005 Implement canonical project-root resolution and invalid-root errors in `internal/orchestrator/workspace/project_root.go` and `internal/orchestrator/workspace/project_root_test.go`
- [X] T006 Implement project-root persistence, lookup, and known-project listing in `internal/storage/sqlite/queries/sessions.sql`, `internal/storage/sqlite/models.go`, `internal/storage/sqlite/store.go`, and `internal/storage/sqlite/store_test.go`
- [X] T007 [P] Implement project bootstrap and switch protocol types without exposing internal persistence identifiers in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, `internal/handlers/ws/workspace_test.go`, and `web/src/shared/lib/workspace-protocol.ts`
- [X] T008 Implement project-aware bootstrap selection and single-active-run switch blocking in `internal/orchestrator/workspace/service.go` and `internal/orchestrator/workspace/service_test.go`
- [X] T009 [P] Implement project-context reset semantics for run, replay, approval, and repository state in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/workspace-store.test.ts`
- [ ] T010 [P] Implement optional all-project history query plumbing with `project_root` and optional `project_label` result fields in `internal/storage/sqlite/models.go`, `internal/storage/sqlite/store.go`, `internal/storage/sqlite/store_test.go`, and `internal/orchestrator/workspace/history.go`
- [ ] T011 [P] Add foundational protocol integration coverage in `tests/integration/project_context_test.go`

**Checkpoint**: Project-root identity, session lookup, protocol contracts, and store reset behavior are ready for story work.

---

## Phase 3: User Story 1 - Work in the right project without manual session setup (Priority: P1) 🎯 MVP

**Goal**: Start Relay against a root, automatically reuse or create the correct project session, and show the active project root in the workspace header.

**Independent Test**: Start Relay from a directory with and without `--root`, confirm the active project root is shown, and verify that first-time roots auto-create a session while returning roots reuse the existing one.

### Tests for User Story 1

- [X] T012 [P] [US1] Add startup root-resolution tests in `cmd/relay/main_test.go` and `internal/app/server_test.go`
- [X] T013 [P] [US1] Add automatic project-context bootstrap tests in `internal/orchestrator/workspace/service_test.go`
- [X] T014 [P] [US1] Add active-project header rendering tests in `web/src/features/workspace-shell/WorkspaceShell.test.tsx` and `web/src/features/workspace-shell/ProjectSwitcher.test.tsx`
- [X] T037 [P] [US1] Add regression tests in `web/src/features/history/SessionSidebar.test.tsx`, `web/src/features/history/NewSessionButton.test.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.test.tsx` proving developers are not asked to create or open sessions manually for project selection

### Implementation for User Story 1

- [X] T015 [US1] Implement `relay serve` root selection order and explicit `--root` handling in `cmd/relay/main.go` and `internal/app/server.go`
- [X] T016 [US1] Implement automatic persisted project-context reuse or creation during bootstrap in `internal/orchestrator/workspace/service.go` and `internal/storage/sqlite/store.go`
- [X] T017 [US1] Expose `active_project_root` and known-project snapshot data in `internal/orchestrator/workspace/service.go`, `internal/handlers/ws/workspace.go`, and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T018 [US1] Render the active project root plus single-project loading, empty, and error states in `web/src/features/workspace-shell/WorkspaceShell.tsx` and `web/src/features/workspace-shell/ProjectSwitcher.tsx`
- [X] T019 [US1] Preserve plain-language startup failures and active-project repository boundaries in `internal/app/server.go` and `internal/orchestrator/workspace/service.go`
- [X] T038 [US1] Remove or repurpose manual session creation and session-opening UI in `web/src/features/history/SessionSidebar.tsx`, `web/src/features/history/NewSessionButton.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx` so project-root selection replaces manual session management for developers

**Checkpoint**: Relay starts in the correct project context with no manual session creation and a visible active-project indicator.

---

## Phase 4: User Story 2 - Switch between known project roots without restarting Relay (Priority: P2)

**Goal**: Let the developer switch known project roots from the header dropdown and fully rehydrate the selected project's canvas, history, and file tree without stale artifacts.

**Independent Test**: Connect two known roots, switch between them from the header dropdown, and confirm the canvas, history, and repository tree all reflect only the selected project's context while blocked switches show a clear message.

### Tests for User Story 2

- [X] T020 [P] [US2] Add project-switch and unavailable-root tests in `internal/orchestrator/workspace/service_test.go`
- [X] T021 [P] [US2] Add WebSocket switch request coverage in `internal/handlers/ws/workspace_test.go` and `tests/integration/project_context_test.go`
- [X] T022 [P] [US2] Add ghost-node regression and switch reset tests in `web/src/features/workspace-shell/WorkspaceShell.test.tsx` and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 2

- [X] T023 [US2] Implement `project.switch.request` handling and active-run blocking in `internal/orchestrator/workspace/service.go` and `internal/handlers/ws/workspace.go`
- [X] T024 [US2] Implement known-project dropdown behavior and switch-triggered snapshot reloads in `web/src/features/workspace-shell/ProjectSwitcher.tsx`, `web/src/features/workspace-shell/WorkspaceShell.tsx`, and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T025 [US2] Clear stale run, replay, approval, repository tree, and orchestration document state on project changes in `web/src/shared/lib/workspace-store.ts` and `web/src/features/canvas/WorkspaceCanvas.tsx`
- [X] T026 [US2] Restore project-scoped loading, unavailable-project, and blocked-switch states in `web/src/features/workspace-shell/WorkspaceShell.tsx` and `web/src/features/workspace-shell/ProjectSwitcher.tsx`

**Checkpoint**: Developers can switch known projects without restarting Relay, and no stale canvas or history state leaks across roots.

---

## Phase 5: User Story 3 - Review history in the correct project scope (Priority: P3)

**Goal**: Show run history for the active project by default and offer an opt-in all-project mode that broadens the query without changing the active project.

**Independent Test**: Create runs under multiple project roots, confirm the History tab defaults to the active project's runs, then enable the all-project mode and verify the merged list includes clear project identity labels without switching the active root.

### Tests for User Story 3

- [X] T027 [P] [US3] Add active-project versus all-project history query tests in `internal/storage/sqlite/store_test.go` and `internal/orchestrator/workspace/service_test.go`
- [X] T028 [P] [US3] Add history protocol coverage for the `all_projects` toggle in `internal/handlers/ws/workspace_test.go` and `tests/integration/project_history_test.go`
- [X] T029 [P] [US3] Add history toggle and project-label rendering tests in `web/src/features/history/RunHistoryPanel.test.tsx` and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`
- [X] T039 [P] [US3] Add payload-shape tests in `internal/handlers/ws/workspace_test.go` and `web/src/shared/lib/workspace-store.test.ts` proving all-project history results include per-run `project_root` and optional `project_label`

### Implementation for User Story 3

- [X] T030 [US3] Implement active-project and all-project history filtering in `internal/storage/sqlite/store.go`, `internal/orchestrator/workspace/history.go`, and `internal/handlers/ws/workspace.go`
- [X] T031 [US3] Extend history query payloads and socket calls with `all_projects` support in `web/src/shared/lib/workspace-protocol.ts` and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T032 [US3] Render the all-project toggle, project labels, and project-scoped empty/error states in `web/src/features/history/RunHistoryPanel.tsx` and `web/src/features/history/RunHistoryListItem.tsx`
- [X] T033 [US3] Preserve active-project context while toggling history scope in `web/src/shared/lib/workspace-store.ts` and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [X] T040 [US3] Extend run-history summaries and `run.history.result` payloads with per-run `project_root` and optional `project_label` in `internal/orchestrator/workspace/service.go`, `internal/handlers/ws/workspace.go`, and `web/src/shared/lib/workspace-protocol.ts`

**Checkpoint**: History defaults to the active project, and the all-project view is an explicit review mode rather than a project switch.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, validation, and regression-proofing across all stories.

- [X] T034 [P] Update operator-facing documentation for startup root selection and multi-project behavior in `README.md`
- [X] T035 Run the focused Go, integration, and Vitest commands from `specs/012-multi-project-support/quickstart.md` and fix any feature regressions they expose
- [X] T036 [P] Verify core coverage, the SC-002 2-second switch target, and final performance/security regressions for `internal/orchestrator/workspace`, `internal/storage/sqlite`, and `web/src/shared/lib/workspace-store.ts`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Phase 1 and blocks all story work.
- **Phase 3: User Story 1** depends on Phase 2.
- **Phase 4: User Story 2** depends on Phase 2 and is safest after User Story 1 establishes startup project context.
- **Phase 5: User Story 3** depends on Phase 2 and on the project-context data from User Story 1; it can proceed alongside late User Story 2 work once switching payloads exist.
- **Phase 6: Polish** depends on the user stories the team intends to ship.

### User Story Dependencies

- **User Story 1 (P1)** is the MVP and has no dependency on other stories once Foundational work is done.
- **User Story 2 (P2)** depends on project-context identity and bootstrap metadata from User Story 1.
- **User Story 3 (P3)** depends on project-context identity from User Story 1 and reuses the switching/bootstrap payloads introduced for User Story 2.

### Within Each User Story

- Story tests should be written before implementation and should fail before code changes begin.
- Backend persistence and orchestrator logic should land before protocol handlers.
- Protocol handlers and socket helpers should land before UI wiring.
- Loading, empty, and error states must be in place before a story is considered complete.

### Parallel Opportunities

- `T002`, `T003`, and `T004` can run in parallel in Phase 1.
- `T007`, `T009`, `T010`, and `T011` can run in parallel once `T005` and `T006` establish canonical roots and project-aware persistence.
- In User Story 1, `T012`, `T013`, `T014`, and `T037` can run in parallel.
- In User Story 2, `T020`, `T021`, and `T022` can run in parallel.
- In User Story 3, `T027`, `T028`, `T029`, and `T039` can run in parallel.
- `T034` and `T036` can run in parallel during polish while `T035` executes validation.

---

## Parallel Example: User Story 1

```bash
# Launch the story-specific tests together:
Task: "Add startup root-resolution tests in cmd/relay/main_test.go and internal/app/server_test.go"
Task: "Add automatic project-context bootstrap tests in internal/orchestrator/workspace/service_test.go"
Task: "Add active-project header rendering tests in web/src/features/workspace-shell/WorkspaceShell.test.tsx and web/src/features/workspace-shell/ProjectSwitcher.test.tsx"
```

## Parallel Example: User Story 2

```bash
# Launch the switch-specific validation tasks together:
Task: "Add project-switch and unavailable-root tests in internal/orchestrator/workspace/service_test.go"
Task: "Add WebSocket switch request coverage in internal/handlers/ws/workspace_test.go and tests/integration/project_context_test.go"
Task: "Add ghost-node regression and switch reset tests in web/src/features/workspace-shell/WorkspaceShell.test.tsx and web/src/shared/lib/workspace-store.test.ts"
```

## Parallel Example: User Story 3

```bash
# Launch the history-scope test tasks together:
Task: "Add active-project versus all-project history query tests in internal/storage/sqlite/store_test.go and internal/orchestrator/workspace/service_test.go"
Task: "Add history protocol coverage for the all_projects toggle in internal/handlers/ws/workspace_test.go and tests/integration/project_history_test.go"
Task: "Add history toggle and project-label rendering tests in web/src/features/history/RunHistoryPanel.test.tsx and web/src/features/workspace-shell/WorkspaceShell.test.tsx"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate startup root selection, automatic project-context creation, and active-project visibility before moving on.

### Incremental Delivery

1. Ship User Story 1 to establish correct project identity and automatic project-context behavior.
2. Add User Story 2 to make project switching safe and restart-free.
3. Add User Story 3 to broaden history browsing without changing the active project.
4. Finish with polish, validation, and documentation.

### Suggested MVP Scope

- **MVP**: Phase 1, Phase 2, and Phase 3 only, including `T037` and `T038`.
- This delivers correct startup scoping, automatic project-context creation, manual-session UI retirement, and active-project visibility without requiring multi-project history aggregation yet.

---

## Notes

- Every task follows the required checklist format with task ID, optional parallel marker, required story marker for story phases, and explicit file paths.
- No task bypasses the Relay layer order or introduces direct frontend database access.
- Project switching is intentionally implemented through the existing workspace WebSocket path rather than a new HTTP endpoint.