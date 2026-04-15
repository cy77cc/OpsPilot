package project

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestRegisterProjectHandlers_ListProjects_RequiresJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	RegisterProjectHandlers(v1, &svc.ServiceContext{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
