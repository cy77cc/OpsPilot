// Package deployment 提供部署管理相关的工具实现。
//
// 本文件实现部署操作工具集，包括：
//   - 部署目标查询和管理
//   - 配置项读取和对比
//   - 集群和服务清单查询
package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
)

// =============================================================================
// 输入类型定义
// =============================================================================

// DeploymentTargetListInput 部署目标列表查询输入。
type DeploymentTargetListInput struct {
	Env     string `json:"env,omitempty" jsonschema_description:"optional environment filter"`
	Status  string `json:"status,omitempty" jsonschema_description:"optional target status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional target keyword filter"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max targets,default=50"`
}

// DeploymentTargetDetailInput 部署目标详情查询输入。
type DeploymentTargetDetailInput struct {
	TargetID int `json:"target_id" jsonschema_description:"required,deployment target id"`
}

// DeploymentBootstrapStatusInput 引导状态查询输入。
type DeploymentBootstrapStatusInput struct {
	TargetID int `json:"target_id" jsonschema_description:"required,deployment target id"`
}

// ConfigAppListInput 配置应用列表查询输入。
type ConfigAppListInput struct {
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on service name"`
	Env     string `json:"env,omitempty" jsonschema_description:"optional env filter"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max apps,default=50"`
}

// ConfigItemGetInput 配置项查询输入。
type ConfigItemGetInput struct {
	AppID int    `json:"app_id" jsonschema_description:"required,service id as config app id"`
	Key   string `json:"key" jsonschema_description:"required,config key"`
	Env   string `json:"env,omitempty" jsonschema_description:"optional env"`
}

// ConfigDiffInput 配置对比输入。
type ConfigDiffInput struct {
	AppID int    `json:"app_id" jsonschema_description:"required,service id as config app id"`
	EnvA  string `json:"env_a" jsonschema_description:"required,compare env a"`
	EnvB  string `json:"env_b" jsonschema_description:"required,compare env b"`
}

// ClusterInventoryInput 集群清单查询输入。
type ClusterInventoryInput struct {
	Status  string `json:"status,omitempty" jsonschema_description:"optional cluster status filter"`
	Keyword string `json:"keyword,omitempty" jsonschema_description:"optional keyword on name/endpoint"`
	Limit   int    `json:"limit,omitempty" jsonschema_description:"max clusters,default=50"`
}

// ServiceInventoryInput 服务清单查询输入。
type ServiceInventoryInput struct {
	Keyword     string `json:"keyword,omitempty" jsonschema_description:"optional keyword on service name/owner"`
	RuntimeType string `json:"runtime_type,omitempty" jsonschema_description:"optional runtime type filter,k8s/compose/helm"`
	Env         string `json:"env,omitempty" jsonschema_description:"optional environment filter"`
	Status      string `json:"status,omitempty" jsonschema_description:"optional service status filter"`
	Limit       int    `json:"limit,omitempty" jsonschema_description:"max services,default=50"`
}

// NewDeploymentTools 创建所有部署工具。
//
// 部署工具全部为只读工具，不修改任何状态。
func NewDeploymentTools(ctx context.Context) []tool.InvokableTool {
	return NewDeploymentReadonlyTools(ctx)
}

// NewDeploymentReadonlyTools 创建部署只读工具子集。
//
// 返回只读工具列表，包括：
//   - 部署目标查询（deployment_target_list, deployment_target_detail）
//   - 引导状态查询（deployment_bootstrap_status）
//   - 配置查询（config_app_list, config_item_get, config_diff）
//   - 清单查询（cluster_list_inventory, service_list_inventory）
//
// 这些工具不修改任何状态，可安全用于诊断和巡检场景。
func NewDeploymentReadonlyTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{
		DeploymentTargetList(ctx),
		DeploymentTargetDetail(ctx),
		DeploymentBootstrapStatus(ctx),
		ConfigAppList(ctx),
		ConfigItemGet(ctx),
		ConfigDiff(ctx),
		ClusterListInventory(ctx),
		ServiceListInventory(ctx),
	}
}

// NewDeploymentWriteTools 创建部署可写工具子集
func NewDeploymentWriteTools(ctx context.Context) []tool.InvokableTool {
	return []tool.InvokableTool{}
}

func depsFromContextOrFallback(ctx context.Context) *svc.ServiceContext {
	return svc.GetServiceContext(ctx)
}

type DeploymentTargetListOutput struct {
	Total int              `json:"total"`
	List  []map[string]any `json:"list"`
}

func DeploymentTargetList(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"deployment_target_list",
		"Query deployment target list. Optional parameters: env/status/keyword/limit. Example: {\"env\":\"prod\",\"limit\":20}.",
		func(ctx context.Context, input *DeploymentTargetListInput, opts ...tool.Option) (*DeploymentTargetListOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
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
			query := svcCtx.DB.Model(&model.DeploymentTarget{})
			if env := strings.TrimSpace(input.Env); env != "" {
				query = query.Where("env = ?", env)
			}
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ?", pattern)
			}
			var rows []model.DeploymentTarget
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			list := make([]map[string]any, 0, len(rows))
			for _, item := range rows {
				list = append(list, map[string]any{
					"id":               item.ID,
					"name":             item.Name,
					"env":              item.Env,
					"status":           item.Status,
					"target_type":      item.TargetType,
					"runtime_type":     item.RuntimeType,
					"cluster_id":       item.ClusterID,
					"credential_id":    item.CredentialID,
					"readiness_status": item.ReadinessStatus,
				})
			}
			return &DeploymentTargetListOutput{
				Total: len(list),
				List:  list,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type DeploymentTargetDetailOutput struct {
	Target model.DeploymentTarget       `json:"target"`
	Nodes  []model.DeploymentTargetNode `json:"nodes"`
}

func DeploymentTargetDetail(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"deployment_target_detail",
		"Query deployment target detail. target_id is required. Example: {\"target_id\":12}.",
		func(ctx context.Context, input *DeploymentTargetDetailInput, opts ...tool.Option) (*DeploymentTargetDetailOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.TargetID <= 0 {
				return nil, fmt.Errorf("target_id is required")
			}
			var target model.DeploymentTarget
			if err := svcCtx.DB.First(&target, input.TargetID).Error; err != nil {
				return nil, err
			}
			var nodes []model.DeploymentTargetNode
			_ = svcCtx.DB.Where("target_id = ?", target.ID).Order("id asc").Find(&nodes).Error
			return &DeploymentTargetDetailOutput{
				Target: target,
				Nodes:  nodes,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type DeploymentBootstrapStatusOutput struct {
	TargetID        uint                              `json:"target_id"`
	TargetName      string                            `json:"target_name"`
	BootstrapJobID  string                            `json:"bootstrap_job_id"`
	TargetStatus    string                            `json:"target_status"`
	ReadinessStatus string                            `json:"readiness_status"`
	BootstrapJob    *model.EnvironmentInstallJob      `json:"bootstrap_job,omitempty"`
	Steps           []model.EnvironmentInstallJobStep `json:"steps,omitempty"`
}

func DeploymentBootstrapStatus(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"deployment_bootstrap_status",
		"Query deployment target bootstrap status. target_id is required. Example: {\"target_id\":12}.",
		func(ctx context.Context, input *DeploymentBootstrapStatusInput, opts ...tool.Option) (*DeploymentBootstrapStatusOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.TargetID <= 0 {
				return nil, fmt.Errorf("target_id is required")
			}
			var target model.DeploymentTarget
			if err := svcCtx.DB.First(&target, input.TargetID).Error; err != nil {
				return nil, err
			}
			result := &DeploymentBootstrapStatusOutput{
				TargetID:        target.ID,
				TargetName:      target.Name,
				BootstrapJobID:  target.BootstrapJobID,
				TargetStatus:    target.Status,
				ReadinessStatus: target.ReadinessStatus,
			}
			if strings.TrimSpace(target.BootstrapJobID) == "" {
				return result, nil
			}
			var job model.EnvironmentInstallJob
			if err := svcCtx.DB.Where("id = ?", target.BootstrapJobID).First(&job).Error; err == nil {
				result.BootstrapJob = &job
				var steps []model.EnvironmentInstallJobStep
				_ = svcCtx.DB.Where("job_id = ?", job.ID).Order("id asc").Find(&steps).Error
				result.Steps = steps
			}
			return result, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ConfigAppListOutput struct {
	Total int              `json:"total"`
	List  []map[string]any `json:"list"`
}

func ConfigAppList(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"config_app_list",
		"Query config app list. Optional parameters: keyword/env/limit. Example: {\"env\":\"prod\"}.",
		func(ctx context.Context, input *ConfigAppListInput, opts ...tool.Option) (*ConfigAppListOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
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
			if env := strings.TrimSpace(input.Env); env != "" {
				query = query.Where("env = ?", env)
			}
			var services []model.Service
			if err := query.Order("id desc").Limit(limit).Find(&services).Error; err != nil {
				return nil, err
			}
			list := make([]map[string]any, 0, len(services))
			for _, svc := range services {
				list = append(list, map[string]any{"app_id": svc.ID, "name": svc.Name, "env": svc.Env, "owner": svc.Owner})
			}
			return &ConfigAppListOutput{
				Total: len(list),
				List:  list,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ConfigItemGetOutput struct {
	AppID     int    `json:"app_id"`
	Env       string `json:"env"`
	Key       string `json:"key"`
	Value     any    `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

func ConfigItemGet(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"config_item_get",
		"Query config item value. app_id and key are required, optional env. Example: {\"app_id\":12,\"key\":\"DATABASE_URL\"}.",
		func(ctx context.Context, input *ConfigItemGetInput, opts ...tool.Option) (*ConfigItemGetOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.AppID <= 0 {
				return nil, fmt.Errorf("app_id is required")
			}
			key := strings.TrimSpace(input.Key)
			if key == "" {
				return nil, fmt.Errorf("key is required")
			}
			env := strings.TrimSpace(input.Env)
			if env == "" {
				env = "staging"
			}
			var set model.ServiceVariableSet
			if err := svcCtx.DB.Where("service_id = ? AND env = ?", input.AppID, env).Order("updated_at desc").First(&set).Error; err != nil {
				return nil, err
			}
			values := map[string]any{}
			_ = json.Unmarshal([]byte(set.ValuesJSON), &values)
			return &ConfigItemGetOutput{
				AppID:     input.AppID,
				Env:       env,
				Key:       key,
				Value:     values[key],
				UpdatedAt: set.UpdatedAt.Format("2006-01-02 15:04:05"),
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ConfigDiffOutput struct {
	AppID     int              `json:"app_id"`
	EnvA      string           `json:"env_a"`
	EnvB      string           `json:"env_b"`
	DiffCount int              `json:"diff_count"`
	Diff      []map[string]any `json:"diff"`
}

func ConfigDiff(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"config_diff",
		"Compare config difference. app_id, env_a, env_b are required. Example: {\"app_id\":12,\"env_a\":\"staging\",\"env_b\":\"prod\"}.",
		func(ctx context.Context, input *ConfigDiffInput, opts ...tool.Option) (*ConfigDiffOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
			if svcCtx == nil || svcCtx.DB == nil {
				return nil, fmt.Errorf("service context is nil")
			}
			if input.AppID <= 0 {
				return nil, fmt.Errorf("app_id is required")
			}
			envA := strings.TrimSpace(input.EnvA)
			envB := strings.TrimSpace(input.EnvB)
			if envA == "" {
				return nil, fmt.Errorf("env_a is required")
			}
			if envB == "" {
				return nil, fmt.Errorf("env_b is required")
			}
			readEnv := func(env string) (map[string]any, error) {
				var set model.ServiceVariableSet
				if err := svcCtx.DB.Where("service_id = ? AND env = ?", input.AppID, env).Order("updated_at desc").First(&set).Error; err != nil {
					return nil, err
				}
				out := map[string]any{}
				_ = json.Unmarshal([]byte(set.ValuesJSON), &out)
				return out, nil
			}
			a, err := readEnv(envA)
			if err != nil {
				return nil, err
			}
			b, err := readEnv(envB)
			if err != nil {
				return nil, err
			}
			diff := make([]map[string]any, 0)
			seen := map[string]struct{}{}
			for k, av := range a {
				seen[k] = struct{}{}
				bv := b[k]
				if fmt.Sprintf("%v", av) != fmt.Sprintf("%v", bv) {
					diff = append(diff, map[string]any{"key": k, "env_a": av, "env_b": bv})
				}
			}
			for k, bv := range b {
				if _, ok := seen[k]; ok {
					continue
				}
				diff = append(diff, map[string]any{"key": k, "env_a": nil, "env_b": bv})
			}
			return &ConfigDiffOutput{
				AppID:     input.AppID,
				EnvA:      envA,
				EnvB:      envB,
				DiffCount: len(diff),
				Diff:      diff,
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ClusterListInventoryOutput struct {
	Total          int              `json:"total"`
	List           []map[string]any `json:"list"`
	FiltersApplied map[string]any   `json:"filters_applied"`
}

func ClusterListInventory(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"cluster_list_inventory",
		"Query cluster inventory list. Optional parameters: status/keyword/limit. Example: {\"status\":\"active\"}.",
		func(ctx context.Context, input *ClusterInventoryInput, opts ...tool.Option) (*ClusterListInventoryOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
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
			query := svcCtx.DB.Model(&model.Cluster{})
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR endpoint LIKE ?", pattern, pattern)
			}
			var rows []model.Cluster
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			list := make([]map[string]any, 0, len(rows))
			for _, item := range rows {
				list = append(list, map[string]any{
					"id":         item.ID,
					"name":       item.Name,
					"status":     item.Status,
					"type":       item.Type,
					"endpoint":   item.Endpoint,
					"version":    item.Version,
					"updated_at": item.UpdatedAt,
				})
			}
			return &ClusterListInventoryOutput{
				Total: len(list),
				List:  list,
				FiltersApplied: map[string]any{
					"status":  strings.TrimSpace(input.Status),
					"keyword": strings.TrimSpace(input.Keyword),
					"limit":   limit,
				},
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}

type ServiceListInventoryOutput struct {
	Total          int              `json:"total"`
	List           []map[string]any `json:"list"`
	FiltersApplied map[string]any   `json:"filters_applied"`
}

func ServiceListInventory(ctx context.Context) tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"service_list_inventory",
		"Query service inventory list. Optional parameters: status/runtime_type/env/keyword/limit. Example: {\"env\":\"prod\"}.",
		func(ctx context.Context, input *ServiceInventoryInput, opts ...tool.Option) (*ServiceListInventoryOutput, error) {
			svcCtx := depsFromContextOrFallback(ctx)
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
			if status := strings.TrimSpace(input.Status); status != "" {
				query = query.Where("status = ?", status)
			}
			if env := strings.TrimSpace(input.Env); env != "" {
				query = query.Where("env = ?", env)
			}
			if runtime := strings.TrimSpace(input.RuntimeType); runtime != "" {
				query = query.Where("runtime_type = ?", runtime)
			}
			if kw := strings.TrimSpace(input.Keyword); kw != "" {
				pattern := "%" + kw + "%"
				query = query.Where("name LIKE ? OR owner LIKE ?", pattern, pattern)
			}
			var rows []model.Service
			if err := query.Order("id desc").Limit(limit).Find(&rows).Error; err != nil {
				return nil, err
			}
			list := make([]map[string]any, 0, len(rows))
			for _, item := range rows {
				list = append(list, map[string]any{
					"id":            item.ID,
					"name":          item.Name,
					"status":        item.Status,
					"env":           item.Env,
					"owner":         item.Owner,
					"runtime_type":  item.RuntimeType,
					"config_mode":   item.ConfigMode,
					"render_target": item.RenderTarget,
					"updated_at":    item.UpdatedAt,
				})
			}
			return &ServiceListInventoryOutput{
				Total: len(list),
				List:  list,
				FiltersApplied: map[string]any{
					"status":       strings.TrimSpace(input.Status),
					"env":          strings.TrimSpace(input.Env),
					"runtime_type": strings.TrimSpace(input.RuntimeType),
					"keyword":      strings.TrimSpace(input.Keyword),
					"limit":        limit,
				},
			}, nil
		},
	)
	if err != nil {
		panic(err)
	}
	return t
}
