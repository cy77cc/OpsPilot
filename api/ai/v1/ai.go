package v1

import (
	"time"

	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

// ChatRequest is the request body for sending a chat message to the AI assistant.
type ChatRequest struct {
	SessionID string         `json:"sessionId"`
	Message   string         `json:"message" binding:"required"`
	Context   map[string]any `json:"context"`
}

// AISession represents an AI chat session with its message history.
type AISession struct {
	ID        string           `json:"id"`
	Scene     string           `json:"scene,omitempty"`
	Title     string           `json:"title"`
	Messages  []map[string]any `json:"messages"`
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
	Risk       tools.ToolRisk `json:"risk"`
	Mode       tools.ToolMode `json:"mode"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"createdAt"`
	ExpiresAt  time.Time      `json:"expiresAt"`
	RequestUID uint64         `json:"requestUid"`
	ReviewUID  uint64         `json:"reviewUid,omitempty"`
	Meta       tools.ToolMeta `json:"-"`
}

// ExecutionRecord tracks the result of an AI tool execution.
type ExecutionRecord struct {
	ID         string            `json:"id"`
	Tool       string            `json:"tool"`
	Params     map[string]any    `json:"params"`
	Mode       tools.ToolMode    `json:"mode"`
	Status     string            `json:"status"`
	Result     *tools.ToolResult `json:"result,omitempty"`
	ApprovalID string            `json:"approvalId,omitempty"`
	RequestUID uint64            `json:"requestUid"`
	CreatedAt  time.Time         `json:"createdAt"`
	FinishedAt *time.Time        `json:"finishedAt,omitempty"`
	Error      string            `json:"error,omitempty"`
}
