# Research: Live Agent Orchestration

## Decision 1: Use a coordinator goroutine plus one cancellable goroutine per agent execution

- Decision: Represent each orchestration run as one coordinator goroutine in `internal/orchestrator/workspace` that owns DAG progress and cancellation, plus one goroutine per spawned agent that calls `agent.Run(ctx, task)` directly and streams visible output back through a run-scoped fan-out bridge.
- Rationale: The feature requires strict dependency ordering, concurrent Coder and Tester execution, and explicit run-level halts. A coordinator-owned DAG keeps those rules centralized while per-agent goroutines preserve streaming isolation and clean cancellation. Making the goroutine itself the runner avoids inventing an extra runtime abstraction the feature does not need.
- Alternatives considered:
  - Run all agents serially in one goroutine: rejected because it cannot satisfy concurrent Coder and Tester execution.
  - Let each agent spawn the next agent directly: rejected because it smears orchestration rules across implementations and violates observability.
  - Introduce an external queue or worker pool: rejected because the constitution forbids adding secondary runtime infrastructure and the scope is single-user local execution.

## Decision 2: Replace profile-selected orchestration with concrete built-in agent implementations behind the existing package boundary

- Decision: Implement Planner, Coder, Reviewer, Tester, and Explainer as concrete structs that satisfy the `Agent` interface, each carrying its own fixed system prompt, resolved model, and declared tool permissions.
- Rationale: The current registry selects a `Profile` and creates one generic execution path. The orchestration feature needs role-scoped lifecycle visibility, explicit permissions, and stable identity per agent instance. Concrete structs make those differences testable without breaking the architectural rule that concrete agent types stay inside the agent package, and using the `Agent` interface directly avoids confusion about a second runner abstraction.
- Alternatives considered:
  - Continue using a single generic execution path with role profiles only: rejected because orchestration-specific state and identity would remain implicit and harder to reason about.
  - Move orchestration logic into the frontend and keep a single backend agent: rejected because the DAG and run state must remain authoritative on the server.
  - Make prompts user-editable in config: rejected because product behavior and safety boundaries must remain code-defined.

## Decision 3: Extend the WebSocket protocol with orchestration events while keeping transcript delivery ordered per agent

- Decision: Add orchestration-specific events `agent_spawned`, `agent_state_changed`, `task_assigned`, `handoff_start`, `handoff_complete`, `agent_error`, `run_complete`, and `run_error`, and continue delivering transcript chunks as ordered per-agent stream events keyed by `agent_id`.
- Rationale: The canvas needs lifecycle and graph-handoff visibility that the existing single-agent `state_change` and `complete` events do not express. Including `agent_id` on every orchestration-mode event lets the frontend route updates to the correct node even while multiple agents are active.
- Alternatives considered:
  - Reuse only the existing single-agent event names: rejected because they cannot represent spawn, handoff, and per-agent error semantics clearly enough.
  - Collapse all orchestration updates into one generic JSON event: rejected because it weakens type safety and complicates frontend routing.
  - Maintain separate sockets per agent: rejected because WebSocket remains a single application channel and additional channels add failure complexity.

## Decision 4: Persist orchestration state as one run plus many agent executions plus an append-only ordered event journal

- Decision: Store one orchestration run as the top-level record, one child record per spawned agent execution, and an append-only event journal with run order plus agent identity for replay and reconnect.
- Rationale: The feature must reopen completed runs, preserve partial failures, and reattach active runs without duplicating nodes. Distinguishing run-level and agent-level records gives storage the same shape as the product behavior.
- Alternatives considered:
  - Store only a flat run event stream with no agent execution records: rejected because queries for canvas state, side-panel selection, and final per-agent summaries become harder and more expensive.
  - Store a single JSON blob per run: rejected because append, replay, and validation of concurrent ordering are more error-prone.
  - Persist only final transcripts: rejected because handoffs, spawn timing, and failures would be lost.

## Decision 5: Keep orchestration mode prompt-only and exclude tool execution entirely for this feature

- Decision: The orchestration mode will not expose repository-reading tools, `write_file`, or `run_command`, and its persisted transcripts will exclude `tool_call` and `tool_result` activity.
- Rationale: The spec explicitly narrows this feature to prompt-only work even though the broader live-run foundation already supports tools. Keeping that separation explicit prevents accidental inheritance of repo access or approval flows into multi-agent orchestration.
- Alternatives considered:
  - Reuse the full single-agent tool pipeline in orchestration mode: rejected because it violates the feature scope and expands safety risk.
  - Hide tool events in the frontend while still allowing tool execution: rejected because the behavior would still exist server-side and break the spec.
  - Build two partially overlapping orchestration implementations: rejected because mode separation should be behavioral, not a second backend architecture.

## Decision 6: Reuse saved provider access and run-history foundations from the live-run feature

- Decision: Continue using the existing provider-access configuration and history surfaces established in the live-run feature as prerequisites for orchestration mode.
- Rationale: The new feature connects the canvas to a real backend; it does not need a separate credentials or history subsystem. Reuse reduces user confusion and avoids duplicating storage or preferences logic.
- Alternatives considered:
  - Create orchestration-only credentials or history tables: rejected because the product already has those surfaces.
  - Make orchestration mode ephemeral only: rejected because replay is part of the spec.

## Decision 7: Treat canvas updates as patch operations keyed by `agent_id`, with dagre only on spawn

- Decision: Route live events into React Flow using explicit append and patch operations: `setNodes(prev => [...prev, newNode])` only for `agent_spawned`, `setNodes(prev => prev.map(...))` for state and transcript-driven node metadata updates, and rerun dagre only after a node is added.
- Rationale: The current isolated canvas uses a reducer-backed document model. Under live concurrent updates, recalculating graph layout or rebuilding node arrays on every event risks reintroducing the disappearing-node regression the prior phase already guarded against. Spawn-only layout keeps the graph stable while agents stream.
- Alternatives considered:
  - Recompute nodes and dagre layout on every event: rejected because it is the most likely path to flicker and node disappearance under load.
  - Skip dagre entirely after the first node: rejected because the graph still needs readable placement as new agents spawn.
  - Store node state only outside React Flow and regenerate the full graph on render: rejected because it obscures incremental update behavior and increases reconciliation churn.

## Decision 8: Use one stream bridge per active orchestration run with dedicated per-agent channels feeding a single ordered dispatcher

- Decision: Give each active agent goroutine its own buffered channel for transcript and lifecycle output, then normalize those events through a run-scoped dispatcher that stamps sequence metadata before WebSocket fan-out and persistence.
- Rationale: The user requirement explicitly calls for concurrent streaming across agents. Dedicated channels avoid one noisy agent blocking another, while a dispatcher preserves a canonical stored order for replay and reconnect.
- Alternatives considered:
  - One shared untyped channel for all agents: rejected because agent attribution and backpressure handling become brittle.
  - Let each agent write directly to WebSocket and storage: rejected because ordering, cancellation, and replay invariants would fragment across goroutines.
  - Serialize all transcript output through the coordinator goroutine only: rejected because it would create unnecessary contention during parallel streaming.

## Decision 9: Treat agent-scoped failures and run-level halts as separate terminal signals

- Decision: Preserve `agent_error` as a node-scoped terminal event that can leave the overall run recoverable, and emit `run_error` only when the DAG can no longer continue or a raw stage execution failure stops the orchestration.
- Rationale: The product requirement is to keep partial failures inspectable without collapsing the full graph. Splitting these signals lets the frontend preserve a failed node's transcript while still distinguishing a planner halt or unrecoverable stage failure from a contained agent problem.
- Alternatives considered:
  - Treat every agent failure as a run halt: rejected because it would hide useful partial output and violate the user-facing failure model.
  - Keep only node-level failures with no run-level terminal event: rejected because the canvas and history surfaces need one authoritative halted-run signal.

## Decision 10: Rehydrate active orchestration runs through the existing bootstrap plus open-run flow

- Decision: Reuse `workspace.bootstrap`, `session.opened`, and `preferences.saved` snapshots to surface `active_run_id`, then issue one `agent.run.open` request per newly observed active run so replay and live reattachment happen through the existing run-open path.
- Rationale: This keeps reconnect behavior inside the same replay mechanism used for saved runs, avoids a second hydration protocol, and prevents duplicate node creation because the replay path already normalizes events by stable `agent_id`.
- Alternatives considered:
  - Add a separate active-run hydration channel: rejected because it would duplicate replay logic and create two sources of truth.
  - Push raw active-run state inside every bootstrap payload with no follow-up open call: rejected because it would bloat bootstrap payloads and bypass ordered event replay.