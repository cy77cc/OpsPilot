package http

import (
	ssehandler "github.com/cy77cc/OpsPilot/internal/modules/ai/handler/sse"
	"github.com/gin-gonic/gin"
)

func writeChatEvent(writer *ssehandler.SSEWriter, c *gin.Context, event string, data any) {
	if err := writer.WriteEvent(event, data); err == nil {
		c.Writer.Flush()
	}
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return nil
	}
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return nil
}
