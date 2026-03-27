# AI Tool Approval Unification Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify high-risk tool approval semantics across all agents so approval always goes through middleware interrupts, with whitelist-driven behavior for `host_exec_readonly`.

**Architecture:** Centralize risk/approval decisions in middleware using a two-layer policy model (DB override + code default, fail-closed). Remove tool-layer pseudo approval states (`status=suspended`) and keep approval lifecycle in the interrupt/task/outbox flow. Align all tool-calling agents to one middleware chain and harden lifecycle edges (TTL, rejection feedback, audit fields, concurrent approval sequencing).

**Tech Stack:** Go (Eino ADK, Gin, Gorm), existing AI runtime projector/logic, Go test.

---

## File Map

- Create: `internal/ai/tools/common/risk_registry.go`
- Create: `internal/ai/tools/common/risk_registry_test.go`
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/middleware/approval_test.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Modify: `internal/ai/tools/common/approval_orchestrator_test.go`
- Modify: `internal/ai/tools/host/tools.go`
- Modify: `internal/ai/tools/host/tools_test.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Modify: `internal/ai/agents/qa/qa.go`
- Create: `internal/ai/agents/qa/qa_test.go`
- Modify: `internal/ai/runtime/event_types.go` (if grouped approval payload fields are added)
- Modify: `internal/ai/runtime/event_types_test.go` (if payload schema changes)

## Chunk 1: Risk Decision Unification

### Task 1: Introduce code-default risk registry (single fallback source)

**Files:**
- Create: `internal/ai/tools/common/risk_registry.go`
- Test: `internal/ai/tools/common/risk_registry_test.go`

- [ ] **Step 1: Write failing tests for risk lookup**
```go
func TestRiskRegistry_HostExecReadonlyIsConditional(t *testing.T) {
	reg := NewDefaultRiskRegistry()
	policy, ok := reg.Lookup("host_exec_readonly")
	if !ok || !policy.ConditionalByCommandClass {
		t.Fatalf("expected conditional host_exec_readonly policy")
	}
}
```

- [ ] **Step 2: Run tests to confirm failure**
Run: `go test ./internal/ai/tools/common -run TestRiskRegistry_ -count=1`  
Expected: FAIL (registry not implemented)

- [ ] **Step 3: Implement minimal registry**
```go
type DefaultRiskPolicy struct {
	NeedsApproval             bool
	ConditionalByCommandClass bool
}
```

- [ ] **Step 4: Run tests to confirm pass**
Run: `go test ./internal/ai/tools/common -run TestRiskRegistry_ -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/tools/common/risk_registry.go internal/ai/tools/common/risk_registry_test.go
git commit -m "feat(ai): add default risk registry for tool approval fallback"
```

### Task 2: Add risk policy persistence model and migration

**Files:**
- Modify: `internal/model/ai_approval.go` (or the existing AI model file that hosts approval/risk entities)
- Create: `storage/migrations/20260322_xxxx_create_ai_tool_risk_policies.sql`
- Test: `storage/migration/ai_approval_migration_test.go`

- [ ] **Step 1: Write failing migration/model test**
```go
func TestAIMigration_CreatesAIToolRiskPoliciesTable(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./storage/migration -run TestAIMigration_CreatesAIToolRiskPoliciesTable -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal schema + model**
```sql
CREATE TABLE IF NOT EXISTS ai_tool_risk_policies (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  tool_name VARCHAR(128) NOT NULL,
  scene VARCHAR(64) NOT NULL DEFAULT '',
  command_class VARCHAR(64) NOT NULL DEFAULT '',
  approval_required TINYINT(1) NOT NULL DEFAULT 0,
  risk_level VARCHAR(16) NOT NULL DEFAULT 'medium',
  policy_version VARCHAR(64) NOT NULL DEFAULT '',
  enabled TINYINT(1) NOT NULL DEFAULT 1
  -- plus created_at/updated_at/deleted_at following repo conventions
);
```

- [ ] **Step 4: Run test to confirm pass**
Run: `go test ./storage/migration -run TestAIMigration_CreatesAIToolRiskPoliciesTable -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/model/ai_approval.go storage/migrations/20260322_xxxx_create_ai_tool_risk_policies.sql storage/migration/ai_approval_migration_test.go
git commit -m "feat(ai): add ai_tool_risk_policies schema and model"
```

### Task 3: Wire middleware to registry + host policy precheck + fail-closed DB fallback

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Test: `internal/ai/tools/middleware/approval_test.go`
- Test: `internal/ai/tools/common/approval_orchestrator_test.go`
- Create: `internal/ai/tools/common/risk_policy_repository.go` (if not already present)

- [ ] **Step 1: Write failing tests for DB error fallback and host allowlist behavior**
```go
func TestApprovalMiddleware_HostExecReadonlyAllowlistedSkipsInterrupt(t *testing.T) {}
func TestApprovalMiddleware_HostExecReadonlyNonAllowlistedInterrupts(t *testing.T) {}
func TestApprovalOrchestrator_DBErrorFallsBackFailClosed(t *testing.T) {}
```

- [ ] **Step 2: Run tests to confirm failure**
Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run 'TestApproval(Middleware|Orchestrator)_' -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal decision path**
```go
type RiskPolicyRepository interface {
	ListEnabledByToolName(ctx context.Context, toolName string) ([]model.AIToolRiskPolicy, error)
}

// pseudocode
if toolName in hostExecFamily {
  decision := hostPolicyEngine.Evaluate(...)
  if decision == allowReadonly { return noApproval }
  return requireApproval
}
// DB policy errors => fallback to default registry; uncertain => require approval
```

- [ ] **Step 4: Run tests to confirm pass**
Run: `go test ./internal/ai/tools/middleware ./internal/ai/tools/common -run 'TestApproval(Middleware|Orchestrator)_' -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/middleware/approval_test.go internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/common/approval_orchestrator_test.go internal/ai/tools/common/risk_policy_repository.go
git commit -m "fix(ai): unify approval decision path with fail-closed fallback"
```

### Task 4: Remove pseudo approval state from host tool layer

**Files:**
- Modify: `internal/ai/tools/host/tools.go`
- Test: `internal/ai/tools/host/tools_test.go`

- [ ] **Step 1: Write failing tests to reject `status=suspended` approval signaling**
```go
func TestHostExecReadonly_DoesNotReturnSuspendedForApprovalWait(t *testing.T) {}
```

- [ ] **Step 2: Run tests to confirm failure**
Run: `go test ./internal/ai/tools/host -run TestHostExecReadonly_ -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal change**
```go
// remove approval-wait encoding from HostExecOutput path
// host tool returns execution result only; approval gate handled in middleware
```

- [ ] **Step 4: Run tests to confirm pass**
Run: `go test ./internal/ai/tools/host -run TestHostExecReadonly_ -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/tools/host/tools.go internal/ai/tools/host/tools_test.go
git commit -m "refactor(ai): remove pseudo suspended approval state from host exec tools"
```

## Chunk 2: Approval Lifecycle Hardening

### Task 5: Implement approval TTL expiration convergence

**Files:**
- Modify: `internal/service/ai/logic/approval_worker.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write failing TTL test**
```go
func TestApprovalWorker_ExpiresPendingApprovalAfterTTL(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./internal/service/ai/logic -run TestApprovalWorker_ExpiresPendingApprovalAfterTTL -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal expiration flow**
```go
if task.Status == "pending" && task.ExpiresAt.Before(now) {
  // mark expired/timeout_rejected and finalize run without runtime fatal
}
```

- [ ] **Step 4: Run test to confirm pass**
Run: `go test ./internal/service/ai/logic -run TestApprovalWorker_ExpiresPendingApprovalAfterTTL -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_worker_test.go
git commit -m "feat(ai): enforce approval ttl expiration convergence"
```

### Task 6: Inject explicit rejection feedback into agent-visible flow

**Files:**
- Modify: `internal/ai/tools/middleware/approval.go`
- Modify: `internal/service/ai/logic/logic.go`
- Test: `internal/ai/tools/middleware/approval_test.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing rejection-feedback test**
```go
func TestApprovalReject_ProducesExplicitPolicyRejectionFeedback(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./internal/ai/tools/middleware ./internal/service/ai/logic -run TestApprovalReject_ -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal feedback contract**
```go
return "tool execution rejected by user/admin policy: <reason>", nil
```

- [ ] **Step 4: Run test to confirm pass**
Run: `go test ./internal/ai/tools/middleware ./internal/service/ai/logic -run TestApprovalReject_ -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/tools/middleware/approval.go internal/ai/tools/middleware/approval_test.go internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "fix(ai): provide explicit rejection feedback to avoid blind retries"
```

### Task 7: Add standardized approval audit fields

**Files:**
- Modify: `internal/ai/tools/common/approval_orchestrator.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Test: `internal/ai/tools/common/approval_orchestrator_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write failing audit payload tests**
```go
func TestApprovalRequestedPayload_ContainsStandardAuditFields(t *testing.T) {}
func TestApprovalDecidedPayload_ContainsApproverMetadata(t *testing.T) {}
```

- [ ] **Step 2: Run tests to confirm failure**
Run: `go test ./internal/ai/tools/common ./internal/service/ai/logic -run 'TestApproval(Requested|Decided)Payload_' -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal payload enrichment**
```go
eventPayload["decision_source"] = decisionSource
eventPayload["matched_rule_id"] = ...
eventPayload["approver_id"] = ...
```

- [ ] **Step 4: Run tests to confirm pass**
Run: `go test ./internal/ai/tools/common ./internal/service/ai/logic -run 'TestApproval(Requested|Decided)Payload_' -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/tools/common/approval_orchestrator.go internal/ai/tools/common/approval_orchestrator_test.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_worker_test.go
git commit -m "feat(ai): standardize approval audit payload fields"
```

### Task 8: Add deterministic sequencing metadata for concurrent approval-required calls

**Files:**
- Modify: `internal/ai/runtime/event_types.go` (if schema field added)
- Modify: `internal/ai/runtime/event_types_test.go`
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing test for deterministic replay order**
```go
func TestWaitingApproval_ReplaysPendingApprovalsDeterministically(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./internal/service/ai/logic -run TestWaitingApproval_ReplaysPendingApprovalsDeterministically -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal deterministic ordering**
```go
// seq comes from persisted run event sequence (ai_run_events.seq) rather than outbox id.
// when loading pending approvals, order by Seq ascending (and ID asc as stable tie-breaker).
sort.SliceStable(pendingApprovals, func(i, j int) bool { return pendingApprovals[i].seq < pendingApprovals[j].seq })
```

- [ ] **Step 4: Run tests to confirm pass**
Run: `go test ./internal/service/ai/logic ./internal/ai/runtime -run 'TestWaitingApproval_|TestEventTypeToolApproval' -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go internal/ai/runtime/event_types.go internal/ai/runtime/event_types_test.go
git commit -m "fix(ai): enforce deterministic pending approval sequencing"
```

## Chunk 3: Agent Wiring and Coverage

### Task 9: Ensure QA agent mounts approval middleware

**Files:**
- Modify: `internal/ai/agents/qa/qa.go`
- Create: `internal/ai/agents/qa/qa_test.go`

- [ ] **Step 1: Write failing middleware wiring test**
```go
func TestQAAgent_MountsApprovalMiddleware(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./internal/ai/agents/qa -run TestQAAgent_MountsApprovalMiddleware -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal wiring**
```go
approvalMW := tools.ApprovalToolMiddleware(nil)
ToolCallMiddlewares: []compose.ToolMiddleware{normalizerMW, approvalMW}
```

- [ ] **Step 4: Run test to confirm pass**
Run: `go test ./internal/ai/agents/qa -run TestQAAgent_MountsApprovalMiddleware -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/agents/qa/qa.go internal/ai/agents/qa/qa_test.go
git commit -m "fix(ai): mount approval middleware for qa agent"
```

### Task 10: Add coverage test for high-risk tool registry completeness

**Files:**
- Modify: `internal/ai/tools/common/risk_registry_test.go`
- Modify: `internal/ai/tools/tools.go` (if helper export is needed)

- [ ] **Step 1: Write failing completeness test**
```go
func TestRiskRegistry_CoversAllKnownHighRiskTools(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./internal/ai/tools/common -run TestRiskRegistry_CoversAllKnownHighRiskTools -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal helper and table**
```go
var HighRiskToolNames = []string{"host_exec_change", "k8s_delete_pod", ...}
```

- [ ] **Step 4: Run test to confirm pass**
Run: `go test ./internal/ai/tools/common -run TestRiskRegistry_CoversAllKnownHighRiskTools -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai/tools/common/risk_registry_test.go internal/ai/tools/tools.go
git commit -m "test(ai): enforce high-risk tool registry coverage"
```

## Chunk 4: Verification and Rollout Safety

### Task 11: Add migration compatibility for in-flight legacy suspended runs

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Write failing compatibility test**
```go
func TestLegacySuspendedRun_RemainsActionableDuringMigration(t *testing.T) {}
```

- [ ] **Step 2: Run test to confirm failure**
Run: `go test ./internal/service/ai/logic -run TestLegacySuspendedRun_RemainsActionableDuringMigration -count=1`  
Expected: FAIL

- [ ] **Step 3: Implement minimal compatibility parser**
```go
// tolerate legacy suspended marker in stored payloads during rollout window
```

- [ ] **Step 4: Run test to confirm pass**
Run: `go test ./internal/service/ai/logic -run TestLegacySuspendedRun_RemainsActionableDuringMigration -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "fix(ai): keep legacy suspended runs actionable during migration window"
```

### Task 12: Final targeted regression suite

**Files:**
- No code changes (verification gate)

- [ ] **Step 1: Run backend regression suite**
Run: `go test ./internal/ai/tools/... ./internal/service/ai/logic/... ./internal/ai/agents/... -count=1`  
Expected: PASS

- [ ] **Step 2: Verify approval-specific tests only**
Run: `go test ./internal/... -run 'Approval|waiting_approval|host_exec_readonly' -count=1`  
Expected: PASS

- [ ] **Step 3: Commit verification notes**
```bash
git add -A
git commit -m "test(ai): verify unified tool approval regression suite"
```

### Task 13: Track and schedule legacy compatibility cleanup

**Files:**
- Modify: `docs/superpowers/plans/2026-03-22-ai-tool-approval-unification.md` (this plan, mark explicit cleanup ticket)
- Optional modify: `docs/superpowers/plans/2026-03-xx-...` (follow-up cleanup plan once rollout is complete)

- [ ] **Step 1: Add explicit cleanup tracking entry**
```markdown
Cleanup Tracking:
- [ ] Remove legacy suspended compatibility parser after rollout window and in-flight run drain.
- Owner: AI Runtime
- Exit criteria: no legacy suspended run observed for 14 consecutive days.
```

- [ ] **Step 2: Commit tracking update**
```bash
git add docs/superpowers/plans/2026-03-22-ai-tool-approval-unification.md
git commit -m "docs(ai): track post-rollout cleanup for legacy suspended compatibility path"
```

## Execution Notes

1. Keep each task small and linear: fail test -> minimal fix -> pass test -> commit.
2. Avoid opportunistic refactors outside approval semantics.
3. Preserve API compatibility for current frontend event consumers.
4. Remove migration compatibility path in a follow-up cleanup change after in-flight runs drain (tracked in Task 13 with explicit exit criteria).

## Cleanup Tracking

- [ ] Remove legacy suspended compatibility parser after rollout window and in-flight run drain.
- Owner: AI Runtime
- Exit criteria: no legacy suspended run observed for 14 consecutive days.
