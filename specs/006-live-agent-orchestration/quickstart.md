# Quickstart: Live Agent Orchestration

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- A writable home directory for Relay config and SQLite state
- Valid provider access already configured for Relay's backend-driven live runs

## Development Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
npm --prefix web install
npm --prefix web run typecheck
go test ./internal/agents ./internal/orchestrator/workspace ./internal/handlers/ws ./internal/storage/sqlite
```

Focused replay coverage:

```bash
go test ./tests/integration -run 'TestAgentStreaming_SubmitDeliversOrderedEventsAndRejectsSecondActiveRun|TestAgentStreaming_OpenRunReattachesActiveStreamAfterReconnect|TestRunHistoryReplay_RestartHydratesBootstrapAndReplaysRun'
```

Replay file validation:

```bash
go test ./tests/integration/run_history_replay_test.go
```

Production build validation:

```bash
make build
./bin/relay serve --help
```

For a packaged runtime smoke test, unset `RELAY_DEV` so Relay serves the embedded frontend instead of the Next.js dev proxy, and prefer a fresh port to avoid colliding with any existing local session:

```bash
env -u RELAY_DEV ./bin/relay serve --no-browser --port 4851
curl -sf http://127.0.0.1:4851/api/healthz
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

Optional backend smoke check after the server starts:

```bash
curl -sf http://127.0.0.1:4747/api/healthz
```

## Expected Behavior

- Relay boots into the existing workspace shell.
- The canvas remains available for the active session.
- Before the first orchestration run, the canvas and side panel show explicit idle or instructional states.
- During an active run, the canvas appends nodes only on spawn events and patches existing nodes for subsequent updates.
- If provider access is unavailable, goal submission fails with a plain-language blocked state.

## Manual Validation Flow

The commands above were validated on 2026-03-24. The remaining checks in this section are browser-driven and still need manual confirmation in a live workspace session.

If your shell already exports `RELAY_DEV=true`, packaged binary checks will report `frontend_mode:"proxied"` and the root page will intentionally return the development-server unavailable screen until `npm --prefix web run dev` is running.

1. Start Relay and confirm workspace bootstrap succeeds.
2. Submit a goal and confirm the Planner node appears first on the canvas within the expected responsiveness target.
3. Confirm the Planner node opens live output in the side panel when selected.
4. Let Planner complete and confirm both Coder and Tester nodes spawn without refreshing the page.
5. While Coder and Tester are both active, pan and zoom the canvas and switch node selection repeatedly; confirm the canvas remains responsive and the side panel always shows the currently selected agent.
6. Confirm new nodes are added only when agents spawn and that existing nodes do not disappear when only state or transcript events arrive.
7. Reproduce the prior disappearing-node risk by letting multiple transcript events arrive quickly and confirm live updates use patch semantics without full relayout churn.
8. Complete a successful run and confirm Reviewer then Explainer appear in sequence before a final `run_complete` event.
9. Open the completed run from history and confirm all nodes restore with the same identities, final states, and preserved transcripts.
10. Trigger an agent-scoped failure and confirm the affected node moves to an error state while the run remains inspectable and continues only if downstream rules allow it.
11. Trigger a run-level failure and confirm no further nodes spawn, the run enters a halted terminal state, and the canvas preserves already started nodes.
12. Reload the browser during an active run and confirm replay occurs without duplicate nodes, after which live updates resume for the active run.

## Focused Test Commands

Backend orchestration and WebSocket coverage:

```bash
go test ./internal/orchestrator/workspace ./internal/handlers/ws ./internal/storage/sqlite ./tests/integration
```

Current broader backend regression sweep:

```bash
go test ./tests/integration -run 'TestAgentStreaming_|TestWorkspaceWebSocket_' -count=1
```

Frontend live-canvas coverage:

```bash
npm --prefix web test -- src/features/canvas/AgentCanvas.test.tsx src/features/workspace-shell/WorkspaceShell.test.tsx
```

Frontend canvas model and layout regression coverage:

```bash
npm --prefix web test -- src/features/canvas/layoutGraph.test.ts src/features/canvas/AgentCanvas.test.tsx src/features/workspace-shell/WorkspaceShell.test.tsx
```

## Failure Recovery Expectations

- Missing provider access blocks orchestration start with a human-readable remediation message.
- If the Planner fails before downstream spawn, the run halts cleanly and no later nodes appear.
- If one parallel agent fails, its node remains inspectable and the run records that failure without losing prior output.
- If the WebSocket reconnects during an active run, replay plus reattachment restores visibility without duplicate nodes or lost transcript chunks.
- If the frontend accidentally reruns dagre on state-only events, the disappearing-node regression should be caught by component coverage before release.