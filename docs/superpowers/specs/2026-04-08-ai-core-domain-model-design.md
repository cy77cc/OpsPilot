# AI Core Domain Model Design

- Date: 2026-04-08
- Parent: docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md
- Scope: object model for Session, Task, Run, Turn, ToolCall, Approval, Event, EvaluationCase
- Goal: define durable object boundaries and ownership before runtime implementation

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary, the event-first truth model, the deletion posture, and the state vocabulary defined in the parent blueprint.

If this document conflicts with the parent blueprint on object ownership, lifecycle state, reuse posture, or naming, the blueprint wins.

The 2026-04-08 redesign spec is a reference input, but it is subordinate to the blueprint and to this document's object-level decisions.

## 2. Object Relationships And Normalization

The core ownership chain is:

- `Session` owns collaboration continuity
- `Task` owns the business objective
- `Run` owns one verifiable execution attempt
- `Turn` owns one internal decision round
- `ToolCall` owns one structured tool invocation attempt
- `Approval` owns one human decision gate
- `Event` owns immutable runtime facts
- `EvaluationCase` owns one reusable regression scenario

The important references are:

- `Session -> Task`
- `Task -> Run`
- `Run -> Turn`
- `Turn -> ToolCall`
- `Run -> Approval`
- `Run -> Event`
- `Task/Run -> EvaluationCase`

Normalization rules:

1. Store ownership in the smallest object that can answer the question it owns.
2. Store stable summaries in the owning row, not in downstream projections.
3. Store large payloads and repeatable blobs separately, then reference them.
4. Never derive durable state from assistant text, prompt glue, or UI projection state.
5. Treat `Event` as append-only truth; everything else is either a durable snapshot or a read model built from events.
6. When a decision needs a fresh attempt, create a new object instance instead of mutating the old one in place.

Embedded versus referenced:

| Object | Embed by default | Reference by default |
| --- | --- | --- |
| `Session` | current title, latest summary, user preference snapshot, collaboration status | tasks, runs, message history blobs, evidence content |
| `Task` | objective summary, route/domain snapshot, task status, priority, unresolved question summary | session, latest run, source message or request, evidence ids |
| `Run` | run status, route/domain, current turn summary, checkpoint id, final outcome summary | task, session, turns, approvals, events, large trace or content blobs |
| `Turn` | ordinal, phase, decision summary, selected agent, stop reason | run, tool calls, approvals, evidence content |
| `ToolCall` | tool identity, arguments preview, result preview, policy outcome, timing summary | run, turn, tool contract id, large args/result content ids |
| `Approval` | pending snapshot, risk explanation, decision metadata, resume target summary | run, tool call, checkpoint, preview content ids, policy rule ids |
| `Event` | canonical envelope, ordering fields, causality ids, compact payload | large shared content blobs, read-model projections, transport-specific shape |
| `EvaluationCase` | expected route, expected outcome, assertion set, fixture metadata | live runs, live sessions, large transcripts, generated report artifacts |

## 3. Session

### Responsibility

`Session` is the long-lived collaboration container. It preserves the user's working context across multiple tasks and runs, and it answers: what is the user trying to do over time?

### Core Field Groups

- Identity: `session_id`, `user_id`, optional domain or scene tag if the migration still needs it
- Collaboration summary: title, latest summary, current objective snapshot, last meaningful user intent
- Preference state: user preferences, tone or interaction hints, durable collaboration settings
- Linkage: task ids, latest task id, latest run id, active run id if a run is in flight
- Lifecycle metadata: created_at, updated_at, deleted_at or archive marker

### Ownership Boundaries

- Owns conversation continuity, not execution state.
- Owns the durable session header and summaries, not the full runtime transcript as the source of truth.
- Can point at many tasks, but does not own task semantics.

### Lifecycle Notes

- Created when a user starts a durable conversation or when the system needs a named collaboration context.
- May stay open across many tasks and many runs.
- Can be updated by summaries and preference changes without changing its identity.
- Should not be used as a substitute for execution state.

### Non-Responsibilities

- No tool execution state machine.
- No approval lifecycle ownership.
- No event stream ownership.
- No routing or policy decisions.

### Current-System Reuse Guidance

Likely reusable:

- `AIChatSession`
- `AIChatMessage` as transcript evidence, not as the truth model
- existing session summary data
- existing session title and user-scoped listing behavior

Likely disposable:

- `scene`-driven control flow if it only exists to steer prompt assembly
- any session logic inferred from assistant message status
- duplicate session state embedded in runtime glue

## 4. Task

### Responsibility

`Task` is the semantic business goal extracted from a session. It represents what the system is trying to achieve, not a specific model invocation.

### Core Field Groups

- Identity: `task_id`, `session_id`, optional source request id
- Objective: task title, objective summary, problem statement, desired outcome
- Routing snapshot: initial classification snapshot, business-level routing context, route confidence, rationale summary
- Ownership state: task status, priority, current owner role if the workflow needs one
- Progress linkage: latest run id, active run id, unresolved question summary
- Provenance: source message id, source request metadata, creation reason
- Lifecycle metadata: created_at, updated_at, closed_at or archived_at

### Ownership Boundaries

- Owns the business objective and its durable identity.
- Can spawn multiple runs for retries, replans, or materially fresh attempts.
- Does not own in-run approval resume; approval resume continues the existing run.
- Should not own tool state, approval state, or event sequencing.

### Lifecycle Notes

- Usually created from a session request or router decision.
- May remain open while several runs are attempted against the same goal.
- Can be superseded by a follow-up task when the objective materially changes.
- Should be the anchor for evaluation and outcome assessment.

### Non-Responsibilities

- No conversation transcript ownership.
- No model loop ownership.
- No tool contract or policy ownership.
- No event stream storage.

### Current-System Reuse Guidance

Likely reusable:

- `AIRun` metadata such as `assistant_type`, `intent_type`, `risk_level`, and `progress_summary` as migration source material
- `AIDiagnosisReport` as a durable outcome artifact for some task families
- `AITraceSpan` and `AIUsageLog` as supporting evidence for task-level analysis

Likely disposable:

- scene-specific prompt fragments used only to infer intent
- hidden task state encoded inside prompt assembly or message text
- duplicate intent labels that only exist because the current model lacks a task table

## 5. Run

### Responsibility

`Run` is one concrete, verifiable execution attempt for a task. It answers: what happened during this specific execution attempt?

### Core Field Groups

- Identity: `run_id`, `task_id`, `session_id`, client request id or idempotency key
- Routing and domain: effective execution route for this attempt, route kind, domain, interaction mode, agent family, route confidence
- Lifecycle state: `created`, `routing`, `planning`, `executing`, `waiting_approval`, `resuming`, `completed`, `failed`, `cancelled`
- Current execution snapshot: current turn id, current phase, pending approval id, checkpoint id
- Outcome summary: final status, progress summary, error code, error message, completion summary
- Timing and observability: started_at, last_event_at, finished_at, trace id
- Provenance: user message id or source request id, parent task reference

### Ownership Boundaries

- Owns one durable execution instance and the state transitions that belong to it.
- Owns the root identity for replay, approval, resume, and auditing.
- Owns the authoritative execution route for this attempt.
- Does not own the collaboration transcript or the task objective itself.

### Lifecycle Notes

- A task may produce zero, one, or many runs.
- A run may pause for approval and later resume, but approval resume is in-run continuation and keeps the same run identity.
- Terminal states should be durable and externally meaningful.
- Any substantial reroute or fresh attempt should create a new run rather than mutating a completed one into a different attempt.
- Retry, replan, or materially fresh execution attempts create new runs; approval resume does not.

### Non-Responsibilities

- No collaboration-summary ownership.
- No direct tool registry ownership.
- No frontend projection ownership.
- No prompt-assembly ownership.

### Current-System Reuse Guidance

Likely reusable:

- `AIRun`
- `AIRunEvent`
- `AIRunProjection`
- `AIRunContent`
- `AICheckpoint`
- `AITraceSpan`
- `AIUsageLog`

Likely disposable:

- run-state duplication that only exists in chat handler glue
- scene-based orchestration flags inside prompt construction
- assistant message status fields that are used as a hidden execution engine

## 6. Turn

### Responsibility

`Turn` is one internal runtime decision and execution round inside a run. It answers: what did the runtime decide and do in this round?

### Core Field Groups

- Identity: `turn_id`, `run_id`, turn ordinal or sequence number
- Phase: `context_ready`, `model_decided`, `tool_selected`, `tool_running`, `tool_finished`, `awaiting_human`, `answering`, `turn_done`
- Decision snapshot: selected agent, selected route, model decision summary, stop reason
- Evidence summary: context summary, open questions, latest evidence pointers
- Linkage: tool call ids, approval ids, event ids, parent turn id if the runtime supports a chain
- Timing: started_at, finished_at

### Ownership Boundaries

- Owns one atomic runtime loop iteration.
- Bridges the top-level run state and the lower-level tool and approval actions.
- Does not own the run lifecycle or the collaboration container.

### Lifecycle Notes

- Created when the runtime begins a new decision round.
- Should be written once and then closed.
- If the runtime needs a materially new decision round, it should open a new turn rather than rewriting the prior one.
- Turn boundaries should make replay and debugging easier, not harder.

### Non-Responsibilities

- No session ownership.
- No run-level completion semantics.
- No policy engine ownership.
- No direct persistence of large tool payloads.

### Current-System Reuse Guidance

Likely reusable:

- `MetaPayload.turn` from the existing event payloads as a sequencing hint
- `AIRunEvent.Seq` as a durable ordering primitive
- existing projection ordering and block boundaries that already approximate turn slices

Likely disposable:

- any turn-like state kept only inside iterator loops or goroutine locals
- any prompt-side notion of a turn that is not persisted

## 7. ToolCall

### Responsibility

`ToolCall` is one structured tool invocation record. It answers: what external capability was invoked and what happened?

### Core Field Groups

- Identity: `tool_call_id`, `run_id`, `turn_id`
- Tool contract reference: tool id, tool name, tool version or contract version
- Input snapshot: arguments preview, arguments content id, normalized input shape
- Policy metadata: policy decision, risk level, approval required flag, policy rule reference
- Execution state: pending, running, succeeded, failed, skipped, blocked
- Result snapshot: result preview, result content id, status, result summary
- Timing and observability: started_at, finished_at, duration, error summary, retry count
- Correlation: idempotency key, causality event ids, parent tool call id if a wrapper or nested call exists

### Ownership Boundaries

- Owns one invocation attempt only.
- May reference large input and output blobs, but does not own the tool registry or policy rules.
- Belongs to a turn, but can be queried independently for audit and replay.

### Lifecycle Notes

- Created before execution begins.
- Must not be reused for a second execution attempt after a clear failure or approval boundary change.
- A retry creates a new `ToolCall`.
- Large arguments and large results should be stored as separate content and referenced, not duplicated.

### Non-Responsibilities

- No tool definition ownership.
- No approval decision ownership.
- No prompt rendering ownership.
- No session or task semantics.

### Current-System Reuse Guidance

Likely reusable:

- `AIRunEvent` tool-call and tool-result payloads
- `AIRunContent` for tool arguments and tool outputs
- `ProjectionExecutorItem` and `ProjectionToolResult`
- `tool_call_id`-based indexes and correlation fields

Likely disposable:

- ad hoc tool call state embedded in assistant text
- tool execution bookkeeping that only exists in the service loop
- duplicated tool result previews stored as the only source of truth

## 8. Approval

### Responsibility

`Approval` is one explicit human decision point. It answers: what action needed human approval and what was the decision?

### Core Field Groups

- Identity: `approval_id`, `run_id`, `task_id` if the workflow wants direct task linkage
- Operation snapshot: `tool_call_id`, tool name, arguments preview, preview payload, risk explanation
- Policy snapshot: matched rule id, policy version, decision source, approval required reason
- Resume data: checkpoint id, resume target, target call id, interrupt context ids
- Decision metadata: status, approved by, approved at, rejected reason, comment, decision source
- Expiration and concurrency: timeout, expires_at, lock_expires_at, claim state
- Audit linkage: event ids, outbox id, causality references

### Ownership Boundaries

- Owns the human gate for one specific operation snapshot.
- Freezes the operation preview at request time so later changes do not rewrite the request history.
- Does not own policy evaluation, checkpoint execution, or runtime resume logic.

### Lifecycle Notes

- Typical states are `pending`, `approved`, `rejected`, and `expired`.
- A decision should be write-once for the approval instance.
- If the same operation is proposed again, it should create a new approval record instead of mutating the old one.
- Resume should use the persisted checkpoint and the persisted target binding, not a synthetic fallback.

### Non-Responsibilities

- No policy engine logic.
- No tool execution logic.
- No session summary logic.
- No frontend-only approval state.

### Current-System Reuse Guidance

Likely reusable:

- `AIApprovalTask`
- `AIApprovalOutboxEvent`
- `AICheckpoint`
- existing approval preview payload shape from `EventTypeToolApproval`
- approval expirer and approval worker persistence patterns

Likely disposable:

- approval state that is only implied by UI cards
- fallback resume logic that uses synthetic checkpoints or hidden message metadata
- approval data duplicated only to feed current projection code

## 9. Event

### Responsibility

`Event` is the canonical immutable fact record for runtime behavior. It answers: what did the system observe, decide, execute, and conclude?

### Core Field Groups

- Envelope: `event_id`, `run_id`, `session_id`, `task_id`, `turn_id`, `tool_call_id`, `approval_id`
- Ordering: sequence number, logical ordering key, causal ordering key
- Type and actor: event type, actor type, status, source component
- Payload: compact JSON payload, content ids for large blobs, preview summary
- Causality: caused_by_event_id, parent_event_id, correlation id
- Timing: occurred_at, created_at
- Visibility: optional internal classification only when the runtime needs a narrow non-transport label; this is not a projection hint, transport shape, or SSE concern

### Ownership Boundaries

- Owns the durable fact stream.
- Must be append-only and immutable in the logical sense.
- Read models, projections, and UI blocks may be rebuilt from events, but events themselves do not become projections.

### Lifecycle Notes

- Events are written once.
- Corrections happen by appending a new event, not by editing the original event.
- Event payloads should be small enough to replay efficiently; large evidence should be referenced.
- Causal references should be explicit when one event is the result of another event.

### Non-Responsibilities

- No mutable run state ownership.
- No projection ownership.
- No transport formatting ownership.
- No direct UI semantics.

### Current-System Reuse Guidance

Likely reusable:

- `AIRunEvent`
- `AIApprovalOutboxEvent` as a delivery companion rather than as canonical runtime truth
- `internal/ai/runtime/event_types.go` payload structs and validation logic
- existing ordering by `run_id` and `seq`

Likely disposable:

- event families that only exist in the service layer but are not part of the canonical runtime model
- duplicated event state in UI code
- any event storage path that treats projections as source of truth

## 10. EvaluationCase

### Responsibility

`EvaluationCase` is a reusable regression scenario for route, tool, approval, and outcome quality. It answers: did the runtime behave correctly for this scenario?

### Core Field Groups

- Identity: `case_id`, name, description, tags, version
- Scenario shape: input request, user intent, domain, route expectation, risk expectation
- Fixture references: seed session id, seed task id, seed run id, event fixture ids, content fixture ids
- Assertions: expected route, expected tool family, expected approval behavior, expected outcome, allowed variance
- Judge config: scoring rules, pass threshold, deterministic versus tolerant checks
- Provenance: origin run id, origin report id, origin author or source
- Lifecycle metadata: created_at, updated_at, superseded_by, archived_at

### Ownership Boundaries

- Owns a stable scenario definition, not a live runtime record.
- May reference real run artifacts as fixtures, but should not own production truth.
- Should be able to drive replay-based and transcript-based checks without depending on current UI state.

### Lifecycle Notes

- Can be created from a real run, a curated regression, or a bug report.
- Should be versioned rather than mutated when the expected behavior changes intentionally.
- Can point to reusable fixtures from the current system, then outlive the specific implementation that produced those fixtures.

### Non-Responsibilities

- No live execution ownership.
- No session or run mutation.
- No policy enforcement.
- No production telemetry aggregation by itself.

### Current-System Reuse Guidance

Likely reusable:

- `AIRunProjection`
- `AIRunContent`
- `AIDiagnosisReport`
- `AITraceSpan`
- `AIUsageLog`
- existing run and approval event streams as fixtures

Likely disposable:

- one-off assertions embedded in ad hoc test scripts
- scene-specific prompt fixtures that are not normalized into evaluation cases
- current code paths that manually inspect transcripts instead of reading case definitions

## 11. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-module-redesign-design.md`
- `docs/superpowers/specs/2026-04-08-ai-runtime-kernel-design.md`
- `docs/superpowers/specs/2026-04-08-ai-tool-contract-policy-design.md`
- `docs/superpowers/specs/2026-04-08-ai-event-projection-design.md`
- `docs/superpowers/specs/2026-04-08-ai-evaluation-harness-design.md`
