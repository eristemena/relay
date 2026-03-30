# Data Model: Repository File Tree Sidebar

## Scope

This feature adds an in-memory repository tree cache, durable touched-file records for the active run, a live `file_touched` stream payload, and client-side replay-dock filter state.

## Repository Tree Cache

Represents the connected repository's directory structure held in memory by the workspace service.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `repository_root` | text | Connected repository preferences and validation | Required; absolute path to the connected local Git repository |
| `paths` | array of text | `go-git` recursive worktree iteration | Required; repository-relative normalized paths using forward slashes |
| `rebuilt_at` | timestamp | Workspace service | Required; tracks when the cache was last regenerated |
| `status` | enum | Workspace service | `loading`, `ready`, or `error` |
| `message` | text nullable | Workspace service | Human-readable status or failure text |

### Rules

- The cache is rebuilt when the connected repository root changes or becomes valid again.
- The cache is not persisted to SQLite.
- The path list includes the full recursive repository structure after `.gitignore` filtering, but the client may render only a shallow subset initially.

## Tree Entry Projection

Represents one client-visible folder or file row derived from the flat path list.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `path` | text | Repository tree cache | Required; repository-relative canonical path |
| `name` | text | Derived from path basename | Required |
| `kind` | enum | Derived from path shape | `file` or `directory` |
| `depth` | integer | Derived from slash count | Required and non-negative |
| `parent_path` | text nullable | Derived from path | Null only for root-level entries |
| `is_expanded` | boolean | Client UI state | Relevant only for directories |
| `is_visible` | boolean | Client UI state | Derived from expansion state and current filter |

### Rules

- The client derives hierarchy from the flat path list; the backend does not need to persist nested children blobs.
- Initial visibility is limited to depth `0` and `1` until the developer expands deeper folders.
- Tree entries remain read-only regardless of focus or selection state.

## Touched File Record

Represents one deduplicated file touch persisted for a run and agent.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Active run context | Required; foreign key to the relevant run |
| `agent_id` | text | Active agent execution context | Required |
| `file_path` | text | Tool execution path | Required; repository-relative canonical path |
| `touch_type` | enum | Tool execution path | `read` or `proposed` |

### Rules

- The logical identity is `(run_id, agent_id, file_path, touch_type)` so duplicate touches can be ignored safely.
- `read` is recorded when `read_file` completes successfully.
- `proposed` is recorded when a `write_file` proposal is created, before approval.
- The table stores touch presence, not approval outcome; approval state remains in existing approval records.

## Touched File Store Snapshot

Represents the current run's touched-file state as hydrated into the frontend store.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Active run context | Required |
| `by_agent` | map | Derived from touched file records and `file_touched` events | Keys are `agent_id`; values are sets of touched file records |
| `all_paths` | set of text | Derived from all agent maps | Required |
| `selected_agent_id` | text nullable | Canvas selection state | Null means workspace-wide view |

### Rules

- The client keeps one workspace-wide union view and zero or one selected-agent filter view at a time.
- The agent-specific view is always derived locally from `by_agent`; no extra backend request is required.
- Reconnect hydration loads the current run's touched snapshot before live updates resume.

## File Touched Event

Represents the real-time stream message emitted when a new touch is recorded.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `session_id` | text | Run execution context | Required |
| `run_id` | text | Run execution context | Required |
| `agent_id` | text | Run execution context | Required |
| `role` | text | Run execution context | Required |
| `file_path` | text | Tool execution path | Required; repository-relative canonical path |
| `touch_type` | enum | Tool execution path | `read` or `proposed` |
| `replay` | boolean | Stream metadata | `false` for live runs; may be reused later for replayed historical state if needed |

### Rules

- The event is emitted only after the touch has been accepted for persistence or deduped successfully.
- The event is independent of approval-result messages; a `proposed` touch does not imply the file was written.
- Events must arrive quickly enough that the sidebar updates feel contemporaneous with the tool activity.

## Right Rail Tab State

Represents the developer's active right-rail detail mode.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `active_tab` | enum | Client UI state | `replay` or `repository_tree` |
| `selected_agent_id` | text nullable | Canvas selection state | Null for full tree |
| `expanded_paths` | set of text | Client UI state | Stores expanded directories only |
| `tree_status` | enum | Tree cache hydration plus touched sync state | `loading`, `ready`, `error`, or `empty` |
| `message` | text nullable | Client UI state | Plain-language loading, empty, or error text |

### Rules

- Changing top-level right-rail tabs must not clear the selected run, current canvas state, or left-side run-history browsing state.
- Clearing the selected agent restores the workspace-wide touched union without rebuilding the tree.
- Tree expansion state should survive live touch updates and reconnect within the same workspace session.

## Relationships

- One `Repository Tree Cache` produces many `Tree Entry Projection` rows in the client model.
- One run has zero or many `Touched File Record` rows.
- One agent has zero or many `Touched File Record` rows within a run.
- The `Touched File Store Snapshot` is built from many `Touched File Record` rows plus incremental `File Touched Event` messages.
- One `Right Rail Tab State` can reference zero or one selected agent filter at a time when both Historical Replay and File Tree are available for a reopened saved run.