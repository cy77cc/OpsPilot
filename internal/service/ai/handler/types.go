// Package handler provides AI service HTTP handlers
package handler

import (
	coreai "github.com/cy77cc/k8s-manage/internal/ai"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/cy77cc/k8s-manage/internal/svc"
)

// ChatRequest represents the request body for chat endpoint
type ChatRequest struct {
	SessionID string         `json:"sessionId"`
	Message   string         `json:"message" binding:"required"`
	Context   map[string]any `json:"context"`
}

// AIHandler handles AI service HTTP requests
type AIHandler struct {
	svcCtx   *svc.ServiceContext
	ai       *coreai.AIAgent
	sessions *logic.SessionStore
	runtime  *logic.RuntimeStore
}

// NewAIHandler creates a new AIHandler instance
func NewAIHandler(svcCtx *svc.ServiceContext) *AIHandler {
	sessions := logic.NewSessionStore(svcCtx.DB, svcCtx.Rdb)
	runtime := logic.NewRuntimeStore(svcCtx.DB)
	return &AIHandler{
		svcCtx:   svcCtx,
		ai:       svcCtx.AI,
		sessions: sessions,
		runtime:  runtime,
	}
}
