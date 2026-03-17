package model

import (
	"time"

	"gorm.io/gorm"
)

// AIChatSession stores a user's chat session container.
type AIChatSession struct {
	ID        string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	UserID    uint64         `gorm:"column:user_id;not null;index:idx_ai_chat_sessions_user_scene_updated,priority:1;index:idx_ai_chat_sessions_user_id" json:"user_id"`
	Scene     string         `gorm:"column:scene;type:varchar(32);not null;default:'ai';index:idx_ai_chat_sessions_user_scene_updated,priority:2" json:"scene"`
	Title     string         `gorm:"column:title;type:varchar(255);not null;default:''" json:"title"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;index:idx_ai_chat_sessions_user_scene_updated,priority:3,sort:desc" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIChatSession) TableName() string { return "ai_chat_sessions" }

// AIChatMessage stores the final persisted messages for a session.
type AIChatMessage struct {
	ID           string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID    string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_chat_messages_session_created,priority:1;uniqueIndex:uk_ai_chat_messages_session_seq,priority:1;index:idx_ai_chat_messages_session_role,priority:1" json:"session_id"`
	SessionIDNum int            `gorm:"column:session_id_num;not null;default:0;uniqueIndex:uk_ai_chat_messages_session_seq,priority:2" json:"session_id_num"`
	Role         string         `gorm:"column:role;type:varchar(16);not null;default:'assistant';index:idx_ai_chat_messages_session_role,priority:2" json:"role"`
	Content      string         `gorm:"column:content;type:longtext;not null" json:"content"`
	Status       string         `gorm:"column:status;type:varchar(16);not null;default:'done'" json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_chat_messages_session_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIChatMessage) TableName() string { return "ai_chat_messages" }

// AIRun stores one model execution bound to a session and a user/assistant message pair.
type AIRun struct {
	ID                 string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	SessionID          string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_runs_session_id" json:"session_id"`
	UserMessageID      string         `gorm:"column:user_message_id;type:varchar(64);not null;index:idx_ai_runs_user_message_id" json:"user_message_id"`
	AssistantMessageID string         `gorm:"column:assistant_message_id;type:varchar(64);index:idx_ai_runs_assistant_message_id" json:"assistant_message_id"`
	Status             string         `gorm:"column:status;type:varchar(16);not null;default:'running';index:idx_ai_runs_status_created,priority:1" json:"status"`
	AssistantType      string         `gorm:"column:assistant_type;type:varchar(64)" json:"assistant_type"`
	IntentType         string         `gorm:"column:intent_type;type:varchar(32)" json:"intent_type"`
	ProgressSummary    string         `gorm:"column:progress_summary;type:text" json:"progress_summary"`
	RiskLevel          string         `gorm:"column:risk_level;type:varchar(16)" json:"risk_level"`
	TraceID            string         `gorm:"column:trace_id;type:varchar(128)" json:"trace_id"`
	ErrorMessage       string         `gorm:"column:error_message;type:text" json:"error_message"`
	TraceJSON          string         `gorm:"column:trace_json;type:longtext;not null" json:"trace_json"`
	StartedAt          time.Time      `gorm:"column:started_at;autoCreateTime" json:"started_at"`
	FinishedAt         *time.Time     `gorm:"column:finished_at" json:"finished_at"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_runs_status_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIRun) TableName() string { return "ai_runs" }

// AIDiagnosisReport stores structured diagnosis output for a run.
type AIDiagnosisReport struct {
	ID                  string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID               string         `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex" json:"run_id"`
	SessionID           string         `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_diagnosis_reports_session_created,priority:1" json:"session_id"`
	Summary             string         `gorm:"column:summary;type:text;not null" json:"summary"`
	ReportJSON          string         `gorm:"column:report_json;type:longtext" json:"report_json"`
	EvidenceJSON        string         `gorm:"column:evidence_json;type:longtext" json:"evidence_json"`
	RootCausesJSON      string         `gorm:"column:root_causes_json;type:longtext" json:"root_causes_json"`
	RecommendationsJSON string         `gorm:"column:recommendations_json;type:longtext" json:"recommendations_json"`
	RiskLevel           string         `gorm:"column:risk_level;type:varchar(16)" json:"risk_level"`
	GeneratedAt         time.Time      `gorm:"column:generated_at;autoCreateTime" json:"generated_at"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_diagnosis_reports_session_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIDiagnosisReport) TableName() string { return "ai_diagnosis_reports" }

// AIScenePrompt stores active prompt fragments bound to a scene.
type AIScenePrompt struct {
	ID           uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Scene        string         `gorm:"column:scene;type:varchar(32);not null;index:idx_ai_scene_prompts_scene_active_order,priority:1" json:"scene"`
	PromptText   string         `gorm:"column:prompt_text;type:text;not null" json:"prompt_text"`
	DisplayOrder int            `gorm:"column:display_order;not null;default:0;index:idx_ai_scene_prompts_scene_active_order,priority:3" json:"display_order"`
	IsActive     bool           `gorm:"column:is_active;not null;default:true;index:idx_ai_scene_prompts_scene_active_order,priority:2" json:"is_active"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIScenePrompt) TableName() string { return "ai_scene_prompts" }

// AISceneConfig stores scene-level routing/tool constraints.
type AISceneConfig struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Scene            string         `gorm:"column:scene;type:varchar(32);not null;uniqueIndex" json:"scene"`
	Description      string         `gorm:"column:description;type:text" json:"description"`
	ConstraintsJSON  string         `gorm:"column:constraints_json;type:longtext" json:"constraints_json"`
	AllowedToolsJSON string         `gorm:"column:allowed_tools_json;type:longtext" json:"allowed_tools_json"`
	BlockedToolsJSON string         `gorm:"column:blocked_tools_json;type:longtext" json:"blocked_tools_json"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AISceneConfig) TableName() string { return "ai_scene_configs" }

// AITraceSpan stores aggregated model trace spans for dashboard analytics.
type AITraceSpan struct {
	ID         string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID      string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	SessionID  string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	Scene      string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Status     string         `gorm:"column:status;type:varchar(16);index" json:"status"`
	ModelName  string         `gorm:"column:model_name;type:varchar(128)" json:"model_name"`
	Tokens     int64          `gorm:"column:tokens;not null;default:0" json:"tokens"`
	DurationMS int64          `gorm:"column:duration_ms;not null;default:0" json:"duration_ms"`
	StartTime  time.Time      `gorm:"column:start_time;not null;index" json:"start_time"`
	EndTime    *time.Time     `gorm:"column:end_time" json:"end_time"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AITraceSpan) TableName() string { return "ai_trace_spans" }

// AIUsageLog stores usage and approval metrics for analytics queries.
type AIUsageLog struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RunID            string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	SessionID        string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	UserID           uint64         `gorm:"column:user_id;not null;default:0;index" json:"user_id"`
	Scene            string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Status           string         `gorm:"column:status;type:varchar(16);index" json:"status"`
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
	MetadataJSON     string         `gorm:"column:metadata_json;type:longtext" json:"metadata_json"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIUsageLog) TableName() string { return "ai_usage_logs" }

// AICheckpoint stores serialized ADK checkpoint blobs for resume flows.
type AICheckpoint struct {
	CheckpointID string         `gorm:"column:checkpoint_id;type:varchar(64);primaryKey" json:"checkpoint_id"`
	SessionID    string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	RunID        string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	UserID       uint64         `gorm:"column:user_id;not null;default:0;index" json:"user_id"`
	Scene        string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Payload      []byte         `gorm:"column:payload;type:longblob;not null" json:"payload"`
	ExpiresAt    *time.Time     `gorm:"column:expires_at;index" json:"expires_at"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AICheckpoint) TableName() string { return "ai_checkpoints" }

// AIApprovalTask 存储工具审批任务，用于 Human-in-the-Loop 工作流。
//
// 当高风险工具需要人工审批时，系统会创建审批任务记录，
// 用户确认或拒绝后，状态更新并通过 checkpoint 恢复执行。
type AIApprovalTask struct {
	ID               uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ApprovalID       string         `gorm:"column:approval_id;type:varchar(64);not null;uniqueIndex" json:"approval_id"`
	CheckpointID     string         `gorm:"column:checkpoint_id;type:varchar(64);not null;index" json:"checkpoint_id"`
	SessionID        string         `gorm:"column:session_id;type:varchar(64);not null;index" json:"session_id"`
	RunID            string         `gorm:"column:run_id;type:varchar(64);not null;index" json:"run_id"`
	UserID           uint64         `gorm:"column:user_id;not null;default:0;index" json:"user_id"`
	ToolName         string         `gorm:"column:tool_name;type:varchar(64);not null" json:"tool_name"`
	ToolCallID       string         `gorm:"column:tool_call_id;type:varchar(64);not null" json:"tool_call_id"`
	ArgumentsJSON    string         `gorm:"column:arguments_json;type:longtext;not null" json:"arguments_json"`
	PreviewJSON      string         `gorm:"column:preview_json;type:longtext;not null" json:"preview_json"`
	Status           string         `gorm:"column:status;type:varchar(16);not null;default:'pending';index" json:"status"` // pending, approved, rejected, expired
	ApprovedBy       uint64         `gorm:"column:approved_by;not null;default:0" json:"approved_by"`
	DisapproveReason string         `gorm:"column:disapprove_reason;type:text" json:"disapprove_reason"`
	Comment          string         `gorm:"column:comment;type:text" json:"comment"`
	TimeoutSeconds   int            `gorm:"column:timeout_seconds;not null;default:300" json:"timeout_seconds"`
	ExpiresAt        *time.Time     `gorm:"column:expires_at;index" json:"expires_at"`
	DecidedAt        *time.Time     `gorm:"column:decided_at" json:"decided_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (AIApprovalTask) TableName() string { return "ai_approval_tasks" }
