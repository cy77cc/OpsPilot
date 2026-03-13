package model

import "time"

// AIExecution stores async tool execution state for the redesigned AI module.
type AIExecution struct {
	ID               string     `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID        string     `gorm:"column:session_id;type:varchar(64);index:idx_ai_executions_session_plan_step" json:"session_id"`
	PlanID           string     `gorm:"column:plan_id;type:varchar(64);index:idx_ai_executions_session_plan_step" json:"plan_id"`
	StepID           string     `gorm:"column:step_id;type:varchar(64);index:idx_ai_executions_session_plan_step" json:"step_id"`
	CheckpointID     string     `gorm:"column:checkpoint_id;type:varchar(128);index" json:"checkpoint_id"`
	ApprovalID       string     `gorm:"column:approval_id;type:varchar(64);index" json:"approval_id"`
	RequestUserID    uint64     `gorm:"column:request_user_id;index:idx_ai_executions_user_created" json:"request_user_id"`
	ToolName         string     `gorm:"column:tool_name;type:varchar(128);index" json:"tool_name"`
	ToolMode         string     `gorm:"column:tool_mode;type:varchar(32)" json:"tool_mode"`
	RiskLevel        string     `gorm:"column:risk_level;type:varchar(16)" json:"risk_level"`
	Scene            string     `gorm:"column:scene;type:varchar(128);index" json:"scene"`
	Status           string     `gorm:"column:status;type:varchar(32);index" json:"status"`
	ParamsJSON       string     `gorm:"column:params_json;type:longtext" json:"params_json"`
	ResultJSON       string     `gorm:"column:result_json;type:longtext" json:"result_json"`
	MetadataJSON     string     `gorm:"column:metadata_json;type:longtext" json:"metadata_json"`
	ErrorMessage     string     `gorm:"column:error_message;type:text" json:"error_message"`
	DurationMs       int64      `gorm:"column:duration_ms;default:0" json:"duration_ms"`
	PromptTokens     int64      `gorm:"column:prompt_tokens;default:0" json:"prompt_tokens"`
	CompletionTokens int64      `gorm:"column:completion_tokens;default:0" json:"completion_tokens"`
	TotalTokens      int64      `gorm:"column:total_tokens;default:0" json:"total_tokens"`
	EstimatedCostUSD float64    `gorm:"column:estimated_cost_usd;type:decimal(20,8);default:0" json:"estimated_cost_usd"`
	StartedAt        *time.Time `gorm:"column:started_at" json:"started_at,omitempty"`
	FinishedAt       *time.Time `gorm:"column:finished_at" json:"finished_at,omitempty"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_executions_user_created" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIExecution) TableName() string { return "ai_executions" }
