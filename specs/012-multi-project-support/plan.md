# Implementation Plan: Multi-Project Support

**Branch**: `012-multi-project-support` | **Date**: 2026-04-05 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/012-multi-project-support/spec.md`
**Input**: Feature specification from `/specs/012-multi-project-support/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Make project root the identity boundary for Relay's persisted workspace context so one running Relay instance can move between multiple codebases without cross-project state leakage. The backend will extend `sessions` with a canonical `project_root`, resolve startup root selection with `--root` first and current working directory second, automatically restore or create the persisted project context for that root, and scope history queries by joining run-history documents back to project-root metadata. The frontend will replace manual project-selection flows with a header project switcher, retire manual developer-facing session creation and open flows, treat project changes as a full project-context reset in the workspace store, and preserve an opt-in all-project history mode that removes only the project-root filter while keeping the active project unchanged. This plan assumes a fresh local Relay database created with the multi-project schema rather than an in-place upgrade path for pre-feature databases, even though the schema change still lands through the normal SQLite migration file used for clean initialization.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library first, especially `path/filepath`, `os`, `context`, `strings`, `sync`, and `net/http`; existing `github.com/spf13/cobra` for CLI flags; existing `nhooyr.io/websocket` workspace transport; existing SQLite store and React workspace store; no new third-party dependency is required for project identity, switching, or history scoping  
**Storage**: SQLite only; `sessions` gains a `project_root` column, while runs, events, approvals, and touched files remain keyed by `session_id` or `run_id`  
**Testing**: `go test` for `internal/storage/sqlite`, `internal/orchestrator/workspace`, `internal/handlers/ws`, `internal/app`, and `cmd/relay`; table-driven Go tests for startup root resolution, project-context lookup and creation, blocked switching, and project-scoped history queries; Vitest plus React Testing Library for header switcher rendering, store reset behavior, history scope toggling, and no-ghost-node regression coverage; integration coverage in `tests/integration` for workspace bootstrap and project switch protocol changes; switch-performance validation that demonstrates the 2-second target in SC-002  
**Target Platform**: Local Relay development on macOS and Linux workstations with browser UI on localhost; browser runtime remains driven by the Go backend over WebSocket  
**Project Type**: Full-stack Relay backend/frontend enhancement to workspace identity, project-context persistence, and history browsing  
**Performance Goals**: WebSocket status and switch responses remain under the existing 100ms event-dispatch expectations once emitted; switching projects rehydrates active-project header, canvas, history, and repository tree within 2 seconds in normal validation; React Flow canvas remains responsive because project switching clears stale documents before rendering the target project  
**Constraints**: Dark mode only; WebSocket remains the only Go-to-React communication channel; SQLite remains the only persistent store; one active project context at a time; project roots are stored and compared as cleaned absolute strings; handler-level approval enforcement for file writes and shell commands must remain unchanged across projects  
**Supported Project Root Rule**: A valid root for this feature is any existing readable directory that Relay can bind as the active repository and workspace scope; Git metadata and project-manifest detection are not required  
**Scale/Scope**: Single-user local workspace; dozens of known project roots over time; one active project and zero or one active run at a time; history queries can aggregate across all known projects but live canvas and repository state always belong to only the active project

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: the design stays within existing Go and strict TypeScript boundaries, prefers standard-library path resolution, and does not require new debugging or logging exceptions.
- [x] Test impact is defined: table-driven Go tests cover root normalization, project-context auto-provisioning, blocked switching, and history scoping; WebSocket protocol changes for project switching and bootstrap enrichment require integration coverage; frontend tests cover header switcher behavior and store-reset regression paths so ghost nodes are caught.
- [x] Architecture remains compliant: handlers continue to own protocol mapping, `internal/orchestrator/workspace` continues to own bootstrap and project-context selection, storage remains SQLite-only, frontend changes stay in feature-based folders, and the requested project listing endpoint is intentionally satisfied through the existing workspace socket rather than adding a second transport.
- [x] UX and governance impact is defined: the header exposes the active project root and switcher, switching and startup failures are plain-language, empty states are defined for single-project and no-history cases, and existing handler-level approval enforcement remains project-bound but otherwise unchanged.
- [x] Security and performance constraints are covered: project roots are canonicalized before persistence, file-system and shell boundaries stay scoped to the active project root, switching is blocked when a run is still active, workspace-store reset avoids stale cross-project artifacts, and query changes remain bounded rather than introducing N+1 project lookups.

## Project Structure

### Documentation (this feature)

```text
specs/012-multi-project-support/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── workspace-project-context.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── relay/
    └── main.go

internal/
├── app/
│   └── server.go
├── handlers/
│   └── ws/
│       ├── protocol.go
│       ├── workspace.go
│       └── workspace_test.go
├── orchestrator/
│   └── workspace/
│       ├── service.go
│       ├── runs.go
│       ├── history.go
│       ├── history_materialize.go
│       └── service_test.go
├── storage/
│   └── sqlite/
│       ├── migrations/
│       ├── queries/
│       │   └── sessions.sql
│       ├── models.go
│       ├── store.go
│       └── store_test.go
└── config/
    └── config.go

tests/
└── integration/

web/
└── src/
    ├── features/
    │   ├── workspace-shell/
    │   │   ├── WorkspaceShell.tsx
    │   │   ├── WorkspaceShell.test.tsx
    │   │   └── ProjectSwitcher.tsx
    │   ├── history/
    │   │   ├── RunHistoryPanel.tsx
    │   │   ├── RunHistoryListItem.tsx
    │   │   └── SessionSidebar.tsx
    │   └── canvas/
    │       └── WorkspaceCanvas.tsx
    └── shared/
        └── lib/
            ├── useWorkspaceSocket.ts
            ├── workspace-protocol.ts
            ├── workspace-store.ts
            └── workspace-store.test.ts
```

**Structure Decision**: Keep startup root resolution in `cmd/relay` plus `internal/app`, keep automatic project-context selection and project switching in `internal/orchestrator/workspace`, keep transport changes in `internal/handlers/ws`, and keep persistence changes in `internal/storage/sqlite`. On the frontend, place the new dropdown in `features/workspace-shell`, extend `features/history` for all-project filtering and project labels, and update the central workspace store to clear project-scoped artifacts when the active project changes. Existing manual session UI in `features/history` must be removed or repurposed so developers no longer create or open sessions manually. No layer exceptions are required.

## Complexity Tracking

No constitution violations are required by this design.
