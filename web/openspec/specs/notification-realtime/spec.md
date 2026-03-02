# Notification Realtime

实时通知推送能力，基于 WebSocket 实现即时通知。

## Requirements

### REQ-001: WebSocket 连接
前端通过 WebSocket 连接接收实时通知。

- 连接地址: `/ws/notifications`
- 认证: 连接时携带 JWT Token
- 支持自动重连

### REQ-002: 连接状态管理
前端需要管理 WebSocket 连接状态。

- 状态: `connecting` | `connected` | `disconnected`
- 显示连接状态指示器（可选）
- 断线时自动重连

### REQ-003: 重连机制
WebSocket 断开后自动重连。

- 使用指数退避策略
- 初始间隔 1 秒，最大重试间隔 30 秒
- **仅在非正常关闭时重连**（状态码 1000、1001、1005 为正常关闭）
- 重连成功后同步未读状态

### REQ-003-1: 防止重复连接
确保 WebSocket 不会因 React 重渲染而重复创建。

- `useEffect` 依赖数组仅包含 `userId`
- 不应包含 `connect`、`disconnect` 等函数引用
- 组件卸载时使用正常关闭状态码 (1000)

### REQ-004: 心跳检测
保持连接活跃，检测断线。

- 客户端每 30 秒发送 ping
- 服务端响应 pong
- 超时 60 秒未响应则重连

### REQ-005: 消息格式
WebSocket 消息使用 JSON 格式。

**新通知消息:**
```json
{
  "type": "new",
  "notification": {
    "id": "123",
    "type": "alert",
    "title": "CPU 使用率超过 90%",
    "severity": "critical",
    ...
  }
}
```

**状态更新消息:**
```json
{
  "type": "update",
  "id": "123",
  "read_at": "2024-01-15T10:30:00Z"
}
```

### REQ-006: 多标签页同步
同一用户多个标签页间同步通知状态。

- 一个标签页标记已读，其他标签页同步更新
- 使用 BroadcastChannel API 实现

### REQ-007: 降级策略
WebSocket 不可用时降级为轮询。

- 检测 WebSocket 连接失败
- 自动切换到 30 秒轮询
- WebSocket 恢复后切回实时推送

## Frontend Implementation

```typescript
interface NotificationContextValue {
  // 状态
  notifications: UserNotification[];
  unreadCount: number;
  wsConnected: boolean;
  wsStatus: 'connecting' | 'connected' | 'disconnected';

  // 操作
  connect: () => void;
  disconnect: () => void;
  markAsRead: (id: string) => Promise<void>;
  dismiss: (id: string) => Promise<void>;
  confirm: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
}
```

## Backend Implementation

```go
// WebSocket Hub
type NotificationHub struct {
    clients    map[string]*Client  // user_id -> Client
    register   chan *Client
    unregister chan *Client
}

// 推送通知
func (h *NotificationHub) PushNotification(userID string, notification *Notification) {
    if client, ok := h.clients[userID]; ok {
        client.Send(notification)
    }
}
```
