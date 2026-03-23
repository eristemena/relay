# Data Model: Local Relay Workspace

## Session

Represents one locally persisted Relay work thread that can be reopened after a restart.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `id` | text | SQLite | Primary key; immutable; generated as a stable opaque identifier (`uuid` or `ulid`) |
| `display_name` | text | SQLite | Required; user-visible title; defaults to a generated name for new sessions |
| `created_at` | datetime | SQLite | Required; set once at creation |
| `updated_at` | datetime | SQLite | Required; updated whenever session metadata or snapshot changes |
| `last_opened_at` | datetime | SQLite | Required; updated whenever the session becomes active |
| `status` | text enum | SQLite | Required; one of `active`, `idle`, `archived` |
| `snapshot_json` | text | SQLite | Required; JSON blob containing only the minimal workspace state needed to restore the shell for this feature |

### Validation

- `display_name` must be non-empty after trimming.
- `status` must use only supported enum values.
- `snapshot_json` must never contain secrets or raw credential values.
- Unfinished sessions remain valid records and must remain selectable after restart.

### State Transitions

- New session: record created with `status=active` and an empty snapshot.
- Open different session: previously active session becomes `idle`; selected session becomes `active`.
- Shutdown/restart: the last active session may return as `idle` in storage but must still be reopenable.

## Preference Set

Represents the persisted local Relay configuration stored in `~/.relay/config.toml`.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `port` | integer | TOML | Required; preferred Relay listening port; defaults to `4747`; must be within `1-65535` |
| `open_browser_on_start` | boolean | TOML | Required; defaults to `true` |
| `appearance_variant` | text | TOML | Optional; must be one of the supported dark-mode variants only |
| `last_session_id` | text nullable | TOML | Optional; if present, must reference an existing session or be ignored safely |
| `credentials` | array/object | TOML | Optional; provider-specific secret entries stored locally and never sent to the frontend |

### Validation

- Invalid or unreadable preference values must fall back field-by-field without discarding unrelated valid preferences.
- If the preferred port is unavailable at startup, Relay must choose a free runtime port without overwriting the saved preferred port unless the user explicitly saves a new value.
- Unsupported appearance values must be ignored while preserving supported dark-mode behavior.
- Credentials must be redacted in logs and omitted from all frontend payloads.

## Credential Entry

Nested record inside the preference set for one provider credential.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `provider` | text | TOML | Required; stable provider key such as `openai` or `anthropic` |
| `label` | text | TOML | Optional; user-facing nickname |
| `secret` | text | TOML | Required when entry exists; stored locally only |
| `updated_at` | datetime | TOML | Optional; last write timestamp for UI display or troubleshooting |

## Workspace Snapshot

Minimal persisted UI state stored in `Session.snapshot_json` for this feature phase.

| Field | Type | Rules |
|-------|------|-------|
| `active_panel` | text | Optional; identifies the last visible panel in the workspace shell |
| `canvas_state` | object | Optional; minimal state for showing an empty or restored canvas shell |
| `has_activity` | boolean | Required; indicates whether the session is still effectively empty |
| `recoverable_error` | object nullable | Optional; stores only display-safe error metadata if the last run ended with a recoverable state |

### Rules

- The snapshot is intentionally minimal for this phase and excludes AI transcript content, repository linkage, and tool execution history.
- The snapshot must be safe to send to the browser without server-side redaction.

## Workspace Connection

Ephemeral runtime state for one browser connection over `/ws`.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `connection_id` | text | Memory | Unique per WebSocket connection |
| `active_session_id` | text nullable | Memory | Session currently displayed for that browser client |
| `client_build` | text nullable | Memory | Optional frontend build identifier for diagnostics |
| `active_port` | integer | Memory | Actual Relay port assigned for the current run; may differ from the preferred saved port |
| `connected_at` | datetime | Memory | Set when the socket is accepted |
| `last_seen_at` | datetime | Memory | Updated on ping/pong or message receipt |

### Rules

- A browser refresh creates a new connection but should reuse the same active session when possible by requesting a fresh bootstrap snapshot.
- Connection state must be cancellable via `context.Context` and cleaned up when the socket closes.

## Relationships

- One `Preference Set` exists per local Relay installation.
- One `Preference Set` may reference zero or one `last_session_id`.
- One `Session` owns one serialized `Workspace Snapshot`.
- One active `Workspace Connection` displays zero or one `Session` at a time.