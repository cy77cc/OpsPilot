// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// KVMPreview 预览 KVM 虚拟机配置。
//
// @Summary 预览 KVM 配置
// @Description 预览在指定主机上创建 KVM 虚拟机的配置参数
// @Tags KVM 虚拟化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "宿主机 ID"
// @Param body body hostlogic.KVMPreviewReq true "预览请求参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/virtualization/kvm/hosts/{id}/preview [post]
func (h *Handler) KVMPreview(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	var req hostlogic.KVMPreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	result, err := h.hostService.KVMPreview(c.Request.Context(), hostID, req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, result)
}

// KVMProvision 创建 KVM 虚拟机。
//
// @Summary 创建 KVM 虚拟机
// @Description 在指定主机上创建 KVM 虚拟机并注册为新主机
// @Tags KVM 虚拟化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "宿主机 ID"
// @Param body body hostlogic.KVMProvisionReq true "创建请求参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/virtualization/kvm/hosts/{id}/provision [post]
func (h *Handler) KVMProvision(c *gin.Context) {
	hostID, ok := parseID(c)
	if !ok {
		return
	}
	var req hostlogic.KVMProvisionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	task, node, err := h.hostService.KVMProvision(c.Request.Context(), getUID(c), hostID, req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"task": task, "node": node})
}

// GetVirtualizationTask 获取虚拟化任务状态。
//
// @Summary 获取虚拟化任务
// @Description 获取 KVM 虚拟机创建任务的状态和结果
// @Tags KVM 虚拟化
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param task_id path string true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/virtualization/tasks/{task_id} [get]
func (h *Handler) GetVirtualizationTask(c *gin.Context) {
	task, err := h.hostService.GetVirtualizationTask(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "task not found")
		return
	}
	httpx.OK(c, task)
}
