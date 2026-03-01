package handler

import (
	"strconv"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	hostlogic "github.com/cy77cc/k8s-manage/internal/service/host/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListSSHKeys(c *gin.Context) {
	list, err := h.hostService.ListSSHKeys(c.Request.Context())
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"list": list, "total": len(list)})
}

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
