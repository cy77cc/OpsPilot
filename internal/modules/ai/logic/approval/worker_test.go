package approval

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	aidaoapproval "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/approval"
	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWorkerRunOnce_CallsResumeFuncForApprovedTask(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-approved",
		userID:             42,
		runID:              "run-approved",
		userMessageID:      "msg-approved-user",
		assistantMessageID: "msg-approved-assistant",
		runStatus:          "waiting_approval",
	})

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	seedApprovalWorkerTask(t, db, &ai.AIApprovalTask{
		ApprovalID:     "approval-approved",
		CheckpointID:   "checkpoint-approved",
		SessionID:      "session-approved",
		RunID:          "run-approved",
		UserID:         42,
		ToolName:       "host_exec",
		ToolCallID:     "tool-call-approved",
		ArgumentsJSON:  `{"command":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
		ExpiresAt:      &expiresAt,
		ResumeTargetID: "interrupt-approved",
	})
	seedApprovalWorkerOutbox(t, db, &ai.AIApprovalOutboxEvent{
		ApprovalID:  "approval-approved",
		EventType:   "ai.approval.decided",
		RunID:       "run-approved",
		SessionID:   "session-approved",
		ToolCallID:  "tool-call-approved",
		PayloadJSON: `{"approval_id":"approval-approved","status":"approved"}`,
		Status:      "pending",
	})

	var resumeCalls atomic.Int32
	worker := NewWorker(&Logic{
		SvcCtx:      &svc.ServiceContext{DB: db},
		ChatDAO:     aidaochat.NewAIChatDAO(db),
		RunDAO:      aidao.NewAIRunDAO(db),
		RunEventDAO: aidao.NewAIRunEventDAO(db),
		ApprovalDAO: aidaoapproval.NewAIApprovalTaskDAO(db),
	}, WithWorkerResume(func(ctx context.Context, task *ai.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
		resumeCalls.Add(1)
		if task == nil || task.ApprovalID != "approval-approved" {
			t.Fatalf("unexpected task passed to resume: %#v", task)
		}
		if params == nil || params.Targets["interrupt-approved"] == nil {
			t.Fatalf("expected resume params for interrupt-approved, got %#v", params)
		}
		return nil, nil
	}))

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim approval event")
	}
	if resumeCalls.Load() != 1 {
		t.Fatalf("expected resume to be called once, got %d", resumeCalls.Load())
	}

	var outbox ai.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-approved", "ai.approval.decided").First(&outbox).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if outbox.Status != "done" {
		t.Fatalf("expected outbox to be marked done, got %q", outbox.Status)
	}
}

func TestWorkerRunOnce_RetriesWhenFinalizeFails(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-rejected",
		userID:             42,
		runID:              "run-rejected",
		userMessageID:      "msg-rejected-user",
		assistantMessageID: "msg-rejected-assistant",
		runStatus:          "waiting_approval",
	})

	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	seedApprovalWorkerTask(t, db, &ai.AIApprovalTask{
		ApprovalID:     "approval-rejected",
		CheckpointID:   "checkpoint-rejected",
		SessionID:      "session-rejected",
		RunID:          "run-rejected",
		UserID:         42,
		ToolName:       "host_exec",
		ToolCallID:     "tool-call-rejected",
		ArgumentsJSON:  `{"command":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "rejected",
		ApprovedBy:     42,
		TimeoutSeconds: 300,
		ExpiresAt:      &expiresAt,
	})
	seedApprovalWorkerOutbox(t, db, &ai.AIApprovalOutboxEvent{
		ApprovalID:  "approval-rejected",
		EventType:   "ai.approval.decided",
		RunID:       "run-rejected",
		SessionID:   "session-rejected",
		ToolCallID:  "tool-call-rejected",
		PayloadJSON: `{"approval_id":"approval-rejected","status":"rejected"}`,
		Status:      "pending",
	})

	callbackName := "test:approval_worker_fail_run_update"
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil {
			return
		}
		if tx.Statement.Schema.Table == "ai_runs" {
			tx.AddError(errors.New("forced run update failure"))
		}
	}); err != nil {
		t.Fatalf("register callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(callbackName)
	})

	worker := NewWorker(&Logic{
		SvcCtx:      &svc.ServiceContext{DB: db},
		ChatDAO:     aidaochat.NewAIChatDAO(db),
		RunDAO:      aidao.NewAIRunDAO(db),
		RunEventDAO: aidao.NewAIRunEventDAO(db),
		ApprovalDAO: aidaoapproval.NewAIApprovalTaskDAO(db),
	})

	claimed, err := worker.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected finalize failure to be returned")
	}
	if !claimed {
		t.Fatal("expected worker to claim approval event")
	}

	var outbox ai.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-rejected", "ai.approval.decided").First(&outbox).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected outbox to stay pending for retry, got %q", outbox.Status)
	}
	if outbox.RetryCount != 1 {
		t.Fatalf("expected retry_count=1, got %d", outbox.RetryCount)
	}
}

func TestExpirerRunOnce_ExpiresRunAndEmitsOutbox(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-expired",
		userID:             42,
		runID:              "run-expired",
		userMessageID:      "msg-expired-user",
		assistantMessageID: "msg-expired-assistant",
		runStatus:          "waiting_approval",
	})

	expiresAt := time.Now().UTC().Add(-2 * time.Minute)
	seedApprovalWorkerTask(t, db, &ai.AIApprovalTask{
		ApprovalID:     "approval-expired",
		CheckpointID:   "checkpoint-expired",
		SessionID:      "session-expired",
		RunID:          "run-expired",
		UserID:         42,
		ToolName:       "host_exec",
		ToolCallID:     "tool-call-expired",
		ArgumentsJSON:  `{"command":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
		ExpiresAt:      &expiresAt,
	})

	expirer := NewExpirer(&Logic{
		SvcCtx:      &svc.ServiceContext{DB: db},
		ChatDAO:     aidaochat.NewAIChatDAO(db),
		RunDAO:      aidao.NewAIRunDAO(db),
		RunEventDAO: aidao.NewAIRunEventDAO(db),
		ApprovalDAO: aidaoapproval.NewAIApprovalTaskDAO(db),
	})

	claimed, err := expirer.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if !claimed {
		t.Fatal("expected expirer to process expired task")
	}

	task, err := aidaoapproval.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-expired")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil || task.Status != "expired" {
		t.Fatalf("expected task to be expired, got %#v", task)
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-expired")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil || run.Status != "expired" {
		t.Fatalf("expected run to be expired, got %#v", run)
	}

	var outbox ai.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-expired", "ai.approval.expired").First(&outbox).Error; err != nil {
		t.Fatalf("reload expired outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected expired outbox to be pending, got %q", outbox.Status)
	}
}

type approvalWorkerRunSeed struct {
	sessionID          string
	userID             uint64
	runID              string
	userMessageID      string
	assistantMessageID string
	runStatus          string
}

func newApprovalWorkerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&ai.AIChatSession{},
		&ai.AIChatMessage{},
		&ai.AIRun{},
		&ai.AIRunEvent{},
		&ai.AIRunProjection{},
		&ai.AIRunContent{},
		&ai.AIApprovalTask{},
		&ai.AIApprovalOutboxEvent{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func seedApprovalWorkerRun(t *testing.T, db *gorm.DB, seed approvalWorkerRunSeed) {
	t.Helper()

	if err := db.Create(&ai.AIChatSession{ID: seed.sessionID, UserID: seed.userID, Scene: "ai", Title: "approval worker test"}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(&ai.AIChatMessage{ID: seed.userMessageID, SessionID: seed.sessionID, SessionIDNum: 1, Role: "user", Content: "please continue", Status: "done"}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&ai.AIChatMessage{ID: seed.assistantMessageID, SessionID: seed.sessionID, SessionIDNum: 2, Role: "assistant", Content: "", Status: "streaming"}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := db.Create(&ai.AIRun{
		ID:                 seed.runID,
		SessionID:          seed.sessionID,
		ClientRequestID:    seed.runID,
		UserMessageID:      seed.userMessageID,
		AssistantMessageID: seed.assistantMessageID,
		Status:             seed.runStatus,
		TraceJSON:          `{}`,
	}).Error; err != nil {
		t.Fatalf("seed run: %v", err)
	}
}

func seedApprovalWorkerTask(t *testing.T, db *gorm.DB, task *ai.AIApprovalTask) {
	t.Helper()
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}
}

func seedApprovalWorkerOutbox(t *testing.T, db *gorm.DB, event *ai.AIApprovalOutboxEvent) {
	t.Helper()
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("seed approval outbox: %v", err)
	}
}
