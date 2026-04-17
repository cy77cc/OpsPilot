package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type stubCommandHandler struct {
	called bool
}

func (s *stubCommandHandler) Handle(*ChatRequest) error {
	s.called = true
	return nil
}

func TestChatHandlerDelegatesToCommandHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &stubCommandHandler{}
	h := NewChatHandler(stub)

	r := gin.New()
	r.POST("/ai/chat", h.HandleChat)

	req := httptest.NewRequest(http.MethodPost, "/ai/chat", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if !stub.called {
		t.Fatal("expected command handler to be called")
	}
}
