# AI Run Event Unified Pipeline & Real Checkpoint Resume Design

- Date: 2026-03-27
- Scope: `internal/service/ai/logic`, `internal/ai/common/middleware/approval`, `internal/ai/common/approval/orchestrator`, approval resume flow
- Goal: Remove duplicated event-iteration logic, delete `ReplayThenTail`, and enforce real interrupt checkpoint resume semantics.

## 1. Problem Statement

Current `service/ai/logic` has multiple event traversal paths with overlapping responsibilities:

- Replay/tail loop in `run_tailer.go` (`ReplayThenTail`)
- Waiting-approval terminal reconstruction in `logic.go` (`emitExistingShellTerminal`)
- Resumability credential assembly in `run_resume_projection.go` and related status scans

This duplication causes behavior drift and makes correctness hard to reason about.

A second critical issue is checkpoint semantics for approval resume:

- Approval resume currently relies on `AIApprovalTask.CheckpointID`.
- Upstream metadata initialization sets checkpoint to `run_id` during chat start.
- This can violate the requirement that resume must use the true interrupt checkpoint, not a synthetic fallback.

## 2. Design Goals

1. Establish one authoritative run-event replay/tail pipeline.
2. Remove `ReplayThenTail` completely and switch all callers in one change.
3. Enforce strict checkpoint semantics for approval resume:
   - Resume must use a real interrupt checkpoint.
   - No fallback to `run_id` when checkpoint is missing/invalid.
4. Preserve external SSE event contract (`meta/delta/tool_approval/tool_result/run_state/done/error`).
5. Keep scope focused: no unrelated refactor.

## 3. Non-Goals

- No protocol redesign of SSE event names.
- No changes to approval policy matching behavior.
- No migration that auto-heals historical bad approval tasks.

## 4. Proposed Architecture

### 4.1 New Units

1. `RunEventPipeline`
- Responsibility: single entry for replay + tail + terminal emission.
- Input: `run_id`, cursor (`last_event_id`), attach mode, emitter, timing options.
- Output: ordered SSE stream consistent with persisted run state.

2. `ApprovalPendingResolver`
- Responsibility: derive latest unresolved approvals per `call_id` from persisted run events.
- Rule: latest `tool_approval` for a call remains pending until a matching `tool_result` resolves it.

3. `CheckpointResolver`
- Responsibility: resolve and validate resume checkpoint for approval task.
- Contract: return fatal error if checkpoint is empty, invalid, or non-resumable.

4. `RunStateEmitter`
- Responsibility: centralized emission for terminal/waiting snapshots (`run_state`, `done`, `error`, plus pending approvals when needed).

### 4.2 Removed Unit

- Delete `RunTailer.ReplayThenTail` and all direct invocations.

## 5. Data Flow

### 5.1 Chat Re-attach (`shell.Reused && last_event_id != ""`)

1. `Chat` routes to `RunEventPipeline.Attach(...)`.
2. Pipeline replays events after cursor and records `replay_last_event_id` (last emitted persisted event id).
3. Pipeline loads current run status:
- open status (`running`, `resuming`, `waiting_approval`) => tail phase
- terminal status => emit terminal snapshot and stop
4. For `waiting_approval`, `ApprovalPendingResolver` emits unresolved approvals in deterministic order.
   - deterministic order is `seq` ascending, then `event_id` lexicographical tie-break.
5. Replay/tail handoff contract (anti-gap / anti-duplication):
- tail query starts strictly from `replay_last_event_id`.
- emission is strictly monotonic by persisted ordering key (`seq`, then `id` tie-break).
- events with `id <= replay_last_event_id` are never re-emitted.
- if new events are appended during replay, they are emitted exactly once in tail phase.
6. Cursor expiry contract (`last_event_id` unknown/expired):
- pipeline emits terminal `error` event with machine code `run_cursor_expired`.
- pipeline stops current attach (no implicit full replay fallback).
- client must retry attach with empty cursor (fresh attach) or refresh run projection first.

### 5.2 Approval Resume (Worker)

1. Worker claims decided outbox event and loads task.
2. `CheckpointResolver.Resolve(task)` validates checkpoint hard.
3. On checkpoint failure:
- mark fatal resume failure
- emit structured failure diagnostics
- do not call `ResumeWithParams`
4. On success:
- call `ResumeWithParams` with resolved checkpoint
- persist projected events using existing append/projection paths
- terminalization uses shared `RunStateEmitter` semantics

### 5.3 Checkpoint Source Semantics

- Approval task checkpoint must originate from the actual interrupt checkpoint captured at approval interrupt creation time (the checkpoint carried by the interrupt/resume state, not `run_id`).
- Chat bootstrap metadata checkpoint (`run_id`) is startup metadata only; it is not a valid substitute for approval resume.
- Canonical source-of-truth: `ApprovalEvalMeta.CheckpointID` passed from approval middleware interrupt context to orchestrator persistence.
- Checkpoint acquisition mechanism:
- middleware resolves checkpoint from interrupt runtime context (`resolveInterruptCheckpoint(ctx)` helper backed by ADK interrupt/checkpoint context).
- if helper cannot resolve a checkpoint, approval task creation is rejected.
- `sceneMeta.CheckpointID` is treated as bootstrap metadata only and cannot be used as approval checkpoint input.
- Validation points:
- middleware: before orchestrator `Evaluate`, require resolved interrupt checkpoint present.
- orchestrator: before task `Create`, enforce non-empty and non-`run_id` checkpoint; reject write on violation.
- worker: before `ResumeWithParams`, resolve and validate checkpoint existence/resumability.

## 6. Error Handling

### 6.1 Checkpoint Errors (Hard)

Introduce/standardize a fatal category for invalid checkpoint resume, e.g. `approval_checkpoint_invalid`.

Required behavior:

- fail fast
- persist run status as `failed_runtime`
- append `run_state` event with `status=failed_runtime` and `reason=approval_checkpoint_invalid`
- append `error` event with machine code `approval_checkpoint_invalid` and user-safe message
- include `approval_id`, `run_id`, `checkpoint_id` in structured logs

Compatibility note:
- `reason` is optional additive field on `run_state`; existing required fields remain unchanged.

### 6.2 Event Payload Decode Errors

- For non-critical replay payload decode failures: skip malformed event, log warning with event id/type.
- For critical persistence/append failures: fail current operation.

### 6.3 Tail Shutdown

- Preserve current graceful behavior for context cancel/deadline.

## 7. Interfaces & Boundaries

### 7.1 `RunEventPipeline` Interface (conceptual)

- `Attach(ctx, runID, lastEventID, emit, options) error`

Boundary rules:

- `RunEventPipeline` owns attach lifecycle decisions.
- `Logic.Chat` becomes orchestrator only; no duplicate replay/tail logic.
- `ApprovalPendingResolver` is pure derivation from event sequence.

### 7.2 `CheckpointResolver` Interface (conceptual)

- `ResolveForApproval(ctx, task) (checkpointID string, err error)`

Boundary rules:

- Worker never calls runner resume before successful resolve.
- No fallback logic in worker.
- Non-resumable detection contract:
- source: checkpoint store lookup by checkpoint id + resumable metadata check.
- errors map to:
  - not found / invalid format / metadata mismatch => `approval_checkpoint_invalid`
  - storage transient error => retryable worker error path

## 8. Migration Plan (Code-Level)

1. Implement `RunEventPipeline` and move replay/tail logic there.
2. Move waiting-approval unresolved-approval derivation to `ApprovalPendingResolver`.
3. Switch `Logic.Chat` reattach branch to new pipeline.
4. Delete `run_tailer.go` usage and remove `ReplayThenTail`.
5. Implement `CheckpointResolver` and wire into approval worker resume path.
6. Add strict validations when building/storing approval task checkpoint.
7. Remove any checkpoint fallback behavior.

## 9. Testing Strategy

### 9.1 Unit Tests

- Pipeline replay ordering and cursor semantics.
- Waiting-approval unresolved snapshot derivation (multiple approvals per same call, resolution via tool_result).
- Terminal emission parity (`done/run_state/error`).
- Checkpoint resolver rejects empty/synthetic/nonexistent checkpoint.
- Replay/tail boundary race cases:
- events appended between replay read and tail loop start are emitted exactly once.
- duplicate protection by `event_id` and ordering key.
- Approval interleaving cases:
- `tool_approval(callA) -> tool_approval(callA-v2) -> tool_result(callA)` results in no pending callA.
- mixed calls (`callA`, `callB`) preserve independent resolution.

### 9.2 Integration Tests

- Reattach while run open: replay then tail until new events arrive.
- Reattach when run already terminal.
- Approval accepted path resumes from real checkpoint and completes.
- Approval accepted with invalid checkpoint hard-fails without fallback.
- Concurrent approval sequences:
- interleaved multi-call approval/result events maintain deterministic unresolved snapshot order.
- attach during active approval churn produces monotonic, non-duplicated SSE stream.

### 9.3 Regression Targets

- Existing SSE contract tests in `internal/service/ai/handler/*` and `internal/service/ai/logic/*`.
- Approval worker reliability tests.

## 10. Observability

Add/normalize metrics and logs around unified pipeline:

- `replay_events_total`
- `tail_loops_total`
- `approval_pending_resolved_total`
- `checkpoint_resolve_fail_total`

Structured logging keys:

- `run_id`, `session_id`, `approval_id`, `checkpoint_id`, `cursor`, `event_id`, `event_type`

## 11. SSE Compatibility Matrix

Compatibility target: no breaking removal/rename of existing required fields.

1. `run_state`
- Required: `run_id`, `status`
- Optional additive: `summary`, `agent`, `reason`
- Allowed status values unchanged from existing taxonomy (including `failed_runtime`)

2. `error`
- Required: `message` (existing)
- Optional additive: `code` (`approval_checkpoint_invalid`, `run_cursor_expired`)

3. `tool_approval`
- Required fields unchanged (`approval_id`, `call_id`, `tool_name`, ...)
- Ordering guarantee strengthened only (no schema break)

4. `done`
- Required fields unchanged (`run_id`, `status`)
- Optional fields unchanged (`summary`)

## 12. Risks & Mitigations

1. Risk: one-shot replacement may break subtle ordering assumptions.
- Mitigation: preserve existing event ordering rules; add golden tests for replay sequences.

2. Risk: historical tasks with bad checkpoint become non-resumable.
- Mitigation: explicit operational visibility and manual remediation path; no unsafe fallback.

3. Risk: behavior differences between terminal snapshot and streamed path.
- Mitigation: centralize emission in `RunStateEmitter` and test both entry modes against same expectations.

## 13. Success Criteria

- No remaining production call path uses `ReplayThenTail`.
- Single replay/tail implementation used by chat reattach.
- Approval resume never uses `run_id` as checkpoint fallback.
- Invalid checkpoint scenarios fail deterministically with diagnosable errors.
- Existing SSE contract remains compatible for clients.

## 14. Rollout & Ops

1. Rollout
- behind migration flag (`ApprovalEventMigrationFlags`) for one-shot cutover control.
- staged deploy: canary -> partial -> full.
- rollback trigger: spike in `checkpoint_resolve_fail_total` or attach error rate.

2. Historical invalid task runbook
- detection: query approval tasks where `checkpoint_id` empty/equal run_id/lookup-miss in checkpoint store.
- alerting: emit structured alerts keyed by `approval_id`, `run_id`, `checkpoint_id`.
- remediation: mark task non-resumable, surface operator action, require user re-run from latest message.
