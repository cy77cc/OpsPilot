// Package websocket 提供 WebSocket 连接处理功能。
//
// 本文件实现 WebSocket Hub 消息中心，管理所有客户端连接，
// 支持用户级别的消息广播和心跳检测。
package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/gorilla/websocket"
)

// WSMessage 是 WebSocket 消息格式。
type WSMessage struct {
	Type         string              `json:"type"`                   // 消息类型 (new/update)
	Notification *UserNotificationWS `json:"notification,omitempty"` // 通知内容
	ID           string              `json:"id,omitempty"`           // 通知 ID
	ReadAt       string              `json:"read_at,omitempty"`      // 已读时间
	DismissedAt  string              `json:"dismissed_at,omitempty"` // 忽略时间
	ConfirmedAt  string              `json:"confirmed_at,omitempty"` // 确认时间
}

// UserNotificationWS 是 WebSocket 通知格式。
type UserNotificationWS struct {
	ID             uint               `json:"id"`              // 用户通知 ID
	UserID         uint64             `json:"user_id"`         // 用户 ID
	NotificationID uint               `json:"notification_id"` // 通知 ID
	ReadAt         *time.Time         `json:"read_at"`         // 已读时间
	DismissedAt    *time.Time         `json:"dismissed_at"`    // 忽略时间
	ConfirmedAt    *time.Time         `json:"confirmed_at"`    // 确认时间
	Notification   model.Notification `json:"notification"`    // 通知详情
}

// Client 是 WebSocket 客户端连接。
type Client struct {
	UserID uint64            // 用户 ID
	Conn   *websocket.Conn   // WebSocket 连接
	Send   chan []byte       // 发送消息队列
	Hub    *Hub              // 所属 Hub
}

// Hub 是 WebSocket 连接中心，管理所有客户端连接。
type Hub struct {
	clients    map[uint64]map[*Client]bool // 按用户 ID 分组的客户端集合
	broadcast  chan *BroadcastMessage      // 广播消息通道
	register   chan *Client                // 注册通道
	unregister chan *Client                // 注销通道
	mu         sync.RWMutex                // 读写锁
}

// BroadcastMessage 是广播消息结构。
type BroadcastMessage struct {
	UserID  uint64  // 目标用户 ID
	Message []byte  // 消息内容
}

// hubInstance 是 Hub 单例实例。
var hubInstance *Hub
var hubOnce sync.Once

// GetHub 获取 Hub 单例实例。
//
// 首次调用时初始化 Hub 并启动运行协程。
func GetHub() *Hub {
	hubOnce.Do(func() {
		hubInstance = &Hub{
			clients:    make(map[uint64]map[*Client]bool),
			broadcast:  make(chan *BroadcastMessage, 256),
			register:   make(chan *Client),
			unregister: make(chan *Client),
		}
		go hubInstance.Run()
	})
	return hubInstance
}

// Run 启动 Hub 事件循环。
//
// 处理客户端注册、注销、消息广播和心跳检测。
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		if err := recover(); err != nil {
			log.Println("[websocket] hub panic:", err)
		}
		ticker.Stop()
	}()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.UserID] == nil {
				h.clients[client.UserID] = make(map[*Client]bool)
			}
			h.clients[client.UserID][client] = true
			h.mu.Unlock()
			log.Printf("WebSocket: 用户 %d 连接，当前连接数: %d", client.UserID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.UserID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.UserID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("WebSocket: 用户 %d 断开连接", client.UserID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			clients, ok := h.clients[msg.UserID]
			h.mu.RUnlock()

			if ok {
				for client := range clients {
					select {
					case client.Send <- msg.Message:
					default:
						close(client.Send)
						h.mu.Lock()
						delete(h.clients[msg.UserID], client)
						h.mu.Unlock()
					}
				}
			}

		case <-ticker.C:
			// 心跳检测：发送 ping 给所有客户端
			h.mu.RLock()
			for _, clients := range h.clients {
				for client := range clients {
					if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						log.Printf("WebSocket: 发送 ping 失败: %v", err)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register 注册客户端到 Hub。
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 从 Hub 注销客户端。
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// PushNotification 推送新通知给指定用户。
func (h *Hub) PushNotification(userID uint64, notif *model.UserNotification) {
	msg := WSMessage{
		Type: "new",
		Notification: &UserNotificationWS{
			ID:             notif.ID,
			UserID:         notif.UserID,
			NotificationID: notif.NotificationID,
			ReadAt:         notif.ReadAt,
			DismissedAt:    notif.DismissedAt,
			ConfirmedAt:    notif.ConfirmedAt,
			Notification:   notif.Notification,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: 序列化消息失败: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: data,
	}
}

// PushUpdate 推送通知状态更新给指定用户。
func (h *Hub) PushUpdate(userID uint64, notifID uint, readAt, dismissedAt, confirmedAt *time.Time) {
	msg := WSMessage{
		Type: "update",
		ID:   string(rune(notifID)),
	}

	if readAt != nil {
		msg.ReadAt = readAt.Format(time.RFC3339)
	}
	if dismissedAt != nil {
		msg.DismissedAt = dismissedAt.Format(time.RFC3339)
	}
	if confirmedAt != nil {
		msg.ConfirmedAt = confirmedAt.Format(time.RFC3339)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("WebSocket: 序列化消息失败: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		UserID:  userID,
		Message: data,
	}
}

// ReadPump 读取客户端消息。
//
// 处理 pong 消息，超时断开连接。
func (c *Client) ReadPump() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("WebSocket: 读取消息失败: %v", err)
		}
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump 写入消息到客户端。
//
// 从发送队列读取消息并写入连接，支持批量发送和心跳检测。
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		if err := recover(); err != nil {
			log.Println("WritePump error:", err)
		}
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
