# Tasks: Static Agent Canvas

**Input**: Design documents from `/specs/005-agent-canvas/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/canvas-ui.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include frontend component and utility coverage for the custom React Flow node, graph layout behavior, toolbar interactions, detail panel behavior, and the no-relayout-on-state-change invariant.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Specs**: `specs/005-agent-canvas/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add approved dependencies, create the canvas feature scaffolding, and document the new isolated canvas surface.

- [x] T001 Add `@xyflow/react` and `@dagrejs/dagre` to the frontend dependency graph in `web/package.json`
- [x] T002 [P] Create the canvas feature scaffolds in `web/src/features/canvas/AgentCanvas.tsx`, `web/src/features/canvas/AgentCanvasNode.tsx`, `web/src/features/canvas/AgentCanvasToolbar.tsx`, and `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [x] T003 [P] Create the canvas model and layout utility scaffolds in `web/src/features/canvas/canvasModel.ts` and `web/src/features/canvas/layoutGraph.ts`
- [x] T004 [P] Update the tech stack and local validation notes for the isolated canvas feature in `README.md` and `specs/005-agent-canvas/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the shared local canvas model, layout boundary, and shell integration required by every story.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T005 Implement the local canvas state model, node factories, edge factories, and structure-vs-state update helpers in `web/src/features/canvas/canvasModel.ts`
- [x] T006 [P] Implement the dagre layout adapter that recalculates positions only for structural graph changes in `web/src/features/canvas/layoutGraph.ts`
- [x] T007 [P] Add canvas-specific dark-mode state styles, forced-colors fallbacks, and focus-visible rules in `web/src/app/globals.css`
- [x] T008 Integrate the isolated canvas entry point into `web/src/features/canvas/WorkspaceCanvas.tsx` while preserving the existing empty-state fallback in `web/src/features/canvas/CanvasEmptyState.tsx`
- [x] T009 [P] Add shared test helpers for rendering the isolated canvas in `web/src/shared/lib/test-helpers.tsx`

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Build a readable agent graph (Priority: P1) 🎯 MVP

**Goal**: Let the developer add agent nodes and immediately see a readable directed workflow with visible node identity and automatic layout.

**Independent Test**: Start from an empty canvas, add a Planner node and then a Coder node, and verify both nodes render with correct labels, state chrome, and a directed edge in a clean automatic layout.

### Tests for User Story 1

- [x] T010 [P] [US1] Add layout utility coverage for stable directed node placement and edge generation in `web/src/features/canvas/layoutGraph.test.ts`
- [x] T011 [P] [US1] Add component coverage for empty-state replacement, node rendering, and add-node toolbar actions in `web/src/features/canvas/AgentCanvas.test.tsx`
- [x] T012 [P] [US1] Add workspace integration coverage for rendering the new canvas surface inside the shell in `web/src/features/workspace-shell/WorkspaceShell.test.tsx`

### Implementation for User Story 1

- [x] T013 [P] [US1] Implement the controlled React Flow canvas shell, node and edge state wiring, and add-node flow in `web/src/features/canvas/AgentCanvas.tsx`
- [x] T014 [P] [US1] Implement the custom agent node card with name, role badge, and state treatment in `web/src/features/canvas/AgentCanvasNode.tsx`
- [x] T015 [P] [US1] Implement the add-node development toolbar with role insertion controls in `web/src/features/canvas/AgentCanvasToolbar.tsx`
- [x] T016 [US1] Replace the placeholder active canvas surface with the isolated graph experience in `web/src/features/canvas/WorkspaceCanvas.tsx`
- [x] T017 [US1] Add explicit empty, validation, and local-only explanatory messaging to the canvas surface in `web/src/features/canvas/AgentCanvas.tsx` and `web/src/features/canvas/CanvasEmptyState.tsx`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Inspect and update a node in place (Priority: P2)

**Goal**: Let the developer inspect a selected node in a side panel and mutate its state without changing its position.

**Independent Test**: Click the Coder node to open its details, change its state to `thinking`, and verify the panel updates while the node remains visible in the same position.

### Tests for User Story 2

- [x] T018 [P] [US2] Add component coverage for node selection, detail panel open and close behavior, and background-click deselection in `web/src/features/canvas/AgentCanvas.test.tsx`
- [x] T019 [P] [US2] Add model-level coverage for state-only updates preserving node identity and coordinates in `web/src/features/canvas/layoutGraph.test.ts` and `web/src/features/canvas/canvasModel.ts`

### Implementation for User Story 2

- [x] T020 [P] [US2] Implement the node detail side panel with selected-node metadata and graph context in `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [x] T021 [P] [US2] Extend the toolbar to support selected-node state mutation controls in `web/src/features/canvas/AgentCanvasToolbar.tsx`
- [x] T022 [P] [US2] Implement selection, deselection, and state-mutation reducers that do not trigger relayout in `web/src/features/canvas/canvasModel.ts`
- [x] T023 [US2] Wire the side panel and state-mutation flow into the main canvas controller in `web/src/features/canvas/AgentCanvas.tsx`
- [x] T024 [US2] Reuse or adapt state semantics for node details without leaking live-run assumptions in `web/src/features/agent-panel/StateBadge.tsx` and `web/src/features/canvas/AgentCanvasNode.tsx`

**Checkpoint**: User Stories 1 and 2 work independently, including node inspection and in-place state mutation.

---

## Phase 5: User Story 3 - Keep the canvas interactive during updates (Priority: P3)

**Goal**: Preserve pan, zoom, click, and stable rendering while nodes are added or state changes occur.

**Independent Test**: Pan and zoom while adding a Tester node and while mutating an existing node state, then verify the canvas remains responsive and no node disappears or flickers.

### Tests for User Story 3

- [x] T025 [P] [US3] Add component coverage for viewport controls remaining usable during node addition and state changes in `web/src/features/canvas/AgentCanvas.test.tsx`
- [x] T026 [P] [US3] Add coverage for preserving selection while adding nodes and for skipping relayout on state-only updates in `web/src/features/canvas/layoutGraph.test.ts` and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`

### Implementation for User Story 3

- [x] T027 [P] [US3] Configure React Flow viewport behavior, background click handling, and interaction settings for uninterrupted pan and zoom in `web/src/features/canvas/AgentCanvas.tsx`
- [x] T028 [P] [US3] Tune structural-update rendering to avoid flicker and preserve stable node keys during node insertion in `web/src/features/canvas/canvasModel.ts` and `web/src/features/canvas/AgentCanvasNode.tsx`
- [x] T029 [P] [US3] Ensure detail-panel and toolbar behavior remain usable during graph updates in `web/src/features/canvas/AgentNodeDetailPanel.tsx` and `web/src/features/canvas/AgentCanvasToolbar.tsx`
- [x] T030 [US3] Finalize responsive layout and 320px reflow behavior for the canvas surface and side panel in `web/src/features/canvas/WorkspaceCanvas.tsx` and `web/src/app/globals.css`

**Checkpoint**: All three user stories work independently, including interaction stability under updates.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, quality checks, accessibility, and full quickstart validation across the canvas feature.

- [x] T031 [P] Document the finished canvas behavior, dependency rationale, and user-story checkpoints in `specs/005-agent-canvas/research.md`, `specs/005-agent-canvas/data-model.md`, and `specs/005-agent-canvas/contracts/canvas-ui.md`
- [x] T032 Verify keyboard access, focus visibility, explicit empty and validation states, forced-colors support, and 320px reflow in `web/src/features/canvas/AgentCanvas.tsx`, `web/src/features/canvas/AgentNodeDetailPanel.tsx`, and `web/src/app/globals.css`
- [x] T033 [P] Run and fix the focused validation suite in `web/src/features/canvas/AgentCanvas.test.tsx`, `web/src/features/canvas/layoutGraph.test.ts`, and `web/src/features/workspace-shell/WorkspaceShell.test.tsx`
- [x] T034 Run the full quickstart validation flow and capture any follow-up doc or UX fixes in `specs/005-agent-canvas/quickstart.md` and `README.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP.
- **Phase 4: US2** depends on Foundational and builds on the canvas created in US1, while remaining independently testable once the graph exists.
- **Phase 5: US3** depends on Foundational and uses the graph plus state-mutation behavior from earlier stories to validate interaction stability.
- **Phase 6: Polish** depends on the completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on US1’s graph and node rendering surfaces but remains independently testable through node selection and state mutation flows.
- **US3**: Depends on US1 and US2 because interaction stability must be verified against actual node insertion and state-mutation behavior.

### Within Each User Story

- Write tests first and confirm they fail before implementation.
- Complete model and layout logic before final UI wiring for that story.
- Finish empty, validation, and accessibility states before marking the story complete.
- Keep the local-only boundary intact; do not introduce WebSocket, backend, or storage coupling while implementing any story.

## Parallel Opportunities

- `T002`, `T003`, and `T004` can run in parallel after dependency selection in `T001`.
- `T006`, `T007`, and `T009` can run in parallel within Foundational work after `T005` begins defining the local canvas model.
- `T010`, `T011`, and `T012` can run in parallel for US1 tests.
- `T013`, `T014`, and `T015` can run in parallel after US1 test scaffolding is in place.
- `T018` and `T019` can run in parallel for US2 test coverage.
- `T020`, `T021`, and `T022` can run in parallel after US2 test scaffolding is in place.
- `T025` and `T026` can run in parallel for US3 test coverage.
- `T027`, `T028`, and `T029` can run in parallel after US3 tests are in place.
- `T031` and `T033` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T010 web/src/features/canvas/layoutGraph.test.ts
Task: T011 web/src/features/canvas/AgentCanvas.test.tsx
Task: T012 web/src/features/workspace-shell/WorkspaceShell.test.tsx

# Parallel US1 implementation work
Task: T013 web/src/features/canvas/AgentCanvas.tsx
Task: T014 web/src/features/canvas/AgentCanvasNode.tsx
Task: T015 web/src/features/canvas/AgentCanvasToolbar.tsx
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T018 web/src/features/canvas/AgentCanvas.test.tsx
Task: T019 web/src/features/canvas/layoutGraph.test.ts + web/src/features/canvas/canvasModel.ts

# Parallel US2 implementation work
Task: T020 web/src/features/canvas/AgentNodeDetailPanel.tsx
Task: T021 web/src/features/canvas/AgentCanvasToolbar.tsx
Task: T022 web/src/features/canvas/canvasModel.ts
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T025 web/src/features/canvas/AgentCanvas.test.tsx
Task: T026 web/src/features/canvas/layoutGraph.test.ts + web/src/features/workspace-shell/WorkspaceShell.test.tsx

# Parallel US3 implementation work
Task: T027 web/src/features/canvas/AgentCanvas.tsx
Task: T028 web/src/features/canvas/canvasModel.ts + web/src/features/canvas/AgentCanvasNode.tsx
Task: T029 web/src/features/canvas/AgentNodeDetailPanel.tsx + web/src/features/canvas/AgentCanvasToolbar.tsx
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate the add-node and automatic-layout flow before expanding scope.

### Incremental Delivery

1. Deliver US1 for the core readable graph experience.
2. Add US2 for node inspection and state mutation.
3. Add US3 for interaction stability and responsiveness under updates.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer owns the local model and layout utility during Foundational work.
2. One developer owns canvas UI components and shell integration once the model boundary is stable.
3. After US1, one developer can focus on node detail behavior while another validates interaction stability and responsive behavior.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the bounds described in the specification.
- Do not introduce backend handlers, WebSocket events, or storage changes while implementing these tasks.
- Keep the UI explicit that all node data and mutations are local to this isolated experience.