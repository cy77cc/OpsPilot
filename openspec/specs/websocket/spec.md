# WebSocket Realtime

实时通信能力规范，基于 WebSocket 实现服务端推送。

## Requirements

### REQ-001: WebSocket 连接管理

前端通过 WebSocket 连接接收实时消息。

- 连接地址: `/ws/notifications`
- 开发环境通过 Vite proxy 代理到后端
- 生产环境直接连接

### REQ-002: 连接状态

前端需要管理 WebSocket 连接状态。

```typescript
type WSConnectionStatus = 'connecting' | 'connected' | 'disconnected';
```

### REQ-003: 重连机制

WebSocket 断开后自动重连，使用指数退避策略。

- 初始重连间隔: 1 秒
- 最大重连间隔: 30 秒
- 仅在非正常关闭时重连（状态码 1000、1001、1005 为正常关闭）

### REQ-004: 防止重复连接

确保 WebSocket 不会因为 React 重渲染而重复创建。

- `useEffect` 依赖仅包含 `userId`
- 组件卸载时正常关闭连接（状态码 1000）

### REQ-005: 降级策略

WebSocket 不可用时自动降级为 HTTP 轮询。

- 轮询间隔: 30 秒
- WebSocket 恢复后停止轮询

## Vite Proxy Configuration

开发环境需要配置 WebSocket 代理：

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://127.0.0.1:8080',
      changeOrigin: true,
    },
    '/ws': {
      target: 'ws://127.0.0.1:8080',
      ws: true,
    },
  },
}
```

## Message Format

WebSocket 消息使用 JSON 格式。

### 新通知消息

```json
{
  "type": "new",
  "notification": {
    "id": 123,
    "user_id": 1,
    "notification": {
      "title": "CPU 使用率超过 90%",
      "type": "alert",
      "severity": "critical"
    }
  }
}
```

### 状态更新消息

```json
{
  "type": "update",
  "id": "123",
  "read_at": "2024-01-15T10:30:00Z"
}
```

## Backend Implementation

后端 WebSocket 处理：

```go
// internal/websocket/handler.go
func HandleWebSocket(c *gin.Context) {
    userID := c.Query("user_id")
    conn, _ := upgrader.Upgrade(c.Writer, c.Request, nil)

    client := &Client{
        UserID: userID,
        Conn:   conn,
        Send:   make(chan []byte, 256),
        Hub:    GetHub(),
    }

    client.Hub.Register(client)
    go client.WritePump()
    go client.ReadPump()
}
```

心跳机制：
- 服务端每 30 秒发送 Ping
- 客户端需要响应 Pong
- 60 秒无响应则断开连接

## Frontend Implementation

```typescript
// hooks/useNotificationWebSocket.ts
const { status, connect, disconnect, send } = useNotificationWebSocket({
  userId: 1,
  onMessage: (msg) => console.log(msg),
  onConnect: () => console.log('connected'),
  onDisconnect: () => console.log('disconnected'),
});
```

## Common Issues

### 频繁重连

**问题**: WebSocket 每隔几百毫秒就断开重连。

**原因**:
1. `useEffect` 依赖数组包含会变化的函数引用
2. 每次重渲染都创建新连接

**解决方案**:
```typescript
// 正确: 只依赖 userId
useEffect(() => {
  if (userId) connect();
  return () => disconnect(1000); // 正常关闭
}, [userId]);

// 错误: 依赖会导致重复执行
useEffect(() => {
  connect();
}, [userId, connect, disconnect]); // connect/disconnect 会导致循环
```

### 开发环境连接失败

**问题**: 开发环境 WebSocket 连接失败。

**解决方案**: 在 `vite.config.ts` 中配置 `/ws` 代理。
