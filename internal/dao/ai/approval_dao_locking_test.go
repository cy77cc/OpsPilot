package ai

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApproveSetsLockLeaseAtomically(t *testing.T) {
	testApproveSetsLockLeaseAtomically(t)
}

func TestDecisionRejectedAfterLock(t *testing.T) {
	testDecisionRejectedAfterLock(t)
}

func TestStealExpiredLock(t *testing.T) {
	testStealExpiredLock(t)
}

func TestApprovalApproveSetsLockLeaseAtomically(t *testing.T) {
	testApproveSetsLockLeaseAtomically(t)
}

func TestApprovalDecisionRejectedAfterLock(t *testing.T) {
	testDecisionRejectedAfterLock(t)
}

func TestApprovalStealExpiredLock(t *testing.T) {
	testStealExpiredLock(t)
}

func testApproveSetsLockLeaseAtomically(t *testing.T) {
	db := newApprovalTaskTestDB(t)
	dao := NewAIApprovalTaskDAO(db)
	ctx := context.Background()
	seedApprovalTask(t, db, &model.AIApprovalTask{
		ApprovalID:    "approval-approve",
		CheckpointID:  "checkpoint-1",
		SessionID:     "session-1",
		RunID:         "run-1",
		UserID:        42,
		ToolName:      "exec_command",
		ToolCallID:    "tool-call-1",
		ArgumentsJSON: `{"cmd":"date"}`,
		PreviewJSON:   `{}`,
		Status:        "pending",
	})

	leaseUntil := time.Now().UTC().Add(2 * time.Minute).Truncate(time.Millisecond)
	updated, err := dao.ApproveWithLease(ctx, "approval-approve", 7, "ship it", leaseUntil)
	if err != nil {
		t.Fatalf("approve with lease: %v", err)
	}
	if !updated {
		t.Fatal("expected approve transition to succeed")
	}

	task, err := dao.GetByApprovalID(ctx, "approval-approve")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	if task.Status != "approved" {
		t.Fatalf("expected approved status, got %q", task.Status)
	}
	if task.ApprovedBy != 7 {
		t.Fatalf("expected approved_by=7, got %d", task.ApprovedBy)
	}
	if task.Comment != "ship it" {
		t.Fatalf("expected comment to persist, got %q", task.Comment)
	}
	if task.DecidedAt == nil {
		t.Fatal("expected decided_at to be set")
	}
	if task.LockExpiresAt == nil {
		t.Fatal("expected lock_expires_at to be set")
	}
	if !task.LockExpiresAt.Equal(leaseUntil) {
		t.Fatalf("expected lock lease %s, got %s", leaseUntil, task.LockExpiresAt.UTC())
	}
}

func testDecisionRejectedAfterLock(t *testing.T) {
	db := newApprovalTaskTestDB(t)
	dao := NewAIApprovalTaskDAO(db)
	ctx := context.Background()
	seedApprovalTask(t, db, &model.AIApprovalTask{
		ApprovalID:    "approval-locked",
		CheckpointID:  "checkpoint-2",
		SessionID:     "session-2",
		RunID:         "run-2",
		UserID:        42,
		ToolName:      "exec_command",
		ToolCallID:    "tool-call-2",
		ArgumentsJSON: `{"cmd":"uptime"}`,
		PreviewJSON:   `{}`,
		Status:        "pending",
	})

	leaseUntil := time.Now().UTC().Add(2 * time.Minute).Truncate(time.Millisecond)
	approved, err := dao.ApproveWithLease(ctx, "approval-locked", 11, "approved", leaseUntil)
	if err != nil {
		t.Fatalf("approve with lease: %v", err)
	}
	if !approved {
		t.Fatal("expected approve transition to succeed")
	}

	rejected, err := dao.RejectPending(ctx, "approval-locked", 12, "too risky", "reject after lease")
	if err != nil {
		t.Fatalf("reject pending after lease: %v", err)
	}
	if rejected {
		t.Fatal("expected reject transition to be blocked after approval lock")
	}

	task, err := dao.GetByApprovalID(ctx, "approval-locked")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	if task.Status != "approved" {
		t.Fatalf("expected task to stay approved, got %q", task.Status)
	}
	if task.ApprovedBy != 11 {
		t.Fatalf("expected approved_by to remain 11, got %d", task.ApprovedBy)
	}
	if task.DisapproveReason != "" {
		t.Fatalf("expected disapprove_reason to remain empty, got %q", task.DisapproveReason)
	}
}

func testStealExpiredLock(t *testing.T) {
	db := newApprovalTaskTestDB(t)
	dao := NewAIApprovalTaskDAO(db)
	ctx := context.Background()
	expiredLease := time.Now().UTC().Add(-1 * time.Minute).Truncate(time.Millisecond)
	seedApprovalTask(t, db, &model.AIApprovalTask{
		ApprovalID:    "approval-expired",
		CheckpointID:  "checkpoint-3",
		SessionID:     "session-3",
		RunID:         "run-3",
		UserID:        42,
		ToolName:      "exec_command",
		ToolCallID:    "tool-call-3",
		ArgumentsJSON: `{"cmd":"hostname"}`,
		PreviewJSON:   `{}`,
		Status:        "approved",
		ApprovedBy:    13,
		Comment:       "approved",
		DecidedAt:     &expiredLease,
		LockExpiresAt: &expiredLease,
	})

	newLease := time.Now().UTC().Add(3 * time.Minute).Truncate(time.Millisecond)
	stolen, err := dao.AcquireOrStealLease(ctx, "approval-expired", newLease)
	if err != nil {
		t.Fatalf("acquire or steal lease: %v", err)
	}
	if !stolen {
		t.Fatal("expected expired lease to be stolen")
	}

	task, err := dao.GetByApprovalID(ctx, "approval-expired")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to exist")
	}
	if task.LockExpiresAt == nil {
		t.Fatal("expected lock_expires_at to remain set")
	}
	if !task.LockExpiresAt.Equal(newLease) {
		t.Fatalf("expected new lease %s, got %s", newLease, task.LockExpiresAt.UTC())
	}
}

func newApprovalTaskTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIApprovalTask{}); err != nil {
		t.Fatalf("migrate approval task table: %v", err)
	}
	return db
}

func seedApprovalTask(t *testing.T, db *gorm.DB, task *model.AIApprovalTask) {
	t.Helper()

	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}
}
