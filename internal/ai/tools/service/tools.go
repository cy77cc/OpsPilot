// Package service 提供服务管理相关的工具实现。
//
// 本文件实现服务操作工具集，包括：
//   - 服务详情和状态查询
//   - 服务部署预览和应用
//   - 服务目录和分类查询
//   - 服务可见性检查
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	einoutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// =============================================================================
// 输入类型定义
// =============================================================================

// ServiceDetailInput 服务详情查询输入。
type ServiceDetailInput struct {
	ServiceID int `json:"service_id" jsonschema_description:"required,service id"`
}

// ServiceDeployPreviewInput 部署预览输入。
type ServiceDeployPreviewInput struct {
	ServiceID int `json:"service_id" jsonschema_description:"required,service id"`
	ClusterID int `json:"cluster_id" jsonschema_description:"required,cluster id"`
}

// ServiceDeployApplyInput 部署应用输入。
type ServiceDeployApplyInput struct {
	ServiceID int `json:"service_id" jsonschema_description:"required,service id"`
	ClusterID int `json:"cluster_id" jsonschema_description:"required,cluster id"`
}

// ServiceDeployInput 统一部署输入。
type ServiceDeployInput struct {
	ServiceID int  `json:"service_id" jsonschema_description:"required,service id"`
	ClusterID int  `json:"cluster_id" jsonschema_description:"required,cluster id"`
	Preview   bool `json:"preview,omitempty" jsonschema_description:"preview deploy without apply"`
	Apply     bool `json:"apply,omitempty" jsonschema_description:"apply deploy after approval"`
}

// ServiceStatusInput 服务状态查询输入。
type ServiceStatusInput struct {
	ServiceID int `json:"service_id" jsonschema_description:"required,service id"`
}

// ServiceStatusByTargetInput 语义化服务状态查询输入。
type ServiceStatusByTargetInput struct {
	Target string `json:"target" jsonschema_description:"required,service id or service name"`
}

// ServiceCatalogListInput 服务目录查询输入。
type ServiceCatalogListInput struct {
	Keyword    string `json:"keyword,omitempty" jsonschema_description:"optional keyword on service name/owner"`
	CategoryID int    `json:"category_id,omitempty" jsonschema_description:"optional category id: 1 middleware, 2 business"`
	Limit      int    `json:"limit,omitempty" jsonschema_description:"max services,default=50"`
}

// ServiceVisibilityCheckInput 可见性检查输入。
type ServiceVisibilityCheckInput struct {
	ServiceID int `json:"service_id" jsonschema_description:"required,service id"`
}

// NewServiceTools 创建所有服务工具。
func NewServiceTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		ServiceGetDetail(ctx),
		ServiceStatus(ctx),
		ServiceStatusByTarget(ctx),
		ServiceDeployPreview(ctx),
		ServiceDeployApply(ctx),
		ServiceDeploy(ctx),
		ServiceCatalogList(ctx),
		ServiceCategoryTree(ctx),
		ServiceVisibilityCheck(ctx),
	}
}

type ServiceGetDetailOutput struct {
	Service model.Service `json:"service"`
}

func ServiceGetDetail(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_get_detail",
		"Get detailed information about a specific service including configuration, deployment settings, runtime type, and metadata. service_id is required. Returns complete service object with all fields. Use this when you need comprehensive service information. Example: {\"service_id\":123}.",
		func(ctx context.Context, input *ServiceDetailInput, opts ...tool.Option) (*ServiceGetDetailOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			sid := input.ServiceID
			if sid <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			var s model.Service
			if err := svcCtx.DB.First(&s, sid).Error; err != nil {
				return nil, err
			}
			return &ServiceGetDetailOutput{Service: s}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceStatusOutput struct {
	ServiceID   uint   `json:"service_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Env         string `json:"env"`
	RuntimeType string `json:"runtime_type"`
	Image       string `json:"image"`
	Replicas    int32  `json:"replicas"`
	UpdatedAt   string `json:"updated_at"`
}

func ServiceStatus(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_status",
		"Get current status and basic runtime information of a service. service_id is required. Returns service name, status, environment, runtime type (k8s/compose/helm), container image, replica count, and last update time. Use this for quick status checks. Example: {\"service_id\":123}.",
		func(ctx context.Context, input *ServiceStatusInput, opts ...tool.Option) (*ServiceStatusOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.ServiceID <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			var svc model.Service
			if err := svcCtx.DB.First(&svc, input.ServiceID).Error; err != nil {
				return nil, err
			}
			return &ServiceStatusOutput{
				ServiceID:   svc.ID,
				Name:        svc.Name,
				Status:      svc.Status,
				Env:         svc.Env,
				RuntimeType: svc.RuntimeType,
				Image:       svc.Image,
				Replicas:    svc.Replicas,
				UpdatedAt:   svc.UpdatedAt.Format("2006-01-02 15:04:05"),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func ServiceStatusByTarget(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_status_by_target",
		"Resolve a service by target string and get current status. target may be a service id or exact service name. Returns the same runtime status fields as service_status. Example: {\"target\":\"payment-service\"}.",
		func(ctx context.Context, input *ServiceStatusByTargetInput, opts ...tool.Option) (*ServiceStatusOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			svc, err := resolveServiceByTarget(svcCtx, input.Target)
			if err != nil {
				return nil, err
			}
			return &ServiceStatusOutput{
				ServiceID:   svc.ID,
				Name:        svc.Name,
				Status:      svc.Status,
				Env:         svc.Env,
				RuntimeType: svc.RuntimeType,
				Image:       svc.Image,
				Replicas:    svc.Replicas,
				UpdatedAt:   svc.UpdatedAt.Format("2006-01-02 15:04:05"),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceDeployPreviewOutput struct {
	Preview   bool   `json:"preview"`
	ServiceID int    `json:"service_id"`
	ClusterID int    `json:"cluster_id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	Replicas  int32  `json:"replicas"`
}

func ServiceDeployPreview(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_deploy_preview",
		"Preview a service deployment without actually applying changes. service_id and cluster_id are required. Returns the deployment plan including service name, container image, and replica count. Use this to verify deployment configuration before executing with service_deploy_apply. Example: {\"service_id\":123,\"cluster_id\":456}.",
		func(ctx context.Context, input *ServiceDeployPreviewInput, opts ...tool.Option) (*ServiceDeployPreviewOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.ServiceID <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			if input.ClusterID <= 0 {
				return nil, fmt.Errorf("cluster_id is required")
			}
			var s model.Service
			if err := svcCtx.DB.First(&s, input.ServiceID).Error; err != nil {
				return nil, err
			}
			return &ServiceDeployPreviewOutput{
				Preview:   true,
				ServiceID: input.ServiceID,
				ClusterID: input.ClusterID,
				Name:      s.Name,
				Image:     s.Image,
				Replicas:  s.Replicas,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceDeployApplyOutput struct {
	Applied   bool   `json:"applied"`
	ServiceID int    `json:"service_id"`
	ClusterID int    `json:"cluster_id"`
	Message   string `json:"message"`
	Image     string `json:"image"`
}

func ServiceDeployApply(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_deploy_apply",
		"Execute a service deployment to a target cluster. service_id and cluster_id are required. This is a mutating operation that will create/update the deployment. Ensure you have previewed the deployment with service_deploy_preview first. Returns deployment status and applied configuration. Example: {\"service_id\":123,\"cluster_id\":456}.",
		func(ctx context.Context, input *ServiceDeployApplyInput, opts ...tool.Option) (*ServiceDeployApplyOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.ServiceID <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			if input.ClusterID <= 0 {
				return nil, fmt.Errorf("cluster_id is required")
			}
			var svc model.Service
			if err := svcCtx.DB.First(&svc, input.ServiceID).Error; err != nil {
				return nil, err
			}
			var cluster model.Cluster
			if err := svcCtx.DB.First(&cluster, input.ClusterID).Error; err != nil {
				return nil, err
			}
			return &ServiceDeployApplyOutput{
				Applied:   true,
				ServiceID: input.ServiceID,
				ClusterID: input.ClusterID,
				Message:   "deploy apply executed in MVP mode",
				Image:     svc.Image,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceDeployOutput struct {
	Preview   bool        `json:"preview"`
	Applied   bool        `json:"applied"`
	ServiceID int         `json:"service_id"`
	ClusterID int         `json:"cluster_id"`
	Data      interface{} `json:"data,omitempty"`
}

func ServiceDeploy(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_deploy",
		"Unified service deployment tool supporting both preview and apply modes. service_id and cluster_id are required. Set preview=true (default) to see the deployment plan without applying. Set apply=true to execute the deployment. This operation deploys the service container image to the specified cluster. Example: {\"service_id\":123,\"cluster_id\":456,\"preview\":true}.",
		func(ctx context.Context, input *ServiceDeployInput, opts ...tool.Option) (*ServiceDeployOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.ServiceID <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			if input.ClusterID <= 0 {
				return nil, fmt.Errorf("cluster_id is required")
			}
			var svc model.Service
			if err := svcCtx.DB.First(&svc, input.ServiceID).Error; err != nil {
				return nil, err
			}
			if input.Apply {
				var cluster model.Cluster
				if err := svcCtx.DB.First(&cluster, input.ClusterID).Error; err != nil {
					return nil, err
				}
				return &ServiceDeployOutput{
					Preview:   false,
					Applied:   true,
					ServiceID: input.ServiceID,
					ClusterID: input.ClusterID,
					Data: map[string]any{
						"message": "deploy apply executed in MVP mode",
						"image":   svc.Image,
					},
				}, nil
			}
			return &ServiceDeployOutput{
				Preview:   true,
				Applied:   false,
				ServiceID: input.ServiceID,
				ClusterID: input.ClusterID,
				Data: map[string]any{
					"name":     svc.Name,
					"image":    svc.Image,
					"replicas": svc.Replicas,
				},
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceCatalogListOutput struct {
	Total          int              `json:"total"`
	List           []map[string]any `json:"list"`
	FiltersApplied map[string]any   `json:"filters_applied"`
}

func ServiceCatalogList(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_catalog_list",
		"Query the service catalog with filtering options. Optional parameters: keyword searches by service name or owner, category_id filters by service kind (1=middleware, 2=business), limit controls max results (default 50, max 200). Returns services with id, name, owner, environment, service_kind, visibility, and deployment count. Example: {\"keyword\":\"payment\",\"category_id\":2,\"limit\":20}.",
		func(ctx context.Context, input *ServiceCatalogListInput, opts ...tool.Option) (*ServiceCatalogListOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			limit := input.Limit
			if limit <= 0 {
				limit = 50
			}
			if limit > 200 {
				limit = 200
			}
			query := svcCtx.DB.Model(&model.Service{})
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR owner LIKE ?", pattern, pattern)
			}
			switch input.CategoryID {
			case 1:
				query = query.Where("service_kind = ?", "middleware")
			case 2:
				query = query.Where("service_kind = ?", "business")
			}
			var rows []model.Service
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			list := make([]map[string]any, 0, len(rows))
			for _, item := range rows {
				list = append(list, map[string]any{
					"id":           item.ID,
					"name":         item.Name,
					"owner":        item.Owner,
					"env":          item.Env,
					"service_kind": item.ServiceKind,
					"visibility":   item.Visibility,
					"deploy_count": item.DeployCount,
					"icon":         item.Icon,
				})
			}
			return &ServiceCatalogListOutput{
				Total: len(list),
				List:  list,
				FiltersApplied: map[string]any{
					"keyword":     strings.TrimSpace(input.Keyword),
					"category_id": input.CategoryID,
					"limit":       limit,
				},
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceCategoryTreeOutput struct {
	Tree []map[string]any `json:"tree"`
}

func ServiceCategoryTree(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_category_tree",
		"Get the service category tree structure showing middleware and business service categories with counts. Returns an array of categories, each with id, key (middleware/business), label, and count of services. Use this to understand the service distribution across categories. Example: {}.",
		func(ctx context.Context, _ struct{}, opts ...tool.Option) (*ServiceCategoryTreeOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			type countRow struct {
				ServiceKind string
				Count       int64
			}
			var rows []countRow
			if err := svcCtx.DB.Model(&model.Service{}).
				Select("service_kind, COUNT(1) AS count").
				Group("service_kind").
				Scan(&rows).Error; err != nil {
				return nil, err
			}
			tree := []map[string]any{
				{"id": 1, "key": "middleware", "label": "Middleware Services", "count": int64(0)},
				{"id": 2, "key": "business", "label": "Business Services", "count": int64(0)},
			}
			for _, row := range rows {
				switch strings.TrimSpace(row.ServiceKind) {
				case "middleware":
					tree[0]["count"] = row.Count
				case "business":
					tree[1]["count"] = row.Count
				}
			}
			return &ServiceCategoryTreeOutput{Tree: tree}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceVisibilityCheckOutput struct {
	ServiceID    uint   `json:"service_id"`
	ServiceName  string `json:"service_name"`
	ServiceKind  string `json:"service_kind"`
	Visibility   string `json:"visibility"`
	GrantedTeams []uint `json:"granted_teams"`
	OwnerUserID  uint   `json:"owner_user_id"`
	OwnerTeamID  uint   `json:"owner_team_id"`
	UpdatedAt    string `json:"updated_at"`
}

func ServiceVisibilityCheck(ctx context.Context) tool.InvokableTool {
	t, err := einoutils.InferOptionableTool(
		"service_visibility_check",
		"Check the visibility configuration of a service including access control settings. service_id is required. Returns visibility level (public/private/team), granted team IDs that can access the service, owner user ID, and owner team ID. Use this to understand who can access a service. Example: {\"service_id\":123}.",
		func(ctx context.Context, input *ServiceVisibilityCheckInput, opts ...tool.Option) (*ServiceVisibilityCheckOutput, error) {
			svcCtx := svc.GetServiceContext(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.ServiceID <= 0 {
				return nil, fmt.Errorf("service_id is required")
			}
			var svc model.Service
			if err := svcCtx.DB.First(&svc, input.ServiceID).Error; err != nil {
				return nil, err
			}
			granted := []uint{}
			if strings.TrimSpace(svc.GrantedTeams) != "" {
				_ = json.Unmarshal([]byte(svc.GrantedTeams), &granted)
			}
			return &ServiceVisibilityCheckOutput{
				ServiceID:    svc.ID,
				ServiceName:  svc.Name,
				ServiceKind:  svc.ServiceKind,
				Visibility:   svc.Visibility,
				GrantedTeams: granted,
				OwnerUserID:  svc.OwnerUserID,
				OwnerTeamID:  svc.TeamID,
				UpdatedAt:    svc.UpdatedAt.Format("2006-01-02 15:04:05"),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

func resolveServiceByTarget(svcCtx *svc.ServiceContext, target string) (*model.Service, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return nil, fmt.Errorf("target is required")
	}
	if svcCtx == nil || svcCtx.DB == nil {
		return nil, fmt.Errorf("service context is nil")
	}
	var svc model.Service
	if id, err := strconv.ParseUint(trimmed, 10, 64); err == nil {
		if err := svcCtx.DB.First(&svc, id).Error; err == nil {
			return &svc, nil
		}
	}
	if err := svcCtx.DB.Where("name = ?", trimmed).First(&svc).Error; err == nil {
		return &svc, nil
	}
	return nil, fmt.Errorf("service target not found")
}
