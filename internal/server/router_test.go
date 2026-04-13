package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestBuildRouterRegistersModuleRoutes(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	var registerCalled bool
	router := buildRouter(nil, func(_ *svc.ServiceContext, engine *gin.Engine) {
		registerCalled = true
		engine.GET("/api/v1/test-only", func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})
	})

	moduleReq := httptest.NewRequest(http.MethodGet, "/api/v1/test-only", nil)
	moduleResp := httptest.NewRecorder()
	router.ServeHTTP(moduleResp, moduleReq)

	if !registerCalled {
		t.Fatal("expected module registration callback to be invoked")
	}
	if moduleResp.Code != http.StatusNoContent {
		t.Fatalf("expected test module route to return 204, got %d", moduleResp.Code)
	}
}
