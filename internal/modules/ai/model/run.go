package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	RunStatusRunning               = "running"
	RunStatusDelegating            = "delegating"
	RunStatusWaitingSubagent       = "waiting_subagent"
	RunStatusWaitingApproval       = "waiting_approval"
	RunStatusResuming              = "resuming"
	RunStatusResumeFailedRetryable = "resume_failed_retryable"
)

type AIRun struct {
	ID                 string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID          string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_runs_session_id;uniqueIndex:uk_ai_runs_session_request,priority:1" json:"session_id"`
	ClientRequestID    string         `gorm:"column:client_request_id;type:varchar(64);not null;default:'';uniqueIndex:uk_ai_runs_session_request,priority:2" json:"client_request_id"`
	UserMessageID      string         `gorm:"column:user_message_id;type:varchar(64);not null;index:idx_ai_runs_user_message_id" json:"user_message_id"`
	AssistantMessageID string         `gorm:"column:assistant_message_id;type:varchar(64);index:idx_ai_runs_assistant_message_id" json:"assistant_message_id"`
	Status             string         `gorm:"column:status;type:varchar(32);not null;default:'running';index:idx_ai_runs_status_created,priority:1" json:"status"`
	AssistantType      string         `gorm:"column:assistant_type;type:varchar(64)" json:"assistant_type"`
	IntentType         string         `gorm:"column:intent_type;type:varchar(32)" json:"intent_type"`
	ProgressSummary    string         `gorm:"column:progress_summary;type:text" json:"progress_summary"`
	RiskLevel          string         `gorm:"column:risk_level;type:varchar(16)" json:"risk_level"`
	TraceID            string         `gorm:"column:trace_id;type:varchar(128)" json:"trace_id"`
	ErrorMessage       string         `gorm:"column:error_message;type:text" json:"error_message"`
	TraceJSON          string         `gorm:"column:trace_json;type:text;not null" json:"trace_json"`
	StartedAt          time.Time      `gorm:"column:started_at;autoCreateTime" json:"started_at"`
	LastEventAt        *time.Time     `gorm:"column:last_event_at" json:"last_event_at"`
	FinishedAt         *time.Time     `gorm:"column:finished_at" json:"finished_at"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_runs_status_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIRun) TableName() string { return "ai_runs" }

type AIRunEvent struct {
	ID          string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID       string    `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex:uk_ai_run_events_run_seq,priority:1;index:idx_ai_run_events_run_type,priority:1" json:"run_id"`
	SessionID   string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_events_session_created,priority:1" json:"session_id"`
	Seq         int       `gorm:"column:seq;not null;uniqueIndex:uk_ai_run_events_run_seq,priority:2" json:"seq"`
	EventType   string    `gorm:"column:event_type;type:varchar(32);not null;index:idx_ai_run_events_run_type,priority:2" json:"event_type"`
	AgentName   string    `gorm:"column:agent_name;type:varchar(64)" json:"agent_name"`
	ToolCallID  string    `gorm:"column:tool_call_id;type:varchar(64);index:idx_ai_run_events_tool_call_id" json:"tool_call_id"`
	PayloadJSON string    `gorm:"column:payload_json;type:text;not null" json:"payload_json"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;index:idx_ai_run_events_session_created,priority:2,sort:desc" json:"created_at"`
}

func (AIRunEvent) TableName() string { return "ai_run_events" }

type AIRunProjection struct {
	ID             string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID          string    `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex:uk_ai_run_projections_run_id" json:"run_id"`
	SessionID      string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_projections_session_id" json:"session_id"`
	Version        int       `gorm:"column:version;not null;default:1" json:"version"`
	Status         string    `gorm:"column:status;type:varchar(32);not null" json:"status"`
	ProjectionJSON string    `gorm:"column:projection_json;type:text;not null" json:"projection_json"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIRunProjection) TableName() string { return "ai_run_projections" }

type AIRunContent struct {
	ID          string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID       string    `gorm:"column:run_id;type:varchar(64);not null;index:idx_ai_run_contents_run_id" json:"run_id"`
	SessionID   string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_contents_session_id" json:"session_id"`
	ContentKind string    `gorm:"column:content_kind;type:varchar(32);not null;index:idx_ai_run_contents_kind" json:"content_kind"`
	Encoding    string    `gorm:"column:encoding;type:varchar(16);not null" json:"encoding"`
	SummaryText string    `gorm:"column:summary_text;type:varchar(500)" json:"summary_text"`
	BodyText    string    `gorm:"column:body_text;type:text" json:"body_text"`
	BodyJSON    string    `gorm:"column:body_json;type:text" json:"body_json"`
	SizeBytes   int64     `gorm:"column:size_bytes;not null;default:0" json:"size_bytes"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (AIRunContent) TableName() string { return "ai_run_contents" }

// IsOpenRunStatus reports whether a run is still open for tailing/reconnect.
func IsOpenRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case RunStatusWaitingApproval,
		RunStatusResuming,
		RunStatusRunning,
		RunStatusResumeFailedRetryable,
		RunStatusDelegating,
		RunStatusWaitingSubagent:
		return true
	default:
		return false
	}
}
