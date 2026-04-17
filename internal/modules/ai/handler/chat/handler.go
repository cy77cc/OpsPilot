// Package chathandler 实现 AI 聊天的 HTTP Handler。
package chathandler

import (
	aicommand "github.com/cy77cc/OpsPilot/internal/modules/ai/app/command"
	aihttp "github.com/cy77cc/OpsPilot/internal/modules/ai/interfaces/http"
	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	svc *Service
}

func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

func (h *HTTPHandler) Chat(c *gin.Context) {
	if h == nil || h.svc == nil {
		c.Status(500)
		return
	}
	commandHandler := aicommand.NewChatCommandHandler(h.svc)
	aihttp.NewChatHandler(commandHandler).HandleChat(c)
}
