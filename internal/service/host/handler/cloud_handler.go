// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/service/host/logic/cloud"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// ListCloudProviders 列出所有已注册的云厂商。
//
// @Summary 列出云厂商
// @Description 获取所有已注册的云厂商信息，用于前端云账号配置下拉选项
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Router /hosts/cloud/providers [get]
func (h *Handler) ListCloudProviders(c *gin.Context) {
	providers := cloud.ListProviders()
	httpx.OK(c, providers)
}

// ListCloudAccounts 列出云账号。
//
// @Summary 列出云账号
// @Description 获取所有云账号列表，支持按云厂商过滤，AccessKey 已脱敏
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param provider query string false "云厂商标识过滤"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/accounts [get]
func (h *Handler) ListCloudAccounts(c *gin.Context) {
	provider := c.Query("provider")
	list, err := h.hostService.ListCloudAccounts(c.Request.Context(), provider)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// CreateCloudAccount 创建云账号。
//
// @Summary 创建云账号
// @Description 创建云厂商账号，AccessKey Secret 将被加密存储
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body hostlogic.CloudAccountReq true "云账号创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/accounts [post]
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

// DeleteCloudAccount 删除云账号。
//
// @Summary 删除云账号
// @Description 删除指定的云厂商账号
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "云账号 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/accounts/{id} [delete]
func (h *Handler) DeleteCloudAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		httpx.Fail(c, xcode.ParamError, "账号ID不能为空")
		return
	}
	var id uint64
	if _, err := fmt.Sscanf(accountID, "%d", &id); err != nil {
		httpx.Fail(c, xcode.ParamError, "账号ID格式错误")
		return
	}
	if err := h.hostService.DeleteCloudAccount(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// TestCloudAccount 测试云账号凭证。
//
// @Summary 测试云账号
// @Description 验证云账号的 AccessKey 是否有效
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param provider path string true "云厂商标识"
// @Param body body hostlogic.CloudAccountReq true "云账号测试请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/providers/{provider}/accounts/test [post]
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

// QueryCloudInstances 查询云实例列表。
//
// @Summary 查询云实例
// @Description 查询指定云账号和地域下的云主机实例列表
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param provider path string true "云厂商标识"
// @Param body body hostlogic.CloudQueryReq true "查询请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/providers/{provider}/instances/query [post]
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

// ImportCloudInstances 导入云实例。
//
// @Summary 导入云实例
// @Description 将选中的云主机实例导入到平台，创建对应的主机记录
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param provider path string true "云厂商标识"
// @Param body body hostlogic.CloudImportReq true "导入请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/providers/{provider}/instances/import [post]
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

// GetCloudImportTask 获取云主机导入任务状态。
//
// @Summary 获取导入任务
// @Description 获取云主机导入任务的状态和结果
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param task_id path string true "任务 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Router /hosts/cloud/import_tasks/{task_id} [get]
func (h *Handler) GetCloudImportTask(c *gin.Context) {
	task, err := h.hostService.GetImportTask(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "task not found")
		return
	}
	httpx.OK(c, task)
}

// ListCloudRegions 列出云厂商支持的地域。
//
// @Summary 列出云地域
// @Description 获取指定云账号支持的地域列表
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param provider path string true "云厂商标识"
// @Param account_id query int true "云账号 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/providers/{provider}/regions [get]
func (h *Handler) ListCloudRegions(c *gin.Context) {
	provider := c.Param("provider")
	accountID := c.Query("account_id")
	if accountID == "" {
		httpx.Fail(c, xcode.ParamError, "account_id is required")
		return
	}

	var id uint64
	if _, err := fmt.Sscanf(accountID, "%d", &id); err != nil {
		httpx.Fail(c, xcode.ParamError, "account_id format error")
		return
	}

	regions, err := h.hostService.ListCloudRegions(c.Request.Context(), provider, id)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, regions)
}

// ListCloudZones 列出云厂商指定地域的可用区。
//
// @Summary 列出云可用区
// @Description 获取指定云账号和地域下的可用区列表
// @Tags 云主机导入
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param provider path string true "云厂商标识"
// @Param account_id query int true "云账号 ID"
// @Param region query string false "地域标识"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/cloud/providers/{provider}/zones [get]
func (h *Handler) ListCloudZones(c *gin.Context) {
	provider := c.Param("provider")
	accountID := c.Query("account_id")
	region := c.Query("region")

	if accountID == "" {
		httpx.Fail(c, xcode.ParamError, "account_id is required")
		return
	}

	var id uint64
	if _, err := fmt.Sscanf(accountID, "%d", &id); err != nil {
		httpx.Fail(c, xcode.ParamError, "account_id format error")
		return
	}

	zones, err := h.hostService.ListCloudZones(c.Request.Context(), provider, id, region)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, zones)
}
