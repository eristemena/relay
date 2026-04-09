# Contract: Workspace Project Context

## Purpose

Define the external interfaces for project-aware workspace bootstrap, project switching, history scope selection, and CLI startup root resolution.

## CLI Contract

### Command

`relay serve [--root ABSOLUTE_PATH]`

### Resolution Rules

1. If `--root` is provided, Relay resolves that path with `filepath.Abs` and `filepath.Clean` and uses it as the startup project root.
2. If `--root` is not provided, Relay resolves the current working directory the same way and uses that value.
3. A supported project root for this feature is an existing, readable directory path that Relay can bind as the active repository and workspace scope; this feature does not require Git metadata or project-manifest detection.
4. If the selected root cannot be resolved, accessed, or validated against that rule, startup fails with a human-readable error.

### Notes

- `--root` is the primary public flag for this feature.
- The resolved root determines which persisted project context is opened or created before the first workspace bootstrap reaches the frontend.

## Workspace Bootstrap Extension

### Message

`workspace.bootstrap`

### Added Payload Fields

```json
{
  "active_project_root": "/absolute/project/root",
  "known_projects": [
    {
      "project_root": "/absolute/project/root",
      "label": "relay",
      "is_active": true,
      "is_available": true,
      "last_opened_at": "2026-04-05T12:34:56Z",
      "blocked_reason": ""
    }
  ]
}
```

### Semantics

- `active_project_root` is always present once bootstrap succeeds.
- `known_projects` contains all switchable project roots known to the workspace.
- Existing workspace bootstrap fields remain present unless this feature explicitly replaces them with project-aware equivalents.
- Internal persistence identifiers may still exist behind the scenes, but they are not required as part of the project-switching contract.

## Project Switch Contract

### Request Message

`project.switch.request`

```json
{
  "project_root": "/absolute/project/root"
}
```

### Success Response

- The server returns a fresh `workspace.bootstrap` payload for the selected project root.
- The response reflects the new `active_project_root` and the selected project's scoped run summaries, repository state, and other workspace data.

### Failure Response

`error`

```json
{
  "code": "project_switch_blocked",
  "message": "Finish or stop the active run before switching projects.",
  "project_root": "/absolute/project/root"
}
```

### Failure Codes

- `project_switch_blocked`: the current active project still has a non-terminal run.
- `project_not_found`: the requested known project root is no longer available.
- `project_root_invalid`: the requested path cannot be resolved or validated.

## Run History Query Extension

### Request Message

`run.history.query`

### Added Payload Field

```json
{
  "all_projects": false,
  "query": "token budget",
  "file_path": "web/src/features/workspace-shell/WorkspaceShell.tsx",
  "date_from": "2026-04-01T00:00:00Z",
  "date_to": "2026-04-05T23:59:59Z"
}
```

### Semantics

- Existing request fields remain available for compatibility if the protocol still carries them, but this feature adds only `all_projects`.
- When `all_projects` is `false` or omitted, the backend filters history to the active project's canonical root.
- When `all_projects` is `true`, the backend removes only the project-root predicate and applies the remaining filters normally.

### Response Expectations

- All-project results include `project_root` for each returned run and may include `project_label` so the frontend can label each run with its source project.
- The active project root remains unchanged regardless of the query mode.

### Result Shape Expectation

Each run summary returned in all-project mode includes:

```json
{
  "id": "run_123",
  "generated_title": "Fix workspace history filter",
  "project_root": "/absolute/project/root",
  "project_label": "relay"
}
```

## Migration and Local State Assumption

- This feature assumes a fresh local Relay database created with the multi-project schema.
- Relay may still apply the schema change through the normal SQLite migration mechanism for clean initialization, but compatibility with older pre-feature local databases is not part of this feature contract.