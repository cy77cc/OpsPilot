package v1

// ChatRequest defines the request body for AI chat.
type ChatRequest struct {
	SessionID       string `json:"session_id,omitempty"`
	ClientRequestID string `json:"client_request_id,omitempty"`
	LastEventID     string `json:"last_event_id,omitempty"`
	Message         string `json:"message"`
	Scene           string `json:"scene,omitempty"`
	Context         any    `json:"context,omitempty"`
}

// ChatResponse defines the response body for AI chat acceptance.
type ChatResponse struct {
	SessionID string `json:"session_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	Status    string `json:"status"`
}

// CreateSessionRequest defines the request body for creating an AI session.
type CreateSessionRequest struct {
	Title string `json:"title"`
	Scene string `json:"scene"`
}

// CreateSessionResponse defines the response body for creating an AI session.
type CreateSessionResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Scene     string `json:"scene"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ChatMessageResponse defines one message item in a session.
type ChatMessageResponse struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	Content      string `json:"content"`
	Status       string `json:"status,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// SessionResponse defines one AI session with messages.
type SessionResponse struct {
	ID        string                `json:"id"`
	Title     string                `json:"title"`
	Scene     string                `json:"scene,omitempty"`
	Messages  []ChatMessageResponse `json:"messages,omitempty"`
	CreatedAt string                `json:"created_at,omitempty"`
	UpdatedAt string                `json:"updated_at,omitempty"`
}

// RunReportMeta defines summary metadata for linked diagnosis report in run status.
type RunReportMeta struct {
	ID        string `json:"id"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// RunResponse defines the run status payload.
type RunResponse struct {
	ID                 string         `json:"id"`
	SessionID          string         `json:"session_id,omitempty"`
	UserMessageID      string         `json:"user_message_id,omitempty"`
	AssistantMessageID *string        `json:"assistant_message_id,omitempty"`
	Status             string         `json:"status"`
	IntentType         string         `json:"intent_type,omitempty"`
	AssistantType      string         `json:"assistant_type,omitempty"`
	RiskLevel          string         `json:"risk_level,omitempty"`
	TraceID            string         `json:"trace_id,omitempty"`
	ErrorMessage       string         `json:"error_message,omitempty"`
	ProgressSummary    string         `json:"progress_summary,omitempty"`
	Report             *RunReportMeta `json:"report,omitempty"`
}

// SessionSummary defines concise session information.
type SessionSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Scene     string `json:"scene"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// MessageSummary defines concise message information.
type MessageSummary struct {
	ID           string `json:"id"`
	Role         string `json:"role"`
	Content      string `json:"content"`
	Status       string `json:"status,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
}

// SessionDetail defines detailed session information with messages.
type SessionDetail struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Scene     string           `json:"scene"`
	Messages  []MessageSummary `json:"messages,omitempty"`
	CreatedAt string           `json:"created_at,omitempty"`
	UpdatedAt string           `json:"updated_at,omitempty"`
}

// RunReportSummary defines summarized report section in run status.
type RunReportSummary struct {
	ReportID string `json:"report_id"`
	Summary  string `json:"summary,omitempty"`
}

// RunStatusResponse defines backward-compatible run status response.
type RunStatusResponse struct {
	RunID           string            `json:"run_id"`
	Status          string            `json:"status"`
	AssistantType   string            `json:"assistant_type,omitempty"`
	IntentType      string            `json:"intent_type,omitempty"`
	ProgressSummary string            `json:"progress_summary,omitempty"`
	Report          *RunReportSummary `json:"report,omitempty"`
}

// DiagnosisReportResponse defines diagnosis report payload.
type DiagnosisReportResponse struct {
	ID              string   `json:"report_id"`
	RunID           string   `json:"run_id,omitempty"`
	SessionID       string   `json:"session_id,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
	RootCauses      []string `json:"root_causes,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
	Status          string   `json:"status,omitempty"`
	GeneratedAt     string   `json:"generated_at,omitempty"`
}

// =============================================================================
// 审批相关类型
// =============================================================================

// ApprovalPreview 审批预览信息。
type ApprovalPreview struct {
	Action    string   `json:"action"`
	Target    string   `json:"target"`
	RiskLevel string   `json:"risk_level"`
	Impact    string   `json:"impact"`
	Warnings  []string `json:"warnings,omitempty"`
}

// ApprovalInfoResponse 审批请求详情响应。
type ApprovalInfoResponse struct {
	ApprovalID     string          `json:"approval_id"`
	CallID         string          `json:"call_id"`
	ToolName       string          `json:"tool_name"`
	Arguments      string          `json:"arguments"`
	Preview        ApprovalPreview `json:"preview"`
	TimeoutSeconds int             `json:"timeout_seconds,omitempty"`
	CreatedAt      string          `json:"created_at,omitempty"`
	ExpiresAt      string          `json:"expires_at,omitempty"`
}

// SubmitApprovalRequest 提交审批结果请求。
type SubmitApprovalRequest struct {
	Approved         bool   `json:"approved"`
	DisapproveReason string `json:"disapprove_reason,omitempty"`
	Comment          string `json:"comment,omitempty"`
}

// SubmitApprovalResponse 提交审批结果响应。
type SubmitApprovalResponse struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"` // approved, rejected, expired
	Message    string `json:"message,omitempty"`
}

// RetryResumeApprovalRequest requeues a retryable approval-owned run resume.
type RetryResumeApprovalRequest struct {
	TriggerID string `json:"trigger_id"`
}

// RetryResumeApprovalResponse describes the retry-resume enqueue outcome.
type RetryResumeApprovalResponse struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
}

// ResumeApprovalRequest is retained for internal worker recovery flows only.
//
// Deprecated: the web approval flow is submit-only and should rely on pending/get/submit
// plus projection polling or stream updates instead of a direct resume call.
type ResumeApprovalRequest struct {
	SessionID  string `json:"session_id"`
	ApprovalID string `json:"approval_id"`
	Approved   bool   `json:"approved"`
	Reason     string `json:"reason,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

// ApprovalStatusResponse describes the persisted approval decision state.
type ApprovalStatusResponse struct {
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"` // pending, approved, rejected, expired
	ToolName   string `json:"tool_name"`
	ApprovedBy string `json:"approved_by,omitempty"`
	ApprovedAt string `json:"approved_at,omitempty"`
	Comment    string `json:"comment,omitempty"`
}
