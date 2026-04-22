package bootstrap

import (
	"net/http"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestRegisterModulesOwnsAIInteractiveRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	registerAIRoutes(v1, &svc.ServiceContext{})

	routes := engine.Routes()
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = struct{}{}
	}

	expected := []string{
		http.MethodPost + " /api/v1/ai/chat",
		http.MethodPost + " /api/v1/ai/assist/form/stream",
	}

	for _, route := range expected {
		if _, ok := seen[route]; !ok {
			t.Fatalf("expected bootstrap to own %s", route)
		}
	}
}
