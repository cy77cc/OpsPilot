package ai

import "time"

// ChatRequest captures the payload required to send a user message to the AI service.
type ChatRequest struct {
	SessionID string         `json:"session_id,omitempty"`
	Scene     string         `json:"scene,omitempty"`
	Message   string         `json:"message"`
	Intent    string         `json:"intent,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// ChatResponse represents the minimal acknowledgment returned by POST /api/v1/ai/chat.
type ChatResponse struct {
	SessionID string    `json:"session_id"`
	RunID     string    `json:"run_id"`
	MessageID string    `json:"message_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// CreateSessionRequest is the payload for creating a new chat session.
type CreateSessionRequest struct {
	Scene string `json:"scene,omitempty"`
	Title string `json:"title,omitempty"`
}

// CreateSessionResponse is returned after creating a new chat session.
type CreateSessionResponse struct {
	ID        string    `json:"id"`
	Scene     string    `json:"scene"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ChatMessageResponse matches the chat message shape expected by the frontend AI client.
type ChatMessageResponse struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Status    string    `json:"status,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionResponse matches the session shape currently consumed by the frontend AI client.
type SessionResponse struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`
	Messages  []ChatMessageResponse `json:"messages"`
	CreatedAt time.Time             `json:"createdAt"`
	UpdatedAt time.Time             `json:"updatedAt"`
}

// RunResponse mirrors the AI run data returned by GET /api/v1/ai/runs/:runId.
type RunResponse struct {
	ID                 string         `json:"run_id"`
	SessionID          string         `json:"session_id"`
	UserMessageID      string         `json:"user_message_id"`
	AssistantMessageID *string        `json:"assistant_message_id,omitempty"`
	Status             string         `json:"status"`
	IntentType         string         `json:"intent_type"`
	AssistantType      string         `json:"assistant_type"`
	RiskLevel          string         `json:"risk_level"`
	TraceID            string         `json:"trace_id"`
	ErrorMessage       string         `json:"error_message"`
	ProgressSummary    string         `json:"progress_summary"`
	StartedAt          *time.Time     `json:"started_at,omitempty"`
	FinishedAt         *time.Time     `json:"finished_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	Report             *RunReportMeta `json:"report,omitempty"`
}

// RunReportMeta exposes lightweight metadata about a linked diagnosis report.
type RunReportMeta struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DiagnosisReportResponse represents the structured diagnosis data for GET /api/v1/ai/diagnosis/:reportId.
type DiagnosisReportResponse struct {
	ID                  string    `json:"id"`
	RunID               string    `json:"run_id"`
	SessionID           string    `json:"session_id"`
	Summary             string    `json:"summary"`
	ImpactScope         string    `json:"impact_scope"`
	SuspectedRootCauses string    `json:"suspected_root_causes"`
	Evidence            string    `json:"evidence"`
	Recommendations     string    `json:"recommendations"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
