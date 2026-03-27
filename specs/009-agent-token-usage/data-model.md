# Data Model: Agent Token Usage

## Scope

This feature adds typed token-usage persistence to completion events, a startup-backed model context-limit cache, and a derived per-agent token-usage view in the canvas.

## Agent Run Event Extension

Represents the existing `agent_run_events` row with two additional nullable telemetry fields.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `run_id` | text | Existing event log | Required foreign key to `agent_runs` |
| `sequence` | integer | Existing event log | Required and monotonically increasing within a run |
| `event_type` | text | Existing event log | Existing event types remain unchanged |
| `role` | text | Existing event log | Required |
| `model` | text | Existing event log | Required |
| `payload_json` | text | Existing event log | Stores the event payload including token fields when present |
| `tokens_used` | integer nullable | Provider usage extraction | Null when provider usage is unavailable or not authoritative |
| `context_limit` | integer nullable | Model-limit resolver | Null when no valid limit can be resolved |
| `created_at` | text | Existing event log | Required RFC3339 timestamp |

### Rules

- `tokens_used` and `context_limit` are persisted only for terminal events that carry completion usage.
- Older rows remain valid with both new columns null.
- Replayed payloads must merge the typed values and JSON payload without changing the stored event sequence.
- Negative or zero `context_limit` values are treated as invalid and must not drive percentage-based rendering.

## Completion Usage Snapshot

Represents the normalized token telemetry captured from a single provider completion.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `finishReason` | text | Provider final chunk | Required terminal reason |
| `tokensUsed` | integer | Provider `usage.total_tokens` | Must be greater than zero to be treated as authoritative for OpenRouter completions |
| `contextLimit` | integer nullable | Model-limit resolver | May be null when the model limit cannot be resolved |
| `provider` | enum-like text | Runtime context | `openrouter` for current live path; fallback-capable for future local models |
| `usageAvailable` | boolean | Derived | False when provider omitted usage or returned an unusable value |

### Rules

- OpenRouter usage is read only from the final streaming chunk.
- If usage is unavailable, `tokensUsed` is treated as absent for persistence and visualization, not estimated.
- For local or non-OpenRouter model identifiers, `contextLimit` may still be resolved through fallback configuration even when `tokensUsed` is unavailable.

## Model Context Limit Cache

Represents the startup-loaded and TTL-refreshed in-memory registry used to resolve model limits.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `modelName` | text | OpenRouter metadata or local fallback key | Cache lookup key |
| `contextLimit` | integer | OpenRouter `/api/v1/models` metadata or local fallback map | Must be positive to be usable |
| `source` | enum | Resolver | `openrouter_metadata`, `local_config`, or `hardcoded_default` |
| `loadedAt` | timestamp | Cache refresh | Required |
| `expiresAt` | timestamp | TTL policy | Required |

### Rules

- Startup should attempt to populate the cache before the first run when OpenRouter is configured, but cache refresh failure must not prevent Relay from starting.
- Cache refresh operations must run with `context.Context` cancellation support.
- When a model is not found in OpenRouter metadata, the resolver may return a local fallback value if configured or known.

## Agent Token Usage State

Represents the most recent token-usage visualization state for a canvas node in a specific run.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `runId` | text | Run context | Required |
| `agentId` | text | Event payload | Required for orchestration nodes |
| `tokensUsed` | integer nullable | Completion usage snapshot | Null when unavailable |
| `contextLimit` | integer nullable | Completion usage snapshot | Null when unavailable or invalid |
| `fillRatio` | decimal nullable | Derived | `min(tokensUsed / contextLimit, 1)` when both values are valid |
| `riskBand` | enum | Derived | `neutral`, `warning`, `critical`, or `unavailable` |
| `displayLabel` | text | Derived | Plain-language copy for the node and detail view |
| `updatedAt` | timestamp | Event occurrence time | Required when usage state changes |

### Rules

- The displayed usage state for a node is replaced by the most recent terminal event carrying valid usage telemetry for that node.
- `riskBand` becomes `warning` and `critical` based on product-defined thresholds near the context limit.
- If `tokensUsed` exceeds `contextLimit`, `fillRatio` is capped at `1` and `riskBand` becomes `critical`.
- If `tokensUsed` is unavailable, `riskBand` is `unavailable` even when `contextLimit` exists.

## Replay Hydration View

Represents the reconstructed event payload emitted during `OpenRun` replay.

| Field | Type | Source | Rules |
|-------|------|--------|-------|
| `eventType` | text | Stored row | Existing event type |
| `payload` | json object | Stored `payload_json` plus typed columns | Replay payload sent over WebSocket |
| `sequence` | integer | Stored row | Required |
| `replay` | boolean | Replay flow | Always `true` during run history playback |

### Rules

- Replay must preserve old payloads that do not have token columns.
- When typed token columns are present, replay hydration must ensure the emitted payload includes `tokens_used` and `context_limit` even if the original payload JSON did not.
- Replay ordering remains sequence-based and unchanged by the new fields.