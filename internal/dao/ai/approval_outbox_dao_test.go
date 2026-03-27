package ai

import (
	"context"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestApprovalOutboxUniqueKey(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	event := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-1",
		EventType:   "approval_decided",
		RunID:       "run-1",
		SessionID:   "session-1",
		PayloadJSON: `{"state":"first"}`,
		Status:      "pending",
		RetryCount:  0,
		NextRetryAt: nil,
	}
	if err := dao.EnqueueOrTouch(ctx, event); err != nil {
		t.Fatalf("enqueue first event: %v", err)
	}

	dup := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-1",
		EventType:   "approval_decided",
		RunID:       "run-2",
		SessionID:   "session-2",
		PayloadJSON: `{"state":"second"}`,
		Status:      "pending",
		RetryCount:  99,
		NextRetryAt: nil,
	}
	if err := dao.EnqueueOrTouch(ctx, dup); err != nil {
		t.Fatalf("enqueue duplicate event: %v", err)
	}

	var count int64
	if err := db.Model(&model.AIApprovalOutboxEvent{}).
		Where("approval_id = ? AND event_type = ?", "approval-1", "approval_decided").
		Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected a single row for approval_id+event_type, got %d", count)
	}

	var stored model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", "approval-1", "approval_decided").First(&stored).Error; err != nil {
		t.Fatalf("load row: %v", err)
	}
	if stored.RunID != "run-2" || stored.SessionID != "session-2" || stored.PayloadJSON != `{"state":"second"}` {
		t.Fatalf("expected duplicate enqueue to touch existing row, got %#v", stored)
	}
	if stored.RetryCount != 0 {
		t.Fatalf("expected retry_count to remain observational, got %d", stored.RetryCount)
	}

	if !db.Migrator().HasIndex(&model.AIApprovalOutboxEvent{}, "uk_ai_approval_outbox_events_approval_event") {
		t.Fatal("expected unique index uk_ai_approval_outbox_events_approval_event")
	}
	if !db.Migrator().HasIndex(&model.AIApprovalOutboxEvent{}, "idx_ai_approval_outbox_events_queue") {
		t.Fatal("expected queue index idx_ai_approval_outbox_events_queue")
	}
}

func TestApprovalOutboxEnqueueDuplicateAfterDoneKeepsDoneStatus(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	retryAt := time.Now().Add(30 * time.Minute).UTC().Truncate(time.Millisecond)
	event := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-done",
		EventType:   "approval_decided",
		RunID:       "run-initial",
		SessionID:   "session-initial",
		PayloadJSON: `{"state":"initial"}`,
		Status:      "pending",
	}
	if err := dao.EnqueueOrTouch(ctx, event); err != nil {
		t.Fatalf("enqueue initial event: %v", err)
	}

	if err := db.Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ?", event.ID).
		Updates(map[string]any{
			"status":        "done",
			"next_retry_at": &retryAt,
		}).Error; err != nil {
		t.Fatalf("seed done status: %v", err)
	}

	dup := &model.AIApprovalOutboxEvent{
		ApprovalID:  event.ApprovalID,
		EventType:   event.EventType,
		RunID:       "run-updated",
		SessionID:   "session-updated",
		PayloadJSON: `{"state":"updated"}`,
		Status:      "pending",
	}
	if err := dao.EnqueueOrTouch(ctx, dup); err != nil {
		t.Fatalf("enqueue duplicate event: %v", err)
	}

	var stored model.AIApprovalOutboxEvent
	if err := db.First(&stored, event.ID).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if stored.Status != "done" {
		t.Fatalf("expected duplicate enqueue to keep done status, got %q", stored.Status)
	}
	if stored.NextRetryAt == nil || !stored.NextRetryAt.Equal(retryAt) {
		t.Fatalf("expected duplicate enqueue to preserve next_retry_at %v, got %#v", retryAt, stored.NextRetryAt)
	}
	if stored.RunID != "run-initial" || stored.SessionID != "session-initial" || stored.PayloadJSON != `{"state":"initial"}` {
		t.Fatalf("expected duplicate enqueue on done row to be no-op, got %#v", stored)
	}
}

func TestApprovalOutboxClaimPendingDoesNotReturnStaleCandidate(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	staleCandidate := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-stale",
		EventType:   "approval_decided",
		RunID:       "run-stale",
		SessionID:   "session-stale",
		PayloadJSON: `{"state":"pending"}`,
		Status:      "pending",
	}
	if err := dao.EnqueueOrTouch(ctx, staleCandidate); err != nil {
		t.Fatalf("enqueue stale candidate: %v", err)
	}
	claimable := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-claimable",
		EventType:   "approval_decided",
		RunID:       "run-claimable",
		SessionID:   "session-claimable",
		PayloadJSON: `{"state":"pending"}`,
		Status:      "pending",
	}
	if err := dao.EnqueueOrTouch(ctx, claimable); err != nil {
		t.Fatalf("enqueue claimable event: %v", err)
	}

	type ctxKey string
	const forceZeroRowsKey ctxKey = "force-zero-rows"
	callbackName := "test:approval_outbox_force_zero_rows"
	forcedCount := 0
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil {
			return
		}
		if tx.Statement.Schema.Table != "ai_approval_outbox_events" {
			return
		}
		if tx.Statement.Context == nil || tx.Statement.Context.Value(forceZeroRowsKey) == nil {
			return
		}
		if forcedCount > 0 {
			return
		}
		updates, ok := tx.Statement.Dest.(map[string]any)
		if !ok || updates["status"] != "processing" {
			return
		}
		forcedCount++
		tx.Statement.AddClause(clause.Where{Exprs: []clause.Expression{
			clause.Expr{SQL: "1 = 0"},
		}})
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Callback().Update().Remove(callbackName)
	})

	claimed, err := dao.ClaimPending(context.WithValue(ctx, forceZeroRowsKey, true))
	if err != nil {
		t.Fatalf("claim pending: %v", err)
	}
	if claimed == nil {
		t.Fatalf("expected claim to make forward progress after stale update miss")
	}

	if forcedCount != 1 {
		t.Fatalf("expected one forced zero-row update, got %d", forcedCount)
	}
}

func TestApprovalOutboxClaimDoneRetryLifecycle(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	now := time.Now()
	first := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-2",
		EventType:   "approval_decided",
		RunID:       "run-2",
		SessionID:   "session-2",
		PayloadJSON: `{"state":"pending"}`,
		Status:      "pending",
		NextRetryAt: nil,
	}
	if err := dao.EnqueueOrTouch(ctx, first); err != nil {
		t.Fatalf("enqueue first event: %v", err)
	}

	claimed, err := dao.ClaimPending(ctx)
	if err != nil {
		t.Fatalf("claim pending: %v", err)
	}
	if claimed == nil || claimed.ApprovalID != first.ApprovalID || claimed.EventType != first.EventType {
		t.Fatalf("expected to claim first pending event, got %#v", claimed)
	}

	var claimedStored model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", first.ApprovalID, first.EventType).First(&claimedStored).Error; err != nil {
		t.Fatalf("reload claimed event: %v", err)
	}
	if claimedStored.Status != "processing" {
		t.Fatalf("expected claimed row to move to processing, got %q", claimedStored.Status)
	}

	if err := dao.MarkDone(ctx, claimed.ID); err != nil {
		t.Fatalf("mark done: %v", err)
	}
	var done model.AIApprovalOutboxEvent
	if err := db.First(&done, claimed.ID).Error; err != nil {
		t.Fatalf("reload done event: %v", err)
	}
	if done.Status != "done" {
		t.Fatalf("expected done status, got %q", done.Status)
	}
	if err := dao.MarkRetry(ctx, done.ID, now.Add(20*time.Minute)); err != nil {
		t.Fatalf("mark retry on done row: %v", err)
	}
	var stillDone model.AIApprovalOutboxEvent
	if err := db.First(&stillDone, done.ID).Error; err != nil {
		t.Fatalf("reload done row after retry attempt: %v", err)
	}
	if stillDone.Status != "done" {
		t.Fatalf("expected done row to stay done after MarkRetry CAS miss, got %q", stillDone.Status)
	}

	second := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-3",
		EventType:   "resume_failed",
		RunID:       "run-3",
		SessionID:   "session-3",
		PayloadJSON: `{"state":"retry"}`,
		Status:      "pending",
		NextRetryAt: &now,
	}
	if err := dao.EnqueueOrTouch(ctx, second); err != nil {
		t.Fatalf("enqueue second event: %v", err)
	}
	secondClaimed, err := dao.ClaimPending(ctx)
	if err != nil {
		t.Fatalf("claim second event: %v", err)
	}
	if secondClaimed == nil || secondClaimed.ID != second.ID {
		t.Fatalf("expected second event to be claimed, got %#v", secondClaimed)
	}
	if err := dao.MarkRetry(ctx, second.ID, now.Add(10*time.Minute)); err != nil {
		t.Fatalf("mark retry: %v", err)
	}
	var retried model.AIApprovalOutboxEvent
	if err := db.First(&retried, second.ID).Error; err != nil {
		t.Fatalf("reload retried event: %v", err)
	}
	if retried.Status != "pending" {
		t.Fatalf("expected retry to keep row pending, got %q", retried.Status)
	}
	if retried.RetryCount != 1 {
		t.Fatalf("expected retry_count to increment, got %d", retried.RetryCount)
	}
	if retried.NextRetryAt == nil || retried.NextRetryAt.Before(now.Add(9*time.Minute)) {
		t.Fatalf("expected next_retry_at to move forward, got %#v", retried.NextRetryAt)
	}

	third := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-4",
		EventType:   "approval_decided",
		RunID:       "run-4",
		SessionID:   "session-4",
		PayloadJSON: `{"state":"pending"}`,
		Status:      "pending",
	}
	if err := dao.EnqueueOrTouch(ctx, third); err != nil {
		t.Fatalf("enqueue third event: %v", err)
	}
	if err := dao.MarkDone(ctx, third.ID); err != nil {
		t.Fatalf("mark done on pending row: %v", err)
	}
	var stillPending model.AIApprovalOutboxEvent
	if err := db.First(&stillPending, third.ID).Error; err != nil {
		t.Fatalf("reload third row: %v", err)
	}
	if stillPending.Status != "pending" {
		t.Fatalf("expected pending row to stay pending after MarkDone CAS miss, got %q", stillPending.Status)
	}
}

func TestApprovalOutboxReclaimsStaleProcessingRow(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	stale := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-stale-processing",
		EventType:   "approval_decided",
		RunID:       "run-stale",
		SessionID:   "session-stale",
		PayloadJSON: `{"state":"processing"}`,
		Status:      "pending",
	}
	if err := dao.EnqueueOrTouch(ctx, stale); err != nil {
		t.Fatalf("enqueue stale event: %v", err)
	}
	staleAt := time.Now().Add(-(approvalOutboxProcessingLease + 15*time.Second))
	if err := db.Model(&model.AIApprovalOutboxEvent{}).
		Where("id = ?", stale.ID).
		Updates(map[string]any{
			"status":     "processing",
			"updated_at": staleAt,
		}).Error; err != nil {
		t.Fatalf("seed stale processing row: %v", err)
	}

	claimed, err := dao.ClaimPending(ctx)
	if err != nil {
		t.Fatalf("claim stale processing row: %v", err)
	}
	if claimed == nil || claimed.ID != stale.ID {
		t.Fatalf("expected stale processing row to be reclaimed, got %#v", claimed)
	}

	var stored model.AIApprovalOutboxEvent
	if err := db.First(&stored, stale.ID).Error; err != nil {
		t.Fatalf("reload reclaimed row: %v", err)
	}
	if stored.Status != "processing" {
		t.Fatalf("expected reclaimed row to remain processing, got %q", stored.Status)
	}
	if !stored.UpdatedAt.After(staleAt) {
		t.Fatalf("expected reclaimed row updated_at to advance beyond %v, got %v", staleAt, stored.UpdatedAt)
	}
}

func newApprovalOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIApprovalOutboxEvent{}); err != nil {
		t.Fatalf("migrate outbox table: %v", err)
	}
	return db
}
