// Package dao 提供数据访问对象。
package dao

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"gorm.io/gorm"
)

// UsageLogDAO 是 AI 使用统计的数据访问对象。
type UsageLogDAO struct {
	db *gorm.DB
}

// NewUsageLogDAO 创建 UsageLogDAO 实例。
func NewUsageLogDAO(db *gorm.DB) *UsageLogDAO {
	return &UsageLogDAO{db: db}
}

// Create 创建使用统计记录。
func (d *UsageLogDAO) Create(ctx context.Context, log *model.AIUsageLog) error {
	return d.db.WithContext(ctx).Create(log).Error
}

// GetBySessionID 按 SessionID 查询使用统计列表。
func (d *UsageLogDAO) GetBySessionID(ctx context.Context, sessionID string) ([]model.AIUsageLog, error) {
	var logs []model.AIUsageLog
	err := d.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// StatsQuery 统计查询参数。
type StatsQuery struct {
	StartDate time.Time
	EndDate   time.Time
	Scene     string
	UserID    uint64
}

// StatsResult 统计结果。
type StatsResult struct {
	TotalRequests         int64   `json:"total_requests"`
	TotalTokens           int64   `json:"total_tokens"`
	TotalPromptTokens     int64   `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	TotalCostUSD          float64 `json:"total_cost_usd"`
	AvgFirstTokenMs       float64 `json:"avg_first_token_ms"`
	AvgTokensPerSecond    float64 `json:"avg_tokens_per_second"`
	ApprovalRate          float64 `json:"approval_rate"`
	ApprovalPassRate      float64 `json:"approval_pass_rate"`
	ToolErrorRate         float64 `json:"tool_error_rate"`
}

// GetStats 获取统计数据。
func (d *UsageLogDAO) GetStats(ctx context.Context, query StatsQuery) (*StatsResult, error) {
	var result StatsResult
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)

	if query.Scene != "" {
		q = q.Where("scene = ?", query.Scene)
	}
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}

	if err := q.Select(
		"COUNT(*) as total_requests",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens",
		"COALESCE(SUM(completion_tokens), 0) as total_completion_tokens",
		"COALESCE(SUM(estimated_cost_usd), 0) as total_cost_usd",
		"COALESCE(AVG(first_token_ms), 0) as avg_first_token_ms",
		"COALESCE(AVG(tokens_per_second), 0) as avg_tokens_per_second",
	).Scan(&result).Error; err != nil {
		return nil, err
	}

	// 审批统计
	var approvalTotal, approvalPassed int64
	approvalQ := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)
	approvalQ.Where("approval_count > 0").Count(&approvalTotal)
	approvalQ.Where("approval_status = ?", "approved").Count(&approvalPassed)

	if result.TotalRequests > 0 {
		result.ApprovalRate = float64(approvalTotal) / float64(result.TotalRequests)
	}
	if approvalTotal > 0 {
		result.ApprovalPassRate = float64(approvalPassed) / float64(approvalTotal)
	}

	// 工具错误率
	var toolCalls, toolErrors int64
	d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate).
		Select("COALESCE(SUM(tool_call_count), 0)").Scan(&toolCalls)
	d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate).
		Select("COALESCE(SUM(tool_error_count), 0)").Scan(&toolErrors)
	if toolCalls > 0 {
		result.ToolErrorRate = float64(toolErrors) / float64(toolCalls)
	}

	return &result, nil
}

// SceneStats 场景统计。
type SceneStats struct {
	Scene  string `json:"scene"`
	Count  int64  `json:"count"`
	Tokens int64  `json:"tokens"`
}

// GetByScene 按场景分组统计。
func (d *UsageLogDAO) GetByScene(ctx context.Context, query StatsQuery) ([]SceneStats, error) {
	var result []SceneStats
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}
	err := q.Select("scene as scene, COUNT(*) as count, COALESCE(SUM(total_tokens), 0) as tokens").
		Group("scene").
		Order("count DESC").
		Find(&result).Error
	return result, err
}

// DateStats 日期统计。
type DateStats struct {
	Date     string `json:"date"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// GetByDate 按日期分组统计。
func (d *UsageLogDAO) GetByDate(ctx context.Context, query StatsQuery) ([]DateStats, error) {
	var result []DateStats
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)
	if query.Scene != "" {
		q = q.Where("scene = ?", query.Scene)
	}
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}
	err := q.Select(
		"DATE(created_at) as date, COUNT(*) as requests, COALESCE(SUM(total_tokens), 0) as tokens",
	).Group("DATE(created_at)").
		Order("date ASC").
		Find(&result).Error
	return result, err
}

// ListQuery 列表查询参数。
type ListQuery struct {
	StartDate time.Time
	EndDate   time.Time
	Scene     string
	Status    string
	UserID    uint64
	Page      int
	PageSize  int
}

// ListResult 列表结果。
type ListResult struct {
	Total int64             `json:"total"`
	Items []model.AIUsageLog `json:"items"`
}

// List 分页查询使用日志列表。
func (d *UsageLogDAO) List(ctx context.Context, query ListQuery) (*ListResult, error) {
	var result ListResult
	q := d.db.WithContext(ctx).Model(&model.AIUsageLog{}).
		Where("created_at >= ? AND created_at < ?", query.StartDate, query.EndDate)

	if query.Scene != "" {
		q = q.Where("scene = ?", query.Scene)
	}
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	if query.UserID > 0 {
		q = q.Where("user_id = ?", query.UserID)
	}

	if err := q.Count(&result.Total).Error; err != nil {
		return nil, err
	}

	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	offset := (query.Page - 1) * query.PageSize

	err := q.Order("created_at DESC").
		Limit(query.PageSize).
		Offset(offset).
		Find(&result.Items).Error
	return &result, err
}
