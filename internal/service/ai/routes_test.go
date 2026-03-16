package ai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

func TestRegisterAIHandlersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterAIHandlers(v1, &svc.ServiceContext{})

	type route struct {
		name   string
		method string
		path   string
		body   string
	}

	routes := []route{
		{name: "Chat", method: http.MethodPost, path: "/api/v1/ai/chat", body: `{"message":"hello"}`},
		{name: "ListSessions", method: http.MethodGet, path: "/api/v1/ai/sessions"},
		{name: "CreateSession", method: http.MethodPost, path: "/api/v1/ai/sessions", body: `{"scene":"cluster"}`},
		{name: "GetSession", method: http.MethodGet, path: "/api/v1/ai/sessions/session-123"},
		{name: "DeleteSession", method: http.MethodDelete, path: "/api/v1/ai/sessions/session-123"},
		{name: "GetRun", method: http.MethodGet, path: "/api/v1/ai/runs/run-987"},
		{name: "GetDiagnosis", method: http.MethodGet, path: "/api/v1/ai/diagnosis/report-123"},
	}

	for _, routeCase := range routes {
		routeCase := routeCase
		t.Run(routeCase.name, func(t *testing.T) {
			var bodyReader io.Reader
			if routeCase.body != "" {
				bodyReader = strings.NewReader(routeCase.body)
			}

			req := httptest.NewRequest(routeCase.method, routeCase.path, bodyReader)
			if routeCase.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s returned 404", routeCase.name)
			}
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("%s returned %d instead of 401", routeCase.name, rec.Code)
			}
		})
	}
}
