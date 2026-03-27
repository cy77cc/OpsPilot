// Package model 提供数据库模型定义。
//
// 本文件定义部署管理相关的数据模型，包括部署目标、发布记录和审批流程。
package model

import "time"

// DeploymentTarget 是部署目标表模型，定义服务部署的目标环境。
//
// 表名: deployment_targets
// 用途: 定义服务可以部署的集群或主机集合
//
// 类型说明:
//   - TargetType: k8s (Kubernetes) / compose (Docker Compose)
//   - RuntimeType: k8s / compose
//   - ClusterSource: platform_managed / external_managed
type DeploymentTarget struct {
	ID              uint      `gorm:"primaryKey;column:id" json:"id"`                                                // 目标 ID
	Name            string    `gorm:"column:name;type:varchar(128);not null" json:"name"`                           // 目标名称
	TargetType      string    `gorm:"column:target_type;type:varchar(16);not null;index" json:"target_type"`       // 目标类型: k8s/compose
	RuntimeType     string    `gorm:"column:runtime_type;type:varchar(16);not null;default:'k8s';index" json:"runtime_type"` // 运行时类型: k8s/compose
	ClusterID       uint      `gorm:"column:cluster_id;default:0;index" json:"cluster_id"`                          // 集群 ID
	ClusterSource   string    `gorm:"column:cluster_source;type:varchar(32);not null;default:'platform_managed';index" json:"cluster_source"` // 集群来源
	CredentialID    uint      `gorm:"column:credential_id;default:0;index" json:"credential_id"`                    // 凭证 ID
	BootstrapJobID  string    `gorm:"column:bootstrap_job_id;type:varchar(64);default:''" json:"bootstrap_job_id"` // 初始化任务 ID
	ProjectID       uint      `gorm:"column:project_id;default:0;index" json:"project_id"`                         // 项目 ID
	TeamID          uint      `gorm:"column:team_id;default:0;index" json:"team_id"`                              // 团队 ID
	Env             string    `gorm:"column:env;type:varchar(32);default:'staging';index" json:"env"`             // 环境: development/staging/production
	Status          string    `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`              // 状态: active/inactive
	ReadinessStatus string    `gorm:"column:readiness_status;type:varchar(32);default:'unknown'" json:"readiness_status"` // 就绪状态: ready/not_ready/unknown
	CreatedBy       uint      `gorm:"column:created_by;default:0" json:"created_by"`                              // 创建人 ID
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                         // 创建时间
	UpdatedAt       time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                         // 更新时间
}

// TableName 返回部署目标表名。
func (DeploymentTarget) TableName() string { return "deployment_targets" }

// DeploymentTargetNode 是部署目标节点关联表模型，定义目标包含的主机。
//
// 表名: deployment_target_nodes
// 用途: 为 Docker Compose 场景指定部署主机
type DeploymentTargetNode struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`                                     // 关联 ID
	TargetID  uint      `gorm:"column:target_id;not null;index:idx_target_host,priority:1" json:"target_id"` // 目标 ID
	HostID    uint      `gorm:"column:host_id;not null;index:idx_target_host,priority:2" json:"host_id"` // 主机 ID
	Role      string    `gorm:"column:role;type:varchar(16);default:'worker'" json:"role"`          // 角色: manager/worker
	Weight    int       `gorm:"column:weight;default:100" json:"weight"`                           // 权重 (负载均衡)
	Status    string    `gorm:"column:status;type:varchar(32);default:'active'" json:"status"`     // 状态: active/inactive
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                // 更新时间
}

// TableName 返回部署目标节点关联表名。
func (DeploymentTargetNode) TableName() string { return "deployment_target_nodes" }

// DeploymentRelease 是部署发布记录表模型，记录每次部署的详细信息。
//
// 表名: deployment_releases
// 关联:
//   - Service (多对一，通过 service_id)
//   - DeploymentTarget (多对一，通过 target_id)
//
// 状态流转:
//   - pending_approval: 等待审批
//   - approved: 已批准
//   - deploying: 部署中
//   - success: 部署成功
//   - failed: 部署失败
//   - rolled_back: 已回滚
type DeploymentRelease struct {
	ID                 uint       `gorm:"primaryKey;column:id" json:"id"`                                              // 发布 ID
	ServiceID          uint       `gorm:"column:service_id;not null;index" json:"service_id"`                         // 服务 ID
	TargetID           uint       `gorm:"column:target_id;not null;index" json:"target_id"`                           // 目标 ID
	NamespaceOrProject string     `gorm:"column:namespace_or_project;type:varchar(128);default:''" json:"namespace_or_project"` // 命名空间/项目
	RuntimeType        string     `gorm:"column:runtime_type;type:varchar(16);not null;index" json:"runtime_type"`    // 运行时类型: k8s/compose
	Strategy           string     `gorm:"column:strategy;type:varchar(16);default:'rolling'" json:"strategy"`        // 部署策略: rolling/recreate
	TriggerSource      string     `gorm:"column:trigger_source;type:varchar(32);not null;default:'manual';index" json:"trigger_source"` // 触发来源: manual/ci/scheduled
	RevisionID         uint       `gorm:"column:revision_id;default:0;index" json:"revision_id"`                     // 配置版本 ID
	SourceReleaseID    uint       `gorm:"column:source_release_id;default:0;index" json:"source_release_id"`         // 源发布 ID (回滚场景)
	TargetRevision     string     `gorm:"column:target_revision;type:varchar(128);default:''" json:"target_revision"` // 目标版本号
	PreviewContextHash string     `gorm:"column:preview_context_hash;type:varchar(128);default:''" json:"preview_context_hash"` // 预览上下文哈希
	PreviewTokenHash   string     `gorm:"column:preview_token_hash;type:varchar(128);default:''" json:"preview_token_hash"` // 预览令牌哈希
	PreviewExpiresAt   *time.Time `gorm:"column:preview_expires_at" json:"preview_expires_at"`                       // 预览过期时间
	Status             string     `gorm:"column:status;type:varchar(32);default:'pending_approval';index" json:"status"` // 状态
	ManifestSnapshot   string     `gorm:"column:manifest_snapshot;type:longtext" json:"manifest_snapshot"`           // 清单快照 (YAML)
	RuntimeContextJSON string     `gorm:"column:runtime_context_json;type:longtext" json:"runtime_context_json"`     // 运行时上下文 (JSON)
	TriggerContextJSON string     `gorm:"column:trigger_context_json;type:longtext" json:"trigger_context_json"`     // 触发上下文 (JSON)
	ChecksJSON         string     `gorm:"column:checks_json;type:longtext" json:"checks_json"`                       // 检查项 (JSON)
	WarningsJSON       string     `gorm:"column:warnings_json;type:longtext" json:"warnings_json"`                   // 警告项 (JSON)
	DiagnosticsJSON    string     `gorm:"column:diagnostics_json;type:longtext" json:"diagnostics_json"`             // 诊断信息 (JSON)
	VerificationJSON   string     `gorm:"column:verification_json;type:longtext" json:"verification_json"`           // 验证结果 (JSON)
	Operator           uint       `gorm:"column:operator;default:0;index" json:"operator"`                           // 操作人 ID
	CIRunID            uint       `gorm:"column:ci_run_id;default:0;index:idx_deploy_release_ci_run" json:"ci_run_id"` // CI 运行 ID
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`                  // 创建时间
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                        // 更新时间
}

// TableName 返回部署发布记录表名。
func (DeploymentRelease) TableName() string { return "deployment_releases" }

// DeploymentReleaseApproval 是部署发布审批表模型，记录审批流程。
//
// 表名: deployment_release_approvals
// 状态: pending / approved / rejected
type DeploymentReleaseApproval struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`                                     // 审批 ID
	ReleaseID   uint      `gorm:"column:release_id;not null;index" json:"release_id"`                 // 发布 ID
	Ticket      string    `gorm:"column:ticket;type:varchar(96);not null;uniqueIndex" json:"ticket"` // 审批单号
	Decision    string    `gorm:"column:decision;type:varchar(32);not null;default:'pending';index" json:"decision"` // 决定: pending/approved/rejected
	Comment     string    `gorm:"column:comment;type:varchar(1024);default:''" json:"comment"`       // 审批意见
	RequestedBy uint      `gorm:"column:requested_by;default:0" json:"requested_by"`                 // 请求人 ID
	ApproverID  uint      `gorm:"column:approver_id;default:0" json:"approver_id"`                   // 审批人 ID
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`           // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                 // 更新时间
}

// TableName 返回部署发布审批表名。
func (DeploymentReleaseApproval) TableName() string { return "deployment_release_approvals" }

// DeploymentReleaseAudit 是部署发布审计表模型，记录发布操作日志。
//
// 表名: deployment_release_audits
// 用途: 审计追踪
type DeploymentReleaseAudit struct {
	ID            uint      `gorm:"primaryKey;column:id" json:"id"`                                  // 审计 ID
	ReleaseID     uint      `gorm:"column:release_id;not null;index" json:"release_id"`              // 发布 ID
	CorrelationID string    `gorm:"column:correlation_id;type:varchar(96);index" json:"correlation_id"` // 关联 ID
	TraceID       string    `gorm:"column:trace_id;type:varchar(96);index" json:"trace_id"`          // 链路追踪 ID
	Action        string    `gorm:"column:action;type:varchar(64);not null;index" json:"action"`     // 操作类型
	Actor         uint      `gorm:"column:actor;default:0" json:"actor"`                             // 操作人 ID
	DetailJSON    string    `gorm:"column:detail_json;type:longtext" json:"detail_json"`             // 详情 (JSON)
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`        // 创建时间
}

// TableName 返回部署发布审计表名。
func (DeploymentReleaseAudit) TableName() string { return "deployment_release_audits" }

// ServiceGovernancePolicy 是服务治理策略表模型，定义服务的流量、弹性等策略。
//
// 表名: service_governance_policies
// 用途: 服务网格治理配置
type ServiceGovernancePolicy struct {
	ID                   uint      `gorm:"primaryKey;column:id" json:"id"`                                                        // 策略 ID
	ServiceID            uint      `gorm:"column:service_id;not null;index:idx_service_env_governance,priority:1" json:"service_id"` // 服务 ID
	Env                  string    `gorm:"column:env;type:varchar(32);not null;index:idx_service_env_governance,priority:2" json:"env"` // 环境
	TrafficPolicyJSON    string    `gorm:"column:traffic_policy_json;type:longtext" json:"traffic_policy_json"`                  // 流量策略 (JSON)
	ResiliencePolicyJSON string    `gorm:"column:resilience_policy_json;type:longtext" json:"resilience_policy_json"`            // 弹性策略 (JSON)
	AccessPolicyJSON     string    `gorm:"column:access_policy_json;type:longtext" json:"access_policy_json"`                    // 访问策略 (JSON)
	SLOPolicyJSON        string    `gorm:"column:slo_policy_json;type:longtext" json:"slo_policy_json"`                          // SLO 策略 (JSON)
	UpdatedBy            uint      `gorm:"column:updated_by;default:0" json:"updated_by"`                                        // 更新人 ID
	UpdatedAt            time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                                    // 更新时间
}

// TableName 返回服务治理策略表名。
func (ServiceGovernancePolicy) TableName() string { return "service_governance_policies" }

// AIOPSInspection 是 AIOps 巡检表模型，记录部署前后的智能巡检结果。
//
// 表名: aiops_inspections
// 用途: 自动化部署质量检查
//
// 阶段说明:
//   - pre: 部署前检查
//   - post: 部署后检查
//   - periodic: 周期性巡检
type AIOPSInspection struct {
	ID              uint      `gorm:"primaryKey;column:id" json:"id"`                              // 巡检 ID
	ReleaseID       uint      `gorm:"column:release_id;default:0;index" json:"release_id"`        // 发布 ID
	TargetID        uint      `gorm:"column:target_id;default:0;index" json:"target_id"`          // 目标 ID
	ServiceID       uint      `gorm:"column:service_id;default:0;index" json:"service_id"`        // 服务 ID
	Stage           string    `gorm:"column:stage;type:varchar(16);not null" json:"stage"`        // 阶段: pre/post/periodic
	Summary         string    `gorm:"column:summary;type:text" json:"summary"`                    // 摘要
	FindingsJSON    string    `gorm:"column:findings_json;type:longtext" json:"findings_json"`    // 发现问题 (JSON)
	SuggestionsJSON string    `gorm:"column:suggestions_json;type:longtext" json:"suggestions_json"` // 优化建议 (JSON)
	Status          string    `gorm:"column:status;type:varchar(32);default:'done'" json:"status"` // 状态
	CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`   // 创建时间
}

// TableName 返回 AIOps 巡检表名。
func (AIOPSInspection) TableName() string { return "aiops_inspections" }
