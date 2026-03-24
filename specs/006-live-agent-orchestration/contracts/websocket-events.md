# WebSocket Contract: Live Agent Orchestration

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Always served by Go in both development and production

This feature extends the existing workspace and live-run protocol with orchestration-mode events. Bootstrap, session, preferences, and run-open requests remain valid. The orchestration mode adds new event types while keeping the same shared envelope.

## Envelope

All messages continue to use the shared JSON envelope.

```json
{
  "type": "agent.run.submit",
  "request_id": "req-orch-1",
  "payload": {
    "session_id": "session_123",
    "task": "Design a rollout plan for a live orchestration graph"
  }
}
```

## Client -> Server Messages

### `workspace.bootstrap.request`

Unchanged. The bootstrap response now includes orchestration-aware run summaries and any active orchestration run ID.

### `agent.run.submit`

Starts one orchestration-mode run for the given session using the submitted goal text.

### `agent.run.open`

Requests replay of a previously saved orchestration run and reattaches to live delivery if that run is still active.

## Server -> Client Messages

### `workspace.bootstrap`

Unchanged envelope shape, but the selected run summary may now represent an orchestration run rather than a single-agent run.

If the payload includes `active_run_id`, the frontend may issue one `agent.run.open` request for that run to replay stored events and reattach to the active stream.

### `agent_spawned`

Emitted when the orchestrator creates a new agent node.

```json
{
  "type": "agent_spawned",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_planner_1",
    "sequence": 1,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "label": "Planner",
    "spawn_order": 1,
    "occurred_at": "2026-03-24T12:00:00Z"
  }
}
```

### `agent_state_changed`

Emitted when one agent changes visible state.

```json
{
  "type": "agent_state_changed",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_planner_1",
    "sequence": 2,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "state": "thinking",
    "message": "Planner is breaking the goal into stages.",
    "occurred_at": "2026-03-24T12:00:00Z"
  }
}
```

### `task_assigned`

Emitted when the coordinator assigns prompt text to one agent.

```json
{
  "type": "task_assigned",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_coder_1",
    "sequence": 5,
    "replay": false,
    "role": "coder",
    "model": "anthropic/claude-sonnet-4-5",
    "task_text": "Draft the implementation approach based on the planner output.",
    "occurred_at": "2026-03-24T12:00:02Z"
  }
}
```

### `handoff_start`

Emitted when a dependency handoff begins.

```json
{
  "type": "handoff_start",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_planner_1",
    "sequence": 6,
    "replay": false,
    "from_agent_id": "agent_planner_1",
    "to_agent_id": "agent_coder_1",
    "reason": "planner_completed",
    "occurred_at": "2026-03-24T12:00:03Z"
  }
}
```

### `handoff_complete`

Emitted when a dependency handoff is fully registered and the downstream agent is eligible to continue.

### `token`

Emitted for one visible chunk of transcript text for an agent.

```json
{
  "type": "token",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_tester_1",
    "sequence": 9,
    "replay": false,
    "role": "tester",
    "model": "deepseek/deepseek-chat",
    "text": "I will validate the plan against the current constraints.",
    "occurred_at": "2026-03-24T12:00:04Z"
  }
}
```

### `agent_error`

Emitted when one agent reaches an errored terminal state but the run itself has not yet been declared unrecoverable.

```json
{
  "type": "agent_error",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_tester_1",
    "sequence": 12,
    "replay": false,
    "role": "tester",
    "model": "deepseek/deepseek-chat",
    "code": "agent_generation_failed",
    "message": "Tester could not finish its summary, but the run can continue with preserved output.",
    "terminal": true,
    "occurred_at": "2026-03-24T12:00:05Z"
  }
}
```

### `run_complete`

Terminal event for a successful orchestration run.

```json
{
  "type": "run_complete",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_explainer_1",
    "sequence": 20,
    "replay": false,
    "role": "explainer",
    "model": "google/gemini-2.0-flash-001",
    "summary": "The orchestration completed with planner, code, test, review, and explanation stages.",
    "occurred_at": "2026-03-24T12:00:08Z"
  }
}
```

### `run_error`

Terminal event for an unrecoverable orchestration failure.

```json
{
  "type": "run_error",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_planner_1",
    "sequence": 4,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "code": "planner_required",
    "message": "The run stopped because the planner did not complete and downstream work could not continue.",
    "terminal": true,
    "occurred_at": "2026-03-24T12:00:01Z"
  }
}
```

## Behavioral Rules

- Every orchestration-mode event includes `run_id` and `sequence`.
- Every agent-scoped orchestration-mode event includes `agent_id` so the frontend can route updates to the correct node.
- `agent_spawned` is the only event that may cause the frontend to append a new node and rerun dagre layout.
- `agent_state_changed`, `task_assigned`, `token`, and `agent_error` must be applied through node patching rather than full graph replacement.
- `agent_error` marks one node as failed and preserves its transcript; it does not imply `run_error` unless the DAG can no longer continue.
- `run_error` is the only event that halts all remaining orchestration activity.
- Planner callback failure is represented as a planner `agent_error` followed by a terminal planner-attributed `run_error`.
- Replayed events reuse the same event types with `replay: true` and must not create duplicate nodes for existing `agent_id` values.
- Prompt-only orchestration mode does not emit `tool_call` or `tool_result` events.
- Unknown client messages still return a standard `error` response envelope.