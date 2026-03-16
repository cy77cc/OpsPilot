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

type Handler struct {
	logic  *dashboardlogic.Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: dashboardlogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

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

// GetOverviewV2 返回增强版主控台概览。
//
// 包含健康概览、资源使用、运行状态、告警、事件和 AI 活动数据。
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
