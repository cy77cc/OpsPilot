// Package handler 提供项目管理相关的 HTTP 处理器。
//
// 本文件实现项目模块的 HTTP Handler，负责请求参数绑定、校验和响应封装。
package handler

import (
	v1 "github.com/cy77cc/OpsPilot/api/project/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/project/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// ProjectHandler 是项目管理的 HTTP 处理器。
//
// 职责:
//   - 处理项目的创建、列表查询
//   - 处理项目的批量部署请求
type ProjectHandler struct {
	logic *logic.ProjectLogic
}

// NewProjectHandler 创建项目处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 项目处理器实例
func NewProjectHandler(svcCtx *svc.ServiceContext) *ProjectHandler {
	return &ProjectHandler{logic: logic.NewProjectLogic(svcCtx)}
}

// CreateProject 创建项目。
//
// @Summary 创建项目
// @Description 创建新项目
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body v1.CreateProjectReq true "项目创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req v1.CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.CreateProject(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

// ListProjects 获取项目列表。
//
// @Summary 获取项目列表
// @Description 获取所有项目信息
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /projects [get]
func (h *ProjectHandler) ListProjects(c *gin.Context) {
	resp, err := h.logic.ListProjects(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"data": resp, "total": len(resp)})
}

// DeployProject 部署项目。
//
// @Summary 部署项目
// @Description 将项目下所有服务部署到指定集群
// @Tags 项目管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body v1.DeployProjectReq true "部署请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /projects/deploy [post]
func (h *ProjectHandler) DeployProject(c *gin.Context) {
	var req v1.DeployProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if err := h.logic.DeployProject(c.Request.Context(), req); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}
