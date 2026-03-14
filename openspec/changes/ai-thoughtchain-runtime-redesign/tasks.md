## 1. Legacy Runtime Inventory And Removal

- [ ] 1.1 Inventory all AI chat primary-path uses of legacy `turn/block`, `phase/step`, detached approval, and compatibility SSE events across backend and frontend entry points.
- [ ] 1.2 Remove legacy AI chat primary-path event emission and consumption for `turn/block`, `phase_started`, `phase_complete`, `plan_generated`, `step_started`, `step_complete`, `replan_triggered`, and detached `approval_required` semantics.
- [ ] 1.3 Remove obsolete frontend runtime mergers, fallback JSON dumping paths, and unavailable-state transitions that depend on legacy protocol families.
- [ ] 1.4 Remove duplicate approval resume and confirmation branches that exist only to support the old chat runtime.

## 2. ThoughtChain Runtime Contract

- [ ] 2.1 Define the canonical thoughtChain streaming contract, node model, and payload types in backend and frontend API contract layers.
- [ ] 2.2 Split node payload semantics into `headline`, `body`, `structured`, and `raw` layers so plan/replan summaries, tool payloads, and final-answer content do not share one generic text slot.
- [ ] 2.3 Implement backend chain lifecycle emission for `chain_started`, `chain_meta`, `node_open`, `node_delta`, `node_replace`, `node_close`, `chain_paused`, `chain_resumed`, `chain_completed`, and `chain_error`.
- [ ] 2.4 Preserve markdown fidelity in SSE parsing and visible chunk normalization by removing user-visible trimming and by unwrapping only explicit complete response envelopes.
- [ ] 2.5 Implement canonical session replay structures that reconstruct assistant history from thoughtChain nodes and final answer state.

## 3. Unified Approval Flow

- [ ] 3.1 Implement approval as a first-class `approval` node that pauses the active chain before gated execution starts.
- [ ] 3.2 Add the unified chain approval decision API using `chain_id` and `approval node_id`, and resume or terminate the same chain based on the decision.
- [ ] 3.3 Remove remaining frontend and backend dependencies on detached approval ticket semantics for the AI chat primary flow.

## 4. Frontend ThoughtChain Experience

- [ ] 4.1 Rebuild AI assistant state around a single thoughtChain store shared by live streaming and replay restoration.
- [ ] 4.2 Implement the upgraded thoughtChain UI with dedicated node rendering for `plan`, `step`, `tool`, `approval`, `replan`, and `answer`.
- [ ] 4.3 Render `plan` and `replan` nodes as structured step lists or item groups instead of serialized JSON or flattened text blobs.
- [ ] 4.4 Render `tool` nodes as beautified raw-result views with structured tables/cards for recognized shapes and explicit raw fallback for unknown payloads.
- [ ] 4.5 Keep the final answer as a separate markdown-first surface while allowing process nodes such as `replan` to show detailed phase summaries in their own body area.
- [ ] 4.6 Fix new-conversation recommended prompt submission so an assistant chain container is created immediately and never falls into a false unavailable state before first response events.

## 5. Persistence And Replay

- [ ] 5.1 Persist runtime-first assistant replay data, including native runtime state and final-answer markdown, instead of allowing completed turns to rely on empty compatibility blocks.
- [ ] 5.2 Define and implement recorder flush points for chain lifecycle and final-answer lifecycle events, including terminal flush on `final_answer_done` and completion/error.
- [ ] 5.3 Make replay restoration prefer persisted runtime and final-answer state before compatibility `blocks` or legacy thoughtChain fallbacks.

## 6. Observability And Metrics

- [ ] 6.1 Add backend thoughtChain lifecycle callbacks for chain start, node updates, approval resolution, replan, completion, and failure.
- [ ] 6.2 Export thoughtChain chain/node counters, durations, approval wait metrics, and replan metrics through the existing Prometheus integration.
- [ ] 6.3 Add trace/span propagation for `trace_id`, `chain_id`, `node_id`, `scene`, `tool`, and status across normal execution and approval resume paths.

## 7. Validation And Regression Coverage

- [ ] 7.1 Add backend tests covering thoughtChain lifecycle ordering, approval pause/resume, replan behavior, runtime-first persistence, and non-emission of removed legacy primary-path events.
- [ ] 7.2 Add frontend tests covering markdown-safe SSE parsing, thoughtChain rendering, structured tool/result presentation, approval node interaction, replay/live consistency, and recommended prompt race handling.
- [ ] 7.3 Add regression tests asserting a completed assistant turn cannot persist as empty replay content.
- [ ] 7.4 Run `openspec validate --json` and project test suites relevant to AI runtime changes, then remove any temporary migration shims before marking the change complete.
