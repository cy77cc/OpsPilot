package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// ListCloudProviders 列出所有已注册的云厂商。
//
// GET /api/v1/hosts/cloud/providers
func (h *Handler) ListCloudProviders(c *gin.Context) {
	providers := cloud.ListProviders()
	httpx.OK(c, providers)
}

func (h *Handler) ListCloudAccounts(c *gin.Context) {
	provider := c.Query("provider")
	list, err := h.hostService.ListCloudAccounts(c.Request.Context(), provider)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) CreateCloudAccount(c *gin.Context) {
	var req hostlogic.CloudAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	item, err := h.hostService.CreateCloudAccount(c.Request.Context(), getUID(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) TestCloudAccount(c *gin.Context) {
	provider := c.Param("provider")
	var req hostlogic.CloudAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	req.Provider = provider
	result, err := h.hostService.TestCloudAccount(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, result)
}

func (h *Handler) QueryCloudInstances(c *gin.Context) {
	provider := c.Param("provider")
	var req hostlogic.CloudQueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	req.Provider = provider
	list, err := h.hostService.QueryCloudInstances(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

func (h *Handler) ImportCloudInstances(c *gin.Context) {
	provider := c.Param("provider")
	var req hostlogic.CloudImportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	req.Provider = provider
	task, nodes, err := h.hostService.ImportCloudInstances(c.Request.Context(), getUID(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"task": task, "created": nodes})
}

func (h *Handler) GetCloudImportTask(c *gin.Context) {
	task, err := h.hostService.GetImportTask(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "task not found")
		return
	}
	httpx.OK(c, task)
}
