package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/ai/common/approval"
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
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-submit", ApprovalEventTypeDecided).First(&outbox).Error; err != nil {
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

func TestSubmitApproval_DuplicateDecisionIsIdempotentAndDoesNotRepeatWrites(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-duplicate-submit",
		userID:             44,
		runID:              "run-duplicate-submit",
		userMessageID:      "msg-duplicate-submit-user",
		assistantMessageID: "msg-duplicate-submit-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-duplicate-submit",
		CheckpointID:     "checkpoint-duplicate-submit",
		SessionID:        "session-duplicate-submit",
		RunID:            "run-duplicate-submit",
		UserID:           44,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-duplicate-submit",
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
	input := SubmitApprovalInput{
		ApprovalID: "approval-duplicate-submit",
		Approved:   true,
		Comment:    "ship it",
		UserID:     44,
	}

	first, err := l.SubmitApproval(context.Background(), input)
	if err != nil {
		t.Fatalf("first submit approval: %v", err)
	}
	// Intentionally omit the current context idempotency key to lock the broader
	// future contract from the plan: duplicate decisions should not repeat writes.
	second, err := l.SubmitApproval(context.Background(), input)
	if err != nil {
		t.Fatalf("second submit approval: %v", err)
	}
	if first == nil || second == nil {
		t.Fatalf("expected both submit results, got first=%#v second=%#v", first, second)
	}
	if first.Status != "approved" || second.Status != "approved" {
		t.Fatalf("expected approved results, got first=%#v second=%#v", first, second)
	}
	if first.Message != second.Message {
		t.Fatalf("expected duplicate submit to replay the same message, first=%#v second=%#v", first, second)
	}

	task, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-duplicate-submit")
	if err != nil {
		t.Fatalf("reload approval task: %v", err)
	}
	if task == nil || task.Status != "approved" {
		t.Fatalf("expected approved task after duplicate submit, got %#v", task)
	}
	if task.ApprovedBy != 44 {
		t.Fatalf("expected approved_by=44, got %d", task.ApprovedBy)
	}

	var outboxCount int64
	if err := db.Model(&model.AIApprovalOutboxEvent{}).
		Where("approval_id = ? AND event_type = ?", "approval-duplicate-submit", ApprovalEventTypeDecided).
		Count(&outboxCount).Error; err != nil {
		t.Fatalf("count approval_decided outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected a single approval_decided outbox row, got %d", outboxCount)
	}

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-duplicate-submit")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != "waiting_approval" {
		t.Fatalf("expected duplicate submit to leave run waiting_approval, got %q", run.Status)
	}

	var eventCount int64
	if err := db.Model(&model.AIRunEvent{}).Where("run_id = ?", "run-duplicate-submit").Count(&eventCount).Error; err != nil {
		t.Fatalf("count run events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected no run resume events from duplicate submit, got %d", eventCount)
	}
}

func TestWorkerSkipsExpiredAndRejectedTasks(t *testing.T) {
	testWorkerSkipsExpiredAndRejectedTasks(t)
}

func TestApprovalWorkerSkipsExpiredAndRejectedTasks(t *testing.T) {
	testWorkerSkipsExpiredAndRejectedTasks(t)
}

func TestApprovalWorkerDoesNotConsumeApprovalRequested(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-requested",
		userID:             9,
		runID:              "run-requested",
		userMessageID:      "msg-requested-user",
		assistantMessageID: "msg-requested-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-requested",
		CheckpointID:     "checkpoint-requested",
		SessionID:        "session-requested",
		RunID:            "run-requested",
		UserID:           9,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-requested",
		ArgumentsJSON:    `{"cmd":"whoami"}`,
		PreviewJSON:      `{}`,
		Status:           "rejected",
		ApprovedBy:       9,
		DisapproveReason: "policy denied",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(5 * time.Minute)),
		DecidedAt:        ptrTime(now),
		DecisionSource:   ptrString("policy"),
		PolicyVersion:    ptrString("v1"),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-requested",
		EventType:   "approval_requested",
		RunID:       "run-requested",
		SessionID:   "session-requested",
		PayloadJSON: `{"approval_id":"approval-requested"}`,
		Status:      "pending",
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-requested",
		EventType:   "approval_decided",
		RunID:       "run-requested",
		SessionID:   "session-requested",
		PayloadJSON: `{"approval_id":"approval-requested","approved":true}`,
		Status:      "pending",
	})

	resumeCalls := 0
	worker := NewApprovalWorker(newApprovalWorkerTestLogic(db),
		WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			resumeCalls++
			return nil, nil
		}),
	)

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !claimed {
		t.Fatal("expected approval_decided outbox to be claimed")
	}
	if resumeCalls != 0 {
		t.Fatalf("expected no resume calls for rejected approval_decided, got %d", resumeCalls)
	}

	var requestedOutbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-requested", "approval_requested").First(&requestedOutbox).Error; err != nil {
		t.Fatalf("reload approval_requested outbox: %v", err)
	}
	if requestedOutbox.Status != "pending" {
		t.Fatalf("expected approval_requested outbox to remain pending, got %q", requestedOutbox.Status)
	}
	if requestedOutbox.RetryCount != 0 {
		t.Fatalf("expected approval_requested outbox retry_count to remain 0, got %d", requestedOutbox.RetryCount)
	}
	if requestedOutbox.NextRetryAt != nil {
		t.Fatalf("expected approval_requested outbox next_retry_at to stay nil, got %#v", requestedOutbox.NextRetryAt)
	}

	var decidedOutbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-requested", "approval_decided").First(&decidedOutbox).Error; err != nil {
		t.Fatalf("reload approval_decided outbox: %v", err)
	}
	if decidedOutbox.Status != "done" {
		t.Fatalf("expected approval_decided outbox to be done, got %q", decidedOutbox.Status)
	}
}

func TestApprovalWorkerConsumesCanonicalApprovalDecidedEvent(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-canonical-decided",
		userID:             11,
		runID:              "run-canonical-decided",
		userMessageID:      "msg-canonical-user",
		assistantMessageID: "msg-canonical-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-canonical-decided",
		CheckpointID:     "checkpoint-canonical-decided",
		SessionID:        "session-canonical-decided",
		RunID:            "run-canonical-decided",
		UserID:           11,
		ToolName:         "exec_command",
		ToolCallID:       "tool-call-canonical-decided",
		ArgumentsJSON:    `{"cmd":"whoami"}`,
		PreviewJSON:      `{}`,
		Status:           "rejected",
		ApprovedBy:       11,
		DisapproveReason: "manual reject",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(5 * time.Minute)),
		DecidedAt:        ptrTime(now),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-canonical-decided",
		EventType:   ApprovalEventTypeDecided,
		RunID:       "run-canonical-decided",
		SessionID:   "session-canonical-decided",
		PayloadJSON: `{"approval_id":"approval-canonical-decided","approved":false}`,
		Status:      "pending",
	})

	worker := NewApprovalWorker(newApprovalWorkerTestLogic(db))
	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim canonical approval decided outbox")
	}

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-canonical-decided", ApprovalEventTypeDecided).First(&outbox).Error; err != nil {
		t.Fatalf("reload canonical approval decided outbox: %v", err)
	}
	if outbox.Status != "done" {
		t.Fatalf("expected canonical approval decided outbox done, got %q", outbox.Status)
	}
}

func TestApprovalWorkerDoesNotResumeConvergedApprovalDecisions(t *testing.T) {
	statuses := []string{"completed", "completed_with_tool_errors", "cancelled"}
	for _, runStatus := range statuses {
		t.Run(runStatus, func(t *testing.T) {
			db := newApprovalWorkerTestDB(t)
			now := time.Now().UTC().Truncate(time.Millisecond)
			seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
				sessionID:          "session-" + runStatus,
				userID:             10,
				runID:              "run-" + runStatus,
				userMessageID:      "msg-user-" + runStatus,
				assistantMessageID: "msg-assistant-" + runStatus,
				runStatus:          runStatus,
			})
			seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
				ApprovalID:       "approval-" + runStatus,
				CheckpointID:     "checkpoint-" + runStatus,
				SessionID:        "session-" + runStatus,
				RunID:            "run-" + runStatus,
				UserID:           10,
				ToolName:         "exec_command",
				ToolCallID:       "tool-call-" + runStatus,
				ArgumentsJSON:    `{"cmd":"date"}`,
				PreviewJSON:      `{}`,
				Status:           "approved",
				ApprovedBy:       10,
				Comment:          "already converged",
				TimeoutSeconds:   300,
				ExpiresAt:        ptrTime(now.Add(5 * time.Minute)),
				DecidedAt:        ptrTime(now),
				LockExpiresAt:    ptrTime(now.Add(2 * time.Minute)),
				DecisionSource:   ptrString("user"),
				PolicyVersion:    ptrString("v1"),
				DisapproveReason: "",
			})
			seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
				ApprovalID:  "approval-" + runStatus,
				EventType:   "approval_decided",
				RunID:       "run-" + runStatus,
				SessionID:   "session-" + runStatus,
				PayloadJSON: `{"approval_id":"approval-` + runStatus + `","approved":true}`,
				Status:      "pending",
			})

			resumeCalls := 0
			worker := NewApprovalWorker(newApprovalWorkerTestLogic(db),
				WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
					resumeCalls++
					return nil, nil
				}),
			)

			claimed, err := worker.RunOnce(context.Background())
			if err != nil {
				t.Fatalf("run worker: %v", err)
			}
			if !claimed {
				t.Fatal("expected approval_decided outbox to be claimed")
			}
			if resumeCalls != 0 {
				t.Fatalf("expected no resume calls for converged run, got %d", resumeCalls)
			}

			var outbox model.AIApprovalOutboxEvent
			if err := db.Where("approval_id = ? AND event_type = ?", "approval-"+runStatus, "approval_decided").First(&outbox).Error; err != nil {
				t.Fatalf("reload approval_decided outbox: %v", err)
			}
			if outbox.Status != "done" {
				t.Fatalf("expected converged run outbox to be done, got %q", outbox.Status)
			}
		})
	}
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

func TestApprovalWorkerBackfillsResumeTargetFromToolApprovalEvent(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-resume-backfill",
		userID:             66,
		runID:              "run-resume-backfill",
		userMessageID:      "msg-resume-backfill-user",
		assistantMessageID: "msg-resume-backfill-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-resume-backfill",
		CheckpointID:     "checkpoint-resume-backfill",
		SessionID:        "session-resume-backfill",
		RunID:            "run-resume-backfill",
		UserID:           66,
		ToolName:         "host_exec",
		ToolCallID:       "outer-agent-call-id",
		ArgumentsJSON:    `{"command":"date"}`,
		PreviewJSON:      `{}`,
		Status:           "approved",
		ApprovedBy:       66,
		Comment:          "resume with corrected call id",
		TimeoutSeconds:   300,
		ExpiresAt:        ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:        ptrTime(now),
		LockExpiresAt:    ptrTime(now.Add(2 * time.Minute)),
		DecisionSource:   ptrString("user"),
		PolicyVersion:    ptrString("v2"),
		DisapproveReason: "",
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-resume-backfill",
		EventType:   "approval_decided",
		RunID:       "run-resume-backfill",
		SessionID:   "session-resume-backfill",
		PayloadJSON: `{"approval_id":"approval-resume-backfill","approved":true,"call_id":"stale-call-id"}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-resume-backfill", "session-resume-backfill", &seq, "meta", map[string]any{
		"run_id":     "run-resume-backfill",
		"session_id": "session-resume-backfill",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}
	if err := l.appendRunEvent(context.Background(), "run-resume-backfill", "session-resume-backfill", &seq, "tool_approval", map[string]any{
		"approval_id": "approval-resume-backfill",
		"target_id":   "interrupt-context-id",
		"call_id":     "inner-tool-call-id",
		"tool_name":   "host_exec",
	}); err != nil {
		t.Fatalf("seed tool_approval event: %v", err)
	}

	var capturedParams *adk.ResumeParams
	worker := NewApprovalWorker(l, WithApprovalWorkerResume(func(ctx context.Context, task *model.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
		capturedParams = params
		iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
		go func() {
			gen.Send(adk.EventFromMessage(schema.AssistantMessage("resumed output", nil), nil, schema.Assistant, ""))
			gen.Close()
		}()
		return iter, nil
	}))

	claimed, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run worker: %v", err)
	}
	if !claimed {
		t.Fatal("expected worker to claim approval_decided outbox")
	}
	if capturedParams == nil {
		t.Fatal("expected resume params to be provided")
	}
	if _, ok := capturedParams.Targets["interrupt-context-id"].(*approval.ApprovalResult); !ok {
		t.Fatalf("expected corrected interrupt target id, got %#v", capturedParams.Targets)
	}
	if _, wrong := capturedParams.Targets["outer-agent-call-id"]; wrong {
		t.Fatalf("expected outer stale call id to be ignored, got %#v", capturedParams.Targets)
	}
	if _, wrong := capturedParams.Targets["inner-tool-call-id"]; wrong {
		t.Fatalf("expected tool call id to be ignored when target_id exists, got %#v", capturedParams.Targets)
	}

	reloaded, err := aidao.NewAIApprovalTaskDAO(db).GetByApprovalID(context.Background(), "approval-resume-backfill")
	if err != nil {
		t.Fatalf("reload approval task: %v", err)
	}
	if reloaded == nil {
		t.Fatal("expected approval task to exist")
	}
	if reloaded.ToolCallID != "interrupt-context-id" {
		t.Fatalf("expected tool_call_id backfilled to interrupt target id, got %q", reloaded.ToolCallID)
	}
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

	target, ok := capturedParams.Targets["tool-call-resume"].(*approval.ApprovalResult)
	if !ok {
		t.Fatalf("expected resume target payload, got %#v", capturedParams.Targets)
	}
	if !target.Approved {
		t.Fatalf("expected persisted approval in resume params, got %#v", target)
	}
	if target.Comment != "looks good" {
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

func TestApprovalWorkerResumeInterruptKeepsRunWaitingApproval(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-resume-interrupt",
		userID:             54,
		runID:              "run-resume-interrupt",
		userMessageID:      "msg-resume-interrupt-user",
		assistantMessageID: "msg-resume-interrupt-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:       "approval-resume-interrupt",
		CheckpointID:     "checkpoint-resume-interrupt",
		SessionID:        "session-resume-interrupt",
		RunID:            "run-resume-interrupt",
		UserID:           54,
		ToolName:         "host_exec",
		ToolCallID:       "tool-call-resume-interrupt",
		ArgumentsJSON:    `{"command":"echo test"}`,
		PreviewJSON:      `{}`,
		Status:           "approved",
		ApprovedBy:       54,
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
		ApprovalID:  "approval-resume-interrupt",
		EventType:   "approval_decided",
		RunID:       "run-resume-interrupt",
		SessionID:   "session-resume-interrupt",
		PayloadJSON: `{"approval_id":"approval-resume-interrupt","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-resume-interrupt", "session-resume-interrupt", &seq, "meta", map[string]any{
		"run_id":     "run-resume-interrupt",
		"session_id": "session-resume-interrupt",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(ctx context.Context, task *model.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				gen.Send(&adk.AgentEvent{
					AgentName: "executor",
					Action: &adk.AgentAction{
						Interrupted: &adk.InterruptInfo{
							Data: map[string]any{
								"status":          "suspended",
								"approval_id":     "approval-second",
								"call_id":         "call-second",
								"tool_name":       "host_exec",
								"preview":         map[string]any{"action": "echo test"},
								"timeout_seconds": 300,
							},
						},
					},
				})
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

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-resume-interrupt")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil {
		t.Fatal("expected run to exist")
	}
	if run.Status != "waiting_approval" {
		t.Fatalf("expected waiting_approval after resume interrupt, got %q", run.Status)
	}

	events, err := aidao.NewAIRunEventDAO(db).ListByRun(context.Background(), "run-resume-interrupt")
	if err != nil {
		t.Fatalf("list run events: %v", err)
	}
	assertRunEventPresent(t, events, "tool_approval", "")
	assertRunEventPresent(t, events, "run_state", "waiting_approval")
	assertRunEventAbsent(t, events, "run_state", "completed")
	assertRunEventAbsent(t, events, "done", "completed")

	decided := mustLoadApprovalOutboxByTypes(t, db, "approval-resume-interrupt", approvalDecidedEventTypes()...)
	if decided.Status != "done" {
		t.Fatalf("expected approval_decided outbox done after interrupt persistence, got %q", decided.Status)
	}
	_ = mustLoadApprovalOutbox(t, db, "approval-resume-interrupt", RunEventTypeResuming)
	_ = mustLoadApprovalOutbox(t, db, "approval-resume-interrupt", RunEventTypeResumed)

	var completedCount int64
	if err := db.Model(&model.AIApprovalOutboxEvent{}).
		Where("approval_id = ? AND event_type = ?", "approval-resume-interrupt", RunEventTypeCompleted).
		Count(&completedCount).Error; err != nil {
		t.Fatalf("count completed lifecycle outbox: %v", err)
	}
	if completedCount != 0 {
		t.Fatalf("expected no completed lifecycle outbox for waiting_approval interrupt, got %d", completedCount)
	}
}

func TestApprovalWorker_EmitsCompletedRunStateBeforeDone(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-terminal-complete",
		userID:             61,
		runID:              "run-terminal-complete",
		userMessageID:      "msg-terminal-complete-user",
		assistantMessageID: "msg-terminal-complete-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-terminal-complete",
		CheckpointID:   "checkpoint-terminal-complete",
		SessionID:      "session-terminal-complete",
		RunID:          "run-terminal-complete",
		UserID:         61,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-terminal-complete",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     61,
		Comment:        "approved",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:      ptrTime(now),
		LockExpiresAt:  ptrTime(now.Add(2 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-terminal-complete",
		EventType:   ApprovalEventTypeDecided,
		RunID:       "run-terminal-complete",
		SessionID:   "session-terminal-complete",
		PayloadJSON: `{"approval_id":"approval-terminal-complete","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-terminal-complete", "session-terminal-complete", &seq, "meta", map[string]any{
		"run_id":     "run-terminal-complete",
		"session_id": "session-terminal-complete",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("completed output", nil), nil, schema.Assistant, ""))
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

	events := loadApprovalWorkerRunEvents(t, db, "run-terminal-complete")
	assertRunEventOrder(t, events,
		runEventMatcher{eventType: "run_state", status: "completed"},
		runEventMatcher{eventType: "done", status: "completed"},
	)
	assertRunEventAbsent(t, events, "error", "")

	decided := mustLoadApprovalOutbox(t, db, "approval-terminal-complete", ApprovalEventTypeDecided)
	if decided.Status != "done" {
		t.Fatalf("expected approval_decided outbox done after completion, got %q", decided.Status)
	}
	completed := mustLoadApprovalOutbox(t, db, "approval-terminal-complete", RunEventTypeCompleted)
	payload := decodeApprovalWorkerPayload(t, completed.PayloadJSON)
	if status, _ := payload["status"].(string); status != "completed" {
		t.Fatalf("expected completed lifecycle payload status completed, got %#v", payload)
	}
}

func TestApprovalWorker_EmitsFailedRunStateBeforeError(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-terminal-failed",
		userID:             62,
		runID:              "run-terminal-failed",
		userMessageID:      "msg-terminal-failed-user",
		assistantMessageID: "msg-terminal-failed-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-terminal-failed",
		CheckpointID:   "checkpoint-terminal-failed",
		SessionID:      "session-terminal-failed",
		RunID:          "run-terminal-failed",
		UserID:         62,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-terminal-failed",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     62,
		Comment:        "approved",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:      ptrTime(now),
		LockExpiresAt:  ptrTime(now.Add(2 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-terminal-failed",
		EventType:   ApprovalEventTypeDecided,
		RunID:       "run-terminal-failed",
		SessionID:   "session-terminal-failed",
		PayloadJSON: `{"approval_id":"approval-terminal-failed","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-terminal-failed", "session-terminal-failed", &seq, "meta", map[string]any{
		"run_id":     "run-terminal-failed",
		"session_id": "session-terminal-failed",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				gen.Send(&adk.AgentEvent{Err: errors.New("fatal resume event")})
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

	events := loadApprovalWorkerRunEvents(t, db, "run-terminal-failed")
	assertRunEventOrder(t, events,
		runEventMatcher{eventType: "run_state", status: "failed"},
		runEventMatcher{eventType: "error"},
	)
	assertRunEventAbsent(t, events, "done", "")

	run, err := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-terminal-failed")
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if run == nil || run.Status == "resume_failed_retryable" {
		t.Fatalf("expected fatal failure to converge without retryable status, got %#v", run)
	}

	decided := mustLoadApprovalOutbox(t, db, "approval-terminal-failed", ApprovalEventTypeDecided)
	if decided.Status != "done" {
		t.Fatalf("expected approval_decided outbox done after fatal failure, got %q", decided.Status)
	}
	resumeFailed := mustLoadApprovalOutbox(t, db, "approval-terminal-failed", RunEventTypeResumeFailed)
	payload := decodeApprovalWorkerPayload(t, resumeFailed.PayloadJSON)
	if retryable, _ := payload["retryable"].(bool); retryable {
		t.Fatalf("expected fatal resume_failed payload to be non-retryable, got %#v", payload)
	}
}

func TestApprovalWorker_LeavesRetryableResumeFailureWithoutDone(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-retryable-resume-failure",
		userID:             63,
		runID:              "run-retryable-resume-failure",
		userMessageID:      "msg-retryable-resume-failure-user",
		assistantMessageID: "msg-retryable-resume-failure-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-retryable-resume-failure",
		CheckpointID:   "checkpoint-retryable-resume-failure",
		SessionID:      "session-retryable-resume-failure",
		RunID:          "run-retryable-resume-failure",
		UserID:         63,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-retryable-resume-failure",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     63,
		Comment:        "approved",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:      ptrTime(now),
		LockExpiresAt:  ptrTime(now.Add(2 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-retryable-resume-failure",
		EventType:   ApprovalEventTypeDecided,
		RunID:       "run-retryable-resume-failure",
		SessionID:   "session-retryable-resume-failure",
		PayloadJSON: `{"approval_id":"approval-retryable-resume-failure","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-retryable-resume-failure", "session-retryable-resume-failure", &seq, "meta", map[string]any{
		"run_id":     "run-retryable-resume-failure",
		"session_id": "session-retryable-resume-failure",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			return nil, errors.New("checkpoint store unavailable")
		}),
	)

	claimed, err := worker.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected retryable resume failure to return an error")
	}
	if !claimed {
		t.Fatal("expected worker to claim approval_decided outbox")
	}

	events := loadApprovalWorkerRunEvents(t, db, "run-retryable-resume-failure")
	assertRunEventPresent(t, events, "run_state", "resume_failed_retryable")
	assertRunEventAbsent(t, events, "done", "")
	assertRunEventAbsent(t, events, "error", "")

	var outbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-retryable-resume-failure", ApprovalEventTypeDecided).First(&outbox).Error; err != nil {
		t.Fatalf("reload approval_decided outbox: %v", err)
	}
	if outbox.Status != "pending" {
		t.Fatalf("expected retryable failure to keep outbox pending, got %q", outbox.Status)
	}
	if outbox.RetryCount != 1 {
		t.Fatalf("expected retry_count=1 after retryable failure, got %d", outbox.RetryCount)
	}

	resumeFailed := mustLoadApprovalOutbox(t, db, "approval-retryable-resume-failure", RunEventTypeResumeFailed)
	payload := decodeApprovalWorkerPayload(t, resumeFailed.PayloadJSON)
	if retryable, _ := payload["retryable"].(bool); !retryable {
		t.Fatalf("expected retryable resume_failed payload, got %#v", payload)
	}
}

func TestApprovalWorker_RejectAndExpireEndAsCancelledWithoutDoneOrError(t *testing.T) {
	testCases := []struct {
		name          string
		approvalID    string
		sessionID     string
		runID         string
		taskStatus    string
		expiresOffset time.Duration
	}{
		{
			name:          "rejected",
			approvalID:    "approval-cancelled-rejected",
			sessionID:     "session-cancelled-rejected",
			runID:         "run-cancelled-rejected",
			taskStatus:    "rejected",
			expiresOffset: 10 * time.Minute,
		},
		{
			name:          "expired",
			approvalID:    "approval-cancelled-expired",
			sessionID:     "session-cancelled-expired",
			runID:         "run-cancelled-expired",
			taskStatus:    "expired",
			expiresOffset: -1 * time.Minute,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := newApprovalWorkerTestDB(t)
			now := time.Now().UTC().Truncate(time.Millisecond)
			seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
				sessionID:          tc.sessionID,
				userID:             64,
				runID:              tc.runID,
				userMessageID:      tc.runID + "-user",
				assistantMessageID: tc.runID + "-assistant",
				runStatus:          "waiting_approval",
			})
			seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
				ApprovalID:       tc.approvalID,
				CheckpointID:     "checkpoint-" + tc.name,
				SessionID:        tc.sessionID,
				RunID:            tc.runID,
				UserID:           64,
				ToolName:         "exec_command",
				ToolCallID:       "tool-call-" + tc.name,
				ArgumentsJSON:    `{"cmd":"date"}`,
				PreviewJSON:      `{}`,
				Status:           tc.taskStatus,
				ApprovedBy:       64,
				Comment:          tc.name,
				TimeoutSeconds:   300,
				ExpiresAt:        ptrTime(now.Add(tc.expiresOffset)),
				DecidedAt:        ptrTime(now),
				LockExpiresAt:    ptrTime(now.Add(2 * time.Minute)),
				DecisionSource:   ptrString("user"),
				PolicyVersion:    ptrString("v1"),
				DisapproveReason: "blocked",
			})
			seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
				ApprovalID:  tc.approvalID,
				EventType:   ApprovalEventTypeDecided,
				RunID:       tc.runID,
				SessionID:   tc.sessionID,
				PayloadJSON: fmt.Sprintf(`{"approval_id":%q,"approved":false}`, tc.approvalID),
				Status:      "pending",
			})

			l := newApprovalWorkerTestLogic(db)
			seq := 0
			if err := l.appendRunEvent(context.Background(), tc.runID, tc.sessionID, &seq, "meta", map[string]any{
				"run_id":     tc.runID,
				"session_id": tc.sessionID,
				"turn":       1,
			}); err != nil {
				t.Fatalf("seed meta event: %v", err)
			}

			worker := NewApprovalWorker(l)
			claimed, err := worker.RunOnce(context.Background())
			if err != nil {
				t.Fatalf("run worker: %v", err)
			}
			if !claimed {
				t.Fatal("expected worker to claim approval_decided outbox")
			}

			events := loadApprovalWorkerRunEvents(t, db, tc.runID)
			assertRunEventPresent(t, events, "run_state", "cancelled")
			assertRunEventAbsent(t, events, "done", "")
			assertRunEventAbsent(t, events, "error", "")

			decided := mustLoadApprovalOutbox(t, db, tc.approvalID, ApprovalEventTypeDecided)
			if decided.Status != "done" {
				t.Fatalf("expected approval_decided outbox done for %s, got %q", tc.name, decided.Status)
			}
			completed := mustLoadApprovalOutbox(t, db, tc.approvalID, RunEventTypeCompleted)
			payload := decodeApprovalWorkerPayload(t, completed.PayloadJSON)
			if status, _ := payload["status"].(string); status != "cancelled" {
				t.Fatalf("expected completed lifecycle payload status cancelled for %s, got %#v", tc.name, payload)
			}
		})
	}
}

func TestSubmitApproval_PersistsAuditRecordBeforeResumingEvent(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-audit-before-resuming",
		userID:             65,
		runID:              "run-audit-before-resuming",
		userMessageID:      "msg-audit-before-resuming-user",
		assistantMessageID: "msg-audit-before-resuming-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-audit-before-resuming",
		CheckpointID:   "checkpoint-audit-before-resuming",
		SessionID:      "session-audit-before-resuming",
		RunID:          "run-audit-before-resuming",
		UserID:         65,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-audit-before-resuming",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(10 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})

	l := newApprovalWorkerTestLogic(db)
	result, err := l.SubmitApproval(context.Background(), SubmitApprovalInput{
		ApprovalID: "approval-audit-before-resuming",
		Approved:   true,
		Comment:    "ship it",
		UserID:     65,
	})
	if err != nil {
		t.Fatalf("submit approval: %v", err)
	}
	if result == nil || result.Status != "approved" {
		t.Fatalf("expected approved result, got %#v", result)
	}

	resumeObservedAudit := false
	worker := NewApprovalWorker(l,
		WithApprovalWorkerResume(func(ctx context.Context, task *model.AIApprovalTask, params *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
			var decided model.AIApprovalOutboxEvent
			if err := db.Where("approval_id = ? AND event_type = ?", task.ApprovalID, ApprovalEventTypeDecided).First(&decided).Error; err != nil {
				t.Fatalf("load approval_decided outbox: %v", err)
			}
			var resuming model.AIApprovalOutboxEvent
			if err := db.Where("approval_id = ? AND event_type = ?", task.ApprovalID, RunEventTypeResuming).First(&resuming).Error; err != nil {
				t.Fatalf("load ai.run.resuming outbox: %v", err)
			}

			payload := decodeApprovalWorkerPayload(t, decided.PayloadJSON)
			if payload["approved_by"] != float64(65) {
				t.Fatalf("expected approved_by=65 in audit payload, got %#v", payload)
			}
			if payload["comment"] != "ship it" {
				t.Fatalf("expected comment in audit payload, got %#v", payload)
			}
			if decided.Sequence >= resuming.Sequence {
				t.Fatalf("expected approval_decided sequence before resuming, decided=%d resuming=%d", decided.Sequence, resuming.Sequence)
			}

			target, ok := params.Targets[task.ToolCallID].(*approval.ApprovalResult)
			if !ok {
				t.Fatalf("expected resume params target for %q, got %#v", task.ToolCallID, params.Targets)
			}
			if target.Comment != "ship it" {
				t.Fatalf("expected resume params to include persisted comment, got %#v", target)
			}
			resumeObservedAudit = true

			iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
			go func() {
				gen.Send(adk.EventFromMessage(schema.AssistantMessage("done", nil), nil, schema.Assistant, ""))
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
	if !resumeObservedAudit {
		t.Fatal("expected resume path to verify audit ordering")
	}
}

func TestApprovalWorker_RetriesMissingCompletedLifecycleEventAfterConvergence(t *testing.T) {
	db := newApprovalWorkerTestDB(t)
	now := time.Now().UTC().Truncate(time.Millisecond)
	seedApprovalWorkerRun(t, db, approvalWorkerRunSeed{
		sessionID:          "session-completed-retry",
		userID:             67,
		runID:              "run-completed-retry",
		userMessageID:      "msg-completed-retry-user",
		assistantMessageID: "msg-completed-retry-assistant",
		runStatus:          "waiting_approval",
	})
	seedApprovalWorkerTask(t, db, &model.AIApprovalTask{
		ApprovalID:     "approval-completed-retry",
		CheckpointID:   "checkpoint-completed-retry",
		SessionID:      "session-completed-retry",
		RunID:          "run-completed-retry",
		UserID:         67,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-completed-retry",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		ApprovedBy:     67,
		Comment:        "approved",
		TimeoutSeconds: 300,
		ExpiresAt:      ptrTime(now.Add(10 * time.Minute)),
		DecidedAt:      ptrTime(now),
		LockExpiresAt:  ptrTime(now.Add(2 * time.Minute)),
		DecisionSource: ptrString("user"),
		PolicyVersion:  ptrString("v1"),
	})
	seedApprovalWorkerOutbox(t, db, &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-completed-retry",
		EventType:   ApprovalEventTypeDecided,
		RunID:       "run-completed-retry",
		SessionID:   "session-completed-retry",
		PayloadJSON: `{"approval_id":"approval-completed-retry","approved":true}`,
		Status:      "pending",
	})

	l := newApprovalWorkerTestLogic(db)
	seq := 0
	if err := l.appendRunEvent(context.Background(), "run-completed-retry", "session-completed-retry", &seq, "meta", map[string]any{
		"run_id":     "run-completed-retry",
		"session_id": "session-completed-retry",
		"turn":       1,
	}); err != nil {
		t.Fatalf("seed meta event: %v", err)
	}

	failCompletedOutbox := true
	callbackName := "test:fail_completed_lifecycle_outbox_once"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if !failCompletedOutbox || tx.Statement == nil || tx.Statement.Schema == nil || tx.Statement.Schema.Table != "ai_approval_outbox_events" {
			return
		}
		event, ok := tx.Statement.Dest.(*model.AIApprovalOutboxEvent)
		if !ok || event.EventType != RunEventTypeCompleted {
			return
		}
		failCompletedOutbox = false
		tx.AddError(errors.New("simulated completed lifecycle outbox failure"))
	}); err != nil {
		t.Fatalf("register completed outbox callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Create().Remove(callbackName)
	})

	worker := NewApprovalWorker(l, WithApprovalWorkerResume(func(context.Context, *model.AIApprovalTask, *adk.ResumeParams) (*adk.AsyncIterator[*adk.AgentEvent], error) {
		iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
		go func() {
			gen.Send(adk.EventFromMessage(schema.AssistantMessage("completed output", nil), nil, schema.Assistant, ""))
			gen.Close()
		}()
		return iter, nil
	}))

	claimed, err := worker.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected first run to fail when completed lifecycle outbox insert fails")
	}
	if !claimed {
		t.Fatal("expected first run to claim approval_decided outbox")
	}

	run, loadErr := aidao.NewAIRunDAO(db).GetRun(context.Background(), "run-completed-retry")
	if loadErr != nil {
		t.Fatalf("reload converged run: %v", loadErr)
	}
	if run == nil || run.Status != "completed" {
		t.Fatalf("expected run to remain converged as completed, got %#v", run)
	}

	var decidedOutbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-completed-retry", ApprovalEventTypeDecided).First(&decidedOutbox).Error; err != nil {
		t.Fatalf("reload approval_decided outbox: %v", err)
	}
	if err := db.Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ?", decidedOutbox.ID).
		Updates(map[string]any{
			"status":        "pending",
			"next_retry_at": ptrTime(now.Add(-1 * time.Minute)),
		}).Error; err != nil {
		t.Fatalf("reset approval_decided retry window: %v", err)
	}

	claimed, err = worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("retry converged worker: %v", err)
	}
	if !claimed {
		t.Fatal("expected retry run to claim approval_decided outbox")
	}

	var completedOutbox model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-completed-retry", RunEventTypeCompleted).First(&completedOutbox).Error; err != nil {
		t.Fatalf("expected completed lifecycle outbox on retry: %v", err)
	}
	if err := db.Where("id = ?", decidedOutbox.ID).First(&decidedOutbox).Error; err != nil {
		t.Fatalf("reload approval_decided after retry: %v", err)
	}
	if decidedOutbox.Status != "done" {
		t.Fatalf("expected approval_decided outbox to be done after converged retry, got %q", decidedOutbox.Status)
	}
}

type runEventMatcher struct {
	eventType string
	status    string
}

func loadApprovalWorkerRunEvents(t *testing.T, db *gorm.DB, runID string) []model.AIRunEvent {
	t.Helper()
	events, err := aidao.NewAIRunEventDAO(db).ListByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run events for %s: %v", runID, err)
	}
	return events
}

func assertRunEventOrder(t *testing.T, events []model.AIRunEvent, first, second runEventMatcher) {
	t.Helper()
	firstIndex := findRunEventIndex(t, events, first)
	secondIndex := findRunEventIndex(t, events, second)
	if firstIndex >= secondIndex {
		t.Fatalf("expected %s/%s before %s/%s, got events %#v", first.eventType, first.status, second.eventType, second.status, summarizeRunEvents(t, events))
	}
}

func assertRunEventPresent(t *testing.T, events []model.AIRunEvent, eventType, status string) {
	t.Helper()
	_ = findRunEventIndex(t, events, runEventMatcher{eventType: eventType, status: status})
}

func assertRunEventAbsent(t *testing.T, events []model.AIRunEvent, eventType, status string) {
	t.Helper()
	for _, event := range events {
		if event.EventType != eventType {
			continue
		}
		if status == "" || decodeApprovalWorkerPayload(t, event.PayloadJSON)["status"] == status {
			t.Fatalf("expected %s/%s to be absent, got events %#v", eventType, status, summarizeRunEvents(t, events))
		}
	}
}

func findRunEventIndex(t *testing.T, events []model.AIRunEvent, matcher runEventMatcher) int {
	t.Helper()
	for index, event := range events {
		if event.EventType != matcher.eventType {
			continue
		}
		if matcher.status == "" || decodeApprovalWorkerPayload(t, event.PayloadJSON)["status"] == matcher.status {
			return index
		}
	}
	t.Fatalf("expected event %s/%s, got %#v", matcher.eventType, matcher.status, summarizeRunEvents(t, events))
	return -1
}

func summarizeRunEvents(t *testing.T, events []model.AIRunEvent) []string {
	t.Helper()
	summary := make([]string, 0, len(events))
	for _, event := range events {
		status, _ := decodeApprovalWorkerPayload(t, event.PayloadJSON)["status"].(string)
		if status == "" {
			summary = append(summary, event.EventType)
			continue
		}
		summary = append(summary, event.EventType+":"+status)
	}
	return summary
}

func decodeApprovalWorkerPayload(t *testing.T, raw string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("decode payload %q: %v", raw, err)
	}
	return payload
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
