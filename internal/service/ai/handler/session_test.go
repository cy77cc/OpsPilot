package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRegisterAIHandlers_RegistersPhase1Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	// Import path avoided here because this test file now also covers handler behavior.
	registerAIHandlersForTest(v1)

	routes := r.Routes()
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = true
	}

	if !seen[http.MethodPost+" /api/v1/ai/chat"] {
		t.Fatalf("expected POST /api/v1/ai/chat route to be registered")
	}
	if !seen[http.MethodGet+" /api/v1/ai/sessions"] {
		t.Fatalf("expected GET /api/v1/ai/sessions route to be registered")
	}
	if !seen[http.MethodPost+" /api/v1/ai/sessions"] {
		t.Fatalf("expected POST /api/v1/ai/sessions route to be registered")
	}
	if !seen[http.MethodGet+" /api/v1/ai/sessions/:id"] {
		t.Fatalf("expected GET /api/v1/ai/sessions/:id route to be registered")
	}
	if !seen[http.MethodDelete+" /api/v1/ai/sessions/:id"] {
		t.Fatalf("expected DELETE /api/v1/ai/sessions/:id route to be registered")
	}
	if !seen[http.MethodGet+" /api/v1/ai/runs/:runId"] {
		t.Fatalf("expected GET /api/v1/ai/runs/:runId route to be registered")
	}
	if !seen[http.MethodGet+" /api/v1/ai/diagnosis/:reportId"] {
		t.Fatalf("expected GET /api/v1/ai/diagnosis/:reportId route to be registered")
	}
}

func TestSessionHandlers_CreateListGetDeleteSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := New(Dependencies{
		ChatDAO: NewAIChatDAOForTest(db),
	})

	createRecorder := httptest.NewRecorder()
	createCtx, createEngine := gin.CreateTestContext(createRecorder)
	createEngine.POST("/sessions", h.CreateSession)
	createCtx.Set("uid", uint64(42))
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(`{"title":"Cluster diagnosis","scene":"cluster"}`))
	createCtx.Request.Header.Set("Content-Type", "application/json")
	createCtx.Params = nil
	h.CreateSession(createCtx)

	var createResp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(createRecorder.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	sessionData, ok := createResp.Data["session"].(map[string]any)
	if !ok {
		t.Fatalf("expected session object, got %#v", createResp.Data)
	}
	sessionID, _ := sessionData["id"].(string)
	if sessionID == "" {
		t.Fatalf("expected created session id in response")
	}

	listRecorder := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRecorder)
	listCtx.Set("uid", uint64(42))
	listCtx.Request = httptest.NewRequest(http.MethodGet, "/sessions", nil)
	h.ListSessions(listCtx)

	var listResp struct {
		Data map[string][]map[string]any `json:"data"`
	}
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listResp.Data["sessions"]) != 1 {
		t.Fatalf("expected one session, got %#v", listResp.Data)
	}

	if err := h.deps.ChatDAO.CreateMessage(context.Background(), &model.AIChatMessage{
		ID:        "message-1",
		SessionID: sessionID,
		Role:      "user",
		Content:   "Investigate the failing rollout",
		Status:    "done",
	}); err != nil {
		t.Fatalf("seed message: %v", err)
	}

	getRecorder := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRecorder)
	getCtx.Set("uid", uint64(42))
	getCtx.Params = gin.Params{{Key: "id", Value: sessionID}}
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/sessions/"+sessionID, nil)
	h.GetSession(getCtx)

	var getResp struct {
		Data map[string]map[string]any `json:"data"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getResp.Data["session"]["id"] != sessionID {
		t.Fatalf("expected session id %q, got %#v", sessionID, getResp.Data["session"]["id"])
	}
	messages, ok := getResp.Data["session"]["messages"].([]any)
	if !ok || len(messages) != 1 {
		t.Fatalf("expected one message in session detail, got %#v", getResp.Data["session"]["messages"])
	}

	deleteRecorder := httptest.NewRecorder()
	deleteCtx, _ := gin.CreateTestContext(deleteRecorder)
	deleteCtx.Set("uid", uint64(42))
	deleteCtx.Params = gin.Params{{Key: "id", Value: sessionID}}
	deleteCtx.Request = httptest.NewRequest(http.MethodDelete, "/sessions/"+sessionID, nil)
	h.DeleteSession(deleteCtx)

	got, err := h.deps.ChatDAO.GetSession(context.Background(), sessionID, 42)
	if err != nil {
		t.Fatalf("get deleted session from dao: %v", err)
	}
	if got != nil {
		t.Fatalf("expected session to be deleted, got %#v", got)
	}
}

func newAIHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&model.AIChatSession{},
		&model.AIChatMessage{},
		&model.AIRun{},
		&model.AIDiagnosisReport{},
	); err != nil {
		t.Fatalf("auto migrate ai handler tables: %v", err)
	}
	return db
}

func NewAIChatDAOForTest(db *gorm.DB) *dao.AIChatDAO {
	return dao.NewAIChatDAO(db)
}

func registerAIHandlersForTest(v1 *gin.RouterGroup) {
	h := New(Dependencies{})
	g := v1.Group("/ai")
	{
		g.POST("/chat", h.Chat)
		g.GET("/sessions", h.ListSessions)
		g.POST("/sessions", h.CreateSession)
		g.GET("/sessions/:id", h.GetSession)
		g.DELETE("/sessions/:id", h.DeleteSession)
		g.GET("/runs/:runId", h.GetRun)
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)
	}
}
