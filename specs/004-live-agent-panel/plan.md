# Implementation Plan: Live Agent Panel

**Branch**: `004-live-agent-panel` | **Date**: 2026-03-23 | **Spec**: `/Volumes/xpro/erisristemena/made-by-ai/relay/specs/004-live-agent-panel/spec.md`
**Input**: Feature specification from `/specs/004-live-agent-panel/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Deliver Relay Phase 2 as a live single-agent execution panel layered on top of the existing local workspace: a developer submits a task through the browser, the Go backend selects one specialized agent with a code-owned system prompt and code-owned tool allowlist, streams ordered token, tool, and state events from OpenRouter through the existing WebSocket connection, and persists each run plus its event log in SQLite for later replay. The design uses OpenRouter's OpenAI-compatible API through `sashabaranov/go-openai` with a custom base URL, keeps per-role model assignment in `config.toml` under `[agents]`, stores the OpenRouter API key under `[openrouter]`, reads `project_root` from `config.toml` for repo-scoped read/search tools, and preserves the required handler -> orchestrator -> agents -> tools -> storage boundaries.

## Technical Context

**Language/Version**: Go 1.26 backend; TypeScript (strict) with Next.js 16.2 App Router frontend  
**Primary Dependencies**: Go standard library (`context`, `net/http`, `encoding/json`, `sync`, `time`, `os/exec`, `database/sql`, `errors`, `path/filepath`), Cobra for CLI entrypoints, `nhooyr.io/websocket` for Relay WebSocket transport, `github.com/pelletier/go-toml/v2` for local config, SQLite with sqlc-generated queries and repository migrations, `github.com/sashabaranov/go-openai` for OpenAI-compatible chat completion streaming against OpenRouter, Next.js 16.2, Tailwind CSS, shadcn/ui, React Flow, Framer Motion  
**Storage**: SQLite for sessions, agent runs, and append-only run events; TOML config at `~/.relay/config.toml` for `project_root`, `[openrouter]` credentials, and `[agents]` role-to-model assignments  
**Testing**: `go test ./...` with table-driven unit tests for config parsing, agent construction, prompt/model selection, tool allowlists, tool happy/error paths, gateway streaming normalization, approval gating, and SQLite persistence; Go integration tests for WebSocket protocol ordering, OpenRouter stream bridging, unsupported tool-calling behavior, project-root configuration errors, approval rejection paths, and replayed run history; Vitest plus React Testing Library for the command bar, streaming thought viewer, state badges, tool-event rendering, history review, and settings flows  
**Target Platform**: Local developer workstations with macOS as the first target, browser UI on localhost, and the existing single-binary Relay runtime  
**Project Type**: Relay backend/frontend feature extending the local workspace with live LLM execution and persisted event replay  
**Performance Goals**: First visible token in the UI within 500ms in at least 95% of accepted runs, backend-to-frontend WebSocket event dispatch under 100ms, strict event ordering across token, tool, and state events, and no UI freeze while live output is arriving  
**Constraints**: Dark mode only; WebSocket is the only backend-to-frontend runtime channel; SQLite is the only data store; exactly one active agent run at a time; OpenRouter API key and `project_root` must stay server-side in config; model assignment is configurable but system prompts are fixed in code per concrete agent type; tool permissions are fixed in code and enforced at agent construction time; repo-scoped reads and searches must stay within the configured `project_root`; handler-level approval remains mandatory for file writes and shell commands; prompt and response content plus secrets must not be written to application logs  
**Scale/Scope**: Single-user local workstation, one active Relay process, one live agent run at a time, five built-in agent roles, and hundreds to low thousands of persisted runs and run-event records over time

## Constitution Check

*GATE: Passed before Phase 0 research and re-checked after Phase 1 design.*

- [x] Code quality rules are satisfied: the plan keeps Go code idiomatic, prefers the standard library for orchestration around the new dependency, preserves strict TypeScript, and treats code-owned prompts as product logic rather than runtime configuration.
- [x] Test impact is defined: table-driven Go tests cover config parsing, agent construction, prompt/model selection, tool permission enforcement, tool happy/error paths, gateway streaming normalization, approval gating, and store behavior; integration tests cover WebSocket protocol changes, event ordering, replay, and graceful tool-support failure; frontend component tests cover the live panel states, tool timeline, and history review.
- [x] Architecture remains compliant: handlers accept requests and approval decisions, orchestrator owns run lifecycle and sequencing, concrete agents stay behind the `Agent` interface, tools remain isolated from handlers, and storage remains SQLite-only with WebSocket as the only runtime browser channel.
- [x] UX and governance impact is defined: the design includes visible waiting, streaming, tool-running, completed, approval-blocked, and errored states; explicit empty and recoverable error states; and no relaxation of handler-level approval for write or shell tools.
- [x] Security and performance constraints are covered: API keys and `project_root` remain in `~/.relay/config.toml`, secrets are redacted from persisted events and outbound payloads, repo-scoped tools stay inside the configured root, every run has a `context.Context` cancellation path, append-only event persistence avoids transcript reconstruction ambiguity, and ordered dispatch is explicit across the OpenRouter -> Go -> WebSocket pipeline.

## Project Structure

### Documentation (this feature)

```text
specs/004-live-agent-panel/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── websocket-events.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── relay/
    └── main.go

internal/
├── app/
│   └── server.go
├── agents/
│   ├── agent.go
│   ├── registry.go
│   ├── planner.go
│   ├── coder.go
│   ├── reviewer.go
│   ├── tester.go
│   ├── explainer.go
│   └── openrouter/
│       ├── client.go
│       └── client_test.go
├── config/
│   ├── config.go
│   └── config_test.go
├── handlers/
│   └── ws/
│       ├── protocol.go
│       └── workspace.go
├── orchestrator/
│   └── workspace/
│       ├── service.go
│       ├── service_test.go
│       ├── preferences_test.go
│       ├── runs.go
│       └── history.go
├── storage/
│   └── sqlite/
│       ├── migrations/
│       │   └── 0002_agent_runs.sql
│       ├── queries/
│       │   ├── sessions.sql
│       │   ├── agent_runs.sql
│       │   └── agent_run_events.sql
│       ├── models.go
│       ├── store.go
│       └── store_test.go
└── tools/
    ├── catalog.go
    ├── catalog_test.go
    ├── read_file.go
    ├── search_codebase.go
    ├── write_file.go
    └── run_command.go

tests/
└── integration/
    ├── agent_streaming_test.go
    ├── tool_call_ordering_test.go
    ├── run_history_replay_test.go
    └── websocket_reconnect_test.go

web/
├── src/
│   ├── app/
│   ├── features/
│   │   ├── agent-panel/
│   │   ├── history/
│   │   ├── preferences/
│   │   └── workspace-shell/
│   └── shared/
│       └── lib/
└── package.json
```

**Structure Decision**: Extend the existing workspace feature rather than creating a separate app surface. Sessions remain the parent workspace container introduced in Phase 1, while this phase adds agent runs and run events beneath the active session. The backend gains explicit `internal/agents` and `internal/tools` packages to satisfy the mandated Relay layering, and the frontend adds a dedicated `features/agent-panel` area for the command bar, stream viewer, state badge, and replay UI while reusing the existing history, preferences, and workspace-shell features. Repository-reading tools are bounded by the configured `project_root`; mutating tools remain available only to the allowed roles and still require handler-level approval before execution.

## Complexity Tracking

No constitution violations are required by this design.
