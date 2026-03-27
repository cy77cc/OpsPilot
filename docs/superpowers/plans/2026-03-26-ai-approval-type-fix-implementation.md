# AI Approval Type Fix Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix approval resume payload typing so approval decisions always resume tool execution with `*approval.ApprovalResult` and never get dropped by `GetResumeContext`.

**Architecture:** Register `ApprovalResult` in its owner package (`internal/ai/common/approval`) and standardize both resume producers (`ApprovalWorker` and deprecated `Logic.ResumeApproval`) to emit the same typed payload. Keep audit/outbox payloads unchanged (still JSON maps) and only change ADK resume targets. Validate behavior with focused logic tests and targeted approval test runs.

**Tech Stack:** Go, GORM, Eino ADK (`ResumeWithParams`, `GetResumeContext`), existing approval worker tests

---

## File Structure

- Create: `internal/ai/common/approval/register.go`
  - Own schema registration for `ApprovalResult` to avoid middleware import-order coupling.
- Modify: `internal/ai/common/middleware/approval.go`
  - Remove `ApprovalResult` registration responsibility; keep only local interrupt-state registration/gob registrations.
- Modify: `internal/service/ai/logic/approval_worker.go`
  - Replace `map[string]any` resume payload with `*approval.ApprovalResult`.
- Modify: `internal/service/ai/logic/approval_worker_test.go`
  - Assert typed resume target payload (`*approval.ApprovalResult`) at both existing assertion points.
- Modify: `internal/service/ai/logic/logic.go`
  - Update deprecated `ResumeApproval` to emit `*approval.ApprovalResult` and set decision timestamp for both approve/reject.
- Modify: `internal/service/ai/logic/logic_test.go`
  - Add one focused test covering deprecated resume payload construction behavior (typed payload + decision timestamp semantics).

## Scope Check

This is a single subsystem change (approval resume typing) with one shared data contract (`ApprovalResult`), so one implementation plan is sufficient.

## Chunk 1: ApprovalWorker + Registration Path

### Task 1: Lock failing tests for worker resume target typing

**Files:**
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Write failing assertions in `TestApprovalWorker_ResumesApprovedTask`**

```go
target, ok := capturedParams.Targets["tool-call-resume"].(*approval.ApprovalResult)
if !ok {
    t.Fatalf("expected *approval.ApprovalResult, got %#v", capturedParams.Targets)
}
if !target.Approved || target.Comment != "looks good" {
    t.Fatalf("unexpected approval result: %#v", target)
}
```

- [ ] **Step 2: Write failing assertions in `TestApprovalWorker_SubmitApprovalWritesAuditEvents`**

```go
target, ok := params.Targets[task.ToolCallID].(*approval.ApprovalResult)
if !ok {
    t.Fatalf("expected *approval.ApprovalResult for %q, got %#v", task.ToolCallID, params.Targets)
}
if target.Comment != "ship it" {
    t.Fatalf("expected Comment='ship it', got %q", target.Comment)
}
```

- [ ] **Step 3: Run the focused worker tests and confirm failure**

Run: `go test ./internal/service/ai/logic -run "TestApprovalWorker_ResumesApprovedTask|TestApprovalWorker_SubmitApprovalWritesAuditEvents" -v`  
Expected: FAIL with type assertion mismatch (`map[string]any` vs `*approval.ApprovalResult`).

- [ ] **Step 4: Commit failing baseline**

```bash
git add internal/service/ai/logic/approval_worker_test.go
git commit -m "test: require typed approval resume payload in worker tests"
```

### Task 2: Implement typed worker payload + owner-package registration

**Files:**
- Create: `internal/ai/common/approval/register.go`
- Modify: `internal/ai/common/middleware/approval.go`
- Modify: `internal/service/ai/logic/approval_worker.go`
- Modify: `internal/service/ai/logic/approval_worker_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`

- [ ] **Step 1: Add owner-package schema registration**

```go
// internal/ai/common/approval/register.go
package approval

import "github.com/cloudwego/eino/schema"

func init() {
    schema.Register[*ApprovalResult]()
}
```

- [ ] **Step 2: Keep middleware init focused on local registrations**

```go
func init() {
    schema.RegisterName[approvalInterruptState]("_opspilot_approval_interrupt_state")
    gob.Register(map[string]any{})
    gob.Register([]any{})
}
```

- [ ] **Step 3: Replace worker resume payload map with typed struct**

```go
result := &approval.ApprovalResult{
    Approved: task != nil && task.Status == "approved",
}
if task != nil {
    if task.DisapproveReason != "" {
        result.DisapproveReason = &task.DisapproveReason
    }
    result.Comment = task.Comment
    result.ApprovedBy = fmt.Sprintf("%d", task.ApprovedBy)
    if task.DecidedAt != nil {
        result.ApprovedAt = task.DecidedAt // decision timestamp (approve/reject)
    }
}
targets[task.ToolCallID] = result
```

- [ ] **Step 4: Run the same focused worker tests and confirm pass**

Run: `go test ./internal/service/ai/logic -run "TestApprovalWorker_ResumesApprovedTask|TestApprovalWorker_SubmitApprovalWritesAuditEvents" -v`  
Expected: PASS.

- [ ] **Step 5: Commit implementation**

```bash
git add internal/ai/common/approval/register.go internal/ai/common/middleware/approval.go internal/service/ai/logic/approval_worker.go internal/service/ai/logic/approval_worker_test.go
git commit -m "fix: use typed ApprovalResult payload in approval worker resume"
```

## Chunk 2: Deprecated ResumeApproval Path + Regression Coverage

### Task 3: Add failing test for deprecated resume payload semantics

**Files:**
- Modify: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Add a focused failing test around resume payload shape**

```go
func TestResumeApproval_UsesTypedApprovalResultPayload(t *testing.T) {
    // seed approval task + stub runner resume hook to capture ResumeParams
    // call ResumeApproval with Approved=false and Reason set
    // assert target type is *approval.ApprovalResult
    // assert DisapproveReason present
    // assert ApprovedAt is non-nil (decision timestamp semantics)
}
```

- [ ] **Step 2: Run the single test and confirm failure**

Run: `go test ./internal/service/ai/logic -run TestResumeApproval_UsesTypedApprovalResultPayload -v`  
Expected: FAIL because current code builds `map[string]any`.

- [ ] **Step 3: Commit failing baseline**

```bash
git add internal/service/ai/logic/logic_test.go
git commit -m "test: lock typed payload contract for deprecated ResumeApproval"
```

### Task 4: Implement deprecated path fix and complete verification

**Files:**
- Modify: `internal/service/ai/logic/logic.go`
- Modify: `internal/service/ai/logic/logic_test.go`
- Test: `internal/service/ai/logic/approval_worker_test.go`
- Test: `internal/service/ai/logic/logic_test.go`

- [ ] **Step 1: Switch `ResumeApproval` payload to `*approval.ApprovalResult`**

```go
approvalResult := &approval.ApprovalResult{
    Approved:   input.Approved,
    Comment:    input.Comment,
    ApprovedBy: fmt.Sprintf("%d", input.UserID),
}
if !input.Approved && input.Reason != "" {
    approvalResult.DisapproveReason = &input.Reason
}
now := time.Now()
approvalResult.ApprovedAt = &now // decision timestamp (approve/reject)
```

- [ ] **Step 2: Ensure imports compile cleanly**

Add/keep:
- `github.com/cy77cc/OpsPilot/internal/ai/common/approval`
- `fmt` (already present)

Remove any newly unused imports if introduced.

- [ ] **Step 3: Run targeted logic tests**

Run: `go test ./internal/service/ai/logic -run "TestResumeApproval_UsesTypedApprovalResultPayload|TestApprovalWorker_ResumesApprovedTask|TestApprovalWorker_SubmitApprovalWritesAuditEvents" -v`  
Expected: PASS.

- [ ] **Step 4: Run approval-focused package verification**

Run: `go test ./internal/service/ai/logic/... -run Approval -v`  
Expected: PASS for approval-related tests.

- [ ] **Step 5: Run middleware package verification**

Run: `go test ./internal/ai/common/middleware/... -v`  
Expected: PASS; no registration regression after moving `ApprovalResult` registration to owner package.

- [ ] **Step 6: Commit final change set**

```bash
git add internal/service/ai/logic/logic.go internal/service/ai/logic/logic_test.go
git commit -m "fix: align deprecated resume path with typed approval payload contract"
```

## Implementation Rules

- Apply `@test-driven-development`: tests first, minimal implementation, then broaden verification.
- Apply `@verification-before-completion`: do not claim completion until all verification commands pass.
- Keep changes DRY/YAGNI: no payload schema changes beyond typed resume target contract.
- Keep commits small and sequential as listed above.

## Final Verification Checklist

- [ ] `buildApprovalResumeParams` returns `*approval.ApprovalResult` target payload.
- [ ] Deprecated `ResumeApproval` returns `*approval.ApprovalResult` target payload.
- [ ] `ApprovedAt` is treated as decision timestamp in both approve and reject cases.
- [ ] Registration for `ApprovalResult` lives in `internal/ai/common/approval/register.go`.
- [ ] Approval worker tests updated and passing.
- [ ] Approval-focused logic tests passing.
- [ ] Middleware tests passing.

