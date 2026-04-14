package chathandler

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	ssehandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/sse"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	ai "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
	"github.com/gin-gonic/gin"
)

const terminalAssistantErrorMessage = "生成中断，请稍后重试。"

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

func (h *HTTPHandler) buildResumableCredentials(ctx context.Context, run *ai.AIRun) *logic.ResumableCredentials {
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

func (h *HTTPHandler) runByAssistantMessageID(ctx context.Context, sessionID string) map[string]*ai.AIRun {
	if h == nil || h.svc == nil {
		return map[string]*ai.AIRun{}
	}
	return h.svc.RunByAssistantMessageID(ctx, sessionID)
}

func (h *HTTPHandler) runBySessionAndAssistantMessageID(ctx context.Context, sessions []ai.AIChatSession) map[string]map[string]*ai.AIRun {
	if h == nil || h.svc == nil {
		return map[string]map[string]*ai.AIRun{}
	}
	return h.svc.RunBySessionAndAssistantMessageID(ctx, sessions)
}

func writeChatEvent(writer *ssehandler.SSEWriter, c *gin.Context, event string, data any) {
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
