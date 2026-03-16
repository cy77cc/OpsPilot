package ai

type ChatRequest struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	SessionID string `json:"session_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	Status    string `json:"status"`
}

type CreateSessionRequest struct {
	Title string `json:"title"`
	Scene string `json:"scene"`
}

type SessionSummary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Scene     string `json:"scene"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type MessageSummary struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type SessionDetail struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Scene     string           `json:"scene"`
	Messages  []MessageSummary `json:"messages,omitempty"`
	CreatedAt string           `json:"created_at,omitempty"`
	UpdatedAt string           `json:"updated_at,omitempty"`
}

type RunReportSummary struct {
	ReportID string `json:"report_id"`
	Summary  string `json:"summary,omitempty"`
}

type RunStatusResponse struct {
	RunID            string             `json:"run_id"`
	Status           string             `json:"status"`
	AssistantType    string             `json:"assistant_type,omitempty"`
	IntentType       string             `json:"intent_type,omitempty"`
	ProgressSummary  string             `json:"progress_summary,omitempty"`
	Report           *RunReportSummary  `json:"report,omitempty"`
}

type DiagnosisReportResponse struct {
	ReportID         string   `json:"report_id"`
	RunID            string   `json:"run_id,omitempty"`
	SessionID        string   `json:"session_id,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Evidence         []string `json:"evidence,omitempty"`
	RootCauses       []string `json:"root_causes,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`
	GeneratedAt      string   `json:"generated_at,omitempty"`
}
