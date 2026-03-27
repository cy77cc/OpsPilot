// Package model 定义 CI/CD 持续集成部署相关的数据库模型。
//
// 本文件包含以下模型:
//   - CICDServiceCIConfig: 服务 CI 配置
//   - CICDServiceCIRun: CI 运行记录
//   - CICDDeploymentCDConfig: 部署 CD 配置
//   - CICDRelease: 发布记录
//   - CICDReleaseApproval: 发布审批记录
//   - CICDAuditEvent: 审计事件
package model

import "time"

// CICDServiceCIConfig 是服务 CI 配置表模型。
//
// 存储服务的持续集成配置信息，包括代码仓库、构建步骤、触发模式等。
//
// 表名: cicd_service_ci_configs
type CICDServiceCIConfig struct {
	ID             uint      `gorm:"primaryKey;column:id" json:"id"`                                      // 配置 ID
	ServiceID      uint      `gorm:"column:service_id;not null;index:idx_cicd_service_ci_service" json:"service_id"` // 关联服务 ID
	RepoURL        string    `gorm:"column:repo_url;type:varchar(512);not null" json:"repo_url"`         // 代码仓库地址
	Branch         string    `gorm:"column:branch;type:varchar(128);default:'main'" json:"branch"`       // 分支名称
	BuildStepsJSON string    `gorm:"column:build_steps_json;type:longtext" json:"build_steps_json"`      // 构建步骤 JSON
	ArtifactTarget string    `gorm:"column:artifact_target;type:varchar(512);not null" json:"artifact_target"` // 制品目标路径
	TriggerMode    string    `gorm:"column:trigger_mode;type:varchar(32);not null;default:'manual'" json:"trigger_mode"` // 触发模式: manual/source-event/both
	Status         string    `gorm:"column:status;type:varchar(32);not null;default:'active'" json:"status"` // 配置状态
	UpdatedBy      uint      `gorm:"column:updated_by;not null;default:0" json:"updated_by"`             // 最后更新用户 ID
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                  // 创建时间
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                  // 更新时间
}

// TableName 返回表名。
func (CICDServiceCIConfig) TableName() string { return "cicd_service_ci_configs" }

// CICDServiceCIRun 是 CI 运行记录表模型。
//
// 记录每次 CI 构建运行的详细信息，包括触发类型、状态、触发者等。
//
// 表名: cicd_service_ci_runs
type CICDServiceCIRun struct {
	ID          uint      `gorm:"primaryKey;column:id" json:"id"`                                   // 运行记录 ID
	ServiceID   uint      `gorm:"column:service_id;not null;index:idx_cicd_ci_runs_service" json:"service_id"` // 关联服务 ID
	CIConfigID  uint      `gorm:"column:ci_config_id;not null;index" json:"ci_config_id"`           // 关联 CI 配置 ID
	TriggerType string    `gorm:"column:trigger_type;type:varchar(32);not null" json:"trigger_type"` // 触发类型: manual/source-event
	Status      string    `gorm:"column:status;type:varchar(32);not null;default:'queued'" json:"status"` // 运行状态: queued/running/success/failed
	Reason      string    `gorm:"column:reason;type:varchar(512);default:''" json:"reason"`         // 触发原因
	TriggeredBy uint      `gorm:"column:triggered_by;not null;default:0;index" json:"triggered_by"` // 触发用户 ID
	TriggeredAt time.Time `gorm:"column:triggered_at;not null" json:"triggered_at"`                 // 触发时间
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`               // 创建时间
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`               // 更新时间
}

// TableName 返回表名。
func (CICDServiceCIRun) TableName() string { return "cicd_service_ci_runs" }

// CICDDeploymentCDConfig 是部署 CD 配置表模型。
//
// 存储部署目标的持续部署配置，包括发布策略、审批要求等。
//
// 表名: cicd_deployment_cd_configs
type CICDDeploymentCDConfig struct {
	ID                 uint      `gorm:"primaryKey;column:id" json:"id"`                                                        // 配置 ID
	DeploymentID       uint      `gorm:"column:deployment_id;not null;index:uk_cicd_deploy_env,priority:1" json:"deployment_id"` // 关联部署目标 ID
	Env                string    `gorm:"column:env;type:varchar(32);not null;index:uk_cicd_deploy_env_runtime,priority:2" json:"env"` // 环境名称
	RuntimeType        string    `gorm:"column:runtime_type;type:varchar(16);not null;default:'k8s';index:uk_cicd_deploy_env_runtime,priority:3" json:"runtime_type"` // 运行时类型: k8s/compose
	Strategy           string    `gorm:"column:strategy;type:varchar(32);not null;default:'rolling'" json:"strategy"`          // 发布策略: rolling/blue-green/canary
	StrategyConfigJSON string    `gorm:"column:strategy_config_json;type:longtext" json:"strategy_config_json"`                // 策略配置 JSON
	ApprovalRequired   bool      `gorm:"column:approval_required;not null;default:false" json:"approval_required"`             // 是否需要审批
	UpdatedBy          uint      `gorm:"column:updated_by;not null;default:0" json:"updated_by"`                               // 最后更新用户 ID
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                                   // 创建时间
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                                   // 更新时间
}

// TableName 返回表名。
func (CICDDeploymentCDConfig) TableName() string { return "cicd_deployment_cd_configs" }

// CICDRelease 是发布记录表模型。
//
// 记录每次发布操作的详细信息，包括版本、策略、状态、审批等。
//
// 表名: cicd_releases
type CICDRelease struct {
	ID                    uint       `gorm:"primaryKey;column:id" json:"id"`                                              // 发布记录 ID
	ServiceID             uint       `gorm:"column:service_id;not null;index:idx_cicd_release_service" json:"service_id"` // 关联服务 ID
	DeploymentID          uint       `gorm:"column:deployment_id;not null;index:idx_cicd_release_deployment" json:"deployment_id"` // 关联部署目标 ID
	Env                   string     `gorm:"column:env;type:varchar(32);not null" json:"env"`                             // 环境名称
	RuntimeType           string     `gorm:"column:runtime_type;type:varchar(16);not null;default:'k8s';index:idx_cicd_release_runtime" json:"runtime_type"` // 运行时类型
	Version               string     `gorm:"column:version;type:varchar(128);not null" json:"version"`                    // 版本号
	Strategy              string     `gorm:"column:strategy;type:varchar(32);not null" json:"strategy"`                   // 发布策略
	Status                string     `gorm:"column:status;type:varchar(32);not null;index;default:'pending_approval'" json:"status"` // 发布状态
	TriggeredBy           uint       `gorm:"column:triggered_by;not null;default:0" json:"triggered_by"`                  // 触发用户 ID
	ApprovedBy            uint       `gorm:"column:approved_by;not null;default:0" json:"approved_by"`                    // 审批用户 ID
	ApprovalComment       string     `gorm:"column:approval_comment;type:varchar(1024);default:''" json:"approval_comment"` // 审批备注
	RollbackFromReleaseID uint       `gorm:"column:rollback_from_release_id;not null;default:0" json:"rollback_from_release_id"` // 回滚来源发布 ID
	DiagnosticsJSON       string     `gorm:"column:diagnostics_json;type:longtext" json:"diagnostics_json"`               // 诊断信息 JSON
	StartedAt             *time.Time `gorm:"column:started_at" json:"started_at"`                                          // 开始时间
	FinishedAt            *time.Time `gorm:"column:finished_at" json:"finished_at"`                                        // 完成时间
	CreatedAt             time.Time  `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`                    // 创建时间
	UpdatedAt             time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                          // 更新时间
}

// TableName 返回表名。
func (CICDRelease) TableName() string { return "cicd_releases" }

// CICDReleaseApproval 是发布审批记录表模型。
//
// 记录发布审批的操作历史，包括审批人、决策和备注。
//
// 表名: cicd_release_approvals
type CICDReleaseApproval struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                                     // 审批记录 ID
	ReleaseID  uint      `gorm:"column:release_id;not null;index" json:"release_id"`                 // 关联发布 ID
	ApproverID uint      `gorm:"column:approver_id;not null;default:0" json:"approver_id"`           // 审批用户 ID
	Decision   string    `gorm:"column:decision;type:varchar(32);not null" json:"decision"`          // 审批决策: approved/rejected
	Comment    string    `gorm:"column:comment;type:varchar(1024);default:''" json:"comment"`        // 审批备注
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`           // 创建时间
}

// TableName 返回表名。
func (CICDReleaseApproval) TableName() string { return "cicd_release_approvals" }

// CICDAuditEvent 是审计事件表模型。
//
// 记录 CI/CD 操作的审计日志，用于追溯和合规审计。
//
// 表名: cicd_audit_events
type CICDAuditEvent struct {
	ID               uint      `gorm:"primaryKey;column:id" json:"id"`                              // 审计事件 ID
	ServiceID        uint      `gorm:"column:service_id;not null;default:0;index" json:"service_id"` // 关联服务 ID
	DeploymentID     uint      `gorm:"column:deployment_id;not null;default:0;index" json:"deployment_id"` // 关联部署目标 ID
	ReleaseID        uint      `gorm:"column:release_id;not null;default:0;index" json:"release_id"` // 关联发布 ID
	EventType        string    `gorm:"column:event_type;type:varchar(64);not null;index" json:"event_type"` // 事件类型
	ActorID          uint      `gorm:"column:actor_id;not null;default:0;index" json:"actor_id"`   // 操作用户 ID
	CommandID        string    `gorm:"column:command_id;type:varchar(96);index" json:"command_id"` // AI 命令 ID
	Intent           string    `gorm:"column:intent;type:varchar(128);index" json:"intent"`        // AI 意图
	PlanHash         string    `gorm:"column:plan_hash;type:varchar(96);index" json:"plan_hash"`   // 计划哈希
	TraceID          string    `gorm:"column:trace_id;type:varchar(96);index" json:"trace_id"`     // 追踪 ID
	ApprovalContext  string    `gorm:"column:approval_context;type:longtext" json:"approval_context"` // 审批上下文 JSON
	ExecutionSummary string    `gorm:"column:execution_summary;type:text" json:"execution_summary"` // 执行摘要
	PayloadJSON      string    `gorm:"column:payload_json;type:longtext" json:"payload_json"`      // 事件负载 JSON
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`   // 创建时间
}

// TableName 返回表名。
func (CICDAuditEvent) TableName() string { return "cicd_audit_events" }
