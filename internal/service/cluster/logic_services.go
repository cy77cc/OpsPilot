// Package cluster 提供 Kubernetes 集群管理服务的核心业务逻辑。
//
// 本文件实现集群服务相关的 HTTP Handler，处理已部署服务的查询请求。
package cluster

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
)

// ClusterServiceInfo 集群服务信息响应结构。
type ClusterServiceInfo struct {
	ID           uint   `json:"id"`            // 服务 ID
	Name         string `json:"name"`          // 服务名称
	ProjectName  string `json:"project_name"`  // 项目名称
	TeamName     string `json:"team_name"`     // 团队名称
	Env          string `json:"env"`           // 环境标识
	LastDeployAt string `json:"last_deploy_at"` // 最后部署时间
	Status       string `json:"status"`        // 部署状态
}

// GetClusterServices 获取集群已部署的服务列表。
//
// @Summary 获取集群服务列表
// @Description 获取部署到指定集群的所有服务信息
// @Tags 集群管理
// @Accept json
// @Produce json
// @Param id path int true "集群 ID"
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} httpx.Response{data=map[string]interface{}}
// @Failure 400 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /clusters/{id}/services [get]
func (h *Handler) GetClusterServices(c *gin.Context) {
	id := httpx.UintFromParam(c, "id")
	if id == 0 {
		httpx.BindErr(c, nil)
		return
	}

	// Query deployment targets linked to this cluster
	var targets []model.DeploymentTarget
	if err := h.svcCtx.DB.WithContext(c.Request.Context()).
		Where("cluster_id = ?", id).
		Find(&targets).Error; err != nil {
		httpx.ServerErr(c, err)
		return
	}

	// Query services that have releases to this cluster
	items := make([]ClusterServiceInfo, 0)
	for _, target := range targets {
		// Get latest release for this target
		var release model.DeploymentRelease
		if err := h.svcCtx.DB.WithContext(c.Request.Context()).
			Where("target_id = ?", target.ID).
			Order("id DESC").
			First(&release).Error; err == nil {

			// Get service info
			var service model.Service
			if err := h.svcCtx.DB.WithContext(c.Request.Context()).
				First(&service, release.ServiceID).Error; err == nil {

				// Get project info
				projectName := ""
				if target.ProjectID > 0 {
					var project model.Project
					if err := h.svcCtx.DB.WithContext(c.Request.Context()).
						First(&project, target.ProjectID).Error; err == nil {
						projectName = project.Name
					}
				}

				items = append(items, ClusterServiceInfo{
					ID:           service.ID,
					Name:         service.Name,
					ProjectName:  projectName,
					TeamName:     "",
					Env:          target.Env,
					LastDeployAt: release.CreatedAt.Format("2006-01-02 15:04:05"),
					Status:       release.Status,
				})
			}
		}
	}

	httpx.OK(c, gin.H{"list": items, "total": len(items)})
}
