package model

import "time"

// AIApproval stores runtime-facing approval records for the redesigned AI module.
type AIApproval struct {
	ID              string     `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID       string     `gorm:"column:session_id;type:varchar(64);index:idx_ai_approvals_session_plan_step" json:"session_id"`
	PlanID          string     `gorm:"column:plan_id;type:varchar(64);index:idx_ai_approvals_session_plan_step" json:"plan_id"`
	StepID          string     `gorm:"column:step_id;type:varchar(64);index:idx_ai_approvals_session_plan_step" json:"step_id"`
	CheckpointID    string     `gorm:"column:checkpoint_id;type:varchar(128);index" json:"checkpoint_id"`
	ApprovalKey     string     `gorm:"column:approval_key;type:varchar(128);uniqueIndex" json:"approval_key"`
	RequestUserID   uint64     `gorm:"column:request_user_id;index:idx_ai_approvals_user_created" json:"request_user_id"`
	ReviewerUserID  uint64     `gorm:"column:reviewer_user_id;index" json:"reviewer_user_id"`
	ToolName        string     `gorm:"column:tool_name;type:varchar(128);index" json:"tool_name"`
	ToolDisplayName string     `gorm:"column:tool_display_name;type:varchar(128)" json:"tool_display_name"`
	ToolMode        string     `gorm:"column:tool_mode;type:varchar(32)" json:"tool_mode"`
	RiskLevel       string     `gorm:"column:risk_level;type:varchar(16);index" json:"risk_level"`
	Status          string     `gorm:"column:status;type:varchar(32);index" json:"status"`
	Scene           string     `gorm:"column:scene;type:varchar(128);index" json:"scene"`
	Summary         string     `gorm:"column:summary;type:text" json:"summary"`
	Reason          string     `gorm:"column:reason;type:varchar(255)" json:"reason"`
	ParamsJSON      string     `gorm:"column:params_json;type:longtext" json:"params_json"`
	PreviewJSON     string     `gorm:"column:preview_json;type:longtext" json:"preview_json"`
	ExecutionID     string     `gorm:"column:execution_id;type:varchar(64);index" json:"execution_id"`
	ApprovedAt      *time.Time `gorm:"column:approved_at" json:"approved_at,omitempty"`
	RejectedAt      *time.Time `gorm:"column:rejected_at" json:"rejected_at,omitempty"`
	ExpiresAt       *time.Time `gorm:"column:expires_at" json:"expires_at,omitempty"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_approvals_user_created" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIApproval) TableName() string { return "ai_approvals" }
