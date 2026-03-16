package model

import "time"

// AIRun stores the lifecycle of a single assistant execution for a user message.
type AIRun struct {
	ID                 string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID          string    `gorm:"column:session_id;type:varchar(64);index:idx_ai_runs_session_id" json:"session_id"`
	UserMessageID      string    `gorm:"column:user_message_id;type:varchar(64)" json:"user_message_id"`
	AssistantMessageID string    `gorm:"column:assistant_message_id;type:varchar(64)" json:"assistant_message_id"`
	IntentType         string    `gorm:"column:intent_type;type:varchar(32)" json:"intent_type"`
	AssistantType      string    `gorm:"column:assistant_type;type:varchar(32)" json:"assistant_type"`
	RiskLevel          string    `gorm:"column:risk_level;type:varchar(32)" json:"risk_level"`
	Status             string    `gorm:"column:status;type:varchar(32);index:idx_ai_runs_status" json:"status"`
	TraceID            string    `gorm:"column:trace_id;type:varchar(64)" json:"trace_id"`
	ErrorMessage       string    `gorm:"column:error_message;type:text" json:"error_message"`
	ProgressSummary    string    `gorm:"column:progress_summary;type:text" json:"progress_summary"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIRun) TableName() string { return "ai_runs" }
