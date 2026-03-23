# Research: Live Agent Panel

## Decision 1: Use `sashabaranov/go-openai` with a custom `BaseURL` pointed at OpenRouter's OpenAI-compatible API

- Decision: Configure the Go client with `openai.DefaultConfig(apiKey)`, override `config.BaseURL` to `https://openrouter.ai/api/v1`, and use chat completions plus `CreateChatCompletionStream` for live streaming.
- Rationale: The upstream `go-openai` client explicitly supports custom `BaseURL` values through `NewClientWithConfig`, and OpenRouter documents a normalized OpenAI-compatible `/api/v1/chat/completions` endpoint with SSE streaming and normalized `delta.content`, `delta.tool_calls`, `finish_reason`, and `usage` fields. This gives Relay a smaller integration surface than hand-rolling request signing and SSE decoding from scratch.
- Alternatives considered:
  - Raw `net/http` plus a bespoke SSE parser: rejected because it duplicates transport logic and increases maintenance cost without adding product value.
  - The official OpenAI Go SDK: rejected for this phase because the repo already requested `sashabaranov/go-openai` and the compatibility surface needed here is satisfied.
  - OpenRouter's provider-specific SDKs: rejected because the product requirement is one gateway capable of routing multiple cloud models behind a single API shape.

## Decision 2: Keep model assignment configurable, but keep prompts and tool permissions fixed in code per concrete agent type

- Decision: Add `[agents]` model assignment fields to `config.toml` for `planner`, `coder`, `reviewer`, `tester`, and `explainer`, while storing each role's `systemPrompt`, resolved `model`, and fixed tool allowlist directly in the corresponding Go agent struct.
- Rationale: The spec requires role behavior to remain a first-class product feature rather than a user-editable prompt playground. Putting model selection in config supports per-role provider choice and future tuning, while keeping prompts and allowed tools in code ensures the persona, output rules, and safety boundaries remain versioned with the product. Making the prompt a first-class part of the agent struct also makes role-specific prompt adherence testable per model.
- Alternatives considered:
  - Store prompts in `config.toml`: rejected because it would make product behavior mutable, harder to test, and easier to misconfigure.
  - Make tool permissions configurable at runtime: rejected because it weakens the required safety boundary and complicates auditability.
  - Hardcode model names with no config override: rejected because the user explicitly requested per-role model assignment in config.

## Decision 3: Enforce tool permissions by construction, not by per-call runtime checks

- Decision: Build a tool catalog in `internal/tools`, then have each concrete agent constructor receive only the subset of tool implementations that its role is allowed to use.
- Rationale: Constructor-time composition matches the requirement that Planner and Reviewer remain read-only, Coder and Tester can write or run commands, and Explainer gets `read_file` only. This reduces the chance of accidentally exposing a broader tool surface later, because the agent never sees disallowed tools in the first place.
- Alternatives considered:
  - A runtime `if allowed` check before every tool invocation: rejected because it is easier to bypass accidentally and obscures the true role surface.
  - One shared tool registry injected into all agents: rejected because it violates least privilege.
  - UI-side permission controls: rejected because permission enforcement must stay server-side.

## Decision 4: Use an append-only ordered event journal for the streaming bridge

- Decision: Normalize every live run update into a single ordered event stream with a monotonically increasing `sequence` number per run, and use that same event representation for WebSocket fan-out and SQLite persistence.
- Rationale: The feature has three async boundaries: OpenRouter SSE, Go orchestration, and WebSocket delivery to React state. A single ordered event journal avoids token loss and out-of-order rendering when token, tool-call, tool-result, and state-change events are interleaved. The same sequence numbers also make replay deterministic for saved runs.
- Alternatives considered:
  - Persist only a final transcript string: rejected because inline tool visibility and replay ordering would be lossy.
  - Broadcast raw provider chunks directly to the browser: rejected because provider chunks do not align with the UI's required event types and would leak provider-specific behavior into the frontend.
  - Maintain separate channels for text and tools: rejected because the feature explicitly requires correct interleaving.

## Decision 5: Persist runs separately from sessions and store run events as append-only rows

- Decision: Extend the Phase 1 session model so each session can own many agent runs, and create a dedicated `agent_runs` table plus an `agent_run_events` table keyed by run ID and sequence.
- Rationale: Sessions already exist as the durable workspace container. Separate run and event tables let Relay preserve partial output for errored runs, reload history efficiently, and replay an exact event timeline without recontacting the model provider.
- Alternatives considered:
  - One run per session: rejected because it would collapse two different concepts and make ongoing workspace use awkward.
  - One JSON blob per run containing the whole timeline: rejected because appending, querying, and replaying individual events becomes harder and more error-prone.
  - Store history only in browser state: rejected because the feature requires review across Relay restarts.

## Decision 6: Treat model tool-calling support as a validated compatibility matrix with graceful runtime failure

- Decision: Validate each default model's tool-calling behavior during Phase 2 development and represent any unsupported tool invocation as a terminal run error with partial history preserved and a human-readable message.
- Rationale: OpenRouter documents that tools are normalized across providers, but support is model-dependent. Some models may ignore tool definitions, emit incompatible deltas, or fail mid-stream. Relay must not hide that mismatch; it should surface it clearly while keeping the rest of the run record intact.
- Alternatives considered:
  - Assume all configured models support tools identically: rejected because the known risk explicitly says they do not.
  - Special-case each provider in the frontend: rejected because compatibility handling belongs in the backend gateway and orchestrator.
  - Disable tools globally for all models: rejected because tool visibility is a core value proposition of this phase.

## Decision 7: Preserve handler-level approval as the execution gate for mutating tools

- Decision: Keep `write_file` and `run_command` inside the tool catalog for the allowed roles, but require the handler layer to approve or reject each mutating tool request before the tool executes.
- Rationale: The constitution requires approval enforcement at the handler level, not only as a UI affordance or tool-layer helper. This preserves the architecture boundary while still allowing the orchestrator and agents to model mutating tool intent.
- Alternatives considered:
  - Enforce approval only inside the tool implementation: rejected because it violates the constitution's server-handler requirement and makes UX approval state harder to represent.
  - Remove mutating tools entirely from this phase: rejected because the phase definition still includes role-level tool permissions for Coder and Tester.
  - Enforce approval only in the frontend: rejected because the server must remain authoritative.
