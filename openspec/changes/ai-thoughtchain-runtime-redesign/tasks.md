## 1. Legacy Runtime Inventory And Removal

- [ ] 1.1 Inventory all AI chat primary-path uses of legacy `turn/block`, `phase/step`, detached approval, and compatibility SSE events across backend and frontend entry points.
- [ ] 1.2 Remove legacy AI chat primary-path event emission and consumption for `turn/block`, `phase_started`, `phase_complete`, `plan_generated`, `step_started`, `step_complete`, `replan_triggered`, and detached `approval_required` semantics.
- [ ] 1.3 Remove obsolete frontend runtime mergers, fallback JSON dumping paths, and unavailable-state transitions that depend on legacy protocol families.
- [ ] 1.4 Remove duplicate approval resume and confirmation branches that exist only to support the old chat runtime.

## 2. ThoughtChain Runtime Contract

- [ ] 2.1 Define the canonical thoughtChain streaming contract, node model, and payload types in backend and frontend API contract layers.
- [ ] 2.2 Implement backend chain lifecycle emission for `chain_started`, `chain_meta`, `node_open`, `node_delta`, `node_replace`, `node_close`, `chain_paused`, `chain_resumed`, `chain_completed`, and `chain_error`.
- [ ] 2.3 Implement canonical session replay structures that reconstruct assistant history from thoughtChain nodes and final answer state.

## 3. Unified Approval Flow

- [ ] 3.1 Implement approval as a first-class `approval` node that pauses the active chain before gated execution starts.
- [ ] 3.2 Add the unified chain approval decision API using `chain_id` and `approval node_id`, and resume or terminate the same chain based on the decision.
- [ ] 3.3 Remove remaining frontend and backend dependencies on detached approval ticket semantics for the AI chat primary flow.

## 4. Frontend ThoughtChain Experience

- [ ] 4.1 Rebuild AI assistant state around a single thoughtChain store shared by live streaming and replay restoration.
- [ ] 4.2 Implement the upgraded thoughtChain UI with dedicated node rendering for `plan`, `step`, `tool`, `approval`, `replan`, and `answer`.
- [ ] 4.3 Fix new-conversation recommended prompt submission so an assistant chain container is created immediately and never falls into a false unavailable state before first response events.

## 5. Observability And Metrics

- [ ] 5.1 Add backend thoughtChain lifecycle callbacks for chain start, node updates, approval resolution, replan, completion, and failure.
- [ ] 5.2 Export thoughtChain chain/node counters, durations, approval wait metrics, and replan metrics through the existing Prometheus integration.
- [ ] 5.3 Add trace/span propagation for `trace_id`, `chain_id`, `node_id`, `scene`, `tool`, and status across normal execution and approval resume paths.

## 6. Validation And Regression Coverage

- [ ] 6.1 Add backend tests covering thoughtChain lifecycle ordering, approval pause/resume, replan behavior, and non-emission of removed legacy primary-path events.
- [ ] 6.2 Add frontend tests covering thoughtChain rendering, approval node interaction, replay/live consistency, and recommended prompt race handling.
- [ ] 6.3 Run `openspec validate --json` and project test suites relevant to AI runtime changes, then remove any temporary migration shims before marking the change complete.
