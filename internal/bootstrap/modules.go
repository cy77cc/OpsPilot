package bootstrap

import (
	"context"
	"net/http"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/middleware"
	aiapi "github.com/cy77cc/OpsPilot/internal/modules/ai/api"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/infra/workers"
	ailogic "github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	appapi "github.com/cy77cc/OpsPilot/internal/modules/application/api"
	automationapi "github.com/cy77cc/OpsPilot/internal/modules/automation/api"
	cicdapi "github.com/cy77cc/OpsPilot/internal/modules/cicd/api"
	clusterapi "github.com/cy77cc/OpsPilot/internal/modules/cluster/api"
	cmdbapi "github.com/cy77cc/OpsPilot/internal/modules/cmdb/api"
	dashboardapi "github.com/cy77cc/OpsPilot/internal/modules/dashboard/api"
	deploymentapi "github.com/cy77cc/OpsPilot/internal/modules/deployment/api"
	hostapi "github.com/cy77cc/OpsPilot/internal/modules/host/api"
	jobsapi "github.com/cy77cc/OpsPilot/internal/modules/jobs/api"
	llmproviderapi "github.com/cy77cc/OpsPilot/internal/modules/llmprovider/api"
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

const aiBackgroundWorkerTick = 2 * time.Second

// RegisterModules wires all HTTP modules into the shared router.
func RegisterModules(ctx context.Context, appCtx *svc.ServiceContext, engine *gin.Engine) {
	engine.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	v1 := engine.Group("/api/v1")
	userapi.RegisterUserHandlers(v1, appCtx)
	if appCtx != nil && appCtx.DB != nil {
		ai := ailogic.NewAILogic(appCtx)
		approvalWorker := ailogic.NewApprovalWorker(ai)
		expirer := ailogic.NewApprovalExpirer(ai)
		_ = workers.NewRunner(func(runCtx context.Context) {
			for runCtx.Err() == nil {
				claimed, _ := approvalWorker.RunOnce(runCtx)
				if !claimed {
					return
				}
			}
		}, aiBackgroundWorkerTick).Start(ctx)
		_ = workers.NewRunner(func(runCtx context.Context) {
			for runCtx.Err() == nil {
				claimed, _ := expirer.RunOnce(runCtx)
				if !claimed {
					return
				}
			}
		}, aiBackgroundWorkerTick).Start(ctx)
	}
	aiapi.RegisterAIHandlers(v1, appCtx)
	llmproviderapi.RegisterAdminAIModelRoutes(v1, appCtx)
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

	ws := engine.Group("/ws", middleware.JWTAuth())
	ws.GET("/notifications", websocket.HandleWebSocket)
}
