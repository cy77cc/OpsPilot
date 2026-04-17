package http

import (
	"net/http"

	"github.com/cy77cc/OpsPilot/internal/modules/ai/app/command"
	"github.com/gin-gonic/gin"
)

type ChatRequest = command.ChatRequest

type ChatCommandHandler interface {
	Handle(*ChatRequest) error
}

type ChatHandler struct {
	commandHandler ChatCommandHandler
}

func NewChatHandler(commandHandler ChatCommandHandler) *ChatHandler {
	return &ChatHandler{commandHandler: commandHandler}
}

func (h *ChatHandler) HandleChat(c *gin.Context) {
	req := &ChatRequest{}
	if h.commandHandler != nil {
		_ = h.commandHandler.Handle(req)
	}
	c.Status(http.StatusOK)
}
