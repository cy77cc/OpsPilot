package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/model"
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

func (h *Handler) Preview(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	var req RenderPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.Preview(req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Transform(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	var req TransformReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.Transform(req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Create(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	var req ServiceCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	h.fillOwnershipFromHeaders(c, &req)
	if !h.checkOwnershipHeaders(c, req.ProjectID, req.TeamID) {
		return
	}
	resp, err := h.logic.Create(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Update(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if !h.checkEditPermission(c, uint(id)) {
		return
	}
	var req ServiceCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	h.fillOwnershipFromHeaders(c, &req)
	if !h.checkOwnershipHeaders(c, req.ProjectID, req.TeamID) {
		return
	}
	resp, err := h.logic.Update(c.Request.Context(), uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) List(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	filters := map[string]string{
		"project_id":      c.Query("project_id"),
		"team_id":         c.Query("team_id"),
		"runtime_type":    c.Query("runtime_type"),
		"service_kind":    c.Query("service_kind"),
		"visibility":      c.Query("visibility"),
		"env":             c.Query("env"),
		"label_selector":  c.Query("label_selector"),
		"q":               c.Query("q"),
		"viewer_uid":      strconv.FormatUint(httpx.UIDFromCtx(c), 10),
		"viewer_is_admin": strconv.FormatBool(httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c))),
		"viewer_team_id":  strings.TrimSpace(c.GetHeader("X-Team-ID")),
	}
	if !httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)) {
		if hp := strings.TrimSpace(c.GetHeader("X-Project-ID")); hp != "" {
			filters["project_id"] = hp
		}
		if ht := strings.TrimSpace(c.GetHeader("X-Team-ID")); ht != "" {
			filters["team_id"] = ht
		}
	}
	list, total, err := h.logic.List(c.Request.Context(), filters)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": total})
}

func (h *Handler) Get(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if !h.checkViewPermission(c, uint(id)) {
		return
	}
	resp, err := h.logic.Get(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Delete(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if !h.checkEditPermission(c, uint(id)) {
		return
	}
	if err := h.logic.Delete(c.Request.Context(), uint(id)); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) UpdateVisibility(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if !h.checkEditPermission(c, uint(id)) {
		return
	}
	var req VisibilityUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.UpdateVisibility(c.Request.Context(), uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) UpdateGrantedTeams(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if !h.checkEditPermission(c, uint(id)) {
		return
	}
	var req GrantTeamsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.UpdateGrantedTeams(c.Request.Context(), uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Deploy(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:deploy") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req DeployReq
	_ = c.ShouldBindJSON(&req)

	item, err := h.logic.Get(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}

	// 若显式传入 cluster_id 则校验服务环境与集群环境类型匹配。
	if req.ClusterID > 0 {
		envMatchErr := h.logic.ValidateEnvMatch(c.Request.Context(), item.Env, req.ClusterID)
		if envMatchErr != nil {
			httpx.Fail(c, xcode.ParamError, envMatchErr.Error())
			return
		}
	}

	env := defaultIfEmpty(req.Env, item.Env)
	if strings.EqualFold(env, "production") && !httpx.HasAnyPermission(h.svcCtx.DB, httpx.UIDFromCtx(c), "service:approve") {
		httpx.Fail(c, xcode.Forbidden, "production deploy requires service:approve")
		return
	}
	recordID, err := h.logic.Deploy(c.Request.Context(), uint(id), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, DeployResp{ReleaseRecordID: recordID, UnifiedReleaseID: recordID, TriggerSource: "manual"})
}

func (h *Handler) DeployPreview(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:deploy") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req DeployReq
	_ = c.ShouldBindJSON(&req)
	resp, err := h.logic.DeployPreview(c.Request.Context(), uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) HelmImport(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	var req HelmImportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.HelmImport(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) HelmRender(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	var req HelmRenderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	rendered, diags, err := h.logic.HelmRender(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"rendered_yaml": rendered, "diagnostics": diags})
}

func (h *Handler) DeployHelm(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:deploy") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.logic.deployHelm(c.Request.Context(), uint(id)); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) Rollback(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:deploy") {
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) Events(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	httpx.OK(c, []gin.H{
		{"id": 1, "service_id": c.Param("id"), "type": "deploy", "level": "info", "message": "service event", "created_at": time.Now()},
	})
}

func (h *Handler) Quota(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	httpx.OK(c, gin.H{
		"cpuLimit":    8000,
		"memoryLimit": 16384,
		"cpuUsed":     1200,
		"memoryUsed":  2048,
	})
}

func (h *Handler) ExtractVariables(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	var req VariableExtractReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.ExtractVariables(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) GetVariableSchema(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	resp, err := h.logic.GetVariableSchema(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"vars": resp})
}

func (h *Handler) GetVariableValues(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	resp, err := h.logic.GetVariableValues(c.Request.Context(), uint(id), c.Query("env"))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) UpsertVariableValues(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req VariableValuesUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.UpsertVariableValues(c.Request.Context(), uint(id), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ListRevisions(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	resp, err := h.logic.ListRevisions(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": resp, "total": len(resp)})
}

func (h *Handler) CreateRevision(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req RevisionCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.CreateRevision(c.Request.Context(), uint(id), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) UpsertDeployTarget(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:write") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req DeployTargetUpsertReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.UpsertDeployTarget(c.Request.Context(), uint(id), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) ListReleaseRecords(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	resp, err := h.logic.ListReleaseRecords(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": resp, "total": len(resp)})
}

func (h *Handler) checkOwnershipHeaders(c *gin.Context, projectID, teamID uint) bool {
	if httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)) {
		return true
	}
	if hp := strings.TrimSpace(c.GetHeader("X-Project-ID")); hp != "" && projectID > 0 {
		if strconv.FormatUint(uint64(projectID), 10) != hp {
			httpx.Fail(c, xcode.Forbidden, "project ownership mismatch")
			return false
		}
	}
	if ht := strings.TrimSpace(c.GetHeader("X-Team-ID")); ht != "" && teamID > 0 {
		if strconv.FormatUint(uint64(teamID), 10) != ht {
			httpx.Fail(c, xcode.Forbidden, "team ownership mismatch")
			return false
		}
	}
	return true
}

func (h *Handler) fillOwnershipFromHeaders(c *gin.Context, req *ServiceCreateReq) {
	if req == nil || httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)) {
		return
	}
	if req.ProjectID == 0 {
		if hp := strings.TrimSpace(c.GetHeader("X-Project-ID")); hp != "" {
			if p, err := strconv.ParseUint(hp, 10, 64); err == nil && p > 0 {
				req.ProjectID = uint(p)
			}
		}
	}
	if req.TeamID == 0 {
		if ht := strings.TrimSpace(c.GetHeader("X-Team-ID")); ht != "" {
			if t, err := strconv.ParseUint(ht, 10, 64); err == nil && t > 0 {
				req.TeamID = uint(t)
			}
		}
	}
}

func (h *Handler) checkViewPermission(c *gin.Context, serviceID uint) bool {
	if httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)) {
		return true
	}
	var svc model.Service
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Select("id", "owner_user_id", "team_id", "visibility", "granted_teams").First(&svc, serviceID).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "service not found")
		return false
	}
	viewerTeamID := uint(0)
	if ht := strings.TrimSpace(c.GetHeader("X-Team-ID")); ht != "" {
		if t, err := strconv.ParseUint(ht, 10, 64); err == nil {
			viewerTeamID = uint(t)
		}
	}
	if !h.logic.CheckViewPermission(&svc, httpx.UIDFromCtx(c), viewerTeamID, false) {
		httpx.Fail(c, xcode.Forbidden, "无权限查看该服务")
		return false
	}
	return true
}

func (h *Handler) checkEditPermission(c *gin.Context, serviceID uint) bool {
	if httpx.IsAdmin(h.svcCtx.DB, httpx.UIDFromCtx(c)) {
		return true
	}
	var svc model.Service
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).Select("id", "owner_user_id", "team_id").First(&svc, serviceID).Error; err != nil {
		httpx.Fail(c, xcode.NotFound, "service not found")
		return false
	}
	viewerTeamID := uint(0)
	if ht := strings.TrimSpace(c.GetHeader("X-Team-ID")); ht != "" {
		if t, err := strconv.ParseUint(ht, 10, 64); err == nil {
			viewerTeamID = uint(t)
		}
	}
	if !h.logic.CheckEditPermission(&svc, httpx.UIDFromCtx(c), viewerTeamID, false) {
		httpx.Fail(c, xcode.Forbidden, "无权限编辑该服务")
		return false
	}
	return true
}
