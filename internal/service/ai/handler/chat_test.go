package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/gin-gonic/gin"
)

func TestChatHandler_PassesClientRequestIDIntoLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)
	agent := &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("ok", nil), nil, schema.Assistant, ""),
		},
	}
	h.logic.AIRouter = agent

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(100))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hi","client_request_id":"req-1"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if agent.capturedRequestID != "req-1" {
		t.Fatalf("expected runtime request id req-1, got %q", agent.capturedRequestID)
	}
}

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

func TestChatStreamsRecoverableToolErrorAndDone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := NewAIHandlerWithDB(db)
	h.logic.AIRouter = &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("checking pods", nil), nil, schema.Assistant, ""),
			func() *adk.AgentEvent {
				event := adk.EventFromMessage(
					schema.ToolMessage(`{"ok":false}`, "call-1", schema.WithToolName("kubectl_get_pods")),
					nil,
					schema.Tool,
					"kubectl_get_pods",
				)
				event.Err = errors.New("tool execution failed")
				return event
			}(),
		},
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(105))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"inspect pods"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	events := decodeSSEEvents(t, recorder.Body.String())
	if len(events) == 0 {
		t.Fatal("expected SSE events to be emitted")
	}

	var (
		sawToolResult bool
		sawDone       bool
	)
	for _, event := range events {
		if event.Event == "error" {
			t.Fatalf("expected recoverable tool failure to avoid terminal error event, got %#v", event)
		}
		if event.Event == "tool_result" {
			sawToolResult = true
		}
		if event.Event == "done" {
			data, ok := event.Data.(map[string]any)
			if !ok {
				t.Fatalf("expected done data to be a map, got %T", event.Data)
			}
			if data["status"] != "completed_with_tool_errors" {
				t.Fatalf("expected done status completed_with_tool_errors, got %#v", data["status"])
			}
			sawDone = true
		}
	}

	if !sawToolResult {
		t.Fatal("expected tool_result event in recoverable failure stream")
	}
	if !sawDone {
		t.Fatal("expected done event after recoverable failure")
	}
}

func TestChatHandler_EmitsSSEErrorInsteadOfJSONEnvelopeOnLateFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	if err := db.Migrator().DropTable(&model.AIRunProjection{}); err != nil {
		t.Fatalf("drop projection table: %v", err)
	}
	h := NewAIHandlerWithDB(db)
	h.logic.AIRouter = &scriptedAgent{
		runEvents: []*adk.AgentEvent{
			adk.EventFromMessage(schema.AssistantMessage("starting run", nil), nil, schema.Assistant, ""),
			{Err: errors.New("fatal agent failure")},
		},
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(106))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"trigger late failure"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	events := decodeSSEEvents(t, recorder.Body.String())
	if len(events) == 0 {
		t.Fatal("expected SSE events to be emitted")
	}

	sawError := false
	for _, event := range events {
		if event.Event == "error" {
			sawError = true
			break
		}
	}
	if !sawError {
		t.Fatalf("expected SSE error event, got %#v", events)
	}
	if strings.Contains(recorder.Body.String(), `"code":3000`) {
		t.Fatalf("expected SSE stream without trailing JSON error envelope, got %q", recorder.Body.String())
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

type scriptedAgent struct {
	runEvents         []*adk.AgentEvent
	capturedRequestID string
}

func (s *scriptedAgent) Name(context.Context) string        { return "scripted-agent" }
func (s *scriptedAgent) Description(context.Context) string { return "scripted agent for tests" }

func (s *scriptedAgent) Run(ctx context.Context, _ *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	s.capturedRequestID = runtimectx.FromContext(ctx).RequestID
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		for _, event := range s.runEvents {
			gen.Send(event)
		}
		gen.Close()
	}()
	return iter
}

func (s *scriptedAgent) Resume(ctx context.Context, _ *adk.ResumeInfo, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return s.Run(ctx, nil)
}
