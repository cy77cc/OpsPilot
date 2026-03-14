## Context

The AI module currently mixes multiple runtime abstractions across backend streaming, frontend rendering, approval handling, and replay:

- frontend combines legacy `thoughtChain` items, `turn/block` replay, and `phase/step` progress events
- backend emits overlapping SSE families for delta text, phase lifecycle, turn/block lifecycle, approval prompts, and compatibility signaling
- approval can leave the primary chat runtime instead of pausing and resuming within one coherent execution model
- new conversations can temporarily appear unavailable because the frontend waits on mismatched runtime signals before constructing the assistant state
- telemetry does not describe one consistent runtime lifecycle that Prometheus can consume

This design intentionally replaces the mixed architecture with one `thoughtChain`-first runtime and deletes obsolete primary-path protocol families before the new runtime becomes default.

## Goals / Non-Goals

**Goals:**
- make `thoughtChain` the single runtime contract for AI chat execution, approval, replay, and rendering
- remove legacy `turn/block`, `phase/step`, and detached approval semantics from the AI chat primary path
- define a single approval decision flow based on `chain_id` and `approval node_id`
- redesign frontend state around one `thoughtChain` store and an upgraded node-based UI
- preserve markdown fidelity across SSE parsing, visible chunk normalization, and replay restoration
- persist runtime-first replay state so completed assistant responses cannot be restored as empty content
- expose chain/node lifecycle callbacks that export metrics to the existing Prometheus integration
- add migration guardrails and tests to prevent legacy protocol concepts from returning

**Non-Goals:**
- changing unrelated RAG, planner quality, expert routing, or business prompt content
- preserving long-term dual runtime compatibility for old clients
- redesigning unrelated deployment or notification approval systems

## Decisions

### Decision: `thoughtChain` is the only primary runtime protocol

The new runtime will represent one assistant response as one chain with ordered nodes. Supported node kinds are `plan`, `step`, `tool`, `approval`, `replan`, `answer`, and narrowly-scoped `status`.

Why:
- it matches the intended user mental model better than forcing the frontend to synthesize a narrative from generic transport events
- it lets live streaming, replay, approval, and telemetry all share the same identity model
- it provides a clean target for deleting protocol overlap

Alternatives considered:
- keep `turn/block` as primary and render `thoughtChain` as a projection: rejected because it keeps the runtime truth under a model the user considers obsolete
- keep both protocols with a compatibility bridge: rejected because dual truth would continue to distort implementation choices

### Decision: delete old primary-path runtime wiring before attaching the new path

The migration order is deliberate:
1. inventory old entry points and dependencies
2. detach old protocol families from the AI chat primary path
3. implement the new `thoughtChain` path on the cleaned boundary

Why:
- the current codebase already shows how transitional compatibility can shape new behavior in the wrong direction
- deletion-first reduces the chance of "temporary" shims becoming permanent

Alternatives considered:
- ship the new runtime in parallel under a feature flag first: rejected because it prolongs protocol duplication and complicates validation

### Decision: approval is a first-class chain node with one decision API

Approval-required execution will open a dedicated `approval` node in `waiting` state. The frontend will resolve it through:

`POST /api/v1/ai/chains/{chain_id}/approvals/{node_id}/decision`

with `approved` and optional `reason`.

Why:
- users can understand approval as part of the same runtime narrative instead of a detached modal or background ticket
- the same node identity can drive pause/resume behavior, replay, tracing, and metrics
- one API simplifies frontend and backend recovery logic

Alternatives considered:
- keep current approval ticket endpoints and map them into chain UI: rejected because the runtime would still branch into detached semantics
- attach approval state onto tool nodes without its own identity: rejected because it weakens observability and resume boundaries

### Decision: frontend uses one chain store for live and replayed assistant state

The assistant UI will be rendered only from `thoughtChain` state. A fresh recommended prompt submission immediately creates the user message plus a placeholder assistant chain container so the UI never enters a false unavailable state while awaiting the first server event.

Why:
- one state source removes the current race between message rendering, replay hydration, and streaming readiness
- the same rendering path can serve live streams and restored sessions

Alternatives considered:
- keep a separate compatibility message model and enrich it with chain state: rejected because it preserves multiple sources of truth

### Decision: node content is split into narrative, structured, and raw layers

ThoughtChain nodes will no longer treat one generic `summary` string as the universal rendering surface. Node payloads should distinguish at least:

- `headline`: short human-readable status under the title
- `body`: detailed phase summary, which may contain markdown
- `structured`: typed plan steps, tool result groups, or other renderable records
- `raw`: explicit raw/debug payloads such as JSON

Why:
- plan JSON, tool result JSON, detailed replanning summaries, and final answer text have different rendering needs
- one overloaded text field is the direct cause of unreadable chain cards and leaked internal payloads
- this boundary lets the UI render rich `ThoughtChain.Item` hierarchies without inventing summaries that the runtime did not actually produce

Alternatives considered:
- keep one `summary` field and teach the UI to guess content type: rejected because it keeps the ambiguity that already caused current failures

### Decision: tool nodes default to beautified raw-result rendering

Tool nodes will present structured raw result views by shape:

- objects become grouped key/value cards
- arrays become compact lists or tables
- recognized business resource shapes such as hosts or clusters may use badges and grouped fields
- unknown payloads fall back to formatted JSON in an explicit disclosure area

Why:
- the user wants faithful raw data, but readable
- AI-authored summaries would require extra post-processing and could drift from the source result
- structured rendering preserves debugging value without producing JSON walls

Alternatives considered:
- summarize tool results into prose first: rejected because it adds another transformation layer the user does not want as the default
- render only formatted JSON: rejected because it remains too log-like for the main ThoughtChain surface

### Decision: streaming parsing must preserve markdown fidelity

The SSE parsing layer and visible chunk normalization must prefer fidelity over cleanup:

- do not trim individual `data:` lines
- do not strip markdown-significant indentation or blank lines
- only unwrap explicit complete protocol envelopes such as `{"response": "..."}`
- do not treat partial JSON fragments as complete envelopes

Why:
- the current frontend parser trims away whitespace that markdown needs for tables, spacing, and code-style rendering
- preserving exact streamed content keeps live rendering and persisted replay consistent

Alternatives considered:
- continue normalizing aggressively and patch markdown rendering later: rejected because the parser itself is the source of corruption

### Decision: session persistence is runtime-first and must flush final answer state

Persistence will store native runtime replay state plus final-answer markdown as the canonical assistant restore model. Compatibility `blocks` remain derived fallback data only.

Persistence must flush on chain lifecycle and final-answer lifecycle boundaries, including terminal `final_answer_done` and completion/error events.

Why:
- the reported `completed` assistant turns with empty `blocks: []` show that the current replay source can claim success while persisting no recoverable content
- runtime-first persistence keeps live and restored rendering aligned

Alternatives considered:
- keep `blocks` as the authoritative replay store and project runtime from them: rejected because it recreates the same data-loss and semantic-loss problems under a legacy model

### Decision: observability is callback-first and exported through Prometheus

The backend runtime will emit explicit callbacks such as `OnChainStarted`, `OnNodeOpened`, `OnApprovalResolved`, and `OnChainCompleted`. These callbacks will feed counters, histograms, and traces keyed by `trace_id`, `chain_id`, `node_id`, `scene`, `tool`, and `status`.

Why:
- callback hooks provide one stable instrumentation surface independent of frontend rendering needs
- Prometheus can consume chain and node metrics through the existing monitoring pipeline without inventing another metrics path

Alternatives considered:
- instrument only at HTTP or handler level: rejected because it cannot capture approval waits, replans, or node-level execution timing accurately

## Risks / Trade-offs

- [Hidden dependencies on legacy events] -> Inventory all legacy consumers before removal, then add regression tests asserting the new path does not emit old primary-path events.
- [Approval regression during migration] -> Promote approval to a chain node before removing old approval resume branches, and require pause/resume tests for every migration slice.
- [Replay/live divergence] -> Use one node model and one frontend rendering path for both live streams and restored sessions.
- [Markdown corruption during streaming] -> Make SSE parsing and visible chunk normalization preserve raw content boundaries, then add parser regression tests with markdown tables and blank lines.
- [Tool cards becoming another unreadable log view] -> Require structured tool renderers plus explicit raw fallback instead of one generic text body.
- [Completed turns restoring as empty content] -> Persist runtime and final-answer state directly, define mandatory flush points, and add replay tests that assert completed assistant turns are never empty.
- [Temporary migration code becoming permanent] -> Allow only minimal short-lived shims and require their removal before marking the change complete.
- [Broader blast radius across frontend and backend] -> Sequence work by protocol cleanup, runtime contract, approval flow, frontend store/UI, then observability and tests.

## Migration Plan

1. Inventory AI chat runtime entry points across `internal/service/ai`, `internal/ai`, `web/src/api/modules/ai.ts`, and `web/src/components/AI`.
2. Detach legacy `turn/block`, `phase/step`, and detached approval semantics from the AI chat primary path.
3. Remove compatibility-only DTOs, event emitters, and frontend runtime mergers that exist only for the old primary path.
4. Implement the new `thoughtChain` streaming contract, markdown-safe parser semantics, node content model, replay contract, and approval decision API.
5. Rebuild the frontend around a single chain store and upgraded node-based assistant UI, including structured `plan`/`replan` steps and beautified tool-result rendering.
6. Add chain/node callbacks and connect metrics to the Prometheus integration.
7. Add regression coverage and validate that old runtime concepts are no longer emitted or rendered on the primary path.

Rollback strategy:
- If rollout must pause mid-change, revert the change before exposing the partial protocol externally rather than keeping long-lived dual primary runtimes.

## Open Questions

- Whether `status` should remain an explicit node kind or be folded into summaries on other node kinds once implementation details are clearer.
