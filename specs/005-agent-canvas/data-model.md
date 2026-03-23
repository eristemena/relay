# Data Model: Static Agent Canvas

## Overview

This feature has no backend persistence. The data model describes the local in-memory state required to render, inspect, and mutate the isolated agent canvas.

## Agent Canvas Document

The top-level local model owned by the canvas feature.

| Field | Type | Rules |
|-------|------|-------|
| `nodes` | `AgentCanvasNode[]` | Ordered collection of visible graph nodes |
| `edges` | `AgentCanvasEdge[]` | Ordered collection of directed connections |
| `selectedNodeId` | `string \| null` | Points to the currently inspected node or `null` when no node is selected |
| `layoutRevision` | `number` | Increments only when graph structure changes |
| `validationMessage` | `string \| null` | Human-readable error shown when an add action cannot be completed |

### Validation

- `layoutRevision` changes only when a node or edge is added, removed, or rewired.
- `selectedNodeId` must reference an existing node or be `null`.
- `validationMessage` must never contain implementation-only diagnostics.

## Agent Canvas Node

Represents one logical agent on the canvas.

| Field | Type | Rules |
|-------|------|-------|
| `id` | `string` | Stable local identifier assigned at creation time |
| `role` | enum | One of `planner`, `coder`, `reviewer`, `tester`, `explainer` |
| `label` | `string` | Human-readable node title shown on the card |
| `state` | enum | One of `idle`, `thinking`, `executing`, `complete`, `error` |
| `details` | `AgentNodeDetails` | Display-safe descriptive content for the detail panel |
| `position` | `{ x: number; y: number }` | Render position last produced by layout |
| `size` | `{ width: number; height: number }` | Layout input used to compute stable directed placement |

### Validation

- `id` is immutable after creation.
- `position` may change only after a structure-changing layout pass.
- `label` should default from `role` if not otherwise customized.
- `size` values must remain consistent across state-only changes to avoid unintended relayout.

## Agent Node Details

The display model rendered in the side panel.

| Field | Type | Rules |
|-------|------|-------|
| `summary` | `string` | Plain-language explanation of the role’s purpose in the graph |
| `currentStateLabel` | `string` | Human-readable label derived from the node’s current state |
| `incomingFrom` | `string[]` | Labels of upstream nodes |
| `outgoingTo` | `string[]` | Labels of downstream nodes |

### Validation

- All detail fields are derived from local graph state and must be recomputed when structure changes.
- `incomingFrom` and `outgoingTo` must remain synchronized with the edge set.

## Agent Canvas Edge

Represents a directed handoff between two nodes.

| Field | Type | Rules |
|-------|------|-------|
| `id` | `string` | Stable local identifier |
| `sourceNodeId` | `string` | Must reference an existing node |
| `targetNodeId` | `string` | Must reference an existing node |
| `kind` | enum | Initial version uses `handoff` only |

### Validation

- `sourceNodeId` and `targetNodeId` must reference different nodes.
- Duplicate handoff edges between the same pair should be avoided in the initial version.

## Toolbar Draft Action

Represents the temporary local intent from the dev toolbar.

| Field | Type | Rules |
|-------|------|-------|
| `action` | enum | One of `add-node`, `change-state`, `clear-selection` |
| `targetNodeId` | `string \| null` | Required for `change-state`; omitted for `add-node` |
| `role` | enum nullable | Required when adding a node |
| `state` | enum nullable | Required when changing node state |

### Validation

- `change-state` requires a selected node.
- `add-node` requires a valid role and a determinable parent or insertion strategy.

## State Transitions

### Canvas lifecycle

- Empty canvas -> first node added -> populated canvas
- Populated canvas + node selected -> detail panel open
- Detail panel open + background click -> selection cleared

### Node state lifecycle

- `idle` -> `thinking`
- `thinking` -> `executing`
- `executing` -> `complete`
- Any non-terminal state -> `error`
- Terminal states may be reassigned by the toolbar because this is a developer-only simulation surface

## Derived Views

### Layout Snapshot

Computed projection of nodes and edges after the last structure-changing layout pass.

| Field | Type | Rules |
|-------|------|-------|
| `nodePositions` | map | Keyed by node ID, used to assert state-only stability |
| `bounds` | `{ width: number; height: number }` | Optional viewport-fit metadata |

### Selected Node View

Computed detail projection for the side panel.

| Field | Type | Rules |
|-------|------|-------|
| `nodeId` | `string` | Matches `selectedNodeId` |
| `title` | `string` | Rendered heading for the detail panel |
| `roleLabel` | `string` | Display-safe role text |
| `stateLabel` | `string` | Display-safe state text |
| `summary` | `string` | Human-readable details content |

## Implementation Notes

- Node IDs are monotonic local identifiers (`node_1`, `node_2`, ...) so selection and detail state stay attached to the same logical node across rerenders.
- The current implementation creates one directed handoff edge from the previously inserted node to the newly inserted node.
- Structural changes run through dagre layout before detail fields are synchronized; state-only mutations update labels and details while preserving the existing `position` values.
- The detail panel remains a derived sibling view rather than owning any independent state beyond the selected node reference.