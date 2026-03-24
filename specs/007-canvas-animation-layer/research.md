# Research: Canvas Animation Layer

## Decision 1: Wrap the node root in `motion.div` and use variants for spawn and state presentation, but never use Framer Motion `layout`

- Decision: Convert the outer canvas node shell from a static `div` to a `motion.div`, use named variants for enter and exit scale-and-fade behavior, and drive state presentation with `animate` updates keyed from existing node state. Keep all durations at 300ms with the shared easing curve `cubic-bezier(0.16, 1, 0.3, 1)`.
- Rationale: The user explicitly wants node spawn and state changes to feel intentional while avoiding ad-hoc timings. Framer Motion already exists in the frontend dependency set, and upstream guidance supports `motion.div` plus `AnimatePresence` for enter and exit behavior. Avoiding the `layout` prop is critical because React Flow already controls positioning and layout recalculation; combining layout animation with that positioning layer risks thrash and visual instability.
- Alternatives considered:
  - Animate nodes with CSS transitions only: rejected because enter and exit coordination plus state variants are harder to express and test consistently.
  - Use Framer Motion `layout` or layout groups: rejected because the user explicitly identified layout thrash risk on the canvas.
  - Rebuild the whole node tree on each event: rejected because the current canvas patch model intentionally preserves node identity and interactivity.

## Decision 2: Drive the edge pulse through a custom React Flow edge component fed by edge data, not by direct event subscriptions inside the edge

- Decision: Add a dedicated custom edge component under `features/canvas` that reads pulse state from edge data, renders the path with standard React Flow geometry helpers, and animates the SVG stroke with dash offset only while a handoff is active.
- Rationale: Relay already receives `handoff_start` and `handoff_complete` events. The cleanest architecture is to translate those events into canvas document edge data in `canvasModel.ts` and let the edge renderer remain purely declarative. React Flow documentation supports custom edge components with `data` props and SVG path rendering, which aligns with the user's requirement to avoid direct event subscriptions from the edge layer.
- Alternatives considered:
  - Use the built-in React Flow animated edge flag: rejected because it is too generic and does not model handoff start and completion windows precisely.
  - Subscribe to workspace events from the edge component itself: rejected because it would duplicate store logic and break the presentation-only boundary.
  - Persist pulse state on the backend: rejected because animation ownership belongs on the client and the spec forbids presentation driving state.

## Decision 3: Represent streaming animation as a frontend-only derived activity window backed by recent token timestamps

- Decision: Keep the visible node state authoritative in the current canvas model, but derive a separate presentation-only `isStreamingActive` flag from two conditions: the node is in an executing-style state and at least one token has arrived within the last 300ms. Track the most recent token timestamp in a ref-backed timer per rendered node and clear the pulse after 300ms of silence, with cleanup on unmount.
- Rationale: The current store and canvas model already react to `token` events and set the node state to `streaming`. The user wants a more precise visual signal that reflects active token arrival rather than a coarse long-lived state. A local timer and timestamp ref satisfies that behavior without asking the backend to manage animation windows.
- Alternatives considered:
  - Leave the current `streaming` state glow unchanged: rejected because it does not distinguish active token arrival from a stale streaming state.
  - Add a backend heartbeat or explicit animation event: rejected because it adds transport complexity for a purely presentational need.
  - Store the silence timeout in the global workspace store: rejected because the lifetime is local to the rendered node and increases shared-state complexity.

## Decision 4: Animate the detail panel with `AnimatePresence` and keyed content transitions, while preserving the existing side-panel semantics

- Decision: Wrap the rendered detail panel surface with `AnimatePresence` and use horizontal `x` transitions from `380` to `0` on enter and `0` to `380` on exit. Keep the panel keyed by selection state so the latest node selection always wins, and avoid reworking the panel into a separate routing or overlay system.
- Rationale: The current `AgentNodeDetailPanel` already owns the selected-node and empty-selection views. `AnimatePresence` is a direct fit for animating entry and exit without changing the surrounding canvas layout model, and the requested motion path is explicit. The main engineering risk is incorrect wrapper placement around React Flow-managed content, so the panel animation should stay outside the React Flow node tree.
- Alternatives considered:
  - Animate the panel with CSS transitions only: rejected because enter and exit coordination across selection changes is more brittle.
  - Animate the whole canvas detail grid as one unit: rejected because that would couple panel motion to canvas rendering and increase layout churn.
  - Mount a second off-canvas drawer component elsewhere in the app tree: rejected because it duplicates current canvas detail behavior.

## Decision 5: Preserve performance and reduced-motion behavior by limiting animated properties and keeping motion declarative

- Decision: Restrict canvas motion to opacity, scale, and color-adjacent presentation layers; do not animate layout, width, height, or graph coordinates. Respect reduced-motion preferences by shortening or disabling non-essential movement while preserving visible state indicators, borders, and labels.
- Rationale: Relay's constitution requires responsive streaming interactions, and the feature spec explicitly calls out performance and truthfulness. The existing canvas already patches node state in place and only reruns layout on spawn, so the motion layer should stay aligned with that pattern. Reduced-motion support is also necessary for accessibility and for keeping the feature compliant with the existing UI guidance.
- Alternatives considered:
  - Animate positions or relayout transitions between node arrangements: rejected because replay motion is out of scope and position animation is the highest-risk source of React Flow thrash.
  - Use richer transform stacks or blur effects: rejected because they add cost without improving state clarity.
  - Disable all animation when reduced motion is enabled: rejected because the interface still needs to communicate state changes clearly.