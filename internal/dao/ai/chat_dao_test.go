package ai

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateMessage_AssignsSessionSequenceAndListsBySequence(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIChatDAO(db)
	ctx := context.Background()

	session := &model.AIChatSession{
		ID:     "session-1",
		UserID: 7,
		Scene:  "ai",
		Title:  "test",
	}
	if err := dao.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	first := &model.AIChatMessage{
		ID:        "msg-1",
		SessionID: session.ID,
		Role:      "user",
		Content:   "hello",
		Status:    "done",
	}
	if err := dao.CreateMessage(ctx, first); err != nil {
		t.Fatalf("create first message: %v", err)
	}
	if first.SessionIDNum != 1 {
		t.Fatalf("expected first session_id_num to be 1, got %d", first.SessionIDNum)
	}

	second := &model.AIChatMessage{
		ID:        "msg-2",
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "world",
		Status:    "done",
	}
	if err := dao.CreateMessage(ctx, second); err != nil {
		t.Fatalf("create second message: %v", err)
	}
	if second.SessionIDNum != 2 {
		t.Fatalf("expected second session_id_num to be 2, got %d", second.SessionIDNum)
	}

	messages, err := dao.ListMessagesBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].ID != first.ID || messages[1].ID != second.ID {
		t.Fatalf("expected messages ordered by session sequence, got %#v", messages)
	}
}

func TestCreateMessage_TouchesSessionUpdatedAt(t *testing.T) {
	db := newAIDAOTestDB(t)
	dao := NewAIChatDAO(db)
	ctx := context.Background()

	session := &model.AIChatSession{
		ID:        "session-2",
		UserID:    9,
		Scene:     "ai",
		Title:     "touch",
		CreatedAt: time.Now().Add(-time.Hour),
		UpdatedAt: time.Now().Add(-time.Hour),
	}
	if err := db.Create(session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	before := session.UpdatedAt

	message := &model.AIChatMessage{
		ID:        "msg-3",
		SessionID: session.ID,
		Role:      "user",
		Content:   "ping",
		Status:    "done",
	}
	if err := dao.CreateMessage(ctx, message); err != nil {
		t.Fatalf("create message: %v", err)
	}

	var refreshed model.AIChatSession
	if err := db.First(&refreshed, "id = ?", session.ID).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if !refreshed.UpdatedAt.After(before) {
		t.Fatalf("expected updated_at to advance, before=%s after=%s", before, refreshed.UpdatedAt)
	}
}

func TestChatDAO_DoesNotExposeRuntimeJSON(t *testing.T) {
	if _, ok := reflect.TypeOf(model.AIChatMessage{}).FieldByName("RuntimeJSON"); ok {
		t.Fatal("did not expect RuntimeJSON field on AIChatMessage")
	}
}

func newAIDAOTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AIRun{},
		&model.AIRunEvent{},
		&model.AIRunProjection{},
		&model.AIRunContent{},
		&model.AICheckpoint{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	return db
}
