// Package host 提供主机管理服务的路由注册。
//
// 本文件注册主机相关的 HTTP 路由，包括：
//   - 主机 CRUD 操作
//   - SSH 连接和命令执行
//   - 文件管理
//   - 云主机导入
//   - KVM 虚拟化
//   - SSH 密钥管理
package host

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/middleware"
	"github.com/cy77cc/OpsPilot/internal/service/host/handler"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// RegisterHostHandlers 注册主机服务路由到 v1 组。
func RegisterHostHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := handler.NewHandler(svcCtx)
	h.StartHealthCollector()

	// 主机管理路由
	g := v1.Group("/hosts", middleware.JWTAuth())
	{
		// 主机来源和云账号
		g.GET("/sources", func(c *gin.Context) {
			httpx.OK(c, []string{"manual_ssh", "cloud_import", "kvm_provision"})
		})
		g.GET("/cloud/providers", h.ListCloudProviders)
		g.GET("/cloud/accounts", h.ListCloudAccounts)
		g.POST("/cloud/accounts", h.CreateCloudAccount)
		g.DELETE("/cloud/accounts/:id", h.DeleteCloudAccount)
		g.POST("/cloud/providers/:provider/accounts/test", h.TestCloudAccount)
		g.GET("/cloud/providers/:provider/regions", h.ListCloudRegions)
		g.GET("/cloud/providers/:provider/zones", h.ListCloudZones)
		g.POST("/cloud/providers/:provider/instances/query", h.QueryCloudInstances)
		g.POST("/cloud/providers/:provider/instances/import", h.ImportCloudInstances)
		g.GET("/cloud/import_tasks/:task_id", h.GetCloudImportTask)

		// KVM 虚拟化
		g.POST("/virtualization/kvm/hosts/:id/preview", h.KVMPreview)
		g.POST("/virtualization/kvm/hosts/:id/provision", h.KVMProvision)
		g.GET("/virtualization/tasks/:task_id", h.GetVirtualizationTask)

		// 主机 CRUD
		g.GET("", h.List)
		g.POST("/probe", h.Probe)
		g.POST("", h.Create)
		g.POST("/batch", h.Batch)
		g.POST("/batch/exec", h.BatchExec)
		g.GET("/:id", h.Get)
		g.PUT("/:id", h.Update)
		g.PUT("/:id/credentials", h.UpdateCredentials)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/actions", h.Action)

		// SSH 操作
		g.POST("/:id/health/check", h.HealthCheck)
		g.POST("/:id/ssh/check", h.SSHCheck)
		g.POST("/:id/ssh/exec", h.SSHExec)

		// 终端会话
		g.POST("/:id/terminal/sessions", h.CreateTerminalSession)
		g.GET("/:id/terminal/sessions/:session_id", h.GetTerminalSession)
		g.DELETE("/:id/terminal/sessions/:session_id", h.DeleteTerminalSession)
		g.GET("/:id/terminal/sessions/:session_id/ws", h.TerminalWebsocket)

		// 文件管理
		g.GET("/:id/files", h.ListFiles)
		g.GET("/:id/files/content", h.ReadFileContent)
		g.PUT("/:id/files/content", h.WriteFileContent)
		g.POST("/:id/files/upload", h.UploadFile)
		g.GET("/:id/files/download", h.DownloadFile)
		g.POST("/:id/files/mkdir", h.MakeDir)
		g.POST("/:id/files/rename", h.RenamePath)
		g.DELETE("/:id/files", h.DeletePath)

		// 其他
		g.GET("/:id/facts", h.Facts)
		g.GET("/:id/tags", h.Tags)
		g.POST("/:id/tags", h.AddTag)
		g.DELETE("/:id/tags/:tag", h.RemoveTag)
		g.GET("/:id/metrics", h.Metrics)
		g.GET("/:id/audits", h.Audits)
	}

	// SSH 密钥管理路由
	cred := v1.Group("/credentials", middleware.JWTAuth())
	{
		cred.GET("/ssh_keys", h.ListSSHKeys)
		cred.POST("/ssh_keys", h.CreateSSHKey)
		cred.DELETE("/ssh_keys/:id", h.DeleteSSHKey)
		cred.POST("/ssh_keys/:id/verify", h.VerifySSHKey)
	}
}
