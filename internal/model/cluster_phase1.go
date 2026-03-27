// Package model 提供数据库模型定义。
//
// 本文件定义集群相关的扩展数据模型，包括命名空间绑定、发布记录、
// HPA 策略、配额策略、部署审批和操作审计等。
package model

import "time"

// ClusterNamespaceBinding 集群命名空间绑定表模型，管理团队与命名空间的访问权限。
//
// 表名: cluster_namespace_bindings
// 索引:
//   - idx_cluster_team_ns: (cluster_id, team_id, namespace) 复合索引
// 用途: 控制团队对特定集群命名空间的访问权限
type ClusterNamespaceBinding struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`                                          // 绑定 ID
	ClusterID uint      `gorm:"column:cluster_id;not null;index:idx_cluster_team_ns,priority:1" json:"cluster_id"` // 集群 ID
	TeamID    uint      `gorm:"column:team_id;not null;index:idx_cluster_team_ns,priority:2" json:"team_id"`       // 团队 ID
	Namespace string    `gorm:"column:namespace;type:varchar(128);not null;index:idx_cluster_team_ns,priority:3" json:"namespace"` // 命名空间名称
	Env       string    `gorm:"column:env;type:varchar(32);default:''" json:"env"`                       // 环境标识: development/staging/production
	Readonly  bool      `gorm:"column:readonly;not null;default:false" json:"readonly"`                  // 是否只读
	CreatedBy uint      `gorm:"column:created_by;not null;default:0" json:"created_by"`                  // 创建人 ID
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                      // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                      // 更新时间
}

// TableName 返回集群命名空间绑定表名。
func (ClusterNamespaceBinding) TableName() string {
	return "cluster_namespace_bindings"
}

// ClusterReleaseRecord 集群发布记录表模型，存储应用发布历史。
//
// 表名: cluster_release_records
// 用途: 记录应用部署、升级、回滚等操作历史
type ClusterReleaseRecord struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`                                     // 记录 ID
	ClusterID   uint      `gorm:"column:cluster_id;not null;index" json:"cluster_id"`                 // 集群 ID
	Namespace   string    `gorm:"column:namespace;type:varchar(128);not null;index" json:"namespace"` // 命名空间
	App         string    `gorm:"column:app;type:varchar(128);not null" json:"app"`                   // 应用名称
	Strategy    string    `gorm:"column:strategy;type:varchar(32);not null;default:'rolling'" json:"strategy"` // 发布策略: rolling/blue-green/canary
	RolloutName string    `gorm:"column:rollout_name;type:varchar(128);not null;default:''" json:"rollout_name"` // Argo Rollout 名称
	Revision    int       `gorm:"column:revision;not null;default:1" json:"revision"`                 // 版本号
	Status      string    `gorm:"column:status;type:varchar(32);not null;default:'pending'" json:"status"` // 状态: pending/running/succeeded/failed
	Operator    string    `gorm:"column:operator;type:varchar(64);not null;default:''" json:"operator"` // 操作人
	PayloadJSON string    `gorm:"column:payload_json;type:longtext" json:"payload_json"`              // 发布参数 (JSON)
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                 // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                 // 更新时间
}

// TableName 返回集群发布记录表名。
func (ClusterReleaseRecord) TableName() string {
	return "cluster_release_records"
}

// ClusterHPAPolicy 集群 HPA 策略表模型，存储水平 Pod 自动伸缩配置。
//
// 表名: cluster_hpa_policies
// 索引:
//   - idx_cluster_ns_hpa: (cluster_id, namespace, name) 复合索引
// 用途: 持久化 HPA 配置，支持策略复用和审计
type ClusterHPAPolicy struct {
	ID                uint      `gorm:"primaryKey;column:id" json:"id"`                                              // 策略 ID
	ClusterID         uint      `gorm:"column:cluster_id;not null;index:idx_cluster_ns_hpa,priority:1" json:"cluster_id"` // 集群 ID
	Namespace         string    `gorm:"column:namespace;type:varchar(128);not null;index:idx_cluster_ns_hpa,priority:2" json:"namespace"` // 命名空间
	Name              string    `gorm:"column:name;type:varchar(128);not null;index:idx_cluster_ns_hpa,priority:3" json:"name"` // HPA 名称
	TargetRefKind     string    `gorm:"column:target_ref_kind;type:varchar(64);not null" json:"target_ref_kind"`     // 目标资源类型: Deployment/StatefulSet
	TargetRefName     string    `gorm:"column:target_ref_name;type:varchar(128);not null" json:"target_ref_name"`    // 目标资源名称
	MinReplicas       int32     `gorm:"column:min_replicas;not null;default:1" json:"min_replicas"`                  // 最小副本数
	MaxReplicas       int32     `gorm:"column:max_replicas;not null;default:1" json:"max_replicas"`                  // 最大副本数
	CPUUtilization    *int32    `gorm:"column:cpu_utilization" json:"cpu_utilization,omitempty"`                     // CPU 使用率阈值 (%)
	MemoryUtilization *int32    `gorm:"column:memory_utilization" json:"memory_utilization,omitempty"`               // 内存使用率阈值 (%)
	RawPolicyJSON     string    `gorm:"column:raw_policy_json;type:longtext" json:"raw_policy_json"`                 // 原始策略 (JSON)
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                          // 创建时间
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                          // 更新时间
}

// TableName 返回集群 HPA 策略表名。
func (ClusterHPAPolicy) TableName() string {
	return "cluster_hpa_policies"
}

// ClusterQuotaPolicy 集群配额策略表模型，存储资源配额和限制范围配置。
//
// 表名: cluster_quota_policies
// 索引:
//   - idx_cluster_ns_quota: (cluster_id, namespace, name) 复合索引
// 用途: 管理 ResourceQuota 和 LimitRange 配置
type ClusterQuotaPolicy struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`                                              // 策略 ID
	ClusterID uint      `gorm:"column:cluster_id;not null;index:idx_cluster_ns_quota,priority:1" json:"cluster_id"` // 集群 ID
	Namespace string    `gorm:"column:namespace;type:varchar(128);not null;index:idx_cluster_ns_quota,priority:2" json:"namespace"` // 命名空间
	Name      string    `gorm:"column:name;type:varchar(128);not null;index:idx_cluster_ns_quota,priority:3" json:"name"` // 策略名称
	Type      string    `gorm:"column:type;type:varchar(32);not null;default:'resourcequota'" json:"type"`   // 类型: resourcequota/limitrange
	SpecJSON  string    `gorm:"column:spec_json;type:longtext" json:"spec_json"`                             // 配置规格 (JSON)
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                          // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                          // 更新时间
}

// TableName 返回集群配额策略表名。
func (ClusterQuotaPolicy) TableName() string {
	return "cluster_quota_policies"
}

// ClusterDeployApproval 集群部署审批表模型，管理生产环境部署的审批流程。
//
// 表名: cluster_deploy_approvals
// 用途: 实现生产环境部署的审批机制，确保操作安全
type ClusterDeployApproval struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`                                   // 审批 ID
	Ticket    string    `gorm:"column:ticket;type:varchar(96);not null;uniqueIndex" json:"ticket"` // 审批票据 (唯一)
	ClusterID uint      `gorm:"column:cluster_id;not null;index" json:"cluster_id"`               // 集群 ID
	Namespace string    `gorm:"column:namespace;type:varchar(128);not null" json:"namespace"`     // 命名空间
	Action    string    `gorm:"column:action;type:varchar(32);not null" json:"action"`            // 操作类型: deploy/rollback/restart
	Status    string    `gorm:"column:status;type:varchar(32);not null;default:'pending'" json:"status"` // 状态: pending/approved/rejected
	RequestBy uint      `gorm:"column:request_by;not null;default:0" json:"request_by"`           // 申请人 ID
	ReviewBy  uint      `gorm:"column:review_by;not null;default:0" json:"review_by"`             // 审批人 ID
	ExpiresAt time.Time `gorm:"column:expires_at" json:"expires_at"`                              // 过期时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`               // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`               // 更新时间
}

// TableName 返回集群部署审批表名。
func (ClusterDeployApproval) TableName() string {
	return "cluster_deploy_approvals"
}

// ClusterOperationAudit 集群操作审计表模型，记录集群操作的审计日志。
//
// 表名: cluster_operation_audits
// 用途: 提供操作追溯和合规审计能力
type ClusterOperationAudit struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                                     // 审计 ID
	ClusterID  uint      `gorm:"column:cluster_id;not null;index" json:"cluster_id"`                 // 集群 ID
	Namespace  string    `gorm:"column:namespace;type:varchar(128);not null;default:''" json:"namespace"` // 命名空间
	Action     string    `gorm:"column:action;type:varchar(64);not null;index" json:"action"`        // 操作类型
	Resource   string    `gorm:"column:resource;type:varchar(64);not null;default:''" json:"resource"` // 资源类型
	ResourceID string    `gorm:"column:resource_id;type:varchar(128);not null;default:''" json:"resource_id"` // 资源标识
	Status     string    `gorm:"column:status;type:varchar(32);not null;default:'success'" json:"status"` // 执行状态: success/failed
	Message    string    `gorm:"column:message;type:varchar(255);not null;default:''" json:"message"` // 操作消息
	OperatorID uint      `gorm:"column:operator_id;not null;default:0" json:"operator_id"`           // 操作人 ID
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                 // 创建时间
}

// TableName 返回集群操作审计表名。
func (ClusterOperationAudit) TableName() string {
	return "cluster_operation_audits"
}
