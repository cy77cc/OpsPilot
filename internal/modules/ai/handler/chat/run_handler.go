package chathandler

import (
	"errors"
	"strconv"

	"github.com/cy77cc/OpsPilot/internal/core/httpx"
	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	"github.com/gin-gonic/gin"
)

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
