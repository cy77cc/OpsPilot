# AI Approval Type Fix Design

**Date:** 2026-03-26
**Status:** Approved
**Type:** Bug Fix

## Problem Statement

The approval resume flow fails due to type mismatch between the resume parameters and the middleware expectations.

### Root Cause

```
Sender: buildApprovalResumeParams returns map[string]any
Receiver: middleware expects *approval.ApprovalResult
Result: Type mismatch causes GetResumeContext to return hasData=false
```

### Impact

- Approval resume never completes successfully
- Users approve operations but execution does not continue
- Run status stuck in "waiting_approval"

## Solution

Align the resume parameter type with Eino framework conventions.

### Changes Overview

| File | Change |
|------|--------|
| `internal/ai/common/approval/register.go` (new) | Add `schema.Register[*ApprovalResult]()` |
| `internal/service/ai/logic/approval_worker.go` | Add import + Modify `buildApprovalResumeParams` |
| `internal/service/ai/logic/logic.go` | Add import + Fix deprecated `ResumeApproval` method |
| `internal/service/ai/logic/approval_worker_test.go` | Add import + Update type assertions (2 locations) |

## Detailed Changes

### 1. Add Type Registration (Type Owner Package)

**File:** `internal/ai/common/approval/register.go` (new file)

```go
package approval

import "github.com/cloudwego/eino/schema"

func init() {
    schema.Register[*ApprovalResult]()
}
```

**Note:** Register `ApprovalResult` in its owner package to avoid import-order coupling with middleware package initialization.

**Rationale:** Eino framework uses gob serialization for interrupt/resume state. Types must be registered before use.

### 2. Fix buildApprovalResumeParams

**File:** `internal/service/ai/logic/approval_worker.go`

**Import Change (REQUIRED):** The file does NOT currently import the approval package. Add this import:

```go
import (
    // ... existing imports ...

    "github.com/cloudwego/eino/adk"
    "github.com/cy77cc/OpsPilot/internal/ai/common/approval"  // ADD THIS LINE
    airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
    // ... rest of imports ...
)
```

**Before (lines 773-795):**
```go
func buildApprovalResumeParams(task *model.AIApprovalTask) *adk.ResumeParams {
    payload := map[string]any{
        "approved":          task != nil && task.Status == "approved",
        "disapprove_reason": "",
        "comment":           "",
        "approved_by":       uint64(0),
        "approved_at":       "",
    }
    if task != nil {
        payload["disapprove_reason"] = task.DisapproveReason
        payload["comment"] = task.Comment
        payload["approved_by"] = task.ApprovedBy
        if task.DecidedAt != nil {
            payload["approved_at"] = task.DecidedAt.UTC().Format(time.RFC3339)
        }
    }

    targets := map[string]any{}
    if task != nil && strings.TrimSpace(task.ToolCallID) != "" {
        targets[task.ToolCallID] = payload
    }
    return &adk.ResumeParams{Targets: targets}
}
```

**After:**
```go
func buildApprovalResumeParams(task *model.AIApprovalTask) *adk.ResumeParams {
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
            // Decision timestamp semantics: set for both approved and rejected.
            result.ApprovedAt = task.DecidedAt
        }
    }

    targets := map[string]any{}
    if task != nil && strings.TrimSpace(task.ToolCallID) != "" {
        targets[task.ToolCallID] = result  // Pass *ApprovalResult, not map
    }
    return &adk.ResumeParams{Targets: targets}
}
```

### 3. Fix Deprecated ResumeApproval Method

**File:** `internal/service/ai/logic/logic.go`

**Import Change (REQUIRED):** The file does NOT currently import the approval package. Add this import:

```go
import (
    // ... existing imports ...

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/schema"
    "github.com/cy77cc/OpsPilot/internal/ai/common/approval"  // ADD THIS LINE
    aicore "github.com/cy77cc/OpsPilot/internal/ai"
    // ... rest of imports ...
)
```

**Before (lines 1691-1723):**
```go
approvalResult := map[string]any{
    "approved":          input.Approved,
    "disapprove_reason": input.Reason,
    "comment":           input.Comment,
    "approved_by":       input.UserID,
    "approved_at":       time.Now().Format(time.RFC3339),
}
// ...
resumeParams := &adk.ResumeParams{
    Targets: map[string]any{
        task.ToolCallID: approvalResult,
    },
}
```

**After:**
```go
approvalResult := &approval.ApprovalResult{
    Approved:        input.Approved,
    Comment:         input.Comment,
    ApprovedBy:      fmt.Sprintf("%d", input.UserID),
}
if !input.Approved && input.Reason != "" {
    approvalResult.DisapproveReason = &input.Reason
}
// Decision timestamp semantics: set for both approved and rejected.
now := time.Now()
approvalResult.ApprovedAt = &now

resumeParams := &adk.ResumeParams{
    Targets: map[string]any{
        task.ToolCallID: approvalResult,  // Pass *ApprovalResult
    },
}
```

### 4. Update Tests

**File:** `internal/service/ai/logic/approval_worker_test.go`

**Import Change (REQUIRED):** Add this import:

```go
import (
    // ... existing imports ...
    "github.com/cy77cc/OpsPilot/internal/ai/common/approval"  // ADD THIS LINE
    // ... rest of imports ...
)
```

**Two locations need updates:**

#### Location 1: Line ~668 in `TestApprovalWorker_ResumesApprovedTask`

```go
// Before:
target, ok := capturedParams.Targets["tool-call-resume"].(map[string]any)
if !ok {
    t.Fatalf("expected resume target payload, got %#v", capturedParams.Targets)
}
if approved, _ := target["approved"].(bool); !approved {
    t.Fatalf("expected persisted approval in resume params, got %#v", target)
}
if comment, _ := target["comment"].(string); comment != "looks good" {
    t.Fatalf("expected persisted comment in resume params, got %#v", target)
}

// After:
target, ok := capturedParams.Targets["tool-call-resume"].(*approval.ApprovalResult)
if !ok {
    t.Fatalf("expected *approval.ApprovalResult, got %#v", capturedParams.Targets)
}
if !target.Approved {
    t.Fatalf("expected Approved=true, got %#v", target)
}
if target.Comment != "looks good" {
    t.Fatalf("expected Comment='looks good', got %q", target.Comment)
}
```

#### Location 2: Line ~1219 in `TestApprovalWorker_SubmitApprovalWritesAuditEvents`

```go
// Before:
target, ok := params.Targets[task.ToolCallID].(map[string]any)
if !ok {
    t.Fatalf("expected resume params target for %q, got %#v", task.ToolCallID, params.Targets)
}
if target["comment"] != "ship it" {
    t.Fatalf("expected resume params to include persisted comment, got %#v", target)
}

// After:
target, ok := params.Targets[task.ToolCallID].(*approval.ApprovalResult)
if !ok {
    t.Fatalf("expected *approval.ApprovalResult for %q, got %#v", task.ToolCallID, params.Targets)
}
if target.Comment != "ship it" {
    t.Fatalf("expected Comment='ship it', got %q", target.Comment)
}
```

## Data Flow After Fix

```
User approves → SubmitApproval → ApprovalWorker
                                      ↓
                      buildApprovalResumeParams
                                      ↓
                      Targets[toolCallID] = *ApprovalResult
                                      ↓
                      runner.ResumeWithParams()
                                      ↓
                      middleware: GetResumeContext[*ApprovalResult](ctx)
                                      ↓
                      Type matches → Execute tool or return rejection
```

## Verification

### Unit Tests

```bash
go test ./internal/service/ai/logic/... -v -run Approval
```

Expected: All tests pass with updated type assertions.

### Integration Test Scenarios

| Scenario | Steps | Expected Result |
|----------|-------|-----------------|
| **Approve flow** | 1. Trigger approval-requiring tool<br>2. Approve via UI<br>3. Watch for tool execution | Tool executes with original arguments |
| **Reject flow** | 1. Trigger approval-requiring tool<br>2. Reject with reason<br>3. Watch for response | Agent receives "tool X disapproved: reason" message |
| **GetResumeContext validation** | Add debug log in middleware | Shows `hasData=true` and correct `result.Approved` value |

### Verification Commands

```bash
# 1. Run all approval-related tests
go test ./internal/service/ai/logic/... -v -run "Approval"

# 2. Run middleware tests
go test ./internal/ai/common/middleware/... -v

# 3. Build and run server for manual test
make build-all && ./opspilot --config configs/config.yaml
```

## Implementation Checklist

- [ ] Add `internal/ai/common/approval/register.go` with `schema.Register[*ApprovalResult]()`
- [ ] Add import `"github.com/cy77cc/OpsPilot/internal/ai/common/approval"` to `approval_worker.go`
- [ ] Modify `buildApprovalResumeParams` in `approval_worker.go`
- [ ] Add import `"github.com/cy77cc/OpsPilot/internal/ai/common/approval"` to `logic.go`
- [ ] Fix `ResumeApproval` in `logic.go`
- [ ] Add import to `approval_worker_test.go`
- [ ] Update test at line ~668 in `approval_worker_test.go`
- [ ] Update test at line ~1219 in `approval_worker_test.go`
- [ ] Run tests: `go test ./internal/service/ai/logic/... -v -run Approval`
- [ ] Run middleware tests: `go test ./internal/ai/common/middleware/... -v`

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Missing import | Low | Medium | Explicit checklist item |
| Test failures after update | Low | Medium | Update all type assertions in tests |
| Registration not initialized | Low | High | Register in owner package (`approval`) |
| Regression in deprecated code | Low | Low | Method already deprecated, fix for consistency |

## References

- Eino examples: `/root/learn/eino-examples/adk/common/tool/approval_wrapper.go`
- Eino examples test: `/root/learn/eino-examples/adk/human-in-the-loop/1_approval/main.go`
- Eino documentation: https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_07_interrupt_resume/
