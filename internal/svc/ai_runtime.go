package svc

import (
	"context"
	"os"

	ccb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic/metrics"
	aiclient "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/client"
	"gorm.io/gorm"
)

func initAIRuntime(ctx context.Context, db *gorm.DB) {
	if config.IsDevelopment() {
		if err := devops.Init(ctx); err != nil {
			logger.L().Warn("Failed to initialize devops", logger.Error(err))
		}
		initCozeloopCallback(ctx)
	}

	if db != nil {
		initAIMetricsCallback(db)
	}

	if err := aiclient.CheckModelHealth(ctx, db); err != nil {
		logger.L().Warn("Failed to check default AI model health",
			logger.Error(err),
		)
	}
}

// initCozeloopCallback 初始化 CozeLoop Trace 回调（仅开发环境）。
func initCozeloopCallback(ctx context.Context) {
	workspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	apiToken := os.Getenv("COZELOOP_API_TOKEN")
	if workspaceID == "" || apiToken == "" {
		logger.L().Debug("CozeLoop callback skipped: environment variables not set")
		return
	}

	client, err := cozeloop.NewClient()
	if err != nil {
		logger.L().Warn("Failed to create cozeloop client", logger.Error(err))
		return
	}

	handler := ccb.NewLoopHandler(client)
	callbacks.AppendGlobalHandlers(handler)

	logger.L().Info("CozeLoop callback initialized",
		logger.String("workspace_id", workspaceID))
}

// initAIMetricsCallback 初始化 AI 助手指标捕获回调。
func initAIMetricsCallback(db *gorm.DB) {
	handler := metrics.NewMetricsHandler(db)
	callbacks.AppendGlobalHandlers(handler.Build())

	logger.L().Info("AI metrics callback initialized")
}
