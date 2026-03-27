package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

func TestApprovalExpirer_MarksExpiredAndWritesOutbox(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-expirer",
		CheckpointID:   "checkpoint-expirer",
		SessionID:      "session-expirer",
		RunID:          "run-expirer",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-expirer",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(-1 * time.Minute)),
	})

	expirer := NewApprovalExpirer(newApprovalWorkerTestLogic(db))
	claimed, err := expirer.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run expirer: %v", err)
	}
	if !claimed {
		t.Fatal("expected expirer to claim an expired approval")
	}

	task, err := newApprovalWorkerTestLogic(db).ApprovalDAO.GetByApprovalID(context.Background(), "approval-expirer")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil {
		t.Fatal("expected expired task to exist")
	}
	if task.Status != "expired" {
		t.Fatalf("expected expired task status, got %q", task.Status)
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-expirer", ApprovalEventTypeExpired).First(&outbox).Error; err != nil {
		t.Fatalf("load expired outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected pending outbox row, got %q", outbox.Status)
	}
	if outbox.ToolCallID != "tool-call-expirer" {
		t.Fatalf("expected tool_call_id to persist, got %q", outbox.ToolCallID)
	}
}

func TestApprovalExpirer_RollsBackOnOutboxFailure(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-expirer-rollback",
		CheckpointID:   "checkpoint-expirer-rollback",
		SessionID:      "session-expirer-rollback",
		RunID:          "run-expirer-rollback",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-expirer-rollback",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(-1 * time.Minute)),
	})

	db.Callback().Create().Before("gorm:create").Register("approval_expirer_outbox_failure", func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Schema != nil && tx.Statement.Schema.Table == "ai_approval_outbox_events" {
			tx.AddError(errors.New("simulated outbox failure"))
		}
	})

	expirer := NewApprovalExpirer(newApprovalWorkerTestLogic(db))
	claimed, err := expirer.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected expirer to fail when outbox write fails")
	}
	if claimed {
		t.Fatal("expected failed expiration attempt not to report a committed claim")
	}

	task, err := newApprovalWorkerTestLogic(db).ApprovalDAO.GetByApprovalID(context.Background(), "approval-expirer-rollback")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task to remain pending after rollback")
	}
	if task.Status != "pending" {
		t.Fatalf("expected pending task after rollback, got %q", task.Status)
	}

	var outboxCount int64
	if err := db.Model(&model.AIApprovalOutboxEvent{}).Where("approval_id = ?", "approval-expirer-rollback").Count(&outboxCount).Error; err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected no outbox rows after rollback, got %d", outboxCount)
	}
}
