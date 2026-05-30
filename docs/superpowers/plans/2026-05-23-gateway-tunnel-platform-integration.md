# Gateway Tunnel 平台侧对接 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 OpsPilot 平台侧实现 Gateway Tunnel 功能，支持通过跳板机管理内网主机（隧道模式 + 代理模式）。

**Architecture:** 新建 `internal/modules/gateway/` 模块，包含 RouteTable（内存+DB 双写路由表）和 TunnelManager（隧道会话管理）。扩展 proto 定义新增网关消息类型，修改 server.go 的 handleAgentMessage 增加网关消息分支，修改 exec_dispatch.go 增加路由逻辑。前端在主机管理页面增加跳板机选择和网关状态展示。

**Tech Stack:** Go, Protobuf, GORM, gRPC, React, TypeScript, Ant Design

---

## 文件结构

### 新建文件
- `internal/modules/gateway/model/route.go` — HostRoute 模型
- `internal/modules/gateway/model/tunnel.go` — TunnelSession 模型
- `internal/modules/gateway/logic/route_table.go` — 路由表（内存缓存 + DB 持久化）
- `internal/modules/gateway/logic/tunnel_manager.go` — 隧道会话管理
- `internal/modules/gateway/logic/message_handler.go` — 网关消息处理函数
- `internal/modules/gateway/handler/gateway.go` — HTTP handler（获取跳板机列表）

### 修改文件
- `proto/agent.proto` — 新增消息类型 + 扩展 oneof
- `proto/agent.pb.go` — 重新生成
- `proto/agent_grpc.pb.go` — 重新生成
- `internal/modules/host/model/node.go` — Node 新增 JumpHostID、GatewayMode 字段
- `internal/modules/opsagent/logic/server.go` — handleAgentMessage 新增网关分支
- `internal/modules/opsagent/logic/exec_dispatch.go` — ExecuteCommand 增加路由逻辑
- `internal/svc/app_context.go` — 新增 RouteTable、TunnelManager
- `internal/bootstrap/modules.go` — 注册 gateway 路由
- `internal/modules/host/api/routes.go` — 新增 /hosts/gateways 路由
- `web/src/api/modules/hosts.ts` — Host/HostCreateParams 类型扩展 + getGatewayHosts
- `web/src/pages/Hosts/HostOnboardingPage.tsx` — 跳板机选择 + 连接模式
- `web/src/pages/Hosts/HostListPage.tsx` — 连接方式列
- `web/src/pages/Hosts/HostDetailPage.tsx` — 网关信息展示

---

### Task 1: Proto 消息定义扩展

**Files:**
- Modify: `proto/agent.proto`
- Regenerate: `proto/agent.pb.go`, `proto/agent_grpc.pb.go`

- [ ] **Step 1: 编辑 agent.proto — 新增消息类型**

在 `proto/agent.proto` 末尾（`message Ack` 之后）追加以下消息定义：

```proto
// --- Gateway Tunnel Messages ---

message TunnelOpen {
  string tunnel_id = 1;
  string agent_id = 2;
  string hostname = 3;
  string ip = 4;
  repeated string capabilities = 5;
}

message TunnelData {
  string tunnel_id = 1;
  bytes payload = 2;
}

message TunnelClose {
  string tunnel_id = 1;
  string reason = 2;
}

message ProxyHostRegister {
  string host_id = 1;
  string hostname = 2;
  string ip = 3;
  repeated string capabilities = 4;
}

message ProxyCommandRequest {
  string host_id = 1;
  string command = 2;
  repeated string args = 3;
  int32 timeout_seconds = 4;
}

message ProxyCommandResponse {
  string host_id = 1;
  string command = 2;
  int32 exit_code = 3;
  bytes stdout = 4;
  bytes stderr = 5;
  int64 duration_ms = 6;
  bool timed_out = 7;
}

message ProxyMetricBatch {
  string host_id = 1;
  repeated Metric metrics = 2;
}

message HealthCheckRequest {
  string ref_id = 1;
  int64 timestamp_ms = 2;
}

message HealthCheckResult {
  string ref_id = 1;
  bool healthy = 2;
  string message = 3;
  map<string, string> details = 4;
}
```

- [ ] **Step 2: 编辑 agent.proto — 扩展 AgentMessage oneof**

将现有 `AgentMessage` 的 oneof 替换为：

```proto
message AgentMessage {
  oneof payload {
    AgentRegistration registration = 1;
    Heartbeat heartbeat = 2;
    MetricBatch metrics = 3;
    ExecOutput exec_output = 4;
    ExecResult exec_result = 5;
    Ack ack = 6;
    HealthCheckResult health_check_result = 7;

    // Gateway
    TunnelOpen tunnel_open = 8;
    TunnelData tunnel_data = 9;
    TunnelClose tunnel_close = 10;
    ProxyHostRegister proxy_register = 11;
    ProxyCommandResponse proxy_response = 12;
    ProxyMetricBatch proxy_metrics = 13;
  }
}
```

- [ ] **Step 3: 编辑 agent.proto — 扩展 PlatformMessage oneof**

将现有 `PlatformMessage` 的 oneof 替换为：

```proto
message PlatformMessage {
  oneof payload {
    ExecuteCommand exec_command = 1;
    ExecuteScript exec_script = 2;
    CancelJob cancel_job = 3;
    ConfigUpdate config_update = 4;
    Ack ack = 5;
    HealthCheckRequest health_check = 6;

    // Gateway
    TunnelData tunnel_data = 7;
    TunnelClose tunnel_close = 8;
    ProxyCommandRequest proxy_command = 9;
  }
}
```

- [ ] **Step 4: 生成 Go 代码**

```bash
cd /root/project/OpsPilot
protoc --go_out=. --go-grpc_out=. proto/agent.proto
```

验证 `proto/agent.pb.go` 中包含 `TunnelOpen`、`TunnelData` 等类型，`proto/agent_grpc.pb.go` 无变化（service 定义未改）。

- [ ] **Step 5: 编译验证**

```bash
go build ./...
```

---

### Task 2: 数据库模型 — Node 扩展

**Files:**
- Modify: `internal/modules/host/model/node.go`

- [ ] **Step 1: Node 结构体新增字段**

在 `Node` 结构体中，`ParentHostID` 字段之后新增：

```go
JumpHostID  *NodeID  `gorm:"column:jump_host_id;index" json:"jump_host_id,omitempty"` // 跳板机 Node ID，nil 表示直连
GatewayMode string   `gorm:"column:gateway_mode;size:16" json:"gateway_mode,omitempty"` // 连接模式: tunnel/proxy/auto
```

- [ ] **Step 2: 编译验证**

```bash
go build ./internal/modules/host/...
```

---

### Task 3: 数据库模型 — Gateway 模块

**Files:**
- Create: `internal/modules/gateway/model/route.go`
- Create: `internal/modules/gateway/model/tunnel.go`

- [ ] **Step 1: 创建 HostRoute 模型**

```go
package model

import "time"

// HostRoute 是主机路由表模型，记录主机的连接路由信息。
//
// 表名: host_routes
type HostRoute struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	HostID    uint64    `gorm:"column:host_id;uniqueIndex" json:"host_id"`
	Direct    bool      `gorm:"column:direct;default:true" json:"direct"`
	GatewayID uint64    `gorm:"column:gateway_id;index" json:"gateway_id"`
	TunnelID  string    `gorm:"column:tunnel_id;size:64" json:"tunnel_id"`
	Mode      string    `gorm:"column:mode;size:16" json:"mode"` // tunnel|proxy
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (HostRoute) TableName() string { return "host_routes" }
```

- [ ] **Step 2: 创建 TunnelSession 模型**

```go
package model

import "time"

// TunnelSession 是隧道会话表模型，记录活跃隧道信息。
//
// 表名: tunnel_sessions
type TunnelSession struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	TunnelID  string    `gorm:"column:tunnel_id;uniqueIndex;size:64" json:"tunnel_id"`
	GatewayID uint64    `gorm:"column:gateway_id;index" json:"gateway_id"`
	HostID    uint64    `gorm:"column:host_id;index" json:"host_id"`
	AgentID   string    `gorm:"column:agent_id;size:64" json:"agent_id"`
	Status    string    `gorm:"column:status;size:16;default:active" json:"status"` // active|closed
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TunnelSession) TableName() string { return "tunnel_sessions" }
```

- [ ] **Step 3: 编译验证**

```bash
go build ./internal/modules/gateway/...
```

---

### Task 4: Gateway 模块 — RouteTable

**Files:**
- Create: `internal/modules/gateway/logic/route_table.go`

- [ ] **Step 1: 实现 RouteTable**

```go
package logic

import (
	"fmt"
	"sync"

	gatewaymodel "github.com/cy77cc/OpsPilot/internal/modules/gateway/model"
	"gorm.io/gorm"
)

// RouteTable 管理主机路由，内存缓存 + DB 持久化。
type RouteTable struct {
	cache sync.Map // hostID(uint64) -> *gatewaymodel.HostRoute
	db    *gorm.DB
}

// NewRouteTable 创建路由表实例。
func NewRouteTable(db *gorm.DB) *RouteTable {
	return &RouteTable{db: db}
}

// LoadFromDB 从数据库加载所有路由到内存缓存。
func (rt *RouteTable) LoadFromDB() error {
	if rt.db == nil {
		return fmt.Errorf("db is nil")
	}
	var routes []gatewaymodel.HostRoute
	if err := rt.db.Find(&routes).Error; err != nil {
		return fmt.Errorf("load host routes: %w", err)
	}
	for i := range routes {
		rt.cache.Store(routes[i].HostID, &routes[i])
	}
	return nil
}

// Get 从缓存获取路由，返回 nil 表示未找到。
func (rt *RouteTable) Get(hostID uint64) *gatewaymodel.HostRoute {
	val, ok := rt.cache.Load(hostID)
	if !ok {
		return nil
	}
	route, _ := val.(*gatewaymodel.HostRoute)
	return route
}

// Set 写入路由到 DB 并更新缓存。
func (rt *RouteTable) Set(route gatewaymodel.HostRoute) error {
	if rt.db == nil {
		return fmt.Errorf("db is nil")
	}
	// Upsert by host_id
	err := rt.db.Where("host_id = ?", route.HostID).
		Assign(route).
		FirstOrCreate(&route).Error
	if err != nil {
		return fmt.Errorf("upsert host route: %w", err)
	}
	rt.cache.Store(route.HostID, &route)
	return nil
}

// Delete 从 DB 和缓存中删除路由。
func (rt *RouteTable) Delete(hostID uint64) error {
	if rt.db != nil {
		if err := rt.db.Where("host_id = ?", hostID).Delete(&gatewaymodel.HostRoute{}).Error; err != nil {
			return fmt.Errorf("delete host route: %w", err)
		}
	}
	rt.cache.Delete(hostID)
	return nil
}

// UpdateTunnelID 更新指定主机的隧道 ID。
func (rt *RouteTable) UpdateTunnelID(hostID uint64, tunnelID string) error {
	if rt.db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := rt.db.Model(&gatewaymodel.HostRoute{}).
		Where("host_id = ?", hostID).
		Update("tunnel_id", tunnelID).Error; err != nil {
		return fmt.Errorf("update tunnel id: %w", err)
	}
	if val, ok := rt.cache.Load(hostID); ok {
		route := val.(*gatewaymodel.HostRoute)
		route.TunnelID = tunnelID
	}
	return nil
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./internal/modules/gateway/...
```

---

### Task 5: Gateway 模块 — TunnelManager

**Files:**
- Create: `internal/modules/gateway/logic/tunnel_manager.go`

- [ ] **Step 1: 实现 TunnelManager**

```go
package logic

import (
	"fmt"
	"sync"

	pb "github.com/cy77cc/OpsPilot/proto"
)

// TunnelSession 是内存中的隧道会话。
type TunnelSession struct {
	TunnelID  string
	GatewayID uint64
	HostID    uint64
	AgentID   string
	Stream    pb.AgentService_ConnectServer
}

// TunnelManager 管理活跃隧道会话。
type TunnelManager struct {
	sessions map[string]*TunnelSession // tunnelID -> session
	mu       sync.RWMutex
}

// NewTunnelManager 创建隧道管理器。
func NewTunnelManager() *TunnelManager {
	return &TunnelManager{
		sessions: make(map[string]*TunnelSession),
	}
}

// Open 注册新的隧道会话。
func (tm *TunnelManager) Open(tunnelID, agentID string, gatewayID, hostID uint64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.sessions[tunnelID] = &TunnelSession{
		TunnelID:  tunnelID,
		GatewayID: gatewayID,
		HostID:    hostID,
		AgentID:   agentID,
	}
}

// Close 关闭并移除隧道会话。
func (tm *TunnelManager) Close(tunnelID, reason string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.sessions, tunnelID)
}

// Get 获取隧道会话。
func (tm *TunnelManager) Get(tunnelID string) (*TunnelSession, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	s, ok := tm.sessions[tunnelID]
	return s, ok
}

// SetStream 设置隧道的 gateway 流。
func (tm *TunnelManager) SetStream(tunnelID string, stream pb.AgentService_ConnectServer) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	s, ok := tm.sessions[tunnelID]
	if !ok {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}
	s.Stream = stream
	return nil
}

// GetStream 获取隧道的 gateway 流。
func (tm *TunnelManager) GetStream(tunnelID string) (pb.AgentService_ConnectServer, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	s, ok := tm.sessions[tunnelID]
	if !ok || s.Stream == nil {
		return nil, false
	}
	return s.Stream, true
}

// Count 返回活跃隧道数。
func (tm *TunnelManager) Count() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.sessions)
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./internal/modules/gateway/...
```

---

### Task 6: Gateway 模块 — 消息处理器

**Files:**
- Create: `internal/modules/gateway/logic/message_handler.go`

- [ ] **Step 1: 实现消息处理函数**

```go
package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	gatewaymodel "github.com/cy77cc/OpsPilot/internal/modules/gateway/model"
	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	pb "github.com/cy77cc/OpsPilot/proto"
	"gorm.io/gorm"
)

// HandleTunnelOpen 处理隧道建立请求（B → 平台）。
func HandleTunnelOpen(ctx context.Context, svcCtx *svc.ServiceContext, gatewayID uint64, msg *pb.TunnelOpen) error {
	if msg == nil {
		return nil
	}
	routeTable := svcCtx.RouteTable
	tunnelMgr := svcCtx.TunnelManager

	tunnelID := msg.GetTunnelId()
	agentID := msg.GetAgentId()

	// 查找该 agent 对应的主机
	var instance struct {
		HostID uint64
	}
	err := svcCtx.DB.WithContext(ctx).
		Table("host_plugin_instances").
		Select("host_id").
		Where("agent_id = ? AND install_status = ?", agentID, "succeeded").
		First(&instance).Error
	if err != nil {
		return fmt.Errorf("find host for agent %s: %w", agentID, err)
	}

	// 注册隧道会话
	tunnelMgr.Open(tunnelID, agentID, gatewayID, instance.HostID)

	// 更新路由表绑定 tunnel_id
	if err := routeTable.UpdateTunnelID(instance.HostID, tunnelID); err != nil {
		return fmt.Errorf("update tunnel id: %w", err)
	}

	// 更新主机状态为在线
	now := time.Now()
	svcCtx.DB.WithContext(ctx).
		Model(&hostmodel.Node{}).
		Where("id = ?", instance.HostID).
		Updates(map[string]any{
			"status":       "active",
			"health_state": "healthy",
			"last_check_at": now,
		})

	return nil
}

// HandleTunnelData 处理隧道数据转发（B → 平台 → 解析为 AgentMessage）。
func HandleTunnelData(ctx context.Context, svcCtx *svc.ServiceContext, gatewayID uint64, msg *pb.TunnelData) error {
	if msg == nil {
		return nil
	}
	tunnelID := msg.GetTunnelId()
	tunnelMgr := svcCtx.TunnelManager

	session, ok := tunnelMgr.Get(tunnelID)
	if !ok {
		return fmt.Errorf("tunnel %s not found", tunnelID)
	}

	// 反序列化 payload 为 AgentMessage
	var agentMsg pb.AgentMessage
	if err := proto.Unmarshal(msg.GetPayload(), &agentMsg); err != nil {
		return fmt.Errorf("unmarshal tunnel payload: %w", err)
	}

	// 按正常 Agent 消息处理，但 hostID 使用隧道绑定的 hostID
	// 这里需要将消息转发给现有的消息处理逻辑
	_ = session
	return handleProxiedAgentMessage(ctx, svcCtx, session.HostID, &agentMsg)
}

// HandleTunnelClose 处理隧道关闭（B → 平台）。
func HandleTunnelClose(ctx context.Context, svcCtx *svc.ServiceContext, gatewayID uint64, msg *pb.TunnelClose) error {
	if msg == nil {
		return nil
	}
	tunnelID := msg.GetTunnelId()
	tunnelMgr := svcCtx.TunnelManager

	session, ok := tunnelMgr.Get(tunnelID)
	if !ok {
		return nil // 已关闭，忽略
	}

	// 关闭隧道
	tunnelMgr.Close(tunnelID, msg.GetReason())

	// 标记主机离线
	svcCtx.DB.WithContext(ctx).
		Model(&hostmodel.Node{}).
		Where("id = ?", session.HostID).
		Updates(map[string]any{
			"status":       "inactive",
			"health_state": "unknown",
		})

	// 清理 DB 中的隧道会话记录
	svcCtx.DB.WithContext(ctx).
		Where("tunnel_id = ?", tunnelID).
		Delete(&gatewaymodel.TunnelSession{})

	return nil
}

// HandleProxyRegister 处理代理主机注册（B → 平台）。
func HandleProxyRegister(ctx context.Context, svcCtx *svc.ServiceContext, gatewayID uint64, msg *pb.ProxyHostRegister) error {
	if msg == nil {
		return nil
	}
	// 更新代理主机状态为在线
	hostID, err := strconv.ParseUint(msg.GetHostId(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host_id: %w", err)
	}

	now := time.Now()
	return svcCtx.DB.WithContext(ctx).
		Model(&hostmodel.Node{}).
		Where("id = ?", hostID).
		Updates(map[string]any{
			"status":       "active",
			"health_state": "healthy",
			"last_check_at": now,
		}).Error
}

// HandleProxyResponse 处理代理命令执行结果（B → 平台）。
// ProxyCommandResponse 没有 task_id，使用 host_id+command 作为复合 key 匹配 waiter。
func HandleProxyResponse(ctx context.Context, svcCtx *svc.ServiceContext, gatewayID uint64, msg *pb.ProxyCommandResponse) error {
	if msg == nil {
		return nil
	}
	// 使用与 executeViaGateway 中相同的复合 key 格式
	waiterKey := fmt.Sprintf("proxy-%s-%s", msg.GetHostId(), msg.GetCommand())
	result := &pb.ExecResult{
		TaskId:     waiterKey,
		ExitCode:   msg.GetExitCode(),
		DurationMs: msg.GetDurationMs(),
		TimedOut:   msg.GetTimedOut(),
	}
	handleExecResult(result)
	return nil
}

// HandleProxyMetrics 处理代理主机指标数据（B → 平台）。
func HandleProxyMetrics(ctx context.Context, svcCtx *svc.ServiceContext, gatewayID uint64, msg *pb.ProxyMetricBatch) error {
	if msg == nil {
		return nil
	}
	hostID, err := strconv.ParseUint(msg.GetHostId(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid host_id: %w", err)
	}

	batch := &pb.MetricBatch{Metrics: msg.GetMetrics()}
	// 复用现有的指标持久化逻辑
	snapshot := NormalizeMetricBatch(hostID, batch)
	if err := svcCtx.DB.WithContext(ctx).Create(snapshot).Error; err != nil {
		return fmt.Errorf("persist proxy metrics: %w", err)
	}
	return svcCtx.DB.WithContext(ctx).
		Model(&hostmodel.Node{}).
		Where("id = ?", hostID).
		Updates(map[string]any{
			"health_state":  snapshot.State,
			"last_check_at": snapshot.CheckedAt,
		}).Error
}

// handleProxiedAgentMessage 处理通过隧道转发的 Agent 消息。
func handleProxiedAgentMessage(ctx context.Context, svcCtx *svc.ServiceContext, hostID uint64, msg *pb.AgentMessage) error {
	switch payload := msg.GetPayload().(type) {
	case *pb.AgentMessage_Registration:
		// 隧道模式下的注册：更新实例状态
		agentID := payload.Registration.GetAgentId()
		now := time.Now()
		return svcCtx.DB.WithContext(ctx).
			Model(&hostpluginmodel.HostPluginInstance{}).
			Where("agent_id = ? AND install_status = ?", agentID, "succeeded").
			Updates(map[string]any{
				"runtime_status": "online",
				"last_seen_at":   &now,
			}).Error
	case *pb.AgentMessage_Heartbeat:
		agentID := payload.Heartbeat.GetAgentId()
		now := time.Now()
		return svcCtx.DB.WithContext(ctx).
			Model(&hostpluginmodel.HostPluginInstance{}).
			Where("agent_id = ?", agentID).
			Updates(map[string]any{
				"runtime_status": "online",
				"health_status":  "healthy",
				"last_seen_at":   &now,
			}).Error
	case *pb.AgentMessage_Metrics:
		batch := payload.Metrics
		snapshot := NormalizeMetricBatch(hostID, batch)
		if err := svcCtx.DB.WithContext(ctx).Create(snapshot).Error; err != nil {
			return err
		}
		return svcCtx.DB.WithContext(ctx).
			Model(&hostmodel.Node{}).
			Where("id = ?", hostID).
			Updates(map[string]any{
				"health_state":  snapshot.State,
				"last_check_at": snapshot.CheckedAt,
			}).Error
	case *pb.AgentMessage_ExecOutput:
		handleExecOutput(payload.ExecOutput)
		return nil
	case *pb.AgentMessage_ExecResult:
		handleExecResult(payload.ExecResult)
		return nil
	case *pb.AgentMessage_Ack:
		// 忽略隧道中的 ack
		return nil
	default:
		return nil
	}
}
```

注意：此文件需要 import `time`、`google.golang.org/protobuf/proto` 和 `hostpluginmodel`。在实现时需要补充完整 import。

- [ ] **Step 2: 编译验证**

```bash
go build ./internal/modules/gateway/...
```

---

### Task 7: 集成到 server.go — 消息处理扩展

**Files:**
- Modify: `internal/modules/opsagent/logic/server.go`

- [ ] **Step 1: 在 handleAgentMessage 中新增网关消息分支**

在 `handleAgentMessage` 函数的 `switch payload` 中，在 `case *pb.AgentMessage_Ack:` 之后、`default:` 之前新增：

```go
case *pb.AgentMessage_TunnelOpen:
    // 从 stream 上下文获取 gateway 的 hostID
    gatewayHostID := s.getGatewayHostID(stream)
    return gateway.HandleTunnelOpen(ctx, s.svcCtx, gatewayHostID, payload.TunnelOpen)
case *pb.AgentMessage_TunnelData:
    gatewayHostID := s.getGatewayHostID(stream)
    return gateway.HandleTunnelData(ctx, s.svcCtx, gatewayHostID, payload.TunnelData)
case *pb.AgentMessage_TunnelClose:
    gatewayHostID := s.getGatewayHostID(stream)
    return gateway.HandleTunnelClose(ctx, s.svcCtx, gatewayHostID, payload.TunnelClose)
case *pb.AgentMessage_ProxyRegister:
    gatewayHostID := s.getGatewayHostID(stream)
    return gateway.HandleProxyRegister(ctx, s.svcCtx, gatewayHostID, payload.ProxyRegister)
case *pb.AgentMessage_ProxyResponse:
    gatewayHostID := s.getGatewayHostID(stream)
    return gateway.HandleProxyResponse(ctx, s.svcCtx, gatewayHostID, payload.ProxyResponse)
case *pb.AgentMessage_ProxyMetrics:
    gatewayHostID := s.getGatewayHostID(stream)
    return gateway.HandleProxyMetrics(ctx, s.svcCtx, gatewayHostID, payload.ProxyMetrics)
```

- [ ] **Step 2: 添加 getGatewayHostID 辅助方法**

在 `server.go` 中新增：

```go
// getGatewayHostID 从 stream 获取 gateway agent 对应的 hostID。
func (s *Server) getGatewayHostID(stream grpc.BidiStreamingServer[pb.AgentMessage, pb.PlatformMessage]) uint64 {
    agentID, err := s.registry.AgentIDByStream(stream)
    if err != nil {
        return 0
    }
    session, ok := s.registry.GetByAgent(agentID)
    if !ok || session == nil {
        return 0
    }
    return session.HostID
}
```

- [ ] **Step 3: 添加 gateway import**

在 `server.go` 的 import 中新增：

```go
gateway "github.com/cy77cc/OpsPilot/internal/modules/gateway/logic"
```

- [ ] **Step 4: 编译验证**

```bash
go build ./internal/modules/opsagent/...
```

---

### Task 8: 集成到 exec_dispatch.go — 发送路由

**Files:**
- Modify: `internal/modules/opsagent/logic/exec_dispatch.go`

- [ ] **Step 1: 修改 ExecuteCommand 增加路由逻辑**

在 `ExecuteCommand` 方法中，`session, taskID, waiter, err := d.prepareDispatch(instance)` 之后、构造 `msg` 之前，插入路由检查：

```go
// 检查是否通过网关路由
if d.svcCtx != nil && d.svcCtx.RouteTable != nil {
    route := d.svcCtx.RouteTable.Get(instance.HostID)
    if route != nil && !route.Direct {
        return d.executeViaGateway(ctx, instance, command, route)
    }
}
```

- [ ] **Step 2: 新增 executeViaGateway 方法**

在 `exec_dispatch.go` 中新增：

```go
func (d *Dispatcher) executeViaGateway(ctx context.Context, instance *hostpluginmodel.HostPluginInstance, command string, route *gatewaymodel.HostRoute) (*DispatchResult, error) {
    if route.Mode == "tunnel" {
        // 隧道模式：通过 tunnel manager 转发
        tunnelMgr := d.svcCtx.TunnelManager
        if tunnelMgr == nil {
            return nil, fmt.Errorf("tunnel manager unavailable")
        }
        stream, ok := tunnelMgr.GetStream(route.TunnelID)
        if !ok {
            return nil, fmt.Errorf("tunnel %s stream not found", route.TunnelID)
        }

        taskID := fmt.Sprintf("opsagent-dispatch-%d", dispatchSeqID.Add(1))
        waiter := &execWaiter{resultCh: make(chan *pb.ExecResult, 1)}
        execWaiters.Store(taskID, waiter)
        defer execWaiters.Delete(taskID)

        // 构造原始 PlatformMessage
        innerMsg := &pb.PlatformMessage{
            Payload: &pb.PlatformMessage_ExecCommand{
                ExecCommand: &pb.ExecuteCommand{
                    TaskId:         taskID,
                    Command:        "sh",
                    Args:           []string{"-c", command},
                    TimeoutSeconds: resolveTimeoutSeconds(ctx, 30),
                },
            },
        }
        payload, _ := proto.Marshal(innerMsg)

        // 包装为 TunnelData 发送
        tunnelMsg := &pb.PlatformMessage{
            Payload: &pb.PlatformMessage_TunnelData{
                TunnelData: &pb.TunnelData{
                    TunnelId: route.TunnelID,
                    Payload:  payload,
                },
            },
        }
        if err := stream.Send(tunnelMsg); err != nil {
            return nil, fmt.Errorf("send via tunnel: %w", err)
        }
        return waitForDispatchResult(ctx, waiter)
    }

    // 代理模式：发送 ProxyCommandRequest
    gatewaySession, ok := d.registry.GetByHostID(route.GatewayID)
    if !ok || gatewaySession == nil {
        return nil, fmt.Errorf("gateway session not found for host %d", route.GatewayID)
    }

    taskID := fmt.Sprintf("proxy-%d-%s", instance.HostID, command) // 与 HandleProxyResponse 的 waiterKey 格式一致
    waiter := &execWaiter{resultCh: make(chan *pb.ExecResult, 1)}
    execWaiters.Store(taskID, waiter)
    defer execWaiters.Delete(taskID)

    proxyMsg := &pb.PlatformMessage{
        Payload: &pb.PlatformMessage_ProxyCommand{
            ProxyCommand: &pb.ProxyCommandRequest{
                HostId:         fmt.Sprintf("%d", instance.HostID),
                Command:        command,
                TimeoutSeconds: resolveTimeoutSeconds(ctx, 30),
            },
        },
    }
    if err := sessionSend(gatewaySession, proxyMsg); err != nil {
        return nil, err
    }
    return waitForDispatchResult(ctx, waiter)
}
```

- [ ] **Step 3: 添加 import**

在 `exec_dispatch.go` 的 import 中新增：

```go
gatewaymodel "github.com/cy77cc/OpsPilot/internal/modules/gateway/model"
"google.golang.org/protobuf/proto"
```

- [ ] **Step 4: 编译验证**

```bash
go build ./internal/modules/opsagent/...
```

---

### Task 9: ServiceContext 扩展

**Files:**
- Modify: `internal/svc/app_context.go`

- [ ] **Step 1: 添加字段到 ServiceContext**

在 `ServiceContext` 结构体中新增：

```go
RouteTable    *gatewaylogic.RouteTable
TunnelManager *gatewaylogic.TunnelManager
```

- [ ] **Step 2: 在 NewServiceContext 中初始化**

在 `NewServiceContext` 的 return 语句中新增初始化：

```go
RouteTable:    gatewaylogic.NewRouteTable(db),
TunnelManager: gatewaylogic.NewTunnelManager(),
```

并在 return 之前调用：

```go
rt := gatewaylogic.NewRouteTable(db)
if err := rt.LoadFromDB(); err != nil {
    log.Printf("warning: load host routes: %v", err)
}
```

- [ ] **Step 3: 添加 import**

```go
gatewaylogic "github.com/cy77cc/OpsPilot/internal/modules/gateway/logic"
```

- [ ] **Step 4: 编译验证**

```bash
go build ./internal/svc/...
```

---

### Task 10: HTTP API — 跳板机列表

**Files:**
- Create: `internal/modules/gateway/handler/gateway.go`
- Modify: `internal/modules/host/api/routes.go`

- [ ] **Step 1: 创建 gateway handler**

```go
package handler

import (
	"net/http"

	"github.com/cy77cc/OpsPilot/internal/svc"
	"github.com/gin-gonic/gin"
)

type GatewayHandler struct {
	svcCtx *svc.ServiceContext
}

func NewGatewayHandler(svcCtx *svc.ServiceContext) *GatewayHandler {
	return &GatewayHandler{svcCtx: svcCtx}
}

// ListGateways 返回可用的跳板机列表。
// GET /api/v1/hosts/gateways
func (h *GatewayHandler) ListGateways(c *gin.Context) {
	type gatewayInfo struct {
		ID            uint64  `json:"id"`
		Name          string  `json:"name"`
		Hostname      string  `json:"hostname"`
		IP            string  `json:"ip"`
		ActiveTunnels int     `json:"active_tunnels"`
		MaxTunnels    int     `json:"max_tunnels"`
	}

	var results []gatewayInfo
	db := h.svcCtx.DB

	// 查询有 gateway 能力且在线的 opsagent 实例对应的主机
	err := db.Raw(`
		SELECT n.id, n.name, n.hostname, n.ip
		FROM nodes n
		JOIN host_plugin_instances hpi ON hpi.host_id = n.id
		JOIN host_plugins hp ON hp.id = hpi.plugin_id
		WHERE hp.plugin_key = 'opsagent'
		  AND hpi.install_status = 'succeeded'
		  AND hpi.runtime_status = 'online'
		  AND hpi.capabilities_json LIKE '%gateway%'
	`).Scan(&results).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 填充隧道数
	if h.svcCtx.TunnelManager != nil {
		count := h.svcCtx.TunnelManager.Count()
		for i := range results {
			results[i].ActiveTunnels = count
			results[i].MaxTunnels = 100 // 默认值
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}
```

- [ ] **Step 2: 在 routes.go 中注册新路由**

在 `RegisterHostHandlers` 的 `g := v1.Group("/hosts", ...)` 块中，`g.GET("", h.List)` 之前新增：

```go
g.GET("/gateways", handler.NewGatewayHandler(svcCtx).ListGateways)
```

需要在 routes.go 的 import 中确保 `handler` 包被正确引用。由于 gateway handler 在不同包，需要直接引用：

```go
gatewayhandler "github.com/cy77cc/OpsPilot/internal/modules/gateway/handler"
```

路由注册改为：

```go
g.GET("/gateways", gatewayhandler.NewGatewayHandler(svcCtx).ListGateways)
```

- [ ] **Step 3: 编译验证**

```bash
go build ./internal/modules/host/...
go build ./internal/modules/gateway/...
```

---

### Task 11: 前端 API 类型扩展

**Files:**
- Modify: `web/src/api/modules/hosts.ts`

- [ ] **Step 1: 扩展 Host 接口**

在 `Host` 接口中新增字段（在 `pluginInstances` 之后）：

```typescript
jumpHostId?: number;
gatewayMode?: 'tunnel' | 'proxy' | 'auto';
jumpHostName?: string;
```

- [ ] **Step 2: 扩展 HostCreateParams 接口**

在 `HostCreateParams` 接口中新增字段（在 `parentHostId` 之后）：

```typescript
jumpHostId?: number;
gatewayMode?: 'tunnel' | 'proxy' | 'auto';
```

- [ ] **Step 3: 新增 GatewayHostInfo 类型和 getGatewayHosts 方法**

在 `hosts.ts` 中新增：

```typescript
export interface GatewayHostInfo {
  id: number;
  name: string;
  hostname: string;
  ip: string;
  activeTunnels: number;
  maxTunnels: number;
}
```

在 `hostApi` 对象中新增方法：

```typescript
getGatewayHosts: async (): Promise<GatewayHostInfo[]> => {
  const res = await apiService.get<{ data: GatewayHostInfo[] }>('/hosts/gateways');
  return (res as any).data ?? [];
},
```

- [ ] **Step 4: 在 API index 中确认导出**

`web/src/api/index.ts` 中 `Api.hosts` 已包含 `hostApi` 的所有方法，无需额外修改。

---

### Task 12: 前端 — 添加主机页面

**Files:**
- Modify: `web/src/pages/Hosts/HostOnboardingPage.tsx`

- [ ] **Step 1: 扩展 StepThreeForm 接口**

在 `StepThreeForm` 接口中新增：

```typescript
jumpHostId?: number;
gatewayMode?: 'tunnel' | 'proxy' | 'auto';
```

- [ ] **Step 2: 添加 state 和数据加载**

新增 state：

```typescript
const [gatewayHosts, setGatewayHosts] = useState<GatewayHostInfo[]>([]);
const [gatewayHostsLoading, setGatewayHostsLoading] = useState(false);
```

在 `useEffect` 中加载跳板机列表：

```typescript
useEffect(() => {
  setGatewayHostsLoading(true);
  Api.hosts.getGatewayHosts()
    .then(setGatewayHosts)
    .catch(() => {})
    .finally(() => setGatewayHostsLoading(false));
}, []);
```

- [ ] **Step 3: 在第三步表单中添加跳板机选择**

在 Step 3 的 Form 中，`installOpsAgent` 字段之后新增：

```tsx
<GuidedFormItem
  label="跳板机"
  field="jumpHostId"
  guidance="选择一个跳板机来管理无法直连的内网主机。留空表示直连。"
>
  <Select
    allowClear
    placeholder="选择跳板机（可选）"
    loading={gatewayHostsLoading}
    options={gatewayHosts.map(g => ({
      label: `${g.name} (${g.ip})`,
      value: g.id,
    }))}
  />
</GuidedFormItem>

{stepThreeValues?.jumpHostId && (
  <GuidedFormItem
    label="连接模式"
    field="gatewayMode"
    guidance="隧道模式：目标主机有 Agent，通过跳板机转发 gRPC 流量。代理模式：目标主机无 Agent，跳板机通过 SSH 代为执行。"
  >
    <Radio.Group>
      <Radio value="tunnel">隧道模式</Radio>
      <Radio value="proxy">代理模式</Radio>
      <Radio value="auto">自动检测</Radio>
    </Radio.Group>
  </GuidedFormItem>
)}
```

- [ ] **Step 4: 在创建主机请求中传递新字段**

在 `handleCreate` 函数构造请求参数时，新增：

```typescript
jump_host_id: stepThreeValues.jumpHostId || undefined,
gateway_mode: stepThreeValues.gatewayMode || undefined,
```

---

### Task 13: 前端 — 主机列表页面

**Files:**
- Modify: `web/src/pages/Hosts/HostListPage.tsx`

- [ ] **Step 1: 新增连接方式列**

在表格 columns 定义中，添加一列（在 hostname 列之后）：

```typescript
{
  title: '连接方式',
  dataIndex: 'jumpHostId',
  key: 'connection',
  width: 120,
  render: (_: any, record: Host) => {
    if (!record.jumpHostId) {
      return <Tag color="green">直连</Tag>;
    }
    return (
      <Tooltip title={`跳板机: ${record.jumpHostName || record.jumpHostId}`}>
        <Tag color="blue">通过网关</Tag>
      </Tooltip>
    );
  },
},
```

确保 import 了 `Tooltip` from antd。

---

### Task 14: 前端 — 主机详情页面

**Files:**
- Modify: `web/src/pages/Hosts/HostDetailPage.tsx`

- [ ] **Step 1: 在描述信息区新增网关字段**

在 Descriptions 组件中新增 items：

```tsx
{host.jumpHostId && (
  <>
    <Descriptions.Item label="跳板机">{host.jumpHostName || host.jumpHostId}</Descriptions.Item>
    <Descriptions.Item label="连接模式">
      <Tag color="blue">{host.gatewayMode === 'tunnel' ? '隧道模式' : host.gatewayMode === 'proxy' ? '代理模式' : '自动检测'}</Tag>
    </Descriptions.Item>
  </>
)}
```

- [ ] **Step 2: 前端编译验证**

```bash
cd web && npm run build
```

---

### Task 15: 整体编译和集成验证

- [ ] **Step 1: 后端全量编译**

```bash
cd /root/project/OpsPilot
go build ./...
```

- [ ] **Step 2: 前端编译**

```bash
cd /root/project/OpsPilot/web
npm run build
```

- [ ] **Step 3: 运行测试**

```bash
cd /root/project/OpsPilot
go test ./...
```

- [ ] **Step 4: 数据库迁移验证**

确认 GORM AutoMigrate 会自动创建 `host_routes` 和 `tunnel_sessions` 表，并为 `nodes` 表添加 `jump_host_id` 和 `gateway_mode` 列。检查 `internal/core/storage/` 中的迁移逻辑或 `cmd/opspilot/migrate` 命令。

- [ ] **Step 5: 提交代码**

```bash
git add proto/ internal/modules/gateway/ internal/modules/host/model/node.go \
  internal/modules/opsagent/logic/server.go internal/modules/opsagent/logic/exec_dispatch.go \
  internal/svc/app_context.go internal/bootstrap/modules.go internal/modules/host/api/routes.go \
  web/src/api/modules/hosts.ts web/src/pages/Hosts/
git commit -m "feat: gateway tunnel platform integration

- Extend proto with tunnel and proxy message types
- Add gateway module with RouteTable and TunnelManager
- Integrate gateway message handling into gRPC server
- Add routing logic to command dispatch
- Add /hosts/gateways API endpoint
- Update frontend with jump host selection and gateway status display"
```
