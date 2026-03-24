# Feature Specification: Canvas Animation Layer

**Feature Branch**: `007-canvas-animation-layer`  
**Created**: 2026-03-24  
**Status**: Draft  
**Input**: User description: "The animation layer for Relay's multi-agent canvas — Framer Motion transitions, pulsing edge effects, and streaming indicators that make the existing live canvas feel intentional rather than instantaneous. The developer watches agents work and gains spatial awareness through motion: nodes scale in as they spawn, states cross-fade instead of snapping, edges ripple when handoffs fire, and a distinct streaming indicator pulses on a node's border while tokens are actively arriving. The side panel slides in from the right rather than cutting instantly. All animations must use the project easing curve (cubic-bezier(0.16, 1, 0.3, 1)), must not degrade canvas performance, and must never interfere with node state — animation is a presentation layer only, it reads state, it never sets it. New canvas features, additional agent roles, and replay animations are out of scope."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read the live canvas through motion (Priority: P1)

As a developer, I can perceive when agents appear, change status, and hand work off to one another through motion cues so the canvas feels legible and intentional during a live run.

**Why this priority**: The primary value of this feature is improving spatial awareness during live orchestration without changing orchestration behavior. If motion does not clarify live activity, the feature misses its purpose.

**Independent Test**: Can be fully tested by running an existing orchestration flow and confirming that node entry, node state changes, edge handoffs, and active streaming all produce distinct motion cues while the underlying run data remains unchanged.

**Acceptance Scenarios**:

1. **Given** the canvas is showing a live run and a new agent node appears, **When** that node first becomes visible, **Then** it enters with a brief appearance transition rather than popping in instantly.
2. **Given** an existing node changes from one visible state to another, **When** the new state is rendered, **Then** the state presentation changes through a smooth transition rather than an abrupt snap.
3. **Given** work advances from one agent to another, **When** the downstream handoff is shown on the canvas, **Then** the connecting edge displays a short-lived motion effect that makes the handoff noticeable.
4. **Given** an agent is actively receiving visible streamed output, **When** the node is rendered in that period, **Then** the node shows a distinct streaming indicator on its border that stops once streaming ends.

---

### User Story 2 - Inspect agent output without abrupt panel changes (Priority: P2)

As a developer, I can open and switch the side panel with motion that preserves context so I can inspect live output without the interface feeling jarring.

**Why this priority**: The side panel is the main detail surface for the canvas. If it appears and changes abruptly, the live canvas feels disconnected even when node-level motion is clear.

**Independent Test**: Can be fully tested by selecting nodes during a live run, switching between them, and confirming that the side panel enters from the right and updates without abrupt cuts while the selected node data remains correct.

**Acceptance Scenarios**:

1. **Given** no node is selected, **When** the developer selects a node, **Then** the side panel enters from the right with a brief transition instead of appearing instantly.
2. **Given** the side panel is already open for one node, **When** the developer selects another node, **Then** the panel content changes in a way that preserves visual continuity and still reflects the newly selected node's true state.
3. **Given** the selected node is actively streaming output, **When** new output arrives, **Then** the panel continues showing the correct live content without visual interruption or stale selection state.

---

### User Story 3 - Keep motion safe, responsive, and truthful (Priority: P3)

As a developer, I can trust that canvas animations are only visual polish and never alter orchestration behavior or degrade interaction performance.

**Why this priority**: Relay is an operator-facing tool. Motion that lags, blocks interaction, or implies false state would reduce trust in the live orchestration view.

**Independent Test**: Can be fully tested by exercising a live run with frequent state changes while panning, zooming, selecting nodes, and reopening the side panel, then confirming interaction stays responsive and visible state always matches the underlying run state.

**Acceptance Scenarios**:

1. **Given** multiple visible node and edge transitions occur during a live run, **When** the developer pans, zooms, or clicks the canvas, **Then** those interactions remain responsive and predictable.
2. **Given** an animation is in progress, **When** the underlying run state changes again, **Then** the canvas resolves to the latest real state without requiring the motion layer to set or infer state.
3. **Given** the developer prefers reduced motion, **When** the live canvas is shown, **Then** non-essential movement is minimized while the interface still communicates agent status and streaming activity clearly.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature is presentation-only. It reads the existing orchestration and canvas state produced by Relay's current handler -> orchestrator -> agent -> tool -> storage flow and does not add a new execution path or alter backend orchestration rules.
- Frontend work must remain within the existing feature-based canvas and side-panel areas rather than introducing new root-level type-based folders.
- No new backend event type is required if the current live canvas already exposes spawn, state, handoff eligibility, selection, and streaming activity. If the existing state model lacks an explicit streaming-activity flag or handoff timing signal, only the minimum protocol extension needed to expose those existing facts may be added, with integration coverage for the affected event flow.

### Approval and Safety Impact

- This feature does not introduce file writes, shell command execution, or any new approval path. Existing handler-level approval rules remain unchanged.
- This feature does not change agent tool access, file-system behavior, or shell behavior. It must not imply broader agent capabilities than the current orchestration mode already permits.

### UX States

- The canvas must preserve visible idle, active, complete, error, and streaming states, with motion layered on top of those states rather than replacing them.
- The side panel must preserve explicit closed, opening, open, switching, and empty-selection states, with human-readable errors remaining plain language if agent output is unavailable.
- Empty and non-selected states must remain explicit and helpful rather than being hidden behind motion.

### Edge Cases

- If a node spawns and reaches another state almost immediately, the motion layer must still settle on the latest visible state without showing stale intermediate status.
- If several handoffs occur close together, each affected edge must remain readable and must not create a persistent distracting effect.
- If streaming starts and stops rapidly for the same node, the streaming indicator must reflect only active streaming periods and must not continue pulsing after streaming ends.
- If the developer rapidly changes node selection while the side panel is animating, the panel must end on the latest selected node without showing mixed content.
- If the canvas is reopened from an already completed run, final states must remain readable even though replay-specific animation is out of scope for this feature.
- If user motion preferences request reduced movement, the interface must preserve meaning while minimizing non-essential animation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST animate newly visible agent nodes with a brief entry transition rather than showing them instantaneously.
- **FR-002**: The system MUST animate visible node state changes so that state presentation transitions smoothly instead of snapping between states.
- **FR-003**: The system MUST show a short-lived motion cue on the relevant connection when work visibly hands off from one agent to another.
- **FR-004**: The system MUST show a distinct node-level streaming indicator while that node is actively receiving visible streamed output.
- **FR-005**: The system MUST stop the streaming indicator when that node is no longer actively receiving visible streamed output.
- **FR-006**: The system MUST open the side panel with a right-origin transition when a node is selected from the canvas.
- **FR-007**: The system MUST preserve visual continuity when the selected node changes while the side panel is already open.
- **FR-008**: The system MUST ensure all animation timing and easing used by this feature follow Relay's shared motion curve.
- **FR-009**: The system MUST treat animation as a presentation layer that reads state and MUST NOT let animation start, stop, infer, or modify orchestration state.
- **FR-010**: The system MUST resolve visible presentation to the latest underlying run state when new state arrives during an in-progress animation.
- **FR-011**: The system MUST preserve canvas responsiveness for pan, zoom, and selection while live node, edge, and panel animations are active.
- **FR-012**: The system MUST keep motion effects bounded in duration so they communicate activity without becoming persistent distractions.
- **FR-013**: The system MUST preserve the existing meaning of idle, active, complete, error, and streaming states rather than replacing those distinctions with animation alone.
- **FR-014**: The system MUST preserve explicit empty and non-selected states for the canvas and side panel where they already exist.
- **FR-015**: The system MUST minimize non-essential movement when the user prefers reduced motion while preserving clear state communication.
- **FR-016**: The system MUST NOT add new canvas behaviors, agent roles, or replay-specific motion as part of this feature.
- **FR-017**: The system MUST apply motion only to existing live-canvas behaviors that already exist in the product scope.
- **FR-018**: The system MUST ensure a node's visible animated state always corresponds to the same agent session shown in the panel and canvas selection model.
- **FR-019**: The system MUST ensure side-panel motion never causes the selected node's transcript or status to display stale or mixed data.
- **FR-020**: The system MUST preserve readability of node borders, labels, and status cues while animation is active.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST NOT bypass handler-level approval enforcement for file writes or shell commands.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and current repo-scoped safety boundaries without introducing a second runtime transport for motion state.
- **CDR-003**: The feature MUST include tests covering node entry motion, state-transition motion, side-panel transition behavior, streaming-indicator behavior, reduced-motion behavior, and interaction responsiveness during active animation; if protocol fields change to expose animation-relevant state, integration coverage is also required.
- **CDR-004**: The feature MUST preserve visible loading states, human-readable error states, and explicit empty states for all affected canvas and panel flows.
- **CDR-005**: The feature MUST document any new dependency introduced for animation presentation and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Canvas Motion State**: The presentation-only interpretation of existing live canvas state that determines whether a node, edge, or panel should appear, transition, pulse, or rest.
- **Streaming Activity Signal**: The live indicator that a specific agent session is actively receiving visible streamed output and should show a streaming cue.
- **Handoff Highlight Window**: The short-lived period during which a connection visually emphasizes that work has moved from one agent to another.
- **Panel Transition Context**: The selected-node presentation state that determines whether the side panel is closed, entering, open, or switching while preserving correct content.
- **Reduced-Motion Preference**: The user preference that limits non-essential movement while preserving the semantic clarity of canvas state changes.

## Assumptions

- The current live canvas already exposes enough state to tell when a node appears, when a node changes status, and when a node is actively streaming output.
- Existing orchestration behavior, node layout, and side-panel content structure remain unchanged in this feature.
- Handoff visibility can be derived from the existing orchestration sequence or from minimal additional metadata that describes already-existing transitions.
- Reduced-motion support should preserve meaning through shorter or less dynamic transitions rather than removing state cues entirely.

## Dependencies

- The existing live agent canvas and side panel must already be capable of rendering stable node identities, node states, edges, and selected-node details.
- The current live orchestration state model must expose the timing of visible node lifecycle changes and streaming activity closely enough for the presentation layer to react.
- Existing design tokens and shared motion conventions must remain the source of truth for timing and easing.

## Out of Scope

- New canvas functionality beyond motion treatment for existing live behavior
- Additional agent roles or changes to orchestration order
- Replay-specific animation for previously completed runs
- Changes to agent capabilities, approvals, file writes, or shell execution
- New backend orchestration logic that exists only to drive decorative motion

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In at least 95% of local validation runs, newly spawned nodes begin their visible entry transition within 100ms of the node becoming visible on the live canvas.
- **SC-002**: In 100% of validated state-transition cases, the final visible node state matches the latest underlying run state even when multiple state changes occur in quick succession.
- **SC-003**: In at least 95% of validation trials during active live motion, developers can pan, zoom, and change node selection without noticeable interaction delay beyond 100ms.
- **SC-004**: In 100% of validated streaming cases, the streaming indicator is present only while visible output is actively arriving for that node and stops once streaming is no longer active.
- **SC-005**: In 100% of reduced-motion validation checks, the live canvas preserves state clarity without relying on full-motion effects.
