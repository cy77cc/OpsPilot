package svc

import (
	"context"
	"path/filepath"
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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ServiceContext struct {
	Clientset      *kubernetes.Clientset       // K8s 客户端
	DB             *gorm.DB                    // GORM 数据库实例
	Rdb            redis.UniversalClient       // Redis 客户端
	Cache          *expirable.LRU[string, any] // 本地缓存 (LRU)
	CacheFacade    *cache.Facade               // L1-first cache facade
	CasbinEnforcer *casbin.Enforcer            // Casbin Enforcer
	Prometheus     prominfra.Client            // Prometheus HTTP API client
}

// MustNewServiceContext 创建服务上下文，如果失败则 panic
func MustNewServiceContext() *ServiceContext {
	ctx := context.Background()
	err := devops.Init(ctx)
	if err != nil {
		logger.L().Warn("Failed to initialize devops", logger.Error(err))
	}
	for _, result := range ai.CheckStartupModelHealth(ctx) {
		fields := []logger.Field{
			logger.String("stage", result.Name),
			logger.String("provider", config.CFG.LLM.Provider),
			logger.String("base_url", aiBaseURL()),
			logger.String("model", firstNonEmpty(result.Model, aiModel())),
		}
		if result.Err != nil {
			fields = append(fields, logger.Error(result.Err))
			logger.L().Warn("AI model startup health check failed", fields...)
			continue
		}
		logger.L().Info("AI model startup health check passed", fields...)
	}

	clientset := MustNewClientset()

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

	return &ServiceContext{
		Clientset:      clientset,
		DB:             db,
		Rdb:            rdb,
		Cache:          l1,
		CacheFacade:    cache.NewFacade(expirable.NewLRU[string, string](5_000, nil, 24*time.Hour), cache.NewRedisL2(rdb)),
		CasbinEnforcer: enforcer,
		Prometheus:     promClient,
	}
}

func aiBaseURL() string {
	return config.CFG.LLM.BaseURL
}

func aiModel() string {
	return config.CFG.LLM.Model
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

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

func MustNewClientset() *kubernetes.Clientset {
	// Try to load kubeconfig from home directory
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// Try to build config from flags (defaulting to kubeconfig path)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		// Fallback to in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			logger.L().Warn("Failed to create K8s config (neither kubeconfig nor in-cluster)", logger.Error(err))
			return nil
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.L().Warn("Failed to create K8s clientset", logger.Error(err))
		return nil
	}

	logger.L().Info("Kubernetes Clientset initialized successfully")
	return clientset
}
