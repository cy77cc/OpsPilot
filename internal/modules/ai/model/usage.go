package model

import (
	"time"

	"gorm.io/gorm"
)

type AITraceSpan struct {
	ID         string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID      string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	SessionID  string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	Scene      string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Status     string         `gorm:"column:status;type:varchar(32);index" json:"status"`
	ModelName        string         `gorm:"column:model_name;type:varchar(128)" json:"model_name"`
	PromptTokens     int64          `gorm:"column:prompt_tokens;not null;default:0" json:"prompt_tokens"`
	CompletionTokens int64          `gorm:"column:completion_tokens;not null;default:0" json:"completion_tokens"`
	Tokens           int64          `gorm:"column:tokens;not null;default:0" json:"tokens"`
	DurationMS int64          `gorm:"column:duration_ms;not null;default:0" json:"duration_ms"`
	StartTime  time.Time      `gorm:"column:start_time;not null;index" json:"start_time"`
	EndTime    *time.Time     `gorm:"column:end_time" json:"end_time"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AITraceSpan) TableName() string { return "ai_trace_spans" }

type AIUsageLog struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RunID            string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	SessionID        string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	UserID           uint64         `gorm:"column:user_id;not null;default:0;index" json:"user_id"`
	Scene            string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Status           string         `gorm:"column:status;type:varchar(32);index" json:"status"`
	PromptTokens     int64          `gorm:"column:prompt_tokens;not null;default:0" json:"prompt_tokens"`
	CompletionTokens int64          `gorm:"column:completion_tokens;not null;default:0" json:"completion_tokens"`
	TotalTokens      int64          `gorm:"column:total_tokens;not null;default:0" json:"total_tokens"`
	EstimatedCostUSD float64        `gorm:"column:estimated_cost_usd;type:decimal(12,6);not null;default:0" json:"estimated_cost_usd"`
	FirstTokenMS     int64          `gorm:"column:first_token_ms;not null;default:0" json:"first_token_ms"`
	TokensPerSecond  float64        `gorm:"column:tokens_per_second;type:decimal(12,4);not null;default:0" json:"tokens_per_second"`
	ApprovalCount    int64          `gorm:"column:approval_count;not null;default:0" json:"approval_count"`
	ApprovalStatus   string         `gorm:"column:approval_status;type:varchar(16);default:''" json:"approval_status"`
	ToolCallCount    int64          `gorm:"column:tool_call_count;not null;default:0" json:"tool_call_count"`
	ToolErrorCount   int64          `gorm:"column:tool_error_count;not null;default:0" json:"tool_error_count"`
	MetadataJSON     string         `gorm:"column:metadata_json;type:text" json:"metadata_json"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIUsageLog) TableName() string { return "ai_usage_logs" }
