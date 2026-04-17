package bootstrap

import (
	"net/http"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestRegisterModulesOwnsAIChatRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	v1 := engine.Group("/api/v1")
	registerAIChatRoute(v1, &svc.ServiceContext{})

	routes := engine.Routes()
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = struct{}{}
	}

	if _, ok := seen[http.MethodPost+" /api/v1/ai/chat"]; !ok {
		t.Fatal("expected bootstrap to own POST /api/v1/ai/chat")
	}
}
