package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetRun(c *gin.Context) {
	run, report, err := h.logic.GetRun(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("runId"))
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
	if report != nil {
		if report.Summary != "" {
			payload["progress_summary"] = report.Summary
		}
		payload["report"] = gin.H{
			"id":      report.ID,
			"summary": report.Summary,
		}
	}
	httpx.OK(c, payload)
}
