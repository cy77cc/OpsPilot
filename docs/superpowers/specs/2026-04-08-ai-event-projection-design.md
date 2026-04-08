# AI Event and Projection Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Inherits:
  - `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
  - `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- References:
  - `docs/superpowers/specs/2026-03-28-chat-event-iterator-approval-ui-design.md`
  - `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`
- Scope: canonical event families, canonical envelope, payload rules, replay projection rules, SSE mapping rules, trace/eval projection rules
- Goal: define the immutable event truth model and the derived read models used by replay, SSE, trace, evaluation, and UI projection

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary, the event-first truth model, the reuse-and-deletion posture, and the runtime approval and checkpoint semantics defined by the parent blueprint and upstream detailed designs.

If this document conflicts with the blueprint, the core domain model, the runtime kernel, or the tool and policy design on object ownership, lifecycle state, approval resume, or naming, the upstream documents win.

Hard boundary:

- canonical events are the source of truth
- SSE payloads are derived public transport events
- UI blocks are derived read models
- replay blocks are derived inspection views
- trace spans and evaluation cases are derived analytics and regression views

Projection layers may transform canonical events, but they must never become the truth model themselves.

Explicit terminology rule:

- same-run refinements use `refinement` or `reassessment`
- `replan` is reserved for a newly created run attempt
- approval resume is in-run continuation

## 2. Canonical Event Families

The runtime emits canonical events across six families.

### 2.1 Run Lifecycle

Canonical run lifecycle events describe attempt creation, route selection, execution start, terminal status, and cancellation.

- `run_created`
- `run_routed`
- `run_started`
- `run_completed`
- `run_failed`
- `run_cancelled`

Semantics:

- `run_created` records the durable creation of a new run attempt.
- `run_routed` records the effective route chosen for that attempt.
- `run_started` records the start of runtime execution after route establishment.
- `run_completed` records a durable successful terminal outcome.
- `run_failed` records a durable non-success terminal outcome.
- `run_cancelled` records explicit user or operator cancellation.

### 2.2 Turn Lifecycle

Canonical turn lifecycle events describe internal runtime decision rounds.

- `turn_started`
- `turn_planned`
- `turn_refined`
- `turn_completed`

Semantics:

- `turn_started` marks the beginning of one coherent decision round.
- `turn_planned` records the first chosen path for that turn.
- `turn_refined` records same-run refinement or reassessment after new evidence or policy input.
- `turn_completed` records closure of the turn after the action or answer is durably recorded.

### 2.3 Tool Lifecycle

Canonical tool lifecycle events describe structured tool selection and execution.

- `tool_selected`
- `tool_started`
- `tool_succeeded`
- `tool_failed`
- `tool_skipped`

Semantics:

- `tool_selected` records the chosen tool contract and invocation intent.
- `tool_started` records the tool invocation start.
- `tool_succeeded` records a successful tool result.
- `tool_failed` records a failed tool result or execution error.
- `tool_skipped` records a deliberate decision not to execute a planned tool call.

### 2.4 Approval Lifecycle

Canonical approval lifecycle events describe human decision gates and resume transitions.

- `approval_requested`
- `approval_granted`
- `approval_rejected`
- `approval_expired`
- `run_resuming`

Semantics:

- `approval_requested` freezes the operation snapshot that needs human review.
- `approval_granted` records an approved request and the intent to resume the same run.
- `approval_rejected` records a rejection for that exact approval snapshot.
- `approval_expired` records that the approval gate is no longer valid.
- `run_resuming` records checkpoint-backed continuation of the same run after approval.

### 2.5 Assistant Output

Canonical assistant output events describe user-facing language emitted by the runtime.

- `assistant_delta`
- `assistant_message_final`
- `assistant_summary_updated`

Semantics:

- `assistant_delta` carries incremental assistant output chunks.
- `assistant_message_final` records the finalized assistant message for the turn.
- `assistant_summary_updated` records a durable summary update for the session or run context.

### 2.6 Evaluation and Safety

Canonical evaluation and safety events describe inspection, judgment, and safety outcomes.

- `route_evaluated`
- `tool_choice_evaluated`
- `outcome_evaluated`
- `safety_violation_detected`

Semantics:

- `route_evaluated` records the route decision and its rationale.
- `tool_choice_evaluated` records the tool family selection or rejection decision.
- `outcome_evaluated` records the judged result for a run or replay case.
- `safety_violation_detected` records a policy or safety breach that the runtime must not hide.

## 3. Canonical Event Envelope

The canonical envelope is the immutable metadata shell around every event.

### 3.1 Required Envelope Fields

Every canonical event should carry the following field groups.

| Field group | Required contents |
| --- | --- |
| Identity | `event_id`, `run_id`, canonical `event_type`, schema or payload version |
| Ownership | `session_id`, `task_id`, `turn_id`, `tool_call_id`, `approval_id` as applicable |
| Ordering | `seq`, logical ordering key, tie-break key |
| Causality | `caused_by_event_id`, `parent_event_id`, correlation id |
| Actor | actor type, source component, status |
| Timing | `occurred_at`, `created_at` |
| Payload | compact JSON payload, content ids for large blobs, preview summary when useful |
| Visibility | optional internal classification when the runtime needs a narrow internal label |
| Traceability | `trace_id`, optional evaluation case linkage when present |

### 3.2 Envelope Rules

1. `event_id` is the stable identity for one canonical fact.
2. `seq` provides monotonic ordering inside one run.
3. `caused_by_event_id` records the direct prior fact that triggered the event when there is one.
4. `parent_event_id` records hierarchical lineage when a derived event is created from another event.
5. `trace_id` links canonical runtime facts to derived trace views.
6. Visibility metadata is internal only and must not be treated as a transport schema.

## 4. Payload Rules

Canonical payloads describe facts, not presentation.

### 4.1 Payload Shape

1. Payloads should be compact JSON with a stable schema version.
2. Large or repeated data belongs in content blobs or other durable artifacts, referenced by content id.
3. Payloads should contain enough structure to replay, inspect, and evaluate the action without reading assistant prose.
4. Payloads should prefer normalized fields over free-form text when both are available.
5. Payloads should record the state transition, the decision rationale, and the minimal evidence summary.

### 4.2 Payload Restrictions

1. Do not store UI-only formatting as canonical payload data.
2. Do not store SSE transport shape as canonical payload data.
3. Do not store prompt glue, template fragments, or transient iterator state as the source of truth.
4. Do not rewrite prior event payloads to correct later understanding; append a new event instead.
5. Do not use assistant text as the only durable record of tool intent or approval intent.

### 4.3 Payload Versioning

1. Canonical event payloads are versioned.
2. A materially changed payload schema should be emitted as a new version rather than silently mutating old semantics.
3. Replay and evaluation code should branch on the recorded payload version when needed.

## 5. Ordering, Causality, And Idempotency

### 5.1 Ordering Rules

1. Canonical events inside a run are ordered by `seq`.
2. If two records share the same ordering position during recovery or comparison, use `event_id` as the tie-breaker.
3. Projection layers must preserve canonical ordering.
4. Replay and SSE attach paths must never emit a later canonical event before an earlier one from the same run.
5. Turn, tool, and approval events should be emitted in canonical persisted `seq` order only.

### 5.2 Causality Rules

1. Every derived event should point back to the canonical fact that caused it when that relationship exists.
2. `approval_requested` should point back to the tool selection or policy decision that required the gate.
3. `approval_granted`, `approval_rejected`, and `approval_expired` should point back to the original approval request.
4. `run_resuming` should point back to the approval grant and the persisted checkpoint-related facts.
5. `turn_refined` should point back to the evidence or policy event that justified same-run refinement.

Causality precedence:

1. `caused_by_event_id` stores the direct trigger event.
2. `parent_event_id` stores structural lineage for derived grouping.
3. Additional prerequisite references belong in correlation fields or payload metadata, not by overloading direct trigger semantics.

### 5.3 Idempotency Rules

1. Canonical events are append-only and immutable.
2. Appending the same logical fact twice is a bug unless the second write is an explicit, distinct corrective event.
3. Projection application must be idempotent: replaying the same event sequence should not duplicate read-model state.
4. SSE emission should be idempotent from the client’s perspective when the same cursor is reattached.
5. The runtime should prefer deterministic logical keys so retries can collapse onto one canonical fact when the storage layer supports that behavior.

## 6. Replay Projection Rules

Replay is a derived read model built from canonical events.

### 6.1 Replay Principles

1. Replay consumes canonical events only.
2. Replay does not reconstruct truth from transcript text alone.
3. Replay should rebuild the visible path from canonical facts, not from UI state.
4. Replay should preserve the canonical ordering and causality chain.
5. Replay should expose unresolved approval state using canonical approval events and the current run state.

### 6.2 Replay Block Model

Replay blocks are stable inspection units derived from event sequences.

| Block type | Derived from | Purpose |
| --- | --- | --- |
| `message` | assistant output events | show user-facing assistant content |
| `plan` | turn and route events | show the active or revised execution plan |
| `tool_call` | tool selection and tool lifecycle events | show the selected operation and its execution state |
| `approval` | approval lifecycle events plus tool context | show the operation requiring human decision |
| `result` | tool success or run completion events | show the observed outcome |
| `error` | failure or safety events | show terminal or recoverable failure context |
| `status` | run and turn lifecycle events | show the current execution state |

### 6.3 Replay Cursor Rules

1. The replay and attach cursor is a canonical persisted `event_id` pointer for one run.
2. Cursor advancement happens only after an SSE or replay emission has acknowledged at least one concrete canonical event id.
3. Summary-only transport frames that do not map to a new canonical event (for example derived `run_state`) must not advance the canonical cursor.
4. Reattach should resume strictly after the recorded canonical `event_id`.
5. If the cursor is unknown, expired, or inconsistent with run history, the attach path should fail explicitly instead of guessing.
6. Cursor expiry should not trigger silent full replay fallback.
7. New events appended during replay should be emitted exactly once in canonical tail order.

### 6.4 Approval Replay Rules

1. `approval_requested` creates the canonical approval block.
2. `approval_granted`, `approval_rejected`, and `approval_expired` close that canonical approval decision block.
3. A projection may keep a derived approval card visible after `approval_granted` until `run_resuming` and downstream execution state are rendered, but that is UI/read-model visibility, not canonical approval openness.
4. Approval replay must preserve the exact persisted operation snapshot, including tool identity, argument preview, risk explanation, checkpoint linkage, and resume target binding.
5. Approval replay must not infer a different resume target from UI state or prompt text.

## 7. SSE Mapping Rules

SSE is a public transport projection, not canonical truth.

### 7.1 Mapping Principles

1. SSE events are derived from canonical events and may be rearranged only into transport-friendly shapes, not into new facts.
2. The SSE layer must preserve canonical ordering semantics.
3. SSE event names should remain compatible with current clients unless a deliberate API break is approved elsewhere.
4. SSE payloads should contain only the fields needed by clients to render the current state or incremental update.
5. SSE payloads must not become a second write model.

### 7.2 Canonical-To-SSE Mapping

| Canonical event family | Typical SSE shape | Notes |
| --- | --- | --- |
| run lifecycle | `run_state`, `done`, `error` | state snapshots and terminalization are transport summaries |
| turn lifecycle | `meta` or `delta` | turn boundaries can appear as stream metadata or derived updates |
| tool lifecycle | `tool_result` or `delta` | preserve tool identity and outcome details |
| approval lifecycle | `tool_approval` | approval is a derived transport projection of the frozen approval snapshot |
| assistant output | `delta` and final assistant message transport events | incremental and terminal assistant text are transport concerns |
| evaluation and safety | `error`, metadata, or inspection-only transport | evaluation details may not be streamed to the end user |

### 7.3 SSE Compatibility Rules

1. Existing SSE field names should be preserved when possible.
2. Any additive transport field must not change canonical event semantics.
3. The `tool_approval` transport event must always be derivable from canonical approval events.
4. Transport events must never own approval state, route state, or run lifecycle truth.
5. If a transport event cannot be derived from canonical events, the projection is broken and should not emit a fabricated substitute.

## 8. Trace And Evaluation Projection Rules

Trace and evaluation projections are specialized read models built from canonical events.

### 8.1 Trace Projection Rules

1. Trace projection groups events by `trace_id`, `run_id`, `turn_id`, and `tool_call_id` where available.
2. Trace spans are derived from canonical lifecycle boundaries, not from UI screens.
3. Trace views should surface causal chains, tool timing, approval pauses, checkpoint resumption, and terminal reasons.
4. Trace projections should preserve enough structure to diagnose ordering bugs and resume failures.
5. Trace projection must never reclassify a canonical event into a different fact.

### 8.2 Evaluation Projection Rules

1. Evaluation cases should be built from canonical event sequences and durable snapshots.
2. Evaluation should consume canonical events directly, not SSE payloads.
3. A replayed run and its derived evaluation case should share the same ordering and causality model.
4. Evaluation outputs should record whether route choice, tool choice, approval behavior, and final outcome matched expectation.
5. When evaluation intentionally tolerates variation, that tolerance must be explicit in the case definition, not implied by the projection.

### 8.3 Trace/Eval Relation To Replay

1. Replay is for inspection.
2. Trace is for operational diagnosis.
3. Evaluation is for regression and correctness judgment.
4. All three are derived from the same canonical event source.

## 9. UI Block Projection Table

The UI should consume derived blocks, not canonical events directly.

| UI block | Source canonical events | Derived content | Disposable or reusable guidance |
| --- | --- | --- | --- |
| `message` | `assistant_delta`, `assistant_message_final` | rendered assistant text, message state, author metadata | reusable if it is derived from canonical assistant output; disposable if it duplicates raw stream fragments |
| `plan` | `run_routed`, `turn_started`, `turn_planned`, `turn_refined` | route summary, current objective, same-run refinement notes | reusable as a read model; disposable if embedded in prompt text only |
| `tool_call` | `tool_selected`, `tool_started`, `tool_succeeded`, `tool_failed`, `tool_skipped` | tool name, arguments preview, timing, status, result preview | reusable if it references canonical tool facts; disposable if it mirrors ad hoc UI state |
| `approval` | `approval_requested`, `approval_granted`, `approval_rejected`, `approval_expired`, `run_resuming` | frozen operation preview, risk explanation, decision status, resume hint | reusable as a derived approval card; disposable if it depends on hidden iterator state |
| `result` | `tool_succeeded`, `run_completed`, `outcome_evaluated` | final tool outcome, run outcome, success summary | reusable when it summarizes canonical outcomes; disposable if it is only a display convenience |
| `error` | `tool_failed`, `run_failed`, `run_cancelled`, `safety_violation_detected` | error code, human-safe message, failure category, recovery hint | reusable for operator diagnosis; disposable if it invents extra meaning not present in canonical events |
| `status` | run and turn lifecycle events | current run state, current turn state, checkpoint state, waiting state | reusable as a live read model; disposable if it is treated as source truth |

UI implementation guidance:

1. The approval block should surface the concrete operation preview before any approve or reject action.
2. The UI should prefer structured summaries first and raw JSON second.
3. Legacy events without a structured preview should degrade gracefully and remain actionable.
4. UI blocks should be rebuildable from the canonical event stream plus durable snapshots.

## 10. Approval And Resume Semantics

Approval and resume are canonical event flows, not UI-only state.

### 10.1 Approval Semantics

1. `approval_requested` freezes the exact operation snapshot that needs human review.
2. The snapshot should include the tool identity, argument preview, risk explanation, checkpoint id, and resume target binding.
3. Once the approval request is persisted, later runtime changes must not rewrite that approval record.
4. If the same operation needs approval again, the runtime should create a new approval record instead of mutating the old one.
5. `approval_granted`, `approval_rejected`, and `approval_expired` apply only to the persisted approval snapshot they reference.

### 10.2 Resume Semantics

1. `run_resuming` records continuation of the same interrupted run.
2. Resume preserves `run_id`, `task_id`, and `session_id`.
3. Resume uses the persisted checkpoint and the persisted target binding.
4. Resume does not create a new run attempt.
5. Resume failure should be projected as a failure event, not hidden behind transport retries.

### 10.3 Resume Binding Rules

1. The approval resume target should bind to the exact interrupted call first.
2. Checkpoint id is required for runner restore, but it is not the primary target selector.
3. If the resume target or checkpoint cannot be validated, the runtime should fail the attempt with an explicit checkpoint or resume error.
4. A resumed run should emit a canonical continuation event before any downstream read model claims the run has resumed.

## 11. Current System Reuse And Disposal Guidance

The projection layer should reuse durable assets that already behave like canonical truth or stable read models.

Likely reusable:

- `AIRunEvent`
- `AIRunProjection`
- `AIApprovalOutboxEvent` as a delivery companion, not as canonical truth
- `AICheckpoint`
- `AITraceSpan`
- `AIUsageLog`
- structured approval preview payloads that already represent the frozen operation snapshot

Likely disposable:

- duplicated projection logic across backend and frontend
- SSE-specific state used as a hidden runtime source of truth
- prompt-assembled control-flow glue
- replay state reconstructed only from assistant prose
- approval state inferred from UI cards instead of canonical events

Rule:

- if a datum is durable truth, preserve it
- if a datum only exists to move the old control flow along, delete it unless a later design explicitly keeps it

## 12. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- `docs/superpowers/specs/2026-03-28-chat-event-iterator-approval-ui-design.md`
- `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`
