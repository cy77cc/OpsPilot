package handler

import (
	"strings"

	"github.com/cy77cc/k8s-manage/internal/httpx"
	hostlogic "github.com/cy77cc/k8s-manage/internal/service/host/logic"
	"github.com/cy77cc/k8s-manage/internal/xcode"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Probe(c *gin.Context) {
	var req hostlogic.ProbeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	resp, err := h.hostService.Probe(c.Request.Context(), getUID(c), req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, resp)
}

func (h *Handler) Create(c *gin.Context) {
	var req hostlogic.CreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	uid := getUID(c)
	node, err := h.hostService.CreateWithProbe(c.Request.Context(), uid, httpx.IsAdmin(h.svcCtx.DB, uid), req)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "probe_expired") || strings.Contains(msg, "probe_not_found") {
			httpx.Fail(c, xcode.ParamError, msg)
		} else {
			httpx.Fail(c, xcode.ParamError, msg)
		}
		return
	}
	httpx.OK(c, node)
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	node, err := h.hostService.Update(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, node)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.hostService.Delete(c.Request.Context(), id); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) Action(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	_ = c.ShouldBindJSON(&req)
	if err := h.hostService.UpdateStatus(c.Request.Context(), id, req.Action); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"id": id, "action": req.Action})
}

func (h *Handler) Batch(c *gin.Context) {
	var req struct {
		HostIDs []uint64 `json:"host_ids"`
		Action  string   `json:"action"`
		Tags    []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	if err := h.hostService.BatchUpdateStatus(c.Request.Context(), req.HostIDs, req.Action); err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) AddTag(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Tag string `json:"tag" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	labels := hostlogic.ParseLabels(node.Labels)
	labels = append(labels, req.Tag)
	_, err = h.hostService.Update(c.Request.Context(), id, map[string]any{"labels": hostlogic.EncodeLabels(labels)})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) RemoveTag(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	tag := c.Param("tag")
	node, err := h.hostService.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "host not found")
		return
	}
	labels := hostlogic.ParseLabels(node.Labels)
	filtered := make([]string, 0, len(labels))
	for _, item := range labels {
		if item != tag {
			filtered = append(filtered, item)
		}
	}
	_, err = h.hostService.Update(c.Request.Context(), id, map[string]any{"labels": hostlogic.EncodeLabels(filtered)})
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, nil)
}

func (h *Handler) UpdateCredentials(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req hostlogic.UpdateCredentialsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	node, probeResp, err := h.hostService.UpdateCredentials(c.Request.Context(), id, req)
	if err != nil {
		httpx.Fail(c, xcode.ParamError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"node": node, "probe": probeResp})
}
