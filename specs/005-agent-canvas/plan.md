# Implementation Plan: Static Agent Canvas

**Branch**: `005-agent-canvas` | **Date**: 2026-03-24 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/005-agent-canvas/spec.md`
**Input**: Feature specification from `/specs/005-agent-canvas/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a frontend-only agent canvas inside the existing Relay workspace shell using a controlled React Flow graph with a custom agent node, a local developer toolbar for inserting and mutating nodes, dagre-based directed layout applied only when graph structure changes, and a side-panel inspector driven by stable node selection. The design keeps all state local to the canvas feature, does not touch WebSocket or backend storage, and explicitly optimizes for zero flicker and uninterrupted pan, zoom, and click interaction during node addition and state updates.

## Technical Context

**Language/Version**: TypeScript 5.8.x in strict mode, React 19.1, Next.js 16.2 App Router frontend  
**Primary Dependencies**: Existing frontend stack (`next`, `react`, `react-dom`, `framer-motion`, `tailwindcss`, `vitest`, `@testing-library/react`) plus planned additions `@xyflow/react` for the controlled graph canvas and `@dagrejs/dagre` for directed graph layout  
**Storage**: No persistent storage; all node, edge, selection, and detail state remains in local client memory for this isolated experience  
**Testing**: `npm --prefix web test` with Vitest and React Testing Library for custom node rendering, toolbar actions, detail panel open and close behavior, layout stability on state-only updates, and shell-level rendering; `npm --prefix web run typecheck` for TypeScript validation  
**Target Platform**: Local Relay browser UI on macOS-first developer workstations, with responsive behavior preserved down to 320px-wide layouts  
**Project Type**: Frontend-only Relay feature integrated into the existing workspace shell  
**Performance Goals**: Node state changes render within the current interaction frame without triggering relayout, layout recalculation occurs only on structure changes, and the canvas remains responsive to pan, zoom, and click input during updates  
**Constraints**: Dark mode only; feature-based frontend folder organization only; no backend handlers, WebSocket protocol changes, or SQLite changes; no `any`; no console debugging; accessible keyboard and focus behavior; no node disappearance or visible flicker during updates  
**Scale/Scope**: Single local developer, one isolated canvas view, low tens of nodes in the initial design surface, no persistence and no multi-user state

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: the change stays entirely in strict TypeScript React code, uses functional components only, preserves the existing token-based styling system, and introduces no banned debug logging.
- [x] Test impact is defined: component tests cover the custom React Flow node, dev toolbar actions, detail panel selection behavior, and the no-relayout-on-state-change invariant required by the spec.
- [x] Architecture remains compliant: no backend layers are modified, frontend work stays in feature-based folders under `web/src/features`, and existing SQLite/WebSocket-only runtime boundaries remain unchanged because the feature is explicitly isolated.
- [x] UX and governance impact is defined: the design includes explicit empty, selected, and validation error states and avoids misleading approval or live-execution affordances in the isolated canvas.
- [x] Security and performance constraints are covered: no secrets, file access, or shell execution are involved; the plan keeps canvas updates local and non-blocking while preserving keyboard, focus, and reflow behavior.

## Project Structure

### Documentation (this feature)

```text
specs/005-agent-canvas/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── canvas-ui.md
└── tasks.md
```

### Source Code (repository root)

```text
web/
├── package.json
├── src/
│   ├── app/
│   │   ├── globals.css
│   │   ├── layout.tsx
│   │   └── page.tsx
│   ├── features/
│   │   ├── agent-panel/
│   │   │   └── StateBadge.tsx
│   │   ├── canvas/
│   │   │   ├── CanvasEmptyState.tsx
│   │   │   ├── WorkspaceCanvas.tsx
│   │   │   ├── AgentCanvas.tsx
│   │   │   ├── AgentCanvasNode.tsx
│   │   │   ├── AgentCanvasToolbar.tsx
│   │   │   ├── AgentNodeDetailPanel.tsx
│   │   │   ├── layoutGraph.ts
│   │   │   ├── canvasModel.ts
│   │   │   ├── AgentCanvas.test.tsx
│   │   │   └── layoutGraph.test.ts
│   │   └── workspace-shell/
│   │       ├── WorkspaceShell.tsx
│   │       └── WorkspaceShell.test.tsx
│   └── shared/
│       └── lib/
│           └── test-helpers.tsx
└── vitest.config.ts
```

**Structure Decision**: Extend the existing `features/canvas` area rather than creating a new application surface or sharing canvas state through the global workspace store. `WorkspaceCanvas.tsx` remains the workspace integration point, while the new isolated graph behavior lives in feature-local files for model shaping, layout, toolbar actions, and detail presentation. `features/agent-panel/StateBadge.tsx` can be reused for state semantics if its current styling contract fits the canvas node without leaking live-run assumptions; otherwise the canvas feature will wrap it rather than fork it.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Add `@xyflow/react` | Relay does not currently have an interaction-capable node graph primitive | Building pan, zoom, selection, edges, and node rendering from scratch would increase surface area and testing burden for no product gain |
| Add `@dagrejs/dagre` | The feature requires automatic directed layout that runs only on structure changes | Hand-written placement rules would become brittle once more than two or three connected roles are added and would not generalize cleanly |
