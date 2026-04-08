# AI Runtime Kernel Design

- Date: 2026-04-08
- Parent: `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- Inherits: `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- Reference: `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`
- Scope: runtime responsibilities, run state machine, turn loop, stop conditions, failure categories, checkpoint and resume semantics, approval transition rules
- Goal: define the durable runtime kernel for `Run`, `Turn`, and approval execution so later implementation can reuse durable data assets and delete control-flow glue

## 1. Inheritance From Blueprint

This document inherits the `Copilot + Runtime` boundary, the event-first truth model, the reuse-and-deletion posture, and the object ownership rules defined by the parent blueprint and the core domain model.

If this document conflicts with the blueprint or the core domain model on object ownership, lifecycle state, or naming, the upstream documents win.

This document does not redefine the durable objects themselves. It defines how the runtime kernel moves those objects through execution.

Explicit rule:

- approval resume is in-run continuation
- in-run plan refinement or reassessment stays in the same run
- retries or materially fresh attempts create new runs
- `replan` is reserved for newly created run attempts, not same-run behavior

## 2. Runtime Responsibilities

The runtime kernel is the execution authority for one verifiable run attempt.

It is responsible for:

1. Owning the `Run` lifecycle from creation to terminal state.
2. Owning the `Turn` loop and deciding when to start, continue, close, or replace a turn.
3. Assembling the execution context from persisted state, events, checkpoints, and current evidence.
4. Making one bounded decision at a time and persisting the result before the next step.
5. Enforcing approval pauses, checkpoint capture, and checkpoint-backed resume.
6. Emitting canonical runtime facts into the event stream.
7. Persisting durable progress markers that allow reconnect, replay, and recovery.
8. Finalizing terminal results and failure metadata.

Optional projections are read models for replay, inspection, and operator views.

They are never required inputs to runtime context assembly.

It is not responsible for:

1. Owning session continuity.
2. Owning task semantics or business objective definition.
3. Owning the tool registry or policy rules.
4. Owning UI projection state.
5. Owning prompt glue, scene glue, or other transient orchestration code.

### 2.1 Ownership Boundaries

The runtime kernel must keep the object boundaries from the core domain model intact.

| Object | Owns | Does not own |
| --- | --- | --- |
| `Task` | the business objective and evaluation target | approval resume, turn sequencing, tool orchestration |
| `Run` | one verifiable execution attempt | session continuity, task semantics, cross-run history |
| `Turn` | one internal decision round inside a run | task identity, run identity, approval policy, tool contracts |
| `Approval` | one human decision gate for one operation snapshot | policy logic, checkpoint creation, resume execution |

The runtime kernel may create new turns inside the same run when a decision round continues. It must not mutate a completed run into a different attempt.

### 2.2 Router Boundary

The router performs pre-run request classification and decides whether to create a `Task` and `Run`.

Runtime `routing` begins only after an operational run already exists.

Runtime `routing` resolves the effective execution route for that already-created run.

It does not decide whether a run should exist.

### 2.3 Run Boundary Decision Matrix

| Scenario | Same run? | Why | Required runtime action |
| --- | --- | --- | --- |
| Router classifies a request and decides to create a `Task`/`Run` | no, this is pre-run creation | the run does not exist yet | create a new task/run pair before runtime routing begins |
| In-run plan refinement or reassessment after new evidence or tool output | yes | same objective and same attempt identity | keep the same run and open a new turn if needed |
| Approval resume after a human decision | yes | same interrupted attempt | restore the checkpoint and continue the same run |
| Transient retry of a recoverable tool or network action inside the same attempt | yes | the attempt identity has not changed | keep the same run and retry under the same task |
| Route shift that only reflects new evidence and does not change the attempt identity | yes | this is in-run plan refinement, not a fresh attempt | keep the same run and record the new route snapshot on the next turn |
| Retry after a terminal failure, rejection, expiry, or explicit rerun request | no | the previous attempt has ended | create a new run under the same task |
| Materially fresh attempt because the objective, ownership, or execution contract changed | no | the attempt identity changed | create a new run and keep the old run immutable |

Route changes are not run boundaries by themselves.

The boundary is whether the attempt identity stays the same.

## 3. Run State Machine

The runtime kernel uses the run states defined in the core domain model:

- `created`
- `routing`
- `planning`
- `executing`
- `waiting_approval`
- `resuming`
- `completed`
- `failed`
- `cancelled`

State meaning:

1. `created` means the run exists but has not started its execution loop.
2. `routing` means the kernel is resolving the effective execution route for an already-created operational run.
3. `planning` means the kernel is shaping the next decision path before active execution.
4. `executing` means the kernel is actively deciding, calling tools, or producing an answer.
5. `waiting_approval` means execution is paused on a persisted human gate.
6. `resuming` means approval has been accepted and the kernel is restoring the checkpoint.
7. `completed` means the run reached a durable successful terminal outcome.
8. `failed` means the run stopped because it could not complete safely or correctly.
9. `cancelled` means the run stopped because execution was explicitly abandoned or interrupted by a user or operator action.

### 3.1 State Transition Table

| From | Trigger | To | Persist Before Transition | Notes |
| --- | --- | --- | --- | --- |
| `created` | kernel claims the already-created operational run and loads initial provenance | `routing` | run record with request and session/task linkage | runtime ownership starts after router creation |
| `routing` | route is resolved and a planning step is required | `planning` | route snapshot, route rationale, initial evidence summary | route fields are observations, not mutation signals |
| `routing` | route is resolved and can execute directly | `executing` | route snapshot, current turn snapshot | allowed for direct answer or direct action paths |
| `planning` | plan is accepted or the plan is reduced to in-run plan refinement | `executing` | plan snapshot, decision summary | planning is a runtime decision phase, not a separate object |
| `planning` | no safe or coherent plan can be produced | `failed` | failure event, error code, failure category | planning failure is terminal only when recovery is impossible |
| `executing` | a policy gate requires human approval | `waiting_approval` | approval snapshot, checkpoint, resume target binding | approval pause is durable and replayable |
| `executing` | the current attempt needs in-run plan refinement after new evidence | `planning` | evidence snapshot, refinement reason | do not use this for materially fresh attempts |
| `executing` | the answer or action is accepted as final | `completed` | final outcome summary, terminal event | finalization must be persisted before leaving the state |
| `executing` | unrecoverable runtime, tool, context, or persistence failure occurs | `failed` | failure event, error details, failure category | use the narrowest accurate category |
| `waiting_approval` | approval is accepted and the target binding validates | `resuming` | approval decision, checkpoint id, call target id | approval resume is continuation, not a new run |
| `waiting_approval` | approval is rejected, expires, or becomes non-resumable | `failed` | approval decision or timeout event, error code | rejection is a terminal outcome for this run |
| `waiting_approval` | explicit user/operator cancel occurs | `cancelled` | cancel event and reason | cancellation is separate from rejection |
| `resuming` | checkpoint restore succeeds | `executing` | resume event, restored checkpoint id | the same run continues after restore |
| `resuming` | checkpoint restore fails | `failed` | failure event, checkpoint error code | use `approval_checkpoint_invalid` for invalid approval resume checkpoints |
| `created`/`routing`/`planning`/`executing`/`waiting_approval`/`resuming` | explicit cancel or shutdown path is requested | `cancelled` | cancel event and stop reason | open states may terminate from a stop request |

Route and domain fields on a turn are persisted as observations of the effective route at that point in time.

They are not mutation signals.

The kernel must not treat an updated turn route snapshot as a command to rewrite the previous route decision.

A route shift that stays inside the same attempt identity is in-run plan refinement and stays in the same run.

A route shift that changes the attempt identity, ownership, or objective creates a new run.

## 4. Turn Loop

`Turn` is one internal decision round inside a run.

The runtime kernel should process each turn as a small observe-decide-act-record loop:

1. Load the current run, task, prior turn, and relevant events.
2. Create a new turn if the current turn is closed or if the kernel is entering a new decision round.
3. Assemble the context from durable state, checkpoints, unresolved approvals, and evidence.
4. Observe the current effective route, open questions, and pending tool or approval state.
5. Decide the next action for this round.
6. Execute one bounded action.
7. Persist the turn phase transition and any new runtime facts.
8. Continue the same turn, pause on approval, or close the turn.

Recommended turn phase flow:

- `context_ready`
- `model_decided`
- `tool_selected`
- `tool_running`
- `tool_finished`
- `awaiting_human`
- `answering`
- `turn_done`

The kernel should persist a turn boundary whenever the decision round changes materially.

That boundary makes replay, inspection, and failure diagnosis easier.

### 4.1 Turn Loop Rules

1. One turn should capture one coherent decision round.
2. Approval wait and approval resume stay inside the same run.
3. A resumed approval should continue the interrupted turn unless the runtime has explicitly ended that decision round.
4. A route change that merely reflects new evidence stays in the same run and may be represented by a new turn.
5. A route change that changes the attempt identity, ownership, or objective must create a new run.
6. Do not store live turn ownership only in goroutine locals or prompt text.

## 5. Stop Conditions

The runtime kernel stops when one of the following conditions becomes true:

1. The objective has been satisfied and the final answer or action has been durably recorded.
2. The user or operator explicitly cancels the run.
3. A policy decision denies the current action and no safe alternate path remains.
4. Approval is rejected, expires, or cannot be resumed safely.
5. The kernel cannot restore the required checkpoint.
6. Tool execution fails in a way that cannot be retried safely inside the same run.
7. Context assembly fails and no safe fallback context exists.
8. Persistence fails and the runtime can no longer guarantee durable correctness.
9. The runtime hits an execution deadline or shutdown condition and cannot continue safely.

Stop conditions map to terminal run states as follows:

| Stop condition | Default terminal state |
| --- | --- |
| successful completion | `completed` |
| explicit cancellation | `cancelled` |
| non-recoverable execution failure | `failed` |
| invalid or non-resumable approval checkpoint | `failed` |
| approval rejection or expiry | `failed` |

The kernel should not keep a run open after it knows the current attempt cannot finish correctly.

## 6. Failure Categories

Failure categories are metadata on a failed or cancelled run.

They are not additional run states.

| Category | Typical origin | Terminal state | Notes |
| --- | --- | --- | --- |
| `routing_failure` | request classification or route selection | `failed` | no stable route could be established |
| `context_failure` | context loading or evidence assembly | `failed` | the kernel could not build a safe decision context |
| `policy_blocked` | policy evaluation | `failed` | the action is blocked and no alternate path is safe |
| `tool_failure` | tool invocation or tool result handling | `failed` | includes tool errors, tool timeouts, and tool contract violations |
| `approval_rejected` | human rejects the approval request | `failed` | ordinary rejection is terminal for the current run |
| `approval_expired` | human gate times out | `failed` | approval no longer represents a valid live decision |
| `approval_checkpoint_invalid` | approval resume checkpoint is missing, synthetic, stale, or non-resumable | `failed` | approval resume must never fall back to `run_id` |
| `persistence_failure` | event append, checkpoint write, or state write fails | `failed` | the runtime cannot guarantee durable truth |
| `runtime_internal_error` | unexpected bug or invariant break | `failed` | use only when no narrower category fits |
| `cancelled_by_user` | explicit user or operator stop | `cancelled` | this is not a failure of the attempt |

The kernel should prefer the narrowest stable category that explains the stop.

If a run can safely continue after a recoverable error, it should stay open and emit the error as a non-terminal event instead of converting the run into a failure.

### 6.1 Policy Decision Transitions

Policy evaluation happens before the kernel commits to a side-effecting tool action.

It determines whether the planned `ToolCall` executes, runs in dry-run mode, pauses for approval, or is blocked.

| Decision | Next run state | Next turn phase | Persisted facts | Does the `ToolCall` execute? |
| --- | --- | --- | --- | --- |
| `allow` | `executing` | `tool_selected` -> `tool_running` -> `tool_finished` | policy allow record, tool call start/result, timing, output summary | yes |
| `dry_run_only` | `executing` | `tool_selected` -> `tool_running` -> `tool_finished` | policy dry-run record, dry-run flag, preview result, no-side-effect marker | yes, but only in dry-run/preview mode |
| `require_approval` | `waiting_approval` | `awaiting_human` | approval snapshot, risk summary, checkpoint, resume target | no until approval resumes the run |
| `deny` | `planning` if a safe alternate path exists, otherwise `failed` | `turn_done` or `planning` | policy denial record, denial reason, blocked tool snapshot, alternate-path note if any | no |

`deny` never executes the planned `ToolCall`.

`dry_run_only` executes the tool in preview mode only; it must not perform the side effect that policy is restricting.

`require_approval` pauses the run and persists the approval gate instead of executing the tool immediately.

If `deny` leaves no safe alternate path, the run fails with the relevant failure category.

## 7. Checkpoint and Resume Semantics

Checkpointing exists so approval and other interrupt points can survive process restart and long idle gaps.

The kernel should treat the checkpoint as a durable resume anchor, not as a convenience field.

### 7.1 Checkpoint Rules

1. A checkpoint must be real and resumable.
2. `run_id` is not a valid synthetic checkpoint fallback.
3. The checkpoint should be captured before entering a durable pause such as approval wait.
4. The checkpoint must be stored with the approval snapshot and the resume target binding.
5. Approval resume should fail fast if the checkpoint cannot be resolved or restored.
6. A checkpoint restore must be idempotent from the perspective of the current run state.

### 7.2 Resume Rules

1. Approval resume is in-run continuation.
2. The resume keeps the same `run_id`, `task_id`, and `session_id`.
3. The resume should continue the interrupted turn unless the runtime has explicitly closed that turn.
4. Approval resume must use the persisted interrupt checkpoint and the persisted target binding.
5. Approval retry after a terminal result, or any materially fresh execution, creates a new run instead of mutating the previous one.

In-run plan refinement stays inside the same run.

This document inherits the runtime reference rule that approval resume is bound to the exact interrupted call target first and to the checkpoint second.

Practical binding priority:

1. `tool_call_id` when available.
2. `approval_id` only when it encodes a call-style target.
3. checkpoint id as the required resume API input, not as the decision target selector.

If the runtime cannot validate the target binding or the checkpoint, it must stop the resume path and mark the attempt as failed with `approval_checkpoint_invalid`.

## 8. Approval Transition Rules

Approval is a runtime transition, not a UI-only concept.

The approval snapshot freezes the operation that needs human review.

The runtime kernel must not rewrite that snapshot after the approval request has been persisted.

| Condition | Kernel action | Next state | Persisted facts |
| --- | --- | --- | --- |
| policy evaluation returns `require_approval` | persist approval snapshot and checkpoint | `waiting_approval` | approval request, risk summary, resume target, checkpoint id |
| human approves | validate checkpoint and target binding | `resuming` | approval decision, approver metadata, comment, timestamp |
| checkpoint restore succeeds after approval | continue the same run | `executing` | resume event and restored checkpoint id |
| human rejects | close the current attempt | `failed` | rejection reason and approver metadata |
| approval expires | close the current attempt | `failed` | timeout or expiry event |
| approval target no longer matches the interrupted call | stop the resume path | `failed` | `approval_checkpoint_invalid` and diagnostic fields |

Approval decisions apply only to the exact approval snapshot that was persisted for that run and turn.

An approval decision must not mutate a different run, a different task, or a different interrupted call.

## 9. Current System Reuse Guidance

The runtime kernel should reuse durable assets that already behave like durable truth.

Likely reusable:

1. `AIRun`
2. `AIRunEvent`
3. `AICheckpoint`
4. `AIRunProjection`
5. `AIApprovalTask`
6. `AIApprovalOutboxEvent`
7. `AITraceSpan`
8. `AIUsageLog`
9. Existing event ordering and checkpoint storage patterns
10. Existing approval preview payloads when they already encode the real interrupted operation

Likely disposable:

1. Duplicated iterator loops that only exist to bridge replay and tail behavior.
2. Scene-based prompt augmentation that encodes control flow instead of data.
3. Chat-service glue that owns routing, approval, and runtime progression in the same method.
4. Synthetic checkpoint fallback logic such as using `run_id` as a resume anchor.
5. Hidden state encoded in assistant text or UI projection state.
6. Goroutine-local turn state that is not persisted as a durable runtime fact.
7. Any control-flow shim that only exists because the old runtime did not have explicit `Task`, `Run`, `Turn`, and `Approval` ownership.

Repository note:

- durable data assets may be reused
- control-flow glue should not be preserved

The practical rule is to keep the durable objects, event tables, checkpoints, and evaluation fixtures if they still fit the target shape, while deleting the orchestration code that merely shuttled between old abstractions.

## 10. Related Specs

- `docs/superpowers/specs/2026-04-08-ai-module-blueprint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-core-domain-model-design.md`
- `docs/superpowers/specs/2026-03-27-ai-run-event-pipeline-checkpoint-design.md`
- `docs/superpowers/specs/2026-04-08-ai-module-redesign-design.md`
