# Feature Specification: Multi-Project Support

**Feature Branch**: `012-multi-project-support`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Multi-project support for Relay so developers can work across multiple codebases without losing history or context. Each project root gets its own session, created automatically the first time that root is connected — the developer never creates sessions manually. The active project root is always visible in the UI, and a project switcher lets the developer move between known roots without restarting Relay; switching updates the canvas, run history, and file tree to reflect the selected project. The History tab shows runs for the active project by default, with an opt-in view across all projects. relay serve defaults to the current working directory as the project root, or accepts an explicit --root flag. Managing multiple simultaneous active runs across different projects, and project archiving or deletion, are out of scope."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Work in the right project without manual setup (Priority: P1)

As a developer, I can start Relay against a project root and have Relay automatically restore or create that project's own saved workspace context so I can begin work in the correct codebase without manual setup first.

**Why this priority**: Automatic project-scoped context creation and reuse is the foundation of the feature. If Relay cannot reliably bind a root to its own saved workspace state without manual setup, multi-project support does not exist in a usable form.

**Independent Test**: Can be fully tested by starting Relay from a project directory with no prior saved context for that root and confirming that project-scoped state is created automatically, the active root is shown in the UI, and no manual setup step is required. The developer is not required to click any "new session" or "open session" control to begin work in the resolved project root.

**Acceptance Scenarios**:

1. **Given** Relay starts from a working directory and no explicit root is supplied, **When** the server initializes, **Then** Relay treats the current working directory as the active project root.
2. **Given** Relay starts with an explicit project root, **When** the server initializes, **Then** Relay uses that root as the active project root instead of the current working directory.
3. **Given** a project root is connected for the first time, **When** Relay finishes initializing that root, **Then** Relay automatically creates persisted project-scoped workspace state for that root without asking the developer to create it manually.
4. **Given** a project root has been connected before, **When** Relay reconnects to that same root, **Then** Relay reuses that root's existing preserved workspace context instead of creating duplicate project state.
5. **Given** Relay has an active project, **When** the main workspace UI is visible, **Then** the active project root is continuously visible in the interface.

---

### User Story 2 - Switch between known project roots without restarting Relay (Priority: P2)

As a developer, I can use a project switcher to move between known project roots so I can continue work across codebases without restarting Relay or losing each project's separate context.

**Why this priority**: Once project-scoped saved contexts exist, the next highest-value workflow is moving between them in one running Relay instance. That is what makes the feature practical day to day.

**Independent Test**: Can be fully tested by connecting at least two project roots, switching between them from the UI, and confirming that the canvas, run history, and file tree each rehydrate to the selected root's saved context without requiring a restart.

**Acceptance Scenarios**:

1. **Given** Relay knows more than one previously connected project root, **When** the developer opens the project switcher, **Then** the switcher lists those known roots and indicates which one is currently active.
2. **Given** the developer selects a different known project root, **When** the switch completes, **Then** the canvas updates to the selected root's project-scoped state rather than showing data from the previously active root.
3. **Given** the developer selects a different known project root, **When** the switch completes, **Then** the run history and file tree update to the selected root's scoped data in the same transition.
4. **Given** the active project has a run that has not reached a terminal state, **When** the developer attempts to switch to another project root, **Then** Relay prevents the switch and explains that cross-project simultaneous active runs are out of scope.
5. **Given** a known project root is no longer reachable, **When** the developer tries to activate it, **Then** Relay leaves the current project unchanged and shows a human-readable error.

---

### User Story 3 - Review history in the correct project scope (Priority: P3)

As a developer, I can see runs for the active project by default and opt into an all-project history view when needed so I can stay focused on the current codebase without losing a broader view of past work.

**Why this priority**: History scoping is where context separation becomes visible and trustworthy. It depends on project-scoped persistence and switching already being in place, but it is required to prevent cross-project confusion.

**Independent Test**: Can be fully tested by creating runs under multiple project roots, opening the History tab for one active root, confirming only that root's runs are shown by default, and then enabling the all-project view to see the combined run list without changing the active project.

**Acceptance Scenarios**:

1. **Given** multiple project roots have saved runs, **When** the developer opens the History tab for the active project, **Then** the list shows only runs associated with that active project by default.
2. **Given** the developer enables the all-project history view, **When** the History tab refreshes, **Then** Relay shows runs across known projects while preserving clear project identity for each run.
3. **Given** the developer disables the all-project history view, **When** the History tab refreshes, **Then** Relay returns to showing only runs for the active project.
4. **Given** the developer is using the all-project history view, **When** they review that list, **Then** the currently active project root remains unchanged until they explicitly switch projects.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature must extend Relay's existing handler -> orchestrator -> agent -> tool -> storage flow by making project root a first-class workspace scope for project-context selection, hydration, and history retrieval rather than introducing direct frontend access to project data or bypassing orchestrator state management.
- Frontend changes must remain within existing feature-based areas such as workspace shell, history, canvas, and repository surfaces. The feature adds project-aware state selection and switching behavior, not a new type-based top-level structure.
- WebSocket remains the only backend/frontend communication channel. The protocol will need coverage for active-project summary data, known-project listings, project-switch requests and responses, and project-scoped hydration of canvas, history, and file-tree state.

### Approval and Safety Impact

- This feature does not introduce new file writes or shell command execution paths by itself. Starting Relay with a project root, creating project-scoped saved context automatically, and switching active projects are metadata and state-selection operations, not approval-requiring side effects.
- Existing handler-level approval enforcement for file writes and shell commands remains unchanged and must continue to apply within whichever project root is active.
- Repository sandboxing and path traversal protection must remain tied to the selected project root so agent file access and shell execution stay bounded to the active project and cannot bleed into another known root.
- Because simultaneous active runs across different projects are out of scope, Relay must not allow a project switch that would create two active project contexts at once.

### UX States

- The workspace shell must show visible loading states for initial project-root hydration, project-switch transitions, and project-scoped restoration of canvas, history, and file-tree data.
- The feature must show human-readable errors when a requested project root cannot be opened, when a known project is unavailable, when project-scoped history cannot be loaded, or when a switch is blocked by an active run.
- The project switcher must show an explicit empty state when Relay only knows the current project and there are no alternate roots to switch to.
- The History tab must show explicit empty states for a project with no runs and for an all-project view with no saved runs at all.

### Edge Cases

- If two different launch paths resolve to the same canonical project root, Relay must treat them as one known project rather than creating duplicate saved project contexts.
- If the developer starts Relay with an explicit root that does not exist or is not accessible, Relay must fail with a clear error instead of silently falling back to another root.
- If the active project has no canvas state, no file tree, or no prior runs yet, Relay must show project-scoped empty states rather than stale data from another project.
- If the developer switches from a project with rich history to one with none, Relay must clear project-specific surfaces before rendering the new root's state.
- If a project root has been renamed, moved, or deleted outside Relay, the switcher must not activate stale context without first confirming the root is still valid.
- If the developer is viewing all-project history while switching active projects, Relay must update the active-project indicator and project-scoped surfaces without silently disabling the developer's chosen history filter.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST treat a project root as the scope boundary for persisted workspace context, run history, canvas state, and repository file-tree state.
- **FR-002**: The system MUST automatically create new project-scoped persisted workspace context the first time a project root is connected.
- **FR-003**: The system MUST NOT require the developer to manually create, name, or initialize project-scoped persisted workspace context for a project root.
- **FR-004**: The system MUST reuse existing project-scoped persisted workspace context when a previously known project root is connected again.
- **FR-005**: The system MUST preserve separate history and context for different project roots so one project's runs and workspace state do not overwrite another project's data.
- **FR-006**: The system MUST show the active project root in the workspace UI at all times.
- **FR-007**: The system MUST provide a project switcher that lists known project roots and indicates which root is active.
- **FR-008**: The system MUST allow the developer to switch the active project root without restarting Relay.
- **FR-009**: The system MUST update the canvas to the selected project's scoped state when the active project changes.
- **FR-010**: The system MUST update the run history to the selected project's scoped data when the active project changes.
- **FR-011**: The system MUST update the repository file tree to the selected project's scoped data when the active project changes.
- **FR-012**: The system MUST clear or replace project-scoped UI state during a switch so stale data from the previously active project is not shown as if it belonged to the newly active project.
- **FR-013**: The system MUST remember known project roots so the developer can switch back to them in later Relay launches.
- **FR-014**: The system MUST default the History tab to runs belonging to the active project root.
- **FR-015**: The system MUST provide an explicit opt-in history view that aggregates runs across all known projects.
- **FR-016**: The system MUST preserve clear project identity for each run shown in the all-project history view.
- **FR-017**: The system MUST NOT change the active project root merely because the developer enables or reviews the all-project history view.
- **FR-018**: The system MUST default `relay serve` to using the current working directory as the project root when no explicit root is supplied.
- **FR-019**: The system MUST allow `relay serve` to accept an explicit `--root` value that selects the project root for that Relay process.
- **FR-020**: The system MUST treat an explicit `--root` value as higher priority than the current working directory.
- **FR-021**: The system MUST reject startup when the requested explicit root is invalid, inaccessible, or not an existing readable directory that Relay can bind as the active repository and workspace scope.
- **FR-022**: The system MUST prevent a project switch that would result in multiple simultaneous active runs across different projects.
- **FR-023**: The system MUST explain why a project switch is blocked when the current active project still has a non-terminal run.
- **FR-024**: The system MUST show a human-readable failure state when a known project root cannot be activated.
- **FR-025**: The system MUST canonicalize project-root identity so the same root is not stored more than once under path aliases or equivalent launch locations.
- **FR-026**: The system MUST keep project-scoped runs, canvas state, and repository state isolated even when the developer alternates rapidly between known project roots.
- **FR-027**: The system MUST treat project archiving and deletion as out of scope for this feature.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST route project selection, persisted-context lookup, context creation, history filtering, and project-scoped hydration through existing handler, orchestrator, tool, and storage boundaries.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped file-system and shell boundaries for the active project root.
- **CDR-003**: The feature MUST include automated coverage for automatic project-context creation, repeated reconnect to the same root, project switching, blocked switching during an active run, history scoping, all-project history mode, and CLI root selection behavior.
- **CDR-004**: The feature MUST define visible loading states, human-readable error states, and explicit empty states for initial project hydration, project switching, known-project selection, and project-scoped history surfaces.
- **CDR-005**: The feature MUST preserve existing handler-level approval enforcement so changing the active project root never weakens file-write or shell-command approval requirements.
- **CDR-006**: The feature MUST preserve repository sandboxing by ensuring agent file access and command execution remain bound to the currently active project root and cannot escape into other known roots.
- **CDR-007**: The feature MUST document any new third-party dependency introduced for project switching, path identity handling, or project-context persistence and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Known Project Root**: A project location Relay recognizes as a distinct workspace scope, including its canonical root identity and visibility in the project switcher.
- **Project Context Record**: The persisted context associated with one project root, including its scoped workspace state and historical runs.
- **Active Project Context**: The currently selected project root plus the canvas, history, file-tree, and other workspace surfaces that must reflect that root.
- **Project Switcher Entry**: The user-visible representation of a known project root in the switcher, including whether it is active and whether it can currently be selected.
- **History Scope Mode**: The current History tab mode, either active-project only or developer-enabled all-project aggregation.

## Assumptions

- Relay will support only one active project context at a time in a running instance.
- A project's persisted context is keyed by the canonical form of its root path rather than by a user-entered label.
- A supported project root for this feature is any existing readable directory that Relay can bind as the active repository and workspace scope; Git metadata and project-manifest detection are not required.
- Switching projects restores the latest saved context for that project rather than replaying every historical run automatically.
- The all-project history view is a review surface only and does not change which project root is active for canvas, file tree, approvals, or future runs.
- The active project root may be displayed as a path, a recognizable project label, or both, as long as the root identity remains unambiguous to the developer.
- This feature assumes a fresh local Relay database for rollout; preserving older pre-feature local databases is out of scope.

## Dependencies

- Existing workspace persistence must be extensible to scope saved context by project root when Relay initializes a fresh database with the multi-project schema.
- Existing canvas, run history, and repository tree surfaces must support rehydration from a selected project scope rather than from a single global scope.
- Existing launch and configuration handling must support selecting a project root at startup.
- Existing approval, repository-bounding, and command-sandbox rules must remain enforceable per active project root.

## Out of Scope

- Running multiple simultaneous active runs across different project roots
- Manual creation of project-scoped saved context by the developer
- Archiving, deleting, or merging project-scoped saved contexts
- Combining multiple project file trees or canvases into a single live workspace view
- Automatic switching of the active project based only on browsing history across all projects

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of first-time project-root connection tests, Relay creates usable project-scoped persisted context without requiring any manual setup step.
- **SC-002**: In at least 95% of validation attempts, switching from one known project root to another updates the active project indicator, canvas, run history, and file tree within 2 seconds.
- **SC-003**: In 100% of validation scenarios, the default History tab view shows only runs for the active project until the developer explicitly enables the all-project view.
- **SC-004**: In 100% of validation scenarios, enabling the all-project history view preserves clear project identity for each listed run and does not change the active project root.
- **SC-005**: In 100% of validation attempts, starting Relay without `--root` uses the current working directory as the active project root, and starting Relay with `--root` uses the requested root instead.

### Validation Notes

- SC-002 is satisfied by recording at least 20 manual or automated switch attempts during validation and confirming at least 19 complete within 2 seconds from project-switch request to the target project's header, history, canvas, and repository surfaces rendering.
