package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AIToolRiskPolicy struct {
	ID                uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ToolName          string    `gorm:"column:tool_name;type:varchar(64);not null;index:idx_ai_tool_risk_policies_tool_enabled,priority:1" json:"tool_name"`
	Scene             *string   `gorm:"column:scene;type:varchar(32)" json:"scene"`
	CommandClass      *string   `gorm:"column:command_class;type:varchar(32)" json:"command_class"`
	ArgumentRulesJSON *string   `gorm:"column:argument_rules;type:text" json:"argument_rules"`
	ApprovalRequired  bool      `gorm:"column:approval_required;not null;default:false" json:"approval_required"`
	RiskLevel         string    `gorm:"column:risk_level;type:varchar(16);not null;default:'medium'" json:"risk_level"`
	Priority          int       `gorm:"column:priority;not null;default:0" json:"priority"`
	Enabled           bool      `gorm:"column:enabled;not null;default:true;index:idx_ai_tool_risk_policies_tool_enabled,priority:2" json:"enabled"`
	PolicyVersion     string    `gorm:"column:policy_version;type:varchar(64);not null;default:''" json:"policy_version"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIToolRiskPolicy) TableName() string { return "ai_tool_risk_policies" }

type AIApprovalTask struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ApprovalID       string         `gorm:"column:approval_id;type:varchar(64);not null;uniqueIndex" json:"approval_id"`
	CheckpointID     string         `gorm:"column:checkpoint_id;type:varchar(64);not null;index" json:"checkpoint_id"`
	SessionID        string         `gorm:"column:session_id;type:varchar(64);not null;index" json:"session_id"`
	RunID            string         `gorm:"column:run_id;type:varchar(64);not null;index" json:"run_id"`
	UserID           uint64         `gorm:"column:user_id;not null;default:0;index" json:"user_id"`
	ToolName         string         `gorm:"column:tool_name;type:varchar(64);not null" json:"tool_name"`
	ToolCallID       string         `gorm:"column:tool_call_id;type:varchar(64);not null" json:"tool_call_id"`
	ResumeTargetID   string         `gorm:"column:resume_target_id;type:varchar(128);not null;default:''" json:"resume_target_id"`
	ArgumentsJSON    string         `gorm:"column:arguments_json;type:text;not null" json:"arguments_json"`
	PreviewJSON      string         `gorm:"column:preview_json;type:text;not null" json:"preview_json"`
	Status           string         `gorm:"column:status;type:varchar(16);not null;default:'pending';index" json:"status"`
	ApprovedBy       uint64         `gorm:"column:approved_by;not null;default:0" json:"approved_by"`
	DisapproveReason string         `gorm:"column:disapprove_reason;type:text" json:"disapprove_reason"`
	Comment          string         `gorm:"column:comment;type:text" json:"comment"`
	TimeoutSeconds   int            `gorm:"column:timeout_seconds;not null;default:300" json:"timeout_seconds"`
	ExpiresAt        *time.Time     `gorm:"column:expires_at;index" json:"expires_at"`
	LockExpiresAt    *time.Time     `gorm:"column:lock_expires_at;index" json:"lock_expires_at"`
	MatchedRuleID    *uint64        `gorm:"column:matched_rule_id;index" json:"matched_rule_id"`
	PolicyVersion    *string        `gorm:"column:policy_version;type:varchar(64)" json:"policy_version"`
	DecisionSource   *string        `gorm:"column:decision_source;type:varchar(32)" json:"decision_source"`
	DecidedAt        *time.Time     `gorm:"column:decided_at" json:"decided_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIApprovalTask) TableName() string { return "ai_approval_tasks" }

type AIApprovalOutboxEvent struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EventID     string     `gorm:"column:event_id;type:varchar(64);not null;uniqueIndex:uk_ai_approval_outbox_events_event_id" json:"event_id"`
	Sequence    int64      `gorm:"column:sequence;not null;uniqueIndex:uk_ai_approval_outbox_events_run_seq,priority:2;index:idx_ai_approval_outbox_events_aggregate_sequence,priority:2" json:"sequence"`
	AggregateID string     `gorm:"column:aggregate_id;type:varchar(64);not null;index:idx_ai_approval_outbox_events_aggregate_sequence,priority:1" json:"aggregate_id"`
	OccurredAt  time.Time  `gorm:"column:occurred_at;not null;index:idx_ai_approval_outbox_events_aggregate_sequence,priority:3" json:"occurred_at"`
	Version     int        `gorm:"column:version;not null;default:1" json:"version"`
	ApprovalID  string     `gorm:"column:approval_id;type:varchar(64);not null;uniqueIndex:uk_ai_approval_outbox_events_approval_event,priority:1" json:"approval_id"`
	ToolCallID  string     `gorm:"column:tool_call_id;type:varchar(64);not null;default:'';index:idx_ai_approval_outbox_events_tool_call_id" json:"tool_call_id"`
	EventType   string     `gorm:"column:event_type;type:varchar(64);not null;uniqueIndex:uk_ai_approval_outbox_events_approval_event,priority:2" json:"event_type"`
	RunID       string     `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex:uk_ai_approval_outbox_events_run_seq,priority:1;index:idx_ai_approval_outbox_events_run_id" json:"run_id"`
	SessionID   string     `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_approval_outbox_events_session_id" json:"session_id"`
	PayloadJSON string     `gorm:"column:payload_json;type:text;not null" json:"payload_json"`
	Status      string     `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_ai_approval_outbox_events_queue,priority:1" json:"status"`
	RetryCount  int        `gorm:"column:retry_count;not null;default:0" json:"retry_count"`
	NextRetryAt *time.Time `gorm:"column:next_retry_at;index:idx_ai_approval_outbox_events_queue,priority:2" json:"next_retry_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_approval_outbox_events_queue,priority:3" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (AIApprovalOutboxEvent) TableName() string { return "ai_approval_outbox_events" }

func (e *AIApprovalOutboxEvent) BeforeCreate(tx *gorm.DB) error {
	if e == nil {
		return nil
	}
	if e.EventID == "" {
		e.EventID = uuid.NewString()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	} else {
		e.OccurredAt = e.OccurredAt.UTC()
	}
	if e.Version <= 0 {
		e.Version = 1
	}
	if e.AggregateID == "" {
		e.AggregateID = e.RunID
	}
	if e.Sequence <= 0 && tx != nil && e.RunID != "" {
		var sequence int64
		if err := tx.Raw("SELECT COALESCE(MAX(sequence), 0) + 1 FROM ai_approval_outbox_events WHERE run_id = ?", e.RunID).Scan(&sequence).Error; err != nil {
			return err
		}
		if sequence > 0 {
			e.Sequence = sequence
		}
	}
	return nil
}
