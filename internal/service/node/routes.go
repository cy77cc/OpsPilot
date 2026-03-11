// Package node 提供节点管理服务的路由注册（已废弃）。
//
// 本文件注册节点相关的 HTTP 路由，此服务已被 host 服务替代。
// 所有响应包含 Deprecation 头指示废弃日期。
package node

import (
	"github.com/cy77cc/OpsPilot/internal/middleware"
	hostlogic "github.com/cy77cc/OpsPilot/internal/service/host/logic"
	"github.com/cy77cc/OpsPilot/internal/service/node/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterNodeHandlers 注册节点服务路由（已废弃）。
func RegisterNodeHandlers(r *gin.RouterGroup, serverCtx *svc.ServiceContext) {
	g := r.Group("node", middleware.JWTAuth())
	h := handler.NewNodeHandler(serverCtx)
	g.Use(func(c *gin.Context) {
		c.Header("Deprecation", "true")
		c.Header("Sunset", hostlogic.NodeSunsetDateRFC)
		c.Next()
	})
	// Add Node permission check
	g.POST("add", middleware.CasbinAuth(serverCtx.CasbinEnforcer, "node:add"), h.Add)
}
