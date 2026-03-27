// Package model 定义数据库模型。
//
// 本文件定义自动化运维相关的数据库模型，包括：
//   - AutomationInventory: 主机清单
//   - AutomationPlaybook: 自动化脚本
//   - AutomationRun: 执行记录
//   - AutomationRunLog: 执行日志
//   - AutomationExecutionAudit: 执行审计
//   - TopologyAccessAudit: 拓扑访问审计
package model

import "time"

// AutomationInventory 是自动化主机清单模型。
//
// 存储自动化任务的主机组配置，用于批量执行任务时指定目标主机。
//
// 表名: automation_inventories
type AutomationInventory struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`          // 清单 ID
	Name      string    `gorm:"column:name;type:varchar(128);not null;index" json:"name"` // 清单名称
	HostsJSON string    `gorm:"column:hosts_json;type:longtext" json:"hosts_json"`        // 主机配置 JSON
	CreatedBy uint      `gorm:"column:created_by;default:0;index" json:"created_by"`      // 创建者 ID
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`       // 创建时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`       // 更新时间
}

// TableName 返回 AutomationInventory 的表名。
//
// 返回: automation_inventories
func (AutomationInventory) TableName() string { return "automation_inventories" }

// AutomationPlaybook 是自动化 Playbook 模型。
//
// 存储 Ansible 风格的自动化脚本定义，包含执行内容和风险等级。
//
// 表名: automation_playbooks
type AutomationPlaybook struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                              // Playbook ID
	Name       string    `gorm:"column:name;type:varchar(128);not null;index" json:"name"`    // Playbook 名称
	ContentYML string    `gorm:"column:content_yml;type:longtext" json:"content_yml"`         // YAML 格式内容
	RiskLevel  string    `gorm:"column:risk_level;type:varchar(32);not null;default:'medium'" json:"risk_level"` // 风险等级 (low/medium/high)
	CreatedBy  uint      `gorm:"column:created_by;default:0;index" json:"created_by"`         // 创建者 ID
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`          // 创建时间
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`          // 更新时间
}

// TableName 返回 AutomationPlaybook 的表名。
//
// 返回: automation_playbooks
func (AutomationPlaybook) TableName() string { return "automation_playbooks" }

// AutomationRun 是自动化任务执行记录模型。
//
// 记录每次自动化任务的执行状态、参数和结果。
//
// 表名: automation_runs
type AutomationRun struct {
	ID         string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`       // 执行记录 ID（格式: run-{timestamp}）
	Action     string    `gorm:"column:action;type:varchar(128);not null;index" json:"action"` // 执行动作类型
	Status     string    `gorm:"column:status;type:varchar(32);not null;index" json:"status"`  // 执行状态 (running/succeeded/failed)
	ResultJSON string    `gorm:"column:result_json;type:longtext" json:"result_json"`          // 执行结果 JSON
	ParamsJSON string    `gorm:"column:params_json;type:longtext" json:"params_json"`          // 执行参数 JSON
	Error      string    `gorm:"column:error;type:text" json:"error"`                          // 错误信息
	OperatorID uint      `gorm:"column:operator_id;default:0;index" json:"operator_id"`        // 操作者 ID
	StartedAt  time.Time `gorm:"column:started_at;index" json:"started_at"`                    // 开始时间
	FinishedAt time.Time `gorm:"column:finished_at" json:"finished_at"`                        // 结束时间
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`           // 创建时间
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`           // 更新时间
}

// TableName 返回 AutomationRun 的表名。
//
// 返回: automation_runs
func (AutomationRun) TableName() string { return "automation_runs" }

// AutomationRunLog 是自动化任务执行日志模型。
//
// 记录任务执行过程中的详细日志，用于追踪和排查问题。
//
// 表名: automation_run_logs
type AutomationRunLog struct {
	ID        uint      `gorm:"primaryKey;column:id" json:"id"`                          // 日志 ID
	RunID     string    `gorm:"column:run_id;type:varchar(64);not null;index" json:"run_id"` // 关联的执行记录 ID
	Level     string    `gorm:"column:level;type:varchar(16);not null;default:'info'" json:"level"` // 日志级别 (info/warning/error)
	Message   string    `gorm:"column:message;type:text;not null" json:"message"`        // 日志消息
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"` // 创建时间
}

// TableName 返回 AutomationRunLog 的表名。
//
// 返回: automation_run_logs
func (AutomationRunLog) TableName() string { return "automation_run_logs" }

// AutomationExecutionAudit 是自动化执行审计模型。
//
// 记录自动化任务的执行审计信息，用于安全审计和合规追踪。
//
// 表名: automation_execution_audits
type AutomationExecutionAudit struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                              // 审计记录 ID
	RunID      string    `gorm:"column:run_id;type:varchar(64);not null;index" json:"run_id"` // 关联的执行记录 ID
	Action     string    `gorm:"column:action;type:varchar(128);not null;index" json:"action"` // 执行动作类型
	Status     string    `gorm:"column:status;type:varchar(32);not null;index" json:"status"`  // 执行状态
	ActorID    uint      `gorm:"column:actor_id;default:0;index" json:"actor_id"`              // 操作者 ID
	DetailJSON string    `gorm:"column:detail_json;type:longtext" json:"detail_json"`          // 详细信息 JSON
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`     // 创建时间
}

// TableName 返回 AutomationExecutionAudit 的表名。
//
// 返回: automation_execution_audits
func (AutomationExecutionAudit) TableName() string { return "automation_execution_audits" }

// TopologyAccessAudit 是拓扑访问审计模型。
//
// 记录用户访问拓扑资源的审计信息，用于权限审计和安全追踪。
//
// 表名: topology_access_audits
type TopologyAccessAudit struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                            // 审计记录 ID
	ActorID    uint      `gorm:"column:actor_id;default:0;index" json:"actor_id"`           // 操作者 ID
	Action     string    `gorm:"column:action;type:varchar(64);not null;index" json:"action"` // 访问动作
	Scope      string    `gorm:"column:scope;type:varchar(128);not null;index" json:"scope"` // 访问范围
	FilterJSON string    `gorm:"column:filter_json;type:longtext" json:"filter_json"`       // 过滤条件 JSON
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`  // 创建时间
}

// TableName 返回 TopologyAccessAudit 的表名。
//
// 返回: topology_access_audits
func (TopologyAccessAudit) TableName() string { return "topology_access_audits" }
