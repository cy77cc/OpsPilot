// Package model 提供 CMDB 领域的数据模型定义。
//
// 本文件定义 CMDB 模块的核心数据结构，包括：
//   - CMDBCI: 配置项（资产）模型
//   - CMDBRelation: 配置项关系模型
//   - CMDBSyncJob: 同步任务模型
//   - CMDBSyncRecord: 同步记录模型
//   - CMDBAudit: 审计日志模型
package model

import (
	"time"

	"gorm.io/gorm"
)

// CMDBCI 是配置管理数据库中的配置项（CI）模型。
//
// 配置项是 CMDB 的核心实体，代表任何需要管理的 IT 资产，
// 如主机、集群、服务、网络设备等。
//
// 表名: cmdb_cis
type CMDBCI struct {
	ID           uint           `gorm:"primaryKey;column:id" json:"id"`                                          // 配置项 ID
	CIUID        string         `gorm:"column:ci_uid;type:varchar(160);not null;uniqueIndex" json:"ci_uid"`      // 配置项唯一标识 (格式: ciType:externalID)
	CIType       string         `gorm:"column:ci_type;type:varchar(64);not null;index" json:"ci_type"`           // 配置项类型 (host, cluster, service 等)
	Name         string         `gorm:"column:name;type:varchar(128);not null;index" json:"name"`                // 配置项名称
	Source       string         `gorm:"column:source;type:varchar(64);not null;default:'manual';index" json:"source"`      // 数据来源 (manual, host, cluster, service 等)
	ExternalID   string         `gorm:"column:external_id;type:varchar(160);not null;default:'';index" json:"external_id"` // 外部系统 ID
	ProjectID    uint           `gorm:"column:project_id;default:0;index" json:"project_id"`                     // 所属项目 ID
	TeamID       uint           `gorm:"column:team_id;default:0;index" json:"team_id"`                           // 所属团队 ID
	Owner        string         `gorm:"column:owner;type:varchar(128);not null;default:''" json:"owner"`         // 负责人
	Status       string         `gorm:"column:status;type:varchar(32);not null;default:'active';index" json:"status"` // 状态 (active, inactive, unknown 等)
	TagsJSON     string         `gorm:"column:tags_json;type:longtext" json:"tags_json"`                         // 标签 JSON
	AttrsJSON    string         `gorm:"column:attrs_json;type:longtext" json:"attrs_json"`                       // 扩展属性 JSON
	LastSyncedAt *time.Time     `gorm:"column:last_synced_at" json:"last_synced_at,omitempty"`                   // 最后同步时间
	CreatedBy    uint           `gorm:"column:created_by;default:0" json:"created_by"`                           // 创建者用户 ID
	UpdatedBy    uint           `gorm:"column:updated_by;default:0" json:"updated_by"`                           // 最后更新者用户 ID
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`                      // 创建时间
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                      // 更新时间
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`                                         // 软删除时间
}

// TableName 返回 CMDBCI 的数据库表名。
func (CMDBCI) TableName() string { return "cmdb_cis" }

// CMDBRelation 是配置项之间的关系模型。
//
// 用于描述配置项之间的依赖、关联关系，如服务运行在集群上、
// 主机属于某个集群等。
//
// 表名: cmdb_relations
type CMDBRelation struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`                                                       // 关系 ID
	FromCIID     uint      `gorm:"column:from_ci_id;not null;index:idx_cmdb_relation_from_to,priority:1" json:"from_ci_id"`     // 源配置项 ID
	ToCIID       uint      `gorm:"column:to_ci_id;not null;index:idx_cmdb_relation_from_to,priority:2" json:"to_ci_id"`         // 目标配置项 ID
	RelationType string    `gorm:"column:relation_type;type:varchar(64);not null;index:idx_cmdb_relation_from_to,priority:3" json:"relation_type"` // 关系类型 (runs_on, depends_on 等)
	CreatedBy    uint      `gorm:"column:created_by;default:0" json:"created_by"`                                        // 创建者用户 ID
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                                   // 创建时间
}

// TableName 返回 CMDBRelation 的数据库表名。
func (CMDBRelation) TableName() string { return "cmdb_relations" }

// CMDBSyncJob 是数据同步任务模型。
//
// 记录从外部数据源同步配置项到 CMDB 的任务执行情况，
// 包括任务状态、执行摘要和错误信息。
//
// 表名: cmdb_sync_jobs
type CMDBSyncJob struct {
	ID           string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                      // 任务 ID (格式: cmdb-sync-{timestamp})
	Source       string    `gorm:"column:source;type:varchar(64);not null;default:'all';index" json:"source"` // 数据源类型 (all, host, cluster, service 等)
	Status       string    `gorm:"column:status;type:varchar(32);not null;default:'running';index" json:"status"` // 任务状态 (running, succeeded, failed)
	SummaryJSON  string    `gorm:"column:summary_json;type:longtext" json:"summary_json"`                // 执行摘要 JSON (created, updated, unchanged, failed 数量)
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message"`                  // 错误信息
	StartedAt    time.Time `gorm:"column:started_at" json:"started_at"`                                   // 开始时间
	FinishedAt   time.Time `gorm:"column:finished_at" json:"finished_at"`                                 // 完成时间
	OperatorID   uint      `gorm:"column:operator_id;default:0;index" json:"operator_id"`                // 操作者用户 ID
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`                    // 创建时间
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                    // 更新时间
}

// TableName 返回 CMDBSyncJob 的数据库表名。
func (CMDBSyncJob) TableName() string { return "cmdb_sync_jobs" }

// CMDBSyncRecord 是同步任务中的单条记录模型。
//
// 记录同步任务中每个配置项的同步操作详情，
// 包括操作类型、状态和变更差异。
//
// 表名: cmdb_sync_records
type CMDBSyncRecord struct {
	ID           uint      `gorm:"primaryKey;column:id" json:"id"`                                // 记录 ID
	JobID        string    `gorm:"column:job_id;type:varchar(64);not null;index" json:"job_id"`   // 所属同步任务 ID
	CIUID        string    `gorm:"column:ci_uid;type:varchar(160);not null;index" json:"ci_uid"`  // 配置项唯一标识
	Action       string    `gorm:"column:action;type:varchar(32);not null;index" json:"action"`   // 操作类型 (created, updated, unchanged, failed)
	Status       string    `gorm:"column:status;type:varchar(32);not null;index" json:"status"`   // 记录状态 (ok, failed)
	DiffJSON     string    `gorm:"column:diff_json;type:longtext" json:"diff_json"`               // 变更差异 JSON
	ErrorMessage string    `gorm:"column:error_message;type:text" json:"error_message"`           // 错误信息
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`             // 创建时间
}

// TableName 返回 CMDBSyncRecord 的数据库表名。
func (CMDBSyncRecord) TableName() string { return "cmdb_sync_records" }

// CMDBAudit 是 CMDB 审计日志模型。
//
// 记录所有配置项和关系的变更操作，
// 用于追溯和合规审计。
//
// 表名: cmdb_audits
type CMDBAudit struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id"`                              // 审计记录 ID
	CIID       uint      `gorm:"column:ci_id;default:0;index" json:"ci_id"`                   // 相关配置项 ID
	RelationID uint      `gorm:"column:relation_id;default:0;index" json:"relation_id"`       // 相关关系 ID
	Action     string    `gorm:"column:action;type:varchar(64);not null;index" json:"action"` // 操作类型 (ci.create, ci.update, ci.delete, relation.create, relation.delete, sync.trigger)
	ActorID    uint      `gorm:"column:actor_id;default:0;index" json:"actor_id"`             // 操作者用户 ID
	BeforeJSON string    `gorm:"column:before_json;type:longtext" json:"before_json"`         // 变更前数据 JSON
	AfterJSON  string    `gorm:"column:after_json;type:longtext" json:"after_json"`           // 变更后数据 JSON
	Detail     string    `gorm:"column:detail;type:text" json:"detail"`                       // 操作详情 (如请求路径、同步任务 ID)
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`    // 创建时间
}

// TableName 返回 CMDBAudit 的数据库表名。
func (CMDBAudit) TableName() string { return "cmdb_audits" }
