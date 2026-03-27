package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildAgentHandlers_WiresApprovalOrchestratorFromServiceContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:agent-handlers-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	ctx := runtimectx.WithServices(context.Background(), &svc.ServiceContext{DB: db})
	handlers, err := BuildAgentHandlers(ctx, []tool.BaseTool{})
	if err != nil {
		t.Fatalf("build handlers: %v", err)
	}
	if len(handlers) == 0 {
		t.Fatal("expected non-empty handlers")
	}

	approvalMw, ok := handlers[0].(*approvalMiddleware)
	if !ok {
		t.Fatalf("expected first handler to be approval middleware, got %T", handlers[0])
	}
	if approvalMw.config == nil || approvalMw.config.Orchestrator == nil {
		t.Fatal("expected approval middleware to wire db-backed orchestrator")
	}
}

func TestBuildAgentHandlers_WithoutServiceContextKeepsFallbackApprovalMiddleware(t *testing.T) {
	handlers, err := BuildAgentHandlers(context.Background(), []tool.BaseTool{})
	if err != nil {
		t.Fatalf("build handlers: %v", err)
	}
	if len(handlers) == 0 {
		t.Fatal("expected non-empty handlers")
	}

	approvalMw, ok := handlers[0].(*approvalMiddleware)
	if !ok {
		t.Fatalf("expected first handler to be approval middleware, got %T", handlers[0])
	}
	if approvalMw.config == nil {
		t.Fatal("expected approval middleware config to be initialized")
	}
	if approvalMw.config.Orchestrator != nil {
		t.Fatal("expected fallback middleware without orchestrator when service context missing")
	}
}
