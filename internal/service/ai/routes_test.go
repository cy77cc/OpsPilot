package ai

import (
	"testing"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func collectAIRoutesForTest(t *testing.T) []string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	v1 := engine.Group("/api/v1")
	RegisterAIHandlers(v1, &svc.ServiceContext{})

	routes := engine.Routes()
	result := make([]string, 0, len(routes))
	for _, route := range routes {
		result = append(result, route.Method+" "+route.Path)
	}
	return result
}

func TestRegisterAIHandlers_RegistersChainApprovalDecisionRoute(t *testing.T) {
	routes := collectAIRoutesForTest(t)
	for _, route := range routes {
		if route == "POST /api/v1/ai/chains/:chain_id/approvals/:node_id/decision" {
			return
		}
	}
	t.Fatalf("expected unified chain approval decision route, got: %v", routes)
}
