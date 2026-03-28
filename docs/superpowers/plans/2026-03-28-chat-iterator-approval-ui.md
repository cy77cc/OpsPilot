# Chat Iterator Unification & Approval Detail UI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate duplicated backend chat-event iterator loops and ship approval UI that shows actionable operation details before approve/reject.

**Architecture:** Extract one shared iterator processor in `internal/service/ai/logic` and route `Chat`, `ApprovalWorker.resumeApprovedTask`, and deprecated `ResumeApproval` through it using per-path callbacks for terminalization. On frontend, enrich `tool_approval` runtime activity with structured preview summary plus raw JSON, and restore approval action controls with robust submitting/conflict/refresh-needed state transitions.

**Tech Stack:** Go (`gorm`, ADK iterator/projector pipeline), TypeScript + React + Ant Design, Vitest, existing AI API module.

---

## File Structure

- Modify: `internal/service/ai/logic/logic.go` (switch chat + deprecated resume call sites to shared processor)
- Modify: `internal/service/ai/logic/approval_worker.go` (switch approval resume to shared processor)
- Create: `internal/service/ai/logic/iterator_processor.go` (shared iterator processor + result contract)
- Create: `internal/service/ai/logic/iterator_processor_test.go` (unit tests for normal/interrupt/recoverable/fatal stream paths)
- Modify: `internal/service/ai/logic/logic_test.go` (chat regression assertions for parity)
- Modify: `internal/service/ai/logic/approval_worker_test.go` (approval worker parity assertions)
- Modify: `web/src/components/AI/types.ts` (approval preview fields)
- Modify: `web/src/components/AI/replyRuntime.ts` (store preview + derive summary)
- Modify: `web/src/components/AI/ToolReference.tsx` (approval detail rendering + action controls)
- Modify: `web/src/components/AI/ToolResultCard.tsx` (raw preview rendering section)
- Modify: `web/src/components/AI/AssistantReply.tsx` (wire approval actions and readonly states)
- Modify: `web/src/components/AI/replyRuntime.test.ts` (preview/summary tests)
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx` (approval UI behavior tests)
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts` (stream approval activity contract tests)
- Modify: `web/src/api/modules/ai.ts` (if needed: helper exports reused by UI, no API contract break)

## Chunk 1: Backend Iterator Processor

### Task 1: Add shared iterator processor contract and failing tests

**Files:**
- Create: `internal/service/ai/logic/iterator_processor_test.go`
- Modify: `internal/service/ai/logic/test_helpers_test.go` (if small helpers are needed for synthetic iterator events)

- [ ] **Step 1: Write failing unit tests for shared processor behavior**

```go
func TestProcessAgentIterator_InterruptEventStopsWithInterrupted(t *testing.T) {
    iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
    go func() {
        gen.Send(&adk.AgentEvent{Err: approval.ErrApprovalRequired})
        gen.Close()
    }()

    res, err := processAgentIterator(context.Background(), iteratorProcessInput{
        Iterator: iter,
        Projector: airuntime.NewStreamProjector(),
        Emit: func(string, any) {},
    })

    require.NoError(t, err)
    require.True(t, res.Interrupted)
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/service/ai/logic -run TestProcessAgentIterator_ -count=1`
Expected: FAIL with `undefined: processAgentIterator` and missing types.

- [ ] **Step 3: Commit failing test scaffold**

```bash
git add internal/service/ai/logic/iterator_processor_test.go internal/service/ai/logic/test_helpers_test.go
git commit -m "test(ai): add failing tests for shared iterator processor"
```

### Task 2: Implement shared processor to satisfy unit tests

**Files:**
- Create: `internal/service/ai/logic/iterator_processor.go`
- Modify: `internal/service/ai/logic/iterator_processor_test.go`

- [ ] **Step 1: Implement minimal processor contract and result type**

```go
type IteratorProcessResult struct {
    Interrupted      bool
    HasToolErrors    bool
    CircuitBroken    bool
    SummaryText      string
    AssistantSnapshot string
    FatalErr         error
}

type iteratorProcessInput struct {
    Iterator  *adk.AsyncIterator[*adk.AgentEvent]
    Projector *airuntime.StreamProjector
    Emit      EventEmitter
    // callbacks omitted here for brevity in plan; include consume/flush/update hooks.
}
```

- [ ] **Step 2: Implement unified loops (`Next` + `MessageStream.Recv`) with shared recoverable/fatal handling**

Run in code:
- call existing `recoverableInterruptEventFromEvent`
- call existing `recoverableToolErrorFromEvent/recoverableToolErrorFromErr`
- call existing projector consume/flush helpers through callbacks

- [ ] **Step 3: Run focused tests**

Run: `go test ./internal/service/ai/logic -run TestProcessAgentIterator_ -count=1`
Expected: PASS for new processor tests.

- [ ] **Step 4: Commit processor**

```bash
git add internal/service/ai/logic/iterator_processor.go internal/service/ai/logic/iterator_processor_test.go
git commit -m "feat(ai): add shared agent iterator processor"
```

### Task 3: Refactor `Logic.Chat` to use shared processor

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write/update chat regression test first**

Add/adjust test to assert:
- interrupt still emits `tool_approval`
- recoverable tool error still does not hard-fail run
- done/run_state ordering remains unchanged

- [ ] **Step 2: Run targeted chat regression test to verify pre-change behavior snapshot**

Run: `go test ./internal/service/ai/logic -run 'TestLogic_Chat.*(Approval|Recoverable|Done|RunState)' -count=1`
Expected: PASS before refactor.

- [ ] **Step 3: Replace inlined iterator loop in `Chat` with `processAgentIterator`**

Implementation requirement:
- keep existing run status updates and terminalization helpers
- no SSE event-name changes

- [ ] **Step 4: Run targeted tests**

Run: `go test ./internal/service/ai/logic -run 'TestLogic_Chat.*' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit chat refactor**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "refactor(ai): route chat iterator handling through shared processor"
```

## Chunk 2: Approval Resume Paths

### Task 4: Refactor `ApprovalWorker.resumeApprovedTask` to shared processor

**Files:**
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Add failing/updated tests for approval worker iterator parity**

Focus assertions:
- `waiting_approval` interrupt persistence unchanged
- `resume_failed_retryable` on retryable failures unchanged
- terminal statuses and outbox behavior unchanged

- [ ] **Step 2: Run approval worker tests first**

Run: `go test ./internal/service/ai/logic -run 'TestApprovalWorker_.*' -count=1`
Expected: PASS before refactor.

- [ ] **Step 3: Swap duplicated loop in `resumeApprovedTask` with shared processor**

Keep path-specific behavior:
- run status writes
- write-model emits
- retryable vs fatal branching

- [ ] **Step 4: Re-run worker tests**

Run: `go test ./internal/service/ai/logic -run 'TestApprovalWorker_.*' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit worker refactor**

```bash
git add internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_worker_test.go
git commit -m "refactor(ai): reuse shared iterator processor in approval worker resume"
```

### Task 5: Refactor deprecated `ResumeApproval` to shared processor

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Add regression test for deprecated resume path parity**

Test should assert:
- still emits `meta`
- still streams projected events
- still emits `done` on completion

- [ ] **Step 2: Run targeted deprecated path test**

Run: `go test ./internal/service/ai/logic -run 'TestLogic_ResumeApproval.*' -count=1`
Expected: PASS before refactor.

- [ ] **Step 3: Replace deprecated path inlined loop with shared processor**

Keep deprecated method signature and external behavior untouched.

- [ ] **Step 4: Re-run targeted tests**

Run: `go test ./internal/service/ai/logic -run 'TestLogic_ResumeApproval.*' -count=1`
Expected: PASS.

- [ ] **Step 5: Commit deprecated path refactor**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "refactor(ai): align deprecated resume flow with shared iterator processor"
```

## Chunk 3: Frontend Approval Detail UI

### Task 6: Add approval preview data model + runtime derivation

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`

- [ ] **Step 1: Write failing runtime tests for preview summary extraction**

```ts
it('stores tool approval preview and derives summary rows', () => {
  const runtime = applyToolApproval(createEmptyAssistantRuntime(), {
    approval_id: 'approval-1',
    call_id: 'call-1',
    tool_name: 'kubectl_apply',
    timeout_seconds: 300,
    preview: { cluster: 'prod', namespace: 'ops', action: 'apply' },
  });
  const activity = runtime.activities[0];
  expect(activity.approvalPreview).toEqual(expect.objectContaining({ cluster: 'prod' }));
  expect(activity.approvalPreviewSummary?.map((row) => row.key)).toContain('cluster');
});
```

- [ ] **Step 2: Run test to verify failure**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts`
Expected: FAIL for missing fields/summary logic.

- [ ] **Step 3: Implement types + `applyToolApproval` summary derivation**

Implementation constraints:
- deterministic summary key order
- bounded row count (2-8)
- graceful fallback when preview empty

- [ ] **Step 4: Re-run test**

Run: `npm run test:run -- web/src/components/AI/replyRuntime.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit runtime/model changes**

```bash
git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/replyRuntime.test.ts
git commit -m "feat(ai-ui): persist approval preview and derive summary fields"
```

### Task 7: Render approval detail card (summary table + raw JSON expand)

**Files:**
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/ToolResultCard.tsx`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Add failing component tests**

Test cases:
- approval card shows key rows (`cluster`, `namespace`, `action`)
- raw JSON is hidden by default and visible after expand
- no-preview fallback text appears for legacy payload

- [ ] **Step 2: Run failing tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: FAIL for missing rendered sections.

- [ ] **Step 3: Implement UI rendering**

Implementation requirements:
- keep existing style system (`createStyles`)
- avoid nested modals for approval summary
- raw JSON rendered through existing formatting path

- [ ] **Step 4: Re-run tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit UI enrichment**

```bash
git add web/src/components/AI/ToolReference.tsx web/src/components/AI/ToolResultCard.tsx web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai-ui): show approval operation details with expandable raw preview"
```

### Task 8: Restore approval approve/reject interactions with robust states

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/api/modules/ai.ts` (only if helper sharing is required)
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Modify: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: Unskip and rewrite failing approval action tests**

Required assertions:
- pending shows approve/reject buttons
- submitting disables repeated actions (idempotent)
- conflict triggers `getApproval` refresh
- refresh failure becomes readonly `refresh-needed`

- [ ] **Step 2: Run targeted tests and confirm failure**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: FAIL on current skipped/missing behavior.

- [ ] **Step 3: Implement action-state transitions**

Implementation sketch:

```ts
type LocalApprovalUiState = 'waiting-approval' | 'submitting' | 'approved' | 'rejected' | 'refresh-needed';
```

Rules:
- submit once per approval id until request resolves
- on conflict, call `aiApi.getApproval(approvalId)` and map server status

- [ ] **Step 4: Re-run targeted tests**

Run: `npm run test:run -- web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit approval interaction recovery**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/api/modules/ai.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(ai-ui): restore approval actions with conflict-refresh and readonly fallback states"
```

## Chunk 4: End-to-End Verification and Cleanup

### Task 9: Run full regression for touched modules

**Files:**
- Modify: none (verification only unless fixes required)

- [ ] **Step 1: Run backend package tests**

Run: `go test ./internal/service/ai/logic -count=1`
Expected: PASS.

- [ ] **Step 2: Run frontend AI module tests**

Run: `npm run test:run -- web/src/components/AI web/src/api/modules/ai.approval.test.ts web/src/api/modules/ai.streamChunk.test.ts`
Expected: PASS.

- [ ] **Step 3: Commit any final test-only or lint fixes**

```bash
git add <fixed-files>
git commit -m "test(ai): stabilize iterator and approval UI regression coverage"
```

### Task 10: Final documentation and handoff note

**Files:**
- Modify: `docs/superpowers/specs/2026-03-28-chat-event-iterator-approval-ui-design.md` (only if implementation deviated)
- Create/Modify: optional short changelog in existing docs if required by repo conventions

- [ ] **Step 1: Document deviations from spec (if any)**
- [ ] **Step 2: Run `git status --short` to ensure clean tree**
- [ ] **Step 3: Produce implementation summary with test evidence**

---

## Execution Notes

1. Follow DRY/YAGNI: do not redesign unrelated runtime or SSE schema.
2. Keep commits small and task-scoped.
3. Any unexpected behavior drift in chat stream ordering should block merge until resolved.
