# Feature Specification: Repository File Tree Right-Rail Panel

**Feature Branch**: `011-repo-file-tree`  
**Created**: 2026-03-29  
**Status**: Draft  
**Input**: User description: "A browsable file tree sidebar for Relay that shows the connected repository's directory structure and highlights which files agents have touched during the current run. The sidebar renders the full recursive directory listing from the connected repo root, collapsible by folder. Any file that an agent has read or proposed a change to during the run is marked with a subtle visual indicator — updated in real time as agents work. Clicking any agent node on the canvas narrows the sidebar to show only the files that specific agent touched, giving the developer a clear answer to \"what did this agent actually look at?\". The tree is read-only — it shows state, it does not allow the developer to open or edit files from it. File tree for repos with no connected project root is out of scope."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Browse the connected repository structure (Priority: P1)

As a developer, I can see the connected repository as a collapsible right-rail panel so I can keep codebase structure visible beside the graph while a run is in progress.

**Why this priority**: The repository tree itself is the foundation of the feature. Without a browsable, repo-rooted tree, the later touched-file indicators and per-agent filtering have no useful surface.

**Independent Test**: Can be fully tested by connecting Relay to a repository, starting a run and confirming the right rail shows File Tree directly, then reopening a saved run and confirming the same right rail exposes top-level tabs for Historical Replay and File Tree.

**Acceptance Scenarios**:

1. **Given** Relay has an active connected repository and a live run, **When** the right rail is visible, **Then** the developer sees the repository root and its recursive directory structure in the File Tree panel beside the graph.
2. **Given** the tree contains nested folders, **When** the developer expands or collapses a folder, **Then** the tree updates that folder's visible children without changing run state or opening files.
3. **Given** the developer clicks a file or folder row in the tree, **When** the row receives focus or selection, **Then** Relay preserves the tree as read-only and does not open an editor, mutate the repository, or trigger navigation away from the current workspace surface.
4. **Given** the repository tree is visible during an active run, **When** live agent events continue arriving, **Then** the dock remains readable and does not block the canvas or workspace controls.

---

### User Story 2 - See which files the run has touched (Priority: P2)

As a developer, I can tell which files agents have read or proposed to change during the current run so I can understand the run's actual repository footprint without leaving the graph view.

**Why this priority**: Trust depends on visibility into agent behavior. After the tree exists, the next highest value is showing touched files live as evidence of what the run has inspected or attempted to modify.

**Independent Test**: Can be fully tested by running an agent workflow against a connected repository and confirming the tree updates in real time to mark files as touched when agents read them or propose changes to them.

**Acceptance Scenarios**:

1. **Given** the repository tree is visible for an active run, **When** any agent reads a file inside the connected repository, **Then** the matching file row gains a visible touched indicator without requiring a page refresh.
2. **Given** the repository tree is visible for an active run, **When** any agent proposes a change to a file, **Then** the matching file row is marked as touched for the current run even before the proposal is approved or rejected.
3. **Given** multiple agents touch the same file during one run, **When** the developer views the unfiltered tree, **Then** that file appears once with a touched state that reflects participation in the current run rather than duplicate rows.
4. **Given** no files have been read or proposed for change yet in the current run, **When** the developer views the tree, **Then** Relay shows the full directory structure with no touched markers rather than implying unseen activity.

---

### User Story 3 - Filter the tree by selected agent activity (Priority: P3)

As a developer, I can click an agent node and narrow the right-rail File Tree to just the files that agent touched so I can answer what that specific agent actually looked at.

**Why this priority**: Per-agent narrowing is the most targeted trust and debugging workflow, but it depends on both the tree and touched-file tracking already being in place.

**Independent Test**: Can be fully tested by running a multi-agent workflow, clicking one canvas node, and confirming the sidebar narrows to only that agent's touched files while preserving a clear way to return to the full tree.

**Acceptance Scenarios**:

1. **Given** at least one agent in the current run has touched repository files, **When** the developer clicks that agent's node on the canvas, **Then** the sidebar narrows to show only the files touched by that selected agent.
2. **Given** the developer is viewing an agent-filtered tree, **When** the selected agent touches another file during the same run, **Then** the filtered view updates in real time to include the newly touched file.
3. **Given** the developer selects an agent that has not touched any repository files, **When** the sidebar applies that agent filter, **Then** Relay shows an explicit empty state explaining that the selected agent has not touched any files in the current run.
4. **Given** the developer has narrowed the tree to one agent, **When** the developer clears the selection or returns to the workspace-wide view, **Then** the sidebar restores the full repository tree with current run touch markers intact.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature extends existing repository-awareness flow through the established handler -> orchestrator -> agent -> tool -> storage boundaries by surfacing already-governed repository structure and agent file-activity state in the right-side workspace detail rail rather than creating a new direct repository access path from the frontend.
- Frontend changes remain inside existing feature-based areas such as workspace shell, canvas, and repository-awareness surfaces. The feature adds a read-only sidebar view and selection-driven filtering behavior, not a new editor or type-based top-level folder structure.
- WebSocket remains the only backend/frontend runtime channel. Protocol coverage is required for full-tree payloads or tree-hydration updates, current-run touched-file updates, and canvas-selection-driven per-agent narrowing state so the sidebar can stay synchronized with live agent work.

### Approval and Safety Impact

- This feature does not introduce new file writes, shell commands, or approval bypasses. The sidebar is strictly read-only and must not allow developers or agents to open, edit, approve, or execute anything from the tree surface.
- The feature relies on existing repository-scoped file-read and proposed-change activity already governed by Relay. Repo-root sandboxing, path traversal protection, and approval enforcement for proposed writes remain unchanged.
- Touched indicators must reflect recorded reads and proposed changes only. The sidebar must not imply that a proposed file change was applied unless that state is already represented elsewhere by existing approval outcomes.

### UX States

- The right-side workspace detail rail must show visible states for loading the File Tree panel, rendering the ready tree, and refreshing touched indicators while a run is active.
- The feature must show human-readable errors when the repository tree cannot be loaded for a connected repository, when live activity state cannot be synchronized, or when an agent-specific filter cannot be resolved.
- The feature must show explicit empty states for an agent filter with no touched files and for a connected repository whose tree exists but contains no visible child entries.
- The feature must not expose a tree surface when no connected repository root exists because that case is out of scope for this feature.

### Edge Cases

- If the connected repository contains deep nesting or many siblings, the tree must remain collapsible and readable without flattening the hierarchy into an unreadable list.
- If the connected repository contains ignored paths such as `node_modules`, the tree must exclude paths ignored by the repository's `.gitignore` rules so expensive, low-value directories do not delay hydration.
- If two or more agents touch the same file during the same run, the workspace-wide view must keep one file entry while still allowing an agent-specific filter to include that file for each relevant agent.
- If an agent is selected before it has touched any files, the filtered sidebar must remain stable and show an explicit no-touched-files message rather than falling back silently to the full tree.
- If the active run ends while an agent filter is applied, the sidebar must preserve the final touched-file state for that run until the developer starts another run or changes context.
- If the WebSocket connection reconnects while the tree is filtered or partially expanded, Relay must restore the visible tree state without duplicating rows or losing the current touched markers.
- If a touched file no longer exists in the current repository snapshot because the repository changed outside Relay during the run, the sidebar must preserve the best available historical marker or explain the missing entry instead of crashing or inventing a new path.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST render a File Tree panel in the right-side workspace detail rail when Relay has a connected repository root and an active run is visible.
- **FR-001a**: The system MUST expose top-level right-rail tabs for Historical Replay and File Tree when the developer reopens a saved run.
- **FR-002**: The system MUST derive the tree from the connected repository root and include the full recursive directory structure available within that repository boundary.
- **FR-002**: The system MUST derive the tree from the connected repository root and include the full recursive directory structure available within that repository boundary, excluding paths ignored by the repository's `.gitignore` rules.
- **FR-003**: The system MUST organize the tree hierarchically with folders and files and MUST allow folders to be expanded and collapsed.
- **FR-004**: The system MUST preserve the tree as read-only and MUST NOT allow opening files, editing files, approving changes, or invoking repository actions from the sidebar.
- **FR-005**: The system MUST show a touched-file indicator for any file that an agent read during the current run.
- **FR-006**: The system MUST show a touched-file indicator for any file that an agent proposed to change during the current run.
- **FR-007**: The system MUST update touched-file indicators in real time as read and proposed-change activity occurs during the current run.
- **FR-008**: The system MUST keep one canonical entry per repository path in the workspace-wide tree even when multiple agents touch the same file.
- **FR-009**: The system MUST allow the developer to click an agent node on the canvas to narrow the File Tree panel to only the files touched by that specific agent during the current run.
- **FR-010**: The system MUST update the agent-filtered tree in real time when the selected agent touches additional files.
- **FR-011**: The system MUST provide a clear way to return from an agent-filtered view to the full repository tree.
- **FR-012**: The system MUST preserve current-run touched markers when the developer exits an agent-filtered view and returns to the workspace-wide tree.
- **FR-013**: The system MUST show an explicit empty state when the selected agent has not touched any files in the current run.
- **FR-014**: The system MUST keep the File Tree panel visible and readable beside the graph during live orchestration activity without blocking canvas interaction.
- **FR-015**: The system MUST preserve folder expansion and collapse state while live touched-file updates arrive for the current run.
- **FR-015**: The system MUST preserve folder expansion and collapse state while live touched-file updates arrive for the current run, with only top-level entries and one nested level visible before the developer expands deeper folders.
- **FR-016**: The system MUST preserve or restore visible tree state after reconnect so expanded folders, selected-agent filtering, and touched markers do not reset unexpectedly during the same run.
- **FR-017**: The system MUST ensure that agent-specific filtering includes files the agent read and files the agent proposed to change.
- **FR-018**: The system MUST ensure that the workspace-wide view reflects touched files across all agents participating in the current run.
- **FR-019**: The system MUST keep the tree bounded to the connected repository and reject repository-escape paths.
- **FR-020**: The system MUST treat the absence of a connected repository root as out of scope and MUST NOT render a substitute file tree for that condition in this feature.
- **FR-021**: The system MUST show human-readable failure feedback when the repository tree or touched-file state cannot be loaded.
- **FR-022**: The system MUST NOT imply that a touched file was modified on disk unless an approved change outcome is already represented elsewhere in Relay.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST route repository tree loading and touched-file activity through existing handler, orchestrator, tool, and storage boundaries rather than introducing direct frontend repository access.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-backed persistence where historical or resumable state is needed, and repo-scoped file-system access for agents.
- **CDR-003**: The feature MUST include automated coverage for repository tree loading, recursive hierarchy rendering, live touched-file indicator updates, per-agent tree narrowing, reconnect recovery, and read-only interaction constraints.
- **CDR-004**: The feature MUST define visible loading states, human-readable error states, and explicit empty states for repository tree hydration, agent-filtered no-results cases, and live activity synchronization failures.
- **CDR-005**: The feature MUST preserve existing approval enforcement and MUST NOT add any sidebar action that can write files, run commands, or bypass developer review.
- **CDR-006**: The feature MUST document any new dependency introduced for tree rendering or virtualization and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Repository Tree Snapshot**: The hierarchical representation of folders and files rooted at the connected repository that is displayed in the sidebar.
- **Tree Node Entry**: One folder or file row in the sidebar, including its repository-relative path, hierarchy position, expansion state if it is a folder, and current visual status.
- **Touched File Marker**: The current-run indicator attached to a file entry when one or more agents have read the file or proposed a change to it.
- **Agent File Touch Set**: The run-scoped set of repository file paths associated with one agent's reads and proposed file changes.
- **Sidebar Filter Context**: The current right-rail File Tree mode, including workspace-wide view or a selected-agent view, plus any preserved expansion state needed to keep the rail stable.

## Assumptions

- Relay already tracks per-agent file reads and proposed file changes as part of existing repository-awareness and approval flows, and this feature reuses that activity instead of inventing a separate source of truth.
- The connected repository tree can be derived from repository-scoped listing capabilities already permitted within Relay's safety boundaries.
- The current run is the only scope for touched indicators in this feature; prior runs are not merged into the live sidebar state.
- Clicking an agent node already establishes or can establish a stable selected-agent context that other workspace surfaces can react to.
- The sidebar may choose a condensed visual treatment for touched indicators as long as the difference between untouched and touched files remains perceivable without relying on color alone.

## Dependencies

- Existing connected-repository support must remain the authoritative source for repository-root validation and repo-bounded file listing.
- Existing agent activity tracking for file reads and proposed changes must remain available to drive touched-file indicators.
- Existing canvas node selection state must be available to the workspace shell so the sidebar can narrow to the selected agent.
- WebSocket delivery must remain available for live synchronization of run activity and any reconnect recovery needed to restore tree state.

## Out of Scope

- Showing a file tree when no connected repository root exists
- Opening files, previewing file contents, or navigating to an editor from the sidebar
- Editing files, approving diffs, rejecting diffs, or executing commands from the sidebar
- Multi-repository browsing or comparing multiple repositories side by side
- Merging touched-file activity from previous runs into the current run sidebar
- Replacing existing approval review surfaces with the file tree sidebar

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In at least 95% of validation runs on supported repositories, the repository tree becomes visible within 2 seconds of loading the connected workspace.
- **SC-002**: In 100% of validated agent file-read and proposed-change events, the corresponding file path is reflected in the sidebar's touched state for the current run without requiring a manual refresh.
- **SC-003**: In at least 95% of multi-agent validation runs, developers can identify the files touched by a selected agent within 5 seconds of clicking that agent node.
- **SC-004**: In 100% of validation checks, interacting with the repository tree never opens files, edits files, or triggers command execution.
- **SC-005**: In at least 95% of reconnect or live-update validation attempts, the sidebar preserves or restores the expected filtered view, touched markers, and folder expansion state without duplicate rows.
