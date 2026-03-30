# Research: Repository File Tree Sidebar

## Decision 1: Put File Tree and Historical Replay in a shared top-level right rail instead of nesting File Tree inside Historical Replay

- Decision: Keep run history in the existing left panel, use the right-side detail rail for File Tree during live runs, and expose top-level Historical Replay and File Tree tabs in that same rail only when the developer reopens a saved run.
- Rationale: Historical Replay and File Tree are separate panels with different purposes. File Tree matters during live work, while Historical Replay matters only for reopened saved runs. Making them peer panels in the same right rail preserves graph adjacency without implying that File Tree belongs inside Historical Replay.
- Alternatives considered:
  - Add a second always-visible sidebar: rejected because it would compress the canvas and create unnecessary layout conflict.
  - Replace run history with the file tree entirely: rejected because replay discovery and saved-run browsing remain important existing workflows.
  - Keep the tree in the left history panel: rejected because it forces the user away from the graph when inspecting file activity.
  - Nest the tree inside the Historical Replay panel: rejected because the two panels are unrelated and have different visibility rules.
  - Put the tree inside the node detail panel: rejected because the tree is workspace context, not agent-detail content.

## Decision 2: Build the repository directory structure once at repository-connect time using existing `go-git` support, while excluding `.gitignore`d paths, and keep it in memory as a flat path list

- Decision: When Relay validates a connected repository, iterate the worktree recursively once with `go-git`, apply the repository's `.gitignore` rules to exclude ignored paths, normalize the remaining results into a flat list of repository-relative paths, and keep that list in the workspace service as the source for tree hydration.
- Rationale: The user explicitly wants directory structure to be built at repo-connect time and stored in memory rather than SQLite, but also does not want wasteful directories such as `node_modules` to delay the UI. A flat list is still simpler to cache and lets the client materialize a hierarchical tree deterministically, while ignore filtering removes low-value work.
- Alternatives considered:
  - Persist the full directory structure in SQLite: rejected because the user explicitly ruled that out and the structure can be rebuilt from the repo root.
  - Scan the filesystem on every sidebar open: rejected because it would be slower and would add repeated I/O for the same connected repository.
  - Send pre-nested JSON from the backend: rejected because a flat list is easier to cache, test, and reuse for expansion logic.

## Decision 3: Respect the large-repository risk by limiting initial render depth to one nested level and expanding deeper folders on demand

- Decision: Materialize the complete filtered flat path list once, but have the frontend render only top-level entries and one nested level initially. When a developer expands a folder, the client derives deeper visible descendants from the already-loaded flat list instead of requesting another backend scan.
- Rationale: The main performance risk in the request is not repository traversal alone, but rendering a very large nested tree immediately in React. Rendering a shallow initial slice while keeping the full flat list in memory gives predictable startup cost without discarding deeper structure.
- Alternatives considered:
  - Render the full nested tree immediately: rejected because repositories with thousands of files can cause unnecessary initial layout and paint work.
  - Paginate directories from the backend: rejected because it adds extra protocol complexity when the full structure is already loaded in memory.
  - Virtualize the entire tree immediately: deferred because initial depth limiting is simpler and may be sufficient without a new dependency.

## Decision 4: Record touched files in SQLite at tool execution and proposal time, not approval time, using a deduplicated `touched_files` table

- Decision: Add a `touched_files` table with logical key `(run_id, agent_id, file_path, touch_type)` where `touch_type` is `read` or `proposed`. Insert a `read` touch after a successful `read_file` execution and insert a `proposed` touch when a `write_file` proposal is created for approval, before any approval decision is made.
- Rationale: The user's timing requirement is correct: waiting until approval would make the sidebar lag behind the agent's actual exploration and would under-report proposed work that is still pending review. SQLite persistence also makes reconnect recovery and selected-run restoration straightforward.
- Alternatives considered:
  - Record touches only in memory: rejected because reconnect and run reopening would lose the touched set.
  - Record writes only after approval: rejected because the sidebar needs to answer what the agent looked at or tried to change, not just what eventually got applied.
  - Store touches only in existing run events: rejected because filtering and reconnect hydration would require reparsing event payloads repeatedly.

## Decision 5: Emit `file_touched` from the tool execution path immediately after the touch is known, not from the agent lifecycle layer

- Decision: Publish a new `file_touched` stream event from the workspace tool-execution path as soon as a read succeeds or a write proposal is accepted into the approval flow. The event is created close to `tool_executor.go`, where run and agent context are already available, rather than from higher-level agent lifecycle transitions.
- Rationale: The user called out the key UX risk: if emission waits for the agent layer to process the result, the canvas may advance before the sidebar catches up. The tool execution path is the earliest layer that both knows the concrete file path and still has the correct run and agent identity.
- Alternatives considered:
  - Emit from agent state transitions: rejected because it delays sidebar updates and couples file touches to unrelated agent progression.
  - Emit inside `internal/tools` directly: rejected because the tools package does not own workspace streaming or run identity.
  - Batch touches and emit periodically: rejected because the feature needs real-time incremental feedback.

## Decision 6: Replace the requested GET tree endpoint with a WebSocket request/result pair to preserve Relay's transport rules

- Decision: Expose initial tree hydration through a new workspace protocol message pair such as `repository.tree.request` and `repository.tree.result`, and keep incremental updates on the same WebSocket channel with `file_touched`.
- Rationale: Relay's constitution is explicit that WebSocket is the only supported communication channel between the Go backend and React frontend. A GET endpoint for tree hydration would introduce an unnecessary second transport even though the workspace already supports request/response interactions over the socket.
- Alternatives considered:
  - Add `GET /api/repo/tree` exactly as requested: rejected because it violates the project constitution without providing a capability the socket cannot already support.
  - Put the full tree directly into every workspace bootstrap snapshot: rejected because it would bloat unrelated bootstrap messages and reload more often than needed.
  - Delay all tree hydration until the first live `file_touched` event: rejected because the full browsable tree must exist before any file is touched.

## Decision 7: Keep per-agent file lists fully client-side by filtering the touched-file store with the selected canvas node

- Decision: The client store maintains a current-run touched-file collection keyed by run and agent. Selecting an agent node applies a client-side filter over that store and the in-memory tree model, with no separate backend query for agent-specific paths.
- Rationale: The user explicitly asked for client-side derivation. Once the touched-file set is hydrated initially and updated incrementally, filtering by selected `agent_id` is deterministic and avoids extra protocol chatter.
- Alternatives considered:
  - Add a backend per-agent query endpoint or message: rejected because it duplicates state the client already has.
  - Recompute agent-specific lists from raw run events in the browser: rejected because a dedicated touched-file store is simpler and more direct.
  - Filter the visible tree by role instead of agent identity: rejected because the request is explicitly about what one specific agent touched.

## Implementation Notes

- `WorkspaceShell.tsx` already reserves a graph-adjacent right rail when run context is visible; that makes a top-level right-rail composition practical without redesigning the full workspace layout.
- `internal/orchestrator/workspace/repository_browser.go` already handles repository-path browsing and validation. The new tree cache should sit alongside that functionality rather than introducing another repository service.
- `tool_executor.go` already has access to run and agent context through `runExecutionContextFromContext(ctx)`, which is the right place to attach touched-file recording and `file_touched` emission without leaking workspace concerns into the agent package.
- Reconnect recovery should bulk-load touched records for the active run so the sidebar reflects already-known touches immediately after bootstrap, then rely on live `file_touched` events for subsequent updates.