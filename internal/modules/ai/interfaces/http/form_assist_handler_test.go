package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/gin-gonic/gin"
)

type stubFormAssistStreamer struct {
	input logic.FormAssistInput
	err   error
}

func (s *stubFormAssistStreamer) StreamAssist(ctx context.Context, input logic.FormAssistInput, emit logic.EventEmitter) error {
	s.input = input
	if emit != nil {
		emit("suggestion", gin.H{"text": "suggested value"})
	}
	return s.err
}

func TestFormAssistHandler_HandleAssist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubFormAssistStreamer{}
	h := NewFormAssistHandler(stub)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("uid", uint64(123))
		c.Next()
	})
	r.POST("/ai/assist/form/stream", h.HandleAssist)

	body := `{"scene":"test","user_prompt":"help me","field_meta":{"key":"name","label":"Name","purpose":"user name"},"form_context":{"age":30}}`
	req := httptest.NewRequest(http.MethodPost, "/ai/assist/form/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", got)
	}

	if stub.input.Scene != "test" {
		t.Errorf("expected scene test, got %q", stub.input.Scene)
	}
	if stub.input.UserID != 123 {
		t.Errorf("expected user id 123, got %d", stub.input.UserID)
	}
	if stub.input.FieldMeta.Key != "name" {
		t.Errorf("expected field key name, got %q", stub.input.FieldMeta.Key)
	}

	if !strings.Contains(rec.Body.String(), "event: suggestion") {
		t.Errorf("expected suggestion event in body, got %q", rec.Body.String())
	}
}

func TestFormAssistHandler_HandleAssist_NotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewFormAssistHandler(nil)

	r := gin.New()
	r.POST("/ai/assist/form/stream", h.HandleAssist)

	body := `{"scene":"test","user_prompt":"help me"}`
	req := httptest.NewRequest(http.MethodPost, "/ai/assist/form/stream", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "event: error") {
		t.Errorf("expected error event, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "An internal error occurred") {
		t.Errorf("expected internal error message, got %q", rec.Body.String())
	}
}
