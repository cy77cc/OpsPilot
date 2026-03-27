// Package model 提供数据库模型定义。
//
// 本文件包含主机运维相关的数据库模型。
package model

import "time"

// HostHealthSnapshot 主机健康检查快照模型。
//
// 存储主机健康检查的结果快照，包括连接状态、资源使用、系统状态等。
// 用于历史趋势分析和告警判断。
//
// 表名: host_health_snapshots
type HostHealthSnapshot struct {
	// ID 主键 ID
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	// HostID 关联的主机 ID
	HostID uint64 `gorm:"column:host_id;index" json:"host_id"`

	// State 整体健康状态 (healthy/degraded/critical/unknown)
	State string `gorm:"column:state;type:varchar(32);index" json:"state"`

	// ConnectivityStatus 连接状态 (healthy/critical/unknown)
	ConnectivityStatus string `gorm:"column:connectivity_status;type:varchar(32)" json:"connectivity_status"`

	// ResourceStatus 资源状态 (healthy/degraded/critical/unknown)
	ResourceStatus string `gorm:"column:resource_status;type:varchar(32)" json:"resource_status"`

	// SystemStatus 系统状态 (healthy/degraded/critical/unknown)
	SystemStatus string `gorm:"column:system_status;type:varchar(32)" json:"system_status"`

	// LatencyMS 连接延迟（毫秒）
	LatencyMS int64 `gorm:"column:latency_ms" json:"latency_ms"`

	// CpuLoad CPU 负载（1 分钟平均）
	CpuLoad float64 `gorm:"column:cpu_load" json:"cpu_load"`

	// MemoryUsedMB 已使用内存（MB）
	MemoryUsedMB int `gorm:"column:memory_used_mb" json:"memory_used_mb"`

	// MemoryTotalMB 总内存（MB）
	MemoryTotalMB int `gorm:"column:memory_total_mb" json:"memory_total_mb"`

	// DiskUsedPct 磁盘使用百分比
	DiskUsedPct float64 `gorm:"column:disk_used_pct" json:"disk_used_pct"`

	// InodeUsedPct Inode 使用百分比
	InodeUsedPct float64 `gorm:"column:inode_used_pct" json:"inode_used_pct"`

	// SummaryJSON 检查详情 JSON
	SummaryJSON string `gorm:"column:summary_json;type:longtext" json:"summary_json"`

	// ErrorMessage 错误信息
	ErrorMessage string `gorm:"column:error_message;type:text" json:"error_message"`

	// CheckedAt 检查时间
	CheckedAt time.Time `gorm:"column:checked_at;index" json:"checked_at"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 返回表名。
func (HostHealthSnapshot) TableName() string { return "host_health_snapshots" }

// AIHostExecutionRecord AI 主机执行记录模型。
//
// 存储 AI 命令/脚本任务在各主机上的执行结果。
// 用于审计和问题排查。
//
// 表名: ai_host_execution_records
type AIHostExecutionRecord struct {
	// ID 主键 ID
	ID uint64 `gorm:"column:id;primaryKey;autoIncrement" json:"id"`

	// ExecutionID 执行批次 ID
	ExecutionID string `gorm:"column:execution_id;type:varchar(64);index" json:"execution_id"`

	// CommandID 命令 ID
	CommandID string `gorm:"column:command_id;type:varchar(64);index" json:"command_id"`

	// HostID 关联的主机 ID
	HostID uint64 `gorm:"column:host_id;index" json:"host_id"`

	// HostIP 主机 IP 地址
	HostIP string `gorm:"column:host_ip;type:varchar(64)" json:"host_ip"`

	// HostName 主机名称
	HostName string `gorm:"column:host_name;type:varchar(128)" json:"host_name"`

	// CommandText 执行的命令文本
	CommandText string `gorm:"column:command_text;type:text" json:"command_text"`

	// ScriptPath 脚本路径（如果是脚本执行）
	ScriptPath string `gorm:"column:script_path;type:varchar(256)" json:"script_path"`

	// Status 执行状态 (pending/running/success/failed/timeout)
	Status string `gorm:"column:status;type:varchar(32);index" json:"status"`

	// StdoutText 标准输出
	StdoutText string `gorm:"column:stdout_text;type:longtext" json:"stdout_text"`

	// StderrText 标准错误
	StderrText string `gorm:"column:stderr_text;type:longtext" json:"stderr_text"`

	// ExitCode 退出码
	ExitCode int `gorm:"column:exit_code" json:"exit_code"`

	// StartedAt 开始时间
	StartedAt *time.Time `gorm:"column:started_at" json:"started_at"`

	// FinishedAt 结束时间
	FinishedAt *time.Time `gorm:"column:finished_at" json:"finished_at"`

	// PolicyJSON 执行策略 JSON（审批信息等）
	PolicyJSON string `gorm:"column:policy_json;type:longtext" json:"policy_json"`

	// CreatedBy 创建者用户 ID
	CreatedBy uint64 `gorm:"column:created_by" json:"created_by"`

	// CreatedAt 创建时间
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 返回表名。
func (AIHostExecutionRecord) TableName() string { return "ai_host_execution_records" }
