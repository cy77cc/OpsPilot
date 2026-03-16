package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

func (h *Handler) CreateSession(c *gin.Context) {
	httpx.OK(c, gin.H{"session": gin.H{}})
}

func (h *Handler) ListSessions(c *gin.Context) {
	httpx.OK(c, gin.H{"sessions": []gin.H{}})
}

func (h *Handler) GetSession(c *gin.Context) {
	httpx.OK(c, gin.H{"session": gin.H{}})
}

func (h *Handler) DeleteSession(c *gin.Context) {
	httpx.OK(c, gin.H{"deleted": true})
}
