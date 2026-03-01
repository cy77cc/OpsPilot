package host

import (
	"github.com/cy77cc/k8s-manage/internal/httpx"
	"github.com/cy77cc/k8s-manage/internal/middleware"
	"github.com/cy77cc/k8s-manage/internal/service/host/handler"
	"github.com/cy77cc/k8s-manage/internal/svc"
	"github.com/gin-gonic/gin"
)

func RegisterHostHandlers(v1 *gin.RouterGroup, svcCtx *svc.ServiceContext) {
	h := handler.NewHandler(svcCtx)
	g := v1.Group("/hosts", middleware.JWTAuth())
	{
		g.GET("/sources", func(c *gin.Context) {
			httpx.OK(c, []string{"manual_ssh", "cloud_import", "kvm_provision"})
		})
		g.GET("/cloud/accounts", h.ListCloudAccounts)
		g.POST("/cloud/accounts", h.CreateCloudAccount)
		g.POST("/cloud/providers/:provider/accounts/test", h.TestCloudAccount)
		g.POST("/cloud/providers/:provider/instances/query", h.QueryCloudInstances)
		g.POST("/cloud/providers/:provider/instances/import", h.ImportCloudInstances)
		g.GET("/cloud/import_tasks/:task_id", h.GetCloudImportTask)
		g.POST("/virtualization/kvm/hosts/:id/preview", h.KVMPreview)
		g.POST("/virtualization/kvm/hosts/:id/provision", h.KVMProvision)
		g.GET("/virtualization/tasks/:task_id", h.GetVirtualizationTask)
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
		g.POST("/:id/ssh/check", h.SSHCheck)
		g.POST("/:id/ssh/exec", h.SSHExec)
		g.POST("/:id/terminal/sessions", h.CreateTerminalSession)
		g.GET("/:id/terminal/sessions/:session_id", h.GetTerminalSession)
		g.DELETE("/:id/terminal/sessions/:session_id", h.DeleteTerminalSession)
		g.GET("/:id/terminal/sessions/:session_id/ws", h.TerminalWebsocket)
		g.GET("/:id/files", h.ListFiles)
		g.GET("/:id/files/content", h.ReadFileContent)
		g.PUT("/:id/files/content", h.WriteFileContent)
		g.POST("/:id/files/upload", h.UploadFile)
		g.GET("/:id/files/download", h.DownloadFile)
		g.POST("/:id/files/mkdir", h.MakeDir)
		g.POST("/:id/files/rename", h.RenamePath)
		g.DELETE("/:id/files", h.DeletePath)
		g.GET("/:id/facts", h.Facts)
		g.GET("/:id/tags", h.Tags)
		g.POST("/:id/tags", h.AddTag)
		g.DELETE("/:id/tags/:tag", h.RemoveTag)
		g.GET("/:id/metrics", h.Metrics)
		g.GET("/:id/audits", h.Audits)
	}

	cred := v1.Group("/credentials", middleware.JWTAuth())
	{
		cred.GET("/ssh_keys", h.ListSSHKeys)
		cred.POST("/ssh_keys", h.CreateSSHKey)
		cred.DELETE("/ssh_keys/:id", h.DeleteSSHKey)
		cred.POST("/ssh_keys/:id/verify", h.VerifySSHKey)
	}
}
