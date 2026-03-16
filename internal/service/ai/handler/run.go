package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetRun(c *gin.Context) {
	httpx.OK(c, gin.H{"run": gin.H{}})
}
