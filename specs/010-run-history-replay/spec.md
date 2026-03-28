# Feature Specification: Run History and Replay

**Feature Branch**: `010-run-history-replay`  
**Created**: 2026-03-28  
**Status**: Draft  
**Input**: User description: "A complete run history and replay system for Relay so developers can revisit, review, and share any past agent run. A toolbar-triggered run history panel lists all past runs with an auto-generated title, date, agent count, and status; selecting one plays it back on the canvas exactly as it happened — nodes spawning, states changing, thought streams appearing — with a scrubber to seek and a speed control (0.5x, 1x, 2x, 5x). For runs that modified files, the developer can view a full before/after diff of every change made during that run. Any run can be exported as a structured markdown report and searched by keyword, file touched, or date range. Replay is driven entirely from stored events — no agents are re-invoked during playback. Cloud sync, link sharing, and video export are out of scope."

## Clarifications

### Session 2026-03-28

- Q: Should the saved-run history surface be specified as a sidebar? → A: No. It is a toolbar-triggered run history panel in the current UI.
- Q: How does markdown export satisfy file-write approval requirements? → A: Export is an explicit developer action. The handler may write immediately only for direct user-initiated export requests, and server-side enforcement must prevent agent-triggered export without a full approval flow.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Replay any past run from history (Priority: P1)

As a developer, I can browse saved runs in the toolbar-triggered run history panel and replay one on the canvas from stored events so I can understand exactly how that run unfolded without invoking agents again.

**Why this priority**: Exact replay is the core value of the feature. If a past run cannot be reopened and played back faithfully from persisted events, the feature does not achieve its main purpose.

**Independent Test**: Can be fully tested by opening a saved run from the run history panel and confirming that the canvas reconstructs the recorded timeline in order, including node creation, state changes, streamed thought text, and terminal status, without creating any new agent activity.

**Acceptance Scenarios**:

1. **Given** saved runs exist for the current workspace session history, **When** the developer opens the run history panel from the workspace toolbar, **Then** each run entry shows an auto-generated title, its recorded date, agent count, and terminal status.
2. **Given** a saved run is listed in history, **When** the developer selects it, **Then** Relay replays the stored event stream into the canvas in recorded order rather than starting a fresh backend run.
3. **Given** a replay is in progress, **When** the recorded event sequence reaches an agent spawn, state change, or transcript event, **Then** the canvas reflects that exact step and associates it with the correct node.
4. **Given** a saved run ended in a completed, halted, errored, or clarification-required state, **When** replay reaches the final stored event, **Then** the canvas ends in the same recorded final state and does not append any new live events.

---

### User Story 2 - Inspect, search, and review recorded run artifacts (Priority: P2)

As a developer, I can search historical runs and inspect the files they changed so I can find relevant past work and review what happened without leaving Relay.

**Why this priority**: Once replay exists, the next highest value is efficient retrieval and inspection. Searchability and change review make run history practical for debugging, auditing, and handoff.

**Independent Test**: Can be fully tested by filtering history with a keyword, a touched file, and a date range, then opening a matching run and verifying that any recorded file modifications expose a full before/after diff for every changed file associated with that run.

**Acceptance Scenarios**:

1. **Given** multiple saved runs exist, **When** the developer searches by keyword, **Then** Relay narrows the history list to runs whose recorded metadata, task text, summaries, or replayable transcript content match the query.
2. **Given** multiple saved runs exist, **When** the developer filters by touched file or date range, **Then** Relay returns only runs whose stored history satisfies all active filters.
3. **Given** a selected run contains recorded file modifications, **When** the developer opens its file-change review surface, **Then** Relay shows the full before and after content for each changed file preserved by that run.
4. **Given** a selected run contains no recorded file modifications, **When** the developer views its change review surface, **Then** Relay shows an explicit empty state instead of a blank panel.

---

### User Story 3 - Control playback and export a reusable report (Priority: P3)

As a developer, I can scrub through replay, change playback speed, and export the run as structured markdown so I can review it at my own pace and share a durable summary outside the live UI.

**Why this priority**: Playback controls and export deepen the value of replay, but they depend on the core history and inspection flows already existing.

**Independent Test**: Can be fully tested by starting a replay, changing the speed to each supported option, seeking to a specific point in the timeline, and exporting the selected run to markdown with the preserved run metadata, timeline, and file-change summary.

**Acceptance Scenarios**:

1. **Given** a saved run is open for replay, **When** the developer selects 0.5x, 1x, 2x, or 5x playback, **Then** the replay timing adjusts to the chosen speed while preserving recorded event order.
2. **Given** a replayable run is open, **When** the developer drags the scrubber to a point in the timeline, **Then** the canvas and supporting panels update to the state produced by all stored events up to that point.
3. **Given** a run is selected, **When** the developer exports it, **Then** Relay generates a structured markdown report containing the run title, date, participants, outcome, timeline summary, and recorded file-change review data when available.
4. **Given** a replay is paused at an intermediate point, **When** the developer exports the run, **Then** the export still represents the full stored run record rather than the partially scrubbed viewport state.
5. **Given** the developer clicks `Export Report`, **When** the export request reaches the server, **Then** the handler treats that request as the explicit approval for the markdown file write and rejects any equivalent export attempt that did not originate from a direct user action.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature must preserve Relay's existing handler -> orchestrator -> agent -> tool -> storage flow by extending the run-history and replay path through handlers, the workspace orchestrator, and SQLite-backed run-event storage rather than introducing direct frontend data access or bypassing stored events.
- Frontend changes must stay within existing feature-based areas such as history, canvas, workspace shell, approvals, and shared workspace state rather than introducing new type-based top-level folders.
- Replay remains WebSocket-driven. Any new history search, replay-control, export, or diff-review interactions must continue using the established backend/frontend protocol surfaces and require integration coverage for event playback, replay seeking hydration, and export initiation.

### Approval and Safety Impact

- This feature does not add any new file writes or shell command execution by agents during replay. Replaying a saved run is strictly read-only and must never re-request developer approval for already-recorded actions.
- Markdown export is the one file-writing path introduced by this feature. It is allowed without a secondary approval dialog only when the request is a direct developer action such as clicking `Export Report`, and that distinction must be enforced server-side at the handler boundary rather than inferred from UI state alone.
- Historical file diffs must be rendered only from previously stored, display-safe event or approval data. The feature must not reopen repository files from disk during replay as a substitute for missing historical content.
- Existing handler-level approval enforcement, repository sandboxing, and path traversal protections remain unchanged for live runs that produce history. Replay only reads already persisted history within those governed boundaries.

### UX States

- The run history panel must show visible states for loading saved runs, displaying filtered results, showing no matching runs, and reporting history-load failure in plain language.
- Replay controls must show clear states for idle-with-no-run-selected, preparing replay, actively replaying, paused, scrubbed-to-position, replay-complete, and replay-failed.
- The diff review surface must show loading, ready, and explicit no-file-changes states.
- Markdown export must show visible progress and a human-readable failure state if Relay cannot generate the report.

### Edge Cases

- If a stored run contains duplicate or out-of-order sequence numbers, Relay must rebuild replay state deterministically from the persisted order rules and must not show duplicated timeline steps.
- If a replayed run references agent nodes or file changes that are incomplete because they predate this feature, Relay must preserve whatever historical data exists and show that missing details are unavailable rather than inferring new data.
- If the developer changes playback speed or scrubber position repeatedly, Relay must keep the canvas, transcript, and diff surfaces synchronized to the latest selected replay position.
- If a search query and filters return no runs, Relay must show an explicit no-results state while preserving the current filter context.
- If a run contains approval and tool events for file modifications, Relay must show the final recorded before/after content for each changed file without implying that replay can reapply or revert those changes.
- If the WebSocket connection reconnects while the developer is browsing history or replaying a saved run, Relay must restore the selected run and replay state without creating duplicate history entries or replay events.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST persist enough run metadata for every saved run to list it in history with an auto-generated title, recorded date, agent count, and terminal status.
- **FR-002**: The system MUST display saved runs in a dedicated run history panel opened from the workspace toolbar that allows a developer to select one run at a time for review.
- **FR-003**: The system MUST generate run titles from stored run context without requiring manual naming by the developer.
- **FR-004**: The system MUST replay a selected run entirely from stored events and MUST NOT invoke any agent, tool, or live orchestration work as part of playback.
- **FR-005**: The system MUST reconstruct replay state in recorded event order, including node creation, node state transitions, transcript updates, approvals, tool activity summaries, and run terminal events when present in history.
- **FR-006**: The system MUST preserve stable node identity across replay so the canvas, detail panel, and related summaries stay attached to the correct recorded agent.
- **FR-007**: The system MUST provide playback controls for play, pause, seek, and replay completion reset for a selected historical run.
- **FR-008**: The system MUST provide playback speed options of 0.5x, 1x, 2x, and 5x for historical replay.
- **FR-009**: The system MUST update the canvas, transcript views, and related run surfaces to match the replay position selected by the scrubber.
- **FR-010**: The system MUST allow the developer to pause replay and inspect the currently reconstructed canvas and supporting panels without losing the selected run.
- **FR-011**: The system MUST allow a developer to search saved runs by keyword.
- **FR-012**: The system MUST allow a developer to filter saved runs by file touched.
- **FR-013**: The system MUST allow a developer to filter saved runs by date range.
- **FR-014**: The system MUST combine keyword search, file-touched filtering, and date-range filtering so the visible history list reflects all active criteria together.
- **FR-015**: The system MUST keep the full unfiltered run history intact while search or filters are active so clearing filters restores the broader list without data loss.
- **FR-016**: The system MUST expose a read-only diff review surface for any saved run that recorded file modifications.
- **FR-017**: The system MUST show the full preserved before and after content for every file changed during a selected run whenever that historical data exists.
- **FR-018**: The system MUST group recorded file changes by run and by file path so a developer can review each modified file independently.
- **FR-019**: The system MUST show an explicit empty state when a selected run has no recorded file modifications.
- **FR-020**: The system MUST generate a structured markdown export for any selected saved run.
- **FR-021**: The exported markdown MUST include the run title, recorded date, final status, participating agents, a chronological summary of the run, and a file-change section when recorded file modifications exist.
- **FR-022**: The system MUST ensure markdown export is derived from stored run metadata and stored events rather than from transient UI state.
- **FR-022A**: The system MUST treat a direct developer export action as the explicit approval for the markdown file write and MUST reject export requests that are not initiated by a direct user action.
- **FR-023**: The system MUST preserve replay fidelity for completed, errored, halted, and clarification-required runs.
- **FR-024**: The system MUST label replayed content clearly enough that developers can distinguish historical playback from an active live run.
- **FR-025**: The system MUST preserve existing live-run behavior so opening a saved run for replay does not mutate or restart the original recorded run.
- **FR-026**: The system MUST retain replay-safe ordering guarantees so stored event streams can be reopened after restart without inventing missing steps.
- **FR-027**: The system MUST expose human-readable history and replay errors when saved runs cannot be loaded, replayed, searched, filtered, diffed, or exported.
- **FR-028**: The system MUST preserve already-recorded approval and tool-result context in replay and export when those events are part of the stored run history.
- **FR-029**: The system MUST keep search and replay responsive while replay is active so the developer can move between saved runs without reloading the entire workspace.
- **FR-030**: The system MUST treat cloud sync, link sharing, and video export as out of scope for this feature.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST route all replay, search, export, and diff-history behavior through existing handler, orchestrator, and storage boundaries rather than adding direct frontend database access.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped safety boundaries for the live runs that create historical records.
- **CDR-003**: The feature MUST include automated coverage for restart-safe replay ordering, history list metadata, keyword and filter behavior, replay scrubber and speed controls, file-diff review, markdown export, and reconnect behavior while a saved run is open.
- **CDR-004**: The feature MUST define visible loading states, human-readable error states, and explicit empty states for history, replay, diff review, and export flows.
- **CDR-005**: The feature MUST reuse stored event data for replay and MUST NOT re-run historical agents, re-trigger tool calls, or prompt for developer approval during playback.
- **CDR-005A**: The feature MUST enforce server-side separation between direct developer export actions and agent-triggered file writes so markdown export can bypass a secondary approval dialog only for explicit user-initiated requests.
- **CDR-006**: The feature MUST ensure recorded file-diff data shown in replay and export remains display-safe and does not require reading current repository files from disk to reconstruct historical changes.
- **CDR-007**: The feature MUST document any new dependency introduced for diff presentation, filtering, or export generation and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Saved Run Summary**: The listable history record for one completed or terminal run, including generated title, recorded date, agent count, status, searchable text, and change indicators.
- **Stored Run Event**: One persisted, ordered event from a historical run that can be replayed to rebuild canvas state, transcript state, tool activity, approvals, and terminal outcome.
- **Replay Timeline**: The ordered playback model derived from stored run events, including duration, current cursor position, playback speed, and replay state.
- **Replay Snapshot**: The reconstructed canvas and supporting panel state that results from applying stored events up to a specific replay position.
- **Run Change Record**: The preserved before/after file-change data associated with one historical run and one file path.
- **Run Search Filter**: The active combination of keyword text, touched-file criterion, and date-range bounds used to narrow visible history entries.
- **Run Export Document**: The structured markdown representation of one historical run generated from stored metadata and stored events.

## Assumptions

- Relay already persists ordered run events and enough run metadata to reopen saved runs after restart, and this feature extends that foundation rather than replacing it.
- Auto-generated run titles can be derived from existing stored run context such as the original goal, summary, or task preview without requiring a new manual title field.
- Historical file-change review is available only for runs whose stored approval or tool history already captured before/after content at the time of execution.
- Replay fidelity is defined by recorded event order and stored payloads, not by wall-clock-perfect animation timing.
- Markdown export is intended as a portable text artifact and does not include live embedded canvas media.
- Search and date filtering operate on locally stored run history for the current Relay workspace rather than on a cloud-synced corpus.

## Dependencies

- Existing SQLite-backed run history and event replay infrastructure must continue to persist ordered event streams across restarts.
- Existing canvas and workspace-store replay logic must support deterministic rebuilding of node, transcript, approval, and tool activity state from stored events.
- Existing run history UI surfaces must be available to extend with richer metadata, filters, and export actions.
- Historical file-change data recorded by approval and tool events must remain available in a display-safe form for runs that modified files.

## Out of Scope

- Re-invoking agents, tools, or approvals during historical playback
- Cloud sync of run history across machines
- Shareable links to hosted replay sessions
- Video or GIF export of replay playback
- Editing or reapplying historical file changes from the replay UI
- Comparing two separate runs side by side in this phase

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In at least 95% of local validation attempts, opening a saved run begins reconstructing the historical canvas within 1 second of selection.
- **SC-002**: In 100% of validated replay scenarios, the replayed final canvas state matches the stored terminal state of the selected run without creating new agent activity.
- **SC-003**: In at least 95% of validation attempts, developers can find a target run using keyword, file-touched, or date-range filters in under 10 seconds.
- **SC-004**: In 100% of validated runs that include recorded file modifications, the review surface exposes each changed file with preserved before and after content.
- **SC-005**: In 100% of validated markdown exports, the generated document contains the selected run's title, date, outcome, participating agents, and any available file-change summary.
