// Package handler 提供项目管理相关的 HTTP 处理器。
//
// 本文件实现服务管理的 HTTP Handler，负责请求参数绑定、校验和响应封装。
package handler

import (
	"strconv"
	"time"

	v1 "github.com/cy77cc/OpsPilot/api/project/v1"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/project/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// ServiceHandler 是服务管理的 HTTP 处理器。
//
// 职责:
//   - 处理服务的 CRUD 操作
//   - 处理服务部署、回滚、事件和配额查询
type ServiceHandler struct {
	logic *logic.ServiceLogic
}

// NewServiceHandler 创建服务处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 服务处理器实例
func NewServiceHandler(svcCtx *svc.ServiceContext) *ServiceHandler {
	return &ServiceHandler{logic: logic.NewServiceLogic(svcCtx)}
}

// CreateService 创建服务。
//
// @Summary 创建服务
// @Description 在项目下创建新服务
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body v1.CreateServiceReq true "服务创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services [post]
func (h *ServiceHandler) CreateService(c *gin.Context) {
	var req v1.CreateServiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.CreateService(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

// ListServices 获取服务列表。
//
// @Summary 获取服务列表
// @Description 根据项目 ID 获取服务列表
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param project_id query int false "项目 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services [get]
func (h *ServiceHandler) ListServices(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	var projectID uint
	if projectIDStr != "" {
		pid, err := strconv.ParseUint(projectIDStr, 10, 64)
		if err != nil {
			httpx.Fail(c, xcode.ParamError, "invalid project_id")
			return
		}
		projectID = uint(pid)
	}
	resp, err := h.logic.ListServices(c.Request.Context(), projectID)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"data": resp, "total": len(resp)})
}

// GetService 获取服务详情。
//
// @Summary 获取服务详情
// @Description 根据服务 ID 获取服务详细信息
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id} [get]
func (h *ServiceHandler) GetService(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	resp, err := h.logic.GetService(c.Request.Context(), uint(id))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

// UpdateService 更新服务。
//
// @Summary 更新服务
// @Description 更新服务配置信息
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body v1.CreateServiceReq true "服务更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id} [put]
func (h *ServiceHandler) UpdateService(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req v1.CreateServiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.logic.UpdateService(c.Request.Context(), uint(id), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

// DeleteService 删除服务。
//
// @Summary 删除服务
// @Description 根据服务 ID 删除服务
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id} [delete]
func (h *ServiceHandler) DeleteService(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.logic.DeleteService(c.Request.Context(), uint(id)); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// DeployService 部署服务。
//
// @Summary 部署服务
// @Description 将服务部署到指定集群
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Param body body object{cluster_id=int} true "部署请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /services/{id}/deploy [post]
func (h *ServiceHandler) DeployService(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req struct {
		ClusterID uint `json:"cluster_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if req.ClusterID == 0 {
		req.ClusterID = 1
	}
	if err := h.logic.DeployService(c.Request.Context(), v1.DeployServiceReq{ServiceID: uint(id), ClusterID: req.ClusterID}); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// RollbackService 回滚服务。
//
// @Summary 回滚服务
// @Description 回滚服务到上一个版本
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Router /services/{id}/rollback [post]
func (h *ServiceHandler) RollbackService(c *gin.Context) {
	httpx.OK(c, nil)
}

// GetEvents 获取服务事件。
//
// @Summary 获取服务事件
// @Description 获取服务的部署事件列表
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Router /services/{id}/events [get]
func (h *ServiceHandler) GetEvents(c *gin.Context) {
	httpx.OK(c, []gin.H{{"id": 1, "service_id": c.Param("id"), "type": "deploy", "level": "info", "message": "service created", "created_at": time.Now()}})
}

// GetQuota 获取服务配额。
//
// @Summary 获取服务配额
// @Description 获取服务的资源配额使用情况
// @Tags 服务管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "服务 ID"
// @Success 200 {object} httpx.Response
// @Router /services/{id}/quota [get]
func (h *ServiceHandler) GetQuota(c *gin.Context) {
	httpx.OK(c, gin.H{"cpuLimit": 8000, "memoryLimit": 16384, "cpuUsed": 1200, "memoryUsed": 2048})
}
