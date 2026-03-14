// Package v1 定义 AI 服务 HTTP 接口的请求/响应数据结构。
//
// 这些类型是 handler 层与外部调用方之间的契约，不包含任何业务逻辑。
package v1

import (
	"time"
)

// ChatRequest is the request body for sending a chat message to the AI assistant.
type ChatRequest struct {
	SessionID string         `json:"sessionId"`
	Message   string         `json:"message" binding:"required"`
	Context   map[string]any `json:"context"`
}

// AIReplayBlock represents a persisted renderable block within a turn.
type AIReplayBlock struct {
	ID          string         `json:"id"`
	BlockType   string         `json:"blockType"`
	Position    int            `json:"position"`
	Status      string         `json:"status,omitempty"`
	Title       string         `json:"title,omitempty"`
	ContentText string         `json:"contentText,omitempty"`
	ContentJSON map[string]any `json:"contentJson,omitempty"`
	Streaming   bool           `json:"streaming,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// AIReplayTurn represents a persisted turn with ordered blocks.
type AIReplayTurn struct {
	ID           string          `json:"id"`
	Role         string          `json:"role"`
	Status       string          `json:"status,omitempty"`
	Phase        string          `json:"phase,omitempty"`
	RuntimeMode  string          `json:"runtimeMode,omitempty"`
	TraceID      string          `json:"traceId,omitempty"`
	ParentTurnID string          `json:"parentTurnId,omitempty"`
	Blocks       []AIReplayBlock `json:"blocks"`
	CreatedAt    time.Time       `json:"createdAt"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	CompletedAt  *time.Time      `json:"completedAt,omitempty"`
}

// AISession represents an AI chat session with its message history.
type AISession struct {
	ID        string           `json:"id"`
	Scene     string           `json:"scene,omitempty"`
	Title     string           `json:"title"`
	RuntimeMode string         `json:"runtimeMode,omitempty"`
	Messages  []map[string]any `json:"messages"`
	Turns     []AIReplayTurn   `json:"turns,omitempty"`
	CreatedAt time.Time        `json:"createdAt"`
	UpdatedAt time.Time        `json:"updatedAt"`
}

// RecommendationRecord is a single AI-generated recommendation entry.
type RecommendationRecord struct {
	ID             string    `json:"id"`
	UserID         uint64    `json:"userId"`
	Scene          string    `json:"scene"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	FollowupPrompt string    `json:"followup_prompt,omitempty"`
	Reasoning      string    `json:"reasoning,omitempty"`
	Relevance      float64   `json:"relevance"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ApprovalTicket represents a pending approval request for a mutating AI tool call.
type ApprovalTicket struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Params     map[string]any `json:"params"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	ExpiresAt  time.Time      `json:"expiresAt"`
	RequestUID uint64         `json:"requestUid"`
	ReviewUID  uint64         `json:"reviewUid,omitempty"`
}

// ExecutionRecord tracks the result of an AI tool execution.
type ExecutionRecord struct {
	ID         string         `json:"id"`
	Tool       string         `json:"tool"`
	Params     map[string]any `json:"params"`
	Status     string         `json:"status"`
	ApprovalID string         `json:"approvalId,omitempty"`
	RequestUID uint64         `json:"requestUid"`
	CreatedAt  time.Time      `json:"createdAt"`
	FinishedAt *time.Time     `json:"finishedAt,omitempty"`
	Error      string         `json:"error,omitempty"`
}

// HostExecutionPlan describes approved host operation execution scope.
type HostExecutionPlan struct {
	ExecutionID string   `json:"execution_id"`
	CommandID   string   `json:"command_id,omitempty"`
	HostIDs     []uint64 `json:"host_ids"`
	Mode        string   `json:"mode"` // command|script
	Command     string   `json:"command,omitempty"`
	ScriptPath  string   `json:"script_path,omitempty"`
	Risk        string   `json:"risk"`
}

// HostExecutionResult is per-host execution output for governed host actions.
type HostExecutionResult struct {
	ExecutionID string    `json:"execution_id"`
	HostID      uint64    `json:"host_id"`
	HostIP      string    `json:"host_ip"`
	HostName    string    `json:"host_name"`
	Status      string    `json:"status"`
	Stdout      string    `json:"stdout"`
	Stderr      string    `json:"stderr"`
	ExitCode    int       `json:"exit_code"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
}
