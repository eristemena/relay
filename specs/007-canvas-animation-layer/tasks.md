# Tasks: Canvas Animation Layer

**Input**: Design documents from `/specs/007-canvas-animation-layer/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/websocket-animation-signals.md, quickstart.md

**Tests**: Tests are REQUIRED for this feature by the specification and constitution. Include Vitest plus React Testing Library coverage for node entry motion, state transitions, handoff edge pulsing, side-panel presence behavior, reduced-motion handling, loading and human-readable error-state preservation, and streaming-timer cleanup. Add Go or WebSocket integration coverage only if implementation proves the existing protocol payloads are insufficient and requires backend changes.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g. `US1`, `US2`, `US3`)
- Include exact file paths in descriptions

## Path Conventions

- **Frontend**: `web/src/app/`, `web/src/features/`, `web/src/shared/`
- **Backend**: `internal/handlers/`, `internal/orchestrator/` only if protocol shaping becomes necessary
- **Specs**: `specs/007-canvas-animation-layer/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare the motion-layer implementation surface and align documentation with the existing Relay frontend structure.

- [x] T001 Create the shared motion preset module for the canvas in `web/src/features/canvas/canvasMotion.ts`
- [x] T002 [P] Scaffold the custom edge and focused component test files in `web/src/features/canvas/AnimatedHandoffEdge.tsx` and `web/src/features/canvas/AnimatedHandoffEdge.test.tsx`
- [x] T003 [P] Update feature validation notes and implementation checkpoints in `specs/007-canvas-animation-layer/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared animation primitives and canvas data wiring required by every story.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 Implement shared 300ms motion presets, easing constants, and reduced-motion helpers in `web/src/features/canvas/canvasMotion.ts`
- [x] T005 [P] Extend canvas edge and node presentation models for pulse state and animation-safe metadata in `web/src/features/canvas/canvasModel.ts`
- [x] T006 [P] Register custom canvas edge types and motion-aware edge mapping in `web/src/features/canvas/AgentCanvas.tsx`
- [x] T007 [P] Add shared CSS keyframes, forced-colors-safe fallbacks, and panel-transition utility classes in `web/src/app/globals.css`
- [x] T008 Verify the existing WebSocket payloads already satisfy the animation contract in `web/src/shared/lib/workspace-protocol.ts` and `web/src/shared/lib/workspace-store.ts`, adding only frontend-derived shaping if needed

**Checkpoint**: Foundation ready. User story work can begin.

---

## Phase 3: User Story 1 - Read the live canvas through motion (Priority: P1) 🎯 MVP

**Goal**: Let the developer read live orchestration activity through node entry motion, state transitions, handoff pulses, and token-driven streaming indicators without changing orchestration behavior.

**Independent Test**: Run an existing orchestration flow and verify node entry, state changes, edge handoffs, and active streaming each produce the intended motion cues while the underlying run data remains unchanged.

### Tests for User Story 1

- [x] T009 [P] [US1] Add component coverage for motion-aware node rendering, state transitions, and token-driven streaming activity in `web/src/features/canvas/AgentCanvas.test.tsx`
- [x] T010 [P] [US1] Add focused custom-edge coverage for active and settled handoff pulse rendering in `web/src/features/canvas/AnimatedHandoffEdge.test.tsx`
- [x] T011 [P] [US1] Add canvas-model coverage for handoff pulse-state derivation and non-duplicating edge updates in `web/src/features/canvas/canvasModel.test.ts`
- [x] T012 [US1] Add timing-focused validation for node entry start and streaming pulse silence windows in `web/src/features/canvas/AgentCanvas.test.tsx` and `web/src/features/canvas/AgentCanvasNode.test.tsx`

### Implementation for User Story 1

- [x] T013 [P] [US1] Implement the custom pulsing handoff edge renderer using React Flow path helpers in `web/src/features/canvas/AnimatedHandoffEdge.tsx`
- [x] T014 [P] [US1] Update canvas event projection to derive edge pulse state from `handoff_start` and `handoff_complete` in `web/src/features/canvas/canvasModel.ts`
- [x] T015 [P] [US1] Wrap the agent node root in `motion.div` variants for enter and state transitions in `web/src/features/canvas/AgentCanvasNode.tsx`
- [x] T016 [US1] Implement token-silence streaming activity timing with ref cleanup on unmount in `web/src/features/canvas/AgentCanvasNode.tsx`
- [x] T017 [US1] Wire motion-aware nodes and custom edge types into the live graph without enabling layout animations in `web/src/features/canvas/AgentCanvas.tsx`
- [x] T018 [US1] Tune node, edge, and streaming-indicator styling to preserve readability and existing state meaning in `web/src/app/globals.css`

**Checkpoint**: User Story 1 is functional and independently testable as the MVP.

---

## Phase 4: User Story 2 - Inspect agent output without abrupt panel changes (Priority: P2)

**Goal**: Let the developer open and switch the selected-node panel with clear motion that preserves context and continues to show the correct live content.

**Independent Test**: Select nodes during a live run, switch between them while output is still arriving, and confirm the detail panel enters from the right and always settles on the latest selected node without stale or mixed content.

### Tests for User Story 2

- [x] T019 [P] [US2] Add selection and panel-presence coverage for empty, entering, and switching states in `web/src/features/canvas/AgentCanvas.test.tsx`
- [x] T020 [P] [US2] Add focused detail-panel coverage for selected-node content continuity and human-readable error-state preservation during panel presence changes in `web/src/features/canvas/AgentNodeDetailPanel.test.tsx`
- [x] T021 [US2] Add loading-state preservation coverage during animated canvas and panel transitions in `web/src/features/canvas/AgentCanvas.test.tsx`

### Implementation for User Story 2

- [x] T022 [US2] Implement `AnimatePresence` panel entry or exit behavior and keyed content switching that preserves the latest selection and empty-selection state in `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [x] T023 [US2] Preserve visible loading and plain-language error states during animated panel and canvas transitions in `web/src/features/canvas/AgentCanvas.tsx`, `web/src/features/canvas/AgentNodeDetailPanel.tsx`, and `web/src/app/globals.css`
- [x] T024 [US2] Update the canvas detail-grid composition so panel motion stays outside the React Flow-managed node tree in `web/src/features/canvas/AgentCanvas.tsx`
- [x] T025 [US2] Refine panel styling and focus-visible behavior during animated presence changes in `web/src/app/globals.css` and `web/src/features/canvas/AgentNodeDetailPanel.tsx`

**Checkpoint**: User Stories 1 and 2 work independently, including live node inspection with animated panel transitions.

---

## Phase 5: User Story 3 - Keep motion safe, responsive, and truthful (Priority: P3)

**Goal**: Ensure the motion layer remains presentation-only, respects reduced-motion preferences, and does not degrade interaction performance or leave stale timers behind.

**Independent Test**: Exercise a live run with frequent state changes while panning, zooming, selecting nodes, and toggling reduced motion, then confirm interaction remains responsive, timers clean up correctly, and the visible state always matches the latest underlying run state.

### Tests for User Story 3

- [x] T026 [P] [US3] Add reduced-motion and interaction-responsiveness coverage against the feature timing targets in `web/src/features/canvas/AgentCanvas.test.tsx`
- [x] T027 [P] [US3] Add timer-cleanup and stale-streaming-indicator coverage for the 300ms silence window in `web/src/features/canvas/AgentCanvasNode.test.tsx`
- [x] T028 [P] [US3] Add regression coverage ensuring no animation path depends on backend-owned motion state in `web/src/shared/lib/workspace-store.test.ts`

### Implementation for User Story 3

- [x] T029 [P] [US3] Implement reduced-motion fallbacks for node, edge, and panel transitions in `web/src/features/canvas/canvasMotion.ts`, `web/src/features/canvas/AgentCanvasNode.tsx`, and `web/src/features/canvas/AgentNodeDetailPanel.tsx`
- [x] T030 [P] [US3] Ensure the canvas model always resolves to the latest authoritative state during overlapping motion windows in `web/src/features/canvas/canvasModel.ts`
- [x] T031 [P] [US3] Audit the graph wiring to keep pan, zoom, and selection responsive and to avoid accidental Framer Motion `layout` usage in `web/src/features/canvas/AgentCanvas.tsx` and `web/src/features/canvas/AgentCanvasNode.tsx`
- [x] T032 [US3] Finalize forced-colors-safe focus styles, reduced-motion-safe pulse fallbacks, and accessible non-selected states in `web/src/app/globals.css` and `web/src/features/canvas/AgentNodeDetailPanel.tsx`

**Checkpoint**: All three user stories work independently, including safe reduced-motion handling and responsiveness guarantees.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, documentation, and regression validation across the motion layer.

- [x] T033 [P] Update implementation notes and test commands in `README.md` and `specs/007-canvas-animation-layer/quickstart.md`
- [x] T034 Run and fix the focused frontend regression suite for canvas motion behavior in `web/src/features/canvas/AgentCanvas.test.tsx`, `web/src/features/canvas/AnimatedHandoffEdge.test.tsx`, `web/src/features/canvas/AgentNodeDetailPanel.test.tsx`, `web/src/features/canvas/AgentCanvasNode.test.tsx`, and `web/src/features/canvas/canvasModel.test.ts`
- [x] T035 [P] Run typecheck and verify no backend protocol change is required; if it is required, document and implement the minimal contract delta in `web/src/shared/lib/workspace-protocol.ts`, `internal/handlers/ws/protocol.go`, and `tests/integration/`
- [x] T036 Validate the manual quickstart flow against the documented 100ms and 300ms timing targets, including reduced motion and active-stream cleanup checks, in `specs/007-canvas-animation-layer/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies and can start immediately.
- **Phase 2: Foundational** depends on Setup and blocks all user stories.
- **Phase 3: US1** depends on Foundational and delivers the MVP.
- **Phase 4: US2** depends on Foundational and on the node-selection surfaces already used by the current canvas; it can proceed cleanly after US1 establishes the motion-aware canvas shell.
- **Phase 5: US3** depends on Foundational and benefits from US1 and US2 because reduced-motion and responsiveness guarantees must cover both node and panel motion.
- **Phase 6: Polish** depends on the completion of the stories included in the release scope.

### User Story Dependencies

- **US1**: No dependency on other user stories once Foundational work is complete.
- **US2**: Depends on the existing selected-node canvas behavior and should follow US1 so panel transitions are validated against the motion-aware node layer.
- **US3**: Depends on US1 and US2 because safety and responsiveness checks must cover the full animated surface.

### Within Each User Story

- Write tests first and confirm they fail before implementation.
- Keep animation state derived locally from existing authoritative store or canvas state.
- Do not add backend-owned animation timers or animation-only event types.
- Preserve explicit empty, non-selected, loading, and human-readable error states before marking the story complete.
- Do not use Framer Motion `layout` anywhere on the canvas.

## Parallel Opportunities

- `T002` and `T003` can run in parallel during Setup.
- `T005`, `T006`, `T007`, and `T008` can run in parallel during Foundational work after `T004` defines the shared motion primitives.
- `T009`, `T010`, and `T011` can run in parallel for US1 tests.
- `T013`, `T014`, and `T015` can run in parallel after US1 tests are in place; `T016` depends on node motion wiring from `T015`.
- `T019` and `T020` can run in parallel for US2 tests; `T021` depends on the animated canvas and panel test harness already being in place.
- `T026`, `T027`, and `T028` can run in parallel for US3 tests.
- `T029`, `T030`, and `T031` can run in parallel after US3 tests are in place.
- `T033` and `T035` can run in parallel during Polish work.

## Parallel Example: User Story 1

```bash
# Parallel US1 test work
Task: T009 web/src/features/canvas/AgentCanvas.test.tsx
Task: T010 web/src/features/canvas/AnimatedHandoffEdge.test.tsx
Task: T011 web/src/features/canvas/canvasModel.test.ts

# Parallel US1 implementation work
Task: T013 web/src/features/canvas/AnimatedHandoffEdge.tsx
Task: T014 web/src/features/canvas/canvasModel.ts
Task: T015 web/src/features/canvas/AgentCanvasNode.tsx
```

## Parallel Example: User Story 2

```bash
# Parallel US2 test work
Task: T019 web/src/features/canvas/AgentCanvas.test.tsx
Task: T020 web/src/features/canvas/AgentNodeDetailPanel.test.tsx

# Sequential US2 implementation work
Task: T022 web/src/features/canvas/AgentNodeDetailPanel.tsx
Task: T023 web/src/features/canvas/AgentCanvas.tsx + web/src/features/canvas/AgentNodeDetailPanel.tsx + web/src/app/globals.css
```

## Parallel Example: User Story 3

```bash
# Parallel US3 test work
Task: T026 web/src/features/canvas/AgentCanvas.test.tsx
Task: T027 web/src/features/canvas/AgentCanvasNode.test.tsx
Task: T028 web/src/shared/lib/workspace-store.test.ts

# Parallel US3 implementation work
Task: T029 web/src/features/canvas/canvasMotion.ts + web/src/features/canvas/AgentCanvasNode.tsx + web/src/features/canvas/AgentNodeDetailPanel.tsx
Task: T030 web/src/features/canvas/canvasModel.ts
Task: T031 web/src/features/canvas/AgentCanvas.tsx + web/src/features/canvas/AgentCanvasNode.tsx
```

## Implementation Strategy

### MVP First

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Validate node entry, state cross-fades, handoff pulse, and streaming pulse behavior before expanding scope.

### Incremental Delivery

1. Deliver US1 for motion-readable live canvas activity.
2. Add US2 for animated selected-node inspection plus loading and error-state preservation.
3. Add US3 for reduced-motion, cleanup, and responsiveness guarantees.
4. Finish with Polish tasks before release validation.

### Parallel Team Strategy

1. One developer owns shared motion primitives and CSS foundations during Phases 1 and 2.
2. One developer owns node and edge motion for US1 while another prepares panel-transition and error-state tests for US2.
3. After node and panel motion stabilize, one developer focuses on reduced-motion and cleanup guarantees while another runs regression and documentation tasks.

## Notes

- [P] tasks touch separate files or can proceed after a clear prerequisite checkpoint.
- Each user story remains independently testable within the limits described in the specification.
- Keep the motion layer presentation-only: it reads store and canvas state, it never authors orchestration state.
- Preserve keyboard access, visible focus, explicit empty states, loading states, human-readable error states, forced-colors compatibility, and 320px reflow while adding motion.
- If implementation reveals a true protocol gap, keep any backend change minimal and cover it with integration tests before release.