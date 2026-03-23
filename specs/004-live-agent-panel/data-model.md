# Data Model: Live Agent Panel

## Session Extension

The Phase 1 `Session` record remains the top-level workspace container. This phase extends it so a session owns an ordered history of agent runs.

### Relationships

- One `Session` has many `AgentRun` records.
- One `Session` may reference one `active_run_id` in runtime state, but persisted history must keep all prior runs.
- Reopening a session restores the selected or latest replayable run without contacting the model provider.

## OpenRouter Settings

Represents the server-only configuration stored under `[openrouter]` in `~/.relay/config.toml`.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `api_key` | text | TOML | Optional at rest; required before a live run can start; never sent to the frontend; redacted in logs and diagnostics |

### Validation

- An empty or missing `api_key` blocks task submission with a human-readable configuration error.
- The API key must be stored with local-file permissions appropriate for secrets.
- The frontend receives only derived credential status, never the raw key.

## Project Root Setting

Represents the manually managed repository root stored in `~/.relay/config.toml`.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `project_root` | text | TOML | Required for repo-scoped tools; must be an absolute local path; edited manually in config for this phase |

### Validation

- `project_root` must be non-empty after trimming and resolve to an existing local directory before `read_file` or `search_codebase` can run.
- Repo-scoped tools must reject paths outside `project_root`.
- The frontend receives only display-safe error status when `project_root` is missing or invalid; it never receives unrestricted directory listings from config.

## Agent Model Settings

Represents the configurable model assignment stored under `[agents]` in `~/.relay/config.toml`.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `planner` | text | TOML | Required; defaults to `anthropic/claude-opus-4` |
| `coder` | text | TOML | Required; defaults to `anthropic/claude-sonnet-4-5` |
| `reviewer` | text | TOML | Required; defaults to `anthropic/claude-sonnet-4-5` |
| `tester` | text | TOML | Required; defaults to `deepseek/deepseek-chat` |
| `explainer` | text | TOML | Required; defaults to `google/gemini-flash-1.5` |

### Validation

- Each model string must be non-empty after trimming.
- Invalid or missing model strings fall back field-by-field to the default assignment rather than invalidating the whole config.
- The selected model string is copied into every emitted run event so the frontend can display a stable badge for both live and replayed runs.

## Agent Role Profile

Code-defined description of a built-in Relay role.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `role` | text enum | Code | One of `planner`, `coder`, `reviewer`, `tester`, `explainer` |
| `system_prompt` | text | Code | Required; fixed in code; not user-editable |
| `allowed_tools` | string array | Code | Required; fixed at construction time |
| `model` | text | Config + Code | Resolved from `[agents]` and injected into the role instance |

### Rules

- `planner` and `reviewer` allow only `read_file` and `search_codebase`.
- `coder` and `tester` allow `read_file`, `search_codebase`, `write_file`, and `run_command`.
- `explainer` allows `read_file` only.
- The role profile is part of the product contract and should be versioned with code changes, not with runtime settings.
- Read-only repo-scoped tools use the configured `project_root` as their filesystem boundary.

## Agent Run

Represents one submitted developer task executed by one selected agent role.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `id` | text | SQLite | Primary key; opaque identifier |
| `session_id` | text | SQLite | Required; foreign key to `Session` |
| `task_text` | text | SQLite | Required; original developer prompt |
| `role` | text enum | SQLite | Required; selected built-in role |
| `model` | text | SQLite | Required; resolved model string used for this run |
| `state` | text enum | SQLite | Required; one of `accepted`, `thinking`, `tool_running`, `completed`, `errored` |
| `started_at` | datetime | SQLite | Required |
| `completed_at` | datetime nullable | SQLite | Set on terminal completion or terminal error |
| `error_code` | text nullable | SQLite | Optional terminal classification |
| `error_message` | text nullable | SQLite | Optional display-safe terminal message |
| `first_token_at` | datetime nullable | SQLite | Optional; set when first visible token arrives |

### Validation

- Only one `AgentRun` may be live at a time across the workspace for this phase.
- `task_text` must be non-empty after trimming.
- Terminal error messages must be display-safe and must not contain secrets.
- `first_token_at` must not be earlier than `started_at`.

### State Transitions

- Submit task: `accepted`
- Agent begins reasoning: `thinking`
- Tool call starts: `tool_running`
- Tool call completes successfully: `thinking`
- Final assistant completion: `completed`
- Provider, validation, or tool failure: `errored`

## Agent Run Event

Append-only, replayable event emitted during a run.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | SQLite | Required; foreign key to `AgentRun` |
| `sequence` | integer | SQLite | Required; strictly increasing per run starting at `1` |
| `event_type` | text enum | SQLite | Required; one of `state_change`, `token`, `tool_call`, `tool_result`, `complete`, `error` |
| `model` | text | SQLite | Required; repeated for replay without joining config |
| `role` | text enum | SQLite | Required |
| `payload_json` | text | SQLite | Required; display-safe event body |
| `created_at` | datetime | SQLite | Required |

### Validation

- Event order is authoritative and determined by `sequence`, not by timestamp alone.
- `payload_json` must be safe for browser replay and must not contain raw API keys or unredacted sensitive tool data.
- `tool_result` events must reference a previously emitted tool call identifier in their payload.
- The terminal event for a successful run is `complete`; the terminal event for a failed run is `error`.

## Run Summary View

Frontend-safe projection used in bootstrap payloads and history lists.

| Field | Type | Rules |
|-------|------|-------|
| `id` | text | Run identifier |
| `task_text_preview` | text | Shortened display-safe task preview |
| `role` | text enum | Built-in role name |
| `model` | text | Visible model badge text |
| `state` | text enum | Final or current state |
| `started_at` | datetime | Used for ordering |
| `completed_at` | datetime nullable | Optional for in-progress runs |
| `has_tool_activity` | boolean | Helps the UI show history affordances |

## Credential Status View

Frontend-safe projection derived from `[openrouter]`.

| Field | Type | Rules |
|-------|------|-------|
| `configured` | boolean | True when a non-empty API key is stored |
| `last_updated_at` | datetime nullable | Optional if the config writer tracks it later |
