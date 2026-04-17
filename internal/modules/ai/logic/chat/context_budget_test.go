package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aidaochat "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/chat"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	runtimecontext "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/context"
)

func TestBuildSessionAgentInput_UsesBudgetedSessionHistory(t *testing.T) {
	db := newChatRuntimeTestDB(t)
	dao := aidaochat.NewAIChatDAO(db)
	ctx := context.Background()

	session := &ai.AIChatSession{
		ID:     "session-1",
		UserID: 7,
		Scene:  "ops",
		Title:  "budgeted",
	}
	if err := dao.CreateSession(ctx, session); err != nil {
		t.Fatalf("create session: %v", err)
	}

	rows := []ai.AIChatMessage{
		{ID: "msg-1", SessionID: session.ID, Role: "system", Content: "pinned-1", Status: "done"},
		{ID: "msg-2", SessionID: session.ID, Role: "user", Content: "h1", Status: "done"},
		{ID: "msg-3", SessionID: session.ID, Role: "assistant", Content: "h2", Status: "done"},
		{ID: "msg-4", SessionID: session.ID, Role: "user", Content: "recent-1", Status: "done"},
		{ID: "msg-5", SessionID: session.ID, Role: "assistant", Content: "recent-2", Status: "done"},
	}
	for i := range rows {
		if err := dao.CreateMessage(ctx, &rows[i]); err != nil {
			t.Fatalf("create history message %d: %v", i, err)
		}
	}

	l := &Logic{ChatDAO: dao}
	shell := ChatShell{
		SessionID: session.ID,
		Scene:     session.Scene,
		UserMessage: &ai.AIChatMessage{
			ID: "current-user",
		},
		AssistantMessage: &ai.AIChatMessage{
			ID: "current-assistant",
		},
	}

	got := buildSessionAgentInput(ctx, l, shell, ChatInput{
		Message: "hello",
		Budget: runtimecontext.Budget{
			Pinned:  1,
			Recent:  2,
			History: 1,
		},
	})

	if len(got) != 6 {
		t.Fatalf("expected 6 messages including current turn, got %d", len(got))
	}
	if got[0].Role != "system" || got[0].Content != "compressed 1 overflow messages" {
		t.Fatalf("expected compressed overflow marker first, got %#v", got[0])
	}
	if got[1].Role != "system" || got[1].Content != "pinned-1" {
		t.Fatalf("expected pinned system message second, got %#v", got[1])
	}
	if got[2].Content != "h2" || got[3].Content != "recent-1" || got[4].Content != "recent-2" {
		t.Fatalf("expected budgeted history selection in the middle, got %#v", got)
	}
	if got[5].Role != "user" || !strings.Contains(got[5].Content, "hello") {
		t.Fatalf("expected current user turn last, got %#v", got[5])
	}
}

func newChatRuntimeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&ai.AIChatSession{}, &ai.AIChatMessage{}); err != nil {
		t.Fatalf("migrate chat tables: %v", err)
	}
	return db
}
