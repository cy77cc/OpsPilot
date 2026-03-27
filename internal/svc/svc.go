// Package svc 提供服务上下文管理。
//
// 本文件实现 ServiceContext，用于管理应用程序运行时依赖，
// 包括数据库连接、Redis 客户端、K8s 客户端、Casbin 权限执行器等。
package svc

import (
	"context"
	"os"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/cloudwego/eino-ext/devops"
	ccb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/cy77cc/OpsPilot/internal/ai/chatmodel"
	"github.com/cy77cc/OpsPilot/internal/cache"
	casbinadapter "github.com/cy77cc/OpsPilot/internal/component/casbin"
	"github.com/cy77cc/OpsPilot/internal/config"
	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/storage"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ServiceContext 封装应用程序运行时依赖。
type ServiceContext struct {
	DB             *gorm.DB                    // GORM 数据库实例
	Rdb            redis.UniversalClient       // Redis 客户端
	Cache          *expirable.LRU[string, any] // 本地缓存 (LRU)
	CacheFacade    *cache.Facade               // L1-first 缓存门面
	CasbinEnforcer *casbin.Enforcer            // Casbin 权限执行器
	Prometheus     prominfra.Client            // Prometheus HTTP API 客户端
	MetricsPusher  *prominfra.MetricsPusher    // Prometheus 指标推送器
}

// MustNewServiceContext 创建服务上下文，如果失败则 panic。
//
// 初始化流程：
//  1. 初始化 devops 组件
//  2. 检查 AI 模型健康状态
//  3. 创建 K8s 客户端
//  4. 创建数据库和 Redis 连接
//  5. 初始化 Casbin 权限执行器
//  6. 创建本地缓存和缓存门面
//  7. 初始化 Prometheus 客户端
func MustNewServiceContext() *ServiceContext {
	ctx := context.Background()

	// 仅开发环境启用 devops 和 cozeloop 回调
	if config.IsDevelopment() {
		if err := devops.Init(ctx); err != nil {
			logger.L().Warn("Failed to initialize devops", logger.Error(err))
		}
		initCozeloopCallback(ctx)
	}

	if err := chatmodel.CheckModelHealth(ctx); err != nil {
		logger.L().Warn("Failed to check AI model health",
			logger.String("base_url", aiBaseURL()),
			logger.String("model", aiModel()),
			logger.Error(err),
		)
	}

	db := storage.MustNewDB()
	rdb := storage.MustNewRdb()

	// Initialize Casbin
	adapter := casbinadapter.NewAdapter(db)
	enforcer, err := casbin.NewEnforcer("resource/casbin/rbac_model.conf", adapter)
	if err != nil {
		// Try absolute path if relative fails, or panic
		// Assuming running from project root
		logger.L().Error("Failed to initialize Casbin Enforcer", logger.Error(err))
		// panic(err) // Optional: panic if auth is critical
	} else {
		if err := enforcer.LoadPolicy(); err != nil {
			logger.L().Error("Failed to load Casbin policy", logger.Error(err))
		}
	}

	l1 := expirable.NewLRU[string, any](5_000, nil, 24*time.Hour)
	promClient := initPrometheusClient()
	metricsPusher := initMetricsPusher()

	return &ServiceContext{
		DB:             db,
		Rdb:            rdb,
		Cache:          l1,
		CacheFacade:    cache.NewFacade(expirable.NewLRU[string, string](5_000, nil, 24*time.Hour), cache.NewRedisL2(rdb)),
		CasbinEnforcer: enforcer,
		Prometheus:     promClient,
		MetricsPusher:  metricsPusher,
	}
}

// aiBaseURL 返回 AI 模型的基础 URL。
func aiBaseURL() string {
	return config.CFG.LLM.BaseURL
}

// aiModel 返回 AI 模型名称。
func aiModel() string {
	return config.CFG.LLM.Model
}

// initPrometheusClient 初始化 Prometheus 客户端。
func initPrometheusClient() prominfra.Client {
	if !config.CFG.Prometheus.Enable {
		logger.L().Info("Prometheus integration is disabled")
		return nil
	}

	cfg := prominfra.Config{
		Address:       config.CFG.Prometheus.Address,
		Host:          config.CFG.Prometheus.Host,
		Port:          config.CFG.Prometheus.Port,
		Timeout:       config.CFG.Prometheus.Timeout,
		MaxConcurrent: config.CFG.Prometheus.MaxConcurrent,
		RetryCount:    config.CFG.Prometheus.RetryCount,
	}

	// 规范化配置以获取最终地址
	normalized := cfg.Normalize()
	if normalized.Address == "" {
		logger.L().Warn("Prometheus client initialization skipped: no address configured",
			logger.String("hint", "set PROMETHEUS_ADDRESS or PROMETHEUS_HOST environment variable"))
		return nil
	}

	c, err := prominfra.NewClient(cfg)
	if err != nil {
		logger.L().Warn("Failed to initialize Prometheus client",
			logger.Error(err),
			logger.String("address", normalized.Address))
		return nil
	}

	logger.L().Info("Prometheus client initialized",
		logger.String("address", normalized.Address))
	return c
}

// initMetricsPusher 初始化指标推送器。
func initMetricsPusher() *prominfra.MetricsPusher {
	if !config.CFG.Prometheus.Enable {
		return nil
	}
	pushgatewayURL := config.CFG.Prometheus.PushgatewayURL
	if pushgatewayURL == "" {
		logger.L().Warn("Pushgateway URL is not configured, metrics push disabled")
		return nil
	}
	pusher, err := prominfra.NewMetricsPusher(pushgatewayURL)
	if err != nil {
		logger.L().Warn("Failed to initialize MetricsPusher", logger.Error(err))
		return nil
	}
	logger.L().Info("MetricsPusher initialized", logger.String("pushgateway_url", pushgatewayURL))
	return pusher
}

// initCozeloopCallback 初始化 CozeLoop Trace 回调（仅开发环境）。
//
// 通过环境变量配置：
//   - COZELOOP_WORKSPACE_ID: 工作空间 ID
//   - COZELOOP_API_TOKEN: API 令牌
//
// 未配置环境变量时静默跳过。
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
