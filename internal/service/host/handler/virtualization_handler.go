package handler

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	hostlogic "github.com/cy77cc/k8s-manage/internal/service/host/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

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

func (h *Handler) GetVirtualizationTask(c *gin.Context) {
	task, err := h.hostService.GetVirtualizationTask(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "task not found")
		return
	}
	httpx.OK(c, task)
}
