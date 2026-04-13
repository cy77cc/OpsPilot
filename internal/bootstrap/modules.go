package bootstrap

import (
	"context"
	"net/http"

	aiapi "github.com/cy77cc/OpsPilot/internal/modules/ai/api"
	aibootstrap "github.com/cy77cc/OpsPilot/internal/modules/ai/bootstrap"
	appapi "github.com/cy77cc/OpsPilot/internal/modules/application/api"
	automationapi "github.com/cy77cc/OpsPilot/internal/modules/automation/api"
	cicdapi "github.com/cy77cc/OpsPilot/internal/modules/cicd/api"
	clusterapi "github.com/cy77cc/OpsPilot/internal/modules/cluster/api"
	cmdbapi "github.com/cy77cc/OpsPilot/internal/modules/cmdb/api"
	dashboardapi "github.com/cy77cc/OpsPilot/internal/modules/dashboard/api"
	deploymentapi "github.com/cy77cc/OpsPilot/internal/modules/deployment/api"
	hostapi "github.com/cy77cc/OpsPilot/internal/modules/host/api"
	jobsapi "github.com/cy77cc/OpsPilot/internal/modules/jobs/api"
	monitoringapi "github.com/cy77cc/OpsPilot/internal/modules/monitoring/api"
	notificationapi "github.com/cy77cc/OpsPilot/internal/modules/notification/api"
	projectapi "github.com/cy77cc/OpsPilot/internal/modules/project/api"
	rbacapi "github.com/cy77cc/OpsPilot/internal/modules/rbac/api"
	topologyapi "github.com/cy77cc/OpsPilot/internal/modules/topology/api"
	userapi "github.com/cy77cc/OpsPilot/internal/modules/user/api"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/cy77cc/OpsPilot/internal/websocket"
	"github.com/gin-gonic/gin"
)

// RegisterModules wires all HTTP modules into the shared router.
func RegisterModules(appCtx *svc.ServiceContext, engine *gin.Engine) {
	engine.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/api/v1")
	userapi.RegisterUserHandlers(v1, appCtx)
	aibootstrap.StartBackgroundProcessors(context.Background(), appCtx)
	aiapi.RegisterAIHandlers(v1, appCtx)
	aiapi.RegisterAdminAIHandlers(v1, appCtx)
	projectapi.RegisterProjectHandlers(v1, appCtx)
	appapi.RegisterServiceHandlers(v1, appCtx)
	cicdapi.RegisterCICDHandlers(v1, appCtx)
	automationapi.RegisterAutomationHandlers(v1, appCtx)
	hostapi.RegisterHostHandlers(v1, appCtx)
	clusterapi.RegisterClusterHandlers(v1, appCtx)
	deploymentapi.RegisterDeploymentHandlers(v1, appCtx)
	monitoringapi.RegisterMonitoringHandlers(v1, appCtx)
	dashboardapi.RegisterDashboardHandlers(v1, appCtx)
	cmdbapi.RegisterCMDBHandlers(v1, appCtx)
	topologyapi.RegisterTopologyHandlers(v1, appCtx)
	rbacapi.RegisterRBACHandlers(v1, appCtx)
	notificationapi.RegisterNotificationHandlers(v1, appCtx)
	jobsapi.RegisterJobsHandlers(v1, appCtx)

	engine.GET("/ws/notifications", websocket.HandleWebSocket)
}
