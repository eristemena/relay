# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement + technical approach from research]

## Technical Context

<!--
  ACTION REQUIRED: Replace the content in this section with the technical details
  for the project. The structure here is presented in advisory capacity to guide
  the iteration process.
-->

**Language/Version**: [e.g., Go 1.24 backend, TypeScript 5.x + React frontend or NEEDS CLARIFICATION]  
**Primary Dependencies**: [list standard library first; justify any third-party additions]  
**Storage**: [SQLite only, or NEEDS CLARIFICATION]  
**Testing**: [e.g., `go test`, Vitest, React Testing Library, integration coverage for WebSocket protocol changes]  
**Target Platform**: [e.g., macOS/Linux desktop environment for development, browser UI for frontend or NEEDS CLARIFICATION]
**Project Type**: [Relay backend/frontend feature, tooling, or platform work]  
**Performance Goals**: [e.g., WebSocket dispatch <100ms, non-blocking React Flow interaction during streaming]  
**Constraints**: [dark mode only, WebSocket-only frontend/backend communication, handler-level approval for file writes and shell commands]  
**Scale/Scope**: [domain-specific, e.g., 10k users, 1M LOC, 50 screens or NEEDS CLARIFICATION]

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [ ] Code quality rules are satisfied: idiomatic Go, standard library preferred,
  exported Go APIs documented, TypeScript strict mode preserved, no banned
  debug logging statements introduced.
- [ ] Test impact is defined: table-driven Go tests where applicable, core
  package coverage preserved at 75%+, tool happy/error-path coverage added,
  WebSocket protocol and React Flow node test obligations identified.
- [ ] Architecture remains compliant: handlers -> orchestrator -> agents ->
  tools -> storage, no concrete agent imports outside the agent package,
  feature-based frontend folders preserved, SQLite/WebSocket-only boundaries
  unchanged.
- [ ] UX and governance impact is defined: visible loading, human-readable
  errors, helpful empty states, and handler-level approval enforcement for
  file writes and shell commands.
- [ ] Security and performance constraints are covered: repo-scoped file access,
  sandboxed shell execution, no prompt/response logging, cancellable
  goroutines, no N+1 SQLite access, and <100ms event dispatch expectations.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
<!--
  ACTION REQUIRED: Replace the placeholder tree below with the concrete layout
  for this feature. Delete unused options and expand the chosen structure with
  real paths (e.g., apps/admin, packages/something). The delivered plan must
  not include Option labels.
-->

```text
backend/
├── cmd/
├── internal/
│   ├── handlers/
│   ├── orchestrator/
│   ├── agents/
│   ├── tools/
│   └── storage/
└── tests/
  ├── integration/
  └── unit/

frontend/
├── src/
│   ├── app/
│   ├── features/
│   │   ├── canvas/
│   │   └── history/
│   └── shared/
└── tests/
```

**Structure Decision**: [Document the selected structure and reference the real
directories captured above. Any deviation from the mandated Relay layering or
feature-based frontend structure must be justified in Complexity Tracking.]

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., extra dependency] | [current need] | [why standard library or existing stack is insufficient] |
| [e.g., layer exception] | [specific problem] | [why mandated handler -> orchestrator -> agent -> tool -> storage flow is insufficient] |
