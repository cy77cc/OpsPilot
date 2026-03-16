package model

import "time"

// AIRun models the Phase 1 run lifecycle that links a chat session to a QA or diagnosis execution.
type AIRun struct {
	ID                 string     `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID          string     `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_runs_session_status,priority:1" json:"session_id"`
	UserMessageID      string     `gorm:"column:user_message_id;type:varchar(64);not null;index:idx_ai_runs_user_message_id" json:"user_message_id"`
	AssistantMessageID *string    `gorm:"column:assistant_message_id;type:varchar(64);index:idx_ai_runs_assistant_message_id" json:"assistant_message_id,omitempty"`
	IntentType         string     `gorm:"column:intent_type;type:varchar(32);not null;default:''" json:"intent_type"`
	AssistantType      string     `gorm:"column:assistant_type;type:varchar(32);not null;default:''" json:"assistant_type"`
	RiskLevel          string     `gorm:"column:risk_level;type:varchar(16);not null;default:''" json:"risk_level"`
	Status             string     `gorm:"column:status;type:varchar(32);not null;default:'';index:idx_ai_runs_session_status,priority:2;index:idx_ai_runs_status" json:"status"`
	TraceID            string     `gorm:"column:trace_id;type:varchar(64);not null;default:'';index:idx_ai_runs_trace_id" json:"trace_id"`
	ErrorMessage       string     `gorm:"column:error_message;type:text;not null" json:"error_message"`
	StartedAt          *time.Time `gorm:"column:started_at;type:datetime(6)" json:"started_at,omitempty"`
	FinishedAt         *time.Time `gorm:"column:finished_at;type:datetime(6)" json:"finished_at,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name for AIRun.
func (AIRun) TableName() string {
	return "ai_runs"
}
