// Package handler provides AI service HTTP handlers
package handler

import (
	"context"

	coreai "github.com/cy77cc/k8s-manage/internal/ai"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/service/ai/logic"
	"github.com/cy77cc/k8s-manage/internal/svc"
)

// ChatRequest represents the request body for chat endpoint
type ChatRequest struct {
	SessionID string         `json:"sessionId"`
	Message   string         `json:"message" binding:"required"`
	Context   map[string]any `json:"context"`
}

type aiToolRunner interface {
	ToolMetas() []tools.ToolMeta
	RunTool(ctx context.Context, toolName string, params map[string]any) (tools.ToolResult, error)
	Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
}

type aiOrchestrator interface {
	ChatStream(ctx context.Context, req coreai.ChatStreamRequest, emit func(event string, payload map[string]any) bool) error
	ResumePayload(ctx context.Context, checkpointID string, targets map[string]any) (map[string]any, error)
}

type aiControlPlane interface {
	ToolPolicy(ctx context.Context, meta tools.ToolMeta, params map[string]any) error
	HasPermission(uid uint64, code string) bool
	IsAdmin(uid uint64) bool
	FindMeta(name string) (tools.ToolMeta, bool)
}

// AIHandler handles AI service HTTP requests
type AIHandler struct {
	svcCtx        *svc.ServiceContext
	ai            aiToolRunner
	orchestrator  aiOrchestrator
	control       aiControlPlane
	sessions      *logic.SessionStore
	runtime       *logic.RuntimeStore
}

// NewAIHandler creates a new AIHandler instance
func NewAIHandler(svcCtx *svc.ServiceContext) *AIHandler {
	sessions := logic.NewSessionStore(svcCtx.DB, svcCtx.Rdb)
	runtime := logic.NewRuntimeStore(svcCtx.DB)
	control := coreai.NewControlPlane(svcCtx.DB, runtime, svcCtx.AI)
	return &AIHandler{
		svcCtx:       svcCtx,
		ai:           svcCtx.AI,
		orchestrator: coreai.NewOrchestrator(svcCtx.AI, sessions, runtime, control),
		control:      control,
		sessions:     sessions,
		runtime:      runtime,
	}
}
