# Gateway Tunnel 平台侧对接设计

## 概述

对接 OpsAgent 的 Gateway Tunnel 功能，使平台能够通过跳板机管理无法直连的内网主机。Agent 端已全部完成，本次实现平台侧四层变更：Proto 定义、数据库模型、gRPC 消息路由、前端 UI。

## 方案选型

采用独立 Gateway 模块 + DB 路由持久化方案：
- 新建 `internal/modules/gateway/` 模块独立管理路由和隧道
- `host_routes` 表持久化路由，平台重启自动恢复
- 内存缓存 + DB 双写保证性能和持久性

---

## 1. Proto 消息定义

### 1.1 新增消息类型

在 `proto/agent.proto` 中新增以下消息：

```proto
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
```

### 1.2 AgentMessage 扩展

```proto
message AgentMessage {
  oneof payload {
    // 现有 (1-6)
    AgentRegistration registration = 1;
    Heartbeat heartbeat = 2;
    MetricBatch metrics = 3;
    ExecOutput exec_output = 4;
    ExecResult exec_result = 5;
    Ack ack = 6;
    HealthCheckResult health_check_result = 7;

    // Gateway 新增 (8-13)
    TunnelOpen tunnel_open = 8;
    TunnelData tunnel_data = 9;
    TunnelClose tunnel_close = 10;
    ProxyHostRegister proxy_register = 11;
    ProxyCommandResponse proxy_response = 12;
    ProxyMetricBatch proxy_metrics = 13;
  }
}
```

### 1.3 PlatformMessage 扩展

```proto
message PlatformMessage {
  oneof payload {
    // 现有 (1-5)
    ExecuteCommand exec_command = 1;
    ExecuteScript exec_script = 2;
    CancelJob cancel_job = 3;
    ConfigUpdate config_update = 4;
    Ack ack = 5;
    HealthCheckRequest health_check = 6;

    // Gateway 新增 (7-9)
    TunnelData tunnel_data = 7;
    TunnelClose tunnel_close = 8;
    ProxyCommandRequest proxy_command = 9;
  }
}
```

### 1.4 补充缺失消息

当前 proto 缺少 `HealthCheckRequest` 和 `HealthCheckResult`，需一并补充：

```proto
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

### 1.5 代码生成

```bash
protoc --go_out=. --go-grpc_out=. proto/agent.proto
```

---

## 2. 数据库模型

### 2.1 Node 模型扩展

`internal/modules/host/model/node.go` 新增字段：

```go
JumpHostID  *uint64 `gorm:"column:jump_host_id;index" json:"jump_host_id,omitempty"`
GatewayMode string  `gorm:"column:gateway_mode;size:16" json:"gateway_mode,omitempty"`
```

- `JumpHostID`：指向跳板机 Node ID，nil 表示直连
- `GatewayMode`：tunnel / proxy / auto，仅 JumpHostID 非空时有效

GORM AutoMigrate 自动加列。

### 2.2 新建 host_routes 表

`internal/modules/gateway/model/route.go`：

```go
type HostRoute struct {
    ID        uint64    `gorm:"primaryKey" json:"id"`
    HostID    uint64    `gorm:"column:host_id;uniqueIndex" json:"host_id"`
    Direct    bool      `gorm:"column:direct" json:"direct"`
    GatewayID uint64    `gorm:"column:gateway_id;index" json:"gateway_id"`
    TunnelID  string    `gorm:"column:tunnel_id;size:64" json:"tunnel_id"`
    Mode      string    `gorm:"column:mode;size:16" json:"mode"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2.3 新建 tunnel_sessions 表

`internal/modules/gateway/model/tunnel.go`：

```go
type TunnelSession struct {
    ID        uint64    `gorm:"primaryKey" json:"id"`
    TunnelID  string    `gorm:"column:tunnel_id;uniqueIndex;size:64" json:"tunnel_id"`
    GatewayID uint64    `gorm:"column:gateway_id;index" json:"gateway_id"`
    HostID    uint64    `gorm:"column:host_id;index" json:"host_id"`
    AgentID   string    `gorm:"column:agent_id;size:64" json:"agent_id"`
    Status    string    `gorm:"column:status;size:16" json:"status"` // active|closed
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 2.4 主机注册流程变更

**隧道模式添加主机**：
1. 前端传 `jump_host_id` + `gateway_mode=tunnel`
2. 写入 `nodes` 表（该主机有 SSH 凭据，因为 C 有自己的 Agent）
3. 创建 `host_routes` 记录（Direct=false, GatewayID=jump_host_id, Mode=tunnel）
4. 等待 C 的 Agent 通过 B 建立隧道后，路由自动绑定 TunnelID

**代理模式添加主机**：
1. 前端传 `jump_host_id` + `gateway_mode=proxy` + SSH 凭据信息
2. 写入 `nodes` 表（存储 SSH 凭据，供 B 代理执行时使用）
3. 创建 `host_routes` 记录（Direct=false, GatewayID=jump_host_id, Mode=proxy）
4. 平台通过 B 的 gRPC 流发送 `ProxyCommandRequest`，B 用存储的 SSH 凭据连接 C

**获取可用跳板机列表**：
- 新增 API：`GET /api/v1/hosts/gateways`，查询 `host_plugin_instances` 中有 gateway 能力且 runtime_status=online 的主机
- 返回格式：`[{id, name, hostname, ip, active_tunnels, max_tunnels}]`

---

## 3. gRPC 消息路由层

### 3.1 模块结构

```
internal/modules/gateway/
├── model/
│   ├── route.go
│   └── tunnel.go
├── logic/
│   ├── route_table.go
│   ├── tunnel_manager.go
│   ├── proxy_handler.go
│   └── message_handler.go
└── handler/
    └── gateway.go
```

### 3.2 RouteTable

```go
type RouteTable struct {
    cache sync.Map  // hostID -> HostRoute
    db    *gorm.DB
}

func (rt *RouteTable) LoadFromDB()                    // 启动时加载全表到 cache
func (rt *RouteTable) Get(hostID uint64) *HostRoute   // 先查 cache
func (rt *RouteTable) Set(route HostRoute)             // 写 DB + 更新 cache
func (rt *RouteTable) Delete(hostID uint64)            // 删 DB + 删 cache
```

### 3.3 TunnelManager

```go
type TunnelManager struct {
    sessions map[string]*TunnelSession
    mu       sync.RWMutex
}

func (tm *TunnelManager) Open(tunnelID, agentID string, gatewayID, hostID uint64)
func (tm *TunnelManager) Close(tunnelID, reason string)
func (tm *TunnelManager) GetStream(tunnelID string) pb.AgentService_ConnectServer
func (tm *TunnelManager) SetStream(tunnelID string, stream pb.AgentService_ConnectServer)
```

### 3.4 消息处理集成

在 `internal/modules/opsagent/logic/server.go` 的 `handleAgentMessage` 中新增分支：

```go
case *pb.AgentMessage_TunnelOpen:
    return gateway.HandleTunnelOpen(svcCtx, gatewayID, msg)
case *pb.AgentMessage_TunnelData:
    return gateway.HandleTunnelData(svcCtx, gatewayID, msg)
case *pb.AgentMessage_TunnelClose:
    return gateway.HandleTunnelClose(svcCtx, gatewayID, msg)
case *pb.AgentMessage_ProxyRegister:
    return gateway.HandleProxyRegister(svcCtx, gatewayID, msg)
case *pb.AgentMessage_ProxyResponse:
    return gateway.HandleProxyResponse(svcCtx, gatewayID, msg)
case *pb.AgentMessage_ProxyMetrics:
    return gateway.HandleProxyMetrics(svcCtx, gatewayID, msg)
```

### 3.5 消息发送路由

在 `internal/modules/opsagent/logic/exec_dispatch.go` 的 `ExecuteCommand` 中，发送前查路由：

```go
route := svcCtx.RouteTable.Get(hostID)
if route != nil && !route.Direct {
    if route.Mode == "tunnel" {
        // 隧道模式：包装原始 PlatformMessage 为 TunnelData
        payload, _ := proto.Marshal(msg)
        tunnelMsg := &pb.PlatformMessage{
            Payload: &pb.PlatformMessage_TunnelData{
                TunnelData: &pb.TunnelData{
                    TunnelId: route.TunnelID,
                    Payload:  payload,
                },
            },
        }
        return sendToGatewayStream(registry, route.GatewayID, tunnelMsg)
    }
    // 代理模式：构造 ProxyCommandRequest
    proxyMsg := &pb.PlatformMessage{
        Payload: &pb.PlatformMessage_ProxyCommand{
            ProxyCommand: &pb.ProxyCommandRequest{
                HostId:         fmt.Sprintf("%d", hostID),
                Command:        cmd.Command,
                Args:           cmd.Args,
                TimeoutSeconds: cmd.Timeout,
            },
        },
    }
    return sendToGatewayStream(registry, route.GatewayID, proxyMsg)
}
// 直连：原有逻辑
```

**代理模式响应处理**：
- 收到 `ProxyCommandResponse` 时，按 `host_id` 查找对应的 exec waiter
- 将 response 转换为 `ExecResult` 格式投递到 waiter channel
- 收到 `ProxyMetricBatch` 时，调用现有 `persistMetricBatch()` 处理

### 3.6 ServiceContext 扩展

`internal/svc/app_context.go` 新增：

```go
RouteTable    *gateway.RouteTable
TunnelManager *gateway.TunnelManager
```

在 `NewServiceContext()` 中初始化并调用 `RouteTable.LoadFromDB()`。

---

## 4. 前端 UI 变更

### 4.1 添加主机页面

`HostOnboardingPage.tsx` 第三步确认页新增：

- **跳板机下拉**：`Select` 组件，调用 `GET /api/v1/hosts/gateways` 获取可用跳板机列表
- **连接模式**：`Radio.Group`，选择跳板机后显示，选项：隧道模式 / 代理模式 / 自动检测
- 代理模式下额外显示 SSH 凭据配置（复用现有 `SSHCredentialTemplate` 组件）

### 4.2 主机列表

`HostListPage.tsx` 表格新增列：

- **连接方式**：`Tag` 组件，直连（green）/ 通过网关（blue，hover 显示跳板机名）
- 网关主机行特殊 badge："Gateway"

### 4.3 主机详情

`HostDetailPage.tsx` 描述信息区新增：

- 跳板机名称
- 连接模式
- 隧道状态（隧道模式）/ SSH 连接状态（代理模式）

### 4.4 API 类型扩展

`web/src/api/modules/hosts.ts`：

```typescript
interface Host {
  // 现有字段...
  jump_host_id?: number;
  gateway_mode?: 'tunnel' | 'proxy' | 'auto';
  jump_host_name?: string;  // 后端 join 查询
}

interface HostCreateParams {
  // 现有字段...
  jump_host_id?: number;
  gateway_mode?: string;
}

// 新增
getGatewayHosts(): Promise<Host[]>
```

---

## 5. 异常处理

| 场景 | 平台表现 | 处理方式 |
|------|----------|----------|
| B 与 A 断连 | 所有通过 B 的主机显示离线 | B 重连后自动重建隧道 |
| B 与 C 断连 | 对应主机显示离线 | 与直连主机断连一致 |
| 隧道建连超时 | 主机连接失败 | 标记离线 |
| B 重启 | 隧道丢失，主机暂时离线 | C 重连 B 触发隧道重建 |
| C 无 Agent 且 SSH 不通 | 主机不可达 | B 上报离线状态 |

## 6. 安全考虑

- 隧道鉴权：C 连 B 用 mTLS + enroll_token；B 连 A 用现有 mTLS
- 隧道隔离：tunnel_id 绑定唯一主机，平台验证匹配
- 代理模式：SSH 密钥仅存 B 本地，代理执行走白名单
- 隧道限流：最大隧道数限制，空闲隧道自动回收

## 7. 实现顺序

1. Proto 扩展 + 代码生成
2. 数据库模型（Node 扩展 + HostRoute + TunnelSession）
3. Gateway 模块（RouteTable + TunnelManager）
4. server.go 消息处理集成
5. exec_dispatch.go 发送路由
6. ServiceContext 初始化
7. 前端 API 类型 + 方法
8. 前端页面变更
