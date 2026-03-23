# WebSocket Contract: Live Agent Panel

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Always served by Go, in both development and production

This phase extends the Phase 1 workspace protocol. Existing bootstrap, session, and preferences messages remain valid. New live-agent messages are added without introducing any REST replacement for runtime state.

## Envelope

All messages continue to use the shared JSON envelope.

```json
{
  "type": "agent.run.submit",
  "request_id": "req-run-1",
  "payload": {
    "session_id": "session_123",
    "task": "Plan the steps to add JWT parsing to Relay"
  }
}
```

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `type` | string | yes | Message discriminator |
| `request_id` | string | no | Echoed when the message is a direct response to a request |
| `payload` | object | yes | Message-specific body |

## Client -> Server Messages

### `workspace.bootstrap.request`

Unchanged from Phase 1, but the bootstrap response now includes run-history data for the active session.

### `agent.run.submit`

Starts a new live run for the given session.

```json
{
  "type": "agent.run.submit",
  "request_id": "req-run-1",
  "payload": {
    "session_id": "session_123",
    "task": "Write a Go function that parses a JWT token from an Authorization header"
  }
}
```

### `agent.run.open`

Requests replay of a previously saved run.

```json
{
  "type": "agent.run.open",
  "request_id": "req-open-run-1",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456"
  }
}
```

## Server -> Client Messages

### `workspace.bootstrap`

Phase 1 payload plus the active-session run list and credential status needed for the live panel.

```json
{
  "type": "workspace.bootstrap",
  "request_id": "req-bootstrap-1",
  "payload": {
    "active_session_id": "session_123",
    "sessions": [],
    "preferences": {
      "preferred_port": 4747,
      "appearance_variant": "midnight",
      "open_browser_on_start": true
    },
    "ui_state": {
      "history_state": "ready",
      "canvas_state": "ready",
      "save_state": "idle"
    },
    "active_run_id": "run_456",
    "run_summaries": [
      {
        "id": "run_456",
        "task_text_preview": "Write a Go function that parses a JWT token...",
        "role": "planner",
        "model": "anthropic/claude-opus-4",
        "state": "completed",
        "started_at": "2026-03-23T12:00:00Z",
        "completed_at": "2026-03-23T12:00:06Z",
        "has_tool_activity": true
      }
    ],
    "credential_status": {
      "configured": true
    }
  }
}
```

### `state_change`

Emitted when a run enters a new visible execution state.

```json
{
  "type": "state_change",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 1,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "state": "thinking",
    "message": "Planner is analyzing the task",
    "occurred_at": "2026-03-23T12:00:00Z"
  }
}
```

### `token`

Emitted for each ordered slice of visible streamed text.

```json
{
  "type": "token",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 2,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "text": "1. Clarify the input source...",
    "first_token_latency_ms": 184,
    "occurred_at": "2026-03-23T12:00:00Z"
  }
}
```

- `first_token_latency_ms` is optional and appears only on the first visible token for a live run so Relay can measure submit-to-first-output latency without delaying streaming.

### `tool_call`

Emitted when the agent requests a tool.

```json
{
  "type": "tool_call",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 3,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "tool_call_id": "tool_789",
    "tool_name": "read_file",
    "input_preview": {
      "path": "/repo/internal/handlers/ws/protocol.go"
    },
    "occurred_at": "2026-03-23T12:00:01Z"
  }
}
```

### `tool_result`

Emitted after a tool invocation completes.

```json
{
  "type": "tool_result",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 4,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "tool_call_id": "tool_789",
    "tool_name": "read_file",
    "status": "ok",
    "result_preview": {
      "summary": "Loaded protocol constants and payload types"
    },
    "occurred_at": "2026-03-23T12:00:01Z"
  }
}
```

### `complete`

Terminal event for a successful run.

```json
{
  "type": "complete",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 18,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "finish_reason": "stop",
    "occurred_at": "2026-03-23T12:00:06Z"
  }
}
```

### `error`

Used for both request-level errors and terminal run errors. When the error belongs to a run, the payload includes run metadata and sequence.

```json
{
  "type": "error",
  "request_id": "req-run-1",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 12,
    "replay": false,
    "role": "tester",
    "model": "deepseek/deepseek-chat",
    "code": "tool_call_unsupported",
    "message": "The configured model did not complete the requested tool call. Choose a different model or rerun without tools.",
    "terminal": true,
    "occurred_at": "2026-03-23T12:00:03Z"
  }
}
```

### `error` for invalid project root

Used when a run or tool action requires repository access but Relay does not have a valid local project root configured.

```json
{
  "type": "error",
  "request_id": "req-run-1",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "sequence": 1,
    "replay": false,
    "role": "planner",
    "model": "anthropic/claude-opus-4",
    "code": "project_root_invalid",
    "message": "Repository-reading tools are unavailable until you set a valid project root in Relay's local configuration.",
    "terminal": true,
    "occurred_at": "2026-03-23T12:00:00Z"
  }
}
```

## Behavioral Rules

- The server must assign a strictly increasing `sequence` per run and persist that same order in SQLite before replaying it later.
- Every live run event payload must include the resolved `model` string and selected `role` so the frontend can render consistent badges during both live streaming and replay.
- `tool_call` and `tool_result` events must remain in the same ordered stream as `token` and `state_change` events; the frontend must not infer ordering by timestamps alone.
- Secrets must never appear in event payloads. Tool inputs and results are sent only as display-safe previews.
- If a run requires repository-reading tools and `project_root` is missing or invalid, the server must reject the request with a clear `error` event rather than attempting the tool call.
- When a saved run is reopened, the server replays the stored event stream using the same event types with `replay: true`.
- If a new live run is requested while another run is active, the server must reject the request with an `error` event rather than running multiple agents concurrently.
- If OpenRouter returns a mid-stream provider error after some tokens were already sent, Relay must forward a terminal `error` event after preserving all earlier ordered events.
- Unknown client message types must result in an `error` event.