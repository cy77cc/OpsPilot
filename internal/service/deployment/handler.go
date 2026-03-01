package deployment

import (
	"fmt"
	"strings"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: NewLogic(svcCtx), svcCtx: svcCtx}
}

func (h *Handler) ListTargets(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:read") {
		return
	}
	list, err := h.logic.ListTargets(c.Request.Context(), httpx.UintFromQuery(c, "project_id"), httpx.UintFromQuery(c, "team_id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) CreateTarget(c *gin.Context) {
	var req TargetUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") || !h.authorizeRuntime(c, req.TargetType, "apply") {
		return
	}
	resp, err := h.logic.CreateTarget(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) GetTarget(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:read") {
		return
	}
	id := httpx.UintFromParam(c, "id")
	resp, err := h.logic.GetTarget(c.Request.Context(), id)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) UpdateTarget(c *gin.Context) {
	var req TargetUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") || !h.authorizeRuntime(c, req.TargetType, "apply") {
		return
	}
	resp, err := h.logic.UpdateTarget(c.Request.Context(), httpx.UintFromParam(c, "id"), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) DeleteTarget(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") {
		return
	}
	if err := h.logic.DeleteTarget(c.Request.Context(), httpx.UintFromParam(c, "id")); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) PutTargetNodes(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") {
		return
	}
	var req struct {
		Nodes []TargetNodeReq `json:"nodes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if err := h.logic.ReplaceTargetNodes(c.Request.Context(), httpx.UintFromParam(c, "id"), req.Nodes); err != nil {
		httpx.ServerErr(c, err)
		return
	}
	resp, _ := h.logic.GetTarget(c.Request.Context(), httpx.UintFromParam(c, "id"))
	httpx.OK(c, resp)
}

func (h *Handler) PreviewRelease(c *gin.Context) {
	var req ReleasePreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	target, terr := h.logic.GetTarget(c.Request.Context(), req.TargetID)
	if terr != nil {
		httpx.ServerErr(c, terr)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:apply") || !h.authorizeRuntime(c, target.RuntimeType, "apply") {
		return
	}
	resp, err := h.logic.PreviewRelease(c.Request.Context(), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ApplyRelease(c *gin.Context) {
	var req ReleasePreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	target, terr := h.logic.GetTarget(c.Request.Context(), req.TargetID)
	if terr != nil {
		httpx.ServerErr(c, terr)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:apply") || !h.authorizeRuntime(c, target.RuntimeType, "apply") {
		return
	}
	resp, err := h.logic.ApplyRelease(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) RollbackRelease(c *gin.Context) {
	row, err := h.logic.GetRelease(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:rollback") || !h.authorizeRuntime(c, row.RuntimeType, "rollback") {
		return
	}
	resp, err := h.logic.RollbackRelease(c.Request.Context(), httpx.UintFromParam(c, "id"), httpx.UIDFromCtx(c))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ApproveRelease(c *gin.Context) {
	row, err := h.logic.GetRelease(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:approve", "deploy:release:apply") || !h.authorizeRuntime(c, row.RuntimeType, "apply") {
		return
	}
	var req ReleaseDecisionReq
	_ = c.ShouldBindJSON(&req)
	resp, err := h.logic.ApproveRelease(c.Request.Context(), row.ID, httpx.UIDFromCtx(c), req.Comment)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) RejectRelease(c *gin.Context) {
	row, err := h.logic.GetRelease(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:approve", "deploy:release:apply") || !h.authorizeRuntime(c, row.RuntimeType, "apply") {
		return
	}
	var req ReleaseDecisionReq
	_ = c.ShouldBindJSON(&req)
	resp, err := h.logic.RejectRelease(c.Request.Context(), row.ID, httpx.UIDFromCtx(c), req.Comment)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ListReleaseTimeline(c *gin.Context) {
	row, err := h.logic.GetRelease(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:read") || !h.authorizeRuntime(c, row.RuntimeType, "read") {
		return
	}
	list, err := h.logic.ListReleaseTimeline(c.Request.Context(), row.ID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) ListReleases(c *gin.Context) {
	runtime := strings.TrimSpace(c.Query("runtime_type"))
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:read") {
		return
	}
	if runtime != "" && !h.authorizeRuntime(c, runtime, "read") {
		return
	}
	rows, err := h.logic.ListReleases(c.Request.Context(), httpx.UintFromQuery(c, "service_id"), httpx.UintFromQuery(c, "target_id"), runtime)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	list := make([]ReleaseSummaryResp, 0, len(rows))
	for i := range rows {
		list = append(list, h.toReleaseSummary(rows[i]))
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) GetRelease(c *gin.Context) {
	row, err := h.logic.GetRelease(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:release:read") || !h.authorizeRuntime(c, row.RuntimeType, "read") {
		return
	}
	httpx.OK(c, h.toReleaseSummary(*row))
}

func (h *Handler) GetGovernance(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:governance:read", "service:read") {
		return
	}
	row, err := h.logic.GetGovernance(c.Request.Context(), httpx.UintFromParam(c, "id"), strings.TrimSpace(c.Query("env")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) PutGovernance(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:governance:write", "service:write") {
		return
	}
	var req GovernanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	row, err := h.logic.UpsertGovernance(c.Request.Context(), httpx.UIDFromCtx(c), httpx.UintFromParam(c, "id"), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, row)
}

func (h *Handler) PreviewClusterBootstrap(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") {
		return
	}
	var req ClusterBootstrapPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.PreviewClusterBootstrap(c.Request.Context(), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ApplyClusterBootstrap(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") {
		return
	}
	var req ClusterBootstrapPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.ApplyClusterBootstrap(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) GetClusterBootstrapTask(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:read") {
		return
	}
	task, err := h.logic.GetClusterBootstrapTask(c.Request.Context(), strings.TrimSpace(c.Param("task_id")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, task)
}

func (h *Handler) authorizeRuntime(c *gin.Context, runtime, action string) bool {
	r := strings.TrimSpace(runtime)
	if r == "" {
		return true
	}
	code := fmt.Sprintf("deploy:%s:%s", r, action)
	return httpx.Authorize(c, h.svcCtx.DB, code)
}

func (h *Handler) toReleaseSummary(row model.DeploymentRelease) ReleaseSummaryResp {
	return ReleaseSummaryResp{
		ID:                 row.ID,
		ServiceID:          row.ServiceID,
		TargetID:           row.TargetID,
		NamespaceOrProject: row.NamespaceOrProject,
		RuntimeType:        row.RuntimeType,
		Strategy:           row.Strategy,
		RevisionID:         row.RevisionID,
		SourceReleaseID:    row.SourceReleaseID,
		TargetRevision:     row.TargetRevision,
		Status:             row.Status,
		LifecycleState:     h.logic.releaseLifecycleState(row.Status),
		DiagnosticsJSON:    row.DiagnosticsJSON,
		VerificationJSON:   row.VerificationJSON,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
		PreviewExpiresAt:   row.PreviewExpiresAt,
	}
}
