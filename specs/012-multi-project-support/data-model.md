# Data Model: Multi-Project Support

## Scope

This feature adds project-root identity to sessions, introduces a known-project view for workspace switching, extends history query scope, and defines the frontend reset semantics required when the active project changes.

## Session

Represents the persisted workspace context for one project root.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `id` | text | Existing session storage | Required; stable session identifier |
| `display_name` | text | Existing session metadata | Required; remains user-visible summary text for the session |
| `project_root` | text | Startup root resolver and project switch flow | Required for multi-project sessions; canonical absolute path using `filepath.Abs` + `filepath.Clean` |
| `created_at` | timestamp | Existing session storage | Required |
| `updated_at` | timestamp | Existing session storage | Required |
| `last_opened_at` | timestamp | Existing session storage | Required |
| `status` | enum | Existing session storage | `active`, `idle`, or `archived` |
| `snapshot_json` | json | Existing session storage | Existing persisted workspace snapshot |

### Rules

- `project_root` is the identity boundary for auto-created sessions.
- Each canonical `project_root` maps to at most one active session record in this feature.
- This feature assumes a fresh SQLite database created with the multi-project schema; upgrade and backfill behavior for older session rows is out of scope.

## Known Project Root

Represents one switchable project in the workspace UI.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `project_root` | text | Session table | Required; canonical absolute path |
| `session_id` | text | Session table | Required; current session for that root |
| `label` | text | Derived from path or session metadata | Required; human-readable switcher label |
| `is_active` | boolean | Workspace bootstrap | Exactly one known project is active |
| `is_available` | boolean | Runtime validation | False when the path no longer exists or cannot be opened |
| `last_opened_at` | timestamp | Session table | Required for ordering |
| `blocked_reason` | text nullable | Orchestrator | Set when switching is temporarily disallowed |

### Rules

- The switcher lists known projects ordered by recent use.
- `blocked_reason` is populated when the active run policy prevents switching.
- Availability is runtime state and does not delete the project record.

## Active Project Context

Represents the project-scoped workspace state currently rendered in the UI.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `active_project_root` | text | Bootstrap or project switch response | Required |
| `active_run_id` | text nullable | Existing workspace bootstrap | Present only when the active project has a live run |
| `canvas_document_ids` | set of text | Frontend store | Must be cleared when the active project changes |
| `history_scope_mode` | enum | Frontend UI state | `active_project` or `all_projects` |
| `repository_root` | text | Existing connected repository state | Must align with `active_project_root` for the active project |

### Rules

- Only one `Active Project Context` exists at a time per Relay process.
- Changing `active_project_root` requires clearing project-scoped frontend caches before new project data is rendered.
- The all-project history mode does not change the active project context.

## Project Switch Request

Represents the user action that changes the active project root.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `project_root` | text | Header project switcher | Required; canonical absolute path |
| `requested_at` | timestamp | Frontend/request envelope | Required |

### Rules

- The orchestrator rejects the request if switching would create multiple simultaneous active runs across projects.
- A successful switch returns a full workspace snapshot for the target project.

## Run History Scope

Represents the filtering mode for the History tab.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `session_id` | text nullable | Existing history request | Optional compatibility field if the existing request shape still carries it |
| `project_root` | text | Active project context | Applied only when `all_projects` is false |
| `all_projects` | boolean | History UI toggle | False by default |
| `query` | text nullable | Existing history filters | Optional full-text search |
| `file_path` | text nullable | Existing history filters | Optional path filter |
| `date_from` | timestamp nullable | Existing history filters | Optional lower bound |
| `date_to` | timestamp nullable | Existing history filters | Optional upper bound |
| `project_root` | text | History query result | Required when `all_projects` is true so each run can be labeled with its source project |
| `project_label` | text nullable | History query result | Optional display-friendly label derived from path or project metadata |

### Rules

- Default mode is `all_projects = false`.
- When `all_projects = true`, history queries omit the `project_root` predicate but keep all other filters.
- Returned runs in all-project mode must include `project_root` and may include `project_label` so the history list can render unambiguous project identity.

## Workspace Bootstrap Project Payload

Represents the additional bootstrap data needed to render and switch projects.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `active_project_root` | text | Orchestrator bootstrap | Required |
| `known_projects` | array of Known Project Root | Orchestrator bootstrap | Required; may contain only the active project |

### Rules

- Bootstrap remains the source of truth after reconnect and after a successful project switch.
- The project switcher empty state is derived when `known_projects` contains only the active entry.

## Relationships

- One `Session` belongs to exactly one canonical `project_root`.
- One `Known Project Root` references exactly one current `Session` in this phase.
- One `Active Project Context` references one `Known Project Root` and may be backed by one active `Session` internally.
- One `Run History Scope` is evaluated against the current `Active Project Context`.
- One workspace bootstrap response delivers one `Active Project Context` plus many `Known Project Root` entries.