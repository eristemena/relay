# Quickstart: Static Agent Canvas

## Prerequisites

- Node.js version compatible with Next.js 16.2
- npm
- Existing Relay frontend dependencies installed in `web/`

## Dependency Setup

Install the canvas dependencies in the frontend workspace if they are not already present.

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
npm --prefix web install @xyflow/react @dagrejs/dagre
```

## Development Workflow

Start Relay in development mode.

```bash
make dev
```

If another Next.js app is already using port `3000`, start the Relay frontend on a free port in the dev-proxy scan range and then restart the Relay backend.

```bash
npm --prefix web run dev -- --port 3001
RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747
```

Expected behavior:

- The workspace shell still loads normally.
- The canvas area shows an explicit empty state before any nodes are added.
- A dev toolbar inside the canvas allows adding roles and mutating the selected node state without contacting the backend.

## Validation Commands

Run the focused frontend checks.

```bash
npm --prefix web run typecheck
npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/canvas/layoutGraph.test.ts src/features/workspace-shell/WorkspaceShell.test.tsx
```

## Manual Validation Flow

1. Open the Relay workspace in the browser and verify the canvas starts in an explicit empty state.
2. Add a Planner node and confirm it appears with the expected node chrome, role badge, and initial state.
3. Add a Coder node and confirm a directional edge appears from Planner to Coder.
4. Confirm the graph repositions cleanly after adding the second node.
5. Click the Coder node and verify the side panel opens with that node’s details.
6. Change the Coder node state to `thinking` from the toolbar and confirm the node style updates immediately.
7. Confirm the Coder node does not move when only its state changes.
8. Add a Tester node and confirm dagre repositions the graph without visible node disappearance or flicker.
9. Pan and zoom while adding nodes and while changing node state, then confirm the canvas remains responsive.
10. Click the canvas background and confirm the detail panel closes without mutating the graph.

## Automated Validation Snapshot

- `npm --prefix web run typecheck`
- `npm --prefix web test`
- `curl -I http://127.0.0.1:4747` returned `200 OK` after restarting Relay against the active Relay Next.js dev server

The targeted canvas tests cover empty-state replacement, node insertion, state-only updates preserving coordinates, detail panel open and close behavior, and shell integration.

## Quickstart Validation Notes

- Validated the Relay dev proxy path on 2026-03-24 with the frontend served from `http://127.0.0.1:3001` and Relay served from `http://127.0.0.1:4747`.
- Confirmed the proxied workspace HTML includes the updated shell header, live agent panel, and isolated canvas surface through the Relay backend.
- Confirmed the focused frontend suite and full frontend suite remain green after the shell and canvas UX refinements.

## Failure Recovery Expectations

- If a toolbar action is missing required information, the canvas shows a human-readable inline validation message and leaves the current graph untouched.
- If no node is selected, state-mutation controls stay disabled or explain why the action is unavailable.
- If the graph contains one node only, selection and panel behavior still work.
- If a layout pass expands the graph beyond the current viewport, pan and zoom remain available so the user can reach all nodes.