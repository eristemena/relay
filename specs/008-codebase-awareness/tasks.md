# Tasks: Codebase Awareness

**Input**: Design documents from `/specs/008-codebase-awareness/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/workspace-codebase-awareness.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include Go unit coverage for repo-aware tools, approval persistence, repository validation, background repository-analysis workers, and run-command sandboxing; Go integration coverage for WebSocket protocol changes, reconnect and pending-approval restoration, and repository-boundary enforcement; plus Vitest and React Testing Library coverage for repository connection UX, approval review surfaces, repository-context state handling, and agent-node file activity indicators.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/config/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `internal/tools/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/008-codebase-awareness/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare dependency, file, and documentation scaffolding for repository-aware Relay work.

- [X] T001 Add the approved repository-awareness dependencies to `go.mod` and `web/package.json`
- [X] T002 [P] Scaffold the feature folders and entry files in `web/src/features/approvals/ApprovalReviewPanel.tsx`, `web/src/features/approvals/ApprovalReviewPanel.test.tsx`, and `web/src/features/codebase/graphModel.ts`
- [X] T003 [P] Scaffold the backend repository worker entry points in `internal/orchestrator/workspace/repository_graph.go` and `internal/orchestrator/workspace/repository_browser.go`
- [X] T004 [P] Update the repository-awareness setup notes and dependency documentation in `README.md` and `specs/008-codebase-awareness/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the shared persistence, protocol, repo validation, and store infrastructure required before any user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 Implement the persisted approval-request schema and queries in `internal/storage/sqlite/migrations/0004_codebase_awareness.sql`, `internal/storage/sqlite/queries/approval_requests.sql`, `internal/storage/sqlite/models.go`, and `internal/storage/sqlite/store.go`
- [X] T006 [P] Extend workspace service interfaces and approval coordination to persist and reload approval requests in `internal/orchestrator/workspace/service.go`, `internal/orchestrator/workspace/runs.go`, and `internal/orchestrator/workspace/tool_executor.go`
- [X] T007 [P] Extend the WebSocket protocol and handler plumbing for repository browse, connected repository, persisted approvals, approval state changes, and graph status events in `internal/handlers/ws/protocol.go`, `internal/handlers/ws/workspace.go`, and `web/src/shared/lib/workspace-protocol.ts`
- [X] T008 [P] Harden repository-root validation and Git-repository checks in `internal/tools/path_guard.go`, `internal/config/config.go`, and `internal/orchestrator/workspace/service.go`
- [X] T009 [P] Expand the tool catalog and executor contracts for `list_files`, `git_log`, `git_diff`, and richer repo-aware previews in `internal/tools/catalog.go`, `internal/orchestrator/workspace/tool_executor.go`, and `internal/agents/registry.go`
- [X] T010 [P] Extend shared workspace-store state for connected repository, pending approval restoration, approval lifecycle events, graph status, and agent file activity in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/useWorkspaceSocket.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Connect a local repository safely (Priority: P1) 🎯 MVP

**Goal**: Let the developer connect one valid local Git repository through startup configuration or a folder picker and give agents bounded read-only repository awareness.

**Independent Test**: Start Relay with `--project-root` or choose a repository from the UI, then confirm repository-aware read-only tools work only inside that repository and invalid selections are rejected with plain-language feedback.

### Tests for User Story 1

- [X] T011 [P] [US1] Add table-driven backend tests for Git repository validation, project-root preference handling, and repo-browse responses in `internal/config/config_test.go`, `internal/orchestrator/workspace/preferences_test.go`, and `internal/handlers/ws/workspace_test.go`
- [X] T012 [P] [US1] Add unit coverage for repo-aware read-only tools and boundary rejection in `internal/tools/catalog_test.go`, `internal/tools/read_file_test.go`, `internal/tools/search_codebase_test.go`, `internal/tools/list_files_test.go`, `internal/tools/git_log_test.go`, and `internal/tools/git_diff_test.go`
- [X] T013 [P] [US1] Add WebSocket integration coverage for repository connection, browse, and bootstrap repository state in `tests/integration/codebase_awareness_test.go` and `tests/integration/workspace_sessions_test.go`
- [X] T014 [P] [US1] Add frontend component and store coverage for project-root connection, folder-picker flow, and repository status states in `web/src/features/preferences/PreferencesPanel.test.tsx`, `web/src/features/workspace-shell/WorkspaceShell.test.tsx`, and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 1

- [X] T015 [P] [US1] Add the startup project-root flag and propagate it into saved config and workspace bootstrap in `cmd/relay/main.go`, `internal/config/config.go`, and `internal/app/server.go`
- [X] T016 [P] [US1] Implement server-backed folder browsing and Git repository validation in `internal/orchestrator/workspace/repository_browser.go`, `internal/handlers/ws/protocol.go`, and `internal/handlers/ws/workspace.go`
- [X] T017 [P] [US1] Implement the repo-aware read-only tools in `internal/tools/list_files.go`, `internal/tools/git_log.go`, `internal/tools/git_diff.go`, `internal/tools/read_file.go`, and `internal/tools/search_codebase.go`
- [X] T018 [US1] Wire repository connection, folder-picker actions, and visible repository status or empty states into `web/src/features/preferences/PreferencesPanel.tsx`, `web/src/features/workspace-shell/WorkspaceStatusBanner.tsx`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`
- [X] T019 [US1] Surface connected-repository state and read-only activity initialization in `web/src/shared/lib/workspace-store.ts`, `web/src/shared/lib/workspace-protocol.ts`, and `web/src/features/workspace-shell/WorkspaceShell.tsx`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Review proposed writes and commands before execution (Priority: P2)

**Goal**: Let the developer review persisted file-diff and command proposals, approve or reject them explicitly, and guarantee that no file write or command execution occurs without server-side approval.

**Independent Test**: Ask an agent to propose a file change and a command, confirm both stay pending across reconnect, reject one without side effects, and approve the other only after the backend revalidates the request and repository state.

### Tests for User Story 2

- [X] T020 [P] [US2] Add table-driven backend tests for the approval-request state machine, stale-request expiration, and post-approval revalidation in `internal/orchestrator/workspace/service_test.go` and `internal/orchestrator/workspace/tool_executor_test.go`
- [X] T021 [P] [US2] Add unit coverage for diff-first writes, command sandboxing, and approval-gated tool happy and error paths in `internal/tools/write_file_test.go`, `internal/tools/run_command_test.go`, and `internal/tools/catalog_test.go`
- [X] T022 [P] [US2] Add integration coverage for persisted pending approvals, reconnect restoration, reject paths, and stale-approval blocking in `tests/integration/tool_call_ordering_test.go` and `tests/integration/codebase_awareness_test.go`
- [X] T023 [P] [US2] Add frontend component coverage for Monaco diff review, command review, approval resolution, and pending-approval restore in `web/src/features/approvals/ApprovalReviewPanel.test.tsx` and `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 2

- [X] T024 [P] [US2] Implement persisted approval-request lifecycle storage and bootstrap hydration in `internal/storage/sqlite/store.go`, `internal/orchestrator/workspace/service.go`, and `internal/orchestrator/workspace/runs.go`
- [X] T025 [P] [US2] Implement diff-first `write_file` previews with base-content hashing and stale-file protection in `internal/tools/write_file.go`, `internal/orchestrator/workspace/tool_executor.go`, and `internal/handlers/ws/protocol.go`
- [X] T026 [P] [US2] Harden `run_command` approval gating and per-execution repo-root revalidation in `internal/tools/run_command.go`, `internal/tools/path_guard.go`, and `internal/orchestrator/workspace/tool_executor.go`
- [X] T027 [P] [US2] Wire approval-request and approval-state-changed events through the frontend store in `web/src/shared/lib/workspace-protocol.ts`, `web/src/shared/lib/workspace-store.ts`, and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T028 [US2] Build the approval review experience with side-by-side Monaco diff mode, command previews, visible loading states, and plain-language outcomes in `web/src/features/approvals/ApprovalReviewPanel.tsx`, `web/src/features/workspace-shell/WorkspaceShell.tsx`, and `web/src/app/globals.css`

**Checkpoint**: User Stories 1 and 2 work independently, including persisted approval review and server-enforced mutation gating.

---

## Phase 5: User Story 3 - Understand repository context and agent activity (Priority: P3)

**Goal**: Let the developer benefit from background-derived repository context and see, on each agent node, which files were read or proposed for change and what approval state those proposals reached.

**Independent Test**: Run an agent flow on a connected repository, confirm repository context builds asynchronously without blocking the workspace, and verify the canvas or detail surfaces show each agent’s read paths, proposed change paths, and approval outcomes.

### Tests for User Story 3

- [X] T029 [P] [US3] Add backend tests for repository-analysis worker scheduling, cache invalidation, and graceful degradation in `internal/orchestrator/workspace/service_test.go` and `internal/orchestrator/workspace/repository_graph_test.go`
- [x] T030 [P] [US3] Add integration coverage for repository-context events and agent file-activity replay in `tests/integration/codebase_awareness_test.go` and `tests/integration/workspace_sessions_test.go`
- [x] T031 [P] [US3] Add frontend coverage for repository-context store states and agent-node file activity display in `web/src/shared/lib/workspace-store.test.ts`, `web/src/features/canvas/AgentCanvasNode.test.tsx`, and `web/src/features/canvas/canvasModel.test.ts`

### Implementation for User Story 3

- [X] T032 [P] [US3] Implement cancellable background repository-context construction and in-memory cache invalidation in `internal/orchestrator/workspace/repository_graph.go`, `internal/orchestrator/workspace/service.go`, and `internal/handlers/ws/workspace.go`
- [X] T033 [P] [US3] Emit and hydrate repository-context status or ready payloads plus repo-aware tool previews needed for activity derivation in `internal/handlers/ws/protocol.go`, `web/src/shared/lib/workspace-protocol.ts`, and `web/src/shared/lib/workspace-store.ts`
- [X] T034 [P] [US3] Maintain the frontend repository-context model used by store hydration and agent activity derivation in `web/src/features/codebase/graphModel.ts` and `web/src/shared/lib/workspace-store.ts`
- [x] T035 [P] [US3] Derive per-agent read paths, proposed change paths, and approval outcomes from tool and approval events in `web/src/features/canvas/canvasModel.ts`, `web/src/features/canvas/AgentCanvasNode.tsx`, and `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [x] T036 [US3] Integrate agent activity views into the workspace shell and canvas layout in `web/src/features/workspace-shell/WorkspaceShell.tsx`, `web/src/features/canvas/WorkspaceCanvas.tsx`, and `web/src/features/canvas/AgentNodeDetailPanel.tsx`

**Checkpoint**: All three user stories work independently, including asynchronous repository-context preparation and per-agent file-awareness tracking.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, performance, coverage, and end-to-end validation across repository awareness, approvals, and agent activity tracking.

- [X] T037 [P] Update the feature documentation and tech-stack notes for `go-git` and `monaco-editor` in `README.md`, `specs/008-codebase-awareness/research.md`, and `specs/008-codebase-awareness/contracts/workspace-codebase-awareness.md`
- [X] T038 Verify `internal/tools`, `internal/orchestrator/workspace`, and any touched core packages remain at or above 75% coverage in `internal/tools/*.go`, `internal/orchestrator/workspace/*.go`, and `tests/integration/codebase_awareness_test.go`
- [X] T039 [P] Run and fix the focused backend and frontend regression suites for repository connection, approval persistence, repository-context delivery, and agent file activity in `tests/integration/codebase_awareness_test.go`, `tests/integration/tool_call_ordering_test.go`, `web/src/features/approvals/ApprovalReviewPanel.test.tsx`, and `web/src/shared/lib/workspace-store.test.ts`
- [X] T040 [P] Verify keyboard access, visible focus, forced-colors behavior, plain-language errors, explicit empty states, and 320px reflow for the new approval and repository-aware UI in `web/src/features/approvals/ApprovalReviewPanel.tsx`, `web/src/features/preferences/PreferencesPanel.tsx`, and `web/src/app/globals.css`
- [X] T041 Run the full quickstart validation flow and capture any follow-up fixes in `specs/008-codebase-awareness/quickstart.md`, `README.md`, and `tests/integration/codebase_awareness_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP.
- **Phase 4: US2** depends on Foundational and on the repository-aware tool and connection surfaces delivered in US1.
- **Phase 5: US3** depends on Foundational and benefits from US1 and US2 because graph and agent-activity views build on connected-repository state plus enriched tool and approval events.
- **Phase 6: Polish** depends on the completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on US1 because approval review requires connected repository context and repo-aware mutation proposals.
- **US3**: Depends on US1 for repository connection and repo-aware reads, and on US2 for the richer approval lifecycle and event payloads used to show proposal outcomes.

### Within Each User Story

- Write tests first and confirm they fail before implementation.
- Complete backend persistence and protocol changes before final UI wiring for that story.
- Preserve handler-level approval enforcement and repo-root sandboxing before marking the story complete.
- Finish loading, empty, and plain-language error states before marking the story complete.
- Revalidate the connected repository before any command execution or diff application.

## Parallel Opportunities

- `T002`, `T003`, and `T004` can run in parallel during Setup.
- `T006`, `T007`, `T008`, `T009`, and `T010` can run in parallel during Foundational work after the approval schema direction in `T005` is established.
- `T011`, `T012`, `T013`, and `T014` can run in parallel for US1 tests.
- `T015`, `T016`, and `T017` can run in parallel after US1 tests are in place; `T018` and `T019` follow once protocol and tool surfaces exist.
- `T020`, `T021`, `T022`, and `T023` can run in parallel for US2 tests.
- `T024`, `T025`, `T026`, and `T027` can run in parallel after US2 tests are in place; `T028` follows once backend payloads are stable.
- `T029`, `T030`, and `T031` can run in parallel for US3 tests.
- `T032`, `T033`, `T034`, and `T035` can run in parallel after US3 tests are in place; `T036` follows once repository-context and activity models are available.
- `T037`, `T039`, and `T040` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T011 internal/config/config_test.go + internal/orchestrator/workspace/preferences_test.go + internal/handlers/ws/workspace_test.go
Task: T012 internal/tools/catalog_test.go + internal/tools/read_file_test.go + internal/tools/search_codebase_test.go + internal/tools/list_files_test.go + internal/tools/git_log_test.go + internal/tools/git_diff_test.go
Task: T013 tests/integration/codebase_awareness_test.go + tests/integration/workspace_sessions_test.go
Task: T014 web/src/features/preferences/PreferencesPanel.test.tsx + web/src/features/workspace-shell/WorkspaceShell.test.tsx + web/src/shared/lib/workspace-store.test.ts

# Parallel US1 implementation work
Task: T015 cmd/relay/main.go + internal/config/config.go + internal/app/server.go
Task: T016 internal/orchestrator/workspace/repository_browser.go + internal/handlers/ws/protocol.go + internal/handlers/ws/workspace.go
Task: T017 internal/tools/list_files.go + internal/tools/git_log.go + internal/tools/git_diff.go + internal/tools/read_file.go + internal/tools/search_codebase.go
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T020 internal/orchestrator/workspace/service_test.go + internal/orchestrator/workspace/tool_executor_test.go
Task: T021 internal/tools/write_file_test.go + internal/tools/run_command_test.go + internal/tools/catalog_test.go
Task: T022 tests/integration/tool_call_ordering_test.go + tests/integration/codebase_awareness_test.go
Task: T023 web/src/features/approvals/ApprovalReviewPanel.test.tsx + web/src/shared/lib/workspace-store.test.ts

# Parallel US2 implementation work
Task: T024 internal/storage/sqlite/store.go + internal/orchestrator/workspace/service.go + internal/orchestrator/workspace/runs.go
Task: T025 internal/tools/write_file.go + internal/orchestrator/workspace/tool_executor.go + internal/handlers/ws/protocol.go
Task: T026 internal/tools/run_command.go + internal/tools/path_guard.go + internal/orchestrator/workspace/tool_executor.go
Task: T027 web/src/shared/lib/workspace-protocol.ts + web/src/shared/lib/workspace-store.ts + web/src/shared/lib/useWorkspaceSocket.ts
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T029 internal/orchestrator/workspace/service_test.go + internal/orchestrator/workspace/repository_graph_test.go
Task: T030 tests/integration/codebase_awareness_test.go + tests/integration/workspace_sessions_test.go
Task: T031 web/src/shared/lib/workspace-store.test.ts + web/src/features/canvas/AgentCanvasNode.test.tsx + web/src/features/canvas/canvasModel.test.ts

# Parallel US3 implementation work
Task: T032 internal/orchestrator/workspace/repository_graph.go + internal/orchestrator/workspace/service.go + internal/handlers/ws/workspace.go
Task: T033 internal/handlers/ws/protocol.go + web/src/shared/lib/workspace-protocol.ts + web/src/shared/lib/workspace-store.ts
Task: T034 web/src/features/codebase/graphModel.ts + web/src/shared/lib/workspace-store.ts
Task: T035 web/src/features/canvas/canvasModel.ts + web/src/features/canvas/AgentCanvasNode.tsx + web/src/features/canvas/AgentNodeDetailPanel.tsx
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate repository connection, repo-aware read-only tools, and repository-boundary enforcement before expanding scope.

### Incremental Delivery

1. Deliver US1 for repository connection and safe read-only code awareness.
2. Add US2 for persisted approval review and mutation gating.
3. Add US3 for background repository context and per-agent file activity.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer owns backend persistence, protocol, and repo-validation work during Foundational tasks.
2. One developer owns frontend store, preferences, and approval-review surfaces after the protocol contracts stabilize.
3. Once approval and repository event plumbing is stable, another developer can build the background graph and canvas activity views in parallel.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the limits described in the specification.
- Do not bypass the Relay architecture boundary of handlers to orchestrator to agents to tools to storage.
- Do not allow file writes or shell commands to execute without a persisted approval record and a fresh backend revalidation step.
- Background repository-context work must stay asynchronous and cancellable; it cannot block workspace bootstrap or approval handling.