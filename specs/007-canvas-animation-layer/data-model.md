# Data Model: Canvas Animation Layer

## Scope

This feature does not add new persisted backend records. It introduces presentation-only frontend state derived from the existing live canvas document and WebSocket event stream.

## Animated Canvas Node View

Represents one React Flow node plus the transient animation metadata needed to render it safely.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `id` | text | Existing canvas node | Stable node key; must match the underlying `agent_id` |
| `state` | enum | Existing canvas node | Authoritative visible node state from the canvas model |
| `selected` | boolean | Existing canvas node | Mirrors current node selection only |
| `summary` | text | Existing canvas node | Display copy only |
| `motionPhase` | enum | Derived | One of `entering`, `steady`, `exiting`; presentation-only |
| `streamingActive` | boolean | Derived | True only while token silence window has not expired |
| `lastTokenAt` | number nullable | Local ref | Timestamp of the most recent token seen by the mounted node |

### Rules

- `state` remains the source of truth for status; `motionPhase` cannot override it.
- `streamingActive` must become false within 300ms after the latest visible token if no newer token arrives.
- Unmounting a node must clear any timer associated with `lastTokenAt`.

## Animated Handoff Edge View

Represents one handoff edge plus its pulse-specific presentation data.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `id` | text | Existing edge model | Stable `from->to` identifier |
| `sourceNodeId` | text | Existing edge model | Existing upstream node key |
| `targetNodeId` | text | Existing edge model | Existing downstream node key |
| `kind` | enum | Existing edge model | `handoff` |
| `pulseState` | enum | Derived | One of `idle`, `active`, `settling` |
| `lastHandoffAt` | string nullable | Derived from event | Optional timestamp of the latest handoff start |

### Rules

- `pulseState` is derived from `handoff_start` and `handoff_complete` and is never authored directly by the edge component.
- Duplicate handoff events for the same edge must update the same edge record, not create a second visual edge.

## Detail Panel Presence State

Represents the side panel's current presence mode for selection-driven transitions.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `selectedNodeId` | text nullable | Existing canvas document | Determines whether the detail panel is showing a node or the empty-selection view |
| `panelMode` | enum | Derived | One of `empty`, `entering`, `open`, `switching`, `exiting` |
| `contentKey` | text | Derived | Stable key for `AnimatePresence` and selection changes |

### Rules

- `contentKey` must change when the selected node changes so panel content transitions apply to the latest selection.
- Panel motion cannot alter the selected node record in the workspace store.

## Motion Preset

Represents the shared motion constants used across the feature.

| Field | Type | Value | Rules |
|-------|------|-------|-------|
| `durationMs` | number | `300` | Shared default duration for node and panel transitions |
| `easing` | string | `cubic-bezier(0.16, 1, 0.3, 1)` | Must be reused without ad-hoc variants |
| `panelEnterX` | number | `380` | Panel starts off-canvas to the right |
| `panelExitX` | number | `380` | Panel exits to the right |
| `streamingSilenceMs` | number | `300` | Streaming pulse clears after silence window |

### Rules

- Shared presets must live in one frontend-owned source of truth rather than being repeated as literal values across components.
- Any reduced-motion alternative must preserve semantic state indicators even if transform intensity is reduced.

## Reduced Motion Preference

Represents the user or browser preference that changes how motion is applied.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `reducedMotion` | boolean | Browser preference | Read-only input into the presentation layer |

### Rules

- When `reducedMotion` is true, non-essential transforms should be minimized or skipped.
- The interface must still show visible state distinctions through borders, labels, and status text.