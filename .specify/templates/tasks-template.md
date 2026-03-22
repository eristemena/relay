---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Tests are REQUIRED when the constitution or feature scope demands
them. Relay defaults include unit coverage for tool changes, integration tests
for WebSocket protocol changes, component tests for custom React Flow nodes, and
coverage preservation for core Go runtime packages.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `backend/cmd/`, `backend/internal/handlers/`,
  `backend/internal/orchestrator/`, `backend/internal/agents/`,
  `backend/internal/tools/`, `backend/internal/storage/`, `backend/tests/`
- **Frontend**: `frontend/src/app/`, `frontend/src/features/`,
  `frontend/src/shared/`, `frontend/tests/`
- Paths shown below assume the Relay structure above - adjust only if the plan
  explicitly documents an approved exception

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create or confirm Relay-compliant project structure per implementation plan
- [ ] T002 Initialize or update language/tooling configuration with Go and TypeScript strictness requirements
- [ ] T003 [P] Configure linting, formatting, and structured logging safeguards
- [ ] T004 [P] Update Tech Stack documentation for any newly approved third-party dependency

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T005 Establish handler-level approval enforcement for file writes and shell commands
- [ ] T006 [P] Implement or update WebSocket event contracts and dispatch plumbing
- [ ] T007 [P] Implement repo-scoped file access guards and sandboxed shell execution constraints
- [ ] T008 Create or update shared storage, logging, and error handling infrastructure without leaking secrets or prompt content
- [ ] T009 Ensure goroutine lifecycle management and context-based cancellation for new background work
- [ ] T010 Confirm SQLite access patterns avoid N+1 queries for shared data paths

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 1 ⚠️

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T011 [P] [US1] Add or update unit tests for changed backend/frontend modules
- [ ] T012 [P] [US1] Add tool happy-path and primary error-path tests for each affected agent tool
- [ ] T013 [P] [US1] Add WebSocket integration coverage if protocol or event payloads change
- [ ] T014 [P] [US1] Add a component test for each new custom React Flow node

### Implementation for User Story 1

- [ ] T015 [P] [US1] Implement data structures or models in the Relay-approved layer and path
- [ ] T016 [US1] Implement orchestration, tool, or UI behavior in the correct layer without skipping boundaries
- [ ] T017 [US1] Add visible loading, empty, and human-readable error states for the story
- [ ] T018 [US1] Ensure approvals, sandboxing, and secret-handling rules remain enforced
- [ ] T019 [US1] Verify no banned logging/debug statements or undocumented exported APIs were introduced

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 2 ⚠️

- [ ] T020 [P] [US2] Add or update unit tests for changed backend/frontend modules
- [ ] T021 [P] [US2] Add tool happy-path and primary error-path tests for each affected agent tool
- [ ] T022 [P] [US2] Add WebSocket integration coverage if protocol or event payloads change
- [ ] T023 [P] [US2] Add a component test for each new custom React Flow node

### Implementation for User Story 2

- [ ] T024 [P] [US2] Implement data structures or models in the Relay-approved layer and path
- [ ] T025 [US2] Implement orchestration, tool, or UI behavior in the correct layer without skipping boundaries
- [ ] T026 [US2] Add visible loading, empty, and human-readable error states for the story
- [ ] T027 [US2] Ensure approvals, sandboxing, and secret-handling rules remain enforced

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Tests for User Story 3 ⚠️

- [ ] T028 [P] [US3] Add or update unit tests for changed backend/frontend modules
- [ ] T029 [P] [US3] Add tool happy-path and primary error-path tests for each affected agent tool
- [ ] T030 [P] [US3] Add WebSocket integration coverage if protocol or event payloads change
- [ ] T031 [P] [US3] Add a component test for each new custom React Flow node

### Implementation for User Story 3

- [ ] T032 [P] [US3] Implement data structures or models in the Relay-approved layer and path
- [ ] T033 [US3] Implement orchestration, tool, or UI behavior in the correct layer without skipping boundaries
- [ ] T034 [US3] Add visible loading, empty, and human-readable error states for the story
- [ ] T035 [US3] Ensure approvals, sandboxing, and secret-handling rules remain enforced

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in docs/ and Tech Stack notes
- [ ] TXXX Verify core package coverage remains at or above 75%
- [ ] TXXX Performance optimization across all stories, including WebSocket dispatch latency and canvas responsiveness
- [ ] TXXX [P] Add any remaining required unit, integration, and component tests
- [ ] TXXX Security hardening for secrets, prompt logging, repo-root sandboxing, and approval enforcement
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Required tests MUST be written and FAIL before implementation
- Models before services
- Services before endpoints or handlers
- Core implementation before integration
- Story complete before moving to next priority
- Approval enforcement, UX states, and security constraints must be implemented before the story is considered complete

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Models within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together (if tests requested):
Task: "Contract test for [endpoint] in tests/contract/test_[name].py"
Task: "Integration test for [user journey] in tests/integration/test_[name].py"

# Launch all models for User Story 1 together:
Task: "Create [Entity1] model in src/models/[entity1].py"
Task: "Create [Entity2] model in src/models/[entity2].py"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
- Do not generate tasks that violate the Relay constitution's layer order, approval rules, or mandatory test obligations
