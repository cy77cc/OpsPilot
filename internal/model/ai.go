// Package model 定义数据库模型。
//
// 本文件实现 AI 模块相关的数据库模型，包括：
//   - 会话管理：AIChatSession, AIChatMessage
//   - 运行记录：AIRun, AIRunEvent, AIRunProjection, AIRunContent
//   - 诊断报告：AIDiagnosisReport
//   - 场景配置：AIScenePrompt, AISceneConfig
//   - 追踪统计：AITraceSpan, AIUsageLog
//   - 检查点：AICheckpoint
//   - 审批流程：AIToolRiskPolicy, AIApprovalTask, AIApprovalOutboxEvent
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AIChatSession 存储用户的聊天会话容器。
//
// 每个会话属于一个用户，关联多个消息（AIChatMessage）。
// 会话支持场景分类（如 ai、diagnosis、change 等），便于按场景管理对话。
//
// 字段说明:
//   - ID: 会话唯一标识（UUID 格式）
//   - UserID: 所属用户 ID
//   - Scene: 会话场景（默认 "ai"）
//   - Title: 会话标题（自动生成或用户设置）
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间（每次消息更新）
//   - DeletedAt: 软删除时间
type AIChatSession struct {
	ID        string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	UserID    uint64         `gorm:"column:user_id;not null;index:idx_ai_chat_sessions_user_scene_updated,priority:1;index:idx_ai_chat_sessions_user_id" json:"user_id"`
	Scene     string         `gorm:"column:scene;type:varchar(32);not null;default:'ai';index:idx_ai_chat_sessions_user_scene_updated,priority:2" json:"scene"`
	Title     string         `gorm:"column:title;type:varchar(255);not null;default:''" json:"title"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;index:idx_ai_chat_sessions_user_scene_updated,priority:3,sort:desc" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 AIChatSession 的数据库表名。
func (AIChatSession) TableName() string { return "ai_chat_sessions" }

// AIChatMessage 存储会话中的消息记录。
//
// 每条消息属于一个会话，支持用户消息和助手消息两种角色。
// 消息按 session_id_num 排序，确保对话顺序正确。
//
// 字段说明:
//   - ID: 消息唯一标识（UUID 格式）
//   - SessionID: 所属会话 ID
//   - SessionIDNum: 会话内序号（从 1 开始递增）
//   - Role: 消息角色（"user" 或 "assistant"）
//   - Content: 消息内容
//   - Status: 消息状态（"done", "streaming", "error" 等）
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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

// TableName 返回 AIChatMessage 的数据库表名。
func (AIChatMessage) TableName() string { return "ai_chat_messages" }

// AIRun 存储模型执行记录。
//
// 每次 AI 对话都会创建一个 Run 记录，关联用户消息和助手消息。
// Run 记录执行状态、意图类型、风险级别、追踪信息等元数据。
//
// 状态值说明:
//   - running: 执行中
//   - completed: 成功完成
//   - completed_with_tool_errors: 完成但有工具错误
//   - failed_runtime: 运行时错误
//   - cancelled: 用户取消
//
// 字段说明:
//   - ID: Run 唯一标识（UUID 格式）
//   - SessionID: 所属会话 ID
//   - ClientRequestID: 客户端请求 ID（用于幂等性）
//   - UserMessageID: 用户消息 ID
//   - AssistantMessageID: 助手消息 ID
//   - Status: 执行状态
//   - AssistantType: 助手类型（qa, diagnosis, change 等）
//   - IntentType: 意图类型
//   - ProgressSummary: 进度摘要
//   - RiskLevel: 风险级别（low, medium, high）
//   - TraceID: 分布式追踪 ID
//   - ErrorMessage: 错误信息
//   - TraceJSON: 完整追踪数据（JSON 格式）
//   - StartedAt: 开始时间
//   - LastEventAt: 最后事件时间
//   - FinishedAt: 完成时间
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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
	TraceJSON          string         `gorm:"column:trace_json;type:longtext;not null" json:"trace_json"`
	StartedAt          time.Time      `gorm:"column:started_at;autoCreateTime" json:"started_at"`
	LastEventAt        *time.Time     `gorm:"column:last_event_at" json:"last_event_at"`
	FinishedAt         *time.Time     `gorm:"column:finished_at" json:"finished_at"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_ai_runs_status_created,priority:2,sort:desc" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 AIRun 的数据库表名。
func (AIRun) TableName() string { return "ai_runs" }

// AIRunEvent 存储 Run 的原始事件流。
//
// 事件按顺序存储，支持事件重放和状态恢复。
// 事件类型包括：delta, tool_call, tool_result, approval_required 等。
//
// 字段说明:
//   - ID: 事件唯一标识
//   - RunID: 所属 Run ID
//   - SessionID: 所属会话 ID（冗余，便于查询）
//   - Seq: 事件序号（从 1 开始递增）
//   - EventType: 事件类型
//   - AgentName: 代理名称
//   - ToolCallID: 工具调用 ID（工具事件时有效）
//   - PayloadJSON: 事件负载数据（JSON 格式）
//   - CreatedAt: 创建时间
type AIRunEvent struct {
	ID          string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID       string    `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex:uk_ai_run_events_run_seq,priority:1;index:idx_ai_run_events_run_type,priority:1" json:"run_id"`
	SessionID   string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_events_session_created,priority:1" json:"session_id"`
	Seq         int       `gorm:"column:seq;not null;uniqueIndex:uk_ai_run_events_run_seq,priority:2" json:"seq"`
	EventType   string    `gorm:"column:event_type;type:varchar(32);not null;index:idx_ai_run_events_run_type,priority:2" json:"event_type"`
	AgentName   string    `gorm:"column:agent_name;type:varchar(64)" json:"agent_name"`
	ToolCallID  string    `gorm:"column:tool_call_id;type:varchar(64);index:idx_ai_run_events_tool_call_id" json:"tool_call_id"`
	PayloadJSON string    `gorm:"column:payload_json;type:longtext;not null" json:"payload_json"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;index:idx_ai_run_events_session_created,priority:2,sort:desc" json:"created_at"`
}

// TableName 返回 AIRunEvent 的数据库表名。
func (AIRunEvent) TableName() string { return "ai_run_events" }

// AIRunProjection 存储 Run 的投影数据。
//
// 投影是从原始事件流中提取的结构化数据，便于前端展示和查询。
// 支持版本控制和增量更新。
//
// 字段说明:
//   - ID: 投影唯一标识
//   - RunID: 所属 Run ID（唯一关联）
//   - SessionID: 所属会话 ID
//   - Version: 投影版本号
//   - Status: 投影状态
//   - ProjectionJSON: 投影数据（JSON 格式）
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
type AIRunProjection struct {
	ID             string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID          string    `gorm:"column:run_id;type:varchar(64);not null;uniqueIndex:uk_ai_run_projections_run_id" json:"run_id"`
	SessionID      string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_projections_session_id" json:"session_id"`
	Version        int       `gorm:"column:version;not null;default:1" json:"version"`
	Status         string    `gorm:"column:status;type:varchar(32);not null" json:"status"`
	ProjectionJSON string    `gorm:"column:projection_json;type:longtext;not null" json:"projection_json"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 返回 AIRunProjection 的数据库表名。
func (AIRunProjection) TableName() string { return "ai_run_projections" }

// AIRunContent 存储 Run 的可延迟加载内容体。
//
// 用于存储大型内容（如工具输出、日志等），避免在投影中直接存储大文本。
// 支持按需加载，减少内存占用。
//
// 字段说明:
//   - ID: 内容唯一标识
//   - RunID: 所属 Run ID
//   - SessionID: 所属会话 ID
//   - ContentKind: 内容类型（如 tool_output, log 等）
//   - Encoding: 编码格式（如 plain, base64）
//   - SummaryText: 内容摘要
//   - BodyText: 文本内容
//   - BodyJSON: JSON 内容
//   - SizeBytes: 内容大小（字节）
//   - CreatedAt: 创建时间
type AIRunContent struct {
	ID          string    `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID       string    `gorm:"column:run_id;type:varchar(64);not null;index:idx_ai_run_contents_run_id" json:"run_id"`
	SessionID   string    `gorm:"column:session_id;type:varchar(64);not null;index:idx_ai_run_contents_session_id" json:"session_id"`
	ContentKind string    `gorm:"column:content_kind;type:varchar(32);not null;index:idx_ai_run_contents_kind" json:"content_kind"`
	Encoding    string    `gorm:"column:encoding;type:varchar(16);not null" json:"encoding"`
	SummaryText string    `gorm:"column:summary_text;type:varchar(500)" json:"summary_text"`
	BodyText    string    `gorm:"column:body_text;type:longtext" json:"body_text"`
	BodyJSON    string    `gorm:"column:body_json;type:longtext" json:"body_json"`
	SizeBytes   int64     `gorm:"column:size_bytes;not null;default:0" json:"size_bytes"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

// TableName 返回 AIRunContent 的数据库表名。
func (AIRunContent) TableName() string { return "ai_run_contents" }

// AIDiagnosisReport 存储结构化诊断报告。
//
// 诊断报告由 Diagnosis Agent 生成，包含问题摘要、证据、根因分析和建议。
// 每个诊断报告关联一个 Run。
//
// 字段说明:
//   - ID: 报告唯一标识
//   - RunID: 关联的 Run ID
//   - SessionID: 所属会话 ID
//   - Summary: 问题摘要
//   - ReportJSON: 完整报告数据（JSON 格式）
//   - EvidenceJSON: 证据列表（JSON 格式）
//   - RootCausesJSON: 根因列表（JSON 格式）
//   - RecommendationsJSON: 建议列表（JSON 格式）
//   - RiskLevel: 风险级别
//   - GeneratedAt: 生成时间
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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

// TableName 返回 AIDiagnosisReport 的数据库表名。
func (AIDiagnosisReport) TableName() string { return "ai_diagnosis_reports" }

// AIScenePrompt 存储场景绑定的提示词片段。
//
// 每个场景可以配置多个提示词片段，按 display_order 排序后拼接。
// 支持启用/禁用控制。
//
// 字段说明:
//   - ID: 提示词唯一标识
//   - Scene: 场景名称
//   - PromptText: 提示词文本
//   - DisplayOrder: 显示顺序
//   - IsActive: 是否启用
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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

// TableName 返回 AIScenePrompt 的数据库表名。
func (AIScenePrompt) TableName() string { return "ai_scene_prompts" }

// AISceneConfig 存储场景级别的路由和工具约束配置。
//
// 用于控制不同场景下的 Agent 行为，如允许/禁止的工具列表。
//
// 字段说明:
//   - ID: 配置唯一标识
//   - Scene: 场景名称（唯一）
//   - Description: 场景描述
//   - ConstraintsJSON: 约束配置（JSON 格式）
//   - AllowedToolsJSON: 允许的工具列表（JSON 格式）
//   - BlockedToolsJSON: 禁止的工具列表（JSON 格式）
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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

// TableName 返回 AISceneConfig 的数据库表名。
func (AISceneConfig) TableName() string { return "ai_scene_configs" }

// AITraceSpan 存储聚合的模型追踪数据，用于仪表板分析。
//
// 记录每次模型调用的元数据，包括 Token 使用量、延迟等。
//
// 字段说明:
//   - ID: 追踪唯一标识
//   - RunID: 关联的 Run ID
//   - SessionID: 所属会话 ID
//   - Scene: 场景名称
//   - Status: 执行状态
//   - ModelName: 模型名称
//   - Tokens: Token 使用量
//   - DurationMS: 执行耗时（毫秒）
//   - StartTime: 开始时间
//   - EndTime: 结束时间
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
type AITraceSpan struct {
	ID         string         `gorm:"column:id;type:varchar(64);primaryKey" json:"id"`
	RunID      string         `gorm:"column:run_id;type:varchar(64);index" json:"run_id"`
	SessionID  string         `gorm:"column:session_id;type:varchar(64);index" json:"session_id"`
	Scene      string         `gorm:"column:scene;type:varchar(32);index" json:"scene"`
	Status     string         `gorm:"column:status;type:varchar(32);index" json:"status"`
	ModelName  string         `gorm:"column:model_name;type:varchar(128)" json:"model_name"`
	Tokens     int64          `gorm:"column:tokens;not null;default:0" json:"tokens"`
	DurationMS int64          `gorm:"column:duration_ms;not null;default:0" json:"duration_ms"`
	StartTime  time.Time      `gorm:"column:start_time;not null;index" json:"start_time"`
	EndTime    *time.Time     `gorm:"column:end_time" json:"end_time"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 AITraceSpan 的数据库表名。
func (AITraceSpan) TableName() string { return "ai_trace_spans" }

// AIUsageLog 存储使用量和审批指标，用于分析查询。
//
// 记录每次对话的 Token 消耗、成本估算、审批统计等指标。
//
// 字段说明:
//   - ID: 日志唯一标识
//   - RunID: 关联的 Run ID
//   - SessionID: 所属会话 ID
//   - UserID: 用户 ID
//   - Scene: 场景名称
//   - Status: 执行状态
//   - PromptTokens: 输入 Token 数
//   - CompletionTokens: 输出 Token 数
//   - TotalTokens: 总 Token 数
//   - EstimatedCostUSD: 估算成本（美元）
//   - FirstTokenMS: 首 Token 延迟（毫秒）
//   - TokensPerSecond: Token 每秒生成速率
//   - ApprovalCount: 审批次数
//   - ApprovalStatus: 审批状态
//   - ToolCallCount: 工具调用次数
//   - ToolErrorCount: 工具错误次数
//   - MetadataJSON: 元数据（JSON 格式）
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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
	MetadataJSON     string         `gorm:"column:metadata_json;type:longtext" json:"metadata_json"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 AIUsageLog 的数据库表名。
func (AIUsageLog) TableName() string { return "ai_usage_logs" }

// AICheckpoint 存储序列化的 ADK 检查点数据，用于恢复流程。
//
// 检查点用于 Human-in-the-Loop 工作流，当需要人工审批时暂停执行，
// 审批完成后通过检查点恢复执行状态。
//
// 字段说明:
//   - CheckpointID: 检查点唯一标识（主键）
//   - SessionID: 所属会话 ID
//   - RunID: 关联的 Run ID
//   - UserID: 用户 ID
//   - Scene: 场景名称
//   - Payload: 序列化的检查点数据（二进制）
//   - ExpiresAt: 过期时间
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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

// TableName 返回 AICheckpoint 的数据库表名。
func (AICheckpoint) TableName() string { return "ai_checkpoints" }

// AIToolRiskPolicy 存储工具的审批策略配置。
//
// 定义哪些工具需要人工审批、风险级别等策略规则。
// 支持按场景、命令类、参数规则进行细粒度控制。
//
// 字段说明:
//   - ID: 策略唯一标识
//   - ToolName: 工具名称
//   - Scene: 场景过滤器（可选）
//   - CommandClass: 命令类过滤器（可选）
//   - ArgumentRulesJSON: 参数规则（JSON 格式）
//   - ApprovalRequired: 是否需要审批
//   - RiskLevel: 风险级别（low, medium, high, critical）
//   - Priority: 优先级（数值越大优先级越高）
//   - Enabled: 是否启用
//   - PolicyVersion: 策略版本
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
type AIToolRiskPolicy struct {
	ID                uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ToolName          string    `gorm:"column:tool_name;type:varchar(64);not null;index:idx_ai_tool_risk_policies_tool_enabled,priority:1" json:"tool_name"`
	Scene             *string   `gorm:"column:scene;type:varchar(32)" json:"scene"`
	CommandClass      *string   `gorm:"column:command_class;type:varchar(32)" json:"command_class"`
	ArgumentRulesJSON *string   `gorm:"column:argument_rules;type:longtext" json:"argument_rules"`
	ApprovalRequired  bool      `gorm:"column:approval_required;not null;default:false" json:"approval_required"`
	RiskLevel         string    `gorm:"column:risk_level;type:varchar(16);not null;default:'medium'" json:"risk_level"`
	Priority          int       `gorm:"column:priority;not null;default:0" json:"priority"`
	Enabled           bool      `gorm:"column:enabled;not null;default:true;index:idx_ai_tool_risk_policies_tool_enabled,priority:2" json:"enabled"`
	PolicyVersion     string    `gorm:"column:policy_version;type:varchar(64);not null;default:''" json:"policy_version"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 返回 AIToolRiskPolicy 的数据库表名。
func (AIToolRiskPolicy) TableName() string { return "ai_tool_risk_policies" }

// AIApprovalTask 存储工具审批任务，用于 Human-in-the-Loop 工作流。
//
// 当高风险工具需要人工审批时，系统会创建审批任务记录，
// 用户确认或拒绝后，状态更新并通过 checkpoint 恢复执行。
//
// 状态值说明:
//   - pending: 等待审批
//   - approved: 已批准
//   - rejected: 已拒绝
//   - expired: 已过期
//
// 字段说明:
//   - ID: 任务唯一标识
//   - ApprovalID: 审批唯一标识（用于客户端查询）
//   - CheckpointID: 关联的检查点 ID
//   - SessionID: 所属会话 ID
//   - RunID: 关联的 Run ID
//   - UserID: 发起用户 ID
//   - ToolName: 工具名称
//   - ToolCallID: 工具调用 ID
//   - ArgumentsJSON: 工具参数（JSON 格式）
//   - PreviewJSON: 预览信息（JSON 格式）
//   - Status: 审批状态
//   - ApprovedBy: 审批人 ID
//   - DisapproveReason: 拒绝原因
//   - Comment: 审批备注
//   - TimeoutSeconds: 超时时间（秒）
//   - ExpiresAt: 过期时间
//   - LockExpiresAt: 锁过期时间（用于分布式处理）
//   - MatchedRuleID: 匹配的策略规则 ID
//   - PolicyVersion: 策略版本
//   - DecisionSource: 决策来源（manual, auto_approve）
//   - DecidedAt: 决策时间
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
//   - DeletedAt: 软删除时间
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
	LockExpiresAt    *time.Time     `gorm:"column:lock_expires_at;index" json:"lock_expires_at"`
	MatchedRuleID    *uint64        `gorm:"column:matched_rule_id;index" json:"matched_rule_id"`
	PolicyVersion    *string        `gorm:"column:policy_version;type:varchar(64)" json:"policy_version"`
	DecisionSource   *string        `gorm:"column:decision_source;type:varchar(32)" json:"decision_source"`
	DecidedAt        *time.Time     `gorm:"column:decided_at" json:"decided_at"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

// TableName 返回 AIApprovalTask 的数据库表名。
func (AIApprovalTask) TableName() string { return "ai_approval_tasks" }

// AIApprovalOutboxEvent 存储审批相关的持久化 Outbox 事件。
//
// Outbox 模式确保事件可靠投递，支持事件溯源和最终一致性。
// 事件按顺序处理，支持重试和错误恢复。
//
// 字段说明:
//   - ID: 事件唯一标识
//   - EventID: 事件 UUID
//   - Sequence: 序列号（用于排序）
//   - AggregateID: 聚合根 ID
//   - OccurredAt: 事件发生时间
//   - Version: 事件版本
//   - ApprovalID: 关联的审批 ID
//   - ToolCallID: 工具调用 ID
//   - EventType: 事件类型
//   - RunID: 关联的 Run ID
//   - SessionID: 所属会话 ID
//   - PayloadJSON: 事件负载（JSON 格式）
//   - Status: 事件状态（pending, processed, failed）
//   - RetryCount: 重试次数
//   - NextRetryAt: 下次重试时间
//   - CreatedAt: 创建时间
//   - UpdatedAt: 更新时间
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
	PayloadJSON string     `gorm:"column:payload_json;type:longtext;not null" json:"payload_json"`
	Status      string     `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_ai_approval_outbox_events_queue,priority:1" json:"status"`
	RetryCount  int        `gorm:"column:retry_count;not null;default:0" json:"retry_count"`
	NextRetryAt *time.Time `gorm:"column:next_retry_at;index:idx_ai_approval_outbox_events_queue,priority:2" json:"next_retry_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime;index:idx_ai_approval_outbox_events_queue,priority:3" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 返回 AIApprovalOutboxEvent 的数据库表名。
func (AIApprovalOutboxEvent) TableName() string { return "ai_approval_outbox_events" }

// BeforeCreate 在创建事件前自动填充默认值。
//
// 自动生成 EventID、OccurredAt、Version 和 Sequence 字段。
// 确保事件数据的完整性和一致性。
//
// 参数:
//   - tx: GORM 数据库事务
//
// 返回: 错误信息
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
