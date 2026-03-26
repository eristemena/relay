# Tasks: Live Agent Panel

**Input**: Design documents from `/specs/004-live-agent-panel/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/, quickstart.md

**Historical Note**: These tasks record the original implementation of the dedicated live execution drawer panel. The current product has since removed that panel in favor of the command bar, canvas node detail, saved runs, and approval review surfaces.

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include Go unit coverage for config, agents, orchestrator, and every tool happy/error path; Go integration coverage for WebSocket protocol ordering, approval gating, replay, and OpenRouter failure handling; plus frontend component coverage for the live panel, tool timeline, history review, and preferences flows.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `cmd/relay/`, `internal/app/`, `internal/config/`, `internal/handlers/`, `internal/orchestrator/`, `internal/storage/`, `internal/agents/`, `internal/tools/`, `tests/integration/`
- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/004-live-agent-panel/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the approved dependencies, scaffold the live-agent feature areas, and document the runtime additions needed by every story.

- [X] T001 Add the OpenRouter streaming dependency and lock the backend module graph in `go.mod` and `go.sum`
- [X] T002 Add any required frontend dependencies and scripts for the live agent panel in `web/package.json` and `web/package-lock.json`
- [X] T003 [P] Scaffold the agent and tool package entrypoints in `internal/agents/agent.go`, `internal/agents/registry.go`, and `internal/tools/catalog.go`
- [X] T004 [P] Scaffold the frontend live-panel feature area in `web/src/features/agent-panel/AgentCommandBar.tsx`, `web/src/features/agent-panel/AgentPanel.tsx`, and `web/src/features/agent-panel/RunTimeline.tsx`
- [X] T005 [P] Update the approved tech stack and local setup notes for OpenRouter-backed agent runs, default role-model assignments, and manual `project_root` configuration in `README.md` and `specs/004-live-agent-panel/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the config, persistence, protocol, approval, orchestration, and transport foundations that block all user stories.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T006 Implement OpenRouter credential loading, manual `project_root` parsing and validation, per-role model defaults, and redacted status serialization in `internal/config/config.go`
- [X] T007 [P] Extend the SQLite schema and sqlc inputs for agent runs and ordered run events in `internal/storage/sqlite/migrations/0002_agent_runs.sql`, `internal/storage/sqlite/queries/agent_runs.sql`, `internal/storage/sqlite/queries/agent_run_events.sql`, and `sqlc.yaml`
- [X] T008 [P] Implement storage models and store methods for run summaries, ordered event append, replay loading, and active-run lookups in `internal/storage/sqlite/models.go` and `internal/storage/sqlite/store.go`
- [X] T009 [P] Define the live-agent WebSocket envelopes, approval-related events, and shared TypeScript DTOs in `internal/handlers/ws/protocol.go` and `web/src/shared/lib/workspace-protocol.ts`
- [X] T010 [P] Implement the role registry, fixed prompts, fixed tool allowlists, and OpenRouter stream normalization in `internal/agents/planner.go`, `internal/agents/coder.go`, `internal/agents/reviewer.go`, `internal/agents/tester.go`, `internal/agents/explainer.go`, and `internal/agents/openrouter/client.go`
- [X] T011 [P] Implement the server-side tool catalog, `project_root`-backed repo guards, and display-safe redaction helpers in `internal/tools/catalog.go`, `internal/tools/read_file.go`, and `internal/tools/search_codebase.go`
- [X] T012 Implement handler-level approval enforcement, approval request or rejection flow, and mutating tool dispatch for `write_file` and `run_command` in `internal/handlers/ws/workspace.go`, `internal/orchestrator/workspace/service.go`, `internal/tools/write_file.go`, and `internal/tools/run_command.go`
- [X] T013 Implement single-active-run lifecycle management, ordered event sequencing, replay hydration, and cancellation plumbing in `internal/orchestrator/workspace/runs.go`, `internal/orchestrator/workspace/history.go`, and `internal/orchestrator/workspace/service.go`
- [X] T014 [P] Extend the frontend socket store for credential status, run summaries, active run state, approval-blocked states, and ordered event replay in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/useWorkspaceSocket.ts`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Run a task and watch the live stream (Priority: P1) 🎯 MVP

**Goal**: Let the developer submit one task, have Relay pick a single specialized agent, and render its live streamed output with clear run identity and active-state feedback.

**Independent Test**: Start Relay with a valid OpenRouter API key, submit one task from the command bar, and verify one agent run starts, streams visible output in order with a live cursor, shows the selected role and model badge, and ends in a reusable completed or errored state.

### Tests for User Story 1

- [X] T015 [P] [US1] Add config and orchestrator coverage for missing credentials, missing or invalid `project_root`, model fallback, and single-run acceptance in `internal/config/config_test.go` and `internal/orchestrator/workspace/service_test.go`
- [X] T016 [P] [US1] Add agent construction coverage for fixed prompts, default model assignments, and per-role allowlists in `internal/agents/registry_test.go` and `internal/agents/openrouter/client_test.go`
- [X] T017 [P] [US1] Add live streaming integration coverage for submit, first-token delivery, stream ordering, and single-active-run rejection in `tests/integration/agent_streaming_test.go`
- [X] T018 [P] [US1] Add component coverage for command submission, waiting-for-output, streaming cursor, role identity, and terminal states in `web/src/features/agent-panel/AgentPanel.test.tsx` and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`

### Implementation for User Story 1

- [X] T019 [P] [US1] Implement automatic role selection, run bootstrap metadata, and stream-to-event conversion in `internal/orchestrator/workspace/runs.go` and `internal/agents/registry.go`
- [X] T020 [P] [US1] Implement the live run submit, request validation, and active-run replay flows over WebSocket in `internal/handlers/ws/workspace.go` and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T021 [P] [US1] Build the command bar and submission UX for natural-language tasks in `web/src/features/agent-panel/AgentCommandBar.tsx` and `web/src/features/agent-panel/AgentPanel.tsx`
- [X] T022 [P] [US1] Build the thought viewer, run header, model badge, and live cursor rendering in `web/src/features/agent-panel/ThoughtViewer.tsx`, `web/src/features/agent-panel/RunHeader.tsx`, and `web/src/features/agent-panel/LiveCursor.tsx`
- [X] T023 [P] [US1] Integrate the live agent panel into the main workspace shell and default empty-state layout in `web/src/app/page.tsx`, `web/src/features/workspace-shell/WorkspaceShell.tsx`, and `web/src/features/canvas/WorkspaceCanvas.tsx`
- [X] T024 [US1] Surface blocked-run, missing-`project_root`, waiting-for-first-output, completed, and errored messaging without exposing secrets in `internal/orchestrator/workspace/service.go`, `web/src/features/agent-panel/AgentPanel.tsx`, and `web/src/features/workspace-shell/WorkspaceStatusBanner.tsx`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - See tool activity and execution state inline (Priority: P2)

**Goal**: Show state transitions and tool activity inline in the live and replayed timeline, with sensitive values redacted and failures preserved in context.

**Independent Test**: Run a task that triggers at least one tool call and verify the panel shows ordered state changes plus `tool_call` and `tool_result` entries with safe previews, distinct statuses, and an inline failure record when a tool or provider step fails.

### Tests for User Story 2

- [X] T025 [P] [US2] Add tool happy-path and primary error-path tests for `read_file` and `search_codebase` plus redaction coverage in `internal/tools/catalog_test.go` and `internal/tools/search_codebase_test.go`
- [X] T026 [P] [US2] Add approval and error-path tests for `write_file` and `run_command` in `internal/tools/write_file_test.go`, `internal/tools/run_command_test.go`, and `internal/orchestrator/workspace/service_test.go`
- [X] T027 [P] [US2] Add integration coverage for tool-call ordering, approval rejection, redacted previews, and terminal failure preservation in `tests/integration/tool_call_ordering_test.go`
- [X] T028 [P] [US2] Add component coverage for state badges, inline tool rows, approval-blocked states, and errored timeline rendering in `web/src/features/agent-panel/RunTimeline.test.tsx` and `web/src/features/agent-panel/StateBadge.test.tsx`

### Implementation for User Story 2

- [X] T029 [P] [US2] Emit ordered state-change, tool-call, tool-result, complete, and error events with display-safe previews in `internal/agents/openrouter/client.go` and `internal/orchestrator/workspace/runs.go`
- [X] T030 [P] [US2] Implement secret redaction and protected preview generation for tool inputs and results in `internal/tools/catalog.go`, `internal/tools/write_file.go`, and `internal/tools/run_command.go`
- [X] T031 [P] [US2] Build the distinct run-state badge and inline tool activity components in `web/src/features/agent-panel/StateBadge.tsx`, `web/src/features/agent-panel/ToolEventRow.tsx`, and `web/src/features/agent-panel/RunTimeline.tsx`
- [X] T032 [P] [US2] Update the shared workspace store to preserve chronological mixed event streams and approval-blocked status for live and replay modes in `web/src/shared/lib/workspace-store.ts` and `web/src/shared/lib/workspace-protocol.ts`
- [X] T033 [US2] Surface inline human-readable failure explanations, approval-required states, and tool-running states in `web/src/features/agent-panel/AgentPanel.tsx` and `web/src/features/agent-panel/ThoughtViewer.tsx`

**Checkpoint**: User Stories 1 and 2 work independently, including inline tool transparency and failure review.

---

## Phase 5: User Story 3 - Configure access and review past runs (Priority: P3)

**Goal**: Persist the OpenRouter API key server-side, keep completed or errored runs across Relay restarts, and let the developer reopen saved runs for read-only review.

**Independent Test**: Save an API key, complete one or more runs, restart Relay, and verify the saved credential status and run history are restored so a previous run can be reopened and replayed without contacting the provider.

### Tests for User Story 3

- [X] T034 [P] [US3] Add unit coverage for credential persistence, manual `project_root` persistence and validation, run summary queries, and replay loading in `internal/config/config_test.go`, `internal/storage/sqlite/store_test.go`, and `internal/orchestrator/workspace/preferences_test.go`
- [X] T035 [P] [US3] Add integration coverage for restart-safe run history replay, bootstrap hydration, and `agent.run.open` behavior in `tests/integration/run_history_replay_test.go`
- [X] T036 [P] [US3] Add component coverage for credential save states, run history empty/loading/error states, and reopening a saved run in `web/src/features/preferences/PreferencesPanel.test.tsx` and `web/src/features/history/RunHistoryPanel.test.tsx`

### Implementation for User Story 3

- [X] T037 [P] [US3] Implement OpenRouter credential save, load, and status projection in `internal/config/config.go`, `internal/orchestrator/workspace/service.go`, and `web/src/features/preferences/PreferencesStatus.tsx`
- [X] T038 [P] [US3] Implement persisted run summary loading, replay fetch, and active-run restoration in `internal/orchestrator/workspace/history.go`, `internal/handlers/ws/workspace.go`, and `web/src/shared/lib/useWorkspaceSocket.ts`
- [X] T039 [P] [US3] Build the run history list, empty state, and replay entry point UI in `web/src/features/history/RunHistoryPanel.tsx`, `web/src/features/history/RunHistoryListItem.tsx`, and `web/src/features/agent-panel/AgentPanel.tsx`
- [X] T040 [US3] Extend the preferences panel with OpenRouter key editing, manual `project_root` guidance, saving, saved, and error states without revealing the full secret in `web/src/features/preferences/PreferencesPanel.tsx` and `web/src/shared/lib/workspace-store.ts`

**Checkpoint**: All three user stories work independently, including restart-safe credentials and run history review.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, regression coverage, performance checks, and accessibility/security hardening across all stories.

- [X] T041 [P] Update the product and feature documentation for agent roles, OpenRouter configuration, approval behavior, and run review in `README.md`, `specs/004-live-agent-panel/quickstart.md`, and `specs/004-live-agent-panel/research.md`
- [x] T042 Verify `internal/agents`, `internal/orchestrator`, and `internal/tools` remain at or above 75% coverage in `internal/agents/openrouter/client_test.go`, `internal/orchestrator/workspace/service_test.go`, and `internal/tools/catalog_test.go`
- [X] T043 [P] Add any remaining cross-story regression coverage for bootstrap hydration, replay ordering, credential status, and project-root errors in `tests/integration/agent_streaming_test.go`, `tests/integration/tool_call_ordering_test.go`, and `tests/integration/run_history_replay_test.go`
- [x] T044 Verify keyboard access, visible focus, explicit empty/loading/error states, and 320px reflow across the live panel and history UI in `web/src/features/agent-panel/AgentPanel.tsx`, `web/src/features/history/RunHistoryPanel.tsx`, and `web/src/app/globals.css`
- [x] T045 Measure and tune first-token latency, WebSocket dispatch ordering, and UI responsiveness under active streaming in `internal/agents/openrouter/client.go`, `internal/orchestrator/workspace/runs.go`, and `web/src/shared/lib/workspace-store.ts`
- [X] T046 Run the full quickstart validation and capture any follow-up fixes in `specs/004-live-agent-panel/quickstart.md`, `tests/integration/agent_streaming_test.go`, and `web/src/features/agent-panel/AgentPanel.test.tsx`

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
- **US2**: Depends on Foundational storage, agent, tool, and WebSocket infrastructure but remains independently testable from US1.
- **US3**: Depends on Foundational config, persistence, and WebSocket infrastructure but remains independently testable from US1 and US2.

### Within Each User Story

- Write tests first and confirm they fail before implementation.
- Complete backend persistence and protocol work before final UI integration for that story.
- Preserve approval enforcement, repo-root sandboxing, and secret redaction before marking the story done.
- Finish empty, loading, waiting, and recoverable error states before the story is considered complete.

## Parallel Opportunities

- `T003`, `T004`, and `T005` can run in parallel after dependency decisions in `T001` and `T002`.
- `T007`, `T008`, `T009`, `T010`, `T011`, and `T014` can run in parallel within Foundational work after `T006`.
- `T015`, `T016`, `T017`, and `T018` can run in parallel for US1 tests.
- `T019`, `T020`, `T021`, `T022`, and `T023` can run in parallel after US1 tests are in place.
- `T025`, `T026`, `T027`, and `T028` can run in parallel for US2 tests.
- `T029`, `T030`, `T031`, and `T032` can run in parallel after US2 tests are in place.
- `T034`, `T035`, and `T036` can run in parallel for US3 tests.
- `T037`, `T038`, and `T039` can run in parallel after US3 tests are in place.
- `T041`, `T043`, and `T044` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T015 internal/config/config_test.go + internal/orchestrator/workspace/service_test.go
Task: T016 internal/agents/registry_test.go + internal/agents/openrouter/client_test.go
Task: T017 tests/integration/agent_streaming_test.go
Task: T018 web/src/features/agent-panel/AgentPanel.test.tsx + web/src/features/workspace-shell/WorkspaceShell.test.tsx

# Parallel US1 implementation work
Task: T019 internal/orchestrator/workspace/runs.go + internal/agents/registry.go
Task: T020 internal/handlers/ws/workspace.go + web/src/shared/lib/useWorkspaceSocket.ts
Task: T021 web/src/features/agent-panel/AgentCommandBar.tsx + web/src/features/agent-panel/AgentPanel.tsx
Task: T022 web/src/features/agent-panel/ThoughtViewer.tsx + web/src/features/agent-panel/RunHeader.tsx + web/src/features/agent-panel/LiveCursor.tsx
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T025 internal/tools/catalog_test.go + internal/tools/search_codebase_test.go
Task: T026 internal/tools/write_file_test.go + internal/tools/run_command_test.go + internal/orchestrator/workspace/service_test.go
Task: T027 tests/integration/tool_call_ordering_test.go
Task: T028 web/src/features/agent-panel/RunTimeline.test.tsx + web/src/features/agent-panel/StateBadge.test.tsx

# Parallel US2 implementation work
Task: T029 internal/agents/openrouter/client.go + internal/orchestrator/workspace/runs.go
Task: T030 internal/tools/catalog.go + internal/tools/write_file.go + internal/tools/run_command.go
Task: T031 web/src/features/agent-panel/StateBadge.tsx + web/src/features/agent-panel/ToolEventRow.tsx + web/src/features/agent-panel/RunTimeline.tsx
Task: T032 web/src/shared/lib/workspace-store.ts + web/src/shared/lib/workspace-protocol.ts
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T034 internal/config/config_test.go + internal/storage/sqlite/store_test.go + internal/orchestrator/workspace/preferences_test.go
Task: T035 tests/integration/run_history_replay_test.go
Task: T036 web/src/features/preferences/PreferencesPanel.test.tsx + web/src/features/history/RunHistoryPanel.test.tsx

# Parallel US3 implementation work
Task: T037 internal/config/config.go + internal/orchestrator/workspace/service.go + web/src/features/preferences/PreferencesStatus.tsx
Task: T038 internal/orchestrator/workspace/history.go + internal/handlers/ws/workspace.go + web/src/shared/lib/useWorkspaceSocket.ts
Task: T039 web/src/features/history/RunHistoryPanel.tsx + web/src/features/history/RunHistoryListItem.tsx + web/src/features/agent-panel/AgentPanel.tsx
Task: T040 web/src/features/preferences/PreferencesPanel.tsx + web/src/shared/lib/workspace-store.ts
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate the live submit and streaming flow end to end before expanding scope.

### Incremental Delivery

1. Deliver US1 for the core single-agent live streaming experience.
2. Add US2 for inline execution transparency and redacted tool visibility.
3. Add US3 for restart-safe credentials and run history replay.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer owns the backend foundation across config, agents, tools, storage, and orchestration in Phase 2.
2. One developer owns the frontend socket store and live panel integration beginning in Foundational and US1.
3. After Foundational work, separate developers can take US2 timeline transparency work and US3 history/preferences work in parallel with limited file overlap.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story is scoped to remain independently testable.
- Do not bypass the Relay architecture boundary of handlers to orchestrator to agents to tools to storage.
- Do not send credentials, raw secrets, or unrestricted file-system content to the frontend in any task.