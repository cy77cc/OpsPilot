package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
)

func TestChatHandler_ReturnsSSEContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(101))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"test message"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", contentType)
	}
}

func TestChatHandler_EmitsMetaEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(102))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	events := decodeSSEEvents(t, recorder.Body.String())
	if len(events) == 0 {
		t.Fatal("expected at least one SSE event")
	}

	metaEvent := events[0]
	if metaEvent.Event != "meta" {
		t.Fatalf("expected first event to be 'meta', got %q", metaEvent.Event)
	}

	data, ok := metaEvent.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected meta data to be a map, got %T", metaEvent.Data)
	}

	if _, ok := data["session_id"]; !ok {
		t.Fatal("expected meta event to contain session_id")
	}
	if _, ok := data["run_id"]; !ok {
		t.Fatal("expected meta event to contain run_id")
	}
	if _, ok := data["turn"]; !ok {
		t.Fatal("expected meta event to contain turn")
	}
}

func TestChatHandler_WithSessionID_ReusesSession(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	// First request creates session
	recorder1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(recorder1)
	c1.Set("uid", uint64(103))
	c1.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"first"}`))
	c1.Request.Header.Set("Content-Type", "application/json")
	h.Chat(c1)

	events1 := decodeSSEEvents(t, recorder1.Body.String())
	metaData1 := events1[0].Data.(map[string]any)
	sessionID := metaData1["session_id"].(string)

	// Second request with session_id
	recorder2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(recorder2)
	c2.Set("uid", uint64(103))
	c2.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"second","session_id":"`+sessionID+`"}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	h.Chat(c2)

	events2 := decodeSSEEvents(t, recorder2.Body.String())
	metaData2 := events2[0].Data.(map[string]any)

	if metaData2["session_id"] != sessionID {
		t.Fatalf("expected session_id to be %q, got %q", sessionID, metaData2["session_id"])
	}
}

func TestChatHandler_PersistsSceneFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(104))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"inspect cluster","scene":"cluster","context":{"route":"/deployment/infrastructure/clusters/42","resource_id":"42"}}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	events := decodeSSEEvents(t, recorder.Body.String())
	metaData := events[0].Data.(map[string]any)
	sessionID := metaData["session_id"].(string)

	var session model.AIChatSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		t.Fatalf("load session: %v", err)
	}

	if session.Scene != "cluster" {
		t.Fatalf("expected scene cluster, got %q", session.Scene)
	}
}

func decodeSSEEvents(t *testing.T, body string) []chatEvent {
	t.Helper()

	lines := strings.Split(body, "\n")
	events := make([]chatEvent, 0)
	current := chatEvent{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if current.Event != "" || current.Data != nil {
				events = append(events, current)
			}
			current = chatEvent{}
			continue
		}
		if strings.HasPrefix(trimmed, "event:") {
			current.Event = strings.TrimSpace(strings.TrimPrefix(trimmed, "event:"))
			continue
		}
		if !strings.HasPrefix(trimmed, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			t.Fatalf("decode SSE event %q: %v", raw, err)
		}
		current.Data = data
	}
	if current.Event != "" || current.Data != nil {
		events = append(events, current)
	}
	return events
}

type chatEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}
