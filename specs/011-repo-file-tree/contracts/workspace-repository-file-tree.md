# WebSocket Contract: Repository File Tree Sidebar

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Go backend remains the only source of repository tree hydration and live touched-file updates

This feature extends the existing workspace protocol with one request/result pair for initial repository tree hydration and one incremental event for touched-file updates. A separate HTTP tree endpoint is intentionally not added because Relay's architecture requires WebSocket as the only backend/frontend runtime channel.

## Shared Rules

- The repository tree is read-only. No tree message may open files, approve changes, reject changes, or execute commands.
- Tree data is always scoped to the connected repository root.
- Touched-file state reports what a run read or proposed to change; it does not imply a file was modified on disk.

## Client -> Server Messages

### `repository.tree.request`

Requests the current connected repository tree and the active run's touched-file snapshot.

```json
{
  "type": "repository.tree.request",
  "request_id": "req_201",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456"
  }
}
```

Rules:

- `session_id` is required.
- `run_id` is optional; when present, the backend includes the current touched-file snapshot for that run.
- If no valid repository root is connected, the backend responds with an `error` envelope using a human-readable message.

## Server -> Client Messages

### `repository.tree.result`

Returns the flat repository path list plus the current touched snapshot.

```json
{
  "type": "repository.tree.result",
  "request_id": "req_201",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "repository_root": "/Users/example/project",
    "status": "ready",
    "message": "Repository tree is ready.",
    "paths": [
      "cmd",
      "cmd/relay",
      "cmd/relay/main.go",
      "internal",
      "internal/tools",
      "internal/tools/read_file.go"
    ],
    "touched_files": [
      {
        "run_id": "run_456",
        "agent_id": "agent_planner_1",
        "file_path": "internal/tools/read_file.go",
        "touch_type": "read"
      }
    ]
  }
}
```

Rules:

- `status` is `loading`, `ready`, or `error`.
- `paths` contains normalized repository-relative paths using forward slashes.
- `touched_files` may be empty when no files have been touched yet or when `run_id` is omitted.
- The client derives the visible tree hierarchy locally from `paths`.

### `file_touched`

Emitted when a file read succeeds or a write proposal is created.

```json
{
  "type": "file_touched",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "agent_id": "agent_coder_1",
    "role": "coder",
    "file_path": "web/src/features/history/RunHistoryPanel.tsx",
    "touch_type": "proposed",
    "replay": false
  }
}
```

Rules:

- `touch_type` is `read` or `proposed`.
- The event is emitted from the tool execution path, not from a later agent lifecycle transition.
- Duplicate touches for the same `(run_id, agent_id, file_path, touch_type)` combination may be ignored by the client.

## Client-Side Derivation Rules

- The workspace-wide tree view is the union of all touched-file records for the current run.
- The selected-agent tree view is the subset where `agent_id` matches the selected canvas node.
- The client renders only the first two levels initially, then reveals deeper descendants when directories are expanded.

## Error Handling

- Tree hydration failures return the standard `error` envelope with `code: repository_tree_failed` and a plain-language message.
- Invalid or disconnected repository state must not produce an empty successful result.
- Touched-file streaming failures must not clear the already loaded tree; the UI preserves the existing tree and shows a synchronization error state.

## Non-Goals

- HTTP `GET /api/repo/tree`
- File opening or editor navigation from the tree
- Approval decisions or diff application from the tree
- Separate backend queries for per-agent touched files once the tree snapshot is loaded