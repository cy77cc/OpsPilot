// Package deployment 提供部署管理服务的 HTTP 处理器。
//
// 本文件包含部署目标、发布管理和集群引导相关的 HTTP 处理器实现。
package deployment

import (
	"fmt"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// Handler 是部署服务的 HTTP 处理器，封装业务逻辑层和服务上下文。
type Handler struct {
	logic  *Logic
	svcCtx *svc.ServiceContext
}

// NewHandler 创建部署服务处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: NewLogic(svcCtx), svcCtx: svcCtx}
}

// ListTargets 获取部署目标列表。
//
// @Summary 获取部署目标列表
// @Description 获取所有部署目标信息，支持按项目和团队筛选
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param project_id query int false "项目 ID"
// @Param team_id query int false "团队 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/targets [get]
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

// CreateTarget 创建部署目标。
//
// @Summary 创建部署目标
// @Description 创建新的部署目标，支持 Kubernetes 和 Docker Compose 类型
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body TargetUpsertReq true "部署目标信息"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/targets [post]
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

// GetTarget 获取部署目标详情。
//
// @Summary 获取部署目标详情
// @Description 根据ID获取部署目标的详细信息
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "部署目标 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/targets/{id} [get]
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

// UpdateTarget 更新部署目标。
//
// @Summary 更新部署目标
// @Description 更新指定部署目标的信息
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "部署目标 ID"
// @Param body body TargetUpsertReq true "部署目标信息"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/targets/{id} [put]
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

// DeleteTarget 删除部署目标。
//
// @Summary 删除部署目标
// @Description 删除指定的部署目标及其关联节点
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "部署目标 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/targets/{id} [delete]
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

// PutTargetNodes 替换部署目标节点列表。
//
// @Summary 替换部署目标节点
// @Description 替换指定部署目标的节点列表，用于 Docker Compose 场景
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "部署目标 ID"
// @Param body body map[string][]TargetNodeReq true "节点列表"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/targets/{id}/nodes [put]
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

// PreviewRelease 预览发布。
//
// @Summary 预览发布
// @Description 预览部署发布，生成解析后的清单和预览令牌
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body ReleasePreviewReq true "发布预览请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/preview [post]
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

// ApplyRelease 执行发布。
//
// @Summary 执行发布
// @Description 执行部署发布，生产环境需要审批流程
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body ReleasePreviewReq true "发布请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/apply [post]
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

// RollbackRelease 回滚发布。
//
// @Summary 回滚发布
// @Description 回滚到上一个发布版本
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/{id}/rollback [post]
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

// ApproveRelease 审批通过发布。
//
// @Summary 审批通过发布
// @Description 审批通过待审批的发布请求
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Param body body ReleaseDecisionReq false "审批意见"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/{id}/approve [post]
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

// RejectRelease 拒绝发布。
//
// @Summary 拒绝发布
// @Description 拒绝待审批的发布请求
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Param body body ReleaseDecisionReq false "拒绝原因"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/{id}/reject [post]
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

// ListReleaseTimeline 获取发布时间线。
//
// @Summary 获取发布时间线
// @Description 获取指定发布的审计时间线事件列表
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/{id}/timeline [get]
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

// ListReleases 获取发布列表。
//
// @Summary 获取发布列表
// @Description 获取部署发布记录列表，支持按服务和目标筛选
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id query int false "服务 ID"
// @Param target_id query int false "目标 ID"
// @Param runtime_type query string false "运行时类型"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases [get]
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

// GetRelease 获取发布详情。
//
// @Summary 获取发布详情
// @Description 根据ID获取发布的详细信息
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/releases/{id} [get]
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

// GetGovernance 获取服务治理策略。
//
// @Summary 获取服务治理策略
// @Description 获取指定服务的治理策略配置
// @Tags 服务治理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param env query string false "环境"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/governance [get]
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

// PutGovernance 更新服务治理策略。
//
// @Summary 更新服务治理策略
// @Description 更新指定服务的治理策略配置，包括流量、弹性、访问和SLO策略
// @Tags 服务治理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body GovernanceReq true "治理策略"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/governance [put]
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

// PreviewClusterBootstrap 预览集群引导。
//
// @Summary 预览集群引导
// @Description 预览 Kubernetes 集群引导安装步骤
// @Tags 集群引导
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body ClusterBootstrapPreviewReq true "引导预览请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/clusters/bootstrap/preview [post]
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

// ApplyClusterBootstrap 执行集群引导。
//
// @Summary 执行集群引导
// @Description 执行 Kubernetes 集群引导安装，创建集群和部署目标
// @Tags 集群引导
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body ClusterBootstrapPreviewReq true "引导请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/clusters/bootstrap/apply [post]
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

// GetClusterBootstrapTask 获取集群引导任务状态。
//
// @Summary 获取集群引导任务状态
// @Description 根据任务ID获取集群引导安装任务的状态和结果
// @Tags 集群引导
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param task_id path string true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/clusters/bootstrap/{task_id} [get]
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

// authorizeRuntime 检查运行时操作权限。
//
// 参数:
//   - c: Gin 上下文
//   - runtime: 运行时类型 (k8s/compose)
//   - action: 操作类型 (read/apply/rollback)
//
// 返回: 有权限返回 true，否则返回 false
func (h *Handler) authorizeRuntime(c *gin.Context, runtime, action string) bool {
	r := strings.TrimSpace(runtime)
	if r == "" {
		return true
	}
	code := fmt.Sprintf("deploy:%s:%s", r, action)
	return httpx.Authorize(c, h.svcCtx.DB, code)
}

// toReleaseSummary 将发布模型转换为响应摘要。
//
// 参数:
//   - row: 发布记录模型
//
// 返回: 发布摘要响应
func (h *Handler) toReleaseSummary(row model.DeploymentRelease) ReleaseSummaryResp {
	return ReleaseSummaryResp{
		ID:                 row.ID,
		UnifiedReleaseID:   row.ID,
		ServiceID:          row.ServiceID,
		TargetID:           row.TargetID,
		NamespaceOrProject: row.NamespaceOrProject,
		RuntimeType:        row.RuntimeType,
		Strategy:           row.Strategy,
		TriggerSource:      row.TriggerSource,
		TriggerContextJSON: row.TriggerContextJSON,
		CIRunID:            row.CIRunID,
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
