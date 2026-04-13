package event

import (
	"testing"

	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

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
	if err := db.Create(&ai.AIChatMessage{ID: seed.assistantMessageID, SessionID: seed.sessionID, SessionIDNum: 2, Role: "assistant", Content: "", Status: "in_progress"}).Error; err != nil {
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
