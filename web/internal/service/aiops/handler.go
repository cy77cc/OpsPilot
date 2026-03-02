package aiops

import (
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{svcCtx: svcCtx}
}

// ListRiskFindings 获取风险发现列表
func (h *Handler) ListRiskFindings(c *gin.Context) {
	ctx := c.Request.Context()
	
	findings, err := h.listRiskFindings(ctx)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	
	httpx.OK(c, gin.H{"list": findings, "total": len(findings)})
}

// ListAnomalies 获取异常检测列表
func (h *Handler) ListAnomalies(c *gin.Context) {
	ctx := c.Request.Context()
	
	anomalies, err := h.listAnomalies(ctx)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	
	httpx.OK(c, gin.H{"list": anomalies, "total": len(anomalies)})
}

// ListSuggestions 获取优化建议列表
func (h *Handler) ListSuggestions(c *gin.Context) {
	ctx := c.Request.Context()
	
	suggestions, err := h.listSuggestions(ctx)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	
	httpx.OK(c, gin.H{"list": suggestions, "total": len(suggestions)})
}
