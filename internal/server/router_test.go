package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
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

func TestStart_ReturnsServiceContextInitError(t *testing.T) {
	t.Parallel()

	previousCfg := config.CFG
	config.CFG.Server.Host = "127.0.0.1"
	config.CFG.Server.Port = 0
	t.Cleanup(func() {
		config.CFG = previousCfg
	})

	previousFactory := newServiceContext
	newServiceContext = func(context.Context) (*svc.ServiceContext, error) {
		return nil, errors.New("service context init failed")
	}
	t.Cleanup(func() {
		newServiceContext = previousFactory
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := Start(ctx)
	if err == nil || err.Error() != "service context init failed" {
		t.Fatalf("expected Start to return service context error, got %v", err)
	}
}

func TestNewShutdownContext_DetachesCancellationButKeepsValues(t *testing.T) {
	t.Parallel()

	type shutdownContextKey string

	parent := context.WithValue(context.Background(), shutdownContextKey("trace_id"), "trace-123")
	parent, cancelParent := context.WithCancel(parent)
	cancelParent()

	shutdownCtx, cancelShutdown := newShutdownContext(parent)
	defer cancelShutdown()

	select {
	case <-shutdownCtx.Done():
		t.Fatal("expected shutdown context to remain active even if parent is already cancelled")
	default:
	}

	if got := shutdownCtx.Value(shutdownContextKey("trace_id")); got != "trace-123" {
		t.Fatalf("expected shutdown context to preserve parent values, got %#v", got)
	}

	deadline, ok := shutdownCtx.Deadline()
	if !ok {
		t.Fatal("expected shutdown context to set a deadline")
	}
	if time.Until(deadline) <= 0 {
		t.Fatalf("expected shutdown deadline in the future, got %v", deadline)
	}
}
