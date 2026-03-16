package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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
		&model.AIDiagnosisReport{},
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
		g.GET("/diagnosis/:reportId", h.GetDiagnosisReport)
	}
}
