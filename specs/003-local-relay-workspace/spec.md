# Feature Specification: Local Relay Workspace

**Feature Branch**: `003-local-relay-workspace`  
**Created**: 2026-03-23  
**Status**: Draft  
**Input**: User description: "A local developer tool called Relay that opens a visual AI coding workspace in the browser when you run a single command."

## Clarifications

### Session 2026-03-23

- Q: Should Relay treat frontend port 3000 and backend port 4747 as fixed ports? → A: No. Both are preferred defaults only; if either port is unavailable, Relay must discover a free port and continue using the assigned port for that run.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Launch the workspace from one command (Priority: P1)

As a developer, I can run a single serve command and immediately land in a browser-based Relay workspace so I can begin a coding session without setup friction.

**Why this priority**: The product promise is a single-command local workspace. If this flow fails, the rest of the experience is unreachable.

**Independent Test**: Can be fully tested by running the serve command on a machine with no prior session history and confirming that the workspace opens automatically and is ready for use within the stated startup target.

**Acceptance Scenarios**:

1. **Given** Relay is available on the developer's machine, **When** the developer runs `relay serve`, **Then** Relay starts locally and opens the workspace in the default browser automatically.
2. **Given** Relay starts successfully, **When** the browser opens, **Then** the developer sees the top navigation bar, session sidebar, and central workspace canvas in a usable default state.
3. **Given** the preferred local port is unavailable, **When** the developer runs `relay serve`, **Then** Relay automatically starts on a free local port and clearly reports the actual address without opening a broken workspace.

---

### User Story 2 - Resume past sessions (Priority: P2)

As a developer, I can see my past coding sessions in the sidebar and reopen one so I can continue work without losing context between runs.

**Why this priority**: Persistent session history is the core differentiator from disposable chat windows and is required for returning to previous work.

**Independent Test**: Can be fully tested by creating at least one session, restarting Relay, and confirming that prior sessions remain listed and reopen with their saved metadata and workspace context.

**Acceptance Scenarios**:

1. **Given** one or more saved sessions exist, **When** the workspace loads, **Then** the sidebar lists those sessions in a recognizable and selectable format.
2. **Given** the developer selects a prior session from the sidebar, **When** the session opens, **Then** the workspace restores that session as the active view.
3. **Given** no prior sessions exist, **When** the workspace loads, **Then** the sidebar shows an explicit empty state with a clear path to start a new session.

---

### User Story 3 - Start a new session and keep preferences (Priority: P3)

As a developer, I can start a new session and keep my local Relay preferences across restarts so the workspace behaves consistently every time I use it.

**Why this priority**: New-session creation and persistent preferences make the tool practical for daily use, but they are still secondary to basic launch and session recovery.

**Independent Test**: Can be fully tested by creating a new session, changing supported preferences, restarting Relay, and confirming that the session is listed and the preferences remain applied.

**Acceptance Scenarios**:

1. **Given** the workspace is open, **When** the developer chooses to start a new session, **Then** Relay creates a new session and makes it the active workspace.
2. **Given** the developer changes a supported preference such as the preferred local port or stored API credentials, **When** Relay is restarted, **Then** the updated preference remains in effect unless Relay must temporarily use a different free port for that run.
3. **Given** a stored preference is invalid or unreadable at startup, **When** Relay launches, **Then** it falls back safely, preserves unaffected preferences, and presents a clear error message describing what needs attention.

## Constitution Alignment *(mandatory)*

### Architecture and Boundaries

- This feature establishes the initial local workspace shell and session lifecycle without introducing any layer bypasses. Startup, session listing, session creation, and preference persistence must continue to flow through the required handler -> orchestrator -> agent -> tool -> storage boundary where applicable, with no direct frontend-to-storage coupling.
- Frontend work must be organized in feature-based areas for workspace shell concerns such as navigation, session history, and canvas presentation rather than root-level type buckets.
- WebSocket remains the only backend-to-frontend communication channel. This feature may introduce initial workspace and session-state events required to render the shell, and any such protocol changes require integration coverage.

### Approval and Safety Impact

- This feature does not introduce agent-driven file writes or shell execution. Any local persistence needed for sessions or preferences must not weaken the existing requirement that future file writes or commands require handler-level developer approval.
- No changes in this feature may expand agent access to the file system or shell. Repo-root sandboxing and path traversal protections remain unchanged because repository access is explicitly out of scope for this phase.

### UX States

- The workspace must show a visible startup/loading state while the local service initializes, a visible loading state while session history is being retrieved, and an in-context saving state when preferences are being stored.
- Errors must be human-readable, including cases such as failure to allocate a usable local port, unreadable saved configuration, browser launch failure, and unavailable session data.
- Empty states must be explicit for a first-run workspace with no sessions and for a newly created session with no activity yet on the canvas.

### Edge Cases

- If automatic browser launch is blocked by the operating system, Relay must still start the local workspace and present the local address clearly so the developer can open it manually.
- If Relay is restarted while a prior session is incomplete, that session must still appear in history and open without being treated as corrupted solely because it was unfinished.
- If the preferred port is unavailable at launch, Relay must automatically try a free local port and only fail with a clear error if no usable port can be allocated.
- If saved preferences contain unsupported appearance values, Relay must ignore only the unsupported value and continue using the supported dark-mode workspace presentation.
- If local session storage cannot be read temporarily, the workspace must explain that saved sessions are unavailable rather than showing an unexplained blank sidebar.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a `relay serve` command that starts the Relay workspace locally from a single user action.
- **FR-002**: The system MUST automatically open the Relay workspace in the developer's default browser after a successful local startup.
- **FR-003**: The system MUST render a workspace layout that includes a persistent left sidebar for session navigation, a top navigation bar, and a central canvas area for session activity.
- **FR-004**: The system MUST persist coding sessions locally so they remain available after Relay stops and restarts.
- **FR-005**: The system MUST list saved sessions in the sidebar whenever session history exists.
- **FR-006**: The system MUST allow the developer to select a saved session from the sidebar and reopen it as the active workspace.
- **FR-007**: The system MUST allow the developer to start a new session from within the workspace.
- **FR-008**: The system MUST persist developer preferences across restarts, including the preferred local port, stored API credentials, and supported workspace appearance preferences.
- **FR-009**: The system MUST remain usable without internet connectivity for all local UI, session history, and preference management flows defined in this feature.
- **FR-010**: The system MUST store all data required for this feature locally on the developer's machine and MUST NOT require authentication, cloud sync, or remote services for core operation.
- **FR-011**: The system MUST present clear recovery guidance when local startup cannot secure a usable port or automatic browser launch cannot be completed.
- **FR-012**: The system MUST preserve the active session context for the current run until the developer switches sessions, starts a new session, or closes Relay.
- **FR-013**: The system MUST be deliverable as a single downloadable binary that does not require an installation workflow beyond obtaining the file and running it.
- **FR-014**: The system MUST treat the configured Relay port as a preferred default and automatically use a free local port for the current run when that preferred port is unavailable.

### Constitution-Derived Requirements *(mandatory)*

- **CDR-001**: The feature MUST preserve Relay's layered architecture and MUST NOT bypass handler-level approval enforcement for any future file writes or shell commands.
- **CDR-002**: The feature MUST preserve WebSocket-only backend/frontend communication, SQLite-only persistence, and repo-scoped file-system access boundaries for future agent capabilities.
- **CDR-003**: The feature MUST include tests covering local startup behavior, session persistence behavior, preference persistence behavior, and any WebSocket protocol changes introduced for workspace shell rendering.
- **CDR-004**: The feature MUST define and implement visible loading states, human-readable error messages, and explicit empty states for workspace startup, session history, and new-session flows.
- **CDR-005**: The feature MUST document any new third-party dependency and update the Tech Stack note in project documentation in the same change.

### Key Entities *(include if feature involves data)*

- **Session**: A saved local coding workspace record representing one developer work thread, including an identifier, display name, creation time, last-opened time, and the minimal state needed to reopen it.
- **Preference Set**: A local collection of developer-configurable Relay settings, including preferred local port selection, stored API credentials, and supported workspace presentation choices.
- **Workspace State**: The runtime view state for the currently open session, including which session is active and whether the workspace is loading, empty, or showing recoverable errors.

## Assumptions

- Preference persistence refers only to settings supported by this product phase. Because Relay is dark-mode only by constitution, any appearance preference in this feature is limited to supported dark-mode presentation options rather than introducing light mode.
- If Relay must fall back from the preferred port to a free port, that fallback applies only to the current run unless the developer explicitly saves a new preferred port.
- Session persistence in this phase stores only local session metadata and workspace state needed to reopen a session. AI transcript content, repository linkage, and tool execution history are outside scope.
- Automatic browser opening is expected on typical local developer environments, but the product must still be usable when the operating system blocks automatic launch.

## Out of Scope

- AI or agent execution behavior
- Authentication, multi-user accounts, or access control
- Cloud sync, remote access, or collaboration features
- Repository connection, file system browsing, or code execution features

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: On a machine that meets Relay's published local requirements, the workspace is ready in the browser within 2 seconds of the developer running the serve command in at least 95% of startup attempts.
- **SC-002**: 100% of sessions created during manual acceptance testing remain available in the session sidebar after Relay is stopped and restarted on the same machine.
- **SC-003**: 100% of supported developer preferences changed during manual acceptance testing remain applied after a restart on the same machine.
- **SC-004**: In offline manual acceptance testing, developers can complete the launch, view-session-history, open-session, and start-new-session flows without internet access.
- **SC-005**: At least 90% of developers in initial usability review can complete the first-run flow of launching Relay and starting a new session without external instructions.
