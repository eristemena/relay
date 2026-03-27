# WebSocket Contract: Agent Token Usage

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Go backend remains the only source of live run events and replay events

This feature does not add new message types. It extends existing terminal event payloads so token usage and context limits can flow through the same live and replay channels already used by the workspace store and canvas.

## Envelope

All messages continue to use the shared JSON envelope.

```json
{
  "type": "complete",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "role": "coder",
    "model": "anthropic/claude-sonnet-4-5",
    "finish_reason": "stop",
    "tokens_used": 4812,
    "context_limit": 200000,
    "occurred_at": "2026-03-27T12:00:00Z",
    "sequence": 18,
    "replay": false
  }
}
```

## Client -> Server Messages

No new client-originated WebSocket messages are required for this feature.

## Server -> Client Messages

### `complete`

The existing single-agent completion event gains two optional fields.

```json
{
  "type": "complete",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "role": "coder",
    "model": "anthropic/claude-sonnet-4-5",
    "finish_reason": "stop",
    "tokens_used": 4812,
    "context_limit": 200000,
    "occurred_at": "2026-03-27T12:00:00Z"
  }
}
```

Rules:

- `tokens_used` is omitted or null when provider usage is unavailable or not authoritative.
- `context_limit` is omitted or null when Relay cannot resolve a valid positive limit.
- The event remains valid for older stored runs that do not include the new fields.

### `agent_state_changed`

When an orchestration stage reaches its terminal completed state, the existing payload may include the same optional token fields so the corresponding canvas node can update through the established patch flow.

```json
{
  "type": "agent_state_changed",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "run_456-reviewer-4",
    "role": "reviewer",
    "model": "anthropic/claude-sonnet-4-5",
    "state": "completed",
    "message": "Reviewer completed the analysis.",
    "tokens_used": 7250,
    "context_limit": 200000,
    "occurred_at": "2026-03-27T12:00:25Z"
  }
}
```

Rules:

- `tokens_used` and `context_limit` are meaningful only when `state` is a terminal completion state.
- Consumers must ignore missing token fields instead of computing estimates.
- The same event shape is used for live streaming and replay.

### `run_complete`

The run-summary event may mirror the final agent's token usage when available so replayed timeline views and run-summary logic stay consistent.

```json
{
  "type": "run_complete",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "run_456-explainer-5",
    "role": "explainer",
    "model": "google/gemini-2.0-flash-001",
    "summary": "The run completed successfully.",
    "tokens_used": 1920,
    "context_limit": 1048576,
    "occurred_at": "2026-03-27T12:00:40Z"
  }
}
```

Rules:

- `run_complete` does not replace the per-agent token payload on `agent_state_changed`; it mirrors the final stage only.
- Consumers that do not need token usage may ignore the new fields without behavior changes.

## Replay Rules

- `OpenRun` replay must emit the same optional `tokens_used` and `context_limit` fields when they are present in the stored event row.
- Older event rows without token columns continue to replay successfully and simply omit the new fields.
- `replay: true` remains the indicator that a payload originated from historical reconstruction.

## Behavioral Rules

- The UI must never infer or display estimated token counts when the payload omits `tokens_used`.
- If `context_limit` is invalid, the client may still display the raw token count but must not compute a percentage-based risk band from it.
- If `tokens_used` exceeds `context_limit`, the client caps the visible bar at full width and treats the state as critical.

## Non-Goals

- New WebSocket message types for token telemetry
- Aggregate multi-run token reporting
- Per-token cost or billing payloads
- Backfilling token usage into events created before this feature