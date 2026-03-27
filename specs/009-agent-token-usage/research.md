# Research: Agent Token Usage

## Decision 1: Capture usage only from the final OpenRouter streaming chunk and treat missing or zero usage as unavailable

- Decision: Extend the OpenRouter streaming client so the terminal callback receives normalized completion metadata containing `finish_reason` plus `usage.total_tokens` when the final chunk includes a non-zero usage object. Intermediate chunks remain token-text only and never update token usage.
- Rationale: The feature risk explicitly states that OpenRouter reports usage only in the final streaming chunk. Treating earlier chunks or zero-valued usage as authoritative would fabricate precision the provider did not supply.
- Alternatives considered:
  - Accumulate client-side token estimates from streamed text: rejected because the feature explicitly excludes estimated counts.
  - Read usage from every chunk: rejected because OpenRouter does not provide reliable usage in intermediate chunks.
  - Require usage to exist for every completion: rejected because Relay must degrade gracefully when the provider omits it.

## Decision 2: Keep Relay on its existing embedded SQL migration path instead of introducing Goose

- Decision: Add the schema change as the next embedded SQL migration file in `internal/storage/sqlite/migrations` and extend the store/query layer to read and write the new nullable columns.
- Rationale: Relay already replays embedded SQL migrations on startup. Introducing Goose for one feature would add a second migration system with no product value.
- Alternatives considered:
  - Introduce Goose just for this migration: rejected because it conflicts with the repository's current migration mechanism.
  - Store token usage only in `payload_json`: rejected because the request explicitly asks for SQLite persistence alongside the event log via new event-table fields.
  - Create a separate token-usage table: rejected because the data is event-scoped and belongs with `agent_run_events`.

## Decision 3: Persist token usage in typed nullable columns and keep payload JSON aligned for replay compatibility

- Decision: Add nullable `tokens_used` and `context_limit` columns to `agent_run_events`, update append/read queries to write those columns for terminal events, and continue storing the event payload JSON with the same fields when they are present.
- Rationale: Typed columns preserve event-adjacent telemetry for future queries and explicit persistence requirements, while aligned JSON payloads keep replay code simple and backward-compatible.
- Alternatives considered:
  - Persist only in columns and synthesize JSON on replay: rejected because it complicates replay and duplicates payload-construction logic.
  - Persist only in JSON: rejected because it does not satisfy the requested event-table column addition.
  - Backfill older rows during migration: rejected because historical backfill is out of scope and older runs must simply remain null-safe.

## Decision 4: Build a startup-loaded, TTL-cached model-limit registry with provider-aware fallback resolution

- Decision: Load OpenRouter model metadata from `/api/v1/models` at startup into an in-memory cache with a TTL refresh path, keyed by model name and carrying context-window limits. Add a resolver that falls back to locally configured or hardcoded defaults when a model is not present in the OpenRouter cache so non-OpenRouter model identifiers do not crash token-bar rendering.
- Rationale: The feature requires `context_limit` from known model metadata and explicitly calls out graceful degradation for Ollama-like models that do not provide OpenRouter usage objects. A resolver abstraction isolates provider-specific data acquisition from event emission.
- Alternatives considered:
  - Hardcode every model limit in config: rejected because OpenRouter models change frequently and the request explicitly points to the startup metadata cache.
  - Fetch `/api/v1/models` on every completion: rejected because it would add unnecessary latency and external dependency to the hot path.
  - Return no limit for unknown models: rejected because local fallback limits are needed to avoid UI crashes and to keep the visualization stable for non-OpenRouter identifiers.

## Decision 5: Attach token usage to the existing completion and terminal agent-state paths rather than inventing a new event type

- Decision: Extend the shared completion payload used by single-agent runs with `tokens_used` and `context_limit`, and also include those optional fields on orchestration terminal state payloads when an agent stage completes so the canvas can update each node through the existing patch flow.
- Rationale: The current canvas reacts to `agent_state_changed` and related run events, while single-agent history already uses `complete`. Reusing those paths keeps live updates aligned with current store logic and avoids a new event class for one piece of metadata.
- Alternatives considered:
  - Add a separate `token_usage` event type: rejected because it would duplicate sequencing and replay logic already present in the terminal event flow.
  - Only attach usage to `run_complete`: rejected because that would update only the final explainer node in orchestrated runs, not every agent node.
  - Only attach usage to `complete`: rejected because orchestrated stages do not emit a per-stage `complete` event today.

## Decision 6: Render token usage in the canvas node with threshold-based styling and explicit unavailable states

- Decision: Store derived token-usage state on each canvas node model and render a simple fill bar in `AgentCanvasNode` using the existing patch-and-rerender flow. The bar uses neutral, amber, and red thresholds when both values are valid, and a distinct unavailable treatment when usage or limits are missing.
- Rationale: The request explicitly calls for a styled fill bar updated via the same `setNodes(prev => prev.map(...))` pattern used for other node state changes. Keeping the computation in the canvas model preserves replay consistency and testability.
- Alternatives considered:
  - Compute token usage only in the React component: rejected because replay and store tests need a deterministic derived state model.
  - Use CSS-only percentage classes without derived node state: rejected because the width and threshold band depend on validated payload data.
  - Hide the section entirely for missing data: rejected because the user needs to distinguish unavailable telemetry from low usage.

## Implementation Notes

- Relay is OpenRouter-only today, so the initial live usage path is provider-specific, but the context-limit resolver should be designed so local-model defaults can be returned without refactoring the store or canvas layers later.
- For older event rows with null token fields, replay continues unchanged and the frontend renders the unavailable state.
- The most recent completed usage payload for a given agent node becomes the displayed token state during both live viewing and replay.
- Validation should explicitly cover count-only completions with no resolved context limit plus replayed rows whose stored context limit is invalid, so the UI can prove it stays in fallback or capped-critical states without estimating tokens.