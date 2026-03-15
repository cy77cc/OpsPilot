// Package model 提供数据模型定义。
package model

import "time"

// AIUsageLog 记录单次 AI 执行的统计数据。
//
// 与 ai_executions 表的区别：
//   - ai_executions: 工具执行级别，每次 Tool 调用一条
//   - ai_usage_logs: 请求级别，每次 Run/Resume 一条汇总
type AIUsageLog struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	TraceID   string `gorm:"column:trace_id;type:varchar(36);not null"`
	SessionID string `gorm:"column:session_id;type:varchar(36);not null;index:idx_ai_usage_logs_session_id"`
	PlanID    string `gorm:"column:plan_id;type:varchar(36);not null"`
	TurnID    string `gorm:"column:turn_id;type:varchar(36)"`
	UserID    uint64 `gorm:"column:user_id;index:idx_ai_usage_logs_user_created"`

	Scene     string `gorm:"column:scene;type:varchar(64);index:idx_ai_usage_logs_scene"`
	Operation string `gorm:"column:operation;type:varchar(32)"`
	Status    string `gorm:"column:status;type:varchar(32);index:idx_ai_usage_logs_status"`

	PromptTokens     int     `gorm:"column:prompt_tokens;default:0"`
	CompletionTokens int     `gorm:"column:completion_tokens;default:0"`
	TotalTokens      int     `gorm:"column:total_tokens;default:0"`
	EstimatedCostUSD float64 `gorm:"column:estimated_cost_usd;type:decimal(10,6)"`
	ModelName        string  `gorm:"column:model_name;type:varchar(128)"`

	DurationMs      int     `gorm:"column:duration_ms"`
	FirstTokenMs    int     `gorm:"column:first_token_ms"`
	TokensPerSecond float64 `gorm:"column:tokens_per_second;type:decimal(10,2)"`

	ApprovalCount  int    `gorm:"column:approval_count;default:0"`
	ApprovalStatus string `gorm:"column:approval_status;type:varchar(32);default:'none'"`
	ApprovalWaitMs int    `gorm:"column:approval_wait_ms;default:0"`

	ToolCallCount  int `gorm:"column:tool_call_count;default:0"`
	ToolErrorCount int `gorm:"column:tool_error_count;default:0"`

	ErrorType    string `gorm:"column:error_type;type:varchar(64)"`
	ErrorMessage string `gorm:"column:error_message;type:text"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;index:idx_ai_usage_logs_created_at"`
}

// TableName 返回表名。
func (AIUsageLog) TableName() string {
	return "ai_usage_logs"
}
