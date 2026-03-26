# Research: Codebase Awareness

## Decision 1: Persist approval requests in SQLite before extending any repo-aware tool surface

- Decision: Add a dedicated persisted approval-request model in SQLite as the first design step. The canonical happy-path states are `proposed -> approved -> applied`, with additional terminal states for `rejected`, `blocked`, and `expired`. Each approval record stores the repository root, run and session identity, tool metadata, serialized request payload, review preview payload, and transition timestamps.
- Rationale: The current implementation holds pending approvals in memory only. That is incompatible with the feature requirement that write and command reviews survive UI closure or reconnect. Persisting the approval state machine first gives every later design choice a stable source of truth and prevents accidental execution after reconnect.
- Alternatives considered:
  - Keep approvals in memory and repopulate from run events: rejected because closing the UI mid-review would still lose the actionable pending state.
  - Persist only approved requests: rejected because the user explicitly requires pending review to survive disconnect and because rejected or blocked outcomes need an audit trail.
  - Reuse `agent_run_events` only: rejected because approval lookup and replay need direct queryable state, not event-log reconstruction on every bootstrap.

## Decision 2: Use `go-git` for repository validation, tree inspection, commit history, and working-tree diffs

- Decision: Introduce `github.com/go-git/go-git/v5` as the repository-awareness backend dependency for validating a connected repository, walking the repository tree, reading recent commits, and generating working-tree diffs, while keeping file-content reads in the existing filesystem-bound tools.
- Rationale: The user explicitly requires pure-Go Git integration and no system Git dependency. `go-git` keeps repository introspection inside the Go process and aligns with Relay's local, sandboxed design.
- Alternatives considered:
  - Shell out to `git`: rejected because the user explicitly ruled out a system Git dependency for repo structure and history features.
  - Parse `.git` files manually: rejected because it would be error-prone and incomplete for commit history and diff generation.
  - Store repository snapshots in SQLite: rejected because it would duplicate source-of-truth data and inflate local persistence.

## Decision 3: Keep `run_command` on `os/exec`, but revalidate the repository root before every execution and never accept a caller-supplied working directory

- Decision: Continue using `os/exec` for the `run_command` tool, but tighten execution so the tool resolves the configured repository root to a canonical absolute path on every invocation, confirms it still points to the currently connected Git repository, and pins `Cmd.Dir` to that validated root. The tool contract does not accept a custom working directory.
- Rationale: The user explicitly requires sandboxed command execution with per-execution validation to prevent repository escape. Revalidating on every call closes the gap between initial connection and later execution if the path changes, symlinks move, or a stale approval is replayed.
- Alternatives considered:
  - Validate only when the repository is connected: rejected because the repo path could change before the command actually runs.
  - Allow relative caller-selected directories inside the repo: rejected because it complicates validation and expands the attack surface without adding user-visible value to the initial feature.
  - Route commands through `go-git`: rejected because command execution is not a Git operation and the user explicitly allows `os/exec` for this part.

## Decision 4: Implement the in-product folder picker as a server-backed directory browser that populates `project_root`

- Decision: Keep `project_root` in config as the connected-repository source of truth and add a dedicated server-backed directory-browsing flow for the UI. The folder picker returns browsable local directories from the Relay server, lets the developer choose a folder, and then persists that path through the existing preferences save path.
- Rationale: Browser-native directory pickers do not reliably provide an absolute local filesystem path back to a localhost server. A server-backed picker keeps the choice actionable for backend tools and avoids introducing a desktop runtime outside the current stack.
- Alternatives considered:
  - Browser-native directory picker only: rejected because it does not reliably expose a reusable absolute path for server-side repo tools.
  - Text-input path entry only: rejected because the user explicitly asked for command flag or folder picker, and a browse flow lowers configuration friction.
  - Native OS dialog via shelling out: rejected because it would add platform-specific behavior and blur the command-approval boundary.

## Decision 5: Send write approvals to the browser as Monaco-ready before/after content plus metadata, not just a plain unified diff string

- Decision: Extend the approval payload for `write_file` so the frontend receives the normalized target path, optional original content, proposed content, and diff metadata needed to drive Monaco Editor in side-by-side diff mode. A unified diff string may still be included for audit or fallback rendering, but Monaco is driven from before/after text.
- Rationale: The requested reviewer experience is a side-by-side Monaco diff viewer. Before/after content is the most direct input for Monaco diff mode and also supports stale-file detection when combined with a base content hash.
- Alternatives considered:
  - Show a plain text diff only: rejected because it does not satisfy the requested review experience.
  - Send only the proposed content and recompute the original in the browser: rejected because the browser should not read the filesystem directly and the approval preview must remain exact.
  - Apply the change speculatively and diff in memory on the frontend: rejected because no write may occur before approval.

## Decision 6: Build repository context in a cancellable background goroutine and cache it in memory by repository signature

- Decision: Start repository relationship analysis asynchronously on first valid repository connect. The worker runs in a cancellable goroutine, publishes loading and completion states over the existing WebSocket channel, and caches results in memory keyed by repository root plus a lightweight repository signature such as HEAD commit and dirty-state markers.
- Rationale: The user explicitly requires that repository-context construction never block the main request path. An in-memory cache satisfies the requirement to cache results without introducing additional persistence complexity, while reconnects in the same process can reuse the derived context.
- Alternatives considered:
  - Build the analysis synchronously during bootstrap: rejected because it can stall the workspace for large repositories.
  - Recompute the analysis every time the sidebar opens: rejected because it repeats expensive work and creates visible lag.
  - Persist the derived context in SQLite: rejected because the feature requirement focuses on caching, not durable repository-context storage, and cached results may become stale quickly.

## Decision 7: Derive per-agent file activity from tool-call and approval events instead of adding a dedicated activity table

- Decision: Reuse existing run-event streaming and add richer repo-aware preview payloads so the frontend can derive each agent's read files, proposed-change files, and approval outcomes from `tool_call`, `approval_request`, `approval_state_changed`, and `tool_result` events.
- Rationale: Relay already has an append-only run event model and a live canvas store. Extending those payloads avoids a second mutable persistence surface for activity that is fundamentally run-scoped and replay-friendly.
- Alternatives considered:
  - Add a separate `agent_file_activity` table: rejected because it duplicates information already present in event payloads and complicates replay.
  - Track activity only in frontend memory: rejected because reconnect and replay would lose historical context.
  - Emit a synthetic summary event only at run end: rejected because the canvas needs live updates during the run.

## Implementation Notes

- `go-git` is the shipped repository-awareness backend dependency for validating `project_root`, listing repository directories, reading commit history, and generating Git diff context without invoking the system `git` executable.
- `monaco-editor` is the shipped frontend dependency for side-by-side `write_file` approval review, driven from persisted before and after content plus diff metadata.
- Relay streams `repository_graph_status` events as asynchronous background repository-context updates so frontend activity views can stay current without blocking the workspace.