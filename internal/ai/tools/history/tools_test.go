package history

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLoadSessionHistory_RecentOnlyReturnsFinalMessages(t *testing.T) {
	t.Parallel()

	db := newHistoryToolTestDB(t)
	seedHistorySession(t, db, model.AIChatSession{
		ID:        "sess-1",
		UserID:    9,
		Scene:     "ai",
		Title:     "history",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	seedHistoryMessage(t, db, model.AIChatMessage{ID: "m1", SessionID: "sess-1", SessionIDNum: 1, Role: "user", Content: "第一次提问", Status: "done"})
	seedHistoryMessage(t, db, model.AIChatMessage{ID: "m2", SessionID: "sess-1", SessionIDNum: 2, Role: "assistant", Content: "第一次最终回答", Status: "done"})
	seedHistoryMessage(t, db, model.AIChatMessage{ID: "m3", SessionID: "sess-1", SessionIDNum: 3, Role: "assistant", Content: "", Status: "streaming"})
	seedHistoryMessage(t, db, model.AIChatMessage{ID: "m4", SessionID: "sess-1", SessionIDNum: 4, Role: "user", Content: "第二次提问", Status: "done"})
	seedHistoryMessage(t, db, model.AIChatMessage{ID: "m5", SessionID: "sess-1", SessionIDNum: 5, Role: "assistant", Content: "第二次最终回答", Status: "done"})

	toolCtx := runtimectx.WithServices(context.Background(), &svc.ServiceContext{DB: db})
	toolCtx = runtimectx.WithAIMetadata(toolCtx, runtimectx.AIMetadata{
		SessionID: "sess-1",
		UserID:    9,
		Scene:     "ai",
	})

	result, err := LoadSessionHistory(toolCtx).InvokableRun(toolCtx, `{"mode":"recent","max_turns":2}`)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}

	for _, fragment := range []string{"第一次提问", "第一次最终回答", "第二次提问", "第二次最终回答"} {
		if !strings.Contains(result, fragment) {
			t.Fatalf("expected result to contain %q, got %s", fragment, result)
		}
	}
	if strings.Contains(result, `"activities"`) {
		t.Fatalf("expected structured runtime details to be excluded, got %s", result)
	}
}

func TestLoadSessionHistory_CompactCompressesOlderMessages(t *testing.T) {
	t.Parallel()

	db := newHistoryToolTestDB(t)
	seedHistorySession(t, db, model.AIChatSession{
		ID:        "sess-compact",
		UserID:    10,
		Scene:     "ai",
		Title:     "compact",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
	for i := 1; i <= 8; i++ {
		seedHistoryMessage(t, db, model.AIChatMessage{
			ID:           "u" + string(rune('a'+i)),
			SessionID:    "sess-compact",
			SessionIDNum: i*2 - 1,
			Role:         "user",
			Content:      strings.Repeat("用户问题内容很长 ", 20),
			Status:       "done",
		})
		seedHistoryMessage(t, db, model.AIChatMessage{
			ID:           "a" + string(rune('a'+i)),
			SessionID:    "sess-compact",
			SessionIDNum: i * 2,
			Role:         "assistant",
			Content:      "助手最终回答 " + strings.Repeat("详细说明 ", 20),
			Status:       "done",
		})
	}

	toolCtx := runtimectx.WithServices(context.Background(), &svc.ServiceContext{DB: db})
	toolCtx = runtimectx.WithAIMetadata(toolCtx, runtimectx.AIMetadata{
		SessionID: "sess-compact",
		UserID:    10,
		Scene:     "ai",
	})

	result, err := LoadSessionHistory(toolCtx).InvokableRun(toolCtx, `{"mode":"compact","max_turns":2,"max_chars":1200}`)
	if err != nil {
		t.Fatalf("load compact history: %v", err)
	}

	if !strings.Contains(result, "Earlier conversation summary") {
		t.Fatalf("expected compact output to contain summary header, got %s", result)
	}
	if !strings.Contains(result, "Recent conversation") {
		t.Fatalf("expected compact output to contain recent header, got %s", result)
	}
	if len(result) > 1400 {
		t.Fatalf("expected compact output to stay bounded, got length %d", len(result))
	}
}

func newHistoryToolTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.AIChatSession{}, &model.AIChatMessage{}); err != nil {
		t.Fatalf("migrate history tables: %v", err)
	}
	return db
}

func seedHistorySession(t *testing.T, db *gorm.DB, session model.AIChatSession) {
	t.Helper()
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func seedHistoryMessage(t *testing.T, db *gorm.DB, message model.AIChatMessage) {
	t.Helper()
	if err := db.Create(&message).Error; err != nil {
		t.Fatalf("seed message: %v", err)
	}
}
