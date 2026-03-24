# Implementation Plan: Canvas Animation Layer

**Branch**: `007-canvas-animation-layer` | **Date**: 2026-03-24 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/007-canvas-animation-layer/spec.md`
**Input**: Feature specification from `/specs/007-canvas-animation-layer/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a presentation-only motion layer to Relay's existing live agent canvas. The implementation keeps orchestration state authoritative in the current workspace store and canvas document model, wraps agent nodes in `motion.div` for enter and state transitions, introduces a custom React Flow edge that pulses based on handoff edge data, derives streaming-border activity from recent token arrival without adding server-owned animation state, and animates the node detail panel with `AnimatePresence`. The design explicitly avoids Framer Motion layout animations and limits motion to opacity, scale, and color-adjacent presentation so React Flow interaction remains responsive.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript 5.8 in strict mode; React 19.1; Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library and existing backend packages remain unchanged by default; existing frontend dependencies `@xyflow/react` for the canvas and existing `framer-motion` for node and panel animation; existing global CSS for state glow styling; no new third-party dependency is required  
**Storage**: SQLite only for existing run persistence; no new persisted animation state  
**Testing**: Vitest plus React Testing Library for component and interaction coverage in `web/src/features/canvas`; existing Go/WebSocket tests only if protocol payloads must expand to expose animation-relevant state; browser validation for reduced motion and live responsiveness  
**Target Platform**: Local Relay browser UI on macOS-first development machines, with behavior expected to remain browser-safe for standard desktop environments  
**Project Type**: Frontend-focused Relay feature layered on the existing live orchestration canvas, with optional minimal protocol shaping if current event payloads prove insufficient  
**Performance Goals**: Preserve sub-100ms perceived interaction delay for pan, zoom, and node selection during live updates; start node entry animation within 100ms of spawn visibility; clear streaming pulse within 300ms of token silence; avoid layout thrash on the React Flow canvas  
**Constraints**: Dark mode only; WebSocket remains the only backend/frontend runtime channel; animation is presentation-only and cannot set orchestration state; use the project easing curve `cubic-bezier(0.16, 1, 0.3, 1)` and a 300ms standard duration; do not use Framer Motion `layout` on the canvas; animate only opacity, scale, and color-adjacent presentation; preserve reduced-motion behavior plus visible loading states, human-readable error states, and explicit empty/non-selected states  
**Scale/Scope**: One existing live canvas surface, one existing detail panel surface, five built-in agent roles already rendered by the current graph, and motion driven by existing live events such as `agent_spawned`, `agent_state_changed`, `token`, `handoff_start`, and `handoff_complete`

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: changes are concentrated in strict TypeScript frontend feature files and existing CSS, no backend architectural expansion is planned by default, and no banned debug logging is introduced.
- [x] Test impact is defined: React Flow node and canvas component tests will be extended for motion entry, state transitions, edge pulse behavior, side-panel presence behavior, reduced-motion handling, and cleanup of streaming timers; backend integration tests are only required if animation-relevant protocol fields change.
- [x] Architecture remains compliant: handlers -> orchestrator -> agents -> tools -> storage remains untouched for the core execution path, frontend changes stay inside feature-based canvas folders plus shared protocol/store files if needed, and WebSocket/SQLite-only boundaries remain intact.
- [x] UX and governance impact is defined: motion preserves explicit idle, loading, error, empty, and non-selected states; side-panel changes keep human-readable plain-language errors and readable selection transitions; no file-write or shell-command approval behavior changes.
- [x] Security and performance constraints are covered: no new file or shell access is introduced, no prompt/response logging changes are required, motion is derived from existing safe payloads, and the design explicitly avoids layout-driven canvas thrash.

## Project Structure

### Documentation (this feature)

```text
specs/007-canvas-animation-layer/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── websocket-animation-signals.md
└── tasks.md
```

### Source Code (repository root)

```text
web/
├── package.json
└── src/
    ├── app/
    │   └── globals.css
    ├── features/
    │   ├── canvas/
    │   │   ├── AgentCanvas.tsx
    │   │   ├── AgentCanvas.test.tsx
    │   │   ├── AgentCanvasNode.tsx
    │   │   ├── AgentNodeDetailPanel.tsx
    │   │   ├── WorkspaceCanvas.tsx
    │   │   ├── canvasModel.ts
    │   │   ├── layoutGraph.ts
    │   │   └── AnimatedHandoffEdge.tsx
    │   └── agent-panel/
    │       └── StateBadge.tsx
    └── shared/
        └── lib/
            ├── workspace-protocol.ts
            └── workspace-store.ts

internal/
├── handlers/
│   └── ws/
│       └── protocol.go
└── orchestrator/
    └── workspace/
```

**Structure Decision**: Keep the implementation centered in `web/src/features/canvas`, because the feature is fundamentally a motion treatment over the existing graph and detail panel. `AgentCanvas.tsx`, `AgentCanvasNode.tsx`, `AgentNodeDetailPanel.tsx`, `canvasModel.ts`, and `globals.css` absorb most of the work; a dedicated custom edge component under the same feature folder holds the handoff pulse implementation. Shared protocol or store files change only if the current event payloads do not expose enough information to derive animation state safely. Backend Go files are listed only as conditional touch points for a minimal protocol-field extension, not as a planned core workstream.

## Complexity Tracking

No constitution violations are required by this design.
