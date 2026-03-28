# Quickstart: Run History and Replay

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- Existing Relay workspace bootstrapped locally
- A local Relay data directory with saved runs to replay, or test coverage that seeds run history and approval diffs in SQLite

## Development Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
go test ./internal/orchestrator/workspace ./internal/storage/sqlite ./internal/handlers/ws
npm --prefix web install
npm --prefix web run typecheck
```

No new third-party package is required if replay scheduling, markdown export, and FTS-backed queries use the existing Go and SQLite stack.

Focused frontend validation after implementation:

```bash
npm --prefix web test -- src/features/history/replay/RunChangeReviewPanel.test.tsx src/features/canvas/AgentCanvas.test.tsx src/features/history/RunHistoryPanel.test.tsx src/features/history/replay/ReplayControls.test.tsx src/features/history/replay/ReplayTimeline.test.tsx src/shared/lib/workspace-store.test.ts
```

Focused integration validation after protocol and replay changes:

```bash
go test ./tests/integration -run 'TestRunHistoryReplay_|TestWorkspaceHistory_|TestReplay'
```

## Run Relay

Start Relay in development mode:

```bash
make dev
```

Optional health check after startup:

```bash
curl -sf http://127.0.0.1:4747/api/healthz
```

## Expected Behavior

- The run history panel shows saved runs with generated title, recorded date, agent count, and final status.
- Selecting a saved run starts historical replay from stored SQLite events without invoking any agent or tool.
- Playback controls allow play, pause, 0.5x, 1x, 2x, and 5x speeds, plus seek via a timestamp-driven scrubber.
- Seeking rebuilds the canvas, transcript, token-usage fill bars, and run-summary surfaces to the target timestamp quickly for sessions under 10 minutes.
- Historical file changes render from stored before and after content, even if the current repository contents differ.
- Search combines keyword text across titles, goals, summaries, replay-safe transcript snippets, and touched-file matching with date range filters using SQLite FTS5 plus structural predicates.
- Export writes a markdown report for the selected run under `~/.relay/exports/` only when the request comes from an explicit developer action in the workspace UI.

## Manual Validation Flow

1. Start Relay and open a workspace that already contains at least one saved run with persisted events.
2. Open the run history surface and confirm each entry shows a generated title, date, agent count, and final status.
3. Select a saved run and confirm the canvas replays from historical events rather than showing new live activity.
4. Change playback speed across all supported options and confirm event order stays stable while timing changes.
5. Drag the scrubber to multiple positions and confirm the canvas, transcript, node details, approvals, and token-usage bars reflect the target timestamp.
6. Search by keyword and by touched file, then add a date range filter and confirm the visible history list reflects the combined criteria.
7. Open a run with historical file changes and confirm the diff review surface shows full preserved before and after content for each file.
8. Export the selected run from `Export Report` and confirm a markdown file appears in `~/.relay/exports/` with run metadata, timeline summary, and file-change sections.
9. Attempt an equivalent export through a non-user path in automated coverage and confirm the server rejects it instead of writing a file.
10. Reconnect the WebSocket while replay is selected and confirm the same run and replay state are restored without duplicate history entries.

## Focused Test Commands

Replay scheduling, seek reconstruction, and historical export:

```bash
go test ./internal/orchestrator/workspace ./internal/handlers/ws -run 'Test.*(Replay|Seek|Export|HistoryQuery)'
```

SQLite persistence, FTS search, and change-record extraction:

```bash
go test ./internal/storage/sqlite -run 'Test.*(RunHistory|RunChange|FTS|Export)'
```

Frontend history, scrubber, and workspace-store behavior:

```bash
npm --prefix web test -- src/features/history/replay/RunChangeReviewPanel.test.tsx src/features/canvas/AgentCanvas.test.tsx src/features/history/RunHistoryPanel.test.tsx src/features/history/replay/ReplayControls.test.tsx src/features/history/replay/ReplayTimeline.test.tsx src/shared/lib/workspace-store.test.ts
```

Type safety validation:

```bash
npm --prefix web run typecheck
```

## Failure Recovery Expectations

- If historical events are incomplete for an older run, replay remains deterministic but surfaces missing details explicitly rather than inventing them.
- If FTS search data is unavailable for a run, the history query fails with a human-readable error instead of silently omitting records.
- If export cannot write to `~/.relay/exports/`, Relay reports the failure in plain language and leaves replay state unchanged.
- If export is requested without a direct developer action reaching the handler boundary, Relay rejects the request in plain language and writes no file.
- If a run has no historical file changes, the diff review surface shows an explicit empty state.
- If a replay seek request arrives during active playback, the scheduler pauses, rebuilds state at the requested timestamp, and resumes only when instructed.