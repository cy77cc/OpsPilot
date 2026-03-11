// Package model 提供数据库模型定义。
//
// 本文件定义项目和服务相关的数据模型，包括服务配置、Helm 发布和渲染快照。
package model

import (
	"time"
)

// Project 是项目表模型，存储项目基本信息。
//
// 表名: projects
// 关联: Service (一对多，通过 project_id)
type Project struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`                          // 项目 ID
	Name        string    `gorm:"column:name;type:varchar(64);not null;unique" json:"name"` // 项目名称 (唯一)
	Description string    `gorm:"column:description;type:varchar(256)" json:"description"`  // 项目描述
	OwnerID     int64     `gorm:"column:owner_id" json:"owner_id"`                         // 负责人 ID
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`      // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`      // 更新时间
	Services    []Service `gorm:"foreignKey:ProjectID" json:"services,omitempty"`          // 关联服务列表
}

// TableName 返回项目表名。
func (Project) TableName() string {
	return "projects"
}

// Service 是服务表模型，存储服务配置和部署信息。
//
// 表名: services
// 关联:
//   - Project (多对一，通过 project_id)
//   - DeploymentRelease (一对多)
//
// 类型说明:
//   - RuntimeType: k8s / compose / helm
//   - ConfigMode: standard (标准模式) / custom (自定义模式)
//   - ServiceKind: web / worker / cronjob
//   - Visibility: team / project / public
type Service struct {
	ID                        uint      `gorm:"primaryKey;column:id" json:"id"`                                               // 服务 ID
	ProjectID                 uint      `gorm:"column:project_id;not null" json:"project_id"`                                // 项目 ID
	TeamID                    uint      `gorm:"column:team_id;default:0;index" json:"team_id"`                              // 团队 ID
	OwnerUserID               uint      `gorm:"column:owner_user_id;default:0" json:"owner_user_id"`                        // 负责人 ID
	Owner                     string    `gorm:"column:owner;type:varchar(64);default:''" json:"owner"`                      // 负责人姓名
	Env                       string    `gorm:"column:env;type:varchar(32);default:'staging';index" json:"env"`             // 环境
	RuntimeType               string    `gorm:"column:runtime_type;type:varchar(16);default:'k8s';index" json:"runtime_type"` // 运行时: k8s/compose/helm
	ConfigMode                string    `gorm:"column:config_mode;type:varchar(16);default:'standard'" json:"config_mode"`  // 配置模式: standard/custom
	ServiceKind               string    `gorm:"column:service_kind;type:varchar(32);default:'web'" json:"service_kind"`     // 服务类型: web/worker/cronjob
	Visibility                string    `gorm:"column:visibility;type:varchar(16);default:'team'" json:"visibility"`        // 可见性: team/project/public
	GrantedTeams              string    `gorm:"column:granted_teams;type:json" json:"granted_teams"`                        // 授权团队 (JSON)
	Icon                      string    `gorm:"column:icon;type:varchar(256);default:''" json:"icon"`                       // 图标
	Tags                      string    `gorm:"column:tags;type:json" json:"tags"`                                          // 标签 (JSON)
	DeployCount               int       `gorm:"column:deploy_count;default:0" json:"deploy_count"`                          // 部署次数
	RenderTarget              string    `gorm:"column:render_target;type:varchar(16);default:'k8s'" json:"render_target"`  // 渲染目标: k8s/compose
	LabelsJSON                string    `gorm:"column:labels_json;type:json" json:"labels_json"`                            // 标签 (JSON)
	StandardJSON              string    `gorm:"column:standard_config_json;type:json" json:"standard_config_json"`          // 标准配置 (JSON)
	CustomYAML                string    `gorm:"column:custom_yaml;type:mediumtext" json:"custom_yaml"`                      // 自定义 YAML
	TemplateVer               string    `gorm:"column:source_template_version;type:varchar(32);default:'v1'" json:"source_template_version"` // 模板版本
	LastRevisionID            uint      `gorm:"column:last_revision_id;default:0" json:"last_revision_id"`                  // 最后版本 ID
	DefaultTargetID           uint      `gorm:"column:default_target_id;default:0" json:"default_target_id"`               // 默认目标 ID
	DefaultDeploymentTargetID uint      `gorm:"column:default_deployment_target_id;default:0" json:"default_deployment_target_id"` // 默认部署目标 ID
	RuntimeStrategyJSON       string    `gorm:"column:runtime_strategy_json;type:longtext" json:"runtime_strategy_json"`   // 运行时策略 (JSON)
	TemplateEngineVersion     string    `gorm:"column:template_engine_version;type:varchar(16);default:'v1'" json:"template_engine_version"` // 模板引擎版本
	Status                    string    `gorm:"column:status;type:varchar(32);default:'draft';index" json:"status"`        // 状态: draft/active/archived
	Name                      string    `gorm:"column:name;type:varchar(64);not null" json:"name"`                          // 服务名称
	Type                      string    `gorm:"column:type;type:varchar(32);not null" json:"type"`                          // 类型: stateless/stateful
	Image                     string    `gorm:"column:image;type:varchar(256);not null" json:"image"`                      // 镜像地址
	Replicas                  int32     `gorm:"column:replicas;default:1" json:"replicas"`                                  // 副本数
	ServicePort               int32     `gorm:"column:service_port" json:"service_port"`                                    // 服务端口
	ContainerPort             int32     `gorm:"column:container_port" json:"container_port"`                                // 容器端口
	NodePort                  int32     `gorm:"column:node_port" json:"node_port"`                                          // NodePort
	EnvVars                   string    `gorm:"column:env_vars;type:json" json:"env_vars"`                                  // 环境变量 (JSON)
	Resources                 string    `gorm:"column:resources;type:json" json:"resources"`                                // 资源配置 (JSON)
	YamlContent               string    `gorm:"column:yaml_content;type:mediumtext" json:"yaml_content"`                   // YAML 内容
	CreatedAt                 time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                         // 创建时间
	UpdatedAt                 time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                         // 更新时间
}

// TableName 返回服务表名。
func (Service) TableName() string {
	return "services"
}

// ServiceHelmRelease 是服务 Helm 发布表模型，存储 Helm Chart 配置。
//
// 表名: service_helm_releases
// 用途: Helm 类型的服务配置管理
type ServiceHelmRelease struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`                                   // 发布 ID
	ServiceID    uint      `gorm:"column:service_id;not null;index" json:"service_id"`              // 服务 ID
	ChartName    string    `gorm:"column:chart_name;type:varchar(128);not null" json:"chart_name"`   // Chart 名称
	ChartVersion string    `gorm:"column:chart_version;type:varchar(64);default:''" json:"chart_version"` // Chart 版本
	ChartRef     string    `gorm:"column:chart_ref;type:varchar(512);default:''" json:"chart_ref"`   // Chart 引用 (本地路径/仓库)
	ValuesYAML   string    `gorm:"column:values_yaml;type:mediumtext" json:"values_yaml"`            // Values 配置 (YAML)
	RenderedYAML string    `gorm:"column:rendered_yaml;type:longtext" json:"rendered_yaml"`          // 渲染结果 (YAML)
	Status       string    `gorm:"column:status;type:varchar(32);default:'imported'" json:"status"`  // 状态: imported/deployed/failed
	CreatedBy    uint      `gorm:"column:created_by;default:0" json:"created_by"`                    // 创建人 ID
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`               // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`               // 更新时间
}

// TableName 返回服务 Helm 发布表名。
func (ServiceHelmRelease) TableName() string {
	return "service_helm_releases"
}

// ServiceRenderSnapshot 是服务渲染快照表模型，存储服务配置渲染结果。
//
// 表名: service_render_snapshots
// 用途: 保存配置渲染历史，支持回滚和对比
type ServiceRenderSnapshot struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`                               // 快照 ID
	ServiceID    uint      `gorm:"column:service_id;not null;index" json:"service_id"`          // 服务 ID
	Target       string    `gorm:"column:target;type:varchar(16);not null" json:"target"`       // 目标: k8s/compose/helm
	Mode         string    `gorm:"column:mode;type:varchar(16);not null" json:"mode"`           // 模式: standard/custom
	RenderedYAML string    `gorm:"column:rendered_yaml;type:longtext" json:"rendered_yaml"`     // 渲染结果 (YAML)
	Diagnostics  string    `gorm:"column:diagnostics_json;type:json" json:"diagnostics_json"`   // 诊断信息 (JSON)
	CreatedBy    uint      `gorm:"column:created_by;default:0" json:"created_by"`               // 创建人 ID
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`          // 创建时间
}

// TableName 返回服务渲染快照表名。
func (ServiceRenderSnapshot) TableName() string {
	return "service_render_snapshots"
}
