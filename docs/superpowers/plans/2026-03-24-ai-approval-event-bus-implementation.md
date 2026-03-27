# AI Approval Event Bus Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a fully event-driven approval and resume pipeline with `<1s` approval visibility, optimistic frontend updates, and deterministic post-approval recovery states.

**Architecture:** Introduce a unified event envelope over DB outbox + in-process pub/sub, drive approval/run state transitions through explicit domain events, and project those events into SSE and UI state machines. Keep compatibility during migration by dual-read/dual-write gates, then converge to the new event path.

**Tech Stack:** Go (Gin, GORM), existing AI runtime/event pipeline, TypeScript + React + Vitest, SSE, DB outbox.

---

## File Structure (Locked Before Tasks)

### Backend domain/event contracts
- Create: `internal/service/ai/logic/approval_event_contract.go`
  - Responsibility: single source for envelope fields (`event_id`, `event_type`, `occurred_at`, `sequence`, `aggregate_id`, etc.) and typed payload builders.
- Create: `internal/service/ai/logic/approval_event_contract_test.go`
  - Responsibility: serialization, required-field, and versioning coverage.

### Backend write model + outbox
- Create: `internal/service/ai/logic/approval_write_model.go`
  - Responsibility: extract approval status transitions, CAS checks, and outbox dual-write orchestration from large files.
- Create: `internal/service/ai/logic/approval_write_model_test.go`
  - Responsibility: idempotency, atomicity, and transition correctness.
- Create: `storage/migration/20260324160000_add_approval_outbox_event_envelope_fields.sql`
  - Responsibility: add `event_id`, `sequence`, `aggregate_id`, and related indexes to `ai_approval_outbox_events`.
- Modify: `internal/service/ai/logic/logic.go`
  - Responsibility: slim orchestration and API mapping; delegate transition logic to `approval_write_model.go`.
- Modify: `internal/service/ai/logic/approval_worker.go`
  - Responsibility: emit `ai.run.resuming/resumed/resume_failed/completed` events with deterministic `run_id`-scoped sequence usage.
- Modify: `internal/dao/ai/approval_outbox_dao.go`
  - Responsibility: claim/retry APIs needed by dispatcher.
- Modify: `internal/model/ai.go`
  - Responsibility: outbox schema fields required by unified envelope (`event_id`, `sequence`, `aggregate_id`, etc.) and indexes.
- Create: `internal/service/ai/logic/approval_expirer.go`
  - Responsibility: `pending -> expired` scan and `ai.approval.expired` event emission.
- Create: `internal/service/ai/logic/approval_expirer_test.go`

### Dispatcher + SSE bridge
- Create: `internal/service/ai/logic/approval_event_dispatcher.go`
  - Responsibility: outbox -> in-process bus publish + done/retry bookkeeping.
- Create: `internal/service/ai/logic/approval_event_bus.go`
  - Responsibility: publish/subscribe abstraction for same-process consumers.
- Create: `internal/service/ai/logic/approval_event_bus_test.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
  - Responsibility: dispatcher integration assertions for done/retry and at-least-once behavior.
- Modify: `internal/dao/ai/run_event_dao.go`
  - Responsibility: replay query by `last_event_id` for SSE cursor recovery.
- Modify: `internal/dao/ai/run_event_dao_test.go`
  - Responsibility: replay query ordering and gap coverage.
- Modify: `internal/service/ai/handler/chat.go`
  - Responsibility: support initial snapshot + `last_event_id` replay + live stream handoff.
- Modify: `internal/service/ai/handler/sse_writer.go`
  - Responsibility: event id write support and replay-safe output formatting.
- Modify: `internal/service/ai/handler/sse_writer_test.go`
- Modify: `internal/service/ai/handler/chat_test.go`
  - Responsibility: cursor/snapshot/replay contract tests.
- Modify: `internal/service/ai/routes.go`
  - Responsibility: register SSE bridge subscriptions (`ai.approval.*`, `ai.run.*`) during AI service startup.

### Frontend state machine + stream consume
- Modify: `web/src/api/modules/ai.ts`
  - Responsibility: SSE parse path support for `id:` field + `last_event_id` request parameter + new event types.
- Modify: `web/src/components/AI/replyRuntime.ts`
  - Responsibility: deterministic states (`approved_resuming`, `approved_retrying`, `approved_failed_terminal`, `approved_done`, `expired`).
- Modify: `web/src/components/AI/AssistantReply.tsx`
  - Responsibility: optimistic submit fallback policy and eventual consistency reconciliation.
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
  - Responsibility: merge snapshot/replay/live events into one runtime stream.
- Modify tests:
  - `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
  - `web/src/components/AI/__tests__/AssistantReply.test.tsx`
  - `web/src/components/AI/replyRuntime.test.ts`
  - `web/src/api/modules/ai.streamChunk.test.ts`

### Observability + migration controls
- Create: `internal/service/ai/logic/approval_event_metrics.go`
  - Responsibility: latency/retry/lag metrics emit points.
- Modify: `internal/service/ai/routes.go`
  - Responsibility: wire dispatcher/expirer lifecycle startup.
- Create: `internal/service/ai/logic/approval_event_migration_flags.go`
  - Responsibility: dual-write/read feature flags and staged cutover guard.

### Legacy cleanup targets (post-cutover only)
- Modify: `internal/service/ai/logic/logic.go`
  - Responsibility: remove legacy approval replay fallback branches once unified events are the only read path.
- Modify: `internal/service/ai/logic/tool_error_classifier.go`
  - Responsibility: delete transitional interrupt payload backfill paths that only served legacy interrupt shapes.
- Modify: `internal/service/ai/logic/approval_worker.go`
  - Responsibility: remove legacy event payload compatibility emission.
- Modify: `web/src/api/modules/ai.ts`
  - Responsibility: remove legacy stream event normalization branches no longer needed after cutover.
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
  - Responsibility: remove old-event fallback consume path when migration flag disables legacy mode.
- Modify: `web/src/components/AI/replyRuntime.ts`
  - Responsibility: remove deprecated approval state mapping for legacy event names.
- Modify tests:
  - `internal/service/ai/logic/logic_test.go`
  - `internal/service/ai/logic/tool_error_classifier_test.go`
  - `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
  - `web/src/components/AI/replyRuntime.test.ts`

## Chunk 1: Write Model + Event Contract

### Task 1: Add unified approval event contract

**Files:**
- Create: `internal/service/ai/logic/approval_event_contract.go`
- Test: `internal/service/ai/logic/approval_event_contract_test.go`

- [ ] **Step 1: Write failing contract tests (@superpowers:test-driven-development)**

```go
func TestApprovalEnvelope_RequiresCoreFields(t *testing.T)
func TestApprovalEnvelope_SequenceMonotonicPerRunID(t *testing.T)
func TestApprovalPayloadBuilders_EmitStableEventTypes(t *testing.T)
```

- [ ] **Step 2: Run tests to verify failure**
Run: `go test ./internal/service/ai/logic -run 'ApprovalEnvelope|ApprovalPayloadBuilders'`
Expected: FAIL with missing types/functions.

- [ ] **Step 3: Implement minimal contract + payload builders**

```go
type ApprovalEventEnvelope struct {
  EventID     string
  EventType   string
  OccurredAt  time.Time
  Sequence    int64
  Version     int
  RunID       string
  SessionID   string
  ApprovalID  string
  ToolCallID  string
  AggregateID string
  PayloadJSON string
}
func NewApprovalRequestedEnvelope(input ApprovalRequestedInput) (*ApprovalEventEnvelope, error)
func NewApprovalDecidedEnvelope(input ApprovalDecidedInput) (*ApprovalEventEnvelope, error)
func NewApprovalExpiredEnvelope(input ApprovalExpiredInput) (*ApprovalEventEnvelope, error)
func NewRunResumingEnvelope(input RunResumingInput) (*ApprovalEventEnvelope, error)
func NewRunResumedEnvelope(input RunResumedInput) (*ApprovalEventEnvelope, error)
func NewRunResumeFailedEnvelope(input RunResumeFailedInput) (*ApprovalEventEnvelope, error)
func NewRunCompletedEnvelope(input RunCompletedInput) (*ApprovalEventEnvelope, error)
```

- [ ] **Step 4: Re-run targeted tests**
Run: `go test ./internal/service/ai/logic -run 'ApprovalEnvelope|ApprovalPayloadBuilders'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/approval_event_contract.go internal/service/ai/logic/approval_event_contract_test.go
git commit -m "feat(ai): add unified approval event envelope contract"
```

### Task 2: Align approval submit/resume write model with new events

**Files:**
- Create: `internal/service/ai/logic/approval_write_model.go`
- Test: `internal/service/ai/logic/approval_write_model_test.go`
- Create: `storage/migration/20260324160000_add_approval_outbox_event_envelope_fields.sql`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/model/ai.go`
- Modify: `internal/dao/ai/approval_outbox_dao.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Add failing tests for event emission on state transitions**
Add cases covering:
- submit approved -> emits `ai.approval.decided`
- duplicate submit on same `approval_id` -> no second transition, returns snapshot
- CAS conflict (non-pending status or locked lease) -> write denied and status unchanged
- worker lease acquired -> emits `ai.run.resuming`
- resume success -> emits `ai.run.resumed` and `ai.run.completed`
- retryable failure -> emits `ai.run.resume_failed` with `retryable=true`
- state transition + outbox insert atomicity (either both commit, or both rollback)
- lease renewal keeps ownership for long-running resume and allows takeover after expiry

- [ ] **Step 2: Run failing tests**
Run: `go test ./internal/service/ai/logic -run 'SubmitApproval|ApprovalWorker'`
Expected: FAIL with event mismatch.

- [ ] **Step 3: Implement minimal transitions + outbox writes**
- Keep existing lock semantics.
- Replace ad-hoc event JSON with envelope builder outputs.
- Implement sequence allocation with DB locking (chosen approach):
- `SELECT COALESCE(MAX(sequence), 0) + 1 FROM ai_approval_outbox_events WHERE run_id = ? FOR UPDATE` inside the same write transaction.
- Persist allocated value in outbox row so multi-instance workers observe the same order.
- For `approval_id` projections, order events by `occurred_at + sequence` (while `sequence` generation remains `run_id`-scoped monotonic).
- Move transition/CAS logic from `logic.go` into `approval_write_model.go` to keep file size controlled.
- Add/verify lease renewal for long-running resume tasks (`approved/resuming` path), with takeover after lease expiry.

- [ ] **Step 4: Verify tests pass**
Run: `go test ./internal/service/ai/logic -run 'SubmitApproval|ApprovalWorker'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add storage/migration/20260324160000_add_approval_outbox_event_envelope_fields.sql internal/service/ai/logic/approval_write_model.go internal/service/ai/logic/approval_write_model_test.go internal/service/ai/logic/logic.go internal/service/ai/logic/approval_worker.go internal/model/ai.go internal/dao/ai/approval_outbox_dao.go internal/service/ai/logic/approval_worker_test.go
git commit -m "feat(ai): extract approval write model and emit unified events"
```

## Chunk 2: Dispatcher + Expiration + Replay-Safe SSE

### Task 3: Build outbox dispatcher and in-process event bus

**Files:**
- Create: `internal/service/ai/logic/approval_event_bus.go`
- Create: `internal/service/ai/logic/approval_event_dispatcher.go`
- Test: `internal/service/ai/logic/approval_event_bus_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go` (dispatcher integration cases)

- [ ] **Step 1: Write failing tests for publish/subscribe + retry bookkeeping**

```go
func TestApprovalEventBus_PreservesPerAggregateSequence(t *testing.T)
func TestApprovalEventDispatcher_MarkDoneOnSuccess(t *testing.T)
func TestApprovalEventDispatcher_MarkRetryOnFailure(t *testing.T)
```

- [ ] **Step 2: Run failing tests**
Run: `go test ./internal/service/ai/logic -run 'ApprovalEventBus|ApprovalEventDispatcher'`
Expected: FAIL (missing dispatcher/bus).

- [ ] **Step 3: Implement minimal bus and dispatcher**
- `Publish(event)` fan-out to subscribers.
- dispatcher claim pending outbox -> publish -> mark done/retry.
- keep at-least-once semantics.

- [ ] **Step 4: Run tests**
Run: `go test ./internal/service/ai/logic -run 'ApprovalEventBus|ApprovalEventDispatcher'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/approval_event_bus.go internal/service/ai/logic/approval_event_dispatcher.go internal/service/ai/logic/approval_event_bus_test.go internal/service/ai/logic/approval_worker_test.go
git commit -m "feat(ai): add approval event dispatcher and in-process bus"
```

### Task 4: Add `last_event_id` replay contract to SSE layer

**Files:**
- Modify: `internal/dao/ai/run_event_dao.go`
- Modify: `internal/dao/ai/run_event_dao_test.go`
- Modify: `internal/service/ai/handler/chat.go`
- Modify: `internal/service/ai/handler/sse_writer.go`
- Modify: `internal/service/ai/handler/sse_writer_test.go`
- Modify: `internal/service/ai/handler/chat_test.go`

- [ ] **Step 1: Write failing handler tests**
Cases:
- no cursor -> snapshot first then live stream
- with `last_event_id` -> replay gap then live
- expired cursor -> returns `AI_STREAM_CURSOR_EXPIRED` contract

- [ ] **Step 2: Run tests and confirm fail**
Run: `go test ./internal/service/ai/handler -run 'Chat|SSE|Cursor'`
Expected: FAIL with missing cursor behavior.

- [ ] **Step 3: Implement minimal replay-safe SSE behavior**
- parse `last_event_id` from request.
- write `id:` lines in SSE output.
- query/replay buffered events from `run_event_dao` (`ListAfterEventID` style method) then attach live subscriber.

- [ ] **Step 4: Re-run tests**
Run: `go test ./internal/service/ai/handler -run 'Chat|SSE|Cursor'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/dao/ai/run_event_dao.go internal/dao/ai/run_event_dao_test.go internal/service/ai/handler/chat.go internal/service/ai/handler/sse_writer.go internal/service/ai/handler/sse_writer_test.go internal/service/ai/handler/chat_test.go
git commit -m "feat(ai): add last_event_id replay to SSE approval stream"
```

### Task 7: Implement expiration event path and operational metrics (moved earlier for dependency alignment)

**Files:**
- Create: `internal/service/ai/logic/approval_expirer.go`
- Create: `internal/service/ai/logic/approval_expirer_test.go`
- Create: `internal/service/ai/logic/approval_event_metrics.go`
- Modify: `internal/service/ai/routes.go`

- [ ] **Step 1: Add failing tests for expiration and event emission**
Run expiry scanner tests where pending approvals cross `expires_at`, including:
- `pending -> expired` and `ai.approval.expired` outbox write in one transaction.
- forced outbox write failure triggers full rollback (status remains `pending`).

- [ ] **Step 2: Run tests and confirm fail**
Run: `go test ./internal/service/ai/logic -run 'ApprovalExpirer|Expired|Lease'`
Expected: FAIL.

- [ ] **Step 3: Implement minimal expirer + metrics emit points**
- emit `ai.approval.expired`.
- register SSE bridge subscriptions at startup:
- `bus.Subscribe("ai.approval.*", sseBridgeHandler)`
- `bus.Subscribe("ai.run.*", sseBridgeHandler)`
- record required metrics: `approval_request_to_visible_latency_ms`, `approval_decision_to_resuming_latency_ms`, `resume_success_rate`, `resume_retry_count`, `outbox_lag_seconds`.
- optional follow-up metrics (only if time allows in same PR): `sse_replay_success_rate`, `sse_cursor_expired_count`, `approval_conflict_count`.

- [ ] **Step 4: Re-run tests**
Run: `go test ./internal/service/ai/logic -run 'ApprovalExpirer|Expired|Lease'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/approval_expirer.go internal/service/ai/logic/approval_expirer_test.go internal/service/ai/logic/approval_event_metrics.go internal/service/ai/routes.go
git commit -m "feat(ai): add expiration path, metrics, and SSE bridge subscriptions"
```

## Chunk 3: Frontend Optimistic State Machine

### Task 5: Extend stream protocol and runtime status transitions

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Test (existing): `web/src/api/modules/ai.streamChunk.test.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Add failing tests for new states/events**
Cover:
- parse SSE `id:` and new event types
- `resume_failed(retryable=true)` -> `approved_retrying`
- `resume_failed(retryable=false)` -> `approved_failed_terminal`
- `run_resuming` -> `approved_resuming`
- `run_resumed|completed` -> `approved_done`

- [ ] **Step 2: Run tests and confirm fail**
Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/components/AI/replyRuntime.test.ts`
Expected: FAIL with unsupported states/events.

- [ ] **Step 3: Implement minimal parser/runtime updates**
- add event handlers and type definitions.
- add terminal-failure mapping for `resume_failed(retryable=false|max_retry_exceeded=true)` -> `approved_failed_terminal`.
- preserve backward compatibility for legacy `tool_approval` flow.

- [ ] **Step 4: Re-run tests**
Run: `npm run test:run -- src/api/modules/ai.streamChunk.test.ts src/components/AI/replyRuntime.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.streamChunk.test.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat(web): support unified approval resume event states"
```

### Task 6: Implement optimistic submit fallback + eventual convergence

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Add failing UI behavior tests**
Cases:
- approve click -> immediate `approved_resuming`
- submit timeout/5xx/network -> 3s then “结果确认中” and keep converging
- conflict response overrides optimistic state

- [ ] **Step 2: Run failing tests**
Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL on optimistic + fallback assertions.

- [ ] **Step 3: Implement minimal UI convergence logic**
- centralize transient state timeout policy.
- never transition back to `waiting_approval` after approved.

- [ ] **Step 4: Re-run tests**
Run: `npm run test:run -- src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/AssistantReply.test.tsx web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat(web): add optimistic approval-resume convergence policy"
```

## Chunk 4: Migration Gates + Legacy Cleanup + End-to-End Regression

### Task 8: Add migration flags and full regression suite

**Files:**
- Create: `internal/service/ai/logic/approval_event_migration_flags.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/handler/chat.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `web/src/components/AI/historyProjection.test.ts`
- Modify: `web/src/api/modules/ai.approval.test.ts`

- [ ] **Step 1: Write failing migration-gate tests**
- dual-write on, dual-read fallback on.
- cutover on: only unified events active.

- [ ] **Step 2: Run test suites**
Run backend: `go test ./internal/service/ai/logic ./internal/service/ai/handler`
Expected: FAIL before migration-gate implementation.

Run frontend: `npm run test:run -- src/components/AI/historyProjection.test.ts src/api/modules/ai.approval.test.ts`
Expected: FAIL before cutover behavior.

- [ ] **Step 3: Implement minimal feature-flagged migration gates**
- explicit flags for write path and read path.
- wire backend write path gate in `logic.go` and read/replay gate in `chat.go`.
- wire frontend event-consume gate in `ai.ts` + `PlatformChatProvider.ts`.
- ensure rollback switch can restore old behavior safely.

- [ ] **Step 4: Run full targeted regressions**
Run backend:
`go test ./internal/service/ai/logic ./internal/service/ai/handler ./internal/dao/ai`
Expected: PASS.

Run frontend:
`npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/replyRuntime.test.ts src/components/AI/historyProjection.test.ts src/api/modules/ai.approval.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/approval_event_migration_flags.go internal/service/ai/logic/logic.go internal/service/ai/handler/chat.go web/src/api/modules/ai.ts web/src/components/AI/providers/PlatformChatProvider.ts internal/service/ai/logic/logic_test.go internal/service/ai/logic/approval_worker_test.go web/src/components/AI/historyProjection.test.ts web/src/api/modules/ai.approval.test.ts
git commit -m "feat(ai): add migration gates and approval event-bus regression coverage"
```

### Task 9: Remove legacy approval pipeline code after cutover

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/tool_error_classifier.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Test: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/tool_error_classifier_test.go`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Add failing tests that assert legacy paths are disabled**
Cases:
- with migration cutover enabled, legacy fallback branches are not executed.
- legacy-only payload shapes are rejected or ignored according to new contract.

- [ ] **Step 2: Run tests to verify failure**
Run backend: `go test ./internal/service/ai/logic -run 'Legacy|Cutover|Approval'`
Expected: FAIL before cleanup.

Run frontend: `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/replyRuntime.test.ts`
Expected: FAIL before cleanup.

- [ ] **Step 3: Delete legacy branches and dead code**
- remove old replay/normalization compatibility code paths guarded for migration only.
- remove obsolete helper functions/types used exclusively by old event protocol.
- keep rollback safety by preserving feature flags until this task passes and is deployed.

- [ ] **Step 4: Re-run full targeted regression after cleanup**
Run backend:
`go test ./internal/service/ai/logic ./internal/service/ai/handler ./internal/dao/ai ./internal/ai/runtime`
Expected: PASS.

Run frontend:
`npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/replyRuntime.test.ts src/components/AI/historyProjection.test.ts src/api/modules/ai.approval.test.ts src/api/modules/ai.streamChunk.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/tool_error_classifier.go internal/service/ai/logic/approval_worker.go web/src/api/modules/ai.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/replyRuntime.ts internal/service/ai/logic/logic_test.go internal/service/ai/logic/tool_error_classifier_test.go web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "refactor(ai): remove legacy approval event pipeline after cutover"
```

## Final Verification Checklist

- [ ] Run `go test ./internal/service/ai/logic ./internal/service/ai/handler ./internal/dao/ai ./internal/ai/runtime`
- [ ] Run `npm run test:run -- src/components/AI/__tests__/PlatformChatProvider.test.ts src/components/AI/__tests__/AssistantReply.test.tsx src/components/AI/replyRuntime.test.ts src/components/AI/historyProjection.test.ts src/api/modules/ai.approval.test.ts src/api/modules/ai.streamChunk.test.ts`
- [ ] Manually verify: approval appears `<1s`, approve -> immediate `approved_resuming`, retryable failure -> `approved_retrying`, terminal failure -> `approved_failed_terminal`, success -> `approved_done`.
- [ ] Verify DB migration applied: `SELECT event_id, sequence FROM ai_approval_outbox_events LIMIT 1;`
- [ ] Verify SSE event id format includes `id: <event_id>` line for replay-capable events.
- [ ] Verify multi-instance lease behavior: kill worker mid-resume and confirm takeover after lease expiry.
- [ ] Verify frontend fallback on 5xx: force `submitApproval` 503 and confirm `结果确认中` convergence path.
- [ ] Verify `last_event_id` replay: disconnect SSE for 10s, reconnect with cursor, confirm gap filled.
- [ ] Verify legacy cleanup: grep for deprecated approval event compatibility symbols returns no runtime references in active code paths.
