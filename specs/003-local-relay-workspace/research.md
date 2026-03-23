# Research: Local Relay Workspace

## Decision 1: Use Next.js static export for production assets and copy the output into a stable embed directory

- Decision: Configure the frontend with `output: 'export'`, run `next build`, and copy the generated `web/out/` assets into `internal/frontend/embed/` before `go build`.
- Rationale: Current Next.js documentation confirms that static export writes deployable HTML, CSS, and JavaScript assets into an `out` directory, which can be served by any HTTP server. This aligns directly with Go `go:embed`, which requires a compile-time-stable filesystem path and does not support serving a live Node runtime from the compiled binary.
- Alternatives considered:
  - Serve `.next` build artifacts directly: rejected because the standard `.next` output is not intended to be served generically by another HTTP server.
  - Use Next.js standalone server output: rejected because it would require shipping or spawning a Node.js runtime, violating the single-binary production requirement.
  - Rebuild the Go binary after every frontend change in development: rejected because it makes frontend iteration too slow and undermines the required dev workflow.

## Decision 2: Use Go-owned route ordering with a development reverse proxy mounted last against the discovered frontend port

- Decision: Register Go handlers for `/ws` and `/api/healthz` first, then mount a `httputil.ReverseProxy` catch-all to the actual Next.js dev-server port chosen at startup, preferring `3000` but falling back automatically when `3000` is unavailable.
- Rationale: This preserves Relay's ownership of the WebSocket channel and any diagnostic API routes while still enabling Turbopack-based frontend iteration. It also addresses the known risk that the proxy must not forward Relay's own WebSocket path or Go API routes, while removing a brittle assumption that the frontend dev server always owns `3000`.
- Alternatives considered:
  - Proxy every non-root route to Next.js: rejected because it risks swallowing `/ws` and `/api/` traffic depending on registration order.
  - Disable the dev proxy and rely on embedded assets during development: rejected because `go:embed` is compile-time only and would force a rebuild for every UI change.
  - Run Next.js behind Go in production as well: rejected because it adds an external runtime and breaks the single-binary requirement.

## Decision 3: Split local persistence between TOML config for preferences/secrets and SQLite for sessions/state

- Decision: Persist user-editable settings and credentials in `~/.relay/config.toml`, and persist session metadata plus minimal workspace state in SQLite at `~/.relay/relay.db` with goose migrations and sqlc-generated queries.
- Rationale: The spec explicitly requires TOML config and a SQLite database file at those locations. This split keeps secrets out of the frontend and out of database rows that may later be queried or exported, while SQLite remains the right fit for session history, ordering, and recovery across restarts.
- Alternatives considered:
  - Store all settings and sessions in SQLite: rejected because it conflicts with the explicit config-file requirement and blurs secret handling.
  - Store sessions in JSON files: rejected because it complicates ordering, migration, and reliable filtering compared with SQLite.
  - Store credentials in the browser: rejected because secrets must remain local to the Go runtime and never be sent to the frontend.

## Decision 4: Use a WebSocket-first bootstrap protocol for all workspace state shown in the browser

- Decision: After loading the static shell, the browser connects to `/ws` and sends a bootstrap request. The server responds with the complete initial workspace snapshot, session list, active session, and preferences that are safe for the frontend, then streams subsequent session and preference state changes over the same socket.
- Rationale: Relay's constitution requires WebSocket as the only backend-to-frontend communication channel for runtime state. A bootstrap event over `/ws` avoids fragmenting state across REST and WebSocket flows, and it makes refresh/reconnect behavior straightforward because the browser can always request a fresh snapshot on reconnect.
- Alternatives considered:
  - Use REST for initial load and WebSocket only for updates: rejected because it violates the WebSocket-only runtime communication boundary.
  - Hydrate the page with server-rendered data from Next.js: rejected because production assets are statically exported and should remain backend-agnostic.
  - Cache all state in localStorage and reconnect opportunistically: rejected because it risks stale or divergent workspace state.

## Decision 5: Use standard-library browser launch with graceful fallback instead of a browser-opening dependency

- Decision: Open the browser from Go using OS-specific commands invoked through `os/exec` after the local listener is confirmed ready on either the preferred port `4747` or a discovered free port. If launch fails, keep the server running and print the actual local URL plus a plain-language recovery message.
- Rationale: Browser opening is a small, well-bounded responsibility that does not justify a permanent dependency. The standard library keeps the implementation explicit and lets the CLI differentiate between startup failures and post-start browser-launch failures.
- Alternatives considered:
  - Add a dedicated browser-opening package: rejected because standard-library process execution is sufficient and preferred by the constitution unless a dependency adds clear value.
  - Open the browser before confirming the listener is ready: rejected because it can create a broken first-run experience.
  - Treat browser-open failure as fatal: rejected because the spec requires Relay to remain usable when the OS blocks automatic launch.

## Decision 6: Treat configured ports as preferences and keep fallback ports runtime-only unless explicitly saved

- Decision: Store `4747` as the preferred Relay port and `3000` as the preferred frontend dev port, but automatically discover free fallback ports at runtime without overwriting the saved preference unless the developer explicitly chooses a new preferred port.
- Rationale: This minimizes startup failures and preserves predictable defaults while avoiding surprising preference churn caused by transient port conflicts.
- Alternatives considered:
  - Fail immediately on any port collision: rejected because it creates avoidable friction and directly conflicts with the clarified requirement.
  - Persist the fallback port automatically: rejected because a temporary collision should not silently rewrite the user's preferred default.