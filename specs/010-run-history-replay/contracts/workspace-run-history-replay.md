# WebSocket Contract: Run History and Replay

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Go backend remains the only source of replay timing, history search results, diff-review data, and markdown export outcomes

This feature extends the existing workspace protocol with history query, replay control, replay state, history detail, and export messages. Historical content events still use the existing run event types already consumed by the workspace store.

## Shared Rules

- Replay remains read-only. No replay message may trigger agent execution, tool execution, approval prompts, or repository writes other than explicit markdown export.
- Historical content continues to flow through existing event types such as `token`, `tool_call`, `tool_result`, `agent_state_changed`, `run_complete`, and `run_error`.
- All replayed content events include `replay: true` and continue to carry `sequence`, `occurred_at`, `tokens_used`, and `context_limit` when those values exist in storage.

## Client -> Server Messages

### `run.history.query`

Requests a filtered saved-run list without reloading the entire workspace.

```json
{
  "type": "run.history.query",
  "request_id": "req_123",
  "payload": {
    "session_id": "session_123",
    "query": "approval review",
    "file_path": "web/src/features/history/RunHistoryPanel.tsx",
    "date_from": "2026-03-20T00:00:00Z",
    "date_to": "2026-03-28T23:59:59Z"
  }
}
```

Rules:

- Any field except `session_id` may be omitted.
- The backend combines FTS keyword search across generated titles, goals, summaries, replay-safe transcript text, and touched file names with file-path and date-range filters.
- Empty filters return the full saved-run list for the session.

### `agent.run.open`

Keeps its current role of selecting a saved run, but now also creates or restores a replay session for historical runs.

```json
{
  "type": "agent.run.open",
  "request_id": "req_124",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456"
  }
}
```

Rules:

- For a historical run, the backend prepares replay data, emits a workspace snapshot, then begins scheduled playback at `1x` unless a restored replay state exists.
- For an active live run, current open-run behavior remains unchanged.

### `agent.run.replay.control`

Controls the selected historical replay.

```json
{
  "type": "agent.run.replay.control",
  "request_id": "req_125",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "action": "seek",
    "cursor_ms": 42000,
    "speed": 2
  }
}
```

Allowed `action` values:

- `play`
- `pause`
- `seek`
- `set_speed`
- `reset`

Rules:

- `cursor_ms` is required for `seek`.
- `speed` is required for `set_speed` and must be one of `0.5`, `1`, `2`, or `5`.
- `reset` returns replay to position `0` and a paused state unless the backend chooses to autoplay immediately after reset.

### `run.history.details.request`

Requests the normalized diff-review and metadata payload for one saved run.

```json
{
  "type": "run.history.details.request",
  "request_id": "req_126",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456"
  }
}
```

### `run.history.export.request`

Requests markdown export for the selected run.

```json
{
  "type": "run.history.export.request",
  "request_id": "req_127",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456"
  }
}
```

Rules:

- Export always uses the full stored run, not the current scrubbed cursor.
- The handler accepts this message only as a direct developer request from the workspace client and rejects any equivalent export path initiated from agent orchestration or replay automation.
- The backend writes to `~/.relay/exports/` by default and reports the output path on success.

## Server -> Client Messages

### `run.history.result`

Returns filtered run summaries.

```json
{
  "type": "run.history.result",
  "request_id": "req_123",
  "payload": {
    "session_id": "session_123",
    "query": "approval review",
    "file_path": "web/src/features/history/RunHistoryPanel.tsx",
    "date_from": "2026-03-20T00:00:00Z",
    "date_to": "2026-03-28T23:59:59Z",
    "runs": [
      {
        "id": "run_456",
        "generated_title": "Review approval flow",
        "started_at": "2026-03-28T09:00:00Z",
        "completed_at": "2026-03-28T09:02:10Z",
        "agent_count": 5,
        "final_status": "completed",
        "has_file_changes": true
      }
    ]
  }
}
```

Rules:

- The response payload reflects the active filters so the client can preserve filter context in no-results states.
- `runs` may be empty without this being an error.

### `agent.run.replay.state`

Emits replay transport state for the selected run.

```json
{
  "type": "agent.run.replay.state",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "status": "playing",
    "cursor_ms": 42000,
    "duration_ms": 95000,
    "speed": 2,
    "selected_timestamp": "2026-03-28T09:00:42Z"
  }
}
```

Rules:

- `status` is one of `preparing`, `playing`, `paused`, `seeking`, `completed`, or `error`.
- This message is transport metadata only; historical content still arrives through the normal run event types.
- The frontend uses this message to drive scrubber position, play/pause state, and speed control labels.

### `run.history.details.result`

Returns normalized historical diff-review data and detail metadata.

```json
{
  "type": "run.history.details.result",
  "request_id": "req_126",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "generated_title": "Review approval flow",
    "final_status": "completed",
    "agent_count": 5,
    "change_records": [
      {
        "tool_call_id": "tool_1",
        "path": "web/src/features/history/RunHistoryPanel.tsx",
        "original_content": "before\n",
        "proposed_content": "after\n",
        "base_content_hash": "sha256:abc",
        "approval_state": "applied",
        "occurred_at": "2026-03-28T09:01:14Z"
      }
    ]
  }
}
```

Rules:

- `change_records` may be empty, in which case the UI shows an explicit no-file-changes state.
- All content must come from persisted historical data only.
- The details payload always returns the full normalized change-record set for the run; the client may filter that list against `agent.run.replay.state.selected_timestamp` when showing cursor-specific diff review.

### `run.history.export.result`

Reports export completion.

```json
{
  "type": "run.history.export.result",
  "request_id": "req_127",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "status": "completed",
    "export_path": "/Users/example/.relay/exports/2026-03-28-review-approval-flow.md",
    "generated_at": "2026-03-28T09:03:00Z"
  }
}
```

Rules:

- `status` is `started`, `completed`, or `error`.
- On error, the backend sends a standard `error` envelope with a human-readable message and the `run_id`.

## Historical Content Event Rules

- The replay scheduler re-emits stored events in strict sequence order.
- Event pacing is derived from normalized timestamps and scaled by the active replay speed.
- Seek rebuilds state by re-emitting all events up to the target timestamp, but the backend may do so from an internal checkpoint so the client still sees only the correct resulting event stream.
- Existing event payloads continue to carry `tokens_used` and `context_limit` when available so token bars animate during playback.

## Non-Goals

- New LLM or tool execution during replay
- Browser-only export downloads
- Video export or share links
- Reading current repository files to reconstruct historical changes