# WebSocket Contract: Local Relay Workspace

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Always served by Go, in both development and production

## Envelope

All client and server messages use the same JSON envelope.

```json
{
  "type": "workspace.bootstrap.request",
  "request_id": "9c4d4bc4-5a32-4f65-88ca-8d4e0e6a2dbb",
  "payload": {}
}
```

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `type` | string | yes | Message discriminator |
| `request_id` | string | no | Echoed by server responses when present |
| `payload` | object | yes | Message-specific body |

## Client -> Server Messages

### `workspace.bootstrap.request`

Requests the full current UI snapshot after initial page load or reconnect.

```json
{
  "type": "workspace.bootstrap.request",
  "request_id": "req-bootstrap-1",
  "payload": {
    "last_session_id": "session_123"
  }
}
```

### `session.create`

Creates a new session and makes it active.

```json
{
  "type": "session.create",
  "request_id": "req-create-1",
  "payload": {
    "display_name": "New session"
  }
}
```

### `session.open`

Opens an existing saved session.

```json
{
  "type": "session.open",
  "request_id": "req-open-1",
  "payload": {
    "session_id": "session_123"
  }
}
```

### `preferences.save`

Persists supported frontend-safe preference values. Secret values are submitted only when explicitly edited and are never echoed back in plain text. The saved port value is the preferred Relay port, not necessarily the runtime port used for the current session.

```json
{
  "type": "preferences.save",
  "request_id": "req-pref-1",
  "payload": {
    "preferred_port": 4747,
    "appearance_variant": "midnight",
    "credentials": [
      {
        "provider": "openai",
        "label": "Personal key",
        "secret": "submitted-but-never-echoed"
      }
    ]
  }
}
```

## Server -> Client Messages

### `workspace.bootstrap`

Sent in response to `workspace.bootstrap.request` and after reconnect when the server decides a full resync is safer than incremental replay.

```json
{
  "type": "workspace.bootstrap",
  "request_id": "req-bootstrap-1",
  "payload": {
    "active_session_id": "session_123",
    "sessions": [
      {
        "id": "session_123",
        "display_name": "Investigate startup",
        "created_at": "2026-03-23T10:15:00Z",
        "last_opened_at": "2026-03-23T10:20:00Z",
        "status": "active",
        "has_activity": false
      }
    ],
    "preferences": {
      "preferred_port": 4747,
      "appearance_variant": "midnight",
      "has_credentials": true
    },
    "ui_state": {
      "history_state": "ready",
      "canvas_state": "empty",
      "save_state": "idle"
    }
  }
}
```

### `session.created`

Broadcast after a new session is created successfully.

### `session.opened`

Broadcast after a requested session becomes active.

### `preferences.saved`

Confirms that preferences were stored and returns only safe, displayable values.

### `workspace.status`

Pushes loading, saving, or transient service state updates.

Payload shape:

```json
{
  "phase": "history-loading",
  "message": "Loading saved sessions"
}
```

### `error`

Reports a recoverable, user-displayable error.

```json
{
  "type": "error",
  "request_id": "req-open-1",
  "payload": {
    "code": "session_not_found",
    "message": "That session is no longer available. Choose another session or start a new one."
  }
}
```

## Behavioral Rules

- The server must send a fresh `workspace.bootstrap` after browser refresh or any reconnect where incremental state replay is uncertain.
- Secrets must never appear in outbound payloads.
- The browser should treat `workspace.bootstrap` as authoritative and replace its local session list and active-session state.
- The browser must treat any `preferred_port` value as configuration state only; the actual runtime address for the current connection is the address already in use by the browser.
- Unknown message types must produce an `error` event rather than silently failing.
- Long-running backend work must emit `workspace.status` updates so the UI can show visible loading or saving states.