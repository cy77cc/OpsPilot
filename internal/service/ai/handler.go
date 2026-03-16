package ai

import (
	"errors"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
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
	userID := httpx.UIDFromCtx(c)
	sessions, err := h.deps.ChatDAO.ListSessions(c.Request.Context(), userID, strings.TrimSpace(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	items := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		messages, err := h.deps.ChatDAO.ListMessagesBySession(c.Request.Context(), session.ID)
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}

		items = append(items, SessionResponse{
			ID:        session.ID,
			Title:     session.Title,
			Messages:  toChatMessages(messages),
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
		})
	}

	httpx.OK(c, items)
}

// CreateSession is a placeholder for POST /api/v1/ai/sessions.
func (h *Handler) CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	scene := strings.TrimSpace(req.Scene)
	if scene == "" {
		scene = "default"
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "AI session"
	}

	session := &model.AIChatSession{
		ID:     uuid.NewString(),
		UserID: httpx.UIDFromCtx(c),
		Scene:  scene,
		Title:  title,
	}
	if err := h.deps.ChatDAO.CreateSession(c.Request.Context(), session); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, CreateSessionResponse{
		ID:        session.ID,
		Scene:     session.Scene,
		Title:     session.Title,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	})
}

func (h *Handler) GetSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := httpx.UIDFromCtx(c)
	sessionID := c.Param("id")
	session, err := h.deps.ChatDAO.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.NotFound(c, "session not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if session.UserID != userID {
		httpx.NotFound(c, "session not found")
		return
	}

	messages, err := h.deps.ChatDAO.ListMessagesBySession(ctx, sessionID)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, SessionResponse{
		ID:        session.ID,
		Title:     session.Title,
		Messages:  toChatMessages(messages),
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	ctx := c.Request.Context()
	userID := httpx.UIDFromCtx(c)
	sessionID := c.Param("id")
	session, err := h.deps.ChatDAO.GetSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.NotFound(c, "session not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if session.UserID != userID {
		httpx.NotFound(c, "session not found")
		return
	}
	if err := h.deps.ChatDAO.DeleteSession(ctx, sessionID); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	httpx.OK(c, nil)
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

func (h *Handler) GetDiagnosisReport(c *gin.Context) {
	ctx := c.Request.Context()
	userID := httpx.UIDFromCtx(c)
	reportID := c.Param("reportId")
	report, err := h.deps.DiagnosisDAO.GetReport(ctx, reportID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.NotFound(c, "diagnosis report not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}

	session, err := h.deps.ChatDAO.GetSession(ctx, report.SessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.NotFound(c, "diagnosis report not found")
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if session.UserID != userID {
		httpx.NotFound(c, "diagnosis report not found")
		return
	}

	httpx.OK(c, DiagnosisReportResponse{
		ID:                  report.ID,
		RunID:               report.RunID,
		SessionID:           report.SessionID,
		Summary:             report.Summary,
		ImpactScope:         report.ImpactScope,
		SuspectedRootCauses: report.SuspectedRootCauses,
		Evidence:            report.Evidence,
		Recommendations:     report.Recommendations,
		Status:              report.Status,
		CreatedAt:           report.CreatedAt,
		UpdatedAt:           report.UpdatedAt,
	})
}

func toChatMessages(messages []model.AIChatMessage) []ChatMessageResponse {
	items := make([]ChatMessageResponse, 0, len(messages))
	for _, message := range messages {
		items = append(items, ChatMessageResponse{
			ID:        message.ID,
			Role:      message.Role,
			Content:   message.Content,
			Status:    message.Status,
			Timestamp: message.CreatedAt,
		})
	}
	return items
}
