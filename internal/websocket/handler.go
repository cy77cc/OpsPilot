// Package websocket 提供 WebSocket 连接处理功能。
//
// 本文件实现 WebSocket 连接升级和客户端管理，
// 用于实时推送通知消息。
package websocket

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// upgrader 是 HTTP 到 WebSocket 的升级器。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应限制
	},
}

// HandleWebSocket 处理 WebSocket 连接请求。
//
// 从查询参数或上下文中获取用户 ID，
// 升级 HTTP 连接为 WebSocket，注册到 Hub 并启动读写协程。
func HandleWebSocket(c *gin.Context) {
	// 从上下文获取用户ID
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		userID, exists := c.Get("user_id")
		if exists {
			switch v := userID.(type) {
			case uint64:
				userIDStr = strconv.FormatUint(v, 10)
			case float64:
				userIDStr = strconv.FormatUint(uint64(v), 10)
			case int:
				userIDStr = strconv.Itoa(v)
			}
		}
	}

	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
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
