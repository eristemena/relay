# WebSocket Contract: Codebase Awareness

## Endpoint

- Path: `/ws`
- Transport: WebSocket via `nhooyr.io/websocket`
- Ownership: Go backend remains the only server for workspace, repository, and approval events

This feature extends the existing workspace protocol with repository connection details, persisted approval lifecycle updates, a server-backed folder browser, and asynchronous codebase-graph delivery through a single status event.

## Envelope

All messages continue to use the shared JSON envelope.

```json
{
  "type": "preferences.save",
  "request_id": "req-repo-1",
  "payload": {
    "project_root": "/Users/dev/src/relay"
  }
}
```

## Client -> Server Messages

### `preferences.save`

Unchanged message type. For this feature it remains the persistence path for `project_root`, whether the path was typed manually, supplied by startup flag, or selected from the folder picker.

### `repository.browse.request`

Requests one level of directories for the in-product folder picker.

```json
{
  "type": "repository.browse.request",
  "request_id": "req-browse-1",
  "payload": {
    "path": "/Users/dev/src",
    "show_hidden": false
  }
}
```

Rules:

- This message is available to the developer UI only, not to agents.
- The server returns directories only; it does not expose file contents through this picker flow.

### `agent.run.approval.respond`

Unchanged message type, but it now resolves a persisted approval request.

```json
{
  "type": "agent.run.approval.respond",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "tool_call_id": "tool_789",
    "decision": "approved"
  }
}
```

Rules:

- `decision` remains `approved` or `rejected`.
- The backend records the decision in SQLite before execution continues.

## Server -> Client Messages

### `workspace.bootstrap`

Extended to include connected repository status and persisted pending approvals.

```json
{
  "type": "workspace.bootstrap",
  "payload": {
    "active_session_id": "session_123",
    "preferences": {
      "project_root": "/Users/dev/src/relay",
      "project_root_configured": true,
      "project_root_valid": true,
      "project_root_message": "Connected to a local Git repository."
    },
    "connected_repository": {
      "path": "/Users/dev/src/relay",
      "status": "connected",
      "message": "Repository-aware reads stay inside this local Git worktree."
    },
    "pending_approvals": [
      {
        "session_id": "session_123",
        "run_id": "run_456",
        "tool_call_id": "tool_789",
        "tool_name": "write_file",
        "request_kind": "file_write",
        "repository_root": "/Users/dev/src/relay",
        "message": "Relay needs approval before writing this file.",
        "status": "proposed"
      }
    ]
  }
}
```

Rules:

- `pending_approvals` includes requests still in `proposed` or `approved` state so reconnect can restore them.
- `connected_repository` describes the validated repository bound to the current workspace.
- Background repository-context state is not embedded in bootstrap; Relay sends it separately through `repository_graph_status` so analysis work stays asynchronous.

### `repository.browse.result`

Returns one level of directory choices for the folder picker.

```json
{
  "type": "repository.browse.result",
  "request_id": "req-browse-1",
  "payload": {
    "path": "/Users/dev/src",
    "directories": [
      {
        "name": "relay",
        "path": "/Users/dev/src/relay",
        "is_git_repository": true
      }
    ]
  }
}
```

### `approval_request`

Emitted when a persisted approval request enters `proposed` state.

```json
{
  "type": "approval_request",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "role": "coder",
    "model": "anthropic/claude-sonnet-4-5",
    "tool_call_id": "tool_789",
    "tool_name": "write_file",
    "request_kind": "file_write",
    "status": "proposed",
    "message": "Relay needs approval before writing this file.",
    "repository_root": "/Users/dev/src/relay",
    "input_preview": {
      "target_path": "internal/tools/catalog.go"
    },
    "diff_preview": {
      "target_path": "internal/tools/catalog.go",
      "original_content": "package tools\n",
      "proposed_content": "package tools\n\n// updated\n",
      "base_content_hash": "sha256:..."
    },
    "occurred_at": "2026-03-25T12:00:00Z"
  }
}
```

For command requests, `command_preview` replaces `diff_preview` and includes `command`, `args`, and `effective_dir`.

### `approval_state_changed`

Emitted whenever a persisted approval transitions after proposal.

```json
{
  "type": "approval_state_changed",
  "payload": {
    "session_id": "session_123",
    "run_id": "run_456",
    "tool_call_id": "tool_789",
    "tool_name": "write_file",
    "status": "approved",
    "occurred_at": "2026-03-25T12:00:05Z",
    "message": "Relay recorded approval and is revalidating before apply."
  }
}
```

Allowed `status` values:

- `approved`
- `applied`
- `rejected`
- `blocked`
- `expired`

Rules:

- `applied` is emitted only after the tool executes successfully.
- `blocked` or `expired` may occur after approval if repository or file revalidation fails.

### `repository_graph_status`

Emitted whenever the background graph state changes. Relay uses this single event type for `idle`, `loading`, `ready`, and `error` transitions.

```json
{
  "type": "repository_graph_status",
  "payload": {
    "repository_root": "/Users/dev/src/relay",
    "status": "loading",
    "message": "Building repository context in the background."
  }
}
```

Rules:

- Allowed `status` values are `idle`, `loading`, `ready`, and `error`.
- `ready` payloads include `nodes` and `edges` on the same `repository_graph_status` message type.
- `error` payloads carry a plain-language `message` and keep the rest of the workspace usable.
- Graph generation is asynchronous and may return a partial graph.

### Existing `tool_call` and `tool_result`

These event types remain in use, but their preview payloads expand to carry repository-relative file paths, Git history context, search scope, and tool status needed to derive per-agent file activity on the canvas.

## Behavioral Rules

- All repository-aware tool activity remains bounded to the connected repository root.
- No `write_file` or `run_command` execution may occur until an `approval_request` has been persisted and later resolved with `approved`.
- Bootstrap must rehydrate any still-actionable approval requests so the UI can resume review after reconnect.
- Background repository-context events never block workspace bootstrap or approval handling.
- Unknown or invalid repository paths return plain-language `error` payloads rather than silent failures.

## Non-Goals

- Multi-repository sessions
- Remote repository access
- Automatic Git commits, rebases, pushes, or history mutation
- Browser-side filesystem access as the source of truth for server repo operations