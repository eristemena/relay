# Quickstart: Canvas Animation Layer

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- Existing Relay workspace bootstrapped locally
- Existing live orchestration canvas working before motion changes are applied

## Development Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
npm --prefix web install
npm --prefix web run typecheck
npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/canvas/AnimatedHandoffEdge.test.tsx src/features/canvas/AgentNodeDetailPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx src/features/canvas/canvasModel.test.ts src/features/canvas/layoutGraph.test.ts src/shared/lib/workspace-store.test.ts
```

If protocol payloads must change after implementation, validate the relevant backend paths too:

```bash
go test ./internal/handlers/ws ./internal/orchestrator/workspace ./tests/integration
```

## Run Relay

Development mode:

```bash
make dev
```

Manual backend run:

```bash
RELAY_DEV=true go run ./cmd/relay serve --dev --port 4747
```

Optional backend health check after startup:

```bash
curl -sf http://127.0.0.1:4747/api/healthz
```

## Expected Behavior

- New nodes scale and fade in when agents spawn.
- Node status changes cross-fade instead of snapping.
- Handoff edges pulse only during active handoff windows.
- The selected-node panel enters from the right and exits cleanly.
- The streaming border pulse appears only while tokens are actively arriving and clears after 300ms of silence.
- Empty, loading, and plain-language error states remain visible instead of being hidden by motion.
- Canvas pan, zoom, and selection remain responsive during live updates.

## Manual Validation Flow

1. Start Relay and open the live canvas.
2. Submit a goal and confirm the first node enters with the shared 300ms transition.
3. Let the Planner move through thinking and streaming states and confirm state presentation changes smoothly without moving the node.
4. Confirm the node border pulse remains active while visible tokens are arriving and stops shortly after token silence.
5. Let a downstream handoff begin and confirm the correct edge pulses, then returns to rest on handoff completion.
6. Select a node and confirm the detail panel slides in from the right.
7. Switch quickly between two nodes and confirm the panel ends on the latest selection without mixed content.
8. Trigger a reconnect or empty-run canvas state and confirm loading copy and plain-language error copy remain readable while the canvas surface stays stable.
9. Pan, zoom, and click the canvas repeatedly while nodes stream and edges pulse; confirm interaction remains responsive.
10. Enable reduced motion in the browser or OS preference and confirm the interface still communicates state changes with minimized movement.
11. Trigger node unmount or run teardown paths and confirm no stale streaming pulse continues after the node disappears.

## Focused Test Commands

Canvas motion and panel behavior:

```bash
npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/canvas/AnimatedHandoffEdge.test.tsx src/features/canvas/AgentNodeDetailPanel.test.tsx src/features/canvas/AgentCanvasNode.test.tsx
```

Canvas model regression coverage:

```bash
npm --prefix web test -- src/features/canvas/canvasModel.test.ts src/features/canvas/layoutGraph.test.ts src/shared/lib/workspace-store.test.ts
```

Type safety validation:

```bash
npm --prefix web run typecheck
```

## Failure Recovery Expectations

- If `AnimatePresence` placement causes React Flow wrapper instability, isolate node and panel motion tests before reconnecting them to live store events.
- If the streaming pulse continues after node unmount, timer cleanup in the node wrapper is incomplete and the implementation should be rejected.
- If edge pulses persist after `handoff_complete`, the edge data mapping is not settling correctly.
- If loading or plain-language error copy disappears while the canvas is still resolving data, the motion layer is masking required UI states and should be rejected.
- If pan, zoom, or selection become sluggish, remove any accidental layout animation or position-driven transitions before proceeding.