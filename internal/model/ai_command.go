// Package model 提供数据库模型定义。
//
// 本文件定义 AI 命令执行相关的数据模型，用于命令桥的预览和执行历史记录。
package model

import "time"

// AICommandExecution 是 AI 命令执行表模型，存储命令桥预览/执行历史。
//
// 表名: ai_command_executions
// 关联: User (通过 user_id)
//
// 状态流转:
//   - previewing: 预览中
//   - pending: 等待执行
//   - running: 执行中
//   - success: 执行成功
//   - failed: 执行失败
//   - cancelled: 已取消
//
// 用途:
//   - 记录 AI 生成的命令预览
//   - 跟踪命令执行状态和结果
//   - 提供命令历史查询和审计
type AICommandExecution struct {
	ID               string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                                     // 命令执行唯一标识
	UserID           uint64    `gorm:"column:user_id;index:idx_ai_cmd_user_created" json:"user_id"`                         // 请求用户 ID
	Scene            string    `gorm:"column:scene;type:varchar(128);index" json:"scene"`                                   // 场景标识 (如: host, k8s)
	CommandText      string    `gorm:"column:command_text;type:text" json:"command_text"`                                  // 命令文本
	Intent           string    `gorm:"column:intent;type:varchar(128);index" json:"intent"`                                // 意图类型 (如: restart, deploy)
	PlanHash         string    `gorm:"column:plan_hash;type:varchar(96);index" json:"plan_hash"`                           // 执行计划哈希 (用于去重)
	Risk             string    `gorm:"column:risk;type:varchar(16);index" json:"risk"`                                     // 风险等级: high/medium/low
	Status           string    `gorm:"column:status;type:varchar(32);index" json:"status"`                                 // 状态: previewing/pending/running/success/failed/cancelled
	TraceID          string    `gorm:"column:trace_id;type:varchar(96);index" json:"trace_id"`                             // 链路追踪 ID
	ParamsJSON       string    `gorm:"column:params_json;type:longtext" json:"params_json"`                                // 参数 (JSON 格式)
	MissingJSON      string    `gorm:"column:missing_json;type:longtext" json:"missing_json"`                              // 缺失参数 (JSON 格式)
	PlanJSON         string    `gorm:"column:plan_json;type:longtext" json:"plan_json"`                                    // 执行计划 (JSON 格式)
	ResultJSON       string    `gorm:"column:result_json;type:longtext" json:"result_json"`                                // 执行结果 (JSON 格式)
	ApprovalContext  string    `gorm:"column:approval_context;type:longtext" json:"approval_context"`                      // 审批上下文 (JSON 格式)
	ExecutionSummary string    `gorm:"column:execution_summary;type:text" json:"execution_summary"`                        // 执行摘要
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime;index:idx_ai_cmd_user_created" json:"created_at"`   // 创建时间
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                                 // 更新时间
}

// TableName 返回 AI 命令执行表名。
func (AICommandExecution) TableName() string { return "ai_command_executions" }
