package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	aiv1 "github.com/cy77cc/OpsPilot/api/ai/v1"
	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/service/ai/logic"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc *Service
}

func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) Chat(c *gin.Context) {
	var req aiv1.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindErr(c, err)
		return
	}
	lastEventID := strings.TrimSpace(req.LastEventID)
	if queryLastEventID := strings.TrimSpace(c.Query("last_event_id")); queryLastEventID != "" {
		lastEventID = queryLastEventID
	}
	if headerLastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); headerLastEventID != "" {
		lastEventID = headerLastEventID
	}
	req.LastEventID = lastEventID

	c.Status(200)
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	writer := NewSSEWriter(c.Writer)

	if strings.TrimSpace(req.LastEventID) != "" {
		if h.svc == nil || h.svc.RunDAO == nil || h.svc.RunEventDAO == nil {
			writeChatEvent(writer, c, "error", gin.H{
				"code":    "AI_STREAM_CURSOR_EXPIRED",
				"message": "last_event_id is too old; refresh the stream snapshot",
			})
			return
		}
		run, err := h.svc.RunDAO.FindByClientRequestID(c.Request.Context(), req.SessionID, req.ClientRequestID)
		if err != nil {
			writeChatEvent(writer, c, "error", gin.H{"message": err.Error()})
			return
		}
		if run == nil {
			writeChatEvent(writer, c, "error", gin.H{
				"code":    "AI_STREAM_CURSOR_EXPIRED",
				"message": "last_event_id is too old; refresh the stream snapshot",
			})
			return
		}
		cursor, err := h.svc.RunEventDAO.FindByEventID(c.Request.Context(), run.ID, req.LastEventID)
		if err != nil {
			writeChatEvent(writer, c, "error", gin.H{"message": err.Error()})
			return
		}
		if cursor == nil {
			writeChatEvent(writer, c, "error", gin.H{
				"code":    "AI_STREAM_CURSOR_EXPIRED",
				"message": "last_event_id is too old; refresh the stream snapshot",
			})
			return
		}
	}

	if err := h.svc.Chat(c.Request.Context(), logic.ChatInput{
		SessionID:       req.SessionID,
		ClientRequestID: req.ClientRequestID,
		LastEventID:     req.LastEventID,
		Message:         req.Message,
		Scene:           req.Scene,
		Context:         mapFromAny(req.Context),
		UserID:          httpx.UIDFromCtx(c),
	}, func(event string, data any) {
		writeChatEvent(writer, c, event, data)
	}); err != nil {
		if errors.Is(err, aidao.ErrRunEventCursorExpired) {
			writeChatEvent(writer, c, "error", gin.H{
				"code":    "AI_STREAM_CURSOR_EXPIRED",
				"message": "last_event_id is too old; refresh the stream snapshot",
			})
			return
		}
		writeChatEvent(writer, c, "error", gin.H{"message": err.Error()})
		return
	}
}

func (h *HTTPHandler) CreateSession(c *gin.Context) {
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

func (h *HTTPHandler) ListSessions(c *gin.Context) {
	summaries, err := h.svc.ListSessions(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	sessions := make([]model.AIChatSession, 0, len(summaries))
	for _, summary := range summaries {
		sessions = append(sessions, summary.Session)
	}
	runBySessionAndAssistantMessageID := h.runBySessionAndAssistantMessageID(c.Request.Context(), sessions)
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

func (h *HTTPHandler) GetSession(c *gin.Context) {
	session, messages, err := h.svc.GetSession(c.Request.Context(), httpx.UIDFromCtx(c), strings.TrimSpace(c.Query("scene")), c.Param("id"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if session == nil {
		httpx.NotFound(c, "session not found")
		return
	}
	runByAssistantMessageID := h.runByAssistantMessageID(c.Request.Context(), session.ID)
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

func (h *HTTPHandler) DeleteSession(c *gin.Context) {
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

func (h *HTTPHandler) GetRun(c *gin.Context) {
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

func (h *HTTPHandler) GetRunProjection(c *gin.Context) {
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

func (h *HTTPHandler) GetRunContent(c *gin.Context) {
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

func (h *HTTPHandler) GetDiagnosisReport(c *gin.Context) {
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

func sessionSummaryFromModel(session model.AIChatSession) gin.H {
	return gin.H{
		"id":         session.ID,
		"title":      session.Title,
		"scene":      session.Scene,
		"created_at": formatTime(session.CreatedAt),
		"updated_at": formatTime(session.UpdatedAt),
	}
}

const terminalAssistantErrorMessage = "生成中断，请稍后重试。"

func sessionMessageItem(message model.AIChatMessage, run *model.AIRun) gin.H {
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

func (h *HTTPHandler) buildResumableCredentials(ctx context.Context, run *model.AIRun) *logic.ResumableCredentials {
	if h == nil || h.svc == nil {
		return nil
	}
	creds, err := h.svc.BuildResumableCredentials(ctx, run)
	if err != nil {
		return nil
	}
	return creds
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
	case "waiting_approval", "resuming", "running":
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

func (h *HTTPHandler) runByAssistantMessageID(ctx context.Context, sessionID string) map[string]*model.AIRun {
	result := map[string]*model.AIRun{}
	if h == nil || h.svc == nil || h.svc.RunDAO == nil {
		return result
	}
	runs, err := h.svc.RunDAO.ListBySession(ctx, sessionID)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) != "" {
			runCopy := run
			result[run.AssistantMessageID] = &runCopy
		}
	}
	return result
}

func (h *HTTPHandler) runBySessionAndAssistantMessageID(ctx context.Context, sessions []model.AIChatSession) map[string]map[string]*model.AIRun {
	result := map[string]map[string]*model.AIRun{}
	if h == nil || h.svc == nil || h.svc.RunDAO == nil || len(sessions) == 0 {
		return result
	}
	sessionIDs := make([]string, 0, len(sessions))
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
		result[session.ID] = map[string]*model.AIRun{}
	}
	runs, err := h.svc.RunDAO.ListBySessionIDs(ctx, sessionIDs)
	if err != nil {
		return result
	}
	for _, run := range runs {
		if strings.TrimSpace(run.AssistantMessageID) == "" {
			continue
		}
		if _, ok := result[run.SessionID]; !ok {
			result[run.SessionID] = map[string]*model.AIRun{}
		}
		runCopy := run
		result[run.SessionID][run.AssistantMessageID] = &runCopy
	}
	return result
}

func writeChatEvent(writer *SSEWriter, c *gin.Context, event string, data any) {
	if err := writer.WriteEvent(event, data); err == nil {
		c.Writer.Flush()
	}
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return nil
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
