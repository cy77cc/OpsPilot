// Package svc 提供服务上下文管理。
//
// 本文件实现 ServiceContext，用于管理应用程序运行时依赖，
// 包括数据库连接、Redis 客户端、K8s 客户端、Casbin 权限执行器等。
package svc

import (
	"context"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cy77cc/OpsPilot/internal/ai"
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
	err := devops.Init(ctx)
	if err != nil {
		logger.L().Warn("Failed to initialize devops", logger.Error(err))
	}

	if err := ai.CheckModelHealth(ctx); err != nil {
		logger.L().Warn("Failed to check AI model health",
			logger.String("base_url", aiBaseURL()),
			logger.String("model", aiModel()),
			logger.Error(err),
		)
	}

	db := storage.MustNewDB()
	rdb := storage.MustNewRdb()
	if err != nil {
		logger.L().Warn("Failed to initialize AI PlatformRunner",
			logger.String("base_url", aiBaseURL()),
			logger.String("model", aiModel()),
			logger.Error(err),
		)
	}

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
		return nil
	}
	c, err := prominfra.NewClient(prominfra.Config{
		Address:       config.CFG.Prometheus.Address,
		Host:          config.CFG.Prometheus.Host,
		Port:          config.CFG.Prometheus.Port,
		Timeout:       config.CFG.Prometheus.Timeout,
		MaxConcurrent: config.CFG.Prometheus.MaxConcurrent,
		RetryCount:    config.CFG.Prometheus.RetryCount,
	})
	if err != nil {
		logger.L().Warn("Failed to initialize Prometheus client", logger.Error(err))
		return nil
	}
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
