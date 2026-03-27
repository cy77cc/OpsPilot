// Package handler 提供 CI/CD 服务的 HTTP 处理器。
//
// 本文件包含以下处理器:
//   - CI 配置管理: GetServiceCIConfig, PutServiceCIConfig, DeleteServiceCIConfig
//   - CI 运行: TriggerCIRun, ListCIRuns
//   - CD 配置管理: GetDeploymentCDConfig, PutDeploymentCDConfig
//   - 发布管理: TriggerRelease, ListReleases, ApproveRelease, RejectRelease, RollbackRelease
//   - 审计查询: ListApprovals, ServiceTimeline, ListAuditEvents
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

// Handler 是 CI/CD 服务的 HTTP 处理器。
//
// 职责:
//   - 接收 HTTP 请求并验证权限
//   - 解析请求参数
//   - 调用 Logic 层处理业务逻辑
//   - 返回统一格式的响应
type Handler struct {
	logic  *cicdlogic.Logic // 业务逻辑层
	svcCtx *svc.ServiceContext // 服务上下文
}

// NewHandler 创建 CI/CD 处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库、缓存等依赖
//
// 返回: 初始化的 Handler 实例
func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: cicdlogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

// GetServiceCIConfig 获取服务 CI 配置。
//
// @Summary 获取服务 CI 配置
// @Description 根据服务 ID 获取 CI 配置详情
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /cicd/services/{service_id}/ci-config [get]
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

// PutServiceCIConfig 创建或更新服务 CI 配置。
//
// @Summary 创建或更新服务 CI 配置
// @Description 为指定服务创建或更新 CI 配置，包括仓库地址、构建步骤等
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id path int true "服务 ID"
// @Param request body cicdlogic.UpsertServiceCIConfigReq true "CI 配置请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/services/{service_id}/ci-config [put]
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

// DeleteServiceCIConfig 删除服务 CI 配置。
//
// @Summary 删除服务 CI 配置
// @Description 删除指定服务的 CI 配置
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/services/{service_id}/ci-config [delete]
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

// TriggerCIRun 触发 CI 运行。
//
// @Summary 触发 CI 构建
// @Description 为指定服务触发 CI 构建任务
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id path int true "服务 ID"
// @Param request body cicdlogic.TriggerCIRunReq true "触发请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/services/{service_id}/ci-runs/trigger [post]
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

// ListCIRuns 获取 CI 运行列表。
//
// @Summary 获取 CI 运行列表
// @Description 获取指定服务的 CI 构建历史记录
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/services/{service_id}/ci-runs [get]
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

// GetDeploymentCDConfig 获取部署 CD 配置。
//
// @Summary 获取部署 CD 配置
// @Description 根据部署目标 ID 获取 CD 配置详情
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param deployment_id path int true "部署目标 ID"
// @Param env query string false "环境名称"
// @Param runtime_type query string false "运行时类型"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /cicd/deployments/{deployment_id}/cd-config [get]
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

// PutDeploymentCDConfig 创建或更新部署 CD 配置。
//
// @Summary 创建或更新部署 CD 配置
// @Description 为指定部署目标创建或更新 CD 配置，包括发布策略、审批要求等
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param deployment_id path int true "部署目标 ID"
// @Param request body cicdlogic.UpsertDeploymentCDConfigReq true "CD 配置请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/deployments/{deployment_id}/cd-config [put]
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

// TriggerRelease 触发发布。
//
// @Summary 触发发布
// @Description 触发新的发布流程，支持指定服务、部署目标、版本等参数
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body cicdlogic.TriggerReleaseReq true "发布请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/releases [post]
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

// ListReleases 获取发布列表。
//
// @Summary 获取发布列表
// @Description 根据服务 ID 或部署目标 ID 查询发布历史记录
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id query int false "服务 ID"
// @Param deployment_id query int false "部署目标 ID"
// @Param runtime_type query string false "运行时类型"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/releases [get]
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

// ApproveRelease 审批通过发布。
//
// @Summary 审批通过发布
// @Description 审批通过指定的发布请求
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Param request body cicdlogic.ReleaseDecisionReq true "审批请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/releases/{id}/approve [post]
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

// RejectRelease 拒绝发布。
//
// @Summary 拒绝发布
// @Description 拒绝指定的发布请求
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Param request body cicdlogic.ReleaseDecisionReq true "拒绝请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/releases/{id}/reject [post]
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

// RollbackRelease 回滚发布。
//
// @Summary 回滚发布
// @Description 回滚指定的发布到目标版本
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Param request body cicdlogic.RollbackReleaseReq true "回滚请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/releases/{id}/rollback [post]
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

// ListApprovals 获取发布审批列表。
//
// @Summary 获取发布审批列表
// @Description 获取指定发布的审批历史记录
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "发布 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/releases/{id}/approvals [get]
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

// ServiceTimeline 获取服务时间线。
//
// @Summary 获取服务时间线
// @Description 获取指定服务的 CI/CD 操作时间线，包括构建、发布、审批等事件
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/services/{service_id}/timeline [get]
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

// ListAuditEvents 获取审计事件列表。
//
// @Summary 获取审计事件列表
// @Description 查询 CI/CD 审计事件，支持按服务、追踪 ID、命令 ID 过滤
// @Tags CI/CD
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param service_id query int false "服务 ID"
// @Param trace_id query string false "追踪 ID"
// @Param command_id query string false "命令 ID"
// @Param limit query int false "返回数量限制，默认 100，最大 500"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /cicd/audits [get]
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
