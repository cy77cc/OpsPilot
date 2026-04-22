// Package svc 提供服务上下文管理。
//
// 本文件定义 ServiceContext 并协调应用运行时依赖的初始化。
package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/cy77cc/OpsPilot/internal/core/cache"
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/storage"
	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
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

// NewServiceContext 创建服务上下文，并向上返回初始化错误。
func NewServiceContext(ctx context.Context) (*ServiceContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	db, err := storage.NewDB()
	if err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}
	initAIRuntime(ctx, db)

	rdb, err := storage.NewRdb()
	if err != nil {
		if sqlDB, sqlErr := db.DB(); sqlErr == nil {
			_ = sqlDB.Close()
		}
		return nil, fmt.Errorf("init redis: %w", err)
	}
	l1 := expirable.NewLRU[string, any](5_000, nil, 24*time.Hour)

	return &ServiceContext{
		DB:             db,
		Rdb:            rdb,
		Cache:          l1,
		CacheFacade:    cache.NewFacade(expirable.NewLRU[string, string](5_000, nil, 24*time.Hour), cache.NewRedisL2(rdb)),
		CasbinEnforcer: newCasbinEnforcer(db),
		Prometheus:     initPrometheusClient(),
		MetricsPusher:  initMetricsPusher(),
	}, nil
}

func aiBaseURL() string {
	return config.CFG.LLM.BaseURL
}

func aiModel() string {
	return config.CFG.LLM.Model
}
