package command

import (
	"context"
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/logic"
	runtimecontext "github.com/cy77cc/OpsPilot/internal/modules/ai/runtime/context"
)

// ChatRequest is the command input accepted by the AI chat use case.
type ChatRequest struct {
	SessionID       string
	ClientRequestID string
	LastEventID     string
	Message         string
	Scene           string
	Context         map[string]any
	UserID          uint64
}

// ChatUseCase captures the current chat orchestration used by the HTTP boundary.
type ChatUseCase interface {
	ValidateReplayCursor(ctx context.Context, sessionID, clientRequestID, lastEventID string) error
	Chat(ctx context.Context, input logic.ChatInput, emit logic.EventEmitter) error
}

// ChatHandler handles AI chat commands.
type ChatHandler interface {
	Handle(ctx context.Context, req *ChatRequest, emit logic.EventEmitter) error
}

type chatCommandHandler struct {
	useCase ChatUseCase
}

// NewChatCommandHandler creates the default chat command handler.
func NewChatCommandHandler(useCase ChatUseCase) ChatHandler {
	return &chatCommandHandler{useCase: useCase}
}

func (h *chatCommandHandler) Handle(ctx context.Context, req *ChatRequest, emit logic.EventEmitter) error {
	if req == nil {
		return fmt.Errorf("chat request is nil")
	}
	if h == nil || h.useCase == nil {
		return fmt.Errorf("AI service not initialized")
	}
	if err := h.useCase.ValidateReplayCursor(ctx, req.SessionID, req.ClientRequestID, req.LastEventID); err != nil {
		return err
	}
	return h.useCase.Chat(ctx, h.buildChatInput(req), emit)
}

func (h *chatCommandHandler) buildChatInput(req *ChatRequest) logic.ChatInput {
	return logic.ChatInput{
		SessionID:       req.SessionID,
		ClientRequestID: req.ClientRequestID,
		LastEventID:     req.LastEventID,
		Message:         req.Message,
		Scene:           req.Scene,
		Context:         req.Context,
		Budget:          runtimecontext.DefaultBudget,
		UserID:          req.UserID,
	}
}
