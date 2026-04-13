package svc

import (
	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
	prominfra "github.com/cy77cc/OpsPilot/internal/infra/prometheus"
)

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

	normalized := cfg.Normalize()
	if normalized.Address == "" {
		logger.L().Warn("Prometheus client initialization skipped: no address configured",
			logger.String("hint", "set PROMETHEUS_ADDRESS or PROMETHEUS_HOST environment variable"))
		return nil
	}

	client, err := prominfra.NewClient(cfg)
	if err != nil {
		logger.L().Warn("Failed to initialize Prometheus client",
			logger.Error(err),
			logger.String("address", normalized.Address))
		return nil
	}

	logger.L().Info("Prometheus client initialized",
		logger.String("address", normalized.Address))
	return client
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
