// Package service 提供服务目录管理的 HTTP 处理器。
//
// 本文件包含服务管理模块的所有 HTTP Handler 实现，涵盖：
//   - 服务 CRUD 操作
//   - 服务渲染和预览
//   - 变量管理
//   - 版本管理
//   - 部署操作
//   - Helm 部署支持
package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Handler 封装服务管理模块的 HTTP 处理器。
//
// 包含业务逻辑层引用和服务上下文，用于处理服务管理相关的 HTTP 请求。
type Handler struct {
	logic  *Logic
	svcCtx *svc.ServiceContext
}

// NewHandler 创建服务管理模块的 Handler 实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: Handler 实例指针
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: NewLogic(svcCtx), svcCtx: svcCtx}
}

// Preview 预览服务配置渲染结果。
//
// @Summary 预览服务配置渲染
// @Description 根据标准配置或自定义 YAML 预览渲染结果，支持变量替换和语法校验
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body RenderPreviewReq true "渲染预览请求"
// @Success 200 {object} httpx.Response{data=RenderPreviewResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/render/preview [post]
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

// Transform 将标准服务配置转换为自定义 YAML。
//
// @Summary 标准配置转自定义 YAML
// @Description 将标准服务配置转换为自定义 YAML 格式，用于高级定制场景
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body TransformReq true "转换请求"
// @Success 200 {object} httpx.Response{data=TransformResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/transform [post]
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

// Create 创建新服务。
//
// @Summary 创建服务
// @Description 创建新的服务配置，支持标准模式和自定义 YAML 模式
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param X-Project-ID header string false "项目 ID"
// @Param X-Team-ID header string false "团队 ID"
// @Param body body ServiceCreateReq true "服务创建请求"
// @Success 200 {object} httpx.Response{data=ServiceListItem}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services [post]
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

// Update 更新服务配置。
//
// @Summary 更新服务
// @Description 更新指定服务的配置信息，会自动创建新的版本记录
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param X-Project-ID header string false "项目 ID"
// @Param X-Team-ID header string false "团队 ID"
// @Param body body ServiceCreateReq true "服务更新请求"
// @Success 200 {object} httpx.Response{data=ServiceListItem}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id} [put]
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

// List 获取服务列表。
//
// @Summary 获取服务列表
// @Description 根据筛选条件获取服务列表，支持项目、团队、运行时类型等多维度过滤
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param project_id query int false "项目 ID"
// @Param team_id query int false "团队 ID"
// @Param runtime_type query string false "运行时类型 (k8s/compose)"
// @Param service_kind query string false "服务类型 (business/middleware)"
// @Param visibility query string false "可见性 (private/team/team-granted/public)"
// @Param env query string false "环境 (development/staging/production)"
// @Param label_selector query string false "标签选择器"
// @Param q query string false "搜索关键词"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services [get]
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

// Get 获取服务详情。
//
// @Summary 获取服务详情
// @Description 根据 ID 获取指定服务的详细配置信息
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response{data=ServiceListItem}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id} [get]
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

// Delete 删除服务。
//
// @Summary 删除服务
// @Description 删除指定服务，需要编辑权限
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id} [delete]
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

// UpdateVisibility 更新服务可见性。
//
// @Summary 更新服务可见性
// @Description 更新指定服务的可见性设置 (private/team/team-granted/public)
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body VisibilityUpdateReq true "可见性更新请求"
// @Success 200 {object} httpx.Response{data=ServiceListItem}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/visibility [put]
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

// UpdateGrantedTeams 更新服务授权团队。
//
// @Summary 更新服务授权团队
// @Description 更新指定服务的授权团队列表，用于 team-granted 可见性模式
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body GrantTeamsReq true "授权团队更新请求"
// @Success 200 {object} httpx.Response{data=ServiceListItem}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/grant-teams [put]
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

// Deploy 部署服务。
//
// @Summary 部署服务
// @Description 将服务部署到指定集群和命名空间，生产环境部署需要审批权限
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body DeployReq true "部署请求"
// @Success 200 {object} httpx.Response{data=DeployResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/deploy [post]
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

// DeployPreview 预览部署结果。
//
// @Summary 预览部署结果
// @Description 预览服务部署的渲染结果和检查项，不实际执行部署
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body DeployReq true "部署预览请求"
// @Success 200 {object} httpx.Response{data=DeployPreviewResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/deploy/preview [post]
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

// HelmImport 导入 Helm Chart。
//
// @Summary 导入 Helm Chart
// @Description 将 Helm Chart 导入到服务配置中
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body HelmImportReq true "Helm 导入请求"
// @Success 200 {object} httpx.Response{data=model.ServiceHelmRelease}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/helm/import [post]
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

// HelmRender 渲染 Helm 模板。
//
// @Summary 渲染 Helm 模板
// @Description 使用 helm template 命令渲染 Helm Chart
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body HelmRenderReq true "Helm 渲染请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/helm/render [post]
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

// DeployHelm 部署 Helm 服务。
//
// @Summary 部署 Helm 服务
// @Description 部署已导入的 Helm Chart 服务
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/deploy/helm [post]
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

// Rollback 回滚服务部署。
//
// @Summary 回滚服务部署
// @Description 回滚服务到上一个版本（当前为占位实现）
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/rollback [post]
func (h *Handler) Rollback(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:deploy") {
		return
	}
	httpx.OK(c, nil)
}

// Events 获取服务事件列表。
//
// @Summary 获取服务事件列表
// @Description 获取指定服务的事件列表（当前为占位实现）
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/events [get]
func (h *Handler) Events(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "service:read") {
		return
	}
	httpx.OK(c, []gin.H{
		{"id": 1, "service_id": c.Param("id"), "type": "deploy", "level": "info", "message": "service event", "created_at": time.Now()},
	})
}

// Quota 获取服务资源配额。
//
// @Summary 获取服务资源配额
// @Description 获取服务的 CPU 和内存配额及使用情况（当前为占位实现）
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/quota [get]
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

// ExtractVariables 提取服务模板变量。
//
// @Summary 提取服务模板变量
// @Description 从服务配置中提取模板变量定义
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body VariableExtractReq true "变量提取请求"
// @Success 200 {object} httpx.Response{data=VariableExtractResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/variables/extract [post]
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

// GetVariableSchema 获取服务变量 Schema。
//
// @Summary 获取服务变量 Schema
// @Description 获取指定服务的变量 Schema 定义
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/variables/schema [get]
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

// GetVariableValues 获取服务变量值。
//
// @Summary 获取服务变量值
// @Description 获取指定服务在指定环境下的变量值
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param env query string false "环境名称"
// @Success 200 {object} httpx.Response{data=VariableValuesResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/variables/values [get]
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

// UpsertVariableValues 更新服务变量值。
//
// @Summary 更新服务变量值
// @Description 创建或更新指定服务在指定环境下的变量值
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body VariableValuesUpsertReq true "变量值更新请求"
// @Success 200 {object} httpx.Response{data=VariableValuesResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/variables/values [put]
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

// ListRevisions 获取服务版本列表。
//
// @Summary 获取服务版本列表
// @Description 获取指定服务的所有版本记录
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/revisions [get]
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

// CreateRevision 创建服务版本。
//
// @Summary 创建服务版本
// @Description 为指定服务创建新的版本记录
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body RevisionCreateReq true "版本创建请求"
// @Success 200 {object} httpx.Response{data=ServiceRevisionItem}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/revisions [post]
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

// UpsertDeployTarget 更新服务部署目标。
//
// @Summary 更新服务部署目标
// @Description 创建或更新指定服务的默认部署目标配置
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body DeployTargetUpsertReq true "部署目标更新请求"
// @Success 200 {object} httpx.Response{data=DeployTargetResp}
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/deploy-target [put]
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

// ListReleaseRecords 获取服务发布记录。
//
// @Summary 获取服务发布记录
// @Description 获取指定服务的部署发布记录列表
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/releases [get]
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

// checkOwnershipHeaders 校验请求头中的项目/团队归属是否匹配。
//
// 参数:
//   - c: Gin 上下文
//   - projectID: 请求中的项目 ID
//   - teamID: 请求中的团队 ID
//
// 返回: 校验通过返回 true，否则返回 false 并返回错误响应
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

// fillOwnershipFromHeaders 从请求头填充项目/团队归属信息。
//
// 如果请求体中未指定项目/团队 ID，则从请求头 X-Project-ID 和 X-Team-ID 中获取。
//
// 参数:
//   - c: Gin 上下文
//   - req: 服务创建请求，将被修改
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

// checkViewPermission 校验用户是否有查看服务的权限。
//
// 权限规则:
//   - 管理员有所有权限
//   - 服务所有者有权限
//   - 根据可见性设置判断团队权限
//
// 参数:
//   - c: Gin 上下文
//   - serviceID: 服务 ID
//
// 返回: 有权限返回 true，否则返回 false 并返回错误响应
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

// checkEditPermission 校验用户是否有编辑服务的权限。
//
// 权限规则:
//   - 管理员有所有权限
//   - 服务所有者有权限
//   - 团队成员有权限
//
// 参数:
//   - c: Gin 上下文
//   - serviceID: 服务 ID
//
// 返回: 有权限返回 true，否则返回 false 并返回错误响应
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
