package handler

import (
	"strconv"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	topologylogic "github.com/cy77cc/OpsPilot/internal/service/topology/logic"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/xcode"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	logic  *topologylogic.Logic
	svcCtx *svc.ServiceContext
}

func NewHandler(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{logic: topologylogic.NewLogic(svcCtx), svcCtx: svcCtx}
}

func (h *Handler) ServiceTopology(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "topology:read", "cmdb:read", "cmdb:*") {
		return
	}
	serviceID := uint(atoiDefault(c.Param("id"), 0))
	out, err := h.logic.GetServiceTopology(c.Request.Context(), serviceID)
	if err != nil {
		httpx.Fail(c, xcode.NotFound, "service not found")
		return
	}
	h.logic.WriteAccessAudit(c.Request.Context(), uint(httpx.UIDFromCtx(c)), "topology.service", "service", map[string]any{"service_id": serviceID})
	httpx.OK(c, out)
}

func (h *Handler) HostServices(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "topology:read", "cmdb:read", "cmdb:*") {
		return
	}
	hostID := uint(atoiDefault(c.Param("id"), 0))
	out, err := h.logic.GetHostServices(c.Request.Context(), hostID)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.logic.WriteAccessAudit(c.Request.Context(), uint(httpx.UIDFromCtx(c)), "topology.host.services", "host", map[string]any{"host_id": hostID})
	httpx.OK(c, gin.H{"data": out, "total": len(out)})
}

func (h *Handler) ClusterServices(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "topology:read", "cmdb:read", "cmdb:*") {
		return
	}
	clusterID := uint(atoiDefault(c.Param("id"), 0))
	out, err := h.logic.GetClusterServices(c.Request.Context(), clusterID)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.logic.WriteAccessAudit(c.Request.Context(), uint(httpx.UIDFromCtx(c)), "topology.cluster.services", "cluster", map[string]any{"cluster_id": clusterID})
	httpx.OK(c, gin.H{"data": out, "total": len(out)})
}

func (h *Handler) Graph(c *gin.Context) {
	if !httpx.Authorize(c, h.svcCtx.DB, "topology:read", "cmdb:read", "cmdb:*") {
		return
	}
	filter := topologylogic.QueryFilter{
		ProjectID:    uint(atoiDefault(c.Query("project_id"), 0)),
		ClusterID:    uint(atoiDefault(c.Query("cluster_id"), 0)),
		ResourceType: strings.TrimSpace(c.Query("resource_type")),
		Keyword:      strings.TrimSpace(c.Query("keyword")),
	}
	out, err := h.logic.QueryGraph(c.Request.Context(), filter)
	if err != nil {
		httpx.Fail(c, xcode.ServerError, err.Error())
		return
	}
	h.logic.WriteAccessAudit(c.Request.Context(), uint(httpx.UIDFromCtx(c)), "topology.graph.query", "graph", filter)
	httpx.OK(c, out)
}

func atoiDefault(s string, d int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return d
	}
	return n
}
