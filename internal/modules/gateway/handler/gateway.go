package handler

import (
	"net/http"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// GatewayHandler 提供跳板机相关 HTTP 接口。
type GatewayHandler struct {
	svcCtx *svc.ServiceContext
}

// NewGatewayHandler 创建 GatewayHandler 实例。
func NewGatewayHandler(svcCtx *svc.ServiceContext) *GatewayHandler {
	return &GatewayHandler{svcCtx: svcCtx}
}

// ListGateways 返回可用的跳板机主机列表。
// GET /api/v1/hosts/gateways
func (h *GatewayHandler) ListGateways(c *gin.Context) {
	type gatewayInfo struct {
		ID            uint64 `json:"id"`
		Name          string `json:"name"`
		Hostname      string `json:"hostname"`
		IP            string `json:"ip"`
		ActiveTunnels int    `json:"active_tunnels"`
		MaxTunnels    int    `json:"max_tunnels"`
	}

	var results []gatewayInfo
	db := h.svcCtx.DB

	err := db.Raw(`
		SELECT n.id, n.name, n.hostname, n.ip
		FROM nodes n
		JOIN host_plugin_instances hpi ON hpi.host_id = n.id
		JOIN host_plugins hp ON hp.id = hpi.plugin_id
		WHERE hp.plugin_key = 'opsagent'
		  AND hpi.install_status = 'succeeded'
		  AND hpi.runtime_status = 'online'
		  AND hpi.capabilities_json LIKE '%gateway%'
	`).Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fill tunnel counts from TunnelManager
	if h.svcCtx.TunnelManager != nil {
		if tm, ok := h.svcCtx.TunnelManager.(interface{ Count() int }); ok {
			count := tm.Count()
			for i := range results {
				results[i].ActiveTunnels = count
				results[i].MaxTunnels = 100
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
