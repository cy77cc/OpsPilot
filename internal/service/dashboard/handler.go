package dashboard

import (
	"strings"

	dashboardv1 "github.com/cy77cc/k8s-manage/api/dashboard/v1"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: NewLogic(svcCtx), svcCtx: svcCtx}
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
