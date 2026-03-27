// Package deployment 提供部署管理服务的拓扑处理器。
//
// 本文件包含部署拓扑可视化的 HTTP 处理器实现。
package deployment

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

// TopologyHandler 是部署拓扑的 HTTP 处理器。
type TopologyHandler struct {
	svcCtx *svc.ServiceContext
}

// NewTopologyHandler 创建拓扑处理器实例。
//
// 参数:
//   - svcCtx: 服务上下文
//
// 返回: TopologyHandler 实例
func NewTopologyHandler(svcCtx *svc.ServiceContext) *TopologyHandler {
	return &TopologyHandler{svcCtx: svcCtx}
}

// TopologyService 是拓扑服务节点，表示一个部署目标。
type TopologyService struct {
	ID             uint   `json:"id"`                       // 服务 ID
	Name           string `json:"name"`                     // 服务名称
	Environment    string `json:"environment"`              // 环境
	Status         string `json:"status"`                   // 状态
	LastDeployment string `json:"last_deployment,omitempty"` // 最后部署时间
	TargetID       uint   `json:"target_id"`                // 目标 ID
	TargetName     string `json:"target_name,omitempty"`    // 目标名称
	RuntimeType    string `json:"runtime_type,omitempty"`   // 运行时类型
}

// TopologyConnection 是拓扑连接，表示服务之间的依赖关系。
type TopologyConnection struct {
	SourceID uint   `json:"source_id"` // 源服务 ID
	TargetID uint   `json:"target_id"` // 目标服务 ID
	Type     string `json:"type"`      // 连接类型
}

// DeploymentTopology 是部署拓扑结构，包含服务和连接信息。
type DeploymentTopology struct {
	Services    []TopologyService    `json:"services"`    // 服务列表
	Connections []TopologyConnection `json:"connections"` // 连接列表
}

// GetTopology 获取部署拓扑。
//
// @Summary 获取部署拓扑
// @Description 获取部署目标的拓扑结构，用于可视化展示
// @Tags 部署拓扑
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param environment query string false "环境筛选"
// @Success 200 {object} httpx.Response
// @Failure 401 {object} httpx.Response
// @Failure 500 {object} httpx.Response
// @Router /deploy/topology [get]
func (h *TopologyHandler) GetTopology(c *gin.Context) {
	ctx := c.Request.Context()
	env := c.Query("environment")

	topology, err := h.getTopology(ctx, env)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, topology)
}

// getTopology 查询部署拓扑数据。
//
// 参数:
//   - ctx: 上下文
//   - envFilter: 环境筛选条件
//
// 返回: 部署拓扑结构
func (h *TopologyHandler) getTopology(ctx context.Context, envFilter string) (*DeploymentTopology, error) {
	topology := &DeploymentTopology{
		Services:    []TopologyService{},
		Connections: []TopologyConnection{},
	}

	// 查询部署目标
	query := h.svcCtx.DB.WithContext(ctx).Model(&model.DeploymentTarget{})
	if envFilter != "" && envFilter != "all" {
		query = query.Where("env = ?", envFilter)
	}

	var targets []model.DeploymentTarget
	if err := query.Find(&targets).Error; err != nil {
		return nil, err
	}

	// 获取每个目标的最新发布
	for _, target := range targets {
		var latestRelease model.DeploymentRelease
		err := h.svcCtx.DB.WithContext(ctx).
			Model(&model.DeploymentRelease{}).
			Where("target_id = ?", target.ID).
			Order("created_at desc").
			First(&latestRelease).Error

		status := "unknown"
		lastDeployment := ""

		if err == nil {
			status = latestRelease.Status
			lastDeployment = latestRelease.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		} else {
			status = "no_deployments"
		}

		// 根据目标状态判断健康状态
		if target.ReadinessStatus != "" {
			status = target.ReadinessStatus
		}

		service := TopologyService{
			ID:             target.ID,
			Name:           target.Name,
			Environment:    target.Env,
			Status:         status,
			LastDeployment: lastDeployment,
			TargetID:       target.ID,
			TargetName:     target.Name,
			RuntimeType:    target.RuntimeType,
		}

		topology.Services = append(topology.Services, service)
	}

	return topology, nil
}
