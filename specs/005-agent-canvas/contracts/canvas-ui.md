# UI Contract: Static Agent Canvas

## Purpose

Define the user-visible interaction contract for the isolated agent canvas embedded in Relay’s workspace shell.

## Surface Areas

### Workspace Canvas Region

- Owns the empty state, graph viewport, dev toolbar, and node detail panel.
- Must remain available inside the existing workspace shell layout.
- Must not require any backend bootstrap data beyond what the shell already provides.

### Dev Toolbar

- Provides a visible way to add a node role to the graph.
- Provides a visible way to change the currently selected node’s state.
- Must present disabled or explanatory behavior when a requested action cannot run.

### Node Detail Panel

- Opens when a node is selected.
- Shows the selected node’s display name, role, current state, and local graph context.
- Closes when the user clicks the canvas background or otherwise clears selection.

## Interaction Contract

### Add Node

**Trigger**: User activates the toolbar control to add a role.

**Expected result**:

- A new node appears on the graph with a stable local ID.
- The new node is connected to the intended upstream node.
- The graph recalculates layout after the structural change.
- Existing nodes remain visible during the transition.

### Select Node

**Trigger**: User clicks a rendered node.

**Expected result**:

- The clicked node becomes the selected node.
- The detail panel opens with information derived from that node.
- The selected node remains selected until another node is chosen or selection is cleared.

### Clear Selection

**Trigger**: User clicks the canvas background.

**Expected result**:

- The detail panel closes.
- No node is added, deleted, or moved.
- Viewport interaction remains available.

### Change Node State

**Trigger**: User activates a toolbar control for the selected node.

**Expected result**:

- The selected node updates its visual state immediately.
- The selected node keeps the same position.
- No full-graph layout recalculation occurs.
- No other node disappears or flickers as a side effect.

### Pan And Zoom

**Trigger**: User uses pointer or trackpad interactions supported by the canvas.

**Expected result**:

- The viewport responds without blocking node selection, detail inspection, or toolbar usage.
- Interaction remains available during and after structural layout updates.

## View Model Contract

### Node card fields

- `title`: visible agent name
- `roleBadge`: visible role label
- `stateIndicator`: visible state label and state-specific treatment
- `selected`: whether the node is currently inspected

### Detail panel fields

- `title`: selected node name
- `role`: selected node role
- `state`: selected node state label
- `summary`: local descriptive text
- `incoming`: upstream handoff labels
- `outgoing`: downstream handoff labels

## Invariants

- State-only mutations must not change node coordinates.
- Layout runs only after structural graph changes.
- All nodes retain stable identity across rerenders.
- The isolated canvas must not expose approval controls, live-run status, or backend-connected affordances.
- The canvas must present an explicit empty state before the first node exists.

## Error Contract

- Validation errors are shown inline in plain language within the canvas surface.
- Invalid actions do not partially mutate the graph.
- The detail panel never opens for a node ID that no longer exists.

## Implemented Checkpoints

- The toolbar now requires an explicit role selection before insertion and shows a validation message when the action is incomplete.
- The selected-node controls remain available while the graph grows; adding a downstream node preserves the current selection.
- The detail panel copy explicitly states that the surface is local-only and does not reflect a live Relay run.
- The workspace shell always renders the isolated canvas for an active session, even before any backend activity exists.