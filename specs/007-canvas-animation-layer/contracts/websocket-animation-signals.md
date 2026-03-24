# Contract: WebSocket Animation Signals for the Canvas

## Purpose

Define how the existing live-run WebSocket protocol maps to the new canvas animation layer. This feature does not introduce animation-specific transport messages. The browser derives animation from existing orchestration events and existing canvas selection state.

## Transport Boundary

- Path: `/ws`
- Message envelope: unchanged existing workspace envelope
- Ownership: backend emits orchestration facts, frontend derives motion state

## Event-to-Animation Mapping

### `agent_spawned`

- Existing meaning: a new agent node now exists in the run.
- Animation mapping: append a node to the canvas document, rerun layout once, and start the node's enter animation.
- New transport requirement: none.

### `agent_state_changed`

- Existing meaning: the visible state of one agent changed.
- Animation mapping: patch the corresponding node's authoritative state and animate the node's state presentation to the latest state.
- New transport requirement: none, provided `agent_id` and `state` remain present.

### `token`

- Existing meaning: one visible chunk of streamed output arrived for an agent.
- Animation mapping: append transcript text as today and refresh the node's local `lastTokenAt` timestamp so the streaming border pulse remains active for 300ms after the most recent token.
- New transport requirement: none, provided `agent_id`, `text`, and ordered delivery semantics remain unchanged.

### `handoff_start`

- Existing meaning: a dependency handoff has started between two agents.
- Animation mapping: set the edge presentation data for the `from_agent_id -> to_agent_id` edge to an active pulse state.
- New transport requirement: none, provided both agent identifiers remain present.

### `handoff_complete`

- Existing meaning: the handoff is completed and the downstream agent can continue.
- Animation mapping: clear or settle the same edge pulse state so the edge returns to its resting appearance.
- New transport requirement: none.

### `run_complete` and `run_error`

- Existing meaning: the run reached a terminal state.
- Animation mapping: stop any animation state that depends on live streaming or active handoff progression, while preserving final visual state.
- New transport requirement: none.

## Derived UI Contracts

### Node Renderer Contract

The node renderer receives:

```ts
{
  id: string;
  state: "queued" | "assigned" | "thinking" | "streaming" | "completed" | "clarification_required" | "errored" | "cancelled" | "blocked";
  selected: boolean;
  summary: string;
  streamingActive: boolean;
}
```

Rules:

- `state` remains authoritative.
- `streamingActive` is derived locally and cannot be persisted or sent back to the backend.
- The node renderer may animate only presentation properties.

### Edge Renderer Contract

The edge renderer receives:

```ts
{
  id: string;
  source: string;
  target: string;
  data: {
    pulseState: "idle" | "active" | "settling";
  };
}
```

Rules:

- The edge renderer must not subscribe to workspace events directly.
- `pulseState` is derived upstream in the canvas model or store mapping layer.

### Detail Panel Contract

The detail panel receives either a selected node view or the empty-selection view. Presence animation is keyed from current selection and never changes backend state.

## Non-Goals

- No animation-only WebSocket message types
- No backend-owned timers for border pulses or panel motion
- No replay-specific animation contract in this feature