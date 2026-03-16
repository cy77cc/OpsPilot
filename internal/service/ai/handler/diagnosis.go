package handler

import (
	"encoding/json"

	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDiagnosisReport(c *gin.Context) {
	report, err := h.logic.GetDiagnosisReport(c.Request.Context(), httpx.UIDFromCtx(c), c.Param("reportId"))
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
