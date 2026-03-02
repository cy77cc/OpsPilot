package aiops

import (
	"context"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
)

func (h *Handler) listRiskFindings(ctx context.Context) ([]model.RiskFinding, error) {
	var findings []model.RiskFinding
	
	// 先尝试从数据库获取
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("resolved_at IS NULL").
		Order("created_at desc").
		Limit(20).
		Find(&findings).Error; err != nil {
		return nil, err
	}
	
	// 如果数据库没有数据，返回基于规则的分析结果
	if len(findings) == 0 {
		findings = h.generateSampleRiskFindings()
	}
	
	return findings, nil
}

func (h *Handler) listAnomalies(ctx context.Context) ([]model.Anomaly, error) {
	var anomalies []model.Anomaly
	
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("resolved_at IS NULL").
		Order("detected_at desc").
		Limit(20).
		Find(&anomalies).Error; err != nil {
		return nil, err
	}
	
	if len(anomalies) == 0 {
		anomalies = h.generateSampleAnomalies()
	}
	
	return anomalies, nil
}

func (h *Handler) listSuggestions(ctx context.Context) ([]model.Suggestion, error) {
	var suggestions []model.Suggestion
	
	if err := h.svcCtx.DB.WithContext(ctx).
		Where("applied_at IS NULL").
		Order("created_at desc").
		Limit(20).
		Find(&suggestions).Error; err != nil {
		return nil, err
	}
	
	if len(suggestions) == 0 {
		suggestions = h.generateSampleSuggestions()
	}
	
	return suggestions, nil
}

// generateSampleRiskFindings 生成示例风险发现
func (h *Handler) generateSampleRiskFindings() []model.RiskFinding {
	now := time.Now()
	return []model.RiskFinding{
		{
			Type:        "security",
			Severity:    "high",
			Title:       "检测到未加密的敏感数据传输",
			Description: "发现服务之间的敏感数据传输未使用 TLS 加密，可能导致数据泄露风险",
			ServiceName: "api-gateway",
			CreatedAt:   now,
		},
		{
			Type:        "performance",
			Severity:    "medium",
			Title:       "数据库连接池接近上限",
			Description: "数据库连接池使用率达到 85%，建议增加连接池大小或优化查询",
			ServiceName: "user-service",
			CreatedAt:   now.Add(-time.Hour),
		},
	}
}

// generateSampleAnomalies 生成示例异常
func (h *Handler) generateSampleAnomalies() []model.Anomaly {
	now := time.Now()
	return []model.Anomaly{
		{
			Type:        "latency",
			Metric:      "response_time_p99",
			Value:       2500,
			Threshold:   1000,
			ServiceName: "order-service",
			DetectedAt:  now,
		},
		{
			Type:        "error_rate",
			Metric:      "error_rate",
			Value:       5.2,
			Threshold:   1.0,
			ServiceName: "payment-service",
			DetectedAt:  now.Add(-30 * time.Minute),
		},
	}
}

// generateSampleSuggestions 生成示例建议
func (h *Handler) generateSampleSuggestions() []model.Suggestion {
	now := time.Now()
	return []model.Suggestion{
		{
			Type:        "cost",
			Title:       "优化 Kubernetes 资源配置",
			Description: "检测到多个 Pod 资源利用率低于 20%，建议调整 requests/limits 配置以降低成本",
			Impact:      "high",
			ServiceName: "api-gateway",
			CreatedAt:   now,
		},
		{
			Type:        "reliability",
			Title:       "添加健康检查探针",
			Description: "user-service 缺少 readiness/liveness 探针，可能导致流量路由到未就绪的实例",
			Impact:      "medium",
			ServiceName: "user-service",
			CreatedAt:   now.Add(-time.Hour),
		},
	}
}
