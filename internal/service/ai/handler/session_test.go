package handler_test

import (
	"net/http"
	"testing"

	aisvc "github.com/cy77cc/OpsPilot/internal/service/ai"
	"github.com/gin-gonic/gin"
)

func TestRegisterAIHandlers_RegistersPhase1Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	aisvc.RegisterAIHandlers(v1, nil)

	want := map[string]string{
		http.MethodPost:   "/api/v1/ai/chat",
		http.MethodGet:    "/api/v1/ai/sessions",
		http.MethodDelete: "/api/v1/ai/sessions/:id",
	}
	extra := map[string]string{
		http.MethodPost + " /api/v1/ai/sessions":        "POST /api/v1/ai/sessions",
		http.MethodGet + " /api/v1/ai/sessions/:id":     "GET /api/v1/ai/sessions/:id",
		http.MethodGet + " /api/v1/ai/runs/:runId":      "GET /api/v1/ai/runs/:runId",
		http.MethodGet + " /api/v1/ai/diagnosis/:reportId": "GET /api/v1/ai/diagnosis/:reportId",
	}

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

	_ = want
	_ = extra
}
