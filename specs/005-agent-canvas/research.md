# Research: Static Agent Canvas

## Decision 1: Use `@xyflow/react` in controlled mode for the isolated canvas

- Decision: Build the canvas on `@xyflow/react` using controlled `nodes` and `edges` collections, custom `nodeTypes`, and standard viewport controls for pan and zoom.
- Rationale: Current React Flow guidance supports custom node components and controlled state via `useNodesState` and `useEdgesState`, which fits the spec’s requirement for immediate visual state mutation, stable node identity, and uninterrupted canvas interaction. The library also provides built-in selection, viewport management, and edge rendering that would be expensive to recreate inside Relay.
- Alternatives considered:
  - Build the canvas with plain absolutely positioned divs and SVG lines: rejected because pan, zoom, hit testing, selection, and future expansion would become custom infrastructure work.
  - Use a more general diagramming library: rejected because the feature only needs node-graph interaction, not a full whiteboard or design surface.

## Decision 2: Use dagre for directed layout and run it only on structural graph changes

- Decision: Use `@dagrejs/dagre` to calculate node positions whenever a node or edge is added, and preserve the existing coordinates when only node metadata or node state changes.
- Rationale: Dagre’s API is purpose-built for client-side directed graph layout using explicit node width, height, and edge relationships. It produces stable `{ x, y }` positions after `setNode`, `setEdge`, and `layout`, which maps cleanly to React Flow node positions. Separating structure changes from state-only updates is the simplest way to guarantee FR-007 and FR-008.
- Alternatives considered:
  - Re-run layout on every render: rejected because it would violate the non-functional requirement that state changes must not reposition nodes.
  - Use manual placement rules: rejected because they would become fragile once branches or longer workflows are added.

## Decision 3: Keep canvas state local to `features/canvas` instead of the workspace store

- Decision: Model the graph, selection state, and toolbar actions inside a feature-local controller or reducer under `features/canvas`, and pass only the minimum props from `WorkspaceCanvas`.
- Rationale: The spec explicitly requires backend isolation. The global workspace store is WebSocket-oriented and optimized for live Relay runtime data. Reusing it for this feature would create unnecessary coupling and risk implying backend-connected behavior where none exists.
- Alternatives considered:
  - Put the canvas into the shared workspace store: rejected because it mixes local prototype state with runtime Relay state and makes later separation harder.
  - Scatter independent `useState` hooks across multiple components: rejected because the relayout and selection invariants are easier to enforce from a single state transition boundary.

## Decision 4: Use stable node IDs plus a monotonic layout revision counter

- Decision: Assign each node a stable local ID at creation time and track a separate layout revision or structure signature so layout work only occurs when structure changes.
- Rationale: The spec requires the same logical node to preserve identity across selection, detail inspection, and visual state changes. Stable IDs also make it straightforward to compare previous and next structures in tests and assert that state-only updates do not alter node coordinates.
- Alternatives considered:
  - Derive IDs from array index: rejected because adding nodes would reindex earlier nodes and break selection stability.
  - Infer relayout needs from deep object equality of the whole node array: rejected because node-state changes would look like structural changes.

## Decision 5: Reuse existing Relay visual tokens and state semantics rather than inventing a separate canvas palette

- Decision: Build canvas nodes and detail surfaces on the existing dark-mode token set in `globals.css`, reusing the established glow semantics for idle, thinking, complete, and error, and extending them for the executing state.
- Rationale: Relay already defines the product’s visual language and state glow contract in CSS tokens. Reusing those tokens keeps the isolated canvas visually consistent with the rest of the workspace and avoids introducing raw colors or duplicate state semantics.
- Alternatives considered:
  - Create a separate canvas-only palette: rejected because it would diverge from Relay’s design rules and add unnecessary styling maintenance.
  - Reuse only generic border states with no glow: rejected because the feature explicitly calls for distinct visual treatment by node state.

## Decision 6: Treat the detail panel as a sibling side panel, not an in-node popover

- Decision: Open node details in a dedicated side panel within the canvas surface and close it when the user clicks the canvas background or otherwise clears selection.
- Rationale: The spec explicitly requests a side panel. A sibling panel is also more accessible and stable than an overlay anchored to a moving node, especially while the graph can pan and zoom.
- Alternatives considered:
  - Put details into a floating node toolbar: rejected because it would compete with zoom and selection behavior and would not satisfy the requested layout.
  - Open details in a modal dialog: rejected because the canvas must remain interactive while inspecting nodes.

## Decision 7: Cover viewport and relayout invariants with component tests instead of relying on visual inspection alone

- Decision: Add focused component tests that verify node rendering, selection and close behavior, toolbar-triggered state changes, and stable positions when only node state mutates.
- Rationale: The key risks in this feature are regression-oriented: invisible node drops, unintentional relayout, and lost interactivity. Those are all better guarded by deterministic tests around model transitions and rendered output than by manual review alone.
- Alternatives considered:
  - Rely only on manual validation: rejected because the main failure modes are subtle regressions that are easy to reintroduce.
  - Add only snapshot tests: rejected because the important behavior is interactive and stateful rather than static markup.

## Implementation Checkpoints

- User Story 1 is implemented with a controlled React Flow surface, custom agent nodes, a local add-node toolbar, and dagre-driven left-to-right layout.
- User Story 2 is implemented with a sibling detail panel, background-click deselection, and state-only mutations that preserve node coordinates.
- User Story 3 is implemented with persistent viewport controls, stable node IDs, and structural-only relayout triggered during node insertion.
- The finished frontend validation suite covers empty-state replacement, graph structure updates, selection and detail rendering, local state mutation, and workspace-shell integration.