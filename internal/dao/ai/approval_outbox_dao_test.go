package ai

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApprovalOutboxUniqueKeyDoesNotResurrectDone(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	expiresAt := time.Now().Add(10 * time.Minute)
	seed := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-1",
		EventType:   "approval_decided",
		RunID:       "run-1",
		SessionID:   "session-1",
		PayloadJSON: `{"decision":"done"}`,
		Status:      "done",
		RetryCount:  2,
		NextRetryAt: &expiresAt,
	}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed done event: %v", err)
	}

	requestedRetry := time.Now().Add(5 * time.Minute)
	if err := dao.EnqueueOrTouch(ctx, &model.AIApprovalOutboxEvent{
		ApprovalID:  seed.ApprovalID,
		EventType:   seed.EventType,
		RunID:       "run-2",
		SessionID:   "session-2",
		PayloadJSON: `{"decision":"pending"}`,
		Status:      "pending",
		NextRetryAt: &requestedRetry,
	}); err != nil {
		t.Fatalf("duplicate enqueue: %v", err)
	}

	var got model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", seed.ApprovalID, seed.EventType).First(&got).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if got.Status != "done" {
		t.Fatalf("expected done status to remain unchanged, got %q", got.Status)
	}
	if got.RetryCount != seed.RetryCount {
		t.Fatalf("expected retry_count to remain %d, got %d", seed.RetryCount, got.RetryCount)
	}
	if got.RunID != "run-2" || got.SessionID != "session-2" || got.PayloadJSON != `{"decision":"pending"}` {
		t.Fatalf("expected touch fields to update, got %#v", got)
	}
}

func TestApprovalOutboxUniqueKeyKeepsProcessingState(t *testing.T) {
	db := newApprovalOutboxTestDB(t)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	seed := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-2",
		EventType:   "approval_requested",
		RunID:       "run-1",
		SessionID:   "session-1",
		PayloadJSON: `{"step":"first"}`,
		Status:      "processing",
		RetryCount:  1,
	}
	if err := db.Create(seed).Error; err != nil {
		t.Fatalf("seed processing event: %v", err)
	}

	if err := dao.EnqueueOrTouch(ctx, &model.AIApprovalOutboxEvent{
		ApprovalID:  seed.ApprovalID,
		EventType:   seed.EventType,
		RunID:       "run-2",
		SessionID:   "session-2",
		PayloadJSON: `{"step":"second"}`,
		Status:      "pending",
	}); err != nil {
		t.Fatalf("duplicate enqueue: %v", err)
	}

	var got model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", seed.ApprovalID, seed.EventType).First(&got).Error; err != nil {
		t.Fatalf("reload event: %v", err)
	}
	if got.Status != "processing" {
		t.Fatalf("expected processing status to remain unchanged, got %q", got.Status)
	}
	if got.RetryCount != seed.RetryCount {
		t.Fatalf("expected retry_count to remain %d, got %d", seed.RetryCount, got.RetryCount)
	}
	if got.RunID != "run-2" || got.SessionID != "session-2" || got.PayloadJSON != `{"step":"second"}` {
		t.Fatalf("expected touch fields to update, got %#v", got)
	}
}

func TestApprovalOutboxClaimPending_IsSingleWinnerUnderContention(t *testing.T) {
	db := newConcurrentApprovalOutboxTestDB(t)
	enableSQLiteBusyTimeoutApprovalOutbox(t, db)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	event := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-claim",
		EventType:   "approval_requested",
		RunID:       "run-claim",
		SessionID:   "session-claim",
		PayloadJSON: `{"tool":"dangerous"}`,
		Status:      "pending",
	}
	if err := db.Create(event).Error; err != nil {
		t.Fatalf("seed pending event: %v", err)
	}

	const workers = 2
	start := make(chan struct{})
	results := make(chan *model.AIApprovalOutboxEvent, workers)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := dao.ClaimPending(ctx)
			if err != nil {
				t.Errorf("claim pending: %v", err)
				return
			}
			results <- got
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var claimed []*model.AIApprovalOutboxEvent
	for result := range results {
		if result != nil {
			claimed = append(claimed, result)
		}
	}

	if len(claimed) != 1 {
		t.Fatalf("expected exactly one worker to claim the event, got %d claims: %#v", len(claimed), claimed)
	}
	if claimed[0].ApprovalID != event.ApprovalID || claimed[0].EventType != event.EventType {
		t.Fatalf("unexpected claimed event: %#v", claimed[0])
	}

	var refreshed model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", event.ApprovalID, event.EventType).First(&refreshed).Error; err != nil {
		t.Fatalf("reload claimed event: %v", err)
	}
	if refreshed.Status != "processing" {
		t.Fatalf("expected event to be processing after claim, got %q", refreshed.Status)
	}
}

func TestApprovalOutboxClaimPending_SkipsStaleCandidate(t *testing.T) {
	db := newConcurrentApprovalOutboxTestDB(t)
	enableSQLiteBusyTimeoutApprovalOutbox(t, db)
	dao := NewAIApprovalOutboxDAO(db)
	ctx := context.Background()

	first := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-stale-1",
		EventType:   "approval_requested",
		RunID:       "run-stale-1",
		SessionID:   "session-stale-1",
		PayloadJSON: `{"tool":"first"}`,
		Status:      "pending",
	}
	second := &model.AIApprovalOutboxEvent{
		ApprovalID:  "approval-stale-2",
		EventType:   "approval_requested",
		RunID:       "run-stale-2",
		SessionID:   "session-stale-2",
		PayloadJSON: `{"tool":"second"}`,
		Status:      "pending",
	}
	if err := db.Create(first).Error; err != nil {
		t.Fatalf("seed first event: %v", err)
	}
	if err := db.Create(second).Error; err != nil {
		t.Fatalf("seed second event: %v", err)
	}

	originalUpdater := approvalOutboxClaimUpdater
	defer func() { approvalOutboxClaimUpdater = originalUpdater }()
	var forced atomic.Bool
	approvalOutboxClaimUpdater = func(tx *gorm.DB, id uint64, updatedAt time.Time) (int64, error) {
		if !forced.Load() {
			forced.Store(true)
			if err := tx.Model(&model.AIApprovalOutboxEvent{}).
				Where("id = ?", id).
				Updates(map[string]any{
					"status":        "processing",
					"next_retry_at": nil,
					"updated_at":    updatedAt,
				}).Error; err != nil {
				return 0, err
			}
			return 0, nil
		}
		return originalUpdater(tx, id, updatedAt)
	}

	got, err := dao.ClaimPending(ctx)
	if err != nil {
		t.Fatalf("claim pending: %v", err)
	}
	if got == nil {
		t.Fatal("expected ClaimPending to skip stale candidate and return another row")
	}
	if got.ApprovalID != second.ApprovalID {
		t.Fatalf("expected second event to be claimed, got %#v", got)
	}

	var firstRefreshed model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", first.ApprovalID, first.EventType).First(&firstRefreshed).Error; err != nil {
		t.Fatalf("reload first event: %v", err)
	}
	if firstRefreshed.Status != "processing" {
		t.Fatalf("expected stale candidate to remain processing, got %q", firstRefreshed.Status)
	}

	var secondRefreshed model.AIApprovalOutboxEvent
	if err := db.Where("approval_id = ? AND event_type = ?", second.ApprovalID, second.EventType).First(&secondRefreshed).Error; err != nil {
		t.Fatalf("reload second event: %v", err)
	}
	if secondRefreshed.Status != "processing" {
		t.Fatalf("expected second event to be claimed after stale attempt, got %q", secondRefreshed.Status)
	}
}

func TestApprovalOutboxSchemaIndexes(t *testing.T) {
	db := newApprovalOutboxTestDB(t)

	if !db.Migrator().HasTable(&model.AIApprovalOutboxEvent{}) {
		t.Fatal("expected ai_approval_outbox_events table")
	}
	if !db.Migrator().HasIndex(&model.AIApprovalOutboxEvent{}, "uk_ai_approval_outbox_events_approval_event") {
		t.Fatal("expected unique index uk_ai_approval_outbox_events_approval_event")
	}
	if !db.Migrator().HasIndex(&model.AIApprovalOutboxEvent{}, "idx_ai_approval_outbox_events_status_next_retry_created") {
		t.Fatal("expected queue index idx_ai_approval_outbox_events_status_next_retry_created")
	}
}

func newApprovalOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIApprovalOutboxEvent{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func newConcurrentApprovalOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "approval_outbox.db")
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		t.Fatalf("enable sqlite wal mode: %v", err)
	}
	if err := db.AutoMigrate(&model.AIApprovalOutboxEvent{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}

func enableSQLiteBusyTimeoutApprovalOutbox(t *testing.T, db *gorm.DB) {
	t.Helper()

	if execErr := db.Exec("PRAGMA busy_timeout = 5000").Error; execErr != nil {
		t.Fatalf("set sqlite busy timeout: %v", execErr)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(8)
	sqlDB.SetMaxIdleConns(8)
}
