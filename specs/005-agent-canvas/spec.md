# Feature Specification: Static Agent Canvas

**Feature Branch**: `005-agent-canvas`  
**Created**: 2026-03-24  
**Status**: Draft  
**Input**: User description: "A static, interactive agent canvas for Relay — a React Flow graph that renders agent nodes with live visual states and a per-node detail panel, built in complete isolation from the backend."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Build a readable agent graph (Priority: P1)

As a developer, I can add agent nodes to a visual canvas and immediately see them arranged as a readable directed workflow so I can understand the handoff order between roles without manually placing nodes.

**Why this priority**: The feature’s primary value is the canvas itself. If developers cannot add nodes and see a coherent graph with directional handoff, the feature does not exist in a useful form.

**Independent Test**: Can be fully tested by starting with an empty canvas, adding a Planner node and then a Coder node, and confirming both nodes appear with a directional connection and a clean automatic layout.

**Acceptance Scenarios**:

1. **Given** an empty canvas, **When** the developer adds a Planner node, **Then** the node appears in a visible default position with its name, role badge, and current state shown.
2. **Given** a canvas with a Planner node, **When** the developer adds a Coder node as the next role, **Then** the canvas shows a directional connection from Planner to Coder and positions both nodes in a readable workflow layout.
3. **Given** a canvas with existing nodes, **When** the developer adds a Tester node, **Then** the canvas recalculates positions for the full graph and preserves directional handoff clarity for all visible nodes.

---

### User Story 2 - Inspect and update a node in place (Priority: P2)

As a developer, I can select a node to inspect its details and change its state from a development control surface so I can validate visual behavior without losing the node’s place on the canvas.

**Why this priority**: Once the graph exists, the next most important value is being able to verify node-level behavior and state transitions without involving backend events.

**Independent Test**: Can be fully tested by selecting the Coder node, opening its detail panel, changing its state to thinking, and confirming the node updates instantly while staying in the same position.

**Acceptance Scenarios**:

1. **Given** a populated canvas, **When** the developer clicks a node, **Then** a side panel opens showing that node’s details.
2. **Given** a node detail panel is open, **When** the developer clicks outside the selected node and panel, **Then** the panel closes.
3. **Given** a selected node, **When** the developer changes its state through the development controls, **Then** the node’s visual treatment updates immediately and the node remains visible in the same position.

---

### User Story 3 - Keep the canvas interactive during updates (Priority: P3)

As a developer, I can keep panning, zooming, and clicking nodes while new nodes are added or states change so I can evaluate the canvas as an interactive design surface rather than a static screenshot.

**Why this priority**: Smooth interaction under change is a critical quality requirement, but it depends on the graph and node details already existing.

**Independent Test**: Can be fully tested by interacting with the canvas while adding a node and while changing a node state, then confirming that zoom, pan, and click interactions continue working without nodes disappearing or flickering.

**Acceptance Scenarios**:

1. **Given** a canvas with multiple nodes, **When** the developer zooms or pans during node creation, **Then** the canvas remains responsive and the interaction completes normally.
2. **Given** a canvas with multiple nodes, **When** the developer changes a node state, **Then** the canvas remains interactive and no other node disappears, jumps, or flickers.
3. **Given** an automatic layout update caused by a newly added node, **When** the repositioning completes, **Then** the resulting graph remains stable and selectable.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature is intentionally isolated from the backend and does not change the handler -> orchestrator -> agent -> tool -> storage flow. It acts as a frontend-only simulation surface for agent graph behavior.
- Frontend work must stay within feature-based areas for the canvas, agent panel, and shared UI primitives rather than introducing new type-based top-level folders.
- This feature does not add or modify WebSocket events, backend persistence, or tool execution flows. Integration coverage is not required for backend protocols because all state is local to the frontend simulation.

### Approval and Safety Impact

- This feature does not introduce file writes, shell command execution, or agent tool access. No new approval path is required because the feature does not cross into governed backend actions.
- This feature must not imply that simulated node states reflect real backend execution. The UI must remain clearly local and development-oriented so it does not weaken existing sandboxing or approval expectations elsewhere in Relay.

### UX States

- The canvas must show an explicit empty state before the first node is added.
- The node detail panel must show a clear selected state when open and close cleanly when focus returns to the canvas background.
- Development controls must present clear, human-readable state options for idle, thinking, executing, complete, and error.
- If a node cannot be added because required node information is incomplete, the interface must show an inline, human-readable error and keep the current canvas unchanged.

### Edge Cases

- If the developer changes a node state repeatedly in quick succession, the final selected state must be the one that remains visible without intermediate flicker causing the node to disappear.
- If the developer adds a node while another node is selected, the selected node must remain selected unless the developer explicitly changes selection.
- If the graph contains only one node, the canvas must still support selection, detail viewing, zoom, and pan.
- If the developer closes the panel by clicking the canvas background, the canvas must not treat that click as a request to add, delete, or move a node.
- If automatic layout places nodes farther apart after a new node is added, all nodes and edges must remain visible or reachable through normal pan and zoom interaction.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a visual canvas that represents agent workflows as connected nodes and directed edges.
- **FR-002**: The system MUST allow a developer to add a new agent node to the canvas from a development control surface.
- **FR-003**: The system MUST render each node with the agent’s display name, role badge, and current state.
- **FR-004**: The system MUST provide a distinct visual treatment for the idle, thinking, executing, complete, and error states.
- **FR-005**: The system MUST create a directional connection between related nodes so the work handoff order is visually understandable.
- **FR-006**: The system MUST automatically position nodes in a readable directed layout when a new node is added.
- **FR-007**: The system MUST recalculate node positions only when the graph structure changes through node addition or connection changes.
- **FR-008**: The system MUST NOT recalculate node positions when only a node’s visual state changes.
- **FR-009**: The system MUST preserve each node’s visibility during state changes so a node never disappears as a result of a visual state update.
- **FR-010**: The system MUST allow a developer to click a node to open a side panel showing that node’s details.
- **FR-011**: The system MUST close the node detail panel when the developer clicks away from the active selection area.
- **FR-012**: The system MUST provide development controls that can programmatically change a selected node’s state.
- **FR-013**: The system MUST update the selected node’s visual state immediately after a programmatic state change.
- **FR-014**: The system MUST preserve the selected node’s canvas position during a state-only update.
- **FR-015**: The system MUST allow zooming, panning, and node selection throughout node creation and node state changes.
- **FR-016**: The system MUST keep the canvas interactive while automatic layout is applied after a new node is added.
- **FR-017**: The system MUST provide an explicit empty-state experience before any nodes exist on the canvas.
- **FR-018**: The system MUST behave entirely without backend data, backend events, or backend persistence.
- **FR-019**: The system MUST make clear in the interface that node data and node state mutations are local to the isolated canvas experience.
- **FR-020**: The system MUST preserve stable node identity so a node’s details, selection state, and rendered position remain associated with the same logical node across visual updates.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST remain frontend-only and MUST NOT introduce backend handlers, orchestrator calls, agent execution, tool execution, or storage changes.
- **CDR-002**: The feature MUST preserve Relay’s feature-based frontend organization and place canvas-related UI under the existing canvas and agent-panel feature areas or shared UI libraries where appropriate.
- **CDR-003**: The feature MUST include automated frontend tests for custom node rendering, node selection behavior, detail panel open and close behavior, state mutation behavior, and stability of node positioning during state-only updates.
- **CDR-004**: The feature MUST define visible empty, selected, and validation error states for the isolated canvas experience.
- **CDR-005**: The feature MUST document any new frontend dependency introduced for graph layout or canvas rendering and update the Tech Stack note in project documentation in the same change.
- **CDR-006**: The feature MUST avoid introducing UI affordances that imply real execution approval, live streaming, or backend-connected agent activity in this isolated experience.

### Key Entities *(include if feature involves data)*

- **Agent Node**: A visual representation of one agent role on the canvas, including display name, role label, current state, stable identity, and position.
- **Node Connection**: A directional relationship between two agent nodes that communicates work handoff order.
- **Canvas Graph**: The complete set of nodes and directional connections currently shown in the isolated experience.
- **Node Detail View**: The side panel representation of the currently selected node, including its descriptive details and editable local state.
- **Local Node State**: The development-controlled state value assigned to a node for visual testing in the isolated canvas.
- **Layout Snapshot**: The current arrangement of nodes on the canvas after the most recent structure-changing layout pass.

## Assumptions

- The isolated canvas is intended for local design and interaction validation rather than for real agent execution.
- Developers add nodes through a simple development-oriented control surface rather than by dragging from an external catalog.
- Node addition follows a logical workflow order, with each newly added node connected according to the developer’s chosen predecessor or default sequence.
- Node detail content is limited to information already available in the local canvas model and does not require fetching remote metadata.
- The initial version supports adding and updating nodes, but not manually dragging nodes to override the automatic layout.

## Dependencies

- Relay’s frontend workspace must already support the shared shell and canvas styling foundations needed for the isolated graph view.
- Any graph rendering or layout dependency selected for this feature must be compatible with Relay’s existing frontend stack and testing approach.

## Out of Scope

- Real backend-driven agent execution
- WebSocket-driven live updates
- Persisting canvas state across sessions
- Saving or loading agent graphs from disk
- Multi-user collaboration on the canvas
- Manual node placement as the primary layout mechanism
- Editing backend agent definitions, prompts, or orchestration rules

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In manual validation, 100% of newly added nodes appear on the canvas with a readable connection path and without requiring manual repositioning.
- **SC-002**: In manual validation, 100% of state-only updates preserve the selected node’s visible position.
- **SC-003**: In manual validation, developers can open the correct detail panel for a clicked node and close it again by clicking outside the selection area in 100% of tested attempts.
- **SC-004**: During local interaction testing, the canvas remains responsive to pan, zoom, and click input throughout node addition and state mutation flows in at least 95% of attempts.
- **SC-005**: No visible node disappearance or flicker is observed during acceptance testing for repeated state changes or for adding one new node to an existing graph.
