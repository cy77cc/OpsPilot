// Package ai 提供 AI 编排服务的 HTTP 接口。
package ai

import (
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// GetUsageStats 获取使用统计概览。
func (h *HTTPHandler) GetUsageStats(c *gin.Context) {
	if h.usageLogDAO == nil {
		httpx.Fail(c, xcode.ServerError, "usage statistics not available")
		return
	}

	startDate, endDate := parseDateRange(c)
	userID := httpx.UIDFromCtx(c)
	isAdmin := httpx.IsAdmin(h.svcCtx.DB, userID)

	query := dao.StatsQuery{
		StartDate: startDate,
		EndDate:   endDate,
		Scene:     c.Query("scene"),
	}

	if !isAdmin {
		query.UserID = userID
	}

	stats, err := h.usageLogDAO.GetStats(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	byScene, err := h.usageLogDAO.GetByScene(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	byDate, err := h.usageLogDAO.GetByDate(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, gin.H{
		"total_requests":           stats.TotalRequests,
		"total_tokens":             stats.TotalTokens,
		"total_prompt_tokens":      stats.TotalPromptTokens,
		"total_completion_tokens":  stats.TotalCompletionTokens,
		"total_cost_usd":           stats.TotalCostUSD,
		"avg_first_token_ms":       stats.AvgFirstTokenMs,
		"avg_tokens_per_second":    stats.AvgTokensPerSecond,
		"approval_rate":            stats.ApprovalRate,
		"approval_pass_rate":       stats.ApprovalPassRate,
		"tool_error_rate":          stats.ToolErrorRate,
		"by_scene":                 byScene,
		"by_date":                  byDate,
	})
}

// GetUsageLogs 获取使用日志列表。
func (h *HTTPHandler) GetUsageLogs(c *gin.Context) {
	if h.usageLogDAO == nil {
		httpx.Fail(c, xcode.ServerError, "usage statistics not available")
		return
	}

	startDate, endDate := parseDateRange(c)
	userID := httpx.UIDFromCtx(c)
	isAdmin := httpx.IsAdmin(h.svcCtx.DB, userID)

	query := dao.ListQuery{
		StartDate: startDate,
		EndDate:   endDate,
		Scene:     c.Query("scene"),
		Status:    c.Query("status"),
		Page:      parseInt(c.Query("page"), 1),
		PageSize:  parseInt(c.Query("page_size"), 20),
	}

	if !isAdmin {
		query.UserID = userID
	}

	result, err := h.usageLogDAO.List(c.Request.Context(), query)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, result)
}

// parseDateRange 解析日期范围参数。
func parseDateRange(c *gin.Context) (start, end time.Time) {
	now := time.Now()

	startDate := c.Query("start_date")
	if startDate != "" {
		start, _ = time.Parse("2006-01-02", startDate)
	} else {
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	}

	endDate := c.Query("end_date")
	if endDate != "" {
		end, _ = time.Parse("2006-01-02", endDate)
	} else {
		end = start.AddDate(0, 0, 1)
	}

	return start, end
}

func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var v int
	_, _ = fmt.Sscanf(s, "%d", &v)
	if v <= 0 {
		return defaultValue
	}
	return v
}
