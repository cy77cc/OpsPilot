// Package model 提供数据库模型定义。
//
// 本文件定义 AI 确认请求相关的数据模型，用于存储用户对 AI 变更操作的确认状态。
package model

import "time"

// ConfirmationRequest 是确认请求表模型，存储用户对 AI 变更操作的确认状态。
//
// 表名: ai_confirmations
// 关联: User (通过 request_user_id)
//
// 状态流转:
//   - pending: 等待确认
//   - confirmed: 已确认
//   - cancelled: 已取消
//   - expired: 已过期
//
// 用途:
//   - 低风险操作的一次性确认
//   - 短期确认请求 (通常几分钟内有效)
//   - 区别于 AIApprovalTask (用于高风险操作的多级审批)
type ConfirmationRequest struct {
	ID            string     `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`                                         // 确认请求唯一标识
	RequestUserID uint64     `gorm:"column:request_user_id;index:idx_ai_confirmation_user_created" json:"request_user_id"`    // 请求用户 ID
	TraceID       string     `gorm:"column:trace_id;type:varchar(96);index" json:"trace_id"`                                  // 链路追踪 ID
	ToolName      string     `gorm:"column:tool_name;type:varchar(128);index" json:"tool_name"`                               // 工具名称
	ToolMode      string     `gorm:"column:tool_mode;type:varchar(32);index" json:"tool_mode"`                                // 工具模式: auto/confirm/preview
	RiskLevel     string     `gorm:"column:risk_level;type:varchar(16);index" json:"risk_level"`                              // 风险等级: high/medium/low
	ParamsJSON    string     `gorm:"column:params_json;type:longtext" json:"params_json"`                                     // 工具参数 (JSON 格式)
	PreviewJSON   string     `gorm:"column:preview_json;type:longtext" json:"preview_json"`                                   // 预览数据 (JSON 格式)
	Status        string     `gorm:"column:status;type:varchar(32);index" json:"status"`                                      // 状态: pending/confirmed/cancelled/expired
	Reason        string     `gorm:"column:reason;type:varchar(255)" json:"reason"`                                           // 取消原因
	ExpiresAt     time.Time  `gorm:"column:expires_at;index" json:"expires_at"`                                               // 过期时间
	ConfirmedAt   *time.Time `gorm:"column:confirmed_at" json:"confirmed_at,omitempty"`                                       // 确认时间
	CancelledAt   *time.Time `gorm:"column:cancelled_at" json:"cancelled_at,omitempty"`                                       // 取消时间
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_confirmation_user_created" json:"created_at"` // 创建时间
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`                                      // 更新时间
}

// TableName 返回确认请求表名。
func (ConfirmationRequest) TableName() string { return "ai_confirmations" }
