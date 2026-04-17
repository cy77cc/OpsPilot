package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/app/command"
	aidao "github.com/cy77cc/OpsPilot/internal/modules/ai/dao/run"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/gin-gonic/gin"
)

type stubCommandHandler struct {
	req    *command.ChatRequest
	emitFn logic.EventEmitter
	err    error
}

func (s *stubCommandHandler) Handle(_ context.Context, req *command.ChatRequest, emit logic.EventEmitter) error {
	s.req = req
	s.emitFn = emit
	if emit != nil {
		emit("status", gin.H{"ok": true})
	}
	return s.err
}

func TestChatHandlerDelegatesToCommandHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubCommandHandler{}
	h := NewChatHandler(stub)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uid", uint64(7))
		c.Next()
	})
	r.POST("/ai/chat", h.HandleChat)

	req := httptest.NewRequest(http.MethodPost, "/ai/chat", strings.NewReader(`{"message":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if stub.req == nil {
		t.Fatal("expected command handler to be called")
	}
}

func TestChatHandler_PreservesSSEHeadersAndRequestMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubCommandHandler{}
	h := NewChatHandler(stub)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uid", uint64(42))
		c.Next()
	})
	r.POST("/ai/chat", h.HandleChat)

	req := httptest.NewRequest(http.MethodPost, "/ai/chat?last_event_id=query-event", strings.NewReader(`{"message":"hi","client_request_id":"req-1","last_event_id":"body-event","scene":"ops","context":{"team":"platform"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Last-Event-ID", "header-event")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", got)
	}
	if stub.req == nil {
		t.Fatal("expected command handler to receive a request")
	}
	if stub.req.LastEventID != "header-event" {
		t.Fatalf("expected header precedence for last_event_id, got %q", stub.req.LastEventID)
	}
	if stub.req.UserID != 42 {
		t.Fatalf("expected uid 42, got %d", stub.req.UserID)
	}
	if stub.req.Message != "hi" || stub.req.ClientRequestID != "req-1" || stub.req.Scene != "ops" {
		t.Fatalf("unexpected command mapping: %#v", stub.req)
	}
	if stub.req.Context["team"] != "platform" {
		t.Fatalf("expected context to be preserved, got %#v", stub.req.Context)
	}
	if !strings.Contains(rec.Body.String(), "event: status") {
		t.Fatalf("expected streamed status event, got %q", rec.Body.String())
	}
}

func TestChatHandler_WritesCursorExpiredErrorEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(&stubCommandHandler{err: aidao.ErrRunEventCursorExpired})

	r := gin.New()
	r.POST("/ai/chat", h.HandleChat)

	req := httptest.NewRequest(http.MethodPost, "/ai/chat", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "AI_STREAM_CURSOR_EXPIRED") {
		t.Fatalf("expected cursor-expired SSE event, got %q", rec.Body.String())
	}
}

func TestChatHandler_WritesNonCursorErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewChatHandler(&stubCommandHandler{err: errors.New("boom")})

	r := gin.New()
	r.POST("/ai/chat", h.HandleChat)

	req := httptest.NewRequest(http.MethodPost, "/ai/chat", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("expected non-cursor error to be surfaced, got %q", rec.Body.String())
	}
}
