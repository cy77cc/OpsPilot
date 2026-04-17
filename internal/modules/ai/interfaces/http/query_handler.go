package http

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

const terminalAssistantErrorMessage = "生成中断，请稍后重试。"

type ChatQueryService interface {
	CreateSession(ctx context.Context, userID uint64, title, scene string) (*ai.AIChatSession, error)
	ListSessions(ctx context.Context, userID uint64, scene string) ([]logic.SessionSummary, error)
	GetSession(ctx context.Context, userID uint64, scene, sessionID string) (*ai.AIChatSession, []ai.AIChatMessage, error)
	DeleteSession(ctx context.Context, userID uint64, sessionID string) (bool, error)
	GetRun(ctx context.Context, userID uint64, runID string) (*ai.AIRun, *ai.AIDiagnosisReport, error)
	BuildResumableCredentials(ctx context.Context, run *ai.AIRun) (*logic.ResumableCredentials, error)
	GetRunProjectionPayload(ctx context.Context, userID uint64, runID string, query logic.RunProjectionQuery) (any, error)
	GetRunContent(ctx context.Context, userID uint64, contentID string) (*ai.AIRunContent, error)
	GetDiagnosisReport(ctx context.Context, userID uint64, reportID string) (*ai.AIDiagnosisReport, error)
	RunByAssistantMessageID(ctx context.Context, sessionID string) map[string]*ai.AIRun
	RunBySessionAndAssistantMessageID(ctx context.Context, sessions []ai.AIChatSession) map[string]map[string]*ai.AIRun
}

type ChatQueryHandler struct {
	svc ChatQueryService
}

func NewChatQueryHandler(svc ChatQueryService) *ChatQueryHandler {
	return &ChatQueryHandler{svc: svc}
}

func (h *ChatQueryHandler) CreateSession(c *gin.Context) {
	var req aiv1.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	session, err := h.svc.CreateSession(c.Request.Context(), httpx.UIDFromCtx(c), req.Title, req.Scene)
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.OK(c, gin.H{})
		return
	}
	httpx.OK(c, sessionSummaryFromModel(*session))
}

func (h *ChatQueryHandler) ListSessions(c *gin.Context) {
	summaries, err := h.svc.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	sessions := make([]ai.AIChatSession, 0, len(summaries))
	for _, summary := range summaries {
		sessions = append(sessions, summary.Session)
	}
	runBySessionAndAssistantMessageID := h.svc.RunBySessionAndAssistantMessageID(c.Request.Context(), sessions)
	items := make([]gin.H, 0, len(summaries))
	for _, summary := range summaries {
		item := sessionSummaryFromModel(summary.Session)
		if summary.LastMessage != nil {
			run := runBySessionAndAssistantMessageID[summary.Session.ID][summary.LastMessage.ID]
			lastMessage := sessionMessageItem(*summary.LastMessage, run)
			mergeResumableCredentials(lastMessage, h.buildResumableCredentials(c.Request.Context(), run))
			item["last_message"] = lastMessage
		}
		items = append(items, item)
	}
	httpx.OK(c, items)
}

func (h *ChatQueryHandler) GetSession(c *gin.Context) {
	session, messages, err := h.svc.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.NotFound(c, "session not found")
		return
	}
	runByAssistantMessageID := h.svc.RunByAssistantMessageID(c.Request.Context(), session.ID)
	messageItems := make([]gin.H, 0, len(messages))
	for _, message := range messages {
		run := runByAssistantMessageID[message.ID]
		item := sessionMessageItem(message, run)
		mergeResumableCredentials(item, h.buildResumableCredentials(c.Request.Context(), run))
		messageItems = append(messageItems, item)
	}
	httpx.OK(c, gin.H{
		"id":         session.ID,
		"title":      session.Title,
		"scene":      session.Scene,
		"messages":   messageItems,
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
	})
}

func (h *ChatQueryHandler) DeleteSession(c *gin.Context) {
	ok, err := h.svc.DeleteSession(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if !ok {
		httpx.NotFound(c, "session not found")
		return
	}
	httpx.OK(c, nil)
}

func (h *ChatQueryHandler) GetRun(c *gin.Context) {
	run, report, err := h.svc.GetRun(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("runId"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if run == nil {
		httpx.NotFound(c, "run not found")
		return
	}
	progressSummary := run.ProgressSummary
	payload := gin.H{
		"id":                   run.ID,
		"run_id":               run.ID,
		"session_id":           run.SessionID,
		"user_message_id":      run.UserMessageID,
		"assistant_message_id": run.AssistantMessageID,
		"status":               run.Status,
		"assistant_type":       run.AssistantType,
		"intent_type":          run.IntentType,
		"progress_summary":     progressSummary,
		"risk_level":           run.RiskLevel,
		"trace_id":             run.TraceID,
		"trace_json":           run.TraceJSON,
		"error_message":        run.ErrorMessage,
		"started_at":           formatTime(run.StartedAt),
		"created_at":           formatTime(run.CreatedAt),
		"updated_at":           formatTime(run.UpdatedAt),
	}
	if run.FinishedAt != nil {
		payload["finished_at"] = formatTime(*run.FinishedAt)
	}
	mergeResumableCredentials(payload, h.buildResumableCredentials(c.Request.Context(), run))
	if report != nil {
		if report.Summary != "" {
			payload["progress_summary"] = report.Summary
		}
		payload["report"] = gin.H{
			"id":        report.ID,
			"report_id": report.ID,
			"summary":   report.Summary,
		}
	}
	httpx.OK(c, payload)
}

func (h *ChatQueryHandler) GetRunProjection(c *gin.Context) {
	query := logic.RunProjectionQuery{}
	if rawCursor, ok := c.GetQuery("cursor"); ok {
		query.Paginate = true
		query.Cursor = rawCursor
		if rawLimit := c.Query("limit"); rawLimit != "" {
			limit, err := strconv.Atoi(rawLimit)
			if err != nil {
				httpx.BadRequest(c, "invalid projection limit")
				return
			}
			query.Limit = limit
		}
	}
	projection, err := h.svc.GetRunProjectionPayload(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("runId"), query)
	if err != nil {
		if errors.Is(err, logic.ErrInvalidProjectionCursor) {
			httpx.BadRequest(c, err.Error())
			return
		}
		httpx.ServerErr(c, err)
		return
	}
	if projection == nil {
		httpx.NotFound(c, "projection not found")
		return
	}
	httpx.OK(c, projection)
}

func (h *ChatQueryHandler) GetRunContent(c *gin.Context) {
	content, err := h.svc.GetRunContent(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if content == nil {
		httpx.NotFound(c, "content not found")
		return
	}
	httpx.OK(c, gin.H{
		"id":           content.ID,
		"run_id":       content.RunID,
		"session_id":   content.SessionID,
		"content_kind": content.ContentKind,
		"encoding":     content.Encoding,
		"summary_text": content.SummaryText,
		"body_text":    content.BodyText,
		"body_json":    content.BodyJSON,
		"size_bytes":   content.SizeBytes,
		"created_at":   formatTime(content.CreatedAt),
	})
}

func (h *ChatQueryHandler) GetDiagnosisReport(c *gin.Context) {
	report, err := h.svc.GetDiagnosisReport(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("reportId"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if report == nil {
		httpx.NotFound(c, "diagnosis report not found")
		return
	}
	httpx.OK(c, gin.H{
		"report_id":       report.ID,
		"run_id":          report.RunID,
		"session_id":      report.SessionID,
		"summary":         report.Summary,
		"evidence":        decodeStringArray(report.EvidenceJSON),
		"root_causes":     decodeStringArray(report.RootCausesJSON),
		"recommendations": decodeStringArray(report.RecommendationsJSON),
		"generated_at":    formatTime(report.GeneratedAt),
	})
}

func (h *ChatQueryHandler) buildResumableCredentials(ctx context.Context, run *ai.AIRun) *logic.ResumableCredentials {
	if h == nil || h.svc == nil {
		return nil
	}
	creds, err := h.svc.BuildResumableCredentials(ctx, run)
	if err != nil {
		return nil
	}
	return creds
}

func sessionSummaryFromModel(session ai.AIChatSession) gin.H {
	return gin.H{
		"id":         session.ID,
		"title":      session.Title,
		"scene":      session.Scene,
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
	}
}

func sessionMessageItem(message ai.AIChatMessage, run *ai.AIRun) gin.H {
	item := gin.H{
		"id":             message.ID,
		"session_id_num": message.SessionIDNum,
		"role":           message.Role,
		"status":         message.Status,
		"created_at":     formatTime(message.CreatedAt),
		"content":        message.Content,
	}
	if run != nil {
		item["run_id"] = run.ID
		if isResumableRunStatus(run.Status) {
			item["status"] = run.Status
		}
		if isTerminalAssistantRun(run.Status) {
			item["status"] = "error"
			item["error_message"] = terminalAssistantErrorMessage
		}
	}
	return item
}

func mergeResumableCredentials(item gin.H, creds *logic.ResumableCredentials) {
	if item == nil || creds == nil {
		return
	}
	if creds.RunID != "" {
		item["run_id"] = creds.RunID
	}
	if creds.ClientRequestID != "" {
		item["client_request_id"] = creds.ClientRequestID
	}
	if creds.LatestEventID != "" {
		item["latest_event_id"] = creds.LatestEventID
	}
	if creds.ApprovalID != "" {
		item["approval_id"] = creds.ApprovalID
	}
	if creds.Resumable {
		item["resumable"] = true
	}
}

func isTerminalAssistantRun(status string) bool {
	switch strings.TrimSpace(status) {
	case "failed", "failed_runtime", "expired":
		return true
	default:
		return false
	}
}

func isResumableRunStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case "waiting_approval", "resuming", "running", "resume_failed_retryable":
		return true
	default:
		return false
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func decodeStringArray(raw string) []string {
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}
