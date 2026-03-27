// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

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
