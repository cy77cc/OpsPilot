// Package model 提供服务管理模块的数据模型定义。
//
// 本文件定义服务版本、变量集、部署目标、发布记录等模型。
package model

import "time"

// ServiceRevision 表示服务版本记录模型。
//
// 记录服务配置的历史版本，支持版本回滚和审计。
//
// 表名: service_revisions
type ServiceRevision struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`                                     // 版本 ID
	ServiceID      uint      `gorm:"column:service_id;not null;index" json:"service_id"`                 // 服务 ID
	RevisionNo     uint      `gorm:"column:revision_no;not null" json:"revision_no"`                     // 版本号
	ConfigMode     string    `gorm:"column:config_mode;type:varchar(16);not null" json:"config_mode"`    // 配置模式 (standard/custom)
	RenderTarget   string    `gorm:"column:render_target;type:varchar(16);not null" json:"render_target"` // 渲染目标 (k8s/compose)
	StandardConfig string    `gorm:"column:standard_config_json;type:longtext" json:"standard_config_json"` // 标准配置 JSON
	CustomYAML     string    `gorm:"column:custom_yaml;type:longtext" json:"custom_yaml"`                // 自定义 YAML
	VariableSchema string    `gorm:"column:variable_schema_json;type:longtext" json:"variable_schema_json"` // 变量 Schema JSON
	CreatedBy      uint      `gorm:"column:created_by;default:0" json:"created_by"`                      // 创建者 ID
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                 // 创建时间
}

// TableName 返回表名。
func (ServiceRevision) TableName() string {
	return "service_revisions"
}

// ServiceVariableSet 表示服务变量集模型。
//
// 存储服务在不同环境下的变量值。
//
// 表名: service_variable_sets
type ServiceVariableSet struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                                  // 变量集 ID
	ServiceID  uint      `gorm:"column:service_id;not null;index:idx_service_env,priority:1" json:"service_id"` // 服务 ID
	Env        string    `gorm:"column:env;type:varchar(32);not null;index:idx_service_env,priority:2" json:"env"` // 环境名称
	ValuesJSON string    `gorm:"column:values_json;type:longtext" json:"values_json"`             // 变量值 JSON
	SecretKeys string    `gorm:"column:secret_keys_json;type:longtext" json:"secret_keys_json"`   // 敏感变量键 JSON
	UpdatedBy  uint      `gorm:"column:updated_by;default:0" json:"updated_by"`                   // 更新者 ID
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`              // 更新时间
}

// TableName 返回表名。
func (ServiceVariableSet) TableName() string {
	return "service_variable_sets"
}

// ServiceDeployTarget 表示服务部署目标模型。
//
// 存储服务的默认部署目标配置。
//
// 表名: service_deploy_targets
type ServiceDeployTarget struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`                                         // 目标 ID
	ServiceID    uint      `gorm:"column:service_id;not null;index:idx_target_service_default,priority:1" json:"service_id"` // 服务 ID
	ClusterID    uint      `gorm:"column:cluster_id;not null;default:0" json:"cluster_id"`                 // 集群 ID
	Namespace    string    `gorm:"column:namespace;type:varchar(128);not null;default:'default'" json:"namespace"` // 命名空间
	DeployTarget string    `gorm:"column:deploy_target;type:varchar(16);not null;default:'k8s'" json:"deploy_target"` // 部署目标 (k8s/compose)
	PolicyJSON   string    `gorm:"column:policy_json;type:longtext" json:"policy_json"`                    // 部署策略 JSON
	IsDefault    bool      `gorm:"column:is_default;not null;default:true;index:idx_target_service_default,priority:2" json:"is_default"` // 是否默认
	UpdatedBy    uint      `gorm:"column:updated_by;default:0" json:"updated_by"`                          // 更新者 ID
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                     // 更新时间
}

// TableName 返回表名。
func (ServiceDeployTarget) TableName() string {
	return "service_deploy_targets"
}

// ServiceReleaseRecord 表示服务发布记录模型。
//
// 记录服务部署发布的历史记录。
//
// 表名: service_release_records
type ServiceReleaseRecord struct {
	ID                uint      `gorm:"primaryKey;column:id" json:"id"`                                         // 记录 ID
	ServiceID         uint      `gorm:"column:service_id;not null;index" json:"service_id"`                     // 服务 ID
	RevisionID        uint      `gorm:"column:revision_id;not null;default:0" json:"revision_id"`               // 版本 ID
	ClusterID         uint      `gorm:"column:cluster_id;not null;default:0" json:"cluster_id"`                 // 集群 ID
	Namespace         string    `gorm:"column:namespace;type:varchar(128);not null;default:'default'" json:"namespace"` // 命名空间
	Env               string    `gorm:"column:env;type:varchar(32);not null;default:'staging'" json:"env"`      // 环境名称
	DeployTarget      string    `gorm:"column:deploy_target;type:varchar(16);not null;default:'k8s'" json:"deploy_target"` // 部署目标
	Status            string    `gorm:"column:status;type:varchar(32);not null;default:'created'" json:"status"` // 状态
	RenderedYAML      string    `gorm:"column:rendered_yaml;type:longtext" json:"rendered_yaml"`                // 渲染后的 YAML
	VariablesSnapshot string    `gorm:"column:variables_snapshot_json;type:longtext" json:"variables_snapshot_json"` // 变量快照 JSON
	Error             string    `gorm:"column:error;type:longtext" json:"error"`                                // 错误信息
	Operator          uint      `gorm:"column:operator;not null;default:0" json:"operator"`                     // 操作者 ID
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                     // 创建时间
}

// TableName 返回表名。
func (ServiceReleaseRecord) TableName() string {
	return "service_release_records"
}
