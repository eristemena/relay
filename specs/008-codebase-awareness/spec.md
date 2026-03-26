# Feature Specification: Codebase Awareness

**Feature Branch**: `008-codebase-awareness`  
**Created**: 2026-03-25  
**Status**: Draft  
**Input**: User description: "Codebase awareness for Relay — the developer connects Relay to a local git repository via a command flag or folder picker, and agents can then read files, list directories, search across all code, and see recent commit history and diffs. When an agent wants to modify a file it proposes the change as a diff, which the developer reviews in a side-by-side Monaco viewer and explicitly approves or rejects before anything is written to disk; the same approval flow applies to shell commands. Relay also derives repository context in the background so each agent node on the canvas can track which files it has read or proposed changes to. No file is written and no command is executed without an explicit developer approval action — this is a hard requirement, enforced server-side. Multi-repo support, automatic git commits, and remote repository access are out of scope."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Connect a local repository safely (Priority: P1)

As a developer, I can connect Relay to one local Git repository and let agents inspect its code and recent history without giving them permission to write files or run commands on their own.

**Why this priority**: Repository connection and read-only code awareness are the foundation for every higher-level workflow in this feature. Without a safe connected repository, the rest of the feature has no value.

**Independent Test**: Can be fully tested by starting Relay with a repository path or selecting a local repository in the UI, then confirming agents can browse files, search code, and view recent Git history while all write and command actions remain approval-gated.

**Acceptance Scenarios**:

1. **Given** Relay starts with a valid local Git repository path, **When** the workspace opens, **Then** the connected repository becomes the active codebase context for agent reads and Git inspection.
2. **Given** Relay starts without a repository path, **When** the developer selects a valid local Git repository folder, **Then** Relay connects that repository and exposes code-reading and recent-history capabilities to agents.
3. **Given** a repository is connected, **When** an agent requests to read a file, list a directory, search the codebase, or inspect recent commits or diffs, **Then** the request completes within the connected repository boundary only.
4. **Given** the selected folder is not a valid local Git repository, **When** the developer attempts to connect it, **Then** Relay rejects the connection with a plain-language error and does not enable repository-aware tools.

---

### User Story 2 - Review proposed writes and commands before execution (Priority: P2)

As a developer, I can review every proposed file change or shell command and explicitly approve or reject it before Relay writes to disk or executes anything.

**Why this priority**: Explicit approval is the core safety guarantee in the request. The feature fails its primary trust requirement if proposed mutations can bypass review.

**Independent Test**: Can be fully tested by asking an agent to modify a file and run a shell command, then confirming both requests stay pending until the developer approves them and that rejection leaves the repository unchanged.

**Acceptance Scenarios**:

1. **Given** an agent wants to modify a file, **When** it proposes the change, **Then** Relay presents the proposed diff in a side-by-side review surface and does not write the file until the developer explicitly approves it.
2. **Given** an agent wants to run a shell command, **When** it submits the request, **Then** Relay presents the pending command for developer review and does not execute it until the developer explicitly approves it.
3. **Given** a file change or command request is pending, **When** the developer rejects it, **Then** Relay records the rejection outcome and leaves the repository and process state unchanged.
4. **Given** an agent or client attempts to bypass the review flow, **When** the server receives the write or command request without an approval decision, **Then** the server denies execution.

---

### User Story 3 - Understand repository context and agent activity (Priority: P3)

As a developer, I can see how the connected codebase fits together and which files each agent has inspected or proposed to change so I can follow agent behavior with confidence.

**Why this priority**: Context visibility increases trust and usability, but it depends on repository connection and approval enforcement already being in place.

**Independent Test**: Can be fully tested by running an agent workflow on a connected repository, then confirming Relay derives repository context in the background and the canvas identifies which files each agent read or proposed to edit.

**Acceptance Scenarios**:

1. **Given** a repository is connected, **When** Relay derives repository relationships in the background, **Then** the workspace remains responsive while that context is prepared for repository-aware activity tracking.
2. **Given** an agent reads files while working, **When** the developer inspects that agent on the canvas, **Then** the agent node shows which files it read during the run.
3. **Given** an agent proposes edits to one or more files, **When** the developer inspects that agent on the canvas, **Then** the agent node shows which files it proposed to change and whether those proposals remain pending, approved, or rejected.
4. **Given** no repository is connected yet, **When** the developer opens repository-aware views, **Then** Relay shows explicit empty states that explain how to connect a local repository.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- The feature preserves Relay's required handler -> orchestrator -> agent -> tool -> storage flow by keeping repository connection, approval decisions, and execution authorization in the existing backend layers rather than allowing UI-only enforcement.
- Frontend work remains inside feature-based areas such as the canvas, sidebar, and workspace features, including repository connection states, review surfaces, and agent activity displays.
- WebSocket remains the only backend/frontend runtime channel. Protocol additions are allowed only for connected repository metadata, pending approval requests, approval outcomes, repository-context payloads, and per-agent file activity, with integration coverage for each changed event shape.

### Approval and Safety Impact

- This feature introduces additional file-write and shell-command approval surfaces, but execution remains blocked until the server records an explicit developer approval action for the specific request.
- Agent file-system and shell capabilities expand only within the connected local repository. Repo-root sandboxing, Git-repository validation, path traversal protection, and repo-root command confinement remain mandatory.

### UX States

- Repository-aware flows must show visible loading states while validating a selected repository, building repository context, loading Git history, and waiting on approval decisions.
- User-visible errors must explain invalid repository selections, unavailable Git history, repository-context preparation failures, rejected approvals, expired pending approvals, and blocked write or command attempts in plain language.
- Explicit empty states are required for no connected repository, no recent agent file activity, no pending approvals, and no recent commit history available in the connected repository.

### Edge Cases

- If Relay starts with a repository path that no longer exists or is no longer a Git repository, the workspace must remain disconnected and explain the problem without exposing repository-aware tools.
- If the developer switches repositories between runs, prior agent activity and pending approvals must not be applied to the newly connected repository.
- If a file changes on disk after an agent prepared a proposed diff but before approval, the approval flow must prevent silently writing outdated content.
- If a pending command approval becomes stale because the connected repository changed or the run ended, Relay must block execution and require a fresh request.
- If the repository is very large or has limited import metadata, Relay must degrade gracefully by deriving only the repository context it can support without blocking the rest of the workspace.
- If Git history or diff information is unavailable for a specific request, Relay must keep core code-reading capability available and show that the history view is temporarily unavailable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow the developer to connect exactly one local Git repository to Relay at a time.
- **FR-002**: The system MUST support establishing that repository connection from either a startup path input or an in-product folder selection flow.
- **FR-003**: The system MUST validate that the selected path is a local Git repository before enabling repository-aware agent capabilities.
- **FR-004**: The system MUST allow agents to read files, list directories, search code, inspect recent commit history, and inspect recent diffs within the connected repository.
- **FR-005**: The system MUST keep all agent file reads, directory listings, searches, Git history access, and diff access bounded to the connected repository.
- **FR-006**: The system MUST reject path traversal or repository-escape attempts for repository-aware agent actions.
- **FR-007**: The system MUST require an explicit developer approval decision before any agent-requested file write is executed.
- **FR-008**: The system MUST require an explicit developer approval decision before any agent-requested shell command is executed.
- **FR-009**: The system MUST present proposed file changes as reviewable diffs before approval.
- **FR-010**: The system MUST preserve the proposed file-change request unchanged between proposal and approval so the developer is approving the exact change that will be written.
- **FR-011**: The system MUST preserve the proposed shell command request unchanged between proposal and approval so the developer is approving the exact command that will be executed.
- **FR-012**: The system MUST allow the developer to explicitly approve or reject each pending file-write or shell-command request.
- **FR-013**: The system MUST deny file writes and shell commands on the server when no explicit approval has been recorded for that specific request.
- **FR-014**: The system MUST prevent a pending approval created for one repository or run from being executed against a different repository or run.
- **FR-015**: The system MUST show the developer the recent commit history and recent diffs for the connected repository.
- **FR-016**: The system MUST derive repository relationship context in the background without blocking the workspace so repository-aware activity views can stay current.
- **FR-017**: The system MUST track, per agent node, which repository files the agent read during the run.
- **FR-018**: The system MUST track, per agent node, which repository files the agent proposed to change during the run.
- **FR-019**: The system MUST show the current state of each proposed change or command request, including pending, approved, rejected, or blocked outcomes.
- **FR-020**: The system MUST preserve explicit disconnected, connecting, connected, approval-pending, approval-resolved, error, and empty states for repository-aware flows.
- **FR-021**: The system MUST show plain-language errors when a repository cannot be connected, a repository-aware action is unavailable, or an approval request cannot be executed.
- **FR-022**: The system MUST ensure repository-aware features remain unavailable when no repository is connected.
- **FR-023**: The system MUST NOT support multiple simultaneous repositories in this feature.
- **FR-024**: The system MUST NOT automatically commit, push, or otherwise mutate Git history as part of this feature.
- **FR-025**: The system MUST NOT access remote repositories as part of this feature.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST NOT bypass handler-level approval enforcement for file writes or shell commands.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped file-system access for agents.
- **CDR-003**: The feature MUST include tool tests for repository-aware read, search, Git-history, diff, file-write, and command approval happy and error paths; WebSocket integration tests for any protocol additions; and React Flow component or model tests for new agent-node activity indicators.
- **CDR-004**: The feature MUST define visible loading states, human-readable error messages, and explicit empty states for repository connection, approval review, repository-context preparation, Git history visibility, and agent activity views.
- **CDR-005**: The feature MUST document any new third-party dependency and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Connected Repository**: The single local Git repository that Relay validates, binds to the current workspace session, and uses as the boundary for repository-aware reads and approvals.
- **Approval Request**: A pending, repository-scoped request for either a file write or shell command that requires an explicit developer decision before execution.
- **Proposed Change**: The exact diff-based representation of an agent-requested file modification that is shown for developer review and becomes executable only after approval.
- **Repository Context Snapshot**: The repository-derived relationship metadata Relay can build in the background to support activity tracking and future repository-aware views.
- **Agent File Activity**: The per-agent record of which files were read and which files were proposed for change during a run.

## Assumptions

- The developer is operating on a local filesystem path that Relay can access directly from the same machine.
- Recent commit history and recent diffs refer to the connected repository's locally available Git metadata.
- Background repository-context derivation may be partial when Relay cannot confidently derive every relationship from the repository contents.
- Approval decisions are specific to a concrete request and are not reusable as blanket permission for later file writes or shell commands.

## Dependencies

- Existing handler-level approval enforcement must remain the authority for executing file writes and shell commands.
- Existing repository sandboxing and tool-layer path guarding must remain the enforcement boundary for agent file access.
- The workspace and canvas event model must be able to carry connected-repository status, approval-request lifecycle, and agent file activity updates to the frontend.

## Out of Scope

- Connecting or operating on more than one repository at the same time
- Automatically committing, rebasing, pushing, or otherwise changing Git history
- Accessing hosted or remote repositories
- Allowing file writes or command execution without explicit developer approval
- Broadening agent access beyond the connected repository boundary

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of validation checks, file writes and shell commands remain blocked until an explicit developer approval is recorded for the exact request being executed.
- **SC-002**: In at least 95% of validation runs on supported repositories, developers can connect a valid local Git repository and begin repository-aware agent reads within 30 seconds.
- **SC-003**: In 100% of validation checks, repository-aware agent actions are confined to the connected repository and rejected when they attempt repository escape.
- **SC-004**: In at least 90% of representative validation runs, developers can identify the files an agent read or proposed to change and review the related proposal state without leaving the active Relay workspace.
- **SC-005**: In 100% of validation checks, disconnected, invalid-repository, pending-approval, rejected-approval, and unavailable-history states show explicit user-facing feedback rather than silent failure or blank UI.
