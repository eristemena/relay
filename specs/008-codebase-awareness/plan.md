# Implementation Plan: Codebase Awareness

**Branch**: `008-codebase-awareness` | **Date**: 2026-03-25 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/008-codebase-awareness/spec.md`
**Input**: Feature specification from `/specs/008-codebase-awareness/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extend Relay from a generic `project_root` preference into a true single-repository working context. The backend will validate one connected local Git repository, expose repo-aware tools for file reads, file listing, code search, commit history, working-tree diff, diff-first writes, and sandboxed command execution, and keep all write or command mutations blocked behind a persisted SQLite approval state machine. Repository introspection will use `go-git` rather than a system Git dependency, while `run_command` continues on `os/exec` with repo-root validation before every execution. The frontend will add a repository connection flow anchored to the existing preferences surface, a server-backed folder browser, a Monaco side-by-side diff reviewer for pending writes, and canvas-level agent file activity derived from tool and approval events while repository relationship data continues to build asynchronously in the background.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library first, including `os/exec`, `context`, `filepath`, and `sync`; new backend dependency `github.com/go-git/go-git/v5` for repository validation, tree traversal, commit log access, and working-tree diffs without a system Git dependency; new frontend dependency `monaco-editor` for side-by-side diff review; existing `@xyflow/react`, `framer-motion`, and workspace protocol/store layers remain in use  
**Storage**: SQLite only, with new persisted approval-request state and possible repository-analysis metadata; existing config file remains the source of truth for `project_root`  
**Testing**: `go test` for `internal/tools`, `internal/orchestrator/workspace`, `internal/handlers/ws`, `internal/config`, and integration coverage under `tests/integration`; table-driven Go tests for approval state transitions and repo sandbox validation; Vitest plus React Testing Library for preferences connection flow, approval review UI, repository-context store states, canvas agent activity indicators, and workspace store protocol changes  
**Target Platform**: Local Relay developer workflow on macOS-first machines with browser UI on localhost; behavior remains compatible with standard desktop browsers supported by Next.js 16  
**Project Type**: Full-stack Relay backend/frontend feature extending repository awareness, approval governance, and live workspace visibility  
**Performance Goals**: WebSocket dispatch remains under 100ms per backend event; repository validation and initial context scheduling complete without blocking workspace bootstrap; repository-context construction runs in a cancellable background goroutine and never blocks the main request path; pending approvals survive UI disconnect and reconnect; canvas and sidebars remain responsive while repository updates and approval events stream in  
**Constraints**: Dark mode only; WebSocket remains the only backend/frontend runtime channel; SQLite remains the only data store; exactly one connected repository at a time; file writes and shell commands require explicit handler-level approval; `run_command` must validate the effective working directory before every execution; no remote repository access or automatic Git mutations; no prompt or response logging; all goroutines require `context.Context` cancellation  
**Scale/Scope**: Single-user local workstation; one active connected repository; repositories ranging from small projects to large local codebases where graph generation must degrade gracefully and stay asynchronous; approval requests may outlive a browser session and must recover on reconnect

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: Go changes stay in the existing handler -> orchestrator -> tools -> storage layers, `go-git` is the only new backend dependency, exported Go APIs added for repo-aware services or models will require godoc comments, and frontend changes remain in strict TypeScript feature folders with no banned debug logging.
- [x] Test impact is defined: table-driven Go tests cover approval state transitions, repo validation, path traversal rejection, go-git read paths, and run-command sandboxing; WebSocket integration tests cover new repository and approval payloads; React component and store tests cover the Monaco review surface, repository-context state handling, and agent node file activity indicators.
- [x] Architecture remains compliant: handlers accept repository and approval actions, orchestrator/workspace owns approval coordination and graph orchestration, tools stay repo-scoped, storage remains SQLite-only, and the frontend keeps feature-based folders for approvals, codebase visibility, canvas, and preferences.
- [x] UX and governance impact is defined: repository connection, background context building, and approval review all expose visible loading states, plain-language errors, and explicit empty states; write and command execution remain impossible without server-recorded approval.
- [x] Security and performance constraints are covered: repo-scoped file access stays enforced by canonical path validation plus Git-repository checks, `run_command` revalidates repo root before every execution, background repository analysis runs in cancellable goroutines, prompt/response logging is unchanged, and approval persistence prevents silent loss of pending actions.

## Project Structure

### Documentation (this feature)

```text
specs/008-codebase-awareness/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── workspace-codebase-awareness.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── relay/
    └── main.go

internal/
├── config/
│   └── config.go
├── handlers/
│   └── ws/
│       ├── protocol.go
│       └── workspace.go
├── orchestrator/
│   └── workspace/
│       ├── service.go
│       ├── runs.go
│       ├── tool_executor.go
│       ├── repository_graph.go
│       └── repository_browser.go
├── storage/
│   └── sqlite/
│       ├── migrations/
│       ├── queries/
│       ├── models.go
│       └── store.go
└── tools/
    ├── catalog.go
    ├── path_guard.go
    ├── read_file.go
    ├── write_file.go
    ├── list_files.go
    ├── search_codebase.go
    ├── git_log.go
    ├── git_diff.go
    └── run_command.go

tests/
└── integration/
    ├── tool_call_ordering_test.go
    ├── workspace_sessions_test.go
    └── codebase_awareness_test.go

web/
└── src/
    ├── features/
    │   ├── approvals/
    │   │   ├── ApprovalReviewPanel.tsx
    │   │   └── ApprovalReviewPanel.test.tsx
    │   ├── canvas/
    │   │   ├── AgentCanvasNode.tsx
    │   │   ├── AgentNodeDetailPanel.tsx
    │   │   └── canvasModel.ts
    │   ├── codebase/
    │   │   └── graphModel.ts
    │   ├── preferences/
    │   │   ├── PreferencesPanel.tsx
    │   │   └── PreferencesPanel.test.tsx
    │   └── workspace-shell/
    │       ├── WorkspaceShell.tsx
    │       └── WorkspaceStatusBanner.tsx
    └── shared/
        └── lib/
            ├── workspace-protocol.ts
            └── workspace-store.ts
```

**Structure Decision**: Keep repository connection and approval orchestration in the existing backend layers instead of creating a parallel repo service. `internal/tools` remains the home for repo-aware tool implementations, `internal/orchestrator/workspace` owns approval persistence and background repository-analysis coordination, and `internal/handlers/ws` owns the protocol surface. On the frontend, keep dedicated `features/approvals` for the Monaco diff reviewer while `features/canvas` consumes derived file-activity state for each agent node and `features/workspace-shell` presents repository connection status and saved workspace defaults. The existing preferences feature remains the place where `project_root` is configured or chosen, rather than introducing a second disconnected repository-settings surface.

## Complexity Tracking

No constitution violations are required by this design.
