package v1

// ChatRequest defines the request body for AI chat.
type ChatRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"`
	Scene     string `json:"scene,omitempty"`
	Context   any    `json:"context,omitempty"`
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
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
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
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
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
