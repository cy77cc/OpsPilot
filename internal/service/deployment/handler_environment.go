// Package deployment 提供部署管理服务的环境引导处理器。
//
// 本文件包含环境引导和凭证管理相关的 HTTP 处理器实现。
package deployment

import (
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

// StartEnvironmentBootstrap 启动环境引导安装。
//
// @Summary 启动环境引导安装
// @Description 启动 Kubernetes 或 Docker Compose 环境的引导安装任务
// @Tags 环境引导
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body EnvironmentBootstrapReq true "引导请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/environments/bootstrap [post]
func (h *Handler) StartEnvironmentBootstrap(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:write") {
		return
	}
	var req EnvironmentBootstrapReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if !h.authorizeRuntime(c, req.RuntimeType, "apply") {
		return
	}
	resp, err := h.logic.StartEnvironmentBootstrap(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

// GetEnvironmentBootstrapJob 获取环境引导任务状态。
//
// @Summary 获取环境引导任务状态
// @Description 根据任务ID获取环境引导安装任务的状态和结果
// @Tags 环境引导
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param job_id path string true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/environments/bootstrap/{job_id} [get]
func (h *Handler) GetEnvironmentBootstrapJob(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:target:read") {
		return
	}
	job, err := h.logic.GetEnvironmentBootstrapJob(c.Request.Context(), strings.TrimSpace(c.Param("job_id")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, job)
}

// RegisterPlatformCredential 注册平台托管凭证。
//
// @Summary 注册平台托管凭证
// @Description 从平台托管集群创建凭证记录
// @Tags 凭证管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body PlatformCredentialRegisterReq true "注册请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/credentials/platform/register [post]
func (h *Handler) RegisterPlatformCredential(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:credential:write", "deploy:target:write") {
		return
	}
	var req PlatformCredentialRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.RegisterPlatformCredential(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

// ImportExternalCredential 导入外部集群凭证。
//
// @Summary 导入外部集群凭证
// @Description 导入外部 Kubernetes 集群的访问凭证
// @Tags 凭证管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body ClusterCredentialImportReq true "导入请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/credentials/import [post]
func (h *Handler) ImportExternalCredential(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:credential:write", "deploy:target:write") {
		return
	}
	var req ClusterCredentialImportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.ImportExternalCredential(c.Request.Context(), httpx.UIDFromCtx(c), req)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

// TestCredential 测试凭证连通性。
//
// @Summary 测试凭证连通性
// @Description 测试指定凭证是否可以正常连接集群
// @Tags 凭证管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "凭证 ID"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/credentials/{id}/test [post]
func (h *Handler) TestCredential(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:credential:read", "deploy:target:read") {
		return
	}
	resp, err := h.logic.TestCredentialConnectivity(c.Request.Context(), httpx.UintFromParam(c, "id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, resp)
}

// ListCredentials 获取凭证列表。
//
// @Summary 获取凭证列表
// @Description 获取所有集群凭证列表，支持按运行时类型筛选
// @Tags 凭证管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param runtime_type query string false "运行时类型"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/credentials [get]
func (h *Handler) ListCredentials(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "deploy:credential:read", "deploy:target:read") {
		return
	}
	list, err := h.logic.ListCredentials(c.Request.Context(), strings.TrimSpace(c.Query("runtime_type")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}
