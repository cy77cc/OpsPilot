package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetDiagnosisReport(c *gin.Context) {
	httpx.OK(c, gin.H{"report": gin.H{}})
}
