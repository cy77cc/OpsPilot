package model

import (
	"time"
)

// Project 项目表
type Project struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(64);not null;unique" json:"name"`
	Description string    `gorm:"column:description;type:varchar(256)" json:"description"`
	OwnerID     int64     `gorm:"column:owner_id" json:"owner_id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Services    []Service `gorm:"foreignKey:ProjectID" json:"services,omitempty"`
}

func (Project) TableName() string {
	return "projects"
}

// Service 服务表
type Service struct {
	ID                        uint      `gorm:"primaryKey;column:id" json:"id"`
	ProjectID                 uint      `gorm:"column:project_id;not null" json:"project_id"`
	TeamID                    uint      `gorm:"column:team_id;default:0;index" json:"team_id"`
	OwnerUserID               uint      `gorm:"column:owner_user_id;default:0" json:"owner_user_id"`
	Owner                     string    `gorm:"column:owner;type:varchar(64);default:''" json:"owner"`
	Env                       string    `gorm:"column:env;type:varchar(32);default:'staging';index" json:"env"`
	RuntimeType               string    `gorm:"column:runtime_type;type:varchar(16);default:'k8s';index" json:"runtime_type"` // k8s/compose/helm
	ConfigMode                string    `gorm:"column:config_mode;type:varchar(16);default:'standard'" json:"config_mode"`    // standard/custom
	ServiceKind               string    `gorm:"column:service_kind;type:varchar(32);default:'web'" json:"service_kind"`
	Visibility                string    `gorm:"column:visibility;type:varchar(16);default:'team'" json:"visibility"`
	GrantedTeams              string    `gorm:"column:granted_teams;type:json" json:"granted_teams"`
	Icon                      string    `gorm:"column:icon;type:varchar(256);default:''" json:"icon"`
	Tags                      string    `gorm:"column:tags;type:json" json:"tags"`
	DeployCount               int       `gorm:"column:deploy_count;default:0" json:"deploy_count"`
	RenderTarget              string    `gorm:"column:render_target;type:varchar(16);default:'k8s'" json:"render_target"`
	LabelsJSON                string    `gorm:"column:labels_json;type:json" json:"labels_json"`
	StandardJSON              string    `gorm:"column:standard_config_json;type:json" json:"standard_config_json"`
	CustomYAML                string    `gorm:"column:custom_yaml;type:mediumtext" json:"custom_yaml"`
	TemplateVer               string    `gorm:"column:source_template_version;type:varchar(32);default:'v1'" json:"source_template_version"`
	LastRevisionID            uint      `gorm:"column:last_revision_id;default:0" json:"last_revision_id"`
	DefaultTargetID           uint      `gorm:"column:default_target_id;default:0" json:"default_target_id"`
	DefaultDeploymentTargetID uint      `gorm:"column:default_deployment_target_id;default:0" json:"default_deployment_target_id"`
	RuntimeStrategyJSON       string    `gorm:"column:runtime_strategy_json;type:longtext" json:"runtime_strategy_json"`
	TemplateEngineVersion     string    `gorm:"column:template_engine_version;type:varchar(16);default:'v1'" json:"template_engine_version"`
	Status                    string    `gorm:"column:status;type:varchar(32);default:'draft';index" json:"status"`
	Name                      string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Type                      string    `gorm:"column:type;type:varchar(32);not null" json:"type"` // stateless / stateful
	Image                     string    `gorm:"column:image;type:varchar(256);not null" json:"image"`
	Replicas                  int32     `gorm:"column:replicas;default:1" json:"replicas"`
	ServicePort               int32     `gorm:"column:service_port" json:"service_port"`
	ContainerPort             int32     `gorm:"column:container_port" json:"container_port"`
	NodePort                  int32     `gorm:"column:node_port" json:"node_port"`
	EnvVars                   string    `gorm:"column:env_vars;type:json" json:"env_vars"`   // JSON string
	Resources                 string    `gorm:"column:resources;type:json" json:"resources"` // JSON string
	YamlContent               string    `gorm:"column:yaml_content;type:mediumtext" json:"yaml_content"`
	CreatedAt                 time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                 time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Service) TableName() string {
	return "services"
}

type ServiceHelmRelease struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	ServiceID    uint      `gorm:"column:service_id;not null;index" json:"service_id"`
	ChartName    string    `gorm:"column:chart_name;type:varchar(128);not null" json:"chart_name"`
	ChartVersion string    `gorm:"column:chart_version;type:varchar(64);default:''" json:"chart_version"`
	ChartRef     string    `gorm:"column:chart_ref;type:varchar(512);default:''" json:"chart_ref"` // local path/repo ref
	ValuesYAML   string    `gorm:"column:values_yaml;type:mediumtext" json:"values_yaml"`
	RenderedYAML string    `gorm:"column:rendered_yaml;type:longtext" json:"rendered_yaml"`
	Status       string    `gorm:"column:status;type:varchar(32);default:'imported'" json:"status"`
	CreatedBy    uint      `gorm:"column:created_by;default:0" json:"created_by"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (ServiceHelmRelease) TableName() string {
	return "service_helm_releases"
}

type ServiceRenderSnapshot struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`
	ServiceID    uint      `gorm:"column:service_id;not null;index" json:"service_id"`
	Target       string    `gorm:"column:target;type:varchar(16);not null" json:"target"` // k8s/compose/helm
	Mode         string    `gorm:"column:mode;type:varchar(16);not null" json:"mode"`     // standard/custom
	RenderedYAML string    `gorm:"column:rendered_yaml;type:longtext" json:"rendered_yaml"`
	Diagnostics  string    `gorm:"column:diagnostics_json;type:json" json:"diagnostics_json"`
	CreatedBy    uint      `gorm:"column:created_by;default:0" json:"created_by"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (ServiceRenderSnapshot) TableName() string {
	return "service_render_snapshots"
}
