package chat

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/model"
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

func TestCreateMessage_ConcurrentRequestsAllocateUniqueSequence(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared&_busy_timeout=10000", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.AIChatSession{}, &model.AIChatMessage{}); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}

	dao := NewAIChatDAO(db)
	ctx := context.Background()
	session := &model.AIChatSession{
		ID:     "session-concurrent",
		UserID: 7,
		Scene:  "ai",
		Title:  "concurrent",
	}
	if err := dao.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	const attempts = 6
	start := make(chan struct{})
	errs := make(chan error, attempts)
	var wg sync.WaitGroup

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs <- dao.CreateMessage(ctx, &model.AIChatMessage{
				ID:        fmt.Sprintf("msg-concurrent-%d", i),
				SessionID: session.ID,
				Role:      "user",
				Content:   fmt.Sprintf("message-%d", i),
				Status:    "done",
			})
		}(i)
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("expected concurrent create to succeed, got %v", err)
		}
	}

	messages, err := dao.ListMessagesBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(messages) != attempts {
		t.Fatalf("expected %d messages, got %d", attempts, len(messages))
	}
	for i, message := range messages {
		if message.SessionIDNum != i+1 {
			t.Fatalf("expected contiguous session_id_num, got %#v", messages)
		}
	}
}

func TestChatDAO_DoesNotExposeLegacyRuntimeField(t *testing.T) {
	legacyFieldName := "Runtime" + "JSON"
	if _, ok := reflect.TypeOf(model.AIChatMessage{}).FieldByName(legacyFieldName); ok {
		t.Fatal("did not expect legacy runtime snapshot field on model.AIChatMessage")
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
