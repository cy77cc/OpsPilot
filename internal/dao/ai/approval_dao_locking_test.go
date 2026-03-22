package ai

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

func TestApproveSetsLockLeaseAtomically(t *testing.T) {
	runApproveSetsLockLeaseAtomically(t)
}

func TestApprovalApproveSetsLockLeaseAtomically(t *testing.T) {
	runApproveSetsLockLeaseAtomically(t)
}

func runApproveSetsLockLeaseAtomically(t *testing.T) {
	t.Helper()
	db := newApprovalLockTestDB(t)
	dao := NewAIApprovalTaskDAO(db)
	ctx := context.Background()

	task := &model.AIApprovalTask{
		ApprovalID:     "approval-lock-1",
		CheckpointID:   "checkpoint-1",
		SessionID:      "session-1",
		RunID:          "run-1",
		UserID:         101,
		ToolName:       "dangerous_tool",
		ToolCallID:     "tool-call-1",
		ArgumentsJSON:  `{"x":1}`,
		PreviewJSON:    `{"preview":true}`,
		Status:         "pending",
		TimeoutSeconds: 300,
	}
	if err := dao.Create(ctx, task); err != nil {
		t.Fatalf("create approval task: %v", err)
	}

	updated, err := dao.ApproveWithLease(ctx, task.ApprovalID, 202, "looks good", 5*time.Minute)
	if err != nil {
		t.Fatalf("approve with lease: %v", err)
	}
	if updated == nil {
		t.Fatal("expected approved task")
	}
	if updated.Status != "approved" {
		t.Fatalf("expected approved status, got %q", updated.Status)
	}
	if updated.ApprovedBy != 202 {
		t.Fatalf("expected approved_by to be set, got %d", updated.ApprovedBy)
	}
	if updated.DecidedAt == nil || updated.DecidedAt.IsZero() {
		t.Fatal("expected decided_at to be set")
	}
	if updated.LockExpiresAt == nil || !updated.LockExpiresAt.After(time.Now()) {
		t.Fatalf("expected lock lease to be set in the future, got %#v", updated.LockExpiresAt)
	}
}

func TestDecisionRejectedAfterLock(t *testing.T) {
	runDecisionRejectedAfterLock(t)
}

func TestApprovalDecisionRejectedAfterLock(t *testing.T) {
	runDecisionRejectedAfterLock(t)
}

func runDecisionRejectedAfterLock(t *testing.T) {
	t.Helper()
	db := newApprovalLockTestDB(t)
	dao := NewAIApprovalTaskDAO(db)
	ctx := context.Background()

	task := &model.AIApprovalTask{
		ApprovalID:     "approval-lock-2",
		CheckpointID:   "checkpoint-2",
		SessionID:      "session-2",
		RunID:          "run-2",
		UserID:         102,
		ToolName:       "dangerous_tool",
		ToolCallID:     "tool-call-2",
		ArgumentsJSON:  `{"x":2}`,
		PreviewJSON:    `{"preview":true}`,
		Status:         "pending",
		TimeoutSeconds: 300,
	}
	if err := dao.Create(ctx, task); err != nil {
		t.Fatalf("create approval task: %v", err)
	}

	if _, err := dao.ApproveWithLease(ctx, task.ApprovalID, 303, "approve first", 5*time.Minute); err != nil {
		t.Fatalf("approve with lease: %v", err)
	}
	if _, err := dao.RejectPending(ctx, task.ApprovalID, 404, "too late", "reject after lock"); err != nil {
		t.Fatalf("reject pending after lock: %v", err)
	}

	got, err := dao.GetByApprovalID(ctx, task.ApprovalID)
	if err != nil {
		t.Fatalf("reload approval task: %v", err)
	}
	if got == nil {
		t.Fatal("expected approval task")
	}
	if got.Status != "approved" {
		t.Fatalf("expected locked approval to remain approved, got %q", got.Status)
	}
}

func TestStealExpiredLock(t *testing.T) {
	runStealExpiredLock(t)
}

func TestApprovalStealExpiredLock(t *testing.T) {
	runStealExpiredLock(t)
}

func runStealExpiredLock(t *testing.T) {
	t.Helper()
	db := newApprovalLockTestDB(t)
	dao := NewAIApprovalTaskDAO(db)
	ctx := context.Background()

	expired := time.Now().Add(-time.Minute)
	task := &model.AIApprovalTask{
		ApprovalID:     "approval-lock-3",
		CheckpointID:   "checkpoint-3",
		SessionID:      "session-3",
		RunID:          "run-3",
		UserID:         103,
		ToolName:       "dangerous_tool",
		ToolCallID:     "tool-call-3",
		ArgumentsJSON:  `{"x":3}`,
		PreviewJSON:    `{"preview":true}`,
		Status:         "approved_locked",
		ApprovedBy:     303,
		LockExpiresAt:  &expired,
		TimeoutSeconds: 300,
	}
	if err := dao.Create(ctx, task); err != nil {
		t.Fatalf("create locked approval task: %v", err)
	}

	updated, err := dao.AcquireOrStealLease(ctx, task.ApprovalID, 5*time.Minute)
	if err != nil {
		t.Fatalf("acquire or steal lease: %v", err)
	}
	if updated == nil {
		t.Fatal("expected expired lock to be stealable")
	}
	if updated.Status != "approved_locked" {
		t.Fatalf("expected approved_locked status, got %q", updated.Status)
	}
	if updated.LockExpiresAt == nil || !updated.LockExpiresAt.After(time.Now()) {
		t.Fatalf("expected lock lease to be refreshed, got %#v", updated.LockExpiresAt)
	}
}

func newApprovalLockTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := newAIDAOTestDB(t)
	if err := db.AutoMigrate(&model.AIApprovalTask{}); err != nil {
		t.Fatalf("migrate approval tasks: %v", err)
	}
	return db
}
