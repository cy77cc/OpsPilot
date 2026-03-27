// Package handler 提供主机管理服务的 HTTP 处理器。
package handler

import (
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

// Probe 探测主机 SSH 连接。
//
// @Summary 探测主机连接
// @Description 探测主机的 SSH 连接是否可达，收集系统信息（操作系统、架构、内核、CPU、内存、磁盘）
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body hostlogic.ProbeReq true "探测请求参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/probe [post]
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

// Create 创建主机。
//
// @Summary 创建主机
// @Description 通过探测令牌或手动输入创建主机记录，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body hostlogic.CreateReq true "创建请求参数"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts [post]
func (h *Handler) Create(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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

// Update 更新主机信息。
//
// @Summary 更新主机
// @Description 更新主机的部分字段，支持 PATCH 语义，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body map[string]interface{} true "更新字段"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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

// Delete 删除主机。
//
// @Summary 删除主机
// @Description 删除指定的主机记录，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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

// Action 执行主机状态变更操作。
//
// @Summary 执行主机操作
// @Description 执行主机状态变更操作（如进入/退出维护模式），需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body object true "操作请求 {action: 'maintenance'|'online'|'offline', reason: string, until: datetime}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/actions [post]
func (h *Handler) Action(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req struct {
		Action string     `json:"action"`
		Reason string     `json:"reason"`
		Until  *time.Time `json:"until"`
	}
	_ = c.ShouldBindJSON(&req)
	var err error
	err = h.hostService.UpdateStatusWithMeta(c.Request.Context(), id, req.Action, req.Reason, req.Until, getUID(c))
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	httpx.OK(c, gin.H{"id": id, "action": req.Action, "reason": req.Reason, "until": req.Until})
}

// Batch 批量更新主机状态。
//
// @Summary 批量更新主机状态
// @Description 批量更新多个主机的状态，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param body body object true "批量请求 {host_ids: [], action: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/batch [post]
func (h *Handler) Batch(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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

// AddTag 为主机添加标签。
//
// @Summary 添加主机标签
// @Description 为指定主机添加一个标签，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body object true "标签请求 {tag: string}"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/tags [post]
func (h *Handler) AddTag(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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

// RemoveTag 移除主机标签。
//
// @Summary 移除主机标签
// @Description 移除指定主机的一个标签，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param tag path string true "标签名称"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 404 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/tags/{tag} [delete]
func (h *Handler) RemoveTag(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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

// UpdateCredentials 更新主机 SSH 凭证。
//
// @Summary 更新主机凭证
// @Description 更新主机的 SSH 凭证（用户名、密码或密钥），并验证新凭证是否有效，需要 host:write 或 host:* 权限
// @Tags 主机管理
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path int true "主机 ID"
// @Param body body hostlogic.UpdateCredentialsReq true "凭证更新请求"
// @Success 200 {object} httpx.Response
// @Failure 400 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 403 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /hosts/{id}/credentials [put]
func (h *Handler) UpdateCredentials(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "host:write", "host:*") {
		return
	}
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
