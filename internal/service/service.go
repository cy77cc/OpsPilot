package service

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/service/ai"
	"github.com/cy77cc/OpsPilot/internal/service/automation"
	"github.com/cy77cc/OpsPilot/internal/service/ai"
	"github.com/cy77cc/OpsPilot/internal/service/cicd"
	"github.com/cy77cc/OpsPilot/internal/service/cluster"
	"github.com/cy77cc/OpsPilot/internal/service/cmdb"
	"github.com/cy77cc/OpsPilot/internal/service/dashboard"
	"github.com/cy77cc/OpsPilot/internal/service/deployment"
	"github.com/cy77cc/OpsPilot/internal/service/host"
	"github.com/cy77cc/OpsPilot/internal/service/jobs"
	"github.com/cy77cc/OpsPilot/internal/service/monitoring"
	"github.com/cy77cc/OpsPilot/internal/service/node"
	"github.com/cy77cc/OpsPilot/internal/service/notification"
	"github.com/cy77cc/OpsPilot/internal/service/project"
	"github.com/cy77cc/OpsPilot/internal/service/rbac"
	servicemgr "github.com/cy77cc/OpsPilot/internal/service/service"
	"github.com/cy77cc/OpsPilot/internal/service/topology"
	"github.com/cy77cc/OpsPilot/internal/service/user"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/websocket"
	webui "github.com/cy77cc/OpsPilot/web"
	"github.com/gin-gonic/gin"
)

func Init(r *gin.Engine, serverCtx *svc.ServiceContext) {
	r.Use(gin.Recovery(), middleware.ContextMiddleware(), middleware.Cors(), middleware.Logger())
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	user.RegisterUserHandlers(v1, serverCtx)
	ai.RegisterAIHandlers(v1, serverCtx)
	node.RegisterNodeHandlers(v1, serverCtx)
	project.RegisterProjectHandlers(v1, serverCtx)
	servicemgr.RegisterServiceHandlers(v1, serverCtx)
	cicd.RegisterCICDHandlers(v1, serverCtx)
	ai.RegisterAIHandlers(v1, serverCtx)
	automation.RegisterAutomationHandlers(v1, serverCtx)
	host.RegisterHostHandlers(v1, serverCtx)
	cluster.RegisterClusterHandlers(v1, serverCtx)
	deployment.RegisterDeploymentHandlers(v1, serverCtx)
	monitoring.RegisterMonitoringHandlers(v1, serverCtx)
	dashboard.RegisterDashboardHandlers(v1, serverCtx)
	cmdb.RegisterCMDBHandlers(v1, serverCtx)
	topology.RegisterTopologyHandlers(v1, serverCtx)
	rbac.RegisterRBACHandlers(v1, serverCtx)
	notification.RegisterNotificationHandlers(v1, serverCtx)
	jobs.RegisterJobsHandlers(v1, serverCtx)

	// WebSocket 路由
	r.GET("/ws/notifications", websocket.HandleWebSocket)

	registerWebStaticRoutes(r)
}

func registerWebStaticRoutes(r *gin.Engine) {
	if config.IsDevelopment() {
		return
	}

	distFS, err := webui.SubDist()
	if err != nil {
		return
	}

	staticServer := http.FileServer(http.FS(distFS))
	r.GET("/assets/*filepath", gin.WrapH(staticServer))
	r.GET("/vite.svg", gin.WrapH(staticServer))
	r.GET("/favicon.ico", gin.WrapH(staticServer))

	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		indexFile, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "frontend not built"})
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexFile)
	})
}
