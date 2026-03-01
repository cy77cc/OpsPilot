package deployment

import (
	"strings"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/gin-gonic/gin"
)

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
