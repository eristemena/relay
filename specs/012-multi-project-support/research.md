# Research: Multi-Project Support

## Decision 1: Canonicalize project roots as cleaned absolute paths before any lookup or persistence

- Decision: Resolve every project root with `filepath.Abs` followed by `filepath.Clean` in Go, and store and compare only that canonical absolute string in SQLite, workspace bootstrap state, and project-switch payloads.
- Rationale: The main correctness risk in this feature is treating two launch paths that refer to the same directory as different projects. Canonical cleaned absolute strings are enough to make session lookup deterministic across startup, reconnect, and project switching without introducing a broader schema overhaul.
- Alternatives considered:
  - Compare raw incoming path strings: rejected because relative paths, redundant separators, and directory traversal segments would produce duplicate project identities.
  - Canonicalize only in the frontend: rejected because the backend owns session creation, persistence, and repository safety boundaries.
  - Use `filepath.EvalSymlinks` for identity: rejected for the initial design because it can fail for moved or temporarily unavailable paths and is not required by the feature request.

## Decision 2: Add `project_root` to `sessions` and keep project scoping session-based rather than redesigning runs or events

- Decision: Extend the `sessions` table with a `project_root` column plus lookup queries by canonical root. Runs, events, approvals, replay artifacts, and touched files remain keyed by `session_id`; project scoping for history becomes a join or `WHERE` clause refinement rather than a new project table or run schema rewrite.
- Rationale: The user explicitly noted that all relevant runtime records are already associated with `session_id`. Adding the root to sessions is the narrowest change that supports automatic session creation, project-aware reconnect, and filtered history without duplicating project metadata across multiple tables.
- Alternatives considered:
  - Introduce a new `projects` table and foreign keys from sessions and runs: rejected because it adds migration and coordination complexity without unlocking a needed capability in this phase.
  - Copy `project_root` onto runs, events, and touched-file rows: rejected because it duplicates session-scoped data and creates avoidable consistency risk.
  - Keep one global session and tag runs by project root only: rejected because the feature requires preserved project-scoped workspace context, not just filtered history.

## Decision 3: Resolve the startup project root with `--root` first, then current working directory, and fail closed on invalid roots

- Decision: `relay serve` will accept a new `--root` flag as the primary startup selector. Startup resolution order is `--root` value, then current working directory, else an explicit error. The resolved path is canonicalized, validated, and used to select or create the active project session before the first workspace bootstrap.
- Rationale: This matches the requested startup behavior while keeping project identity deterministic. Failing on invalid or inaccessible explicit roots avoids silently booting into the wrong project, which would undermine history isolation.
- Alternatives considered:
  - Keep relying on saved config `project_root` as the primary selector: rejected because the feature request makes current-process startup resolution authoritative.
  - Fall back from an invalid `--root` to the current working directory: rejected because it hides operator error and can connect the wrong project.
  - Keep only the existing `--project-root` flag: rejected because the feature request explicitly standardizes on `--root`.

## Decision 4: Keep project enumeration and switching on the existing workspace WebSocket protocol rather than introducing `GET /api/projects`

- Decision: Extend workspace bootstrap and request/result messages to expose known project roots and perform project switching, instead of adding a new `GET /api/projects` HTTP endpoint.
- Rationale: Relay's constitution requires WebSocket as the only Go-to-React communication channel. The codebase already uses request/result messages such as `repository.tree.request` and `run.history.query`, so listing known projects and switching the active one fit the current transport model cleanly.
- Alternatives considered:
  - Add `GET /api/projects` exactly as requested: rejected because it introduces a second frontend/backend transport path for data the workspace socket can already provide.
  - Encode known projects only in a static frontend config file: rejected because project availability and active selection are runtime workspace state.
  - Reuse `session.open` directly from the project switcher with no project abstraction: rejected because the switcher must be project-root based and manual session semantics are out of scope for developers.

## Decision 5: Scope run history by joining run-history documents back to sessions and let the all-project toggle remove only the project-root predicate

- Decision: Keep run-history documents keyed by `session_id`, join them to sessions to filter by `project_root` for the default active-project view, and remove only that root predicate when the developer enables the all-project history mode.
- Rationale: This produces the requested behavior with minimal change. Search, file-path, and date filters remain exactly as they are today; the only new dimension is whether the active project root is applied as a filter.
- Alternatives considered:
  - Duplicate run-history documents per project root: rejected because the session already identifies the project.
  - Fetch all runs and filter them in the browser: rejected because it wastes query work and weakens the source of truth for project isolation.
  - Maintain a separate aggregated history table for all-project mode: rejected because the toggle only relaxes one query predicate.

## Decision 6: Treat project switching as a full project-context reset in the frontend store, not just a canvas reset helper call

- Decision: Reuse the existing empty canvas document helper for per-run reset semantics, but add an explicit project-switch reset path in the workspace store that clears project-scoped run events, transcripts, run-history results and details, replay state, export state, approvals, repository tree state, and orchestration documents when the active project changes.
- Rationale: The current store keeps those maps across bootstrap updates, which is safe for reconnect but unsafe for switching projects. The existing replay reset helper already returns an empty `nodes` and `edges` document, so ghost nodes are not caused by the helper itself; they are caused by preserving run-keyed documents across a new active project.
- Alternatives considered:
  - Rely only on `resetCanvasDocumentForReplay()` during project switch: rejected because it resets one run document but does not clear stale run IDs or history state from the store.
  - Call React Flow `setNodes([])` and `setEdges([])` imperatively from components: rejected because the authoritative state lives in the workspace store and canvas model, not in ad hoc component side effects.
  - Leave existing store retention behavior in place and hope run IDs do not collide: rejected because stale state would still surface the wrong project's nodes, details, or replay metadata.

## Implementation Notes

- `internal/storage/sqlite/queries/sessions.sql` currently persists sessions without `project_root`; migration and store scan/create/open paths are the narrow backend seam for project identity.
- `internal/storage/sqlite/store.go` currently constrains run history with `d.session_id = ?`; project-aware filtering can be implemented by joining sessions and making the root predicate optional for all-project mode.
- `internal/orchestrator/workspace/service.go` already centralizes bootstrap, session creation, session opening, and active-run guards, which makes it the right place to auto-provision project sessions and block cross-project switching while a run remains active.
- `web/src/shared/lib/workspace-store.ts` currently preserves orchestration documents and run-history caches across snapshot changes; project switching must deliberately clear those maps when the active project root or active session changes.
- `web/src/features/workspace-shell/WorkspaceShell.tsx` already coordinates header controls, history loading, repository tree requests, and run selection, so the project switcher belongs there rather than in a separate global shell abstraction.