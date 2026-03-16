package handler

import (
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Chat(c *gin.Context) {
	httpx.OK(c, gin.H{"status": "not_implemented"})
}
