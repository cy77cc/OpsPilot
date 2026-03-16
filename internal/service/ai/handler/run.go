package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetRun(c *gin.Context) {
	if h.deps.RunDAO == nil {
		httpx.OK(c, gin.H{"run": gin.H{}})
		return
	}
	run, err := h.deps.RunDAO.GetRun(c.Request.Context(), c.Param("runId"))
	if err != nil {
		httpx.ServerErr(c, err)
		return
	}
	if run == nil {
		httpx.NotFound(c, "run not found")
		return
	}
	payload := gin.H{
		"run_id":           run.ID,
		"status":           run.Status,
		"assistant_type":   run.AssistantType,
		"intent_type":      run.IntentType,
		"progress_summary": run.ProgressSummary,
	}
	if h.deps.DiagnosisReportDAO != nil {
		report, err := h.deps.DiagnosisReportDAO.GetReportByRunID(c.Request.Context(), run.ID)
		if err != nil {
			httpx.ServerErr(c, err)
			return
		}
		if report != nil {
			payload["report"] = gin.H{
				"report_id": report.ID,
				"summary":   report.Summary,
			}
		}
	}
	httpx.OK(c, gin.H{"run": payload})
}
