# AI Tool Approval Model Repair Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix end-to-end breakpoints in the AI tool approval flow, and ensure a stable, observable closed loop: "approval triggered -> visible and actionable -> execution resumed".

**Architecture:** Use the backend approval state machine as the single source of truth. Middleware is responsible for interrupts and approval task creation; outbox events are consumed by event type responsibility. Both chat UI and notification center consume the same approval entity and call the submit-only API. Outbox consumption must provide at-least-once delivery semantics, so workers and state transitions must be strictly idempotent. Prioritize P0 stop-bleed first, then complete frontend actionability and replay consistency, then clean up legacy paths and contract debt.

**Tech Stack:** Go (Gin/Gorm/Eino ADK), TypeScript + React + Ant Design, Vitest, Go test.

---

## Scope and Boundaries

This plan only covers the AI tool approval model. It does not include approver groups or cross-user delegated approval (that will be a separate follow-up change).

## File Map

- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/ai/tools/host/tools.go`
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Modify: `internal/ai/tools/middleware/approval_test.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Modify: `web/src/components/AI/replyRuntime.test.ts`
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/api/modules/ai.approval.test.ts`
- Optional cleanup: `internal/service/ai/handler/approval.go` (align comments with actual routing semantics)

## Chunk 1: Backend P0 Stop-Bleed (Outbox + Unified Approval Trigger)

### Task 1: Prevent `approval_requested` from being swallowed by the resume worker

**Files:**
- Modify: `internal/service/ai/logic/approval_worker.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write a failing test (should fail now)**
  - Add a case where outbox contains `approval_requested`: worker must not `MarkDone`; it should remain pending or be routed to a dedicated handler.

- [ ] **Step 2: Run test and confirm failure**
  - Run: `go test ./internal/service/ai/logic -run TestApprovalWorkerDoesNotConsumeApprovalRequested -count=1`
  - Expected: FAIL (current implementation swallows the event)

- [ ] **Step 3: Minimal implementation**
  - In `processClaimedEvent`, return an explicit "unhandled but not done" signal for non-`approval_decided` events (recommended: sentinel error such as `ErrOutboxEventNotHandled`, then `MarkRetry` or keep pending in `RunOnce`).
  - Objective: resume worker handles only `approval_decided`.
  - Requirement: repeated claim/retry must not produce duplicated state transitions (idempotent).

- [ ] **Step 4: Run test and confirm pass**
  - Run: `go test ./internal/service/ai/logic -run TestApprovalWorkerDoesNotConsumeApprovalRequested -count=1`
  - Expected: PASS

- [ ] **Step 5: Commit**
  - `git add internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_worker_test.go`
  - `git commit -m "fix(ai): prevent approval worker from consuming approval_requested events"`

### Task 2: Route `host_exec_readonly` into the unified approval interrupt model

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Test: `internal/ai/tools/middleware/approval_test.go`

- [ ] **Step 1: Write a failing test (should fail now)**
  - Cover the case where `host_exec_readonly` hits approval policy and must produce an interruptible approval (`tool_approval` + `approval_id`).

- [ ] **Step 2: Run test and confirm failure**
  - Run: `go test ./internal/ai/tools/middleware -run TestHostExecReadonlyTriggersUnifiedApprovalInterrupt -count=1`
  - Expected: FAIL

- [ ] **Step 3: Minimal implementation**
  - Approach A (recommended): stop returning pseudo-`suspended` from `host_exec_readonly` tool layer; delegate approval decision to middleware interrupt flow. Include this tool in approval classification via `commandClassForTool` + fallback.
  - Preserve execution semantics: real execution happens only after approval passes.
  - Requirement: if `commandClassForTool` cannot classify, fail safe into approval (never silently allow).

- [ ] **Step 4: Run tests and confirm pass**
  - Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common ./internal/ai/tools/host -count=1`
  - Expected: PASS

- [ ] **Step 5: Commit**
  - `git add internal/ai/tools/host/tools.go internal/ai/tools/middleware/approval.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/middleware/approval_test.go`
  - `git commit -m "fix(ai): route host_exec_readonly approval through unified interrupt model"`

## Chunk 2: Frontend P0 Close the Loop (Chat Approval Actions)

### Task 3: Add in-chat approval actions wired to `submitApproval`

**Files:**
- Modify: `web/src/components/AI/types.ts`
- Modify: `web/src/components/AI/replyRuntime.ts`
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Modify: `web/src/components/AI/ToolReference.tsx`
- Modify: `web/src/components/AI/providers/PlatformChatProvider.ts`
- Modify: `web/src/api/modules/ai.ts`
- Test: `web/src/components/AI/replyRuntime.test.ts`
- Test: `web/src/components/AI/__tests__/PlatformChatProvider.test.ts`
- Test: `web/src/api/modules/ai.approval.test.ts`

- [ ] **Step 1: Write failing tests (should fail now)**
  - `tool_approval` activity should carry `approval_id` and allow approve/reject actions.
  - UI test: approval buttons are visible; clicking triggers `submitApproval`.
  - UI test: when `submitApproval` fails, UI rolls back to waiting-approval and shows an error (avoid optimistic/server state drift).
  - UI test: when concurrent approval conflict occurs (409/400 already approved by someone else), UI refreshes and displays latest status.

- [ ] **Step 2: Run frontend tests and confirm failure**
  - Run: `cd web && npm run test:run -- src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/api/modules/ai.approval.test.ts`
  - Expected: FAIL

- [ ] **Step 3: Minimal implementation**
  - Keep `approval_id` and pending status in runtime activity.
  - Render approve/reject actions for `tool_approval` in `AssistantReply/ToolReference`.
  - Call `aiApi.submitApproval(approval_id, { approved, ... })`.
  - On success, update local state to approved/rejected and emit refresh event.
  - On failure, rollback local state and notify user; on conflict, force-refresh via `getApproval/listPendingApprovals`.

- [ ] **Step 4: Run frontend tests and confirm pass**
  - Run: `cd web && npm run test:run -- src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/api/modules/ai.approval.test.ts`
  - Expected: PASS

- [ ] **Step 5: Commit**
  - `git add web/src/components/AI/types.ts web/src/components/AI/replyRuntime.ts web/src/components/AI/AssistantReply.tsx web/src/components/AI/ToolReference.tsx web/src/components/AI/providers/PlatformChatProvider.ts web/src/components/AI/__tests__/PlatformChatProvider.test.ts web/src/components/AI/replyRuntime.test.ts web/src/api/modules/ai.ts web/src/api/modules/ai.approval.test.ts`
  - `git commit -m "feat(ai-web): add in-chat approval actions for tool_approval activities"`

## Chunk 3: Consistency and Replay Hardening

### Task 4: Replay all pending approval events in `waiting_approval` (not only the last one)

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write a failing test (should fail now)**
  - If a run contains multiple `tool_approval` events, reconnect should replay all of them (ordered and/or deduplicated).

- [ ] **Step 2: Run test and confirm failure**
  - Run: `go test ./internal/service/ai/logic -run TestEmitExistingShellTerminalReplaysAllPendingApprovals -count=1`
  - Expected: FAIL

- [ ] **Step 3: Minimal implementation**
  - Update `emitExistingShellTerminal` to replay all relevant `tool_approval` events (recommended: ascending by Seq, deduplicate by `call_id`).

- [ ] **Step 4: Run test and confirm pass**
  - Run: `go test ./internal/service/ai/logic -run TestEmitExistingShellTerminalReplaysAllPendingApprovals -count=1`
  - Expected: PASS

- [ ] **Step 5: Commit**
  - `git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go`
  - `git commit -m "fix(ai): replay all pending tool approvals on waiting_approval resume"`

### Task 5: Converge API/docs/type inconsistencies

**Files:**
- Modify: `internal/service/ai/handler/approval.go`
- Modify: `web/src/api/modules/ai.ts`
- Test: `internal/service/ai/handler/approval_test.go`
- Test: `web/src/api/modules/ai.approval.test.ts`

- [ ] **Step 1: Align comments with submit-only reality**
  - Remove or clearly mark `ResumeApproval` as legacy/internal-only to avoid confusion in incident debugging.
  - If no external usage is confirmed, remove `ResumeApproval` code and stale comments directly (YAGNI) to reduce maintenance and attack surface.

- [ ] **Step 2: Fix frontend `ApprovalTicket` contract mapping**
  - Add `approval_id` and align with backend `AIApprovalTask` shape to avoid future field mismatches.

- [ ] **Step 3: Run contract tests**
  - Run: `go test ./internal/service/ai/handler -count=1`
  - Run: `cd web && npm run test:run -- src/api/modules/ai.approval.test.ts`
  - Expected: PASS

- [ ] **Step 4: Commit**
  - `git add internal/service/ai/handler/approval.go web/src/api/modules/ai.ts internal/service/ai/handler/approval_test.go web/src/api/modules/ai.approval.test.ts`
  - `git commit -m "chore(ai): align approval API docs and frontend contract with submit-only flow"`

## Final Verification Gate

- [ ] **Step 1: Full backend targeted regression**
  - Run: `go test ./internal/service/ai/... ./internal/ai/tools/... -count=1`
  - Expected: PASS

- [ ] **Step 2: Frontend approval regression**
  - Run: `cd web && npm run test:run -- src/components/AI/replyRuntime.test.ts src/components/AI/__tests__/PlatformChatProvider.test.ts src/api/modules/ai.approval.test.ts`
  - Expected: PASS

- [ ] **Step 3: Manual local E2E smoke**
  - Scenario 1: trigger `tool_approval`, approve in chat, run transitions from `waiting_approval` to `completed`.
  - Scenario 2: reject approval, run transitions to `cancelled`, UI stays in sync.
  - Scenario 3: multiple concurrent approval points are all visible after refresh.
  - Scenario 4: notification center and chat are both open; approving in one updates the other via event-driven sync.

- [ ] **Step 4: Release notes**
  - Document risk: state consistency during coexistence of notification-center path and new chat approval actions.
  - Document rollback strategy: in-chat approval actions can be disabled by feature flag.
  - Migration requirement: both entry points must share one state source (global state or unified SSE/WebSocket events) to avoid dual-write divergence.
