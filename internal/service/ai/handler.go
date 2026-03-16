package ai

import (
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler manages the HTTP endpoints for the AI service.
type Handler struct {
	deps *Deps
}

// NewHandler creates a new AI handler.
func NewHandler(deps *Deps) *Handler {
	return &Handler{
		deps: deps,
	}
}

// Chat is a placeholder for POST /api/v1/ai/chat.
func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}

	resp := ChatResponse{
		SessionID: sessionID,
		RunID:     uuid.NewString(),
		MessageID: uuid.NewString(),
		Status:    "accepted",
		Timestamp: time.Now().UTC(),
	}
	httpx.OK(c, resp)
}

// ListSessions is a placeholder for GET /api/v1/ai/sessions.
func (h *Handler) ListSessions(c *gin.Context) {
	now := time.Now().UTC()
	resp := ListSessionsResponse{
		Sessions: []SessionSummary{
			{
				ID:        uuid.NewString(),
				Scene:     "default",
				Title:     "Demo AI session",
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
			},
		},
		Total: 1,
	}
	httpx.OK(c, resp)
}

// CreateSession is a placeholder for POST /api/v1/ai/sessions.
func (h *Handler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	scene := req.Scene
	if scene == "" {
		scene = "default"
	}

	title := req.Title
	if title == "" {
		title = "AI session"
	}

	resp := CreateSessionResponse{
		ID:        uuid.NewString(),
		Scene:     scene,
		Title:     title,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	httpx.OK(c, resp)
}

// GetSession is a placeholder for GET /api/v1/ai/sessions/:id.
func (h *Handler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	now := time.Now().UTC()
	resp := SessionDetail{
		ID:    sessionID,
		Scene: "default",
		Title: "AI session detail",
		Messages: []ChatMessage{
			{
				ID:        uuid.NewString(),
				Role:      "user",
				Content:   "Hello, assistant",
				CreatedAt: now.Add(-2 * time.Minute),
			},
			{
				ID:        uuid.NewString(),
				Role:      "assistant",
				Content:   "Hi there! This is a placeholder response.",
				CreatedAt: now.Add(-time.Minute),
			},
		},
		TotalMessages: 2,
		CreatedAt:     now.Add(-2 * time.Hour),
		UpdatedAt:     now,
	}
	httpx.OK(c, resp)
}

// DeleteSession is a placeholder for DELETE /api/v1/ai/sessions/:id.
func (h *Handler) DeleteSession(c *gin.Context) {
	sessionID := c.Param("id")
	httpx.OK(c, DeleteSessionResponse{
		ID:      sessionID,
		Deleted: true,
	})
}

// GetRun is a placeholder for GET /api/v1/ai/runs/:runId.
func (h *Handler) GetRun(c *gin.Context) {
	runID := c.Param("runId")
	now := time.Now().UTC()
	resp := RunResponse{
		ID:                 runID,
		SessionID:          uuid.NewString(),
		UserMessageID:      uuid.NewString(),
		AssistantMessageID: uuid.NewString(),
		Status:             "pending",
		IntentType:         "diagnosis",
		AssistantType:      "assistant",
		RiskLevel:          "low",
		TraceID:            uuid.NewString(),
		ErrorMessage:       "",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	httpx.OK(c, resp)
}

// GetDiagnosisReport is a placeholder for GET /api/v1/ai/diagnosis/:reportId.
func (h *Handler) GetDiagnosisReport(c *gin.Context) {
	reportID := c.Param("reportId")
	now := time.Now().UTC()
	resp := DiagnosisReportResponse{
		ID:                  reportID,
		RunID:               uuid.NewString(),
		SessionID:           uuid.NewString(),
		Summary:             "Structured diagnosis placeholder",
		ImpactScope:         "cluster",
		SuspectedRootCauses: "Example misconfiguration",
		Evidence:            "Detected degraded pod health",
		Recommendations:     "Review node pressure and restart the affected pod",
		Status:              "draft",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	httpx.OK(c, resp)
}
