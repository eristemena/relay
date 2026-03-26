# Feature Specification: Live Agent Panel

**Feature Branch**: `004-live-agent-panel`  
**Created**: 2026-03-23  
**Status**: Draft  
**Input**: User description: "A live AI agent panel inside Relay where a developer types a task, a specialized AI agent picks it up, and the developer watches the agent's reasoning appear word by word in real time."

## Historical Note

The dedicated Live execution drawer panel described in this phase was later retired as Relay moved to canvas-first inspection. Task submission still lives in the command bar, while live and replayed execution details now live on canvas node detail surfaces, saved runs, and approval review instead of a separate workspace-menu panel.

## Clarifications

### Session 2026-03-23

- Q: Should project root selection be part of this feature, and if so how is it configured? → A: Agents may use repository-reading tools only when the developer manually sets the project root in Relay's local configuration; choosing that folder in the UI is deferred to a later version.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Submit a task and watch it live (Priority: P1)

As a developer, I can type a natural-language task into a command bar and immediately watch one specialized agent stream its visible reasoning and output word by word so I can follow what it is doing while it works.

**Why this priority**: This is the core product promise. If the developer cannot submit one task and watch a live stream from a clearly identified agent, the feature does not deliver its primary value.

**Independent Test**: Can be fully tested by launching Relay, submitting one task, and confirming that exactly one agent starts, shows its role and model identity, streams output continuously with a live cursor, and ends in a completed or errored state.

**Acceptance Scenarios**:

1. **Given** the live agent panel is idle, **When** the developer submits a task, **Then** Relay starts exactly one agent run for that task and switches the panel into an active state.
2. **Given** a run has started, **When** the agent begins producing visible output, **Then** the thought viewer fills incrementally in chronological order and shows a live cursor while streaming continues.
3. **Given** a run is active, **When** the developer looks at the panel header, **Then** they can identify which specialized agent is handling the task and which model is running that agent.
4. **Given** a run finishes, **When** the final visible output is produced, **Then** the panel shows a terminal state and becomes ready for the next task.

---

### User Story 2 - Understand tool use and execution state (Priority: P2)

As a developer, I can see state changes and tool activity inline in the same stream so I can understand how the agent reached its result instead of receiving a delayed answer with no context.

**Why this priority**: Transparency depends on more than streamed text. Developers need to see when the agent is thinking, using a tool, or failing so they can trust the run and diagnose what happened.

**Independent Test**: Can be fully tested by running a task that triggers at least one tool call and confirming that state changes plus tool entries appear inline in the correct order, with enough detail to understand the action and without exposing secrets.

**Acceptance Scenarios**:

1. **Given** an active agent run, **When** the agent changes between idle, thinking, executing a tool, done, or errored, **Then** the panel updates the state indicator with a distinct visual for each state.
2. **Given** the agent invokes a tool, **When** the tool call begins and completes, **Then** the stream shows the tool name, the relevant input context, and the resulting outcome inline at the correct point in the timeline.
3. **Given** a tool input or result contains sensitive values, **When** that event is shown, **Then** Relay redacts or suppresses the sensitive content while preserving enough context for the developer to understand what happened.
4. **Given** a tool call or provider step fails, **When** the run cannot continue, **Then** the panel shows an errored state and preserves the visible partial stream for later review.

---

### User Story 3 - Set access once and revisit prior runs (Priority: P3)

As a developer, I can save my OpenRouter API key and reopen prior runs later so the panel is practical for repeated use instead of acting like a disposable one-off interaction.

**Why this priority**: Saved access and replayable history make the feature usable across sessions, but they are secondary to the live execution experience itself.

**Independent Test**: Can be fully tested by saving an API key, completing one or more runs, restarting Relay, and confirming that the saved access remains available and previous runs can still be reopened and reviewed.

**Acceptance Scenarios**:

1. **Given** the developer opens settings, **When** they enter a valid OpenRouter API key and save it, **Then** Relay stores it for future runs and confirms that the credential is available without exposing the full secret.
2. **Given** the developer has completed at least one run, **When** they return later in the same or a later Relay session, **Then** they can reopen the saved run and review the streamed content, tool activity, selected agent, and final state.
3. **Given** no saved runs exist yet, **When** the developer opens run history, **Then** Relay shows an explicit empty state that explains how saved runs will appear after the first completed task.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature must preserve Relay's required backend flow: handlers accept the developer action, the orchestrator drives the selected agent, agents invoke tools through approved interfaces, and storage persists saved state. No layer may be skipped.
- Frontend changes must remain in feature-based areas such as the workspace shell, streaming panel, history, and preferences features rather than introducing root-level type-based folders.
- This feature requires WebSocket protocol additions for live token streaming, inline tool activity, run state changes, approval-related execution states when applicable, and saved-run hydration. Those protocol changes require integration coverage because WebSocket is the only allowed backend-to-frontend channel.

### Approval and Safety Impact

- This feature introduces visible tool execution but does not expand approval scope beyond existing Relay rules. Any tool that writes files or executes shell commands must still require handler-level developer approval before execution.
- Agent specialization, streaming, and tool visibility must not weaken repository sandboxing, path traversal protection, or secret-handling rules. The UI may show tool activity, but it must not reveal protected credentials or unrestricted file-system data.
- Repository-reading tools are allowed only when the developer has manually configured a valid local project root. If that root is missing or invalid, the feature must block repository-reading tool activity with a plain-language remediation message.

### UX States

- The live agent panel must show visible states for idle, submitting, waiting for first output, streaming output, executing a tool, completed, and errored.
- Settings must show visible saving and saved states, plus plain-language error feedback when access cannot be stored or used.
- Run history must show explicit empty, loading, and recoverable error states. The thought viewer must also show an explicit empty state before the first task is submitted.

### Edge Cases

- If the agent starts but does not emit visible output immediately, Relay must still show a visible in-progress state so the developer knows the task was accepted.
- If the first visible output misses the responsiveness target, Relay must still preserve the run and show that the agent is working rather than leaving the viewer blank.
- If the OpenRouter API key is missing, invalid, or revoked, Relay must prevent the run from proceeding and explain how the developer can fix access.
- If the local project root is missing, unreadable, or outside the intended repository boundary, Relay must prevent repository-reading tool activity and explain that the developer must correct the local configuration manually.
- If a tool call fails mid-run, Relay must show the failed tool event inline and preserve the partial stream for later review.
- If Relay restarts after a completed run, the saved run must remain reviewable even if the original task can no longer be re-executed.
- If the system receives a second run request while one run is already active, Relay must reject or defer the additional request so the single-agent scope remains intact for this version.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a focused command bar where the developer can enter a natural-language task and submit it to the live agent panel.
- **FR-002**: The system MUST start exactly one agent run for each accepted task submission in this feature scope.
- **FR-003**: The system MUST choose one supported specialized agent role for each submitted task rather than presenting the run as a generic assistant response.
- **FR-004**: The system MUST define each supported agent role with a distinct built-in persona, behavior constraints, and output style so that the Planner, Coder, Reviewer, Tester, and Explainer behave differently from one another.
- **FR-005**: The system MUST stream the active agent's visible reasoning and output incrementally in chronological order for the duration of the run.
- **FR-006**: The system MUST display a live cursor or equivalent active-stream indicator while output is still being generated.
- **FR-007**: The system MUST display the active agent role and model badge for every run in a persistent identity area.
- **FR-008**: The system MUST show a distinct visible state indicator for idle, thinking, executing a tool, done, and errored states.
- **FR-009**: The system MUST show tool activity inline in the thought viewer, including the tool name, relevant input context, and resulting outcome in the correct sequence relative to the stream.
- **FR-010**: The system MUST redact, suppress, or otherwise protect secrets and sensitive values from any tool call content or run transcript shown in the UI.
- **FR-011**: The system MUST allow the developer to save one OpenRouter API key in settings for use by all supported models in this feature.
- **FR-012**: The system MUST preserve the saved OpenRouter API key across Relay restarts until the developer replaces or removes it.
- **FR-013**: The system MUST prevent a new run from starting when required access configuration is missing or invalid and MUST present a plain-language explanation of how to fix it.
- **FR-014**: The system MUST persist each completed or errored agent run so the developer can review it later.
- **FR-015**: The system MUST preserve enough information for each saved run to review the original task, streamed content, tool activity, selected agent role, model badge, timestamps, and final state.
- **FR-016**: The system MUST provide a way to access previously saved runs from within Relay.
- **FR-017**: The system MUST preserve partial streamed content and inline tool activity when a run ends in error so the developer can review what happened.
- **FR-018**: The system MUST support only one active agent run at a time in this feature version.
- **FR-019**: The system MUST provide a human-readable empty state for the thought viewer before the first task is submitted and for run history before the first run is saved.
- **FR-020**: The system MUST allow the developer to understand from the live panel whether the system is waiting for output, streaming output, executing a tool, completed, or blocked by an error without needing to inspect logs or refresh the UI.
- **FR-021**: The system MUST allow repository-reading tools only when the developer has manually configured a valid local project root for Relay.
- **FR-022**: The system MUST block repository-reading tool activity when the local project root is missing, invalid, unreadable, or outside the approved workspace boundary and MUST present a plain-language remediation message.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST NOT bypass handler-level approval enforcement for any file writes or shell commands triggered by agents or tools.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped file-system access boundaries for any tool activity exposed by this feature.
- **CDR-003**: The feature MUST include tests covering live stream delivery, run persistence, settings persistence, inline tool activity rendering, WebSocket protocol changes introduced for agent run events, and required tool-path behavior where tool access changes are introduced.
- **CDR-004**: The feature MUST define and implement visible loading states, human-readable error messages, and explicit empty states for command submission, live streaming, settings, and saved-run review.
- **CDR-005**: The feature MUST ensure that secrets such as API keys and tokens are never shown in logs, never exposed in clear text to the frontend, and never revealed unredacted in streamed tool activity or saved run history.
- **CDR-006**: The feature MUST document any new third-party dependency and update the Tech Stack note in project documentation in the same change.
- **CDR-007**: The feature MUST maintain responsive user interaction during active streaming so that viewing or navigating the UI does not freeze while live output is arriving.

### Key Entities *(include if feature involves data)*

- **Agent Run**: A single execution record for one submitted developer task, including the original task, selected agent role, model identity, lifecycle timestamps, final state, and reviewable stream content.
- **Stream Event**: A time-ordered unit of visible run activity, such as streamed text, a state transition, or a tool call or result entry.
- **Tool Activity Record**: A reviewable representation of one tool invocation during a run, including the tool name, protected input summary, protected result summary, timing, and outcome.
- **Agent Role Profile**: The product-defined description of a specialized role, including its purpose, constraints, and expected output style.
- **Project Root Setting**: The locally managed setting that defines the allowed repository boundary for repository-reading tools in this feature.
- **Credential Status**: The developer-facing record that indicates whether an OpenRouter API key is configured and usable without exposing the full secret.
- **Run History**: The saved collection of prior agent runs available for later review within Relay.

## Assumptions

- Agent selection is automatic for this phase; the developer submits a task and Relay chooses the most appropriate supported role rather than asking the developer to select from multiple agents.
- The developer sets the local project root manually in Relay's local configuration; this phase does not provide a folder picker or other UI to edit that path.
- The thought viewer shows only the agent's visible streamed output and execution events intended for the developer, not hidden internal reasoning that should remain private to the model provider.
- Tool visibility is limited to information that is safe to reveal in the operator UI; protected values may be redacted while preserving operational context.
- The saved-run review experience is read-only in this phase. Re-running, branching, editing, or exporting prior runs is outside scope unless covered by another feature.

## Dependencies

- The developer must have a valid OpenRouter account and usable API key for live task execution.
- The developer must set a valid local project root before repository-reading tools can succeed.
- Relay must be able to reach the configured model provider during live runs, even though saved-run review remains available without re-executing the task.
- Local persistence must remain available on the developer's machine so saved runs and credential state can survive Relay restarts.

## Out of Scope

- Multiple agents running simultaneously
- A visual agent canvas or graph view
- File writing or code modification as a user-visible feature goal
- UI for selecting or editing the local project root directory
- Developer-facing controls to author or edit built-in system prompts
- Collaborative or shared run history across users or machines

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In at least 95% of accepted task submissions during local validation, the first visible streamed output appears within 500ms of the run entering an active state.
- **SC-002**: In at least 95% of manual validation runs, developers can correctly identify the current agent state from the panel without consulting logs or developer tools.
- **SC-003**: 100% of completed and errored runs created during manual acceptance testing remain available for review after Relay is restarted on the same machine.
- **SC-004**: 100% of tool calls triggered during acceptance testing appear in chronological order within the corresponding saved run, with protected values redacted where required.
- **SC-005**: At least 90% of developers in initial usability review can submit a task, recognize which specialized agent handled it, and explain what happened during the run by using only the live panel.
