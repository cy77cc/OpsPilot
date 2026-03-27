package handler

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouteContract(t *testing.T) {
	assertAIHandlerRouteContract(t)
}

func TestRegisterAIHandlers_RouteContract(t *testing.T) {
	assertAIHandlerRouteContract(t)
}

func assertAIHandlerRouteContract(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	registerAIHandlersForTest(v1)

	routes := r.Routes()
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = struct{}{}
	}

	expected := []string{
		http.MethodPost + " /api/v1/ai/chat",
		http.MethodGet + " /api/v1/ai/sessions",
		http.MethodPost + " /api/v1/ai/sessions",
		http.MethodGet + " /api/v1/ai/sessions/:id",
		http.MethodDelete + " /api/v1/ai/sessions/:id",
		http.MethodGet + " /api/v1/ai/runs/:runId",
		http.MethodGet + " /api/v1/ai/runs/:runId/projection",
		http.MethodGet + " /api/v1/ai/run-contents/:id",
		http.MethodGet + " /api/v1/ai/diagnosis/:reportId",
		http.MethodGet + " /api/v1/ai/approvals/pending",
		http.MethodGet + " /api/v1/ai/approvals/:id",
		http.MethodPost + " /api/v1/ai/approvals/:id/submit",
		http.MethodPost + " /api/v1/ai/approvals/:id/retry-resume",
	}

	if len(seen) != len(expected) {
		t.Fatalf("expected %d ai routes, got %d: %#v", len(expected), len(seen), routes)
	}
	for _, route := range expected {
		if _, ok := seen[route]; !ok {
			t.Fatalf("missing route %q", route)
		}
	}
}
