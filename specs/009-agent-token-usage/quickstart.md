# Quickstart: Agent Token Usage

## Prerequisites

- Go 1.26
- Node.js LTS compatible with Next.js 16.2
- npm
- A configured Relay workspace with an OpenRouter API key for live provider-backed validation
- Existing Relay workspace bootstrapped locally

## Development Setup

```bash
cd /Volumes/xpro/erisristemena/made-by-ai/relay
go test ./internal/agents/openrouter ./internal/orchestrator/workspace ./internal/storage/sqlite ./internal/handlers/ws
npm --prefix web install
npm --prefix web run typecheck
```

No new third-party package is required if the OpenRouter model metadata refresh uses the Go standard library.

Focused frontend validation after implementation:

```bash
npm --prefix web test -- src/features/canvas/AgentCanvasNode.test.tsx src/features/canvas/AgentCanvas.test.tsx src/shared/lib/workspace-store.test.ts
```

Focused canvas-model validation for token-state derivation:

```bash
npm --prefix web test -- src/features/canvas/canvasModel.test.ts
```

Focused integration validation after protocol and persistence changes:

```bash
go test ./tests/integration -run 'TestAgentStreaming_|TestRunHistoryReplay_'
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

- Relay loads model context-window metadata at startup when OpenRouter is configured, then refreshes it on a bounded TTL instead of querying on every completion.
- Single-agent `complete` events include `tokens_used` and `context_limit` when the final OpenRouter streaming chunk provides authoritative usage data.
- Orchestration agent nodes update their token-usage bar when the corresponding stage reaches completion and usage data is available.
- The canvas token bar transitions from neutral to amber to red as usage approaches the context limit.
- Replayed runs show the same token-usage state for events stored after this feature ships.
- Runs or events without provider usage data show an explicit unavailable state instead of a guessed token count.

## Manual Validation Flow

1. Start Relay with a valid OpenRouter API key and confirm the workspace loads normally.
2. Submit a run that completes successfully and confirm the terminal provider chunk includes usage data in backend validation logs or tests without exposing prompt or response content.
3. Confirm the active canvas updates the relevant agent node with a visible token-usage bar after the stage completes.
4. Validate that low usage renders the neutral state, near-limit usage renders amber, and over-limit or near-exhaustion usage renders red.
5. Submit a run whose model limit cannot be resolved from OpenRouter metadata and confirm the node shows a plain unavailable or raw-count fallback instead of crashing.
6. Replay a run created after this feature and confirm the token-usage bar matches the original stored state.
7. Replay an older run created before this feature and confirm the canvas remains stable while token usage stays unavailable.
8. Validate that missing or zero provider usage in the final chunk does not generate an estimated token value.

## Focused Test Commands

OpenRouter final-chunk usage and fallback resolution:

```bash
go test ./internal/agents/openrouter ./internal/orchestrator/workspace -run 'Test.*(Usage|ContextLimit|Complete)'
```

SQLite persistence and replay hydration:

```bash
go test ./internal/storage/sqlite ./internal/handlers/ws ./tests/integration -run 'Test.*(RunEvent|Replay|TokenUsage)'
```

Frontend canvas and store behavior:

```bash
npm --prefix web test -- src/features/canvas/AgentCanvasNode.test.tsx src/features/canvas/AgentCanvas.test.tsx src/shared/lib/workspace-store.test.ts
```

Type safety validation:

```bash
npm --prefix web run typecheck
```

Validated on 2026-03-27 with the focused backend and frontend commands above.

## Failure Recovery Expectations

- If startup model metadata fetch fails, Relay remains usable and falls back to locally known model limits where available.
- If the final OpenRouter chunk omits usage data, Relay completes the run normally and leaves token usage unavailable.
- If a stored row has null or invalid token fields, replay still succeeds and the canvas renders the unavailable state.
- If token usage exceeds the stored context limit, the bar caps at full width and renders the critical state instead of overflowing layout.