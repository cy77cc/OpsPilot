# AI Approval Command Risk Audit Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a robust high-risk `tool_call` approval flow with DB-driven risk policies, lease-based approval locking, strict idempotent outbox events, and worker-based resume convergence.

**Architecture:** Add a policy-evaluation path (`RiskPolicyStore` + orchestrator) that gates `tool_call` interrupts, persist approval decisions into an outbox, and process resume asynchronously with lock leases and mandatory `resuming` state transitions. Keep `SubmitApproval` write-only and enforce immutable post-lock task state. Preserve auditability via run events and deterministic replay snapshots.

**Tech Stack:** Go, Gin, GORM, MySQL migrations, ADK resumable runner, SSE, Go test

---

## Chunk 1: Risk Policy Foundation

### Task 1: Add DB schema for risk policy and approval lock lease fields

**Files:**
- Create: `storage/migrations/20260321_0005_add_ai_tool_risk_policy_and_approval_locking.sql`
- Modify: `internal/model/ai.go`
- Test: `storage/migration/ai_approval_migration_test.go`

- [ ] **Step 1: Write failing migration test for new columns/tables**

```go
func TestAIApprovalRiskPolicyMigration(t *testing.T) {
    // assert ai_tool_risk_policies exists
    // assert ai_approval_tasks has lock_expires_at/matched_rule_id/policy_version/decision_source
}
```

- [ ] **Step 2: Run targeted migration test to verify failure**

Run: `go test ./storage/migration -run AIApprovalRiskPolicyMigration -v`
Expected: FAIL with missing table/columns

- [ ] **Step 3: Add migration SQL with indexes**

```sql
CREATE TABLE ai_tool_risk_policies (...);
ALTER TABLE ai_approval_tasks
  ADD COLUMN matched_rule_id BIGINT NULL,
  ADD COLUMN policy_version VARCHAR(64) NULL,
  ADD COLUMN decision_source VARCHAR(32) NULL,
  ADD COLUMN lock_expires_at DATETIME NULL;
CREATE INDEX idx_ai_tool_risk_policies_tool_enabled ON ai_tool_risk_policies(tool_name, enabled);
```

- [ ] **Step 4: Update model structs minimally**

```go
type AIToolRiskPolicy struct { ... }
type AIApprovalTask struct { ... LockExpiresAt *time.Time ... }
```

- [ ] **Step 5: Re-run targeted migration test**

Run: `go test ./storage/migration -run AIApprovalRiskPolicyMigration -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add storage/migrations/20260321_0005_add_ai_tool_risk_policy_and_approval_locking.sql internal/model/ai.go storage/migration/ai_approval_migration_test.go
git commit -m "feat: add risk policy and approval lock lease schema"
```

### Task 2: Implement policy DAO with app-layer argument matching contract

**Files:**
- Create: `internal/dao/ai/risk_policy_dao.go`
- Create: `internal/dao/ai/risk_policy_dao_test.go`
- Create: `internal/service/ai/logic/risk_policy_matcher.go`
- Create: `internal/service/ai/logic/risk_policy_matcher_test.go`
- Modify: `internal/service/ai/handler/handler.go`

- [ ] **Step 1: Write failing DAO test for fast query path**

```go
func TestListEnabledByToolName(t *testing.T) {
    // query must be by tool_name + enabled=true only
}
```

- [ ] **Step 2: Run DAO test and verify failure**

Run: `go test ./internal/dao/ai -run RiskPolicy -v`
Expected: FAIL with missing DAO/method

- [ ] **Step 3: Implement DAO and matcher**

```go
func (d *AIToolRiskPolicyDAO) ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error)
func MatchRiskPolicy(rules []model.AIToolRiskPolicy, scene, commandClass string, args map[string]any) (*model.AIToolRiskPolicy, bool)
```

- [ ] **Step 4: Add matcher tests for specificity and priority**

```go
// argument-aware > command_class > scene-only > tool-only
```

- [ ] **Step 5: Re-run tests**

Run: `go test ./internal/dao/ai ./internal/service/ai/logic -run RiskPolicy -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/dao/ai/risk_policy_dao.go internal/dao/ai/risk_policy_dao_test.go internal/service/ai/logic/risk_policy_matcher.go internal/service/ai/logic/risk_policy_matcher_test.go internal/service/ai/handler/handler.go
git commit -m "feat: add db-driven risk policy query and matcher"
```

## Chunk 2: Approval State Integrity And Outbox

### Task 3: Add approval outbox model/DAO with strict idempotency key

**Files:**
- Modify: `internal/model/ai.go`
- Create: `internal/dao/ai/approval_outbox_dao.go`
- Create: `internal/dao/ai/approval_outbox_dao_test.go`
- Create: `storage/migrations/20260321_0006_create_ai_approval_outbox_events.sql`

- [ ] **Step 1: Write failing DAO test for unique `(approval_id, event_type)`**

```go
func TestApprovalOutboxUniqueKey(t *testing.T) {
    // duplicate approval_decided insert should be rejected or upserted deterministically
}
```

- [ ] **Step 2: Run test and verify failure**

Run: `go test ./internal/dao/ai -run ApprovalOutbox -v`
Expected: FAIL with missing table/DAO

- [ ] **Step 3: Implement migration + model + DAO**

```go
type AIApprovalOutboxEvent struct { ApprovalID string; EventType string; RetryCount int }
func (d *AIApprovalOutboxDAO) EnqueueOrTouch(...)
func (d *AIApprovalOutboxDAO) ClaimPending(...)
func (d *AIApprovalOutboxDAO) MarkDone(...)
func (d *AIApprovalOutboxDAO) MarkRetry(...)
```

- [ ] **Step 4: Re-run DAO tests**

Run: `go test ./internal/dao/ai -run ApprovalOutbox -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add storage/migrations/20260321_0006_create_ai_approval_outbox_events.sql internal/model/ai.go internal/dao/ai/approval_outbox_dao.go internal/dao/ai/approval_outbox_dao_test.go
git commit -m "feat: add approval outbox with strict idempotency"
```

### Task 4: Harden approval DAO transitions with lock lease semantics

**Files:**
- Modify: `internal/dao/ai/approval_dao.go`
- Create: `internal/dao/ai/approval_dao_locking_test.go`

- [ ] **Step 1: Write failing tests for transition guards**

```go
func TestApproveSetsLockLeaseAtomically(t *testing.T) {}
func TestDecisionRejectedAfterLock(t *testing.T) {}
func TestStealExpiredLock(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/dao/ai -run Approval.*Lock -v`
Expected: FAIL

- [ ] **Step 3: Implement DAO methods**

```go
func (d *AIApprovalTaskDAO) ApproveWithLease(...)
func (d *AIApprovalTaskDAO) RejectPending(...)
func (d *AIApprovalTaskDAO) AcquireOrStealLease(...)
```

- [ ] **Step 4: Re-run tests**

Run: `go test ./internal/dao/ai -run Approval.*Lock -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/dao/ai/approval_dao.go internal/dao/ai/approval_dao_locking_test.go
git commit -m "feat: enforce approval task lock lease transitions"
```

## Chunk 3: Runtime Integration And Worker Resume

### Task 5: Wire dynamic risk policy into approval middleware/orchestrator

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Create: `internal/service/ai/logic/approval_orchestrator.go`
- Create: `internal/service/ai/logic/approval_orchestrator_test.go`
- Modify: `internal/ai/agents/change/agent.go`

- [ ] **Step 1: Write failing orchestrator tests**

```go
func TestEvaluateRequiresApprovalByPolicy(t *testing.T) {}
func TestEvaluateFallsBackSafeOnPolicyError(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/service/ai/logic -run ApprovalOrchestrator -v`
Expected: FAIL

- [ ] **Step 3: Implement orchestrator and middleware hook**

```go
func (o *ApprovalOrchestrator) Evaluate(ctx context.Context, toolName string, args string, meta EvalMeta) (Decision, error)
```

- [ ] **Step 4: Ensure interrupt creates approval task snapshot + outbox event**

```go
CreateFromInterrupt(...matched_rule_id, policy_version, decision_source, expires_at...)
EnqueueOrTouch(..., eventType="approval_requested")
```

- [ ] **Step 5: Re-run tests**

Run: `go test ./internal/service/ai/logic ./internal/ai/tools/middleware -run Approval -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/logic/approval_orchestrator.go internal/service/ai/logic/approval_orchestrator_test.go internal/ai/tools/middleware/approval.go internal/ai/agents/change/agent.go
git commit -m "feat: integrate db policy based tool-call approval orchestration"
```

### Task 6: Implement write-only submit + async resume worker

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Create: `internal/service/ai/logic/approval_worker.go`
- Create: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/service/ai/handler/approval.go`
- Modify: `internal/service/ai/routes.go`

- [ ] **Step 1: Write failing tests for submit/resume split**

```go
func TestSubmitApprovalOnlyWritesDecisionAndOutbox(t *testing.T) {}
func TestWorkerSkipsExpiredAndRejectedTasks(t *testing.T) {}
func TestWorkerSetsRunResumingAndConverges(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/service/ai/logic -run "SubmitApproval|ApprovalWorker" -v`
Expected: FAIL

- [ ] **Step 3: Implement logic changes**

```go
// SubmitApproval: ownership check + immutable-after-lock + enqueue approval_decided
// ResumeApproval endpoint: optional removal or keep as admin/debug-only path
// Worker: claim outbox -> acquire/steal lease -> set run resuming -> ResumeWithParams -> finalize
```

- [ ] **Step 4: Update handler behavior for SSE-safe errors**

```go
// no JSON server error after SSE headers are sent
```

- [ ] **Step 5: Re-run tests**

Run: `go test ./internal/service/ai/logic ./internal/service/ai/handler -run "Approval|Resume" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_worker_test.go internal/service/ai/handler/approval.go internal/service/ai/routes.go
git commit -m "feat: move approval resume to async worker with lock lease recovery"
```

## Chunk 4: Run Event Convergence And End-To-End Contract

### Task 7: Persist approval/run_state events and block fake completion

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/ai/runtime/project.go`
- Modify: `internal/ai/runtime/types.go`
- Modify: `internal/ai/runtime/event_types_test.go`
- Modify: `internal/ai/runtime/project_test.go`

- [ ] **Step 1: Write failing runtime/logic tests**

```go
func TestInterruptDoesNotFinalizeDone(t *testing.T) {}
func TestMarshalProjectedEventIncludesToolApprovalAndRunState(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `go test ./internal/ai/runtime ./internal/service/ai/logic -run "Interrupt|RunState|ToolApproval" -v`
Expected: FAIL

- [ ] **Step 3: Implement event type and finalize guards**

```go
// add marshalProjectedEvent branches for tool_approval/run_state
// gate Chat terminal done when waiting_approval
```

- [ ] **Step 4: Re-run tests**

Run: `go test ./internal/ai/runtime ./internal/service/ai/logic -run "Interrupt|RunState|ToolApproval" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/service/ai/logic/logic.go internal/ai/runtime/project.go internal/ai/runtime/types.go internal/ai/runtime/event_types_test.go internal/ai/runtime/project_test.go
git commit -m "fix: persist approval state events and prevent fake run completion"
```

### Task 8: Frontend/API contract alignment and integration tests

**Files:**
- Modify: `web/src/api/modules/ai.ts`
- Modify: `web/src/components/AI/CopilotSurface.tsx`
- Create: `web/src/api/modules/ai.approval.test.ts`
- Create: `internal/service/ai/handler/approval_test.go`
- Modify: `api/ai/v1/ai.go`

- [ ] **Step 1: Write failing contract tests**

```ts
it('uses /ai/approvals/:id/submit for decisions')
it('does not call legacy /confirm or /chains/.../decision path')
```

```go
func TestSubmitApprovalRouteContract(t *testing.T) {}
```

- [ ] **Step 2: Run tests and verify failure**

Run: `npm run test:run -- web/src/api/modules/ai.approval.test.ts`
Run: `go test ./internal/service/ai/handler -run Approval -v`
Expected: FAIL

- [ ] **Step 3: Implement contract alignment**

```ts
// remove or deprecate createApproval/confirmApproval/decideChainApproval paths for this flow
// use pending/get/submit and polling/stream updates from run projection
```

- [ ] **Step 4: Re-run tests**

Run: `npm run test:run -- web/src/api/modules/ai.approval.test.ts`
Run: `go test ./internal/service/ai/handler -run Approval -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/api/modules/ai.ts web/src/components/AI/CopilotSurface.tsx web/src/api/modules/ai.approval.test.ts internal/service/ai/handler/approval_test.go api/ai/v1/ai.go
git commit -m "feat: align approval frontend/backend contract for submit-only decisions"
```

## Final Verification Checklist

- [ ] Run backend focused suite:

```bash
go test ./internal/dao/ai ./internal/service/ai/logic ./internal/service/ai/handler ./internal/ai/runtime -v
```

Expected: PASS

- [ ] Run frontend approval API suite:

```bash
npm run test:run -- web/src/api/modules/ai.approval.test.ts
```

Expected: PASS

- [ ] Run migration validation:

```bash
go test ./storage/migration -v
```

Expected: PASS

- [ ] Manual smoke checklist:
- trigger high-risk tool call and verify `waiting_approval`
- submit approve and verify worker changes run to `resuming`
- crash worker simulation and verify lease expiry recovery
- submit reject and verify no resume execution
- verify outbox row uniqueness for `(approval_id, event_type)`

## Plan Review Notes

- Follow `@superpowers:test-driven-development` for each task.
- Use `@superpowers:systematic-debugging` for failing worker or lease-recovery tests.
- Keep commits task-scoped and reversible.
