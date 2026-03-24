# Feature Specification: Live Agent Orchestration

**Feature Branch**: `006-live-agent-orchestration`  
**Created**: 2026-03-24  
**Status**: Draft  
**Input**: User description: "A live multi-agent orchestration layer for Relay that connects the previously built React Flow canvas to a real Go backend — the developer types a goal and watches a team of specialized agents work through it in parallel, their activity streaming live to the canvas. The Planner breaks the goal down, the Coder and Tester run concurrently once the Planner completes, the Reviewer fires when both finish, and the Explainer closes the run with a plain-English summary. Nodes appear on the canvas as agents spawn, states update as they work, and clicking any node opens its live or completed output in the side panel. An individual agent can error without killing the whole run, but a run-level error halts everything with a clear message. Codebase access, file writes, and animations are out of scope — agents work from the prompt only, and the canvas must remain interactive throughout."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Watch an orchestration run unfold live (Priority: P1)

As a developer, I can submit one goal and watch Relay spawn the required specialist agents on the canvas in the expected order so I can understand how the work is being divided and where the run currently stands.

**Why this priority**: The core value is turning the existing static canvas into a real orchestration surface. If the developer cannot start a run and see the agent team appear and progress live, the feature does not deliver its main outcome.

**Independent Test**: Can be fully tested by submitting one goal and confirming that the Planner starts first, Coder and Tester begin only after Planner completes, Reviewer starts only after both concurrent agents finish, and Explainer closes the run with a final summary while all node and state changes appear live on the canvas.

**Acceptance Scenarios**:

1. **Given** the canvas is idle, **When** the developer submits a goal, **Then** Relay starts one orchestration run and creates a Planner node as the first active agent on the canvas.
2. **Given** the Planner has completed successfully, **When** downstream work becomes eligible, **Then** Relay starts both the Coder and Tester agents without requiring the developer to refresh or resubmit the run.
3. **Given** the Coder and Tester have both finished, **When** the run advances to review, **Then** Relay starts the Reviewer and later the Explainer in sequence, ending with a completed run summary.
4. **Given** the run is active, **When** agent states change, **Then** the corresponding canvas nodes update live to reflect their current progress.

---

### User Story 2 - Inspect live and completed agent output without losing context (Priority: P2)

As a developer, I can click any agent node during or after the run to inspect that agent's current or finished output in the side panel so I can follow the work without losing my place on the canvas.

**Why this priority**: Live orchestration is only understandable if the developer can inspect the details behind each node. Node-level visibility is the main interaction model that makes the canvas more than a status diagram.

**Independent Test**: Can be fully tested by opening a run, clicking the active Planner node while it is still producing output, then clicking completed nodes later in the same run and confirming the side panel shows the correct live or finished transcript for the selected node while the canvas remains interactive.

**Acceptance Scenarios**:

1. **Given** an orchestration run is in progress, **When** the developer clicks an active node, **Then** the side panel opens the selected agent's currently available output and continues updating while that agent is still active.
2. **Given** one agent has already completed, **When** the developer clicks that completed node, **Then** the side panel shows that agent's preserved output and final state rather than switching to another agent's content.
3. **Given** a side panel is open for one agent, **When** the developer clicks a different node, **Then** the side panel changes to the newly selected agent without interrupting the underlying run.
4. **Given** the orchestration run is active, **When** the developer pans, zooms, or changes node selection, **Then** the canvas remains responsive throughout the interaction.

---

### User Story 3 - Understand partial failures and run-level halts clearly (Priority: P3)

As a developer, I can distinguish between an individual agent failure and a run-level failure so I understand whether Relay preserved the rest of the run or stopped everything.

**Why this priority**: Multi-agent systems become confusing quickly when failures are ambiguous. Clear failure handling is necessary to maintain trust and to explain why later agents did or did not run.

**Independent Test**: Can be fully tested by observing one run where a single agent fails but the overall run continues to the next allowed step, and another run where a run-level failure stops all remaining activity with a clear message and a stable terminal state.

**Acceptance Scenarios**:

1. **Given** one agent encounters an agent-scoped failure, **When** the failure is reported, **Then** that node moves to an error state, its output remains reviewable, and the rest of the run continues unless a later dependency makes further progress impossible.
2. **Given** the system encounters a run-level failure, **When** the failure is reported, **Then** Relay stops all remaining agent work, marks the run as halted, and presents a clear human-readable reason.
3. **Given** a run stops because of a run-level failure, **When** the developer inspects the canvas, **Then** all started nodes preserve their latest visible state and no additional nodes spawn after the halt.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature must preserve Relay's required backend flow: handlers accept run actions, the workspace orchestrator manages orchestration state, specialized agents execute prompt-only work inside the orchestrator's plan, and storage persists run plus agent event history. No layer may be skipped.
- This feature builds on the earlier live-run and static-canvas capabilities by introducing a distinct multi-agent orchestration mode rather than redefining the isolated canvas prototype or requiring the prior single-agent live panel mode to disappear.
- Frontend changes must remain inside feature-based areas such as the canvas, agent panel, history, and workspace shell features rather than introducing new root-level type-based folders.
- This feature requires WebSocket protocol additions or extensions for orchestration-run lifecycle events, per-agent spawn events, per-agent state transitions, per-agent streamed output, node selection hydration for completed runs, and run-level halt notifications. Those protocol changes require integration coverage because WebSocket is the only allowed backend-to-frontend channel.

### Approval and Safety Impact

- This feature is explicitly prompt-only for agent work. It does not expand agent access to repository files, file writes, or shell commands, so no new approval pathway is introduced and handler-level approval requirements remain unchanged.
- The orchestration layer must preserve existing sandbox expectations by ensuring agent execution in this feature cannot read the codebase, modify files, or trigger terminal commands, even if those abilities exist elsewhere in Relay.

### UX States

- The canvas and side panel must show visible states for idle, submitting, waiting for first agent, active orchestration, individual agent streaming, agent complete, agent error, run complete, and run halted.
- The interface must show plain-language error messaging for both agent-scoped failures and run-level failures.
- Goal submission must show a clear blocked state when the existing live-run model access configuration required by Relay is unavailable or unusable.
- The canvas must show an explicit empty state before the first orchestration run, and the side panel must show an explicit empty or instructional state when no node is selected.

### Edge Cases

- If the existing live-run model access configuration is missing or invalid, Relay must block orchestration start and explain how the developer can restore access before trying again.
- If the Planner fails before any downstream agent starts, Relay must keep the Planner output reviewable and mark the run as halted because the dependency chain cannot proceed.
- If either the Coder or Tester fails while the other is still running, Relay must preserve both nodes' live states independently and decide downstream eligibility based on the completed dependency rules rather than collapsing the entire canvas immediately.
- If the developer clicks between nodes rapidly while output is streaming, the side panel must always show the currently selected node and must not freeze or display mixed output from multiple agents.
- If the WebSocket connection drops during an active run and is later restored, the interface must recover enough orchestration state to avoid duplicating nodes or losing already streamed visible output.
- If a completed run is reopened later, Relay must restore the final node states and preserved node outputs even though no agent is actively streaming.
- If a run-level halt occurs while one or more agents are active, Relay must stop spawning any remaining agents and show a single clear reason for the halt without overwriting preserved agent-level output.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow a developer to submit one natural-language goal to start one orchestration run.
- **FR-002**: The system MUST create the Planner as the first agent in every orchestration run.
- **FR-003**: The system MUST start the Coder and Tester only after the Planner reaches a completed state.
- **FR-004**: The system MUST allow the Coder and Tester to be active concurrently once they are eligible to start.
- **FR-005**: The system MUST start the Reviewer only after both the Coder and Tester have reached terminal states required for review.
- **FR-006**: The system MUST start the Explainer only after the Reviewer reaches its terminal state.
- **FR-007**: The system MUST add a node to the canvas when an agent is spawned for the run.
- **FR-008**: The system MUST update each node's visible state live as that agent progresses through its lifecycle.
- **FR-009**: The system MUST stream each agent's visible output as it is produced and associate that output with the correct node.
- **FR-010**: The system MUST allow the developer to click any active or completed node to view that agent's live or preserved output in the side panel.
- **FR-011**: The system MUST keep the side panel synchronized with the currently selected node while preserving the developer's ability to change selections during an active run.
- **FR-012**: The system MUST preserve completed and errored agent output for later review after live streaming ends.
- **FR-013**: The system MUST preserve canvas interactivity, including pan, zoom, and node selection, while orchestration events continue arriving.
- **FR-014**: The system MUST reuse Relay's existing live-run access configuration and MUST block orchestration start with a plain-language remediation message when that required access is unavailable.
- **FR-015**: The system MUST preserve this feature as a prompt-only orchestration mode and MUST exclude tool-call activity from live node transcripts and preserved run records for runs created by this mode.
- **FR-016**: The system MUST allow an individual agent to enter an error state without automatically terminating the whole run.
- **FR-017**: The system MUST distinguish agent-scoped errors from run-level failures in both the run state and the user-visible messaging.
- **FR-018**: The system MUST halt all remaining orchestration activity when a run-level failure occurs.
- **FR-019**: The system MUST stop spawning downstream agents after a run-level failure has been recorded.
- **FR-020**: The system MUST preserve the latest visible output and final known state for every agent that has already started when the run reaches a terminal state.
- **FR-021**: The system MUST allow completed runs to be reopened with the same node identities, final node states, and preserved per-node output available for inspection.
- **FR-022**: The system MUST execute this feature's agent work from the submitted prompt only.
- **FR-023**: The system MUST NOT allow codebase reads, file writes, or shell command execution as part of this feature's orchestration flow.
- **FR-024**: The system MUST make the absence of codebase access and file-modifying actions consistent with the agent behavior shown in the UI and preserved run records.
- **FR-025**: The system MUST present an explicit idle or instructional state before the first run begins.
- **FR-026**: The system MUST present a clear non-selected state in the side panel when no node is selected.
- **FR-027**: The system MUST ensure per-agent streamed output remains ordered within that agent's transcript.
- **FR-028**: The system MUST ensure the canvas does not create duplicate nodes for the same agent within a single run, including after reconnect or replay.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST NOT bypass handler-level approval enforcement for any file writes or shell commands that exist elsewhere in the product.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped safety boundaries while keeping this specific feature prompt-only.
- **CDR-003**: The feature MUST include tests covering orchestration ordering, concurrent Coder and Tester execution, per-agent stream delivery, node selection and side-panel hydration, run replay into the canvas, agent-scoped error handling, run-level halt handling, and reconnect behavior for active runs.
- **CDR-004**: The feature MUST define and implement visible loading states, human-readable error messages, and explicit empty states for goal submission, canvas rendering, node inspection, and completed-run replay.
- **CDR-005**: The feature MUST ensure the UI and persisted history do not imply or expose file-system access, shell execution, or code-modifying side effects for these agents.
- **CDR-006**: The feature MUST reuse the existing live-run access and history surfaces unless a later feature explicitly replaces them with a unified alternative.
- **CDR-007**: The feature MUST document any new dependency introduced for orchestration presentation or state handling and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Orchestration Run**: One submitted developer goal and the full multi-agent execution it triggers, including lifecycle state, dependency progress, halt status, and final outcome.
- **Agent Session**: The run-scoped record for one specialized agent, including role, spawn order, current state, preserved output, timestamps, and terminal result.
- **Agent Transcript**: The ordered visible output associated with one agent session, whether still streaming or already complete.
- **Access Configuration**: The existing saved live-run model access state required for Relay to start backend-driven orchestration.
- **Run Dependency Stage**: The rule set that determines when downstream agents may begin based on the terminal state of earlier agents.
- **Run Halt Reason**: The human-readable explanation recorded when a run-level failure stops the orchestration.
- **Canvas Node Projection**: The frontend representation of one agent session, including stable node identity, role, state, and selection status.
- **Selected Node View**: The side-panel representation of the currently selected canvas node and its live or preserved transcript.

## Assumptions

- A single orchestration run is active at a time for this feature scope.
- This feature reuses the same saved provider-access setup and run history foundation already established for live backend-driven runs in Relay.
- Planner, Coder, Tester, Reviewer, and Explainer are built-in roles with fixed orchestration order rather than user-configurable workflows.
- Review is still meaningful when both prerequisite agents reach their required terminal states, even if one of them completed with an agent-scoped error that the Reviewer needs to assess.
- Completed-run replay is read-only in this phase; reopening a run restores visibility, not re-execution.
- The existing static canvas layout can be reused for live node positioning without introducing animation-heavy behavior in this phase.

## Dependencies

- Relay's existing canvas and side-panel surfaces must already support stable node rendering and selection behavior.
- Relay's existing live-run access configuration must already support authenticated backend model execution for orchestration runs.
- Relay's backend must be able to persist orchestration runs and agent event history locally so completed runs can be reopened later.
- WebSocket delivery must remain available for live updates during active runs, with replay support available when reopening saved runs.

## Out of Scope

- Agent access to the repository or local file system
- File writes, shell commands, or developer approval flows initiated by this orchestration feature
- User-authored orchestration graphs or custom agent ordering
- Animation-focused canvas polish beyond preserving normal interactivity and readable state changes
- Code generation that edits the developer's workspace
- Multi-user shared orchestration sessions

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In at least 95% of accepted runs during local validation, the Planner node appears on the canvas within 1 second of goal submission.
- **SC-002**: In 100% of validated orchestration runs, the observed agent start order matches the defined dependency flow of Planner, then Coder and Tester in parallel, then Reviewer, then Explainer.
- **SC-003**: In at least 95% of active-run validation attempts, developers can switch node selection while output is streaming without losing canvas responsiveness.
- **SC-004**: In 100% of validated agent-scoped failure cases, the failed node remains inspectable and the UI clearly distinguishes that condition from a run-level halt.
- **SC-005**: In 100% of validated run replay attempts, reopening a completed run restores the same per-node final states and preserved node transcripts that were visible at live completion.
