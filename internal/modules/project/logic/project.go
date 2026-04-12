// Package logic 提供项目管理业务逻辑实现。
//
// 本文件实现项目的创建、查询和部署等核心业务逻辑。
package logic

import (
	"context"

	v1 "github.com/cy77cc/OpsPilot/api/project/v1"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// ProjectLogic 是项目管理的业务逻辑层。
//
// 职责:
//   - 处理项目的 CRUD 操作
//   - 协调服务和集群实现项目级部署
type ProjectLogic struct {
	svcCtx *svc.ServiceContext
}

// NewProjectLogic 创建项目逻辑层实例。
//
// 参数:
//   - svcCtx: 服务上下文，包含数据库连接等依赖
//
// 返回: 项目逻辑层实例
func NewProjectLogic(svcCtx *svc.ServiceContext) *ProjectLogic {
	return &ProjectLogic{
		svcCtx: svcCtx,
	}
}

// CreateProject 创建项目。
//
// 参数:
//   - ctx: 上下文
//   - req: 创建请求，包含名称和描述
//
// 返回: 创建的项目信息，失败返回错误
func (l *ProjectLogic) CreateProject(ctx context.Context, req v1.CreateProjectReq) (v1.ProjectResp, error) {
	project := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		// OwnerID:     ctx.Value("uid").(int64), // TODO: Get from context
	}

	if err := l.svcCtx.DB.Create(project).Error; err != nil {
		return v1.ProjectResp{}, err
	}

	return v1.ProjectResp{
		ID:          project.ID,
		Name:        project.Name,
		Description: project.Description,
		OwnerID:     project.OwnerID,
		CreatedAt:   project.CreatedAt,
	}, nil
}

// ListProjects 获取项目列表。
//
// 参数:
//   - ctx: 上下文
//
// 返回: 项目列表，失败返回错误
func (l *ProjectLogic) ListProjects(ctx context.Context) ([]v1.ProjectResp, error) {
	var projects []model.Project
	if err := l.svcCtx.DB.Find(&projects).Error; err != nil {
		return nil, err
	}

	var res []v1.ProjectResp
	for _, p := range projects {
		res = append(res, v1.ProjectResp{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			OwnerID:     p.OwnerID,
			CreatedAt:   p.CreatedAt,
		})
	}
	return res, nil
}

// DeployProject 部署项目到集群。
//
// 参数:
//   - ctx: 上下文
//   - req: 部署请求，包含项目 ID 和集群 ID
//
// 返回: 成功返回 nil，失败返回错误
//
// 流程:
//  1. 加载项目及其关联服务
//  2. 获取目标集群配置
//  3. 逐个部署服务到集群
func (l *ProjectLogic) DeployProject(ctx context.Context, req v1.DeployProjectReq) error {
	// 1. Get Project with Services
	var project model.Project
	if err := l.svcCtx.DB.Preload("Services").First(&project, req.ProjectID).Error; err != nil {
		return err
	}

	// 2. Get Cluster
	var cluster model.Cluster
	if err := l.svcCtx.DB.First(&cluster, req.ClusterID).Error; err != nil {
		return err
	}

	// 3. Deploy each service
	for _, svc := range project.Services {
		if err := DeployToCluster(ctx, &cluster, svc.YamlContent); err != nil {
			return err
		}
	}

	return nil
}
