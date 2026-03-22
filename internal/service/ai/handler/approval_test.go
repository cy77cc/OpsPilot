package handler_test

import (
	"net/http"
	"testing"

	aiService "github.com/cy77cc/OpsPilot/internal/service/ai"
	"github.com/gin-gonic/gin"
)

func TestSubmitApprovalRouteContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	v1 := r.Group("/api/v1")
	aiService.RegisterAIHandlers(v1, nil)

	routes := r.Routes()
	seen := make(map[string]bool, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = true
	}

	if !seen[http.MethodPost+" /api/v1/ai/approvals/:id/submit"] {
		t.Fatalf("expected submit route to be registered")
	}
	if seen[http.MethodPost+" /api/v1/ai/approvals/:id/confirm"] {
		t.Fatalf("legacy confirm route must not be registered")
	}
	if seen[http.MethodPost+" /api/v1/ai/chains/:chainId/approvals/:nodeId/decision"] {
		t.Fatalf("legacy decision route must not be registered")
	}
	if seen[http.MethodPost+" /api/v1/ai/approvals/:id/resume"] {
		t.Fatalf("legacy resume route must not be registered")
	}
}
