package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
)

func TestRegisterAIHandlers_RegistersPhase1Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	registerAIHandlersForTest(v1)

	routes := r.Routes()
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = true
	}

	expected := []string{
		"POST /api/v1/ai/chat",
		"GET /api/v1/ai/sessions",
		"POST /api/v1/ai/sessions",
		"GET /api/v1/ai/sessions/:id",
		"DELETE /api/v1/ai/sessions/:id",
		"GET /api/v1/ai/runs/:runId",
		"GET /api/v1/ai/runs/:runId/projection",
		"GET /api/v1/ai/run-contents/:id",
		"GET /api/v1/ai/diagnosis/:reportId",
	}

	for _, e := range expected {
		if !seen[e] {
			t.Errorf("missing route %q", e)
		}
	}
}

func TestListSessions_ReturnsEmptyArrayForNewUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(1))
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions", nil)

	h.ListSessions(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	data := response["data"]
	if data == nil {
		t.Fatal("expected data field in response")
	}
}

func TestCreateSession_ReturnsSessionWithID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(2))
	c.Request = httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(`{"title":"Test Session","scene":"ai"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.CreateSession(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	data, ok := response["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}

	if data["id"] == "" {
		t.Fatal("expected session to have an id")
	}
}

func TestGetSession_ReturnsNotFoundForNonexistentSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(3))
	c.Params = gin.Params{{Key: "id", Value: "nonexistent-id"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions/nonexistent-id", nil)

	h.GetSession(c)

	// httpx.NotFound returns HTTP 200 with business error code 2005
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 (httpx convention), got %d", recorder.Code)
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// 2005 is the NotFound business code
	if resp.Code != 2005 {
		t.Fatalf("expected business code 2005 (NotFound), got %d", resp.Code)
	}
}

func TestListSessions_FiltersByScene(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	now := time.Now()
	seedSession(t, db, model.AIChatSession{ID: "sess-host", UserID: 9, Title: "Host", Scene: "host", CreatedAt: now, UpdatedAt: now})
	seedSession(t, db, model.AIChatSession{ID: "sess-cluster", UserID: 9, Title: "Cluster", Scene: "cluster", CreatedAt: now, UpdatedAt: now})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(9))
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions?scene=cluster", nil)

	h.ListSessions(c)

	var response struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(response.Data) != 1 {
		t.Fatalf("expected 1 session, got %d", len(response.Data))
	}
	if response.Data[0]["scene"] != "cluster" {
		t.Fatalf("expected cluster scene, got %v", response.Data[0]["scene"])
	}
}

func TestGetSession_RespectsSceneFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	now := time.Now()
	seedSession(t, db, model.AIChatSession{ID: "sess-cluster", UserID: 10, Title: "Cluster", Scene: "cluster", CreatedAt: now, UpdatedAt: now})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(10))
	c.Params = gin.Params{{Key: "id", Value: "sess-cluster"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions/sess-cluster?scene=host", nil)

	h.GetSession(c)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 2005 {
		t.Fatalf("expected not found code 2005, got %d", resp.Code)
	}
}

func TestGetSession_AssistantMessageIncludesContentAndKeepsRunID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)
	now := time.Now()

	seedSession(t, db, model.AIChatSession{ID: "sess-run", UserID: 20, Title: "With Run", Scene: "ai", CreatedAt: now, UpdatedAt: now})
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-user",
		SessionID:    "sess-run",
		SessionIDNum: 1,
		Role:         "user",
		Content:      "hello",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-assistant",
		SessionID:    "sess-run",
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "world",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
		ID:                 "run-20",
		SessionID:          "sess-run",
		UserMessageID:      "msg-user",
		AssistantMessageID: "msg-assistant",
		Status:             "completed",
		TraceJSON:          "{}",
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(20))
	c.Params = gin.Params{{Key: "id", Value: "sess-run"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions/sess-run", nil)

	h.GetSession(c)

	var response struct {
		Data struct {
			Messages []map[string]any `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(response.Data.Messages))
	}
	if response.Data.Messages[0]["content"] != "hello" {
		t.Fatalf("expected user message to keep content, got %#v", response.Data.Messages[0])
	}
	if response.Data.Messages[1]["run_id"] != "run-20" {
		t.Fatalf("expected assistant message to include run_id, got %#v", response.Data.Messages[1])
	}
	if response.Data.Messages[1]["content"] != "world" {
		t.Fatalf("expected assistant message to include content, got %#v", response.Data.Messages[1])
	}
}

func TestGetSession_IncludesAssistantFallbackBodyAndTerminalErrorState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)
	now := time.Now()

	cases := []struct {
		name         string
		runStatus    string
		expectedBody string
	}{
		{
			name:         "failed_runtime",
			runStatus:    "failed_runtime",
			expectedBody: "partial answer",
		},
		{
			name:         "expired",
			runStatus:    "expired",
			expectedBody: "expired answer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := "sess-" + tc.name
			userMessageID := "msg-user-" + tc.name
			assistantMessageID := "msg-assistant-" + tc.name
			runID := "run-" + tc.name

			seedSession(t, db, model.AIChatSession{ID: sessionID, UserID: 22, Title: "Failed", Scene: "ai", CreatedAt: now, UpdatedAt: now})
			if err := db.Create(&model.AIChatMessage{
				ID:           userMessageID,
				SessionID:    sessionID,
				SessionIDNum: 1,
				Role:         "user",
				Content:      "hello",
				Status:       "done",
			}).Error; err != nil {
				t.Fatalf("seed user message: %v", err)
			}
			if err := db.Create(&model.AIChatMessage{
				ID:           assistantMessageID,
				SessionID:    sessionID,
				SessionIDNum: 2,
				Role:         "assistant",
				Content:      tc.expectedBody,
				Status:       "done",
			}).Error; err != nil {
				t.Fatalf("seed assistant message: %v", err)
			}
			if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
				ID:                 runID,
				SessionID:          sessionID,
				UserMessageID:      userMessageID,
				AssistantMessageID: assistantMessageID,
				Status:             tc.runStatus,
				ErrorMessage:       "internal stack trace",
				TraceJSON:          "{}",
			}); err != nil {
				t.Fatalf("seed run: %v", err)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Set("uid", uint64(22))
			c.Params = gin.Params{{Key: "id", Value: sessionID}}
			c.Request = httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID, nil)

			h.GetSession(c)

			var response struct {
				Data struct {
					Messages []map[string]any `json:"messages"`
				} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			if len(response.Data.Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(response.Data.Messages))
			}

			assistant := response.Data.Messages[1]
			if assistant["content"] != tc.expectedBody {
				t.Fatalf("expected assistant content %q, got %#v", tc.expectedBody, assistant)
			}
			if assistant["status"] != "error" {
				t.Fatalf("expected assistant status error, got %#v", assistant)
			}
			if assistant["error_message"] != "生成中断，请稍后重试。" {
				t.Fatalf("expected terminal error metadata, got %#v", assistant)
			}
			if assistant["run_id"] != runID {
				t.Fatalf("expected run_id %q, got %#v", runID, assistant)
			}
		})
	}
}

func TestGetSession_PreservesAssistantErrorStatusForFailedAndExpiredRuns(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)
	now := time.Now()

	cases := []struct {
		name      string
		runStatus string
	}{
		{name: "failed", runStatus: "failed"},
		{name: "failed_runtime", runStatus: "failed_runtime"},
		{name: "expired", runStatus: "expired"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessionID := "sess-status-" + tc.name
			userMessageID := "msg-user-status-" + tc.name
			assistantMessageID := "msg-assistant-status-" + tc.name
			runID := "run-status-" + tc.name

			seedSession(t, db, model.AIChatSession{ID: sessionID, UserID: 23, Title: "Status", Scene: "ai", CreatedAt: now, UpdatedAt: now})
			if err := db.Create(&model.AIChatMessage{
				ID:           userMessageID,
				SessionID:    sessionID,
				SessionIDNum: 1,
				Role:         "user",
				Content:      "hello",
				Status:       "done",
			}).Error; err != nil {
				t.Fatalf("seed user message: %v", err)
			}
			if err := db.Create(&model.AIChatMessage{
				ID:           assistantMessageID,
				SessionID:    sessionID,
				SessionIDNum: 2,
				Role:         "assistant",
				Content:      "partial answer",
				Status:       "done",
			}).Error; err != nil {
				t.Fatalf("seed assistant message: %v", err)
			}
			if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
				ID:                 runID,
				SessionID:          sessionID,
				UserMessageID:      userMessageID,
				AssistantMessageID: assistantMessageID,
				Status:             tc.runStatus,
				ErrorMessage:       "internal stack trace",
				TraceJSON:          "{}",
			}); err != nil {
				t.Fatalf("seed run: %v", err)
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Set("uid", uint64(23))
			c.Params = gin.Params{{Key: "id", Value: sessionID}}
			c.Request = httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID, nil)

			h.GetSession(c)

			var response struct {
				Data struct {
					Messages []map[string]any `json:"messages"`
				} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}

			assistant := response.Data.Messages[1]
			if assistant["status"] != "error" {
				t.Fatalf("expected assistant status error, got %#v", assistant)
			}
			if assistant["error_message"] != "生成中断，请稍后重试。" {
				t.Fatalf("expected terminal error metadata, got %#v", assistant)
			}
		})
	}
}

func TestListSessions_AssistantMessagesIncludeContentButKeepRunID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)
	now := time.Now()

	seedSession(t, db, model.AIChatSession{ID: "sess-list-run", UserID: 21, Title: "With Run", Scene: "ai", CreatedAt: now, UpdatedAt: now})
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-user-list",
		SessionID:    "sess-list-run",
		SessionIDNum: 1,
		Role:         "user",
		Content:      "hello",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-assistant-list",
		SessionID:    "sess-list-run",
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "world",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
		ID:                 "run-21",
		SessionID:          "sess-list-run",
		UserMessageID:      "msg-user-list",
		AssistantMessageID: "msg-assistant-list",
		Status:             "completed",
		TraceJSON:          "{}",
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(21))
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions", nil)

	h.ListSessions(c)

	var response struct {
		Data []struct {
			Messages []map[string]any `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("expected 1 session, got %d", len(response.Data))
	}
	if len(response.Data[0].Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(response.Data[0].Messages))
	}
	if response.Data[0].Messages[0]["content"] != "hello" {
		t.Fatalf("expected user message to keep content, got %#v", response.Data[0].Messages[0])
	}
	if response.Data[0].Messages[1]["run_id"] != "run-21" {
		t.Fatalf("expected assistant message to include run_id, got %#v", response.Data[0].Messages[1])
	}
	if response.Data[0].Messages[1]["content"] != "world" {
		t.Fatalf("expected assistant message to include content, got %#v", response.Data[0].Messages[1])
	}
}

func seedSession(t *testing.T, db *gorm.DB, session model.AIChatSession) {
	t.Helper()
	if err := db.Create(&session).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func newAIHandlerTestDB(t *testing.T) *gorm.DB {
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
		&model.AIDiagnosisReport{},
		&model.AIScenePrompt{},
		&model.AISceneConfig{},
	); err != nil {
		t.Fatalf("auto migrate ai handler tables: %v", err)
	}
	return db
}

func registerAIHandlersForTest(v1 *gin.RouterGroup) {
	h := &Handler{logic: &logic.Logic{}}
	g := v1.Group("/ai")
	{
		g.POST("/chat", h.Chat)
		g.GET("/sessions", h.ListSessions)
		g.POST("/sessions", h.CreateSession)
		g.GET("/sessions/:id", h.GetSession)
		g.DELETE("/sessions/:id", h.DeleteSession)
		g.GET("/runs/:runId", h.GetRun)
		g.GET("/runs/:runId/projection", h.GetRunProjection)
		g.GET("/run-contents/:id", h.GetRunContent)
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)
	}
}
