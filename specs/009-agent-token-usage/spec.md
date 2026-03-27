# Feature Specification: Agent Token Usage

**Feature Branch**: `009-agent-token-usage`  
**Created**: 2026-03-27  
**Status**: Draft  
**Input**: User description: "Extend Relay's event protocol so that every agent run captures token usage — the complete event gains two new fields, tokens_used and context_limit, sourced from the OpenRouter response usage object and the model's known context window size respectively. Both fields are persisted to SQLite alongside the rest of the event log so that historical runs retain their data. Once stored, each agent node on the canvas displays a token usage fill bar showing how far into its context window the model has consumed — the fill color shifts from neutral to amber to red as the limit approaches, updating live during a run. This applies to live runs and to any prior run whose events were emitted after this change. Estimated client-side token counts, multi-run aggregate views, and per-token cost breakdowns are out of scope for this phase."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Monitor live token usage per agent (Priority: P1)

As a developer watching an active Relay run, I can see how much of each agent's context window has been consumed so I can spot risky runs before they approach model limits.

**Why this priority**: Live visibility is the primary value of the feature. If the current run does not show trustworthy token usage progression, the rest of the feature does not materially help operators.

**Independent Test**: Can be fully tested by running an agent workflow that emits completion events with token usage data and confirming the active canvas updates each affected agent node with a usage bar, percentage-based fill, and threshold color transitions.

**Acceptance Scenarios**:

1. **Given** an agent completes a model call during an active run and the completion event includes `tokens_used` and `context_limit`, **When** Relay streams the event to the canvas, **Then** the corresponding agent node updates its usage bar without requiring a page refresh.
2. **Given** an agent's token usage remains well below its context window, **When** the usage bar renders, **Then** it uses the neutral state rather than a warning state.
3. **Given** an agent's token usage reaches the warning threshold near its context limit, **When** the usage bar updates, **Then** the bar switches to amber.
4. **Given** an agent's token usage reaches the critical threshold near its context limit, **When** the usage bar updates, **Then** the bar switches to red.

---

### User Story 2 - Revisit token usage in prior runs (Priority: P2)

As a developer reviewing a previous Relay run, I can see the stored token usage state for each agent event that was recorded after this feature shipped so I can compare past run behavior without rerunning the workflow.

**Why this priority**: Historical replay preserves the operational value of the metric beyond the live session and makes the event log materially more useful.

**Independent Test**: Can be fully tested by replaying a stored run whose completion events were created after the protocol change and confirming the replayed canvas shows the same token usage bars and thresholds that were available during the original run.

**Acceptance Scenarios**:

1. **Given** a prior run contains persisted completion events with token usage fields, **When** the developer opens that run, **Then** Relay rehydrates and displays the token usage bar for each matching agent node.
2. **Given** a prior run contains older events that predate the new fields, **When** the developer replays the run, **Then** Relay leaves the token usage visualization empty for those events and does not fabricate estimated values.
3. **Given** a run mixes older events and newer events, **When** the replay is shown, **Then** only the events with stored token usage data contribute to the displayed usage state.

---

### User Story 3 - Trust incomplete or unavailable usage states (Priority: P3)

As a developer, I can distinguish between real token usage values and unavailable data so I do not mistake missing telemetry for low usage.

**Why this priority**: Clear fallback behavior is necessary for operator trust, but it is secondary to the core live and historical visibility.

**Independent Test**: Can be fully tested by replaying or streaming completion events where token usage or context limit values are unavailable and confirming Relay keeps the node operable while showing a plain, non-misleading fallback state.

**Acceptance Scenarios**:

1. **Given** a completion event is received without token usage data, **When** the canvas renders that agent node, **Then** the token usage bar remains absent or explicitly unavailable instead of showing a guessed fill level.
2. **Given** a completion event includes token usage data but no known context limit for that model, **When** the node updates, **Then** Relay preserves the reported token count while withholding percentage-based risk coloring that depends on the limit.
3. **Given** token usage data exceeds the recorded context limit because the upstream values are inconsistent, **When** the usage bar is shown, **Then** Relay caps the visible fill at the full bar and still marks the state as critical.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- The feature preserves Relay's required handler -> orchestrator -> agent -> tool -> storage flow by capturing usage data at the agent/provider boundary, carrying it through orchestrator event generation, and persisting it via the existing event log storage layer.
- Frontend changes remain inside feature-based areas responsible for the live agent panel, canvas nodes, and run replay surfaces rather than introducing cross-cutting UI buckets.
- The WebSocket event protocol changes by extending the agent completion event payload with `tokens_used` and `context_limit`. Historical replay payloads must expose the same fields when available, and the protocol change requires integration coverage for both live streaming and replay.

### Approval and Safety Impact

- This feature does not add file writes, shell commands, or new approval flows. Existing handler-level developer approval guarantees remain unchanged.
- Agent file-system behavior, shell behavior, repository sandboxing, and path traversal protections do not change as part of this feature.

### UX States

- Live agent nodes must show a visible token usage state change as new completion events arrive, and replayed runs must show the same state once the relevant event history is loaded.
- User-visible fallback messaging must distinguish between available usage data, unavailable usage data, and unavailable context-limit data in plain language.
- Explicit empty states are required when a run or event has no token usage data because it predates this feature or because the upstream provider did not return the required fields.

### Edge Cases

- If the upstream completion response omits usage information entirely, Relay must preserve the completion event without inventing token counts.
- If Relay knows the token count but not the model's context window size, the UI must show the raw token count without a misleading percentage fill.
- If the stored context limit is zero, negative, or otherwise invalid, Relay must treat the percentage fill as unavailable.
- If an older stored run predates the new fields, replay must continue to work without migration-time failures or client errors.
- If an agent produces multiple completion events within one run, the node must reflect the most recent applicable token usage state for that agent during live viewing and replay.
- If the token count is greater than the stored context limit, Relay must show a full critical bar rather than overflow the UI.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST extend the agent completion event payload to include `tokens_used` when upstream token usage is available for that completion.
- **FR-002**: The system MUST extend the agent completion event payload to include `context_limit` when Relay knows the context window size for the model that produced the completion.
- **FR-003**: The system MUST source `tokens_used` from the upstream OpenRouter usage data returned for the completion event rather than from a client-side estimate.
- **FR-004**: The system MUST source `context_limit` from Relay's known context window size for the specific model used by the agent completion.
- **FR-005**: The system MUST persist `tokens_used` and `context_limit` with the stored event log record so that future replays can reuse the same values.
- **FR-006**: The system MUST continue to persist and replay completion events that do not include the new fields.
- **FR-007**: The system MUST expose persisted `tokens_used` and `context_limit` values when replaying historical runs whose events were stored after this protocol change.
- **FR-008**: The system MUST update the relevant agent node on the live canvas when a completion event with token usage data arrives.
- **FR-009**: The system MUST visualize token usage as a fill bar that represents the share of `tokens_used` relative to `context_limit` whenever both values are available and valid.
- **FR-010**: The system MUST use a neutral visual state for token usage that remains below the warning threshold.
- **FR-011**: The system MUST use an amber visual state when token usage reaches the configured warning threshold near the context limit.
- **FR-012**: The system MUST use a red visual state when token usage reaches the configured critical threshold near the context limit.
- **FR-013**: The system MUST update the token usage visualization during an active run without requiring the developer to reload the workspace.
- **FR-014**: The system MUST show the same token usage visualization during run replay when the stored event data includes the new fields.
- **FR-015**: The system MUST NOT generate estimated token counts for events that lack upstream usage data.
- **FR-016**: The system MUST preserve a distinct unavailable state when token usage data is missing, when context-limit data is missing, or when the stored values are invalid.
- **FR-017**: The system MUST cap the visible fill at the full bar when reported token usage exceeds the stored context limit.
- **FR-018**: The system MUST scope the visualization to per-agent, per-run state and MUST NOT introduce aggregate multi-run token views in this feature.
- **FR-019**: The system MUST NOT introduce per-token cost breakdowns in this feature.
- **FR-020**: The system MUST NOT infer or store client-side estimated token counts in this feature.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST NOT bypass handler-level approval enforcement for file writes or shell commands.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped file-system access for agents.
- **CDR-003**: The feature MUST include storage tests for persisted token usage fields, WebSocket integration tests for the extended completion event payload and replay flow, and React Flow node component or state-model tests for token usage visualization states.
- **CDR-004**: The feature MUST define visible live-update states, human-readable fallback/error states, and explicit empty states for missing historical token usage data.
- **CDR-005**: The feature MUST document any new third-party dependency and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Agent Completion Usage**: The per-completion token telemetry attached to an agent completion event, including the reported token count and the model's context limit when known.
- **Persisted Event Record**: The stored event-log entry that retains completion payload fields for historical replay, including token usage data when present.
- **Agent Token Usage State**: The most recent token-usage visualization state shown for an agent node within a specific run, including raw count, limit, fill percentage, and risk band when derivable.

## Assumptions

- OpenRouter remains the source of provider-reported token usage for the affected agent completions in this phase.
- Relay already maintains or can deterministically resolve a model-to-context-window mapping for models it supports in this flow.
- Historical runs created before this feature will remain replayable without backfilling missing token usage data.
- Token usage visualization is attached to individual agent nodes and reflects the most recent applicable completion event for that node in the current run view.

## Dependencies

- Existing completion events must already identify the agent and run context needed to attach token usage updates to the correct node.
- Event-log persistence and replay must remain the source of truth for historical event visualization.
- Canvas node rendering must support an additional live-updating state indicator without regressing interaction responsiveness.

## Out of Scope

- Estimating tokens on the client when provider usage data is unavailable
- Multi-run aggregate usage dashboards or summaries
- Per-token or per-run cost calculations and billing breakdowns
- Retroactively backfilling token usage into events that were stored before this feature existed
- Changing approval, repository, or non-token event workflows outside the completion event extension needed for this feature

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of validation checks for supported completions, live completion events that include provider usage data expose `tokens_used` and the matching context-limit value to the active canvas within the normal Relay streaming flow.
- **SC-002**: In 100% of validation checks for stored runs created after this feature ships, replayed completion events preserve the same `tokens_used` and `context_limit` values that were originally stored.
- **SC-003**: In at least 95% of representative live and replay validation runs, developers can identify whether an agent is in neutral, warning, or critical token-usage state without leaving the canvas.
- **SC-004**: In 100% of validation checks, runs that lack token usage data show an explicit unavailable or empty state and never display fabricated token estimates.
- **SC-005**: During validation of active streaming runs, token usage visualization updates do not block normal canvas interaction such as inspecting nodes, panning, or zooming.
