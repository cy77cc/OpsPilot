package middleware

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
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
	handlers, err := BuildAgentHandlers(ctx, "kubernetes", []tool.BaseTool{})
	if err != nil {
		t.Fatalf("build handlers: %v", err)
	}
	if len(handlers) == 0 {
		t.Fatal("expected non-empty handlers")
	}

	// 验证第一个中间件是场景路由器
	_, ok := handlers[0].(*SceneRouterMiddleware)
	if !ok {
		t.Fatalf("expected first handler to be scene router middleware, got %T", handlers[0])
	}

	// 验证第二个中间件是审批中间件
	approvalMw, ok := handlers[1].(*approvalMiddleware)
	if !ok {
		t.Fatalf("expected second handler to be approval middleware, got %T", handlers[1])
	}
	if approvalMw.config == nil || approvalMw.config.Orchestrator == nil {
		t.Fatal("expected approval middleware to wire db-backed orchestrator")
	}
}

func TestBuildAgentHandlers_WithoutServiceContextKeepsFallbackApprovalMiddleware(t *testing.T) {
	handlers, err := BuildAgentHandlers(context.Background(), "ai", []tool.BaseTool{})
	if err != nil {
		t.Fatalf("build handlers: %v", err)
	}
	if len(handlers) == 0 {
		t.Fatal("expected non-empty handlers")
	}

	// 验证场景路由器存在
	_, ok := handlers[0].(*SceneRouterMiddleware)
	if !ok {
		t.Fatalf("expected first handler to be scene router middleware, got %T", handlers[0])
	}

	// 验证审批中间件存在
	approvalMw, ok := handlers[1].(*approvalMiddleware)
	if !ok {
		t.Fatalf("expected second handler to be approval middleware, got %T", handlers[1])
	}
	if approvalMw.config == nil {
		t.Fatal("expected approval middleware config to be initialized")
	}
	if approvalMw.config.Orchestrator != nil {
		t.Fatal("expected fallback middleware without orchestrator when service context missing")
	}
}

func TestBuildAgentHandlers_SceneRouterAllowsProvidedToolsInAIScene(t *testing.T) {
	ctx := runtimectx.WithAIMetadata(context.Background(), runtimectx.AIMetadata{Scene: "ai"})
	cpuTool, err := einoutils.InferTool("os_get_cpu_mem", "mock", func(ctx context.Context, input map[string]any) (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("build mock tool: %v", err)
	}

	handlers, err := BuildAgentHandlers(ctx, "ai", []tool.BaseTool{cpuTool})
	if err != nil {
		t.Fatalf("build handlers: %v", err)
	}
	router, ok := handlers[0].(*SceneRouterMiddleware)
	if !ok {
		t.Fatalf("expected first handler to be scene router middleware, got %T", handlers[0])
	}

	endpoint, err := router.WrapInvokableToolCall(context.Background(), func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		return "executed", nil
	}, &adk.ToolContext{Name: "os_get_cpu_mem"})
	if err != nil {
		t.Fatalf("wrap tool call: %v", err)
	}
	out, callErr := endpoint(context.Background(), "{}")
	if callErr != nil {
		t.Fatalf("expected os_get_cpu_mem to be allowed in ai scene when provided by toolset, got err=%v", callErr)
	}
	if out != "executed" {
		t.Fatalf("unexpected tool output: %q", out)
	}
}
