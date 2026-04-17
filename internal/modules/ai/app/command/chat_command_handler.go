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

var chatContextBudget = runtimecontext.Budget{
	Pinned:  1,
	Recent:  12,
	History: 6,
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
	contextPayload := normalizeChatContext(req.Context)
	return h.useCase.Chat(ctx, logic.ChatInput{
		SessionID:       req.SessionID,
		ClientRequestID: req.ClientRequestID,
		LastEventID:     req.LastEventID,
		Message:         req.Message,
		Scene:           req.Scene,
		Context:         contextPayload,
		UserID:          req.UserID,
	}, emit)
}

func normalizeChatContext(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}

	normalized := make(map[string]any, len(raw)+1)
	for key, value := range raw {
		normalized[key] = value
	}
	normalized["context_budget"] = chatContextBudget

	messages := normalizeRuntimeMessages(raw["messages"])
	if len(messages) == 0 {
		return normalized
	}

	normalized["messages"] = runtimecontext.CompressOverflow(messages, chatContextBudget)
	return normalized
}

func normalizeRuntimeMessages(raw any) []runtimecontext.Message {
	switch typed := raw.(type) {
	case []runtimecontext.Message:
		return append([]runtimecontext.Message(nil), typed...)
	case []any:
		result := make([]runtimecontext.Message, 0, len(typed))
		for _, item := range typed {
			msg, ok := normalizeRuntimeMessage(item)
			if !ok {
				continue
			}
			result = append(result, msg)
		}
		return result
	case []map[string]any:
		result := make([]runtimecontext.Message, 0, len(typed))
		for _, item := range typed {
			msg, ok := normalizeRuntimeMessage(item)
			if !ok {
				continue
			}
			result = append(result, msg)
		}
		return result
	default:
		return nil
	}
}

func normalizeRuntimeMessage(raw any) (runtimecontext.Message, bool) {
	switch typed := raw.(type) {
	case runtimecontext.Message:
		return typed, true
	case map[string]any:
		msg := runtimecontext.Message{}
		if role, ok := typed["role"].(string); ok {
			msg.Role = role
		}
		if content, ok := typed["content"].(string); ok {
			msg.Content = content
		}
		if pinned, ok := typed["pinned"].(bool); ok {
			msg.Pinned = pinned
		}
		if msg.Role == "" && msg.Content == "" {
			return runtimecontext.Message{}, false
		}
		return msg, true
	default:
		return runtimecontext.Message{}, false
	}
}
