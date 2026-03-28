package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/gin-gonic/gin"
)

func TestChatHandler_PassesClientRequestIDIntoLogic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)
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

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected accepted SSE response status %d, got %d", http.StatusOK, recorder.Code)
	}
	if agent.capturedRequestID != "req-1" {
		t.Fatalf("expected runtime request id req-1, got %q", agent.capturedRequestID)
	}
}

func TestChatHandler_ReturnsSSEContentType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)

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
	h := newAIHandlerTestHarness(db)

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
	h := newAIHandlerTestHarness(db)

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
	h := newAIHandlerTestHarness(db)

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

func TestGetSession_IncludesResumableRunCredentialsForWaitingApproval(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)
	now := time.Now().UTC().Truncate(time.Millisecond)

	if err := db.AutoMigrate(&model.AIApprovalTask{}); err != nil {
		t.Fatalf("migrate approval task table: %v", err)
	}

	seedSession(t, db, model.AIChatSession{
		ID:        "sess-resumable",
		UserID:    108,
		Title:     "Resumable",
		Scene:     "ai",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-resumable-user",
		SessionID:    "sess-resumable",
		SessionIDNum: 1,
		Role:         "user",
		Content:      "please continue",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-resumable-assistant",
		SessionID:    "sess-resumable",
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "",
		Status:       "in_progress",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
		ID:                 "run-resumable",
		SessionID:          "sess-resumable",
		ClientRequestID:    "req-1",
		UserMessageID:      "msg-resumable-user",
		AssistantMessageID: "msg-resumable-assistant",
		Status:             "waiting_approval",
		TraceJSON:          "{}",
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	for i := 1; i <= 3; i++ {
		if err := db.Create(&model.AIRunEvent{
			ID:          fmt.Sprintf("evt-%d", i),
			RunID:       "run-resumable",
			SessionID:   "sess-resumable",
			Seq:         i,
			EventType:   "run_state",
			PayloadJSON: `{"status":"waiting_approval"}`,
		}).Error; err != nil {
			t.Fatalf("seed run event %d: %v", i, err)
		}
	}
	if err := db.Create(&model.AIApprovalTask{
		ApprovalID:     "approval-resumable",
		CheckpointID:   "checkpoint-resumable",
		SessionID:      "sess-resumable",
		RunID:          "run-resumable",
		UserID:         108,
		ToolName:       "exec_command",
		ToolCallID:     "tool-call-resumable",
		ArgumentsJSON:  `{"cmd":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "pending",
		TimeoutSeconds: 300,
	}).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(108))
	c.Params = gin.Params{{Key: "id", Value: "sess-resumable"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions/sess-resumable", nil)

	h.GetSession(c)

	var response struct {
		Data struct {
			Messages []map[string]any `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var assistant map[string]any
	for _, message := range response.Data.Messages {
		if role, _ := message["role"].(string); role == "assistant" {
			assistant = message
			break
		}
	}
	if assistant == nil {
		t.Fatalf("expected assistant message in session payload, got %#v", response.Data.Messages)
	}
	if assistant["run_id"] != "run-resumable" {
		t.Fatalf("expected assistant run_id to be exposed, got %#v", assistant)
	}
	if assistant["client_request_id"] != "req-1" {
		t.Fatalf("expected assistant client_request_id to be exposed, got %#v", assistant)
	}
	if assistant["latest_event_id"] != "evt-3" {
		t.Fatalf("expected assistant latest_event_id evt-3, got %#v", assistant)
	}
	if assistant["approval_id"] != "approval-resumable" {
		t.Fatalf("expected assistant approval_id to be exposed, got %#v", assistant)
	}
	if assistant["status"] != "waiting_approval" {
		t.Fatalf("expected assistant status waiting_approval, got %#v", assistant)
	}
	if resumable, ok := assistant["resumable"].(bool); !ok || !resumable {
		t.Fatalf("expected assistant resumable=true, got %#v", assistant)
	}
}

func TestGetSession_IncludesResumableRunCredentialsForRetryableResumeFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)
	now := time.Now().UTC().Truncate(time.Millisecond)

	if err := db.AutoMigrate(&model.AIApprovalTask{}); err != nil {
		t.Fatalf("migrate approval task table: %v", err)
	}

	seedSession(t, db, model.AIChatSession{
		ID:        "sess-resume-retryable",
		UserID:    208,
		Title:     "Resume Retryable",
		Scene:     "ai",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-retryable-user",
		SessionID:    "sess-resume-retryable",
		SessionIDNum: 1,
		Role:         "user",
		Content:      "continue",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-retryable-assistant",
		SessionID:    "sess-resume-retryable",
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "",
		Status:       "in_progress",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
		ID:                 "run-resume-retryable",
		SessionID:          "sess-resume-retryable",
		ClientRequestID:    "req-retryable-1",
		UserMessageID:      "msg-retryable-user",
		AssistantMessageID: "msg-retryable-assistant",
		Status:             "resume_failed_retryable",
		TraceJSON:          "{}",
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-retryable-1",
		RunID:       "run-resume-retryable",
		SessionID:   "sess-resume-retryable",
		Seq:         1,
		EventType:   "run_state",
		PayloadJSON: `{"status":"resume_failed_retryable"}`,
	}).Error; err != nil {
		t.Fatalf("seed run event: %v", err)
	}
	if err := db.Create(&model.AIApprovalTask{
		ApprovalID:     "approval-retryable",
		CheckpointID:   "checkpoint-retryable",
		SessionID:      "sess-resume-retryable",
		RunID:          "run-resume-retryable",
		UserID:         208,
		ToolName:       "host_exec",
		ToolCallID:     "call-retryable",
		ArgumentsJSON:  `{"command":"date"}`,
		PreviewJSON:    `{}`,
		Status:         "approved",
		TimeoutSeconds: 300,
	}).Error; err != nil {
		t.Fatalf("seed approval task: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(208))
	c.Params = gin.Params{{Key: "id", Value: "sess-resume-retryable"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/sessions/sess-resume-retryable", nil)

	h.GetSession(c)

	var response struct {
		Data struct {
			Messages []map[string]any `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	var assistant map[string]any
	for _, message := range response.Data.Messages {
		if role, _ := message["role"].(string); role == "assistant" {
			assistant = message
			break
		}
	}
	if assistant == nil {
		t.Fatalf("expected assistant message in session payload, got %#v", response.Data.Messages)
	}
	if assistant["status"] != "resume_failed_retryable" {
		t.Fatalf("expected assistant status resume_failed_retryable, got %#v", assistant)
	}
	if assistant["approval_id"] != "approval-retryable" {
		t.Fatalf("expected assistant approval_id to be exposed, got %#v", assistant)
	}
	if resumable, ok := assistant["resumable"].(bool); !ok || !resumable {
		t.Fatalf("expected assistant resumable=true, got %#v", assistant)
	}
}

func TestChatStreamsRecoverableToolErrorAndDone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)
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
	h := newAIHandlerTestHarness(db)
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

func TestChatHandler_EmitsLiveEventIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(107))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello","client_request_id":"req-live-id"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	raw := recorder.Body.String()
	if !strings.Contains(raw, "id: ") {
		t.Fatalf("expected live SSE stream to include id lines, got %q", raw)
	}
	if strings.Contains(raw, `"event_id"`) {
		t.Fatalf("expected SSE payload to strip event_id field, got %q", raw)
	}
}

func TestChatHandler_ReplaysEventsAfterLastEventID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)

	recorder1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(recorder1)
	c1.Set("uid", uint64(107))
	c1.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello","client_request_id":"req-replay"}`))
	c1.Request.Header.Set("Content-Type", "application/json")
	h.Chat(c1)

	events1 := decodeSSEEvents(t, recorder1.Body.String())
	if len(events1) == 0 {
		t.Fatal("expected first request to emit events")
	}
	metaData, ok := events1[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected meta payload to be a map, got %T", events1[0].Data)
	}
	sessionID, _ := metaData["session_id"].(string)
	runID, _ := metaData["run_id"].(string)
	if sessionID == "" || runID == "" {
		t.Fatalf("expected session_id and run_id in meta payload, got %#v", metaData)
	}

	runEvents, err := aidao.NewAIRunEventDAO(db).ListByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run events: %v", err)
	}
	if len(runEvents) < 2 {
		t.Fatalf("expected at least 2 run events, got %d", len(runEvents))
	}

	recorder2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(recorder2)
	c2.Set("uid", uint64(107))
	c2.Request = httptest.NewRequest(http.MethodPost, "/chat?last_event_id="+runEvents[0].ID, bytes.NewBufferString(`{"message":"hello","session_id":"`+sessionID+`","client_request_id":"req-replay"}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	h.Chat(c2)

	raw := recorder2.Body.String()
	if !strings.Contains(raw, "id: "+runEvents[1].ID) {
		t.Fatalf("expected replayed event id line for %q, got %q", runEvents[1].ID, raw)
	}
	if !strings.Contains(raw, "event: "+runEvents[1].EventType) {
		t.Fatalf("expected replayed event type %q, got %q", runEvents[1].EventType, raw)
	}
}

func TestChatHandler_ReconnectReplaysOnlyEventsAfterLastEventID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)
	now := time.Now().UTC().Truncate(time.Millisecond)

	seedSession(t, db, model.AIChatSession{
		ID:        "sess-reconnect",
		UserID:    109,
		Title:     "Reconnect",
		Scene:     "ai",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-reconnect-user",
		SessionID:    "sess-reconnect",
		SessionIDNum: 1,
		Role:         "user",
		Content:      "hello",
		Status:       "done",
	}).Error; err != nil {
		t.Fatalf("seed user message: %v", err)
	}
	if err := db.Create(&model.AIChatMessage{
		ID:           "msg-reconnect-assistant",
		SessionID:    "sess-reconnect",
		SessionIDNum: 2,
		Role:         "assistant",
		Content:      "",
		Status:       "in_progress",
	}).Error; err != nil {
		t.Fatalf("seed assistant message: %v", err)
	}
	if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
		ID:                 "run-reconnect",
		SessionID:          "sess-reconnect",
		ClientRequestID:    "req-1",
		UserMessageID:      "msg-reconnect-user",
		AssistantMessageID: "msg-reconnect-assistant",
		Status:             "waiting_approval",
		TraceJSON:          "{}",
	}); err != nil {
		t.Fatalf("seed run: %v", err)
	}
	for i := 1; i <= 3; i++ {
		if err := db.Create(&model.AIRunEvent{
			ID:          fmt.Sprintf("evt-%d", i),
			RunID:       "run-reconnect",
			SessionID:   "sess-reconnect",
			Seq:         i,
			EventType:   "run_state",
			PayloadJSON: fmt.Sprintf(`{"status":"checkpoint-%d"}`, i),
		}).Error; err != nil {
			t.Fatalf("seed run event %d: %v", i, err)
		}
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(109))
	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Request = httptest.NewRequest(http.MethodPost, "/chat?last_event_id=evt-2", bytes.NewBufferString(`{"message":"hello","session_id":"sess-reconnect","client_request_id":"req-1"}`)).WithContext(reqCtx)
	c.Request.Header.Set("Content-Type", "application/json")

	streamDone := make(chan struct{})
	go func() {
		defer close(streamDone)
		h.Chat(c)
	}()

	waitFor := func(deadline time.Duration, cond func(string) bool) string {
		timeout := time.Now().Add(deadline)
		for time.Now().Before(timeout) {
			raw := recorder.Body.String()
			if cond(raw) {
				return raw
			}
			time.Sleep(10 * time.Millisecond)
		}
		return recorder.Body.String()
	}

	raw := waitFor(250*time.Millisecond, func(body string) bool {
		return strings.Contains(body, "id: evt-3")
	})
	if !strings.Contains(raw, "id: evt-3") {
		t.Fatalf("expected replay to include evt-3, got %q", raw)
	}

	select {
	case <-streamDone:
		t.Fatalf("expected reconnect stream to remain open for new events, got %q", recorder.Body.String())
	default:
	}

	if err := db.Create(&model.AIRunEvent{
		ID:          "evt-4",
		RunID:       "run-reconnect",
		SessionID:   "sess-reconnect",
		Seq:         4,
		EventType:   "run_state",
		PayloadJSON: `{"status":"checkpoint-4"}`,
	}).Error; err != nil {
		t.Fatalf("seed follow-up event: %v", err)
	}

	raw = waitFor(250*time.Millisecond, func(body string) bool {
		return strings.Contains(body, "id: evt-4")
	})
	if !strings.Contains(raw, "id: evt-4") {
		t.Fatalf("expected reconnect stream to include evt-4, got %q", raw)
	}
	if strings.Contains(raw, "id: evt-1") || strings.Contains(raw, "id: evt-2") {
		t.Fatalf("expected strict incrementality, got %q", raw)
	}

	cancel()
	select {
	case <-streamDone:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected reconnect stream goroutine to exit after request cancellation")
	}
}

func TestChatHandler_UsesLastEventIDHeaderBeforeQueryAndBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)

	recorder1 := httptest.NewRecorder()
	c1, _ := gin.CreateTestContext(recorder1)
	c1.Set("uid", uint64(109))
	c1.Request = httptest.NewRequest(http.MethodPost, "/chat", bytes.NewBufferString(`{"message":"hello","client_request_id":"req-header-precedence"}`))
	c1.Request.Header.Set("Content-Type", "application/json")
	h.Chat(c1)

	events1 := decodeSSEEvents(t, recorder1.Body.String())
	if len(events1) == 0 {
		t.Fatal("expected first request to emit events")
	}
	metaData, ok := events1[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("expected meta payload to be a map, got %T", events1[0].Data)
	}
	sessionID, _ := metaData["session_id"].(string)
	runID, _ := metaData["run_id"].(string)
	if sessionID == "" || runID == "" {
		t.Fatalf("expected session_id and run_id in meta payload, got %#v", metaData)
	}

	runEvents, err := aidao.NewAIRunEventDAO(db).ListByRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("load run events: %v", err)
	}
	if len(runEvents) < 2 {
		t.Fatalf("expected at least 2 run events, got %d", len(runEvents))
	}

	recorder2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(recorder2)
	c2.Set("uid", uint64(109))
	c2.Request = httptest.NewRequest(http.MethodPost, "/chat?last_event_id="+runEvents[1].ID, bytes.NewBufferString(`{"message":"hello","session_id":"`+sessionID+`","client_request_id":"req-header-precedence","last_event_id":"`+runEvents[1].ID+`"}`))
	c2.Request.Header.Set("Content-Type", "application/json")
	c2.Request.Header.Set("Last-Event-ID", runEvents[0].ID)
	h.Chat(c2)

	raw := recorder2.Body.String()
	if !strings.Contains(raw, "id: "+runEvents[1].ID) {
		t.Fatalf("expected header cursor to replay event %q, got %q", runEvents[1].ID, raw)
	}
	if strings.Contains(raw, "id: "+runEvents[0].ID) {
		t.Fatalf("expected header cursor to take precedence over query/body, got %q", raw)
	}
}

func TestChatHandler_ReturnsCursorExpiredError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := newAIHandlerTestDB(t)
	h := newAIHandlerTestHarness(db)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("uid", uint64(108))
	c.Request = httptest.NewRequest(http.MethodPost, "/chat?last_event_id=missing-event", bytes.NewBufferString(`{"message":"hello","client_request_id":"req-expired"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	h.Chat(c)

	if !strings.Contains(recorder.Body.String(), "AI_STREAM_CURSOR_EXPIRED") {
		t.Fatalf("expected cursor expired error payload, got %q", recorder.Body.String())
	}
}

func TestChatHandler_ReplaysTerminalRunStateBeforeTerminalEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name            string
		runID           string
		sessionID       string
		userID          uint64
		terminalState   string
		terminalEvent   string
		terminalPayload string
	}{
		{
			name:            "completed before done",
			runID:           "run-terminal-replay-completed",
			sessionID:       "sess-terminal-replay-completed",
			userID:          110,
			terminalState:   "completed",
			terminalEvent:   "done",
			terminalPayload: `{"run_id":"run-terminal-replay-completed","status":"completed"}`,
		},
		{
			name:            "failed before error",
			runID:           "run-terminal-replay-failed",
			sessionID:       "sess-terminal-replay-failed",
			userID:          111,
			terminalState:   "failed",
			terminalEvent:   "error",
			terminalPayload: `{"run_id":"run-terminal-replay-failed","message":"fatal resume event","recoverable":false}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := newAIHandlerTestDB(t)
			h := newAIHandlerTestHarness(db)

			seedSession(t, db, model.AIChatSession{
				ID:        tc.sessionID,
				UserID:    tc.userID,
				Title:     tc.name,
				Scene:     "ai",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
			if err := db.Create(&model.AIChatMessage{
				ID:           tc.runID + "-user",
				SessionID:    tc.sessionID,
				SessionIDNum: 1,
				Role:         "user",
				Content:      "resume",
				Status:       "done",
			}).Error; err != nil {
				t.Fatalf("seed user message: %v", err)
			}
			if err := db.Create(&model.AIChatMessage{
				ID:           tc.runID + "-assistant",
				SessionID:    tc.sessionID,
				SessionIDNum: 2,
				Role:         "assistant",
				Content:      "",
				Status:       "done",
			}).Error; err != nil {
				t.Fatalf("seed assistant message: %v", err)
			}
			if err := aidao.NewAIRunDAO(db).CreateRun(context.Background(), &model.AIRun{
				ID:                 tc.runID,
				SessionID:          tc.sessionID,
				ClientRequestID:    "req-terminal-replay",
				UserMessageID:      tc.runID + "-user",
				AssistantMessageID: tc.runID + "-assistant",
				Status:             tc.terminalState,
				ErrorMessage:       "fatal resume event",
				TraceJSON:          "{}",
			}); err != nil {
				t.Fatalf("seed run: %v", err)
			}
			for _, event := range []model.AIRunEvent{
				{
					ID:          tc.runID + "-evt-1",
					RunID:       tc.runID,
					SessionID:   tc.sessionID,
					Seq:         1,
					EventType:   "run_state",
					PayloadJSON: `{"run_id":"` + tc.runID + `","status":"resuming"}`,
				},
				{
					ID:          tc.runID + "-evt-2",
					RunID:       tc.runID,
					SessionID:   tc.sessionID,
					Seq:         2,
					EventType:   "run_state",
					PayloadJSON: `{"run_id":"` + tc.runID + `","status":"` + tc.terminalState + `"}`,
				},
				{
					ID:          tc.runID + "-evt-3",
					RunID:       tc.runID,
					SessionID:   tc.sessionID,
					Seq:         3,
					EventType:   tc.terminalEvent,
					PayloadJSON: tc.terminalPayload,
				},
			} {
				if err := db.Create(&event).Error; err != nil {
					t.Fatalf("seed run event %s: %v", event.ID, err)
				}
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Set("uid", tc.userID)
			c.Request = httptest.NewRequest(http.MethodPost, "/chat?last_event_id="+tc.runID+"-evt-1", bytes.NewBufferString(`{"message":"resume","session_id":"`+tc.sessionID+`","client_request_id":"req-terminal-replay"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			h.Chat(c)

			raw := recorder.Body.String()
			if !strings.Contains(raw, "id: "+tc.runID+"-evt-2") || !strings.Contains(raw, "id: "+tc.runID+"-evt-3") {
				t.Fatalf("expected replay to include terminal events, got %q", raw)
			}
			runStateIndex := strings.Index(raw, "id: "+tc.runID+"-evt-2")
			terminalIndex := strings.Index(raw, "id: "+tc.runID+"-evt-3")
			if runStateIndex == -1 || terminalIndex == -1 || runStateIndex >= terminalIndex {
				t.Fatalf("expected terminal run_state before %s, got %q", tc.terminalEvent, raw)
			}
		})
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
