package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

func TestApprovalWriteModel_SubmitDecisionEmitsApprovalDecidedAndIsIdempotent(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	ctx := WithApprovalSubmitIdempotencyKey(context.Background(), "idem-submit-1")
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-submit",
		userID:             42,
		runID:              "run-submit",
		userMessageID:      "msg-submit-user",
		assistantMessageID: "msg-submit-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-submit",
		CheckpointID:   "checkpoint-submit",
		SessionID:      "session-submit",
		RunID:          "run-submit",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-submit",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(5 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})

	writeModel := NewApprovalWriteModel(db)
	result, err := writeModel.SubmitApproval(ctx, SubmitApprovalInput{
		ApprovalID: "approval-submit",
		Approved:   true,
		Comment:    "ship it",
		UserID:     42,
	})
	if err != nil {
		t.Fatalf("submit approval: %v", err)
	}
	if result == nil || result.Status != "approved" {
		t.Fatalf("expected approved result, got %#v", result)
	}

	task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(ctx, "approval-submit")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil || task.Status != "approved" {
		t.Fatalf("expected task to be approved, got %#v", task)
	}
	if task.ApprovedBy != 42 {
		t.Fatalf("expected approved_by=42, got %d", task.ApprovedBy)
	}

	outbox := mustLoadApprovalOutbox(t, db, "approval-submit", ApprovalEventTypeDecided)
	if outbox.EventID == "" || outbox.Sequence <= 0 || outbox.AggregateID != "run-submit" {
		t.Fatalf("expected envelope fields to be populated, got %#v", outbox)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(outbox.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if approved, _ := payload["approved"].(bool); !approved {
		t.Fatalf("expected approved payload=true, got %#v", payload)
	}
	if comment, _ := payload["comment"].(string); comment != "ship it" {
		t.Fatalf("expected comment in payload, got %#v", payload)
	}

	duplicate, err := writeModel.SubmitApproval(ctx, SubmitApprovalInput{
		ApprovalID: "approval-submit",
		Approved:   true,
		Comment:    "ship it",
		UserID:     42,
	})
	if err != nil {
		t.Fatalf("duplicate submit: %v", err)
	}
	if duplicate == nil || duplicate.Status != "approved" {
		t.Fatalf("expected duplicate submit to return approved snapshot, got %#v", duplicate)
	}
	if duplicate.Message != result.Message {
		t.Fatalf("expected duplicate submit to replay first snapshot message, got %#v", duplicate)
	}
	if count := mustCountApprovalOutbox(t, db, "approval-submit", ApprovalEventTypeDecided); count != 1 {
		t.Fatalf("expected a single decision outbox row, got %d", count)
	}
}

func TestApprovalWriteModel_SubmitDecisionRejectsLockedTaskAndRollsBackOnOutboxFailure(t *testing.T) {
	t.Run("locked task returns snapshot", func(t *testing.T) {
		db := newApprovalWorkerTestDB(t)
		now := time.Now().UTC().Truncate(time.Millisecond)
		seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
			sessionID:          "session-locked",
			userID:             42,
			runID:              "run-locked",
			userMessageID:      "msg-locked-user",
			assistantMessageID: "msg-locked-assistant",
			runStatus:          "waiting_approval",
		})
		seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
			ApprovalID:     "approval-locked",
			CheckpointID:   "checkpoint-locked",
			SessionID:      "session-locked",
			RunID:          "run-locked",
			UserID:         42,
			ToolName:       "exec_command",
			ToolCallID:     "tool-call-locked",
			ArgumentsJSON:  `{"cmd":"whoami"}`,
			PreviewJSON:    `{}`,
			Status:         "pending",
			TimeoutSeconds: 300,
			ExpiresAt:      ptrTime(now.Add(5 * time.Minute)),
			LockExpiresAt:  ptrTime(now.Add(5 * time.Minute)),
		})

		writeModel := NewApprovalWriteModel(db)
		result, err := writeModel.SubmitApproval(context.Background(), SubmitApprovalInput{
			ApprovalID: "approval-locked",
			Approved:   true,
			UserID:     42,
		})
		if err != nil {
			t.Fatalf("submit locked approval: %v", err)
		}
		if result == nil || result.Status != "pending" {
			t.Fatalf("expected pending snapshot for locked task, got %#v", result)
		}
		if count := mustCountApprovalOutbox(t, db, "approval-locked", ApprovalEventTypeDecided); count != 0 {
			t.Fatalf("expected no outbox row for locked task, got %d", count)
		}
	})

	t.Run("outbox failure rolls back transition", func(t *testing.T) {
		db := newApprovalWorkerTestDB(t)
		now := time.Now().UTC().Truncate(time.Millisecond)
		seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
			sessionID:          "session-rollback",
			userID:             42,
			runID:              "run-rollback",
			userMessageID:      "msg-rollback-user",
			assistantMessageID: "msg-rollback-assistant",
			runStatus:          "waiting_approval",
		})
		seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
			ApprovalID:     "approval-rollback",
			CheckpointID:   "checkpoint-rollback",
			SessionID:      "session-rollback",
			RunID:          "run-rollback",
			UserID:         42,
			ToolName:       "exec_command",
			ToolCallID:     "tool-call-rollback",
			ArgumentsJSON:  `{"cmd":"hostname"}`,
			PreviewJSON:    `{}`,
			Status:         "pending",
			TimeoutSeconds: 300,
			ExpiresAt:      ptrTime(now.Add(5 * time.Minute)),
		})

		callbackName := "test:approval_write_model_outbox_failure"
		if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement == nil || tx.Statement.Schema == nil {
				return
			}
			if tx.Statement.Schema.Table != "ai_approval_outbox_events" {
				return
			}
			tx.AddError(errors.New("simulated outbox failure"))
		}); err != nil {
			t.Fatalf("register outbox failure callback: %v", err)
		}
		t.Cleanup(func() {
			_ = db.Callback().Create().Remove(callbackName)
		})

		writeModel := NewApprovalWriteModel(db)
		result, err := writeModel.SubmitApproval(context.Background(), SubmitApprovalInput{
			ApprovalID: "approval-rollback",
			Approved:   true,
			UserID:     42,
		})
		if err == nil {
			t.Fatalf("expected outbox failure to abort transaction, got result %#v", result)
		}

		task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-rollback")
		if err != nil {
			t.Fatalf("reload task: %v", err)
		}
		if task == nil || task.Status != "pending" {
			t.Fatalf("expected pending task after rollback, got %#v", task)
		}
		if count := mustCountApprovalOutbox(t, db, "approval-rollback", ApprovalEventTypeDecided); count != 0 {
			t.Fatalf("expected no outbox row after rollback, got %d", count)
		}
	})
}

func TestApprovalWriteModel_RunLifecycleWritesResumeEventsAndStatus(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-resume",
		userID:             42,
		runID:              "run-resume",
		userMessageID:      "msg-resume-user",
		assistantMessageID: "msg-resume-assistant",
		runStatus:          "approved",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-resume",
		CheckpointID:   "checkpoint-resume",
		SessionID:      "session-resume",
		RunID:          "run-resume",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-resume",
		ArgumentsJSON:  `{"cmd":"id"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(5 * time.Minute)),
		LockExpiresAt:  ptrTime(now.Add(5 * time.Minute)),
		DecidedAt:      ptrTime(now),
	})

	writeModel := NewApprovalWriteModel(db)
	if err := writeModel.EmitRunResuming(context.Background(), "approval-resume"); err != nil {
		t.Fatalf("emit run resuming: %v", err)
	}
	if err := writeModel.EmitRunResumed(context.Background(), "approval-resume"); err != nil {
		t.Fatalf("emit run resumed: %v", err)
	}
	if err := writeModel.EmitRunCompleted(context.Background(), "approval-resume", "completed"); err != nil {
		t.Fatalf("emit run completed: %v", err)
	}

	var run model.AIRun
	if err := db.First(&run, "id = ?", "run-resume").Error; err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run.Status != "completed" {
		t.Fatalf("expected completed run status, got %q", run.Status)
	}

	events := mustLoadApprovalOutboxByApprovalID(t, db, "approval-resume")
	if len(events) != 3 {
		t.Fatalf("expected 3 lifecycle events, got %d", len(events))
	}
	if events[0].EventType != RunEventTypeResuming || events[1].EventType != RunEventTypeResumed || events[2].EventType != RunEventTypeCompleted {
		t.Fatalf("unexpected lifecycle event order: %#v", events)
	}
	if !(events[0].Sequence < events[1].Sequence && events[1].Sequence < events[2].Sequence) {
		t.Fatalf("expected monotonic sequence values, got %#v", events)
	}
}

func TestApprovalWriteModel_ResumeFailureEmitsRetryableFlag(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-resume-fail",
		userID:             42,
		runID:              "run-resume-fail",
		userMessageID:      "msg-resume-fail-user",
		assistantMessageID: "msg-resume-fail-assistant",
		runStatus:          "approved",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-resume-fail",
		CheckpointID:   "checkpoint-resume-fail",
		SessionID:      "session-resume-fail",
		RunID:          "run-resume-fail",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-resume-fail",
		ArgumentsJSON:  `{"cmd":"false"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(5 * time.Minute)),
		LockExpiresAt:  ptrTime(now.Add(5 * time.Minute)),
		DecidedAt:      ptrTime(now),
	})

	writeModel := NewApprovalWriteModel(db)
	if err := writeModel.EmitRunResumeFailed(context.Background(), "approval-resume-fail", true, errors.New("boom")); err != nil {
		t.Fatalf("emit run resume failed: %v", err)
	}

	outbox := mustLoadApprovalOutbox(t, db, "approval-resume-fail", RunEventTypeResumeFailed)
	var payload map[string]any
	if err := json.Unmarshal([]byte(outbox.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if retryable, _ := payload["retryable"].(bool); !retryable {
		t.Fatalf("expected retryable payload=true, got %#v", payload)
	}

	var run model.AIRun
	if err := db.First(&run, "id = ?", "run-resume-fail").Error; err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run.Status != "resume_failed_retryable" {
		t.Fatalf("expected retryable failure status, got %q", run.Status)
	}
}

func TestApprovalWriteModel_LeaseRenewalAndTakeover(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-lease",
		CheckpointID:   "checkpoint-lease",
		SessionID:      "session-lease",
		RunID:          "run-lease",
		UserID:         42,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-lease",
		ArgumentsJSON:  `{"cmd":"sleep 1"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(5 * time.Minute)),
		LockExpiresAt:  ptrTime(now.Add(30 * time.Second)),
		DecidedAt:      ptrTime(now),
	})

	writeModel := NewApprovalWriteModel(db)
	renewed, err := writeModel.RenewApprovalLease(context.Background(), "approval-lease", now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("renew approval lease: %v", err)
	}
	if !renewed {
		t.Fatal("expected renewal to succeed")
	}

	task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-lease")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task.LockExpiresAt == nil || !task.LockExpiresAt.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("expected renewed lease to be persisted, got %#v", task.LockExpiresAt)
	}

	if err := db.Model(&model.AIApprovalTask{}).
		Where("approval_id = ?", "approval-lease").
		Updates(map[string]any{"lock_expires_at": now.Add(-1 * time.Minute)}).Error; err != nil {
		t.Fatalf("force lease expiry: %v", err)
	}
	stolen, err := writeModel.AcquireApprovalLease(context.Background(), "approval-lease", now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("acquire approval lease after expiry: %v", err)
	}
	if !stolen {
		t.Fatal("expected expired lease to be acquired")
	}
}

func mustLoadApprovalOutbox(t *testing.T, db *gorm.DB, approvalID, eventType string) model.AIApprovalOutboxEvent {
	t.Helper()
	var event model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", approvalID, eventType).First(&event).Error; err != nil {
		t.Fatalf("load outbox event %s/%s: %v", approvalID, eventType, err)
	}
	return event
}

func mustLoadApprovalOutboxByApprovalID(t *testing.T, db *gorm.DB, approvalID string) []model.AIApprovalOutboxEvent {
	t.Helper()
	var events []model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ?", approvalID).Order("sequence ASC, id ASC").Find(&events).Error; err != nil {
		t.Fatalf("load approval outbox events: %v", err)
	}
	return events
}

func mustCountApprovalOutbox(t *testing.T, db *gorm.DB, approvalID, eventType string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&model.AIApprovalOutboxEvent{}).Where("approval_id = ? AND event_type = ?", approvalID, eventType).Count(&count).Error; err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	return count
}
