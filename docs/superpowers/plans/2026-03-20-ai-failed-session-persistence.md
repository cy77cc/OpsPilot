# AI Failed Session Persistence Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every AI chat turn persist a session/message/run shell at request start, preserve partial output on failure, and force both current and historical UI to converge to an error state instead of hanging in loading.

**Architecture:** Move chat execution onto an idempotent, shell-first lifecycle in `logic.Chat`, where `session + user message + assistant placeholder + run` are created before streaming begins. Split failure handling into a critical transaction for `message/run` terminal state and a best-effort second phase for event/projection persistence, then update the web client to treat `error` or broken streams as terminal and render error details separately from markdown body.

**Tech Stack:** Go, Gin, Gorm, SQL migrations, SSE, React, TypeScript, Vitest

---

## File Map

### Backend protocol and persistence

- Modify: `api/ai/v1/ai.go`
  Add `client_request_id` to chat request and expose assistant error/body fields needed by session detail payloads.
- Modify: `internal/model/ai.go`
  Add idempotency and timeout-related fields/indexes needed by `ai_runs` and, if required, assistant message error display.
- Modify: `internal/dao/ai/run_dao.go`
  Add idempotent lookup helpers, terminal-expiry updates, and tighter status transitions.
- Modify: `internal/dao/ai/chat_dao.go`
  Add helpers needed to create/reuse shell messages safely.
- Modify: `internal/service/ai/logic/logic.go`
  Refactor chat lifecycle into shell creation, unified finalize/fail paths, sanitization, and best-effort projection persistence.
- Modify: `internal/service/ai/handler/chat.go`
  Bind/pass `client_request_id`; keep SSE error payloads terminal and sanitized.
- Modify: `internal/service/ai/handler/session.go`
  Return assistant `content` fallback and error fields so history can render without projection.
- Modify: `internal/service/ai/handler/run.go`
  Expose any new run status fields required by the client.
- Create: `storage/migrations/20260320_0003_add_ai_failed_session_persistence.sql`
  Add schema support for idempotency and expiry tracking.
- Modify: `storage/migration/dev_auto.go`
  Keep dev auto-migrate aligned with the model changes.

### Backend tests

- Modify: `internal/service/ai/logic/logic_test.go`
  Cover shell-first creation, start-up failure persistence, partial-output failure persistence, idempotent retries, and projection-write degradation.
- Modify: `internal/service/ai/handler/chat_test.go`
  Cover `client_request_id`, terminal SSE error behavior, and sanitized payloads.
- Modify: `internal/service/ai/handler/session_test.go`
  Cover assistant content/error fallback in session detail/list responses.
- Modify: `internal/dao/ai/run_dao_test.go`
  Cover idempotent lookup/status expiry helpers.
- Modify: `internal/dao/ai/run_storage_migration_test.go`
  Verify new schema fields and indexes exist after migration.

### Expiry recovery

- Create: `internal/service/ai/logic/run_expirer.go`
  Encapsulate stale-run scan and terminal-expiry updates.
- Create: `internal/service/ai/logic/run_expirer_test.go`
  Cover expiry thresholds and message/run synchronization.
- Modify: `internal/service/ai/handler/handler.go`
  Wire new logic dependencies if needed.

### Frontend API and runtime

- Modify: `web/src/components/AI/types.ts`
  Add `clientRequestId` request field and separate assistant error display data.
- Modify: `web/src/api/modules/ai.ts`
  Send `client_request_id`, carry terminal error metadata, and support disconnect fallback.
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
  Treat `error` and abnormal disconnects as terminal; keep body text and error status separate.
- Modify: `web/src/components/AI/replyRuntime.ts`
  Represent terminal error UI without forcing error text into markdown body.
- Modify: `web/src/components/AI/CopilotSurface.tsx`
  Generate request IDs, remove string-concatenated error markdown fallback, and render retry/error affordances.
- Modify: `web/src/components/AI/historyProjection.ts`
  Prefer persisted message body; fall back gracefully when projection is missing.
- Modify: `web/src/components/AI/AssistantReply.tsx`
  Render error callout/footer independent from markdown body.

### Frontend tests

- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
  Cover terminal error convergence, disconnect fallback, and request-id propagation.
- Modify: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`
  Cover error rendering, retry behavior, and history fallback when projection is missing.
- Modify: `web/src/components/AI/historyProjection.test.ts`
  Cover message-content-first hydration for failed runs.
- Modify: `web/src/components/AI/replyRuntime.test.ts`
  Cover separate body/error rendering state.

## Chunk 1: Backend Shell-First Lifecycle

### Task 1: Extend the chat contract for idempotent requests

**Files:**
- Modify: `api/ai/v1/ai.go`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/api/modules/ai.ts`
- Test: `internal/service/ai/handler/chat_test.go`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`

- [ ] **Step 1: Write the failing backend and frontend contract tests**

```go
func TestChatHandler_PassesClientRequestIDIntoLogic(t *testing.T) {
    // send {"message":"hi","client_request_id":"req-1"}
    // assert request is accepted and shell creation path sees req-1
}
```

```ts
it('passes clientRequestId to aiApi.chatStream', async () => {
  request.run({ message: 'hi', scene: 'ai', clientRequestId: 'req-1' });
  expect(aiApi.chatStream).toHaveBeenCalledWith(
    expect.objectContaining({ clientRequestId: 'req-1' }),
    expect.anything(),
    expect.anything(),
  );
});
```

- [ ] **Step 2: Run targeted tests to verify they fail**

Run: `go test ./internal/service/ai/handler -run 'TestChatHandler_PassesClientRequestIDIntoLogic' -count=1`
Expected: FAIL with unknown JSON field or missing assertion

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: FAIL with missing `clientRequestId` propagation

- [ ] **Step 3: Add the request field through API and client types**

```go
type ChatRequest struct {
    SessionID       string `json:"session_id,omitempty"`
    ClientRequestID string `json:"client_request_id,omitempty"`
    Message         string `json:"message"`
    Scene           string `json:"scene,omitempty"`
    Context         any    `json:"context,omitempty"`
}
```

```ts
export interface ChatRequest {
  message: string;
  sessionId?: string;
  clientRequestId?: string;
  scene?: string;
  context?: SceneContext;
}
```

- [ ] **Step 4: Re-run the targeted tests**

Run: `go test ./internal/service/ai/handler -run 'TestChatHandler_PassesClientRequestIDIntoLogic' -count=1`
Expected: PASS

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
Expected: PASS or fail later tests only

- [ ] **Step 5: Commit**

```bash
git add api/ai/v1/ai.go web/src/components/AI/types.ts web/src/api/modules/ai.ts internal/service/ai/handler/chat_test.go web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat(ai): add idempotent chat request id"
```

### Task 2: Add schema and DAO support for idempotent shell creation

**Files:**
- Create: `storage/migrations/20260320_0003_add_ai_failed_session_persistence.sql`
- Modify: `internal/model/ai.go`
- Modify: `storage/migration/dev_auto.go`
- Modify: `internal/dao/ai/run_dao.go`
- Modify: `internal/dao/ai/chat_dao.go`
- Modify: `internal/service/service.go`
- Test: `internal/dao/ai/run_storage_migration_test.go`
- Test: `internal/dao/ai/run_dao_test.go`

- [ ] **Step 1: Write the failing migration and DAO tests**

```go
func TestRunMigration_AddsClientRequestIDAndExpiryFields(t *testing.T) {}

func TestAIRunDAO_FindByClientRequestID(t *testing.T) {}
```

- [ ] **Step 2: Run the DAO tests to verify they fail**

Run: `go test ./internal/dao/ai -run 'TestRunMigration_AddsClientRequestIDAndExpiryFields|TestAIRunDAO_FindByClientRequestID' -count=1`
Expected: FAIL because columns/helpers do not exist

- [ ] **Step 3: Add the minimal schema and DAO implementation**

```go
type AIRun struct {
    // ...
    ClientRequestID string     `gorm:"column:client_request_id;type:varchar(64);not null;default:'';uniqueIndex:uk_ai_runs_session_request,priority:2"`
    LastEventAt     *time.Time `gorm:"column:last_event_at"`
}
```

```go
func (d *AIRunDAO) CreateOrReuseRunShell(ctx context.Context, userID uint64, sessionID, clientRequestID string, build func() (*model.AIRun, *model.AIChatMessage, *model.AIChatMessage)) (*model.AIRun, bool, error) {
    // transaction + unique constraint + duplicate-key retry lookup
}
```

Implementation notes:
- Do not rely on a plain lookup plus insert sequence.
- Enforce race safety with a unique composite constraint and create-or-reuse transaction semantics.
- If `client_request_id` is empty, treat the request as non-idempotent and skip reuse logic explicitly.
- Catch database duplicate-key errors explicitly (`1062`, `23505`, or driver-equivalent) and downgrade them to “read existing shell” instead of surfacing an error to the caller.
- Add one concurrency-focused DAO test that fires duplicate shell creation attempts and asserts only one shell survives.

- [ ] **Step 4: Re-run the DAO tests**

Run: `go test ./internal/dao/ai -run 'TestRunMigration_AddsClientRequestIDAndExpiryFields|TestAIRunDAO_(FindByClientRequestID|CreateOrReuseRunShell)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add storage/migrations/20260320_0003_add_ai_failed_session_persistence.sql internal/model/ai.go storage/migration/dev_auto.go internal/dao/ai/run_dao.go internal/dao/ai/chat_dao.go internal/service/service.go internal/dao/ai/run_storage_migration_test.go internal/dao/ai/run_dao_test.go
git commit -m "feat(ai): add idempotent run shell schema"
```

### Task 3: Refactor `logic.Chat` into shell-first creation plus unified failure finalization

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/handler/chat.go`
- Test: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/handler/chat_test.go`

- [ ] **Step 1: Write failing lifecycle tests before changing the implementation**

```go
func TestChat_PersistsShellBeforeRunnerStarts(t *testing.T) {}
func TestChat_PersistsFailedStartupTurn(t *testing.T) {}
func TestChat_PersistsPartialBodyOnFatalStreamError(t *testing.T) {}
func TestChat_RetryWithSameClientRequestIDReusesExistingShell(t *testing.T) {}
func TestChat_ProjectionWriteFailureStillFinalizesMessageAndRun(t *testing.T) {}
func TestChat_SuccessStatusesRemainCompletedAndCompletedWithToolErrors(t *testing.T) {}
```

- [ ] **Step 2: Run the focused logic tests to verify failure**

Run: `go test ./internal/service/ai/logic -run 'TestChat_(PersistsShellBeforeRunnerStarts|PersistsFailedStartupTurn|PersistsPartialBodyOnFatalStreamError|RetryWithSameClientRequestIDReusesExistingShell|ProjectionWriteFailureStillFinalizesMessageAndRun|SuccessStatusesRemainCompletedAndCompletedWithToolErrors)' -count=1`
Expected: FAIL because shells are created too late or duplicated

- [ ] **Step 3: Implement the minimal shell-first lifecycle**

```go
shell, reused, err := l.ensureChatShell(ctx, input)
if err != nil { return l.emitSanitizedTopLevelError(err, emit) }

defer func() {
    if finalErr != nil {
        _ = l.failRunCritical(ctx, shell, finalErr, assistantBody, publicError)
        _ = l.persistRunEnhancementsBestEffort(ctx, shell, projectedEvents)
    }
}()
```

Implementation notes:
- Create `session`, `user message`, `assistant message`, and `run` before runner startup.
- Wrap shell creation (`session` if missing, `user message`, `assistant message`, `run`) in one `tx.Transaction(...)` block so the shell is all-or-nothing.
- Update `assistantContent` continuously even on failure.
- Sanitize user-facing error payloads before emitting SSE.
- Keep `message/run` terminal-state update in one DB transaction.
- Move projection/content persistence to a second, best-effort phase.

- [ ] **Step 4: Re-run the focused logic and handler tests**

Run: `go test ./internal/service/ai/logic -run 'TestChat_(PersistsShellBeforeRunnerStarts|PersistsFailedStartupTurn|PersistsPartialBodyOnFatalStreamError|RetryWithSameClientRequestIDReusesExistingShell|ProjectionWriteFailureStillFinalizesMessageAndRun|SuccessStatusesRemainCompletedAndCompletedWithToolErrors)' -count=1`
Expected: PASS

Run: `go test ./internal/service/ai/handler -run 'TestChatHandler_(EmitsSSEErrorInsteadOfJSONEnvelopeOnLateFailure|PassesClientRequestIDIntoLogic)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/handler/chat.go internal/service/ai/logic/logic_test.go internal/service/ai/handler/chat_test.go web/src/components/AI/types.ts web/src/api/modules/ai.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts
git commit -m "feat(ai): persist failed chat turns from shell creation"
```

## Chunk 2: History Fallback and Expiry Recovery

### Task 4: Make session/history APIs return assistant fallback body and terminal error metadata

**Files:**
- Modify: `internal/service/ai/handler/session.go`
- Modify: `api/ai/v1/ai.go`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/historyProjection.ts`
- Test: `internal/service/ai/handler/session_test.go`
- Test: `web/src/components/AI/historyProjection.test.ts`

- [ ] **Step 1: Write the failing session/history tests**

```go
func TestGetSession_IncludesAssistantFallbackBodyAndTerminalErrorState(t *testing.T) {}
func TestGetSession_PreservesAssistantErrorStatusForFailedAndExpiredRuns(t *testing.T) {}
```

```ts
it('falls back to persisted assistant body when projection is missing for failed run', async () => {
  // history message has status error + content "partial answer"
});
```

- [ ] **Step 2: Run the targeted tests to verify they fail**

Run: `go test ./internal/service/ai/handler -run 'TestGetSession_(IncludesAssistantFallbackBodyAndTerminalErrorState|PreservesAssistantErrorStatusForFailedAndExpiredRuns)' -count=1`
Expected: FAIL because assistant content is omitted

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: FAIL because hydration still prefers projection-only path

- [ ] **Step 3: Implement the API and hydration fallback**

```go
func sessionMessageItem(message model.AIChatMessage, run *model.AIRun) gin.H {
    item["content"] = message.Content
    if run != nil && (run.Status == "failed" || run.Status == "failed_runtime" || run.Status == "expired") {
        item["error_message"] = sanitizeForUser(run.ErrorMessage)
        item["status"] = "error"
    }
    return item
}
```

```ts
if (!projection && message.content) {
  return {
    id: message.id,
    role: 'assistant',
    content: message.content,
    runtime: { activities: [], status: { kind: 'error', label: message.error_message || '生成中断，请稍后重试' } },
  };
}
```

- [ ] **Step 4: Re-run the targeted tests**

Run: `go test ./internal/service/ai/handler -run 'TestGetSession_(IncludesAssistantFallbackBodyAndTerminalErrorState|PreservesAssistantErrorStatusForFailedAndExpiredRuns)' -count=1`
Expected: PASS

Run: `npm run test:run -- web/src/components/AI/historyProjection.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/handler/session.go api/ai/v1/ai.go web/src/api/modules/ai.ts web/src/components/AI/historyProjection.ts internal/service/ai/handler/session_test.go web/src/components/AI/historyProjection.test.ts
git commit -m "feat(ai): expose failed assistant fallback bodies"
```

### Task 5: Add stale-run expiry recovery and scheduler wiring

**Files:**
- Create: `internal/service/ai/logic/run_expirer.go`
- Modify: `internal/dao/ai/run_dao.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/service.go`
- Test: `internal/service/ai/logic/run_expirer_test.go`

- [ ] **Step 1: Write the failing expiry tests**

```go
func TestRunExpirer_MarksStaleRunningRunAsExpired(t *testing.T) {}
func TestRunExpirer_PreservesExistingAssistantBodyWhenExpiringRun(t *testing.T) {}
func TestRunExpiryScheduler_StartsBackgroundTicker(t *testing.T) {}
```

- [ ] **Step 2: Run the expiry tests to verify they fail**

Run: `go test ./internal/service/ai/logic -run 'TestRunExpirer_' -count=1`
Expected: FAIL because no expirer exists

- [ ] **Step 3: Implement the smallest expiry service**

```go
type RunExpirer struct {
    RunDAO  *aidao.AIRunDAO
    ChatDAO *aidao.AIChatDAO
}

func (e *RunExpirer) ExpireStaleRuns(ctx context.Context, cutoff time.Time) error {
    // find non-terminal runs older than cutoff, set run.status=expired and assistant.status=error in one transaction
}
```

```go
func StartAIRunExpiryLoop(ctx context.Context, svcCtx *svc.ServiceContext, interval time.Duration) {
    // ticker loop started from service bootstrap
}
```

Implementation notes:
- Wire the scheduler from `internal/service/service.go` startup, not only from tests.
- Keep the ticker interval and expiry cutoff configurable or centralized constants.
- Ensure the loop stops on context cancellation to avoid goroutine leaks in tests.
- Add multi-instance concurrency control to the expiry worker: prefer row-level claiming with `FOR UPDATE SKIP LOCKED` where supported, otherwise use a narrowly scoped Redis lock so only one instance performs a scan window at a time.
- Keep the expiry operation idempotent so duplicate scans are harmless even if the lock/claim path degrades.

- [ ] **Step 4: Re-run the expiry and scheduler tests**

Run: `go test ./internal/service/ai/logic -run 'TestRun(Expirer_|ExpiryScheduler_StartsBackgroundTicker)' -count=1`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/run_expirer.go internal/service/ai/logic/run_expirer_test.go internal/dao/ai/run_dao.go internal/service/ai/logic/logic.go internal/service/service.go
git commit -m "feat(ai): expire stale chat runs"
```

## Chunk 3: Frontend Terminal Error UX

### Task 6: Separate markdown body from terminal error UI in runtime state

**Files:**
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Test: `web/src/components/AI/replyRuntime.test.ts`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Write the failing runtime/render tests**

```ts
it('keeps existing content while storing terminal error status separately in runtime', () => {});
it('renders error callout without appending error text into markdown body', () => {});
```

- [ ] **Step 2: Run the targeted tests to verify they fail**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: FAIL because terminal errors only set a label or rely on concatenated markdown

- [ ] **Step 3: Implement minimal runtime status/error presentation**

```ts
export interface AssistantReplyRuntimeStatus {
  kind: 'streaming' | 'completed' | 'soft-timeout' | 'error' | 'interrupted';
  label: string;
  displayMode?: 'footer' | 'callout';
}
```

```tsx
{runtime?.status?.kind === 'error' ? (
  <Alert type="error" message={runtime.status.label} showIcon />
) : null}
```

- [ ] **Step 4: Re-run the targeted tests**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/replyRuntime.ts web/src/components/AI/types.ts web/src/components/AI/AssistantReply.tsx web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai): render terminal errors outside markdown body"
```

### Task 7: Make the provider and surface converge on terminal failure and retry

**Files:**
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/components/AI/__tests__/CopilotSurface.test.tsx`

- [ ] **Step 1: Write the failing provider/surface tests**

```ts
it('treats stream disconnect without done/error as terminal error', async () => {});
it('generates a new clientRequestId when retrying a failed assistant turn', async () => {});
```

- [ ] **Step 2: Run the targeted tests to verify they fail**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: FAIL because disconnects and retries are not terminalized correctly

- [ ] **Step 3: Implement minimal terminalization and retry behavior**

```ts
const requestId = crypto.randomUUID();
request.run({ message, scene, sessionId, clientRequestId: requestId });
```

```ts
.catch((error) => {
  const normalized = normalizeDisconnectAsTerminalError(error, runtime);
  this.options.callbacks?.onError?.(normalized.error, normalized.info, headers);
})
```

Implementation notes:
- Remove `buildAssistantErrorContent` string concatenation from active-path fallback.
- Keep the existing message body intact on terminal error.
- Add a retry button handler that resubmits into the same session with a fresh `clientRequestId`.
- Assert in tests that retries preserve `sessionId` while regenerating only `clientRequestId`.
- Handle abnormal stream termination from the transport layer as well as SSE `error` events: failed fetch, abrupt body close, and gateway timeout paths must all normalize into the same terminal error runtime state.
- Generate `clientRequestId` with a compatibility-safe helper. Prefer an existing UUID utility if the repo has one; otherwise guard `crypto.randomUUID()` with a fallback so local non-HTTPS or older browsers do not throw.

- [ ] **Step 4: Re-run the targeted tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/AI/providers/PlatformChatProvider.ts web/src/api/modules/ai.ts web/src/components/AI/CopilotSurface.tsx web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx
git commit -m "feat(ai): finalize terminal chat failures in UI"
```

## Final Verification

- [ ] **Step 1: Run focused backend AI tests**

Run: `go test ./internal/service/ai/... -count=1`
Expected: PASS

- [ ] **Step 2: Run focused frontend AI tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/CopilotSurface.test.tsx web/src/components/AI/historyProjection.test.ts web/src/components/AI/replyRuntime.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS

- [ ] **Step 3: Run migration coverage**

Run: `go test ./internal/dao/ai -count=1`
Expected: PASS

- [ ] **Step 4: Manual spot check**

Run the app, trigger:
- startup failure before first token
- partial output then fatal failure
- network disconnect mid-stream

Expected:
- current bubble converges to error
- refresh keeps failed turn in history
- retry creates a new run, old failed run remains visible
