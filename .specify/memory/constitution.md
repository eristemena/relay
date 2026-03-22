<!--
Sync Impact Report
Version change: template -> 1.0.0
Modified principles:
- Placeholder principle 1 -> I. Code Quality Is a Product Feature
- Placeholder principle 2 -> II. Tests Guard the Core Runtime
- Placeholder principle 3 -> III. Layered Architecture Is Mandatory
- Placeholder principle 4 -> IV. Operator-Safe UX and Approvals
- Placeholder principle 5 -> V. Secure, Bounded, Real-Time Execution
Added sections:
- Technical Boundaries
- Delivery Workflow and Quality Gates
Removed sections:
- None
Templates requiring updates:
- ✅ updated /Volumes/xpro/erisristemena/made-by-ai/relay/.specify/templates/plan-template.md
- ✅ updated /Volumes/xpro/erisristemena/made-by-ai/relay/.specify/templates/spec-template.md
- ✅ updated /Volumes/xpro/erisristemena/made-by-ai/relay/.specify/templates/tasks-template.md
Follow-up TODOs:
- None
-->

# Relay Constitution

## Core Principles

### I. Code Quality Is a Product Feature
Relay code MUST be idiomatic in its host language and optimized for long-term
maintenance. Go code MUST prefer the standard library when it provides an
equivalent capability, no function or method may exceed 40 lines without a
documented justification comment, and all exported Go functions, types, and
interfaces MUST have godoc comments. TypeScript MUST run in strict mode, no
`any` types are allowed, and every exported function MUST declare an explicit
return type. React code MUST use functional components only. `fmt.Println`,
`log.Print`, `console.log`, and `debugger` statements are forbidden in committed
code; structured logging is the only approved runtime logging approach. This is
non-negotiable because Relay is an orchestration product whose debugging burden
falls directly on maintainability and predictability.

### II. Tests Guard the Core Runtime
Tests are required deliverables, not optional follow-up work. Go tests MUST use
table-driven patterns by default. The `agent`, `orchestrator`, and `tools`
packages MUST maintain at least 75% coverage because they are Relay's core
runtime boundary. Every agent tool, including file and command tools, MUST have
a unit test covering the primary happy path and primary error path. Any change
to the WebSocket event protocol MUST include an integration test, and every
custom React Flow node component MUST include a component test. This principle
exists because orchestration defects are usually protocol or tool boundary
failures that unit coverage alone will not expose.

### III. Layered Architecture Is Mandatory
Relay backend code MUST follow this layer order only: HTTP/WebSocket handlers ->
orchestrator -> agents -> tools -> storage. Code MUST NOT skip layers and MUST
NOT introduce reverse imports across those boundaries. The `Agent` interface is
the only supported contract between the orchestrator and LLM providers; no code
outside the agent package may import a concrete agent implementation directly.
The frontend MUST use feature-based folders such as `features/canvas` and
`features/history`; root-level type-based buckets such as `components` or
`hooks` are not allowed as the primary organization model. WebSocket events are
the only supported communication channel between the Go backend and React
frontend, and SQLite is the only supported data store. These boundaries keep the
system replaceable, observable, and small enough to evolve without architectural
drift.

### IV. Operator-Safe UX and Approvals
Relay supports dark mode only. Every async operation MUST expose a visible
loading state, every user-visible error MUST be plain language rather than an
internal stack trace, and every empty list or canvas view MUST present an
explicit helpful empty state. Developer approval is required before any file
write or shell command executes, and that enforcement MUST happen at the handler
level on the server, not only in the UI. This principle exists because Relay is
an operator-facing tool where silent waits, blank states, and unenforced
approvals create avoidable risk.

### V. Secure, Bounded, Real-Time Execution
Every goroutine MUST have a cancellation path through `context.Context`; no
background work may run indefinitely without shutdown control. SQLite access MUST
avoid N+1 query patterns by batching multi-row reads. React Flow canvas updates
during active agent streaming MUST not block pan, zoom, or click interaction.
WebSocket messages MUST reach the frontend within 100ms of the backend event
occurring. Secrets such as API keys and tokens MUST be stored only in
`~/.relay/config.toml`, MUST never be hardcoded, logged, or sent to the
frontend, and LLM prompt or response content MUST never appear in application
logs. All agent file system access MUST stay inside the connected repository and
path traversal attempts MUST be rejected in the tool layer. Shell commands MUST
run in a sandboxed subprocess with the working directory locked to the repo
root. Relay must feel responsive while maintaining strict execution boundaries.

## Technical Boundaries

- Backend implementations MUST remain Go-first and comply with the layered
	architecture defined above.
- Frontend implementations MUST remain React and TypeScript based, with strict
	typing and feature-based organization.
- No secondary database, cache, queue, or polling channel may be introduced;
	SQLite and WebSocket events are the only approved persistence and runtime
	transport mechanisms.
- Any newly introduced third-party dependency MUST be justified in the relevant
	change, and the Tech Stack note in project documentation MUST be updated in the
	same change set.
- The `Agent` interface contract MUST NOT change unless every implementing type
	is updated in the same pull request.

## Delivery Workflow and Quality Gates

- Plans, specs, and tasks MUST explicitly describe how the change preserves the
	layer boundaries, approval enforcement, security restrictions, and required UX
	states.
- Reviews MUST reject changes that add direct concrete-agent imports outside the
	agent package, bypass handler-level approval enforcement, or introduce polling
	or alternate backend/frontend communication paths.
- Reviews MUST reject changes that lower coverage below the required threshold in
	the `agent`, `orchestrator`, or `tools` packages, or that modify the WebSocket
	protocol without integration coverage.
- Reviews MUST reject changes that expose secrets, log prompt or response
	content, allow repository escape for file access, or execute shell commands
	outside the sandboxed repo-root context.
- Performance-sensitive changes MUST describe how goroutine cancellation, SQLite
	query behavior, and streaming UI responsiveness were preserved or improved.

## Governance

This constitution supersedes conflicting local conventions for Relay. Amendments
MUST be made in `.specify/memory/constitution.md` and MUST include any required
updates to dependent templates or guidance files in the same change. Versioning
follows semantic versioning for the constitution itself: MAJOR for incompatible
principle changes or removals, MINOR for new principles or materially expanded
requirements, and PATCH for wording clarifications that do not change expected
behavior. Every plan, spec, task list, code review, and release readiness check
MUST include a constitution compliance review. Non-compliant work may proceed
only when the deviation is explicitly documented, justified, and approved before
implementation.

**Version**: 1.0.0 | **Ratified**: 2026-03-23 | **Last Amended**: 2026-03-23
