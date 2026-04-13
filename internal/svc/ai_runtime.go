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
	aiclient "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/client"
)

func initAIRuntime(ctx context.Context) {
	if config.IsDevelopment() {
		if err := devops.Init(ctx); err != nil {
			logger.L().Warn("Failed to initialize devops", logger.Error(err))
		}
		initCozeloopCallback(ctx)
	}

	if err := aiclient.CheckModelHealth(ctx); err != nil {
		logger.L().Warn("Failed to check AI model health",
			logger.String("base_url", aiBaseURL()),
			logger.String("model", aiModel()),
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
