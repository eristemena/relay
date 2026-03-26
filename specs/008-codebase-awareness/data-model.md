# Data Model: Codebase Awareness

## Scope

This feature adds one persisted approval state machine, expands the connected-repository runtime context, and introduces derived frontend views for background repository context and per-agent file activity.

## Connected Repository Context

Represents the single repository currently connected to Relay.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `rootPath` | text | Config `project_root` | Absolute canonical path to the connected repository root |
| `configured` | boolean | Config | True when `project_root` is non-empty |
| `validGitRepo` | boolean | Derived via `go-git` | True only when the path resolves to a readable local Git repository |
| `validationMessage` | text | Derived | Plain-language status shown in UI when invalid or unavailable |
| `headRef` | text nullable | Derived via `go-git` | Current HEAD reference when available |
| `headCommit` | text nullable | Derived via `go-git` | Commit hash used for graph-cache invalidation |
| `graphStatus` | enum | Runtime | One of `idle`, `building`, `ready`, `error` |

### Rules

- Exactly one connected repository may be active at a time.
- Repo-aware tools are unavailable unless `validGitRepo` is true.
- Changing `rootPath` invalidates outstanding approvals and cached graph results tied to the previous repository.

## Approval Request

Represents a persisted approval-gated mutation request.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `toolCallId` | text | Existing tool call identity | Stable identifier across request, decision, and application |
| `sessionId` | text | Run context | Required |
| `runId` | text | Run context | Required |
| `agentId` | text nullable | Derived from run event context | Present for orchestration events when available |
| `role` | enum | Run context | Planner, coder, reviewer, tester, or explainer |
| `model` | text | Run context | Recorded for audit and replay |
| `repositoryRoot` | text | Connected repository | Canonical repo root at proposal time |
| `toolName` | enum | Tool executor | `write_file` or `run_command` |
| `requestKind` | enum | Derived | `file_write` or `command_execution` |
| `status` | enum | Persisted | `proposed`, `approved`, `applied`, `rejected`, `blocked`, or `expired` |
| `requestPayloadJson` | json | Tool executor | Exact normalized input used for later execution |
| `previewPayloadJson` | json | Tool executor | Review-ready preview payload sent to UI |
| `proposedAt` | timestamp | Tool executor | Required |
| `decidedAt` | timestamp nullable | Approval resolution | Set on approve or reject |
| `appliedAt` | timestamp nullable | Tool execution | Set only after successful execution |
| `decisionSource` | text nullable | Handler | Identifies developer approval action |
| `failureMessage` | text nullable | Tool executor | Plain-language reason for blocked, expired, or failed application |

### Rules

- The canonical happy path is `proposed -> approved -> applied`.
- `rejected`, `blocked`, and `expired` are terminal states.
- `appliedAt` may only be set when `status` becomes `applied`.
- An approval request may only be applied if the current connected repository root matches `repositoryRoot`.
- Bootstrap and reconnect flows must query persisted requests whose status is still `proposed` or `approved`.

## Approval Request State Machine

| From | To | Trigger | Notes |
|------|----|---------|-------|
| `proposed` | `approved` | Developer approves | Request stays persisted until execution completes |
| `proposed` | `rejected` | Developer rejects | Terminal |
| `proposed` | `blocked` | Validation fails before decision or replay detects invalid repo/run state | Terminal |
| `proposed` | `expired` | Run ends, repository changes, or request becomes stale | Terminal |
| `approved` | `applied` | Tool executes successfully after revalidation | Terminal happy-path completion |
| `approved` | `blocked` | Revalidation fails before execution | Terminal |
| `approved` | `expired` | Request is no longer safe to apply | Terminal |

## Proposed Change View

Represents a write-file approval preview derived from an approval request.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `toolCallId` | text | Approval request | Foreign key to approval request |
| `targetPath` | text | Normalized tool input | Relative path inside repository |
| `originalContent` | text nullable | Backend preview generation | Source text shown on left side of Monaco |
| `proposedContent` | text | Backend preview generation | Source text shown on right side of Monaco |
| `baseContentHash` | text nullable | Backend preview generation | Used to detect stale files before apply |
| `diffSummary` | json | Backend preview generation | Line counts and file metadata for badges |

### Rules

- `targetPath` must remain within the connected repository.
- If the file content no longer matches `baseContentHash` at apply time, the request transitions to `blocked` or `expired`; it does not write stale content.
- The UI may render fallback text if Monaco is unavailable, but the payload remains the same.

## Command Proposal View

Represents a run-command approval preview derived from an approval request.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `toolCallId` | text | Approval request | Foreign key to approval request |
| `command` | text | Tool input | Required executable name |
| `args` | list of text | Tool input | Ordered arguments |
| `effectiveDir` | text | Connected repository | Always the canonical repo root |
| `commandPreview` | text | Backend preview generation | Human-readable summary shown in review UI |

### Rules

- `effectiveDir` is not user-specified and must be revalidated before execution.
- The review UI shows the exact command and argument vector that will run after approval.

## Repository Context Cache

Represents the asynchronous repository relationship analysis result held in memory.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `repositoryRoot` | text | Connected repository | Cache key component |
| `repositorySignature` | text | Derived via `go-git` | Changes when HEAD or tracked worktree signature changes |
| `status` | enum | Runtime | `idle`, `building`, `ready`, or `error` |
| `nodes` | list | Derived by analysis goroutine | Files or modules included in the graph |
| `edges` | list | Derived by analysis goroutine | Import/dependency relationships |
| `generatedAt` | timestamp nullable | Runtime | Set on successful build |
| `errorMessage` | text nullable | Runtime | Plain-language error when analysis fails |

### Rules

- Repository-context analysis runs off the main request path in a cancellable goroutine.
- Cache entries are invalidated when the connected repository changes or the repository signature changes.
- Partial relationship snapshots are valid when Relay can derive only some relationships confidently.

## Agent File Activity

Represents the run-scoped file awareness shown on each agent node.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `runId` | text | Run context | Required |
| `agentId` | text | Run event context | Required |
| `readPaths` | set of text | Derived from repo-aware tool events | Deduplicated within a run |
| `proposedChangePaths` | set of text | Derived from approval and tool events | Deduplicated within a run |
| `approvalStates` | map | Derived from approval state events | Latest known state per `toolCallId` |

### Rules

- File activity is derived from persisted event payloads and approval records; it is not its own mutable table.
- The canvas must be able to rebuild agent file activity on replay or reconnect.
- Only repository-relative paths are shown in UI.