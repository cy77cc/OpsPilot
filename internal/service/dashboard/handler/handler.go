// Package handler 提供仪表盘 HTTP 处理器。
//
// 本文件实现仪表盘相关的 HTTP 接口，提供系统整体概览数据的查询入口。
package handler

import (
	"strings"

	dashboardv1 "github.com/cy77cc/OpsPilot/api/dashboard/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	dashboardlogic "github.com/cy77cc/OpsPilot/internal/service/dashboard/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 仪表盘 HTTP 处理器。
//
// 负责处理仪表盘相关的 HTTP 请求，包括系统概览数据查询。
type Handler struct {
	logic  *dashboardlogic.Logic // 业务逻辑层
	svcCtx *svc.ServiceContext   // 服务上下文
}

// NewHandler 创建仪表盘处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接和配置
//
// 返回: 仪表盘处理器实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: dashboardlogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

// GetOverview 获取系统概览数据。
//
// @Summary 获取系统概览
// @Description 获取主机、集群、服务健康统计，以及告警、事件、指标和 AI 活动数据
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param time_range query string false "时间范围 (1h/6h/24h)" default(1h)
// @Success 200 {object} httpx.Response{data=dashboardv1.OverviewResponse}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /dashboard/overview [get]
func (h *Handler) GetOverview(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}

	var req dashboardv1.OverviewRequest
	req.TimeRange = strings.TrimSpace(c.Query("time_range"))
	if req.TimeRange != "" && req.TimeRange != "1h" && req.TimeRange != "6h" && req.TimeRange != "24h" {
		httpx.Fail(c, xcode.ParamError, "time_range must be one of: 1h, 6h, 24h")
		return
	}

	resp, err := h.logic.GetOverview(c.Request.Context(), req.TimeRange)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

// GetOverviewV2 获取增强版系统概览数据。
//
// @Summary 获取增强版系统概览
// @Description 获取健康概览、资源使用、运行状态、告警、事件和 AI 活动数据
// @Tags 仪表盘
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param time_range query string false "时间范围 (1h/6h/24h)" default(1h)
// @Success 200 {object} httpx.Response{data=dashboardv1.OverviewResponseV2}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /dashboard/overview/v2 [get]
func (h *Handler) GetOverviewV2(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "monitoring:read") {
		return
	}

	timeRange := strings.TrimSpace(c.Query("time_range"))
	if timeRange != "" && timeRange != "1h" && timeRange != "6h" && timeRange != "24h" {
		httpx.Fail(c, xcode.ParamError, "time_range must be one of: 1h, 6h, 24h")
		return
	}

	resp, err := h.logic.GetOverviewV2(c.Request.Context(), timeRange)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}
