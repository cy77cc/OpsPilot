package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/cy77cc/k8s-manage/internal/model"
	"github.com/gorilla/websocket"
)

// WSMessage WebSocket 消息格式
type WSMessage struct {
	Type         string             `json:"type"`
	Notification *UserNotificationWS `json:"notification,omitempty"`
	ID           string             `json:"id,omitempty"`
	ReadAt       string             `json:"read_at,omitempty"`
	DismissedAt  string             `json:"dismissed_at,omitempty"`
	ConfirmedAt  string             `json:"confirmed_at,omitempty"`
}

// UserNotificationWS WebSocket 通知格式
type UserNotificationWS struct {
	ID             uint              `json:"id"`
	UserID         uint64            `json:"user_id"`
	NotificationID uint              `json:"notification_id"`
	ReadAt         *time.Time        `json:"read_at"`
	DismissedAt    *time.Time        `json:"dismissed_at"`
	ConfirmedAt    *time.Time        `json:"confirmed_at"`
	Notification   model.Notification `json:"notification"`
}

// Client WebSocket 客户端
type Client struct {
	UserID uint64
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *Hub
}

// Hub WebSocket 连接中心
type Hub struct {
	clients    map[uint64]map[*Client]bool // userID -> clients
	broadcast  chan *BroadcastMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	UserID  uint64
	Message []byte
}

var hubInstance *Hub
var hubOnce sync.Once

// GetHub 获取 Hub 单例
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

// Run 启动 Hub
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer func ()  {
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

// Register 注册客户端
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// PushNotification 推送新通知给用户
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

// PushUpdate 推送通知状态更新
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

// ReadPump 读取消息
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

// WritePump 写入消息
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
