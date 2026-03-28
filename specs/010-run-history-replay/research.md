# Research: Run History and Replay

## Decision 1: Audit event completeness before building the scheduler, and treat missing historical events as a hard product gap rather than a replay concern

- Decision: Start implementation with an upstream event audit across the single-agent emitters in `internal/orchestrator/workspace/runs.go`, orchestration emitters in `internal/orchestrator/workspace/orchestration.go`, and approval lifecycle emission in `internal/orchestrator/workspace/service.go`. Define a required replay event matrix and add tests that fail when a terminal or state-bearing step is not persisted.
- Rationale: The user called out the core risk correctly: a deterministic replay engine can only re-emit what was stored. The current `OpenRun` path already proves stored-event replay works, but it also exposes that replay fidelity depends entirely on the completeness of `agent_run_events` plus persisted approval data.
- Alternatives considered:
  - Build the scheduler first and patch missing events later: rejected because it would hide root-cause fidelity gaps behind replay bugs.
  - Infer missing replay states from final run status or current canvas logic: rejected because the request explicitly requires deterministic replay from stored history only.
  - Read current repository files to fill historical gaps: rejected because replay and diff review must remain historical and read-only.

## Decision 2: Keep replay backend-owned with a time-based scheduler that emits existing event types over WebSocket

- Decision: Extend `OpenRun` into a replay-session bootstrap that preloads the selected run's stored events, computes relative playback offsets from event timestamps, and hands that timeline to a backend-owned replay scheduler. The scheduler emits the same stored event types already consumed by the workspace store, plus a small replay-state side channel for play, pause, seek position, and speed.
- Rationale: The user explicitly defined the replay engine as a Go time-based event scheduler that reads SQLite history and re-emits over WebSocket. Keeping timing logic on the backend also preserves Relay's rule that WebSocket is the only backend/frontend runtime channel and avoids duplicating event-order logic in the browser.
- Alternatives considered:
  - Client-side replay timers over the stored event list: rejected because it conflicts with the requested backend-owned scheduler and would split ordering logic across two runtimes.
  - Introduce a new replay-only event format: rejected because the existing event shapes already drive the store and canvas correctly.
  - Poll for replay state instead of streaming: rejected because polling violates the repo's architecture rules and would degrade responsiveness.

## Decision 3: Derive playback timing from persisted event timestamps and clamp pathological gaps rather than trusting raw wall-clock spacing blindly

- Decision: Use the event payload `occurred_at` timestamp when present, fall back to `created_at`, and normalize to monotonic offsets from the first replayable event. Preserve order strictly by sequence, but clamp unusually large silent gaps to a product-defined ceiling so replay remains usable while still reflecting recorded timing.
- Rationale: Stored event sequences already define authoritative order, while timestamps define the time axis needed for the scrubber and scheduler. Some events may have identical or irregular timestamps, so sequence must remain the primary ordering key and timestamps the playback offset source.
- Alternatives considered:
  - Use sequence numbers as equal-duration frames: rejected because the request explicitly calls for time-based playback driven by event timestamps.
  - Use only `created_at` from SQLite rows: rejected because event payload timestamps are closer to the original orchestration moment and should win when valid.
  - Preserve full real-time delays with no clamping: rejected because long idle gaps would make replay unusable.

## Decision 4: Make seek fast by rebuilding from precomputed in-memory checkpoints, not by persisting full replay snapshots

- Decision: When a run is opened for replay, load all its events once and build lightweight checkpoints in memory at fixed time or event intervals. Seeking rehydrates the nearest checkpoint and reapplies only the remaining suffix events to the target timestamp before playback resumes.
- Rationale: The user requires seek to replay all events up to the target timestamp while still feeling instant for sessions under 10 minutes. In-memory checkpoints satisfy both constraints without adding heavy persisted snapshot storage or violating the rule that replay reads from the event log.
- Alternatives considered:
  - Re-run the full event stream from sequence 1 on every seek: rejected because it will not feel instant even for moderate runs.
  - Persist full canvas snapshots in SQLite: rejected because it adds large duplicate state blobs and complicates schema evolution.
  - Allow approximate seek that skips intermediate reducer application: rejected because it breaks deterministic state reconstruction.

## Decision 5: Use SQLite FTS5 for keyword search across run titles, goals, summaries, replay-safe transcript text, and touched file names through a maintained run-history document plus normalized change records

- Decision: Add a persisted run-history document table keyed by `run_id` and an FTS5 virtual table indexing generated title, run goal text, recorded summary text, replay-safe transcript text, and touched file paths. Normalize historical file modifications into run-change records derived from approval diffs so file-name filtering and diff review query the same source.
- Rationale: The feature requires keyword search to match historical metadata plus replayable transcript content, while file-touched filters must still query the same persisted history source. The existing `agent_runs` summary query is not enough because it only returns a truncated task preview and tool-activity boolean. A dedicated history document avoids reparsing every event row on each search.
- Alternatives considered:
  - Search raw `agent_run_events.payload_json` with `LIKE`: rejected because it is slow, unindexed, and too brittle for transcript matching.
  - Search only `agent_runs.task_text`: rejected because it does not cover summaries, replayable transcript text, or touched files.
  - Build an in-memory index at startup: rejected because search must survive restart and remain SQLite-only.

## Decision 6: Source historical before/after diffs from stored approval data and normalize them into run-change records

- Decision: Extract diff previews from persisted approval requests and approval state changes into normalized run-change records keyed by run and file path, preserving original content, proposed content, final approval outcome, and occurrence time. Replay, diff review, and markdown export all read from these normalized records.
- Rationale: Relay already persists safe diff previews for write approvals, including `target_path`, `original_content`, and `proposed_content`. Normalizing those records once avoids reparsing approval JSON for every history screen and satisfies the rule that historical review must not read current repository files from disk.
- Alternatives considered:
  - Read repository files on demand for the "after" view: rejected because it would show current state, not historical state.
  - Keep diff review entirely inside approval JSON blobs: rejected because query and export paths would become repetitive and slow.
  - Restrict diff review to pending approvals only: rejected because the feature needs a review surface for completed historical runs.

## Decision 7: Export markdown on the backend to `~/.relay/exports/` with deterministic sections sourced from persisted history only and treat the request as a direct user action at the handler boundary

- Decision: Add a backend export service that generates markdown from stored run metadata, replayable events, and normalized change records, then writes the file under `~/.relay/exports/` by default using a deterministic filename derived from run date and generated title. The handler accepts export only from an explicit developer action on the workspace client and rejects any equivalent agent-driven or replay-driven path.
- Rationale: The request explicitly fixes the export location and requires the report to be a durable markdown artifact. Backend generation keeps file-system writes inside governed server boundaries, and the constitution requires that the approval distinction between direct developer export and agent-triggered file writes be enforced server-side rather than inferred from UI state.
- Alternatives considered:
  - Generate markdown in the browser and download it client-side: rejected because export location is explicitly local disk under Relay's home directory and writes must stay server-side.
  - Export from transient replay viewport state: rejected because the report must always represent the full stored run.
  - Treat all export-capable code paths as implicitly user-approved: rejected because approval enforcement must remain explicit at the handler boundary.
  - Add HTML or PDF export in the same phase: rejected because only markdown is in scope.

## Decision 8: Re-emit token usage exactly as stored so the existing canvas fill bar animates during playback without new replay-specific logic

- Decision: Treat `tokens_used` and `context_limit` as standard replay payload fields during scheduled playback and seek reconstruction. Replay status messages remain separate from content events so the canvas keeps using the existing token-usage patch flow.
- Rationale: Token usage is already stored in SQLite from the previous feature and `OpenRun` already merges those typed columns into replay payloads. Preserving that behavior during scheduled playback lets the node fill bar animate correctly with minimal new frontend logic.
- Alternatives considered:
  - Add a replay-only token message: rejected because token usage already belongs to terminal agent events.
  - Recompute token usage from transcripts: rejected because the stored values are authoritative and replay must stay deterministic.

## Implementation Notes

- The current `OpenRun` implementation in `history.go` emits all stored events immediately. That path should be retained as the event-loading foundation, then wrapped by a replay-session controller instead of replaced wholesale.
- The event audit must explicitly cover `state_change`, `token`, `tool_call`, `tool_result`, `complete`, `agent_spawned`, `agent_state_changed`, `task_assigned`, `handoff_start`, `handoff_complete`, `agent_error`, `run_complete`, `run_error`, and approval events because those are the pieces the current workspace store knows how to rehydrate.
- Search should combine keyword, date range, and touched-file filters in one SQLite query so replay remains responsive while the history list updates, with replay-safe transcript text precomputed into the search document instead of scanning raw event JSON on demand.
- Reconnect behavior should restore both the selected run and the replay controller state rather than restarting from zero unless the run or session changed.