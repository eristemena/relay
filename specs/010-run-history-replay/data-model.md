# Data Model: Run History and Replay

## Scope

This feature adds a deterministic backend replay session, indexed run-history metadata, normalized run-change records, and UI-facing replay state for seekable playback.

## Run History Document

Represents the listable and searchable historical summary for one saved run.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Existing `agent_runs.id` | Primary key and foreign key to the recorded run |
| `session_id` | text | Existing run/session relationship | Required |
| `generated_title` | text | Derived from goal, summary, or first meaningful task text | Required and regenerated deterministically |
| `goal_text` | text | Existing `agent_runs.task_text` | Required, full untruncated source text |
| `final_status` | text | Derived from run state and terminal event | Required; one of `completed`, `halted`, `errored`, or `clarification_required` |
| `agent_count` | integer | Derived from stored spawn and terminal events | Required and non-negative |
| `started_at` | timestamp | Existing run metadata | Required |
| `completed_at` | timestamp nullable | Existing run metadata | Null only for unusual active-run recovery cases |
| `first_event_at` | timestamp nullable | Derived from replayable event stream | Used for playback duration |
| `last_event_at` | timestamp nullable | Derived from replayable event stream | Used for playback duration |
| `summary_text` | text nullable | Derived from stored `run_complete` or `complete` data | Optional |
| `touched_file_count` | integer | Derived from normalized change records | Required |
| `has_file_changes` | boolean | Derived | Required |
| `exported_at` | timestamp nullable | Backend export service | Null until at least one export succeeds |

### Rules

- `generated_title` must not require manual user input.
- `agent_count` is derived from historical agent identity, not inferred from current canvas nodes.
- `first_event_at` and `last_event_at` must use the same timestamp normalization rules as replay scheduling.

## Run History Search Document

Represents the FTS5-backed searchable text for a run.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Run history document | Primary key in the content table; also the FTS document key |
| `title_text` | text | Generated title | Indexed |
| `goal_text` | text | Full run goal/task | Indexed |
| `summary_text` | text | Terminal summary text | Indexed when present |
| `transcript_text` | text | Replay-safe transcript content derived from stored events | Indexed when present |
| `file_names_text` | text | Space-separated normalized touched file paths and basenames | Indexed |
| `participant_text` | text | Derived roles and labels | Indexed |

### Rules

- Search matches must combine FTS keyword results with structural filters such as date range and file-path predicates.
- `transcript_text` must be normalized from stored historical events only and must exclude any data that would require reopening repository files or recomputing missing transcript state.
- The search document is rebuilt whenever run metadata or normalized change records for that run change.
- FTS search is scoped to the current Relay workspace history only.

## Stored Run Event Replay Item

Represents one replayable event loaded from `agent_run_events` for scheduler use.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Existing event log | Required |
| `sequence` | integer | Existing event log | Required and authoritative ordering key |
| `event_type` | text | Existing event log | Required |
| `payload_json` | json text | Existing event log | Required |
| `occurred_at` | timestamp | Parsed from payload `occurred_at`, else `created_at` | Required for replay timeline calculation |
| `relative_ms` | integer | Derived at replay load time | Milliseconds from first replayable event |
| `tokens_used` | integer nullable | Existing typed columns | Optional; merged into replay payload |
| `context_limit` | integer nullable | Existing typed columns | Optional; merged into replay payload |

### Rules

- Order is always sequence-first even when timestamps are equal or out of order.
- `relative_ms` may be clamped for large gaps, but the clamping policy must not reorder events.
- Events missing a parseable timestamp still participate in replay using fallback ordering and derived offsets.

## Replay Session

Represents the backend-owned playback state for one selected historical run.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `session_id` | text | Generated per viewer/run pairing | Unique for the active replay controller |
| `workspace_session_id` | text | Existing Relay session | Required |
| `run_id` | text | Selected historical run | Required |
| `status` | enum | Replay controller | `preparing`, `playing`, `paused`, `seeking`, `completed`, `error` |
| `speed` | decimal | User-selected control | Allowed values: `0.5`, `1`, `2`, `5` |
| `cursor_ms` | integer | Replay controller | Current playback position from start |
| `duration_ms` | integer | Derived from run events | Total timeline duration |
| `started_at` | timestamp | Replay controller | Required |
| `updated_at` | timestamp | Replay controller | Required |
| `checkpoint_index` | integer | Replay controller | References the nearest in-memory checkpoint |

### Rules

- Only one replay session is active per selected run in the browser store at a time.
- Replay session state is transport state and does not mutate historical run records.
- Reconnect restoration must preserve `cursor_ms`, `speed`, and `status` when the same run remains selected.

## Replay Checkpoint

Represents a lightweight in-memory checkpoint used to accelerate seek.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Replay session context | Required |
| `checkpoint_ms` | integer | Derived from event timeline | Required |
| `last_sequence` | integer | Derived from event application | Required |
| `canvas_document` | json-like state | Derived by applying events | Required |
| `transcript_state` | map | Derived by applying events | Required |
| `approval_state` | map | Derived by applying events | Required |
| `summary_state` | json-like state | Derived by applying events | Required |

### Rules

- Checkpoints are ephemeral and rebuilt from stored events when a run is opened.
- Checkpoints must be small enough to build quickly for sessions under 10 minutes.
- Seek reconstructs state by restoring the nearest checkpoint and replaying the suffix events up to the target cursor.

## Run Change Record

Represents one preserved historical file change available for diff review and export.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Associated approval/request history | Required |
| `tool_call_id` | text | Approval lifecycle | Required |
| `path` | text | Stored diff preview | Required |
| `original_content` | text | Stored diff preview | Required when historical data exists |
| `proposed_content` | text | Stored diff preview | Required when historical data exists |
| `base_content_hash` | text | Stored diff preview | Required |
| `approval_state` | text | Approval lifecycle | Required; one of `proposed`, `approved`, `applied`, `rejected`, `blocked`, `expired` |
| `role` | text | Approval/request context | Optional but preferred |
| `model` | text | Approval/request context | Optional but preferred |
| `occurred_at` | timestamp | Approval occurrence | Required |

### Rules

- Change records are historical artifacts and must not be recomputed from current disk state.
- A run with no change records must surface an explicit empty state.
- Search-by-file uses normalized `path` values from these records.

## Run Export Document

Represents the persisted result of markdown export generation.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Selected run | Required |
| `export_path` | text | Backend export service | Required and must live under `~/.relay/exports/` by default |
| `generated_at` | timestamp | Export service | Required |
| `title` | text | Run history document | Required |
| `final_status` | text | Run history document | Required |
| `participants` | array-like | Derived from replayable events | Required |
| `timeline_markdown` | text | Derived from stored events | Required |
| `changes_markdown` | text nullable | Derived from run change records | Optional |
| `requested_by` | text | Handler export request context | Required; `developer` for this feature |

### Rules

- The export always reflects the full stored run, never the currently scrubbed viewport.
- Export generation is allowed only for a direct developer action at the handler boundary; agent-triggered or replay-triggered paths do not create export documents.
- Export generation failure must not mutate any historical run data.

## Relationships

- One `Run History Document` has many `Stored Run Event Replay Item` rows.
- One `Run History Document` has zero or many `Run Change Record` rows.
- One `Run History Document` has one `Run History Search Document`.
- One active `Replay Session` references one `Run History Document` and many ephemeral `Replay Checkpoint` entries.
- One `Run Export Document` belongs to one `Run History Document`.