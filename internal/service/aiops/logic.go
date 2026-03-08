package aiops

import (
	"context"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
)

// RuleEngine provides basic AIOps analysis rules
type RuleEngine struct {
	handler *Handler
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine(h *Handler) *RuleEngine {
	return &RuleEngine{handler: h}
}

// AnalyzeDeployments analyzes deployment data and generates insights
func (r *RuleEngine) AnalyzeDeployments(ctx context.Context) error {
	// Get deployment statistics
	var releaseCount int64
	var failedCount int64

	r.handler.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentRelease{}).Count(&releaseCount)
	r.handler.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentRelease{}).
		Where("state = ?", "failed").Count(&failedCount)

	// Generate risk findings based on failure rate
	if releaseCount > 0 {
		failureRate := float64(failedCount) / float64(releaseCount) * 100
		if failureRate > 10 {
			r.createRiskFinding(ctx, "deployment", "high", "部署失败率过高",
				"当前部署失败率为 %.1f%%，建议检查部署配置和目标环境状态", failureRate)
		}
	}

	// Analyze releases without approvals
	var unapprovedReleases int64
	r.handler.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentRelease{}).
		Where("state = ? AND approval_id IS NULL", "pending").Count(&unapprovedReleases)
	if unapprovedReleases > 5 {
		r.createSuggestion(ctx, "process", "审批流程优化",
			"发现 %d 个待审批的部署，建议优化审批流程以加快部署速度", "medium", unapprovedReleases)
	}

	return nil
}

// createRiskFinding creates a risk finding record
func (r *RuleEngine) createRiskFinding(ctx context.Context, riskType, severity, title, description string, args ...any) error {
	findings := model.RiskFinding{
		Type:        riskType,
		Severity:    severity,
		Title:       title,
		Description: description,
		CreatedAt:   time.Now(),
	}
	return r.handler.svcCtx.DB.WithContext(ctx).Create(&findings).Error
}

// createSuggestion creates a suggestion record
func (r *RuleEngine) createSuggestion(ctx context.Context, suggType, title, description, impact string, args ...any) error {
	suggestion := model.Suggestion{
		Type:        suggType,
		Title:       title,
		Description: description,
		Impact:      impact,
		CreatedAt:   time.Now(),
	}
	return r.handler.svcCtx.DB.WithContext(ctx).Create(&suggestion).Error
}

// CreateAnomaly creates an anomaly record
func (r *RuleEngine) CreateAnomaly(ctx context.Context, anomalyType, metric string, value, threshold float64, serviceID uint, serviceName string) error {
	anomaly := model.Anomaly{
		Type:        anomalyType,
		Metric:      metric,
		Value:       value,
		Threshold:   threshold,
		ServiceID:   serviceID,
		ServiceName: serviceName,
		DetectedAt:  time.Now(),
	}
	return r.handler.svcCtx.DB.WithContext(ctx).Create(&anomaly).Error
}

// GenerateSampleData generates sample AIOps data for demonstration
// This is useful when there's no real data yet
func (h *Handler) GenerateSampleData(ctx context.Context) error {
	now := time.Now()

	// Sample risk findings
	riskFindings := []model.RiskFinding{
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
		{
			Type:        "reliability",
			Severity:    "low",
			Title:       "服务健康检查间隔过长",
			Description: "检测到健康检查间隔设置为 30 秒，可能影响故障发现速度",
			ServiceName: "order-service",
			CreatedAt:   now.Add(-2 * time.Hour),
		},
	}

	// Sample anomalies
	anomalies := []model.Anomaly{
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
		{
			Type:        "cpu_usage",
			Metric:      "cpu_utilization",
			Value:       92.5,
			Threshold:   80.0,
			ServiceName: "api-gateway",
			DetectedAt:  now.Add(-15 * time.Minute),
		},
	}

	// Sample suggestions
	suggestions := []model.Suggestion{
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
		{
			Type:        "performance",
			Title:       "启用连接池复用",
			Description: "检测到频繁创建数据库连接，建议启用连接池复用以提升性能",
			Impact:      "medium",
			ServiceName: "order-service",
			CreatedAt:   now.Add(-2 * time.Hour),
		},
	}

	// Insert sample data
	for _, rf := range riskFindings {
		h.svcCtx.DB.WithContext(ctx).FirstOrCreate(&rf, model.RiskFinding{
			Title: rf.Title,
		})
	}
	for _, a := range anomalies {
		h.svcCtx.DB.WithContext(ctx).FirstOrCreate(&a, model.Anomaly{
			Metric:    a.Metric,
			Value:     a.Value,
			Threshold: a.Threshold,
		})
	}
	for _, s := range suggestions {
		h.svcCtx.DB.WithContext(ctx).FirstOrCreate(&s, model.Suggestion{
			Title: s.Title,
		})
	}

	return nil
}
