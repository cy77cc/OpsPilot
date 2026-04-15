// Package websocket 提供 WebSocket 连接处理功能。
//
// 本文件实现 WebSocket 连接升级和客户端管理，
// 用于实时推送通知消息。
package websocket

import (
	"net/http"
	"strings"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// upgrader 是 HTTP 到 WebSocket 的升级器。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return isOriginAllowed(r.Header.Get("Origin"), config.CFG.Security.WebSocketAllowOrigins)
	},
}

// HandleWebSocket 处理 WebSocket 连接请求。
//
// 仅从鉴权上下文中获取用户 ID，
// 升级 HTTP 连接为 WebSocket，注册到 Hub 并启动读写协程。
func HandleWebSocket(c *gin.Context) {
	// 仅信任鉴权中间件注入的 uid，禁止查询参数冒充。
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	// 升级 HTTP 连接为 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// 创建客户端
	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    GetHub(),
	}

	// 注册客户端
	client.Hub.Register(client)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()
}

func userIDFromContext(c *gin.Context) (uint64, bool) {
	userID, exists := c.Get("uid")
	if !exists {
		return 0, false
	}

	switch v := userID.(type) {
	case uint:
		return uint64(v), true
	case uint64:
		return v, true
	default:
		return 0, false
	}
}

func isOriginAllowed(origin string, allowOrigins []string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return false
	}

	for _, allowed := range allowOrigins {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}

	return false
}
