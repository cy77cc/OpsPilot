// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/core/httpx/xcode"
	hostlogic "github.com/cy77cc/OpsPilot/internal/modules/host/logic"
	"github.com/gin-gonic/gin"
)

// ListUnifiedCredentials 获取统一凭证列表（包括密钥和模板）。
func (h *Handler) ListUnifiedCredentials(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	list, err := h.hostService.ListUnifiedCredentials(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// GetCredentialStats 获取凭证统计信息。
func (h *Handler) GetCredentialStats(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	stats, err := h.hostService.GetCredentialStats(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, stats)
}

// ListCredentialUsageRecords 获取凭证使用记录。
func (h *Handler) ListCredentialUsageRecords(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	// TODO: 从数据库读取真实的使用记录
	httpx.OK(c, gin.H{"list": []any{}, "total": 0})
}

// ListCredentialPermissions 获取凭证权限记录。
func (h *Handler) ListCredentialPermissions(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	// TODO: 从数据库读取真实的权限记录
	httpx.OK(c, gin.H{"list": []any{}, "total": 0})
}

// GetCredential 获取凭证详情。
func (h *Handler) GetCredential(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	detail, err := h.hostService.GetCredentialDetail(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, detail)
}

// GetCredentialUsageStats 获取凭证使用统计。
func (h *Handler) GetCredentialUsageStats(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	// TODO: 实现真实的统计逻辑
	httpx.OK(c, gin.H{
		"total":       0,
		"success":     0,
		"failed":      0,
		"successRate": 0,
	})
}

// ListSSHKeys 获取 SSH 密钥列表。
//
// @Summary 获取 SSH 密钥列表
// @Description 获取所有 SSH 密钥信息（私钥已脱敏）
// @Tags SSH 密钥管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/ssh_keys [get]
func (h *Handler) ListSSHKeys(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	list, err := h.hostService.ListSSHKeys(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// CreateSSHKey 创建 SSH 密钥。
//
// @Summary 创建 SSH 密钥
// @Description 上传 SSH 私钥并保存，私钥将被加密存储
// @Tags SSH 密钥管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body hostlogic.SSHKeyCreateReq true "密钥创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/ssh_keys [post]
func (h *Handler) CreateSSHKey(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:write", "host:credential:*", "host:*") {
		return
	}
	var req hostlogic.SSHKeyCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	item, err := h.hostService.CreateSSHKey(c.Request.Context(), req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, item)
}

// DeleteSSHKey 删除 SSH 密钥。
//
// @Summary 删除 SSH 密钥
// @Description 删除指定的 SSH 密钥，已被主机引用的密钥无法删除
// @Tags SSH 密钥管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "密钥 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/ssh_keys/{id} [delete]
func (h *Handler) DeleteSSHKey(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:write", "host:credential:*", "host:*") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.hostService.DeleteSSHKey(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

// VerifySSHKey 验证 SSH 密钥。
//
// @Summary 验证 SSH 密钥
// @Description 验证 SSH 密钥是否可以连接到指定主机
// @Tags SSH 密钥管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "密钥 ID"
// @Param body body hostlogic.SSHKeyVerifyReq true "验证请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/ssh_keys/{id}/verify [post]
func (h *Handler) VerifySSHKey(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:write", "host:credential:*", "host:*") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	var req hostlogic.SSHKeyVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	result, err := h.hostService.VerifySSHKey(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, result)
}

// ListCredentialTemplates 获取认证预设列表。
//
// @Summary 获取认证预设列表
// @Description 获取所有 SSH 认证预设模板（密码已脱敏）
// @Tags 认证预设管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/templates [get]
func (h *Handler) ListCredentialTemplates(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:read", "host:credential:*", "host:*") {
		return
	}
	list, err := h.hostService.ListCredentialTemplates(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

// CreateCredentialTemplate 创建认证预设。
//
// @Summary 创建认证预设
// @Description 创建 SSH 认证预设模板，密码将被加密存储
// @Tags 认证预设管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body hostlogic.CredentialTemplateCreateReq true "预设创建请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/templates [post]
func (h *Handler) CreateCredentialTemplate(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:write", "host:credential:*", "host:*") {
		return
	}
	var req hostlogic.CredentialTemplateCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	uid := getUID(c)
	item, err := h.hostService.CreateCredentialTemplate(c.Request.Context(), uid, req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, item)
}

// DeleteCredentialTemplate 删除认证预设。
//
// @Summary 删除认证预设
// @Description 删除指定的 SSH 认证预设模板
// @Tags 认证预设管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "预设 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /credentials/templates/{id} [delete]
func (h *Handler) DeleteCredentialTemplate(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:credential:write", "host:credential:*", "host:*") {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, "invalid id")
		return
	}
	if err := h.hostService.DeleteCredentialTemplate(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, nil)
}
