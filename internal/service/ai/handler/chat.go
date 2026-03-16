package handler

import (
	"github.com/cy77cc/OpsPilot/internal/dao"
	"github.com/cy77cc/OpsPilot/internal/httpx"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	ChatDAO            *dao.AIChatDAO
	RunDAO             *dao.AIRunDAO
	DiagnosisReportDAO *dao.AIDiagnosisReportDAO
}

type Handler struct {
	deps Dependencies
}

func New(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) Chat(c *gin.Context) {
	httpx.OK(c, gin.H{"status": "not_implemented"})
}
