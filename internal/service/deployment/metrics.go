// Package deployment 提供部署管理服务的指标处理器。
//
// 本文件包含部署指标统计的 HTTP 处理器实现。
package deployment

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// MetricsHandler 是指标统计的 HTTP 处理器。
type MetricsHandler struct {
	svcCtx *svc.ServiceContext
}

// NewMetricsHandler 创建指标处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: MetricsHandler 实例
func NewMetricsHandler(svcCtx *svc.ServiceContext) *MetricsHandler {
	return &MetricsHandler{svcCtx: svcCtx}
}

// MetricsSummary 是部署指标汇总数据。
type MetricsSummary struct {
	TotalReleases   int64                 `json:"total_releases"`   // 总发布数
	SuccessRate     float64               `json:"success_rate"`     // 成功率 (%)
	FailureRate     float64               `json:"failure_rate"`     // 失败率 (%)
	AvgDurationSecs float64               `json:"avg_duration_seconds"` // 平均耗时 (秒)
	ByEnvironment   map[string]EnvMetrics `json:"by_environment"`   // 按环境统计
	ByStatus        map[string]int64      `json:"by_status"`        // 按状态统计
	RecentFailures  int64                 `json:"recent_failures"`  // 最近失败数
	RecentReleases  int64                 `json:"recent_releases"`  // 最近发布数
}

// EnvMetrics 是环境维度的指标数据。
type EnvMetrics struct {
	Total       int64   `json:"total"`        // 总数
	SuccessRate float64 `json:"success_rate"` // 成功率 (%)
}

// MetricsTrend 是指标趋势数据点。
type MetricsTrend struct {
	Date            string  `json:"date"`            // 日期
	DeploymentCount int     `json:"deployment_count"` // 部署次数
	SuccessCount    int     `json:"success_count"`    // 成功次数
	FailureCount    int     `json:"failure_count"`    // 失败次数
	SuccessRate     float64 `json:"success_rate"`     // 成功率 (%)
}

// GetMetricsSummary 获取指标汇总。
//
// @Summary 获取指标汇总
// @Description 获取部署发布的指标汇总数据，包括总数、成功率、失败率等
// @Tags 指标统计
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/metrics/summary [get]
func (h *MetricsHandler) GetMetricsSummary(c *gin.Context) {
	ctx := c.Request.Context()

	summary, err := h.getMetricsSummary(ctx)
	if err != nil {
		httpx.BindErr(c, err)
		return
	}

	httpx.OK(c, summary)
}

// GetMetricsTrends 获取指标趋势。
//
// @Summary 获取指标趋势
// @Description 获取部署发布的指标趋势数据，支持日/周/月维度
// @Tags 指标统计
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param range query string false "时间范围: daily/weekly/monthly"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/metrics/trends [get]
func (h *MetricsHandler) GetMetricsTrends(c *gin.Context) {
	ctx := c.Request.Context()
	timeRange := c.DefaultQuery("range", "daily")

	trends, err := h.getMetricsTrends(ctx, timeRange)
	if err != nil {
		httpx.BindErr(c, err)
		return
	}

	httpx.OK(c, trends)
}

// getMetricsSummary 查询指标汇总数据。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 指标汇总数据
func (h *MetricsHandler) getMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	summary := &MetricsSummary{
		ByEnvironment: make(map[string]EnvMetrics),
		ByStatus:      make(map[string]int64),
	}

	// 总发布数
	if err := h.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentRelease{}).Count(&summary.TotalReleases).Error; err != nil {
		return nil, err
	}

	// 按状态统计
	var statusCounts []struct {
		State string
		Count int64
	}
	if err := h.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentRelease{}).
		Select("state, count(*) as count").
		Group("state").
		Scan(&statusCounts).Error; err != nil {
		return nil, err
	}

	var successCount, failureCount int64
	for _, sc := range statusCounts {
		summary.ByStatus[sc.State] = sc.Count
		if sc.State == "applied" || sc.State == "success" {
			successCount = sc.Count
		}
		if sc.State == "failed" {
			failureCount = sc.Count
		}
	}

	// 成功率
	if summary.TotalReleases > 0 {
		summary.SuccessRate = float64(successCount) / float64(summary.TotalReleases) * 100
		summary.FailureRate = float64(failureCount) / float64(summary.TotalReleases) * 100
	}

	// 按环境统计 (从 targets 关联)
	var envCounts []struct {
		Env   string
		Total int64
	}
	if err := h.svcCtx.DB.WithContext(ctx).
		Table("deploy_releases r").
		Select("t.env, count(*) as total").
		Joins("JOIN deploy_targets t ON r.target_id = t.id").
		Group("t.env").
		Scan(&envCounts).Error; err != nil {
		return nil, err
	}

	for _, ec := range envCounts {
		summary.ByEnvironment[ec.Env] = EnvMetrics{
			Total:       ec.Total,
			SuccessRate: 0, // 可以后续优化
		}
	}

	// 最近7天的统计
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	if err := h.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("created_at > ?", sevenDaysAgo).
		Count(&summary.RecentReleases).Error; err != nil {
		return nil, err
	}

	if err := h.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Where("state = ? AND created_at > ?", "failed", sevenDaysAgo).
		Count(&summary.RecentFailures).Error; err != nil {
		return nil, err
	}

	return summary, nil
}

// getMetricsTrends 查询指标趋势数据。
//
// 参数:
//   - ctx: 上下文
//   - timeRange: 时间范围 (daily/weekly/monthly)
//
// 返回: 趋势数据列表
func (h *MetricsHandler) getMetricsTrends(ctx context.Context, timeRange string) ([]MetricsTrend, error) {
	var trends []MetricsTrend

	// 根据时间范围确定分组格式
	var dateFormat string
	var startDate time.Time
	switch timeRange {
	case "weekly":
		dateFormat = "%Y-%u"
		startDate = time.Now().AddDate(0, 0, -28) // 最近4周
	case "monthly":
		dateFormat = "%Y-%m"
		startDate = time.Now().AddDate(0, -6, 0) // 最近6个月
	default: // daily
		dateFormat = "%Y-%m-%d"
		startDate = time.Now().AddDate(0, 0, -7) // 最近7天
	}

	var results []struct {
		Date   string
		Total  int64
		Failed int64
	}

	// 按日期分组统计
	if err := h.svcCtx.DB.WithContext(ctx).
		Model(&model.DeploymentRelease{}).
		Select("DATE_FORMAT(created_at, '"+dateFormat+"') as date, count(*) as total, sum(case when state = 'failed' then 1 else 0 end) as failed").
		Where("created_at > ?", startDate).
		Group("date").
		Order("date").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, r := range results {
		successCount := r.Total - r.Failed
		var successRate float64
		if r.Total > 0 {
			successRate = float64(successCount) / float64(r.Total) * 100
		}
		trends = append(trends, MetricsTrend{
			Date:            r.Date,
			DeploymentCount: int(r.Total),
			SuccessCount:    int(successCount),
			FailureCount:    int(r.Failed),
			SuccessRate:     successRate,
		})
	}

	return trends, nil
}
