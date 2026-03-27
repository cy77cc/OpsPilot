# AI HITL Stream Resume Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make AI HITL approval pause and resume on the same assistant message by ending the original SSE stream, auto-reconnecting to the same `run_id`, and continuing to tail replayed/new events without creating a new assistant reply.

**Architecture:** Keep Eino’s native `interrupt -> checkpoint -> ResumeWithParams` behavior. The backend owns resumable run state, incremental replay, tail semantics, and explicit retry-resume APIs; the frontend owns message identity, resumable credentials, and automatic reattachment to the same `run_id` after local or cross-device approval changes. Execute this plan in a fresh worktree because the current workspace already has unrelated AI refactor changes.

**Tech Stack:** Go, Gin, GORM, Eino v0.8.4, TypeScript, React, Ant Design X, Vitest

---

## File Structure

### Backend

- Modify: `internal/service/ai/chat/handler.go`
  - Keep `/api/v1/ai/chat` as the single chat/reconnect endpoint.
  - Validate reconnect inputs and stream replay/tail results with stable SSE framing.
- Modify: `internal/service/ai/chat/service.go`
  - Expose any new chat/retry methods from logic without moving business rules into handlers.
- Modify: `internal/service/ai/logic/logic.go`
  - Own reconnect semantics and retire the legacy direct resume path.
- Create: `internal/service/ai/logic/run_tailer.go`
  - Isolate “replay then block waiting for new run events” behavior so it does not bloat `logic.go`.
- Create: `internal/service/ai/logic/run_resume_projection.go`
  - Isolate resumable credential/read-model builders from `logic.go`.
- Modify: `internal/service/ai/logic/approval_worker.go`
  - Emit deterministic `run_state` transitions for resume, retryable resume failures, terminal failures, and successful completion.
- Modify: `internal/service/ai/logic/approval_write_model.go`
  - Persist any extra retry scheduling / resumable-read-model fields needed by history and retry APIs.
- Modify: `internal/service/ai/approval/handler.go`
  - Add explicit retry-resume HTTP entrypoint.
- Modify: `internal/service/ai/approval/service.go`
  - Expose retry-resume use case from logic.
- Modify: `internal/service/ai/routes.go`
  - Register the retry-resume route and keep route contract explicit.
- Modify: `internal/service/ai/handler/chat_test.go`
  - Add reconnect, tail-blocking, resumable credential, and terminal event ordering coverage.
- Modify: `internal/service/ai/handler/approval_test.go`
  - Add retry-resume route and behavior coverage.
- Modify: `internal/service/ai/handler/routes_contract_test.go`
  - Lock route contract for the new retry-resume endpoint.
- Modify: `internal/service/ai/logic/approval_worker_test.go`
  - Cover `resume_failed_retryable -> resuming`, terminal `failed`, and retry ownership semantics.
- Create: `internal/service/ai/logic/run_tailer_test.go`
  - Cover replay-then-tail blocking semantics without going through the full HTTP stack.

### Frontend

- Modify: `web/src/api/modules/ai.ts`
  - Add `run_state` parsing, resumable credential types, and explicit retry-resume API.
- Modify: `web/src/components/AI/types.ts`
  - Extend runtime/status types to represent `waiting_approval`, `resuming`, `resume_failed_retryable`, `failed`, and resumable metadata.
- Modify: `web/src/components/AI/replyRuntime.ts`
  - Make runtime reducers deterministic around `run_state`, `tool_approval`, terminal failure, and retryable resume failure.
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
  - Store reconnect credentials, react to cross-tab approval changes, auto-reattach to the same `run_id`, and stop treating approval pauses as terminal.
- Create: `web/src/components/AI/providers/runReconnectController.ts`
  - Hold reconnect orchestration so `PlatformChatProvider.ts` does not absorb all reconnect/retry branching.
- Create: `web/src/components/AI/pendingRunStore.ts`
  - Keep focused local persistence for `{ runId, clientRequestId, lastEventId, approvalId, status }`.
- Modify: `web/src/components/AI/CopilotSurface.tsx`
  - Rehydrate pending historical runs from server-returned resumable credentials and reattach when appropriate.
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
  - Cover auto-reconnect, cross-device approval notification, retry-resume button flow, and incremental replay.
- Modify: `web/src/components/AI/replyRuntime.test.ts`
  - Cover reducer ordering for `run_state`, `error`, `resume_failed_retryable`, and `failed`.
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
  - Cover `run_state`, strict incremental replay parsing, and retry-resume API types.
- Create: `web/src/components/AI/__tests__/pendingRunStore.test.ts`
  - Cover local persistence/recovery rules.

### Legacy cleanup targets

- Modify: `internal/service/ai/logic/logic.go`
  - Mark `ResumeApproval(...)` as deprecated in the same change that introduces the new main flow.
- Delete or stop referencing: any caller paths that use `Logic.ResumeApproval(...)` directly.
- Modify: `web/src/contexts/NotificationContext.tsx`
  - Keep `ai-approval-updated` only as a side-channel trigger, not the primary data source for resumed content.

## Chunk 1: Backend Run Replay, Tail, and Retry Contracts

Execution order inside this chunk is `Task 1 -> Task 3 -> Task 2 -> Task 4`. The state machine and terminal ordering must converge before replay/tail work starts, otherwise reconnect behavior gets built on unstable lifecycle semantics.

### Task 1: Add failing backend tests for reconnect, tail, and resumable credentials

**Files:**
- Modify: `internal/service/ai/handler/chat_test.go`
- Create: `internal/service/ai/logic/run_tailer_test.go`
- Test: `internal/service/ai/handler/chat_test.go`
- Test: `internal/service/ai/logic/run_tailer_test.go`

- [ ] **Step 1: Write a failing handler test for historical session payloads exposing resumable credentials**

```go
func TestGetSession_IncludesResumableRunCredentialsForWaitingApproval(t *testing.T) {
    // seed session + assistant message + run(status=waiting_approval, client_request_id=req-1)
    // seed last run event id evt-3
    // GET /api/v1/ai/sessions/:id
    // assert assistant message payload includes:
    //   run_id, client_request_id, latest_event_id, approval_id, status, resumable=true
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/ai/handler -run TestGetSession_IncludesResumableRunCredentialsForWaitingApproval -v`
Expected: FAIL because session payload does not expose resumable credentials yet.

- [ ] **Step 3: Write a failing handler test for reconnect replay being strictly incremental**

```go
func TestChatHandler_ReconnectReplaysOnlyEventsAfterLastEventID(t *testing.T) {
    // seed run with events evt-1, evt-2, evt-3
    // POST /api/v1/ai/chat with client_request_id=req-1 and last_event_id=evt-2
    // assert stream emits only evt-3 and newer
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run: `go test ./internal/service/ai/handler -run TestChatHandler_ReconnectReplaysOnlyEventsAfterLastEventID -v`
Expected: FAIL if replay emits duplicate history or returns nothing without strict incrementality.

- [ ] **Step 5: Write a failing logic-level tailer test for “no new events yet, run still resumable”**

```go
func TestRunTailer_WaitsForNewEventsWhenRunStillOpen(t *testing.T) {
    // seed run status=resuming, no events after last_event_id
    // start tailer with short context timeout
    // assert it blocks until a new event is appended rather than exiting immediately
}
```

- [ ] **Step 6: Run the tailer test to verify it fails**

Run: `go test ./internal/service/ai/logic -run TestRunTailer_WaitsForNewEventsWhenRunStillOpen -v`
Expected: FAIL because no dedicated replay-then-tail blocker exists yet.

- [ ] **Step 7: Commit the failing test baseline**

```bash
git add internal/service/ai/handler/chat_test.go internal/service/ai/logic/run_tailer_test.go
git commit -m "test: capture ai hitl reconnect and tail requirements"
```

- [ ] **Step 8: Add a failing worker/write-model test for duplicate `SubmitApproval` idempotency**

```go
func TestSubmitApproval_DuplicateDecisionIsIdempotentAndDoesNotRepeatWrites(t *testing.T) {
    // seed approval + run waiting_approval
    // call SubmitApproval twice with same approval decision payload
    // assert approval row/outbox/run-state side effects are written once
}
```

- [ ] **Step 9: Run the duplicate-submit test to verify it fails**

Run: `go test ./internal/service/ai/logic -run TestSubmitApproval_DuplicateDecisionIsIdempotentAndDoesNotRepeatWrites -v`
Expected: FAIL because duplicate submit idempotency is not locked yet.

- [ ] **Step 10: Amend the failing-test baseline commit if this case was added after Step 7**

```bash
git add internal/service/ai/logic/approval_worker_test.go internal/service/ai/logic/approval_write_model.go
git commit --amend --no-edit
```

### Task 2: Implement resumable credential read-models and replay-then-tail semantics

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Create: `internal/service/ai/logic/run_tailer.go`
- Create: `internal/service/ai/logic/run_resume_projection.go`
- Modify: `internal/service/ai/chat/handler.go`
- Modify: `internal/service/ai/chat/service.go`
- Modify: `internal/service/ai/handler/chat_test.go`
- Modify: `internal/service/ai/logic/run_tailer_test.go`

- [ ] **Step 1: Add a small resumable credential shape in logic for assistant messages and run payloads**

```go
type ResumableRunInfo struct {
    RunID           string `json:"run_id"`
    ClientRequestID string `json:"client_request_id"`
    LatestEventID   string `json:"latest_event_id"`
    ApprovalID      string `json:"approval_id,omitempty"`
    Status          string `json:"status"`
    Resumable       bool   `json:"resumable"`
    CanRetryResume  bool   `json:"can_retry_resume"`
}
```

- [ ] **Step 2: Thread `ResumableRunInfo` into session/get-run payload builders**

```go
item["resumable_run"] = buildResumableRunInfo(run, latestEventID, approvalID)
```

- [ ] **Step 3: Implement `run_tailer.go` with one focused responsibility**

```go
type RunTailer struct {
    RunDAO      *aidao.AIRunDAO
    RunEventDAO *aidao.AIRunEventDAO
}

func (t *RunTailer) ReplayThenTail(ctx context.Context, runID, lastEventID string, emit logic.EventEmitter) error
```

- [ ] **Step 4: Write a failing idle-timeout test for safe repeated reattach**

```go
func TestRunTailer_IdleTimeoutAllowsSafeReattachFromSameCursor(t *testing.T) {
    // first reconnect waits and exits on idle timeout without done/error
    // second reconnect with the same client_request_id + last_event_id
    // receives the next appended event exactly once
}
```

- [ ] **Step 5: Run the idle-timeout test to verify it fails**

Run: `go test ./internal/service/ai/logic -run TestRunTailer_IdleTimeoutAllowsSafeReattachFromSameCursor -v`
Expected: FAIL because idle-timeout semantics are not implemented yet.

- [ ] **Step 6: Add failing tailer tests for client disconnect and absolute tail cutoff**

```go
func TestRunTailer_StopsImmediatelyWhenClientContextCancels(t *testing.T) {
    // start replay-then-tail
    // cancel request context
    // assert tailer returns promptly without waiting for idle timeout
}

func TestRunTailer_AbsoluteTailDeadlineForcesGracefulDisconnect(t *testing.T) {
    // seed run status=resuming with no terminal event
    // assert tailer exits on max tail duration without emitting done/error
}
```

- [ ] **Step 7: Run the disconnect/deadline tests to verify they fail**

Run: `go test ./internal/service/ai/logic -run 'TestRunTailer_(StopsImmediatelyWhenClientContextCancels|AbsoluteTailDeadlineForcesGracefulDisconnect)' -v`
Expected: FAIL because cancellation and absolute tail duration are not implemented yet.

- [ ] **Step 8: Move resumable credential builders into `run_resume_projection.go`**

```go
func BuildResumableRunInfo(run *model.AIRun, latestEventID string, approvalID string) *ResumableRunInfo
```

- [ ] **Step 9: Make `/api/v1/ai/chat` reconnect path call replay-then-tail instead of replay-only**

```go
if shell.Reused && strings.TrimSpace(input.LastEventID) != "" {
    return l.replayThenTail(ctx, shell.Run, input.LastEventID, emit)
}
```

- [ ] **Step 10: Keep tailing while run status is `waiting_approval`, `resuming`, or `running`, and stop only on terminal state, client cancellation, absolute tail deadline, or idle timeout**

```go
func isTailOpenStatus(status string) bool {
    switch status {
    case "waiting_approval", "resuming", "running":
        return true
    default:
        return false
    }
}
```

- [ ] **Step 11: Make `ReplayThenTail(...)` treat `ctx.Done()` as first-class shutdown and keep any subscription/poller scoped to request lifetime**

```go
select {
case <-ctx.Done():
    return ctx.Err()
case evt := <-nextEvent:
    return emit(evt)
}
```

- [ ] **Step 12: Add a max tail duration/heartbeat guard so a hung worker cannot pin the SSE forever**

```go
type TailOptions struct {
    IdleTimeout     time.Duration
    MaxTailDuration time.Duration
}
```

- [ ] **Step 13: Re-run focused backend tests**

Run: `go test ./internal/service/ai/handler -run 'Test(GetSession_IncludesResumableRunCredentialsForWaitingApproval|ChatHandler_ReconnectReplaysOnlyEventsAfterLastEventID)' -v`
Expected: PASS

Run: `go test ./internal/service/ai/logic -run 'TestRunTailer_(WaitsForNewEventsWhenRunStillOpen|IdleTimeoutAllowsSafeReattachFromSameCursor|StopsImmediatelyWhenClientContextCancels|AbsoluteTailDeadlineForcesGracefulDisconnect)' -v`
Expected: PASS

- [ ] **Step 14: Commit the minimal backend replay/tail implementation**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/run_tailer.go internal/service/ai/logic/run_resume_projection.go internal/service/ai/chat/handler.go internal/service/ai/chat/service.go internal/service/ai/handler/chat_test.go internal/service/ai/logic/run_tailer_test.go
git commit -m "feat: add ai run replay and tail semantics"
```

### Task 3: Lock deterministic `run_state` ordering and retryable resume transitions

**Files:**
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_write_model.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/service/ai/handler/chat_test.go`

- [ ] **Step 1: Write failing worker tests for terminal state ordering**

```go
func TestApprovalWorker_EmitsCompletedRunStateBeforeDone(t *testing.T) {}
func TestApprovalWorker_EmitsFailedRunStateBeforeError(t *testing.T) {}
func TestApprovalWorker_LeavesRetryableResumeFailureWithoutDone(t *testing.T) {}
func TestApprovalWorker_RejectAndExpireEndAsCancelledWithoutDoneOrError(t *testing.T) {}
```

- [ ] **Step 2: Run the worker tests to verify they fail**

Run: `go test ./internal/service/ai/logic -run 'TestApprovalWorker_(EmitsCompletedRunStateBeforeDone|EmitsFailedRunStateBeforeError|LeavesRetryableResumeFailureWithoutDone|RejectAndExpireEndAsCancelledWithoutDoneOrError)' -v`
Expected: FAIL because current worker behavior is not fully deterministic around `run_state`.

- [ ] **Step 3: Emit canonical `run_state` transitions from worker code before terminal events**

```go
appendRunEvent(..., "run_state", map[string]any{"status": "resuming"})
appendRunEvent(..., "run_state", map[string]any{"status": "completed"})
appendRunEvent(..., "done", ...)
```

- [ ] **Step 4: Add the `failed` terminal state for non-approval fatal errors**

```go
appendRunEvent(..., "run_state", map[string]any{"status": "failed"})
appendRunEvent(..., "error", ...)
```

- [ ] **Step 5: Persist the approval decision and approver identity to the audit/write model before appending `run_state=resuming`**

```go
tx := db.WithContext(ctx).Begin()
persistApprovalDecision(tx, approvalID, approverID, decision, reason)
appendRunEvent(tx, ..., "run_state", map[string]any{"status": "resuming"})
return tx.Commit().Error
```

- [ ] **Step 6: Add a test that rejects any flow where audit persistence lags behind `resuming`**

```go
func TestSubmitApproval_PersistsAuditRecordBeforeResumingEvent(t *testing.T) {}
```

- [ ] **Step 7: Re-run worker and handler ordering tests**

Run: `go test ./internal/service/ai/logic -run 'TestApprovalWorker_(EmitsCompletedRunStateBeforeDone|EmitsFailedRunStateBeforeError|LeavesRetryableResumeFailureWithoutDone|RejectAndExpireEndAsCancelledWithoutDoneOrError)' -v`
Expected: PASS

Run: `go test ./internal/service/ai/handler -run TestChatHandler_ -v`
Expected: PASS for all existing chat handler tests.

- [ ] **Step 8: Re-run the approval-audit ordering test**

Run: `go test ./internal/service/ai/logic -run 'TestSubmitApproval_PersistsAuditRecordBeforeResumingEvent' -v`
Expected: PASS

- [ ] **Step 9: Commit the ordered state-machine changes**

```bash
git add internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_write_model.go internal/service/ai/logic/approval_worker_test.go internal/service/ai/handler/chat_test.go
git commit -m "feat: order ai run state events deterministically"
```

### Task 4: Add explicit retry-resume API and retire direct legacy resume path

**Files:**
- Modify: `internal/service/ai/approval/handler.go`
- Modify: `internal/service/ai/approval/service.go`
- Modify: `internal/service/ai/routes.go`
- Modify: `internal/service/ai/handler/approval_test.go`
- Modify: `internal/service/ai/handler/routes_contract_test.go`
- Modify: `internal/service/ai/logic/logic.go`
- Test: `internal/service/ai/handler/approval_test.go`
- Test: `internal/service/ai/handler/routes_contract_test.go`

- [ ] **Step 1: Write a failing route contract test for retry-resume**

```go
http.MethodPost + " /api/v1/ai/approvals/:id/retry-resume"
```

- [ ] **Step 2: Run the route contract test to verify it fails**

Run: `go test ./internal/service/ai/handler -run TestRouteContract -v`
Expected: FAIL because the retry-resume route does not exist yet.

- [ ] **Step 3: Write a failing approval handler test**

```go
func TestRetryResumeApproval_RequeuesRetryableRun(t *testing.T) {
    // seed approval task + run(status=resume_failed_retryable)
    // POST /api/v1/ai/approvals/:id/retry-resume
    // assert 200 and outbox/worker scheduling state updated
}
func TestRetryResumeApproval_RejectsTerminalOrAlreadyResumingRuns(t *testing.T) {}
func TestRetryResumeApproval_RejectsDuplicateTriggerIdempotently(t *testing.T) {}
func TestRetryResumeApproval_RejectsUnauthorizedCaller(t *testing.T) {}
```

- [ ] **Step 4: Run the handler test to verify it fails**

Run: `go test ./internal/service/ai/handler -run 'TestRetryResumeApproval_' -v`
Expected: FAIL because retry-resume use case and negative-path handling are missing.

- [ ] **Step 5: Implement the minimal retry-resume handler, service, and logic contract**

```go
func (h *HTTPHandler) RetryResume(c *gin.Context) { ... }
func (s *Service) RetryResume(ctx context.Context, approvalID string, userID uint64) error { ... }
```

- [ ] **Step 6: Lock the retry-resume idempotency contract before wiring the route**

```go
type RetryResumeRequest struct {
    TriggerID string `json:"trigger_id"` // caller-generated idempotency key
}

// Semantics:
// - same approval_id + trigger_id repeated by the same authorized caller returns 200 with the original outcome
// - if the first trigger already moved the run to resuming, repeats do not enqueue a second worker job
// - a new trigger_id against a non-retryable or terminal run returns 409
```

- [ ] **Step 7: Add handler tests that pin repeat-response behavior for the same `trigger_id`**

```go
func TestRetryResumeApproval_RepeatedTriggerIDReturnsOriginalOutcomeWithoutRequeue(t *testing.T) {}
func TestRetryResumeApproval_NewTriggerIDAgainstResumingRunReturnsConflict(t *testing.T) {}
```

- [ ] **Step 8: Search for direct `ResumeApproval(...)` callers and remove or redirect them in this chunk**

Run: `rg -n "ResumeApproval\\(" internal web`
Expected: Only the deprecated logic definition remains, or all direct callers are redirected to the new worker-owned path.

- [ ] **Step 9: Mark `Logic.ResumeApproval(...)` deprecated with a hard guard comment**

```go
// Deprecated: retry and approval recovery must go through SubmitApproval/RetryResume + ApprovalWorker.
```

- [ ] **Step 10: Re-run route and approval tests**

Run: `go test ./internal/service/ai/handler -run 'Test(RouteContract|RetryResumeApproval_)' -v`
Expected: PASS

- [ ] **Step 11: Re-run duplicate approval-submit coverage to confirm write-side idempotency still holds**

Run: `go test ./internal/service/ai/logic -run 'Test(SubmitApproval_DuplicateDecisionIsIdempotentAndDoesNotRepeatWrites)' -v`
Expected: PASS

- [ ] **Step 12: Commit the API contract changes**

```bash
git add internal/service/ai/approval/handler.go internal/service/ai/approval/service.go internal/service/ai/routes.go internal/service/ai/handler/approval_test.go internal/service/ai/handler/routes_contract_test.go internal/service/ai/logic/logic.go
git commit -m "feat: add ai retry resume api"
```

## Chunk 2: Frontend Runtime Identity, Auto-Reconnect, and Cleanup

### Task 5: Add API/event contract tests for `run_state`, resumable credentials, and retry-resume

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.streamChunk.test.ts`
- Modify: `web/src/api/modules/ai.approval.test.ts`
- Test: `web/src/api/modules/ai.streamChunk.test.ts`
- Test: `web/src/api/modules/ai.approval.test.ts`

- [ ] **Step 1: Write a failing stream parser test for `run_state` and resumable credential payloads**

```ts
it('parses run_state events and exposes event ids for reconnect', async () => {
  // event: run_state
  // data: {"run_id":"run-1","status":"waiting_approval"}
})
```

- [ ] **Step 2: Run the parser test to verify it fails**

Run: `npm run test:run -- web/src/api/modules/ai.streamChunk.test.ts`
Expected: FAIL because `run_state` is not parsed into a first-class handler yet.

- [ ] **Step 3: Write a failing approval API test for retry-resume**

```ts
it('calls retry-resume api for retryable runs', async () => {
  await aiApi.retryResumeApproval('approval-1')
  expect(fetchSpy).toHaveBeenCalledWith('/ai/approvals/approval-1/retry-resume', ...)
})
```

- [ ] **Step 4: Run the approval API test to verify it fails**

Run: `npm run test:run -- web/src/api/modules/ai.approval.test.ts`
Expected: FAIL because the client method does not exist yet.

- [ ] **Step 5: Add the minimal API/types surface**

```ts
export interface A2UIRunStateEvent {
  run_id: string;
  status: 'running' | 'waiting_approval' | 'resuming' | 'completed' | 'cancelled' | 'resume_failed_retryable' | 'failed';
  approval_id?: string;
  latest_event_id?: string;
}
```

- [ ] **Step 6: Re-run focused frontend API tests**

Run: `npm run test:run -- web/src/api/modules/ai.streamChunk.test.ts web/src/api/modules/ai.approval.test.ts`
Expected: PASS

- [ ] **Step 7: Commit the client contract additions**

```bash
git add web/src/api/modules/ai.ts web/src/api/modules/ai.streamChunk.test.ts web/src/api/modules/ai.approval.test.ts
git commit -m "feat: add ai run state and retry api client"
```

### Task 6: Add a focused pending-run persistence unit and wire it into the provider

**Files:**
- Create: `web/src/components/AI/pendingRunStore.ts`
- Create: `web/src/components/AI/__tests__/pendingRunStore.test.ts`
- Create: `web/src/components/AI/providers/runReconnectController.ts`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Write failing unit tests for the pending-run store**

```ts
it('stores reconnect credentials by run id', () => {})
it('clears credentials when run reaches a terminal state', () => {})
```

- [ ] **Step 2: Run the store tests to verify they fail**

Run: `npm run test:run -- web/src/components/AI/__tests__/pendingRunStore.test.ts`
Expected: FAIL because the store file does not exist yet.

- [ ] **Step 3: Implement the smallest possible persistence helper**

```ts
export interface PendingRunRecord {
  runId: string;
  clientRequestId: string;
  lastEventId?: string;
  approvalId?: string;
  status: string;
}
```

- [ ] **Step 4: Write a failing provider test for auto-reconnect after approval update**

```ts
it('reattaches to the same run after approval update', async () => {
  // start request -> receive tool_approval -> emit approval update
  // assert second chatStream call uses same sessionId/clientRequestId/lastEventId
})
```

- [ ] **Step 5: Run the provider test to verify it fails**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL because provider does not persist or reuse reconnect credentials.

- [ ] **Step 6: Add a failing provider test for silent terminal replay on stale historical local state**

```ts
it('absorbs terminal replay without flashing resuming when history was locally stale', async () => {
  // local pending store says waiting_approval
  // reconnect immediately replays completed + done from server
  // assert UI settles directly to completed without transient resuming flash
})
```

- [ ] **Step 7: Extract reconnect orchestration into `runReconnectController.ts` and wire the provider to update/read `pendingRunStore` on meta, event ids, tool approval, and run_state**

```ts
pendingRunStore.upsert({ runId, clientRequestId, lastEventId, approvalId, status })
await runReconnectController.reattach(...)
```

- [ ] **Step 8: Re-run the focused provider/store tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/pendingRunStore.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 9: Commit the reconnect persistence wiring**

```bash
git add web/src/components/AI/pendingRunStore.ts web/src/components/AI/__tests__/pendingRunStore.test.ts web/src/components/AI/providers/runReconnectController.ts web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat: persist ai pending run reconnect state"
```

### Task 7: Make runtime reducers deterministic around `run_state`, failure, and retryable resume

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write failing reducer tests for canonical states**

```ts
it('maps waiting_approval into a non-terminal runtime state', () => {})
it('maps failed run_state before terminal error rendering', () => {})
it('keeps resume_failed_retryable actionable without auto-retrying', () => {})
```

- [ ] **Step 2: Run the reducer tests to verify they fail**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts`
Expected: FAIL because reducer logic still depends on mixed event semantics.

- [ ] **Step 3: Extend runtime types minimally**

```ts
export type AssistantReplyStatusKind =
  | 'streaming'
  | 'waiting_approval'
  | 'resuming'
  | 'resume_failed_retryable'
  | 'cancelled'
  | 'failed'
  | 'completed'
```

- [ ] **Step 4: Add focused reducer helpers**

```ts
export function applyRunState(runtime: AssistantReplyRuntime, payload: A2UIRunStateEvent): AssistantReplyRuntime
```

- [ ] **Step 5: Add a bulk replay reducer test to guard historical burst ingestion**

```ts
it('reduces a large incremental replay batch without duplicating content or thrashing status transitions', () => {
  // apply tens/hundreds of replayed delta/tool_result/run_state events
  // assert final runtime/message content is deterministic
})
```

- [ ] **Step 6: Re-run reducer tests**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts`
Expected: PASS

- [ ] **Step 7: Commit the runtime-state cleanup**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat: normalize ai reply runtime states"
```

### Task 8: Rehydrate historical sessions, use retry-resume explicitly, and demote legacy refresh-only behavior

**Files:**
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/contexts/NotificationContext.tsx`
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Write a failing CopilotSurface test for reopening a historical waiting-approval conversation**

```tsx
it('rehydrates reconnect credentials from historical session payloads', async () => {
  // load session with assistant message { resumable_run: ... waiting_approval }
  // assert surface restores pending state instead of treating it as a dead historical message
})
it('immediately reattaches when history opens with resumable resuming state', async () => {
  // load session with assistant message { resumable_run: { status: "resuming", ... } }
  // assert reconnect uses the same runId + clientRequestId and waits for follow-up events
})
it('immediately reattaches when history opens with resumable running state', async () => {
  // load session with assistant message { resumable_run: { status: "running", ... } }
  // assert reconnect uses same runId/clientRequestId/lastEventId without creating a new assistant message
})
it('rehydrates retryable historical runs with an actionable retry control', async () => {
  // load session with assistant message { resumable_run: { status: "resume_failed_retryable", approvalId: "approval-1", ... } }
  // assert UI restores retry affordance instead of silently reconnecting or treating the message as terminal
})
```

- [ ] **Step 2: Run the surface test to verify it fails**

Run: `npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: FAIL because history load does not restore reconnect credentials.

- [ ] **Step 3: Write failing provider tests for explicit retry-resume and signal-only approval updates**

```ts
it('calls retry resume api before reattaching a retryable run', async () => {
  // seed pending run store with status=resume_failed_retryable + approvalId
  // click retry
  // assert aiApi.retryResumeApproval(approvalId) is called first
  // assert subsequent chatStream uses same sessionId/clientRequestId/lastEventId
  // assert no new assistant message is inserted
})
it('uses ai-approval-updated only as a reconnect trigger, not as resumed content', async () => {
  // emit ai-approval-updated
  // assert message content/runtime does not change until chatStream returns replay/tail data
})
```

- [ ] **Step 4: Run the provider test to verify it fails**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL because provider has no explicit retry-resume flow.

- [ ] **Step 5: Add a failing surface/provider test for burst replay from historical reconnect**

```ts
it('handles a large historical replay burst without creating duplicate assistant rows or breaking scroll state', async () => {
  // history load -> reconnect -> server replays many queued events quickly
  // assert one assistant message continues and virtualization/lazy list bookkeeping stays stable
})
```

- [ ] **Step 6: Rehydrate pending run records from history payloads in `CopilotSurface.tsx`**

```ts
if (message.resumable_run?.resumable) {
  pendingRunStore.upsert(...)
}
```

- [ ] **Step 7: Make `NotificationContext.tsx` trigger reconnect side-effects only as a signal**

```ts
window.dispatchEvent(new CustomEvent('ai-approval-updated', { detail: { runId, approvalId } }))
```

- [ ] **Step 8: Use `aiApi.retryResumeApproval(...)` for user-initiated retryable runs**

```ts
await aiApi.retryResumeApproval(approvalId)
```

- [ ] **Step 9: Re-run UI flow tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS

- [ ] **Step 10: Commit the historical rehydrate and cleanup work**

```bash
git add web/src/components/AI/CopilotSurface.tsx web/src/components/AI/providers/PlatformChatProvider.ts web/src/contexts/NotificationContext.tsx web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat: resume ai hitl runs from history"
```

## Execution Notes

- Use a fresh worktree before implementation; do not execute this plan in the current dirty workspace.
- Keep the backend and frontend commits separate so replay/tail contract changes can be reviewed independently.
- Do not remove legacy compatibility shims until the new tests are green and all route/client references are updated.
- Treat reconnect playback as potentially bursty: implementation should batch reducer/application work where possible and avoid per-event DOM churn during large historical replays.

## Suggested Verification Sweep

- [ ] Run: `go test ./internal/service/ai/handler -v`
  Expected: PASS
- [ ] Run: `go test ./internal/service/ai/logic -v`
  Expected: PASS
- [ ] Run: `npm run test:run -- web/src/api/modules/ai.streamChunk.test.ts web/src/api/modules/ai.approval.test.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/pendingRunStore.test.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  Expected: PASS
- [ ] Run: `git status --short`
  Expected: Only intended files changed

Plan complete and saved to `docs/superpowers/plans/2026-03-26-ai-hitl-stream-resume.md`. Ready to execute?
