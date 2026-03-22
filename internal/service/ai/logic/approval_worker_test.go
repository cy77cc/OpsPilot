package logic

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubmitApprovalOnlyWritesDecisionAndOutbox(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
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
		ApprovalID:       "approval-submit",
		CheckpointID:     "checkpoint-submit",
		SessionID:        "session-submit",
		RunID:            "run-submit",
		UserID:           42,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-submit",
		ArgumentsJSON:    `{"cmd":"date"}`,
		PreviewJSON:      `{}`,
		Status:           "pending",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(5 * time.Minute)),
		DecisionSource:   ptrString("user"),
		PolicyVersion:    ptrString("v1"),
		DisapproveReason: "",
	})

	l := newApprovalWorkerTestLogic(db)

	result, err := l.SubmitApproval(context.Background(), SubmitApprovalInput{
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

	task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-submit")
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	if task == nil {
		t.Fatal("expected approval task to exist")
	}
	if task.Status != "approved" {
		t.Fatalf("expected approved task status, got %q", task.Status)
	}
	if task.ApprovedBy != 42 {
		t.Fatalf("expected approved_by=42, got %d", task.ApprovedBy)
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

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-submit", "approval_decided").First(&outbox).Error; err != nil {
		t.Fatalf("load approval_decided outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected pending outbox row, got %q", outbox.Status)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(outbox.PayloadJSON), &payload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if payload["approval_id"] != "approval-submit" {
		t.Fatalf("expected approval_id in payload, got %#v", payload)
	}
	if approved, _ := payload["approved"].(bool); !approved {
		t.Fatalf("expected approved payload=true, got %#v", payload)
	}
	if comment, _ := payload["comment"].(string); comment != "ship it" {
		t.Fatalf("expected comment in payload, got %#v", payload)
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-submit")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != "waiting_approval" {
		t.Fatalf("expected write-only submit to keep run waiting_approval, got %q", run.Status)
	}

	var eventCount int64
	if err := db.Model(&model.AIRunEvent{}).Where("run_id = ?", "run-submit").Count(&eventCount).Error; err != nil {
		t.Fatalf("count run events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected no resume-side run events, got %d", eventCount)
	}
}

func TestWorkerSkipsExpiredAndRejectedTasks(t *testing.T) {
	testWorkerSkipsExpiredAndRejectedTasks(t)
}

func TestApprovalWorkerSkipsExpiredAndRejectedTasks(t *testing.T) {
	testWorkerSkipsExpiredAndRejectedTasks(t)
}

func testWorkerSkipsExpiredAndRejectedTasks(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-rejected",
		userID:             7,
		runID:              "run-rejected",
		userMessageID:      "msg-rejected-user",
		assistantMessageID: "msg-rejected-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-expired",
		userID:             8,
		runID:              "run-expired",
		userMessageID:      "msg-expired-user",
		assistantMessageID: "msg-expired-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-rejected",
		CheckpointID:     "checkpoint-rejected",
		SessionID:        "session-rejected",
		RunID:            "run-rejected",
		UserID:           7,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-rejected",
		ArgumentsJSON:    `{"cmd":"uptime"}`,
		PreviewJSON:      `{}`,
		Status:           "rejected",
		ApprovedBy:       7,
		DisapproveReason: "too risky",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(5 * time.Minute)),
		DecidedAt:        ptrTime(now),
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-expired",
		CheckpointID:   "checkpoint-expired",
		SessionID:      "session-expired",
		RunID:          "run-expired",
		UserID:         8,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-expired",
		ArgumentsJSON:  `{"cmd":"hostname"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     8,
		Comment:        "approved earlier",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(-1 * time.Minute)),
		DecidedAt:      ptrTime(now.Add(-2 * time.Minute)),
		LockExpiresAt:  ptrTime(now.Add(2 * time.Minute)),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-rejected",
		EventType:   "approval_decided",
		RunID:       "run-rejected",
		SessionID:   "session-rejected",
		PayloadJSON: `{"approval_id":"approval-rejected","approved":false}`,
		Status:      "pending",
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-expired",
		EventType:   "approval_decided",
		RunID:       "run-expired",
		SessionID:   "session-expired",
		PayloadJSON: `{"approval_id":"approval-expired","approved":true}`,
		Status:      "pending",
	})

	resumeCalls := 0
	worker := NewApprovalWorker(newApprovalWorkerTestLogic(db),
		WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			resumeCalls++
			return nil, nil
		}),
	)

	if claimed, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("process rejected approval: %v", err)
	} else if !claimed {
		t.Fatal("expected rejected approval outbox to be claimed")
	}
	if claimed, err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("process expired approval: %v", err)
	} else if !claimed {
		t.Fatal("expected expired approval outbox to be claimed")
	}

	if resumeCalls != 0 {
		t.Fatalf("expected worker to skip resume calls, got %d", resumeCalls)
	}

	var rejectedOutbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-rejected", "approval_decided").First(&rejectedOutbox).Error; err != nil {
		t.Fatalf("reload rejected outbox: %v", err)
	}
	if rejectedOutbox.Status != "done" {
		t.Fatalf("expected rejected outbox done, got %q", rejectedOutbox.Status)
	}

	var expiredOutbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-expired", "approval_decided").First(&expiredOutbox).Error; err != nil {
		t.Fatalf("reload expired outbox: %v", err)
	}
	if expiredOutbox.Status != "done" {
		t.Fatalf("expected expired outbox done, got %q", expiredOutbox.Status)
	}

	expiredTask, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-expired")
	if err != nil {
		t.Fatalf("reload expired task: %v", err)
	}
	if expiredTask == nil {
		t.Fatal("expected expired task to exist")
	}
	if expiredTask.Status != "expired" {
		t.Fatalf("expected worker to transition expired approval to expired, got %q", expiredTask.Status)
	}
}

func TestWorkerSetsRunResumingAndConverges(t *testing.T) {
	testWorkerSetsRunResumingAndConverges(t)
}

func TestApprovalWorkerSetsRunResumingAndConverges(t *testing.T) {
	testWorkerSetsRunResumingAndConverges(t)
}

func testWorkerSetsRunResumingAndConverges(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-resume",
		userID:             52,
		runID:              "run-resume",
		userMessageID:      "msg-resume-user",
		assistantMessageID: "msg-resume-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-resume",
		CheckpointID:     "checkpoint-resume",
		SessionID:        "session-resume",
		RunID:            "run-resume",
		UserID:           52,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-resume",
		ArgumentsJSON:    `{"cmd":"date"}`,
		PreviewJSON:      `{}`,
		Status:           "approved",
		ApprovedBy:       52,
		Comment:          "looks good",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:        ptrTime(now),
		LockExpiresAt:    ptrTime(now.Add(2 * time.Minute)),
		DecisionSource:   ptrString("user"),
		PolicyVersion:    ptrString("v2"),
		DisapproveReason: "",
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-resume",
		EventType:   "approval_decided",
		RunID:       "run-resume",
		SessionID:   "session-resume",
		PayloadJSON: `{"approval_id":"approval-resume","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-resume", "session-resume", &seq, "meta", map[string]any{
		"run_id":     "run-resume",
		"session_id": "session-resume",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	sawResuming := false
	var capturedParams *adk.ResumeParams
	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(ctx context.Context, task *model.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			capturedParams = params
			run, err := aidao.NewAIRunDAO(db).GetRun(ctx, task.RunID)
			if err != nil {
				t.Fatalf("load run during resume: %v", err)
			}
			sawResuming = run != nil && run.Status == "resuming"

			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("resumed output", nil), nil, schema.Assistant, ""))
				gen.Close()
			}()
			return iter, nil
		}),
	)

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim approval_decided outbox")
	}
	if !sawResuming {
		t.Fatal("expected run status to be resuming before ResumeWithParams")
	}
	if capturedParams == nil {
		t.Fatal("expected resume params to be provided")
	}

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

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-resume")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != "completed" {
		t.Fatalf("expected converged run status completed, got %q", run.Status)
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), "run-resume")
	if err != nil {
		t.Fatalf("load run projection: %v", err)
	}
	if projection == nil {
		t.Fatal("expected run projection to be persisted")
	}
	if projection.Status != "completed" {
		t.Fatalf("expected projection status completed, got %q", projection.Status)
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-resume", "approval_decided").First(&outbox).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if outbox.Status != "done" {
		t.Fatalf("expected outbox done after convergence, got %q", outbox.Status)
	}
}

func TestApprovalWorkerResumeKeepsRunAliveOnRecoverableStreamRecvError(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-resume-recv-error",
		userID:             53,
		runID:              "run-resume-recv-error",
		userMessageID:      "msg-resume-recv-user",
		assistantMessageID: "msg-resume-recv-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-resume-recv-error",
		CheckpointID:     "checkpoint-resume-recv-error",
		SessionID:        "session-resume-recv-error",
		RunID:            "run-resume-recv-error",
		UserID:           53,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-resume-recv-error",
		ArgumentsJSON:    `{"cmd":"date"}`,
		PreviewJSON:      `{}`,
		Status:           "approved",
		ApprovedBy:       53,
		Comment:          "resume it",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:        ptrTime(now),
		LockExpiresAt:    ptrTime(now.Add(2 * time.Minute)),
		DecisionSource:   ptrString("user"),
		PolicyVersion:    ptrString("v2"),
		DisapproveReason: "",
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-resume-recv-error",
		EventType:   "approval_decided",
		RunID:       "run-resume-recv-error",
		SessionID:   "session-resume-recv-error",
		PayloadJSON: `{"approval_id":"approval-resume-recv-error","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-resume-recv-error", "session-resume-recv-error", &seq, "meta", map[string]any{
		"run_id":     "run-resume-recv-error",
		"session_id": "session-resume-recv-error",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	streamReader, streamWriter := schema.Pipe[*schema.Message](1)
	go func() {
		defer streamWriter.Close()
		streamWriter.Send(schema.ToolMessage(`{"ok":false,"status":"error"}`, "tool-call-resume-recv-error", schema.WithToolName("exec_command")), nil)
		streamWriter.Send(nil, errors.New("failed to invoke tool[name:exec_command id:tool-call-resume-recv-error]: command denied"))
	}()

	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(ctx context.Context, task *model.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				gen.Send(adk.EventFromMessage(nil, streamReader, schema.Tool, "exec_command"))
				gen.Close()
			}()
			return iter, nil
		}),
	)

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim approval_decided outbox")
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-resume-recv-error")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != "completed_with_tool_errors" {
		t.Fatalf("expected completed_with_tool_errors, got %q", run.Status)
	}

	projection, err := aidao.NewAIRunProjectionDAO(db).GetByRunID(context.Background(), "run-resume-recv-error")
	if err != nil {
		t.Fatalf("load run projection: %v", err)
	}
	if projection == nil || projection.Status != "completed_with_tool_errors" {
		t.Fatalf("expected completed_with_tool_errors projection, got %#v", projection)
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-resume-recv-error", "approval_decided").First(&outbox).Error; err != nil {
		t.Fatalf("reload outbox: %v", err)
	}
	if outbox.Status != "done" {
		t.Fatalf("expected outbox done after recovery, got %q", outbox.Status)
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
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AIRun{},
		&model.AIRunEvent{},
		&model.AIRunProjection{},
		&model.AIRunContent{},
		&model.AIApprovalTask{},
		&model.AIApprovalOutboxEvent{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func newApprovalWorkerTestLogic(db *gorm.DB) *Logic {
	return &Logic{
		svcCtx:           &svc.ServiceContext{DB: db},
		ChatDAO:          aidao.NewAIChatDAO(db),
		RunDAO:           aidao.NewAIRunDAO(db),
		ApprovalDAO:      aidao.NewAIApprovalTaskDAO(db),
		RunEventDAO:      aidao.NewAIRunEventDAO(db),
		RunProjectionDAO: aidao.NewAIRunProjectionDAO(db),
		RunContentDAO:    aidao.NewAIRunContentDAO(db),
	}
}

func seedApprovalWorkerRun(t *testing.T, db *gorm.DB, seed approvalWorkerRunSeed) {
	t.Helper()

	if err := db.Create(&model.AIChatSession{
		ID:     seed.sessionID,
		UserID: seed.userID,
		Scene:  "ai",
		Title:  "approval worker test",
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           seed.userMessageID,
		SessionID:    seed.sessionID,
		SessionIDNum: 1,
		Role:         "user",
		Content:      "please continue",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           seed.assistantMessageID,
		SessionID:    seed.sessionID,
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "",
		Status:       "in_progress",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := db.Create(&model.AIRun{
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

func seedApprovalWorkerTask(t *testing.T, db *gorm.DB, task *model.AIApprovalTask) {
	t.Helper()
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}
}

func seedApprovalWorkerOutbox(t *testing.T, db *gorm.DB, event *model.AIApprovalOutboxEvent) {
	t.Helper()
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("seed approval outbox: %v", err)
	}
}

func ptrString(v string) *string {
	return &v
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
