package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/ai/agents/diagnosis"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/intent"
	"github.com/cy77cc/OpsPilot/internal/ai/agents/qa"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IntentRouter interface {
	Route(ctx context.Context, message string) (intent.Decision, error)
}

type QAAgent interface {
	Answer(ctx context.Context, req qa.Request) (qa.Result, error)
}

type DiagnosisAgent interface {
	Diagnose(ctx context.Context, req diagnosis.Request) (diagnosis.Result, error)
}

type Dependencies struct {
	ChatDAO            *dao.AIChatDAO
	RunDAO             *dao.AIRunDAO
	DiagnosisReportDAO *dao.AIDiagnosisReportDAO
	IntentRouter       IntentRouter
	QAAgent            QAAgent
	DiagnosisAgent     DiagnosisAgent
}

type Handler struct {
	deps Dependencies
}

func New(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) Chat(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}

	if h.deps.IntentRouter == nil {
		h.deps.IntentRouter = intent.NewRouter()
	}
	if h.deps.QAAgent == nil {
		h.deps.QAAgent = qa.NewAgent()
	}
	if h.deps.DiagnosisAgent == nil {
		h.deps.DiagnosisAgent = diagnosis.NewAgent()
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	userID := httpx.UIDFromCtx(c)
	scene := "ai"
	title := buildSessionTitle(req.Message)

	if session, err := h.deps.ChatDAO.GetSession(c.Request.Context(), sessionID, userID); err != nil {
		httpx.ServerErr(c, err)
		return
	} else if session == nil {
		if err := h.deps.ChatDAO.CreateSession(c.Request.Context(), &model.AIChatSession{
			ID:     sessionID,
			UserID: userID,
			Scene:  scene,
			Title:  title,
		}); err != nil {
			httpx.ServerErr(c, err)
			return
		}
	}

	userMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message,
		Status:    "done",
	}
	if err := h.deps.ChatDAO.CreateMessage(c.Request.Context(), userMessage); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	run := &model.AIRun{
		ID:            uuid.NewString(),
		SessionID:     sessionID,
		UserMessageID: userMessage.ID,
		Status:        "running",
	}
	if err := h.deps.RunDAO.CreateRun(c.Request.Context(), run); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	c.Status(200)
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	writeChatEvent(c, "init", gin.H{"session_id": sessionID, "run_id": run.ID})

	decision, err := h.deps.IntentRouter.Route(c.Request.Context(), req.Message)
	if err != nil {
		_ = h.deps.RunDAO.UpdateRunStatus(c.Request.Context(), run.ID, dao.AIRunStatusUpdate{Status: "failed", ErrorMessage: err.Error()})
		writeChatEvent(c, "error", gin.H{"message": err.Error()})
		return
	}
	writeChatEvent(c, "intent", gin.H{"intent_type": decision.IntentType, "assistant_type": decision.AssistantType, "risk_level": decision.RiskLevel})
	writeChatEvent(c, "status", gin.H{"status": "running"})

	assistantMessage := &model.AIChatMessage{
		ID:        uuid.NewString(),
		SessionID: sessionID,
		Role:      "assistant",
		Status:    "streaming",
	}
	if err := h.deps.ChatDAO.CreateMessage(c.Request.Context(), assistantMessage); err != nil {
		httpx.ServerErr(c, err)
		return
	}

	run.IntentType = decision.IntentType
	run.AssistantType = decision.AssistantType
	run.RiskLevel = decision.RiskLevel
	run.AssistantMessageID = assistantMessage.ID

	switch decision.IntentType {
	case intent.IntentTypeDiagnosis:
		result, err := h.deps.DiagnosisAgent.Diagnose(c.Request.Context(), diagnosis.Request{Message: req.Message})
		if err != nil {
			_ = h.deps.RunDAO.UpdateRunStatus(c.Request.Context(), run.ID, dao.AIRunStatusUpdate{Status: "failed", ErrorMessage: err.Error(), AssistantMessageID: assistantMessage.ID})
			writeChatEvent(c, "error", gin.H{"message": err.Error()})
			return
		}
		for _, progress := range result.Progress {
			writeChatEvent(c, "progress", gin.H{"summary": progress})
		}
		report := &model.AIDiagnosisReport{
			ID:                  uuid.NewString(),
			RunID:               run.ID,
			SessionID:           sessionID,
			Summary:             result.Report.Summary,
			EvidenceJSON:        mustJSON(result.Report.Evidence),
			RootCausesJSON:      mustJSON(result.Report.RootCauses),
			RecommendationsJSON: mustJSON(result.Report.Recommendations),
			GeneratedAt:         time.Now().UTC(),
		}
		if err := h.deps.DiagnosisReportDAO.CreateReport(c.Request.Context(), report); err != nil {
			httpx.ServerErr(c, err)
			return
		}
		if err := h.updateAssistantMessage(c, assistantMessage.ID, sessionID, result.Report.Summary); err != nil {
			httpx.ServerErr(c, err)
			return
		}
		if err := h.deps.RunDAO.UpdateRunStatus(c.Request.Context(), run.ID, dao.AIRunStatusUpdate{
			Status:             "completed",
			AssistantMessageID: assistantMessage.ID,
			ProgressSummary:    result.Report.Summary,
		}); err != nil {
			httpx.ServerErr(c, err)
			return
		}
		writeChatEvent(c, "report_ready", gin.H{"report_id": report.ID, "summary": report.Summary})
	default:
		result, err := h.deps.QAAgent.Answer(c.Request.Context(), qa.Request{Message: req.Message})
		if err != nil {
			_ = h.deps.RunDAO.UpdateRunStatus(c.Request.Context(), run.ID, dao.AIRunStatusUpdate{Status: "failed", ErrorMessage: err.Error(), AssistantMessageID: assistantMessage.ID})
			writeChatEvent(c, "error", gin.H{"message": err.Error()})
			return
		}
		if err := h.updateAssistantMessage(c, assistantMessage.ID, sessionID, result.Text); err != nil {
			httpx.ServerErr(c, err)
			return
		}
		if err := h.deps.RunDAO.UpdateRunStatus(c.Request.Context(), run.ID, dao.AIRunStatusUpdate{
			Status:             "completed",
			AssistantMessageID: assistantMessage.ID,
			ProgressSummary:    result.Text,
		}); err != nil {
			httpx.ServerErr(c, err)
			return
		}
		writeChatEvent(c, "delta", gin.H{"content": result.Text})
	}

	writeChatEvent(c, "done", gin.H{"run_id": run.ID, "status": "completed"})
}

func (h *Handler) updateAssistantMessage(c *gin.Context, messageID, sessionID, content string) error {
	return h.deps.ChatDAO.UpdateMessage(c.Request.Context(), messageID, map[string]any{
		"content": content,
		"status":  "done",
	})
}

func buildSessionTitle(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "New AI session"
	}
	if len(trimmed) > 48 {
		return trimmed[:48]
	}
	return trimmed
}

func writeChatEvent(c *gin.Context, event string, data any) {
	payload, err := airuntime.EncodePublicEvent(event, data)
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"event":"error","data":{"message":%q}}`, err.Error()))
	}
	_, _ = c.Writer.Write([]byte("data: "))
	_, _ = c.Writer.Write(payload)
	_, _ = c.Writer.Write([]byte("\n\n"))
	c.Writer.Flush()
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
