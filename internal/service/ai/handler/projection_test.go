package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestGetRunProjection_ReturnsPersistedProjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	seedProjectionFixture(t, db)
	if err := aidao.NewAIRunProjectionDAO(db).Upsert(context.Background(), &model.AIRunProjection{
		ID:             "proj-1",
		RunID:          "run-1",
		SessionID:      "session-1",
		Version:        1,
		Status:         "completed",
		ProjectionJSON: `{"status":"completed","blocks":[]}`,
	}); err != nil {
		t.Fatalf("seed projection: %v", err)
	}

	h := NewAIHandlerWithDB(db)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Params = gin.Params{{Key: "runId", Value: "run-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/runs/run-1/projection", nil)

	h.GetRunProjection(c)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 1000 || resp.Data["status"] != "completed" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGetRunContent_ReturnsLazyPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	seedProjectionFixture(t, db)
	if err := aidao.NewAIRunContentDAO(db).Create(context.Background(), &model.AIRunContent{
		ID:          "content-1",
		RunID:       "run-1",
		SessionID:   "session-1",
		ContentKind: "executor_content",
		Encoding:    "text",
		BodyText:    "hello",
	}); err != nil {
		t.Fatalf("seed content: %v", err)
	}

	h := NewAIHandlerWithDB(db)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Params = gin.Params{{Key: "id", Value: "content-1"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/run-contents/content-1", nil)

	h.GetRunContent(c)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 1000 || resp.Data["body_text"] != "hello" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func seedProjectionFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	chatDAO := aidao.NewAIChatDAO(db)
	runDAO := aidao.NewAIRunDAO(db)

	if err := chatDAO.CreateSession(context.Background(), &model.AIChatSession{
		ID: "session-1", UserID: 1, Scene: "ai", Title: "test",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := runDAO.CreateRun(context.Background(), &model.AIRun{
		ID: "run-1", SessionID: "session-1", UserMessageID: "msg-1", Status: "completed", TraceJSON: "{}",
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
}
