# Implementation Plan: Repository File Tree Sidebar

**Branch**: `011-repo-file-tree` | **Date**: 2026-03-29 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/011-repo-file-tree/spec.md`
**Input**: Feature specification from `/specs/011-repo-file-tree/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a read-only repository File Tree panel to Relay's right-side workspace detail rail. During live runs, the right rail shows File Tree directly because Historical Replay is not relevant yet. When the developer reopens a saved run, the right rail switches to a top-level tabbed surface with Historical Replay and File Tree as peer panels while run history remains in the existing left panel. The backend will build a flat, recursive repository path list at repository-connect time using `go-git` worktree iteration, while excluding paths ignored by the repository's `.gitignore` rules so large directories such as `node_modules` do not bloat hydration. SQLite will add a `touched_files` table keyed by run, agent, path, and touch type so file-read and file-proposal activity can be recovered on reconnect and filtered per agent. The initial tree snapshot will be delivered over the existing workspace WebSocket protocol rather than a separate HTTP endpoint so the design remains constitution-compliant, while live `file_touched` events will be emitted from the tool execution path immediately when reads succeed or write proposals are created. The frontend will derive the selected-agent view entirely client-side from the touched-file store, show only top-level entries plus one nested level by default for large repositories, and expand deeper folders on demand. Touched status will use a non-color-only indicator on each file row, combining a small icon or badge with accessible text or state so read and proposed activity remain perceivable in normal, high-contrast, and filtered views.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library first, especially `context`, `path/filepath`, `strings`, `slices`, `sync`, and `encoding/json`; existing `github.com/go-git/go-git/v5` for recursive worktree file iteration during repository connection; existing `nhooyr.io/websocket` workspace transport; existing React Flow workspace store and sidebar surfaces; no new third-party dependency is required if tree materialization and client filtering use current stack primitives  
**Storage**: SQLite only for touched-file persistence via a new `touched_files` table and supporting queries; repository directory structure stays in memory inside the workspace service and is rebuilt when the connected repository changes rather than persisted in SQLite  
**Testing**: `go test` for `internal/orchestrator/workspace`, `internal/storage/sqlite`, `internal/handlers/ws`, and `internal/tools`; table-driven Go tests for tree materialization, touched-file persistence, duplicate touch deduplication, reconnect restoration, and event emission timing; Vitest plus React Testing Library for right-rail panel switching, File Tree rendering, expansion state, agent filtering, and read-only behavior; integration coverage in `tests/integration` for workspace bootstrap, tree hydration, and `file_touched` WebSocket protocol changes  
**Target Platform**: Local Relay development on macOS-first workstations with browser UI on localhost; runtime remains browser-based with WebSocket transport only  
**Project Type**: Full-stack Relay backend/frontend enhancement to repository awareness and workspace sidebar UX  
**Performance Goals**: Connected-repository tree hydration starts rendering within 2 seconds for supported repos; WebSocket `file_touched` events arrive within 100ms of the underlying tool-layer activity; initial dock render shows only top-level entries plus one nested level by default so repositories with thousands of files do not stall the UI; ignored directories such as `node_modules` are excluded when `.gitignore` says they should be; canvas pan, zoom, and node selection remain responsive while tree updates stream in  
**Constraints**: Dark mode only; WebSocket remains the only supported backend/frontend runtime channel, so the requested GET tree endpoint is replaced by an equivalent workspace request/result message; SQLite remains the only persistent store; file tree is strictly read-only; touched-file writes happen at read completion and write proposal creation time, not approval time; file access stays bounded to the connected repository; all goroutines require `context.Context` cancellation  
**Scale/Scope**: Single-user local workstation; one connected repository; repos may contain thousands of files and deeply nested directories; one active run is the primary touched-file scope; selected-agent filtering must work without extra round trips once the client has the current run's touched-file set

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: Go changes stay in handler -> orchestrator -> tools -> storage flow, standard library plus existing `go-git` cover the backend design, exported Go APIs added for repository-tree or touched-file services will require godoc comments, and frontend changes remain strict TypeScript with no banned debug logging.
- [x] Test impact is defined: table-driven Go tests cover tree building, touched-file writes from read and write paths, deduped agent/path state, reconnect hydration, and tool-layer event timing; WebSocket integration tests cover new tree request/result and `file_touched` payloads; React component and store tests cover right-rail panel switching, expansion state, agent filtering, and read-only interaction constraints.
- [x] Architecture remains compliant: handlers own protocol changes, `internal/orchestrator/workspace` owns tree caching and touched-file coordination, tool execution remains the earliest safe layer for file-touch recording, storage remains SQLite-only, frontend stays in feature-based folders, and the requested HTTP endpoint is intentionally replaced by a WebSocket request/result to preserve the constitution.
- [x] UX and governance impact is defined: the right-side workspace detail rail exposes File Tree directly for live runs and top-level Historical Replay/File Tree tabs for reopened saved runs, with visible loading, plain-language errors, and explicit empty states for repository tree, agent-filtered no-results, and unavailable repository conditions; the tree remains read-only and does not alter existing handler-level approval enforcement for file writes or shell commands.
- [x] Security and performance constraints are covered: file paths remain repo-scoped and canonicalized, the tree cache is rebuilt only for the connected repo root, touched-file persistence avoids N+1 recovery queries by loading the current run's touched set in bulk, all live update workers remain cancellable, and initial render depth limiting keeps the canvas responsive while preserving sub-100ms event dispatch.

## Project Structure

### Documentation (this feature)

```text
specs/011-repo-file-tree/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── workspace-repository-file-tree.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── handlers/
│   └── ws/
│       ├── protocol.go
│       ├── workspace.go
│       └── workspace_test.go
├── orchestrator/
│   └── workspace/
│       ├── repository_browser.go
│       ├── service.go
│       ├── history.go
│       ├── tool_executor.go
│       ├── tool_executor_test.go
│       ├── repository_browser_test.go
│       └── repository_tree_*.go
├── storage/
│   └── sqlite/
│       ├── migrations/
│       ├── queries/
│       ├── models.go
│       ├── store.go
│       └── store_test.go
└── tools/
    ├── read_file.go
    ├── write_file.go
    ├── read_file_test.go
    └── write_file_test.go

tests/
└── integration/
    └── repository_file_tree_test.go

web/
└── src/
    ├── features/
    │   ├── canvas/
    │   │   ├── WorkspaceCanvas.tsx
    │   │   └── canvasModel.ts
    │   ├── codebase/
    │   │   ├── RepositoryFileTreePanel.tsx
    │   │   ├── RepositoryFileTreePanel.test.tsx
    │   │   └── treeModel.ts
    │   ├── history/
    │   │   ├── RunHistoryPanel.tsx
    │   │   ├── SessionSidebar.tsx
    │   │   └── SidebarTabs.tsx
    │   └── workspace-shell/
    │       ├── WorkspaceShell.tsx
    │       └── WorkspaceShell.test.tsx
    └── shared/
        └── lib/
            ├── workspace-protocol.ts
            ├── workspace-store.ts
            └── workspace-store.test.ts
```

**Structure Decision**: Keep repository tree hydration and touched-file coordination in `internal/orchestrator/workspace` because that layer already owns connected-repository context, run bootstrap, and workspace stream delivery. `internal/handlers/ws` remains the only place where request/result and event protocol types are added. `internal/storage/sqlite` stores durable touched-file state, while `internal/tools` remains unchanged as the bounded execution surface for file access. On the frontend, implement the actual tree surface under `features/history` and compose it into the existing `ReplayDock` through a tabbed dock container in `WorkspaceShell.tsx`, while leaving run history in the existing left panel so the repository tree stays side by side with the graph.

## Complexity Tracking

No constitution violations are required by this design.
