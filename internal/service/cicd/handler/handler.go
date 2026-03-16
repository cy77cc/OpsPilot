package handler

import (
	"strconv"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	cicdlogic "github.com/cy77cc/OpsPilot/internal/service/cicd/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *cicdlogic.Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: cicdlogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

func (h *Handler) GetServiceCIConfig(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:ci:read", "cicd:*") {
		return
	}
	row, err := h.logic.GetServiceCIConfig(c.Request.Context(), httpx.UintFromParam(c, "service_id"))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, err.Error())
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) PutServiceCIConfig(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:ci:write", "cicd:*") {
		return
	}
	var req cicdlogic.UpsertServiceCIConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.UpsertServiceCIConfig(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "service_id"), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) DeleteServiceCIConfig(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:ci:write", "cicd:*") {
		return
	}
	if err := h.logic.DeleteServiceCIConfig(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "service_id")); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) TriggerCIRun(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:ci:run", "cicd:*") {
		return
	}
	var req cicdlogic.TriggerCIRunReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.TriggerCIRun(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "service_id"), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) ListCIRuns(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:ci:read", "cicd:*") {
		return
	}
	rows, err := h.logic.ListCIRuns(c.Request.Context(), httpx.UintFromParam(c, "service_id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) GetDeploymentCDConfig(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:cd:read", "cicd:*") {
		return
	}
	row, err := h.logic.GetDeploymentCDConfig(c.Request.Context(), httpx.UintFromParam(c, "deployment_id"), strings.TrimSpace(c.Query("env")), strings.TrimSpace(c.Query("runtime_type")))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, err.Error())
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) PutDeploymentCDConfig(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:cd:write", "cicd:*") {
		return
	}
	var req cicdlogic.UpsertDeploymentCDConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.UpsertDeploymentCDConfig(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "deployment_id"), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) TriggerRelease(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:release:run", "cicd:*") {
		return
	}
	var req cicdlogic.TriggerReleaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.TriggerRelease(c.Request.Context(), uint(httpx.UIDFromCtx(c)), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) ListReleases(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:cd:read", "cicd:audit:read", "cicd:*") {
		return
	}
	rows, err := h.logic.ListReleases(c.Request.Context(), httpx.UintFromQuery(c, "service_id"), httpx.UintFromQuery(c, "deployment_id"), strings.TrimSpace(c.Query("runtime_type")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) ApproveRelease(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:release:approve", "cicd:*") {
		return
	}
	var req cicdlogic.ReleaseDecisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.ApproveRelease(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "id"), req.Comment)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) RejectRelease(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:release:approve", "cicd:*") {
		return
	}
	var req cicdlogic.ReleaseDecisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.RejectRelease(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "id"), req.Comment)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) RollbackRelease(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:release:rollback", "cicd:*") {
		return
	}
	var req cicdlogic.RollbackReleaseReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.RollbackRelease(c.Request.Context(), uint(httpx.UIDFromCtx(c)), httpx.UintFromParam(c, "id"), req.TargetVersion, req.Comment)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) ListApprovals(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:audit:read", "cicd:release:approve", "cicd:*") {
		return
	}
	rows, err := h.logic.ListApprovals(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) ServiceTimeline(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:audit:read", "cicd:*") {
		return
	}
	rows, err := h.logic.ServiceTimeline(c.Request.Context(), httpx.UintFromParam(c, "service_id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}

func (h *Handler) ListAuditEvents(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "cicd:audit:read", "cicd:*") {
		return
	}
	limit := 100
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	rows, err := h.logic.ListAuditEvents(
		c.Request.Context(),
		httpx.UintFromQuery(c, "service_id"),
		strings.TrimSpace(c.Query("trace_id")),
		strings.TrimSpace(c.Query("command_id")),
		limit,
	)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": rows, "total": len(rows)})
}
