# Gateway Tunnel 平台对接文档

## 1. 概述

Gateway Tunnel 功能使 OpsAgent 充当跳板机，让平台能够管理无法直连的内网主机。平台对内网主机的所有操作与直连主机完全一致，跳板逻辑对平台透明。

### 架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      OpsPilot 平台 (A)                          │
│                                                                 │
│  主机列表：                                                      │
│  ├── B    (Agent，gateway 类型)                                  │
│  ├── C1   (跳板=B，平台感知为普通主机)                            │
│  ├── C2   (跳板=B，平台感知为普通主机)                            │
│  └── Cn   (跳板=B，平台感知为普通主机)                            │
└──────────────┬──────────────────────────────────────────────────┘
               │ gRPC 双向流 (复用现有通道)
               │
┌──────────────┴──────────────────────────────────────────────────┐
│                   OpsAgent (B 主机 - 跳板机)                     │
│                                                                 │
│  Gateway Module：                                               │
│  ├── 隧道模式：C 有 Agent，B 透明转发 gRPC 流量                  │
│  └── 代理模式：C 无 Agent，B 通过 SSH 代为执行命令               │
└──────────────┬──────────────────────────────────────────────────┘
               │ TCP / SSH
               ▼
┌─────────────────────────────────────────────────────────────────┐
│  C1 (有 Agent)    │  C2 (无 Agent)    │  C3 (有 Agent)    │ ... │
│  Agent → B 网关   │  B 代为 SSH 执行  │  Agent → B 网关   │     │
└─────────────────────────────────────────────────────────────────┘
```

### 两种模式对比

| | 隧道模式（C 有 Agent） | 代理模式（C 无 Agent） |
|---|---|---|
| C 在平台的表现 | 普通受管主机 | 普通受管主机 |
| 指标采集 | C 自己采集，经 B 隧道上报 | B 代为采集上报 |
| 命令执行 | 平台发给 C，经 B 隧道转发 | B 代为执行上报 |
| 连接方式 | C 的 Agent → B → A | B → C (SSH) |

---

## 2. Proto 消息定义

### 2.1 新增消息类型

```proto
// --- Gateway Tunnel Messages ---

message TunnelOpen {
  string tunnel_id = 1;          // 隧道唯一标识
  string agent_id = 2;           // C 的 Agent ID
  string hostname = 3;           // C 的主机名
  string ip = 4;                 // C 的 IP 地址
  repeated string capabilities = 5; // C 的能力列表
}

message TunnelData {
  string tunnel_id = 1;          // 隧道标识
  bytes payload = 2;             // 序列化的 AgentMessage 或 PlatformMessage
}

message TunnelClose {
  string tunnel_id = 1;          // 隧道标识
  string reason = 2;             // 关闭原因
}

message ProxyHostRegister {
  string host_id = 1;            // 主机唯一标识
  string hostname = 2;           // 主机名
  string ip = 3;                 // IP 地址
  repeated string capabilities = 4; // 能力列表
}

message ProxyCommandRequest {
  string host_id = 1;            // 目标主机 ID
  string command = 2;            // 要执行的命令
  repeated string args = 3;      // 命令参数
  int32 timeout_seconds = 4;     // 超时时间
}

message ProxyCommandResponse {
  string host_id = 1;            // 目标主机 ID
  string command = 2;            // 执行的命令
  int32 exit_code = 3;           // 退出码
  bytes stdout = 4;              // 标准输出
  bytes stderr = 5;              // 标准错误
  int64 duration_ms = 6;         // 执行耗时（毫秒）
  bool timed_out = 7;            // 是否超时
}

message ProxyMetricBatch {
  string host_id = 1;            // 目标主机 ID
  repeated Metric metrics = 2;   // 指标数据
}
```

### 2.2 AgentMessage 扩展

```proto
message AgentMessage {
  oneof payload {
    // 现有消息 (1-7)
    AgentRegistration registration = 1;
    Heartbeat heartbeat = 2;
    MetricBatch metrics = 3;
    ExecOutput exec_output = 4;
    ExecResult exec_result = 5;
    Ack ack = 6;
    HealthCheckResult health_check_result = 7;

    // Gateway 新增 (8-13)
    TunnelOpen tunnel_open = 8;           // B → 平台：隧道建立
    TunnelData tunnel_data = 9;           // B → 平台：隧道数据
    TunnelClose tunnel_close = 10;        // B → 平台：隧道关闭
    ProxyHostRegister proxy_register = 11; // B → 平台：代理主机注册
    ProxyCommandResponse proxy_response = 12; // B → 平台：代理命令结果
    ProxyMetricBatch proxy_metrics = 13;  // B → 平台：代理指标数据
  }
}
```

### 2.3 PlatformMessage 扩展

```proto
message PlatformMessage {
  oneof payload {
    // 现有消息 (1-6)
    ExecuteCommand exec_command = 1;
    ExecuteScript exec_script = 2;
    CancelJob cancel_job = 3;
    ConfigUpdate config_update = 4;
    Ack ack = 5;
    HealthCheckRequest health_check = 6;

    // Gateway 新增 (7-9)
    TunnelData tunnel_data = 7;           // 平台 → B：隧道数据
    TunnelClose tunnel_close = 8;         // 平台 → B：关闭隧道
    ProxyCommandRequest proxy_command = 9; // 平台 → B：代理命令请求
  }
}
```

---

## 3. 集成流程

### 3.1 隧道模式（C 有 Agent）

#### 流程图

```
C 的 Agent                 B 的 Agent (Gateway)           平台 A
    │                           │                           │
    │── TCP Connect ───────────►│                           │
    │   (目标: B:18081)         │                           │
    │                           │── TunnelOpen ────────────►│
    │                           │   {tunnel_id, agent_id,   │
    │                           │    hostname, ip}          │
    │                           │                           │
    │                           │◄── Ack ──────────────────│
    │                           │                           │
    │◄══════ TCP 双向桥接 ═════►│◄═══ gRPC 双向隧道 ══════►│
    │                           │                           │
    │── AgentMessage ──────────►│── TunnelData ────────────►│
    │   (registration)          │   {tunnel_id, payload}    │
    │                           │                           │
    │◄── PlatformMessage ──────│◄── TunnelData ───────────│
    │   (ack)                   │   {tunnel_id, payload}    │
    │                           │                           │
    │◄═══════════════════════════════════════════════════►│
    │         所有 gRPC 消息经 B 透明转发                     │
```

#### 平台处理逻辑

1. **收到 TunnelOpen**：
   - 解析 `tunnel_id`、`agent_id`、`hostname`、`ip`
   - 创建隧道会话，绑定 `tunnel_id` 到该主机
   - 将 `agent_id` 注册到主机列表（标记为通过网关连接）
   - 返回 Ack

2. **收到 TunnelData**：
   - 根据 `tunnel_id` 查找对应的主机
   - 反序列化 `payload` 为 `AgentMessage`
   - 按正常 Agent 消息处理（注册、心跳、指标、命令结果等）

3. **发送消息给 C**：
   - 将 `PlatformMessage` 序列化为 bytes
   - 构造 `TunnelData { tunnel_id, payload }`
   - 通过 B 的 gRPC 流发送

4. **收到 TunnelClose**：
   - 清理隧道会话
   - 标记该主机离线（与直连主机断连表现一致）

### 3.2 代理模式（C 无 Agent）

#### 流程图

```
平台 A                     B 的 Agent (Gateway)           C (无 Agent)
  │                           │                           │
  │                           │── ProxyHostRegister ─────►│ (注册到平台)
  │                           │   {host_id, hostname, ip} │
  │                           │                           │
  │◄── Ack ──────────────────│                           │
  │                           │                           │
  │── ProxyCommandRequest ──►│                           │
  │   {host_id, command, args}│                           │
  │                           │── SSH Connect ────────────►│
  │                           │── SSH Execute ────────────►│
  │                           │◄── Command Output ────────│
  │                           │                           │
  │◄── ProxyCommandResponse ─│                           │
  │   {host_id, exit_code,    │                           │
  │    stdout, stderr, ...}   │                           │
```

#### 平台处理逻辑

1. **收到 ProxyHostRegister**：
   - 解析 `host_id`、`hostname`、`ip`
   - 注册主机到主机列表（标记为代理模式）
   - 返回 Ack

2. **发送命令给代理主机**：
   - 构造 `ProxyCommandRequest { host_id, command, args, timeout_seconds }`
   - 通过 B 的 gRPC 流发送

3. **收到 ProxyCommandResponse**：
   - 解析 `host_id`、`exit_code`、`stdout`、`stderr`、`duration_ms`、`timed_out`
   - 按正常命令结果处理

4. **收到 ProxyMetricBatch**：
   - 解析 `host_id` 和 `metrics`
   - 按正常指标数据处理

---

## 4. 平台侧变更

### 4.1 主机注册

添加主机时新增 `jump_host` 字段：

```proto
message RegisterRequest {
  string agent_id = 1;
  string hostname = 2;
  string ip = 3;
  string jump_host = 4;  // 新增：跳板机 Agent ID，为空表示直连
}
```

平台 UI 添加主机时新增"跳板机"下拉选项，列出所有已注册的 gateway 类型 Agent。

### 4.2 连接路由表

平台维护路由表：

```go
type HostRoute struct {
    HostID    string
    Direct    bool           // true=直连, false=通过网关
    GatewayID string         // 网关 Agent ID（Direct=false 时有效）
    TunnelID  string         // 隧道 ID（隧道模式时有效）
}
```

### 4.3 消息路由逻辑

```go
func (p *Platform) SendMessage(hostID string, msg *PlatformMessage) error {
    route := p.routeTable.Get(hostID)
    if route.Direct {
        return p.sendDirect(hostID, msg)
    }

    // 通过网关发送
    if route.TunnelID != "" {
        // 隧道模式：包装为 TunnelData
        payload, _ := proto.Marshal(msg)
        return p.sendToGateway(route.GatewayID, &PlatformMessage{
            Payload: &PlatformMessage_TunnelData{
                TunnelData: &TunnelData{
                    TunnelId: route.TunnelID,
                    Payload:  payload,
                },
            },
        })
    }

    // 代理模式：直接发送 ProxyCommandRequest
    return p.sendToGateway(route.GatewayID, msg)
}
```

### 4.4 接收消息路由

```go
func (p *Platform) HandleAgentMessage(gatewayID string, msg *AgentMessage) error {
    switch payload := msg.Payload.(type) {
    case *AgentMessage_TunnelOpen:
        return p.handleTunnelOpen(gatewayID, payload.TunnelOpen)
    case *AgentMessage_TunnelData:
        return p.handleTunnelData(gatewayID, payload.TunnelData)
    case *AgentMessage_TunnelClose:
        return p.handleTunnelClose(gatewayID, payload.TunnelClose)
    case *AgentMessage_ProxyRegister:
        return p.handleProxyRegister(gatewayID, payload.ProxyRegister)
    case *AgentMessage_ProxyResponse:
        return p.handleProxyResponse(gatewayID, payload.ProxyResponse)
    case *AgentMessage_ProxyMetrics:
        return p.handleProxyMetrics(gatewayID, payload.ProxyMetrics)
    default:
        // 正常消息处理
        return p.handleNormalMessage(gatewayID, msg)
    }
}
```

---

## 5. 配置示例

### 5.1 B 的 Agent 配置（跳板机）

```yaml
agent:
  id: "gateway-b-001"
  name: "gateway-host-b"

gateway:
  enabled: true
  listen_addr: ":18081"          # C 的 Agent 连入端口（隧道模式）
  max_tunnels: 100               # 最大隧道数
  tunnel_timeout_seconds: 30     # 隧道建立超时
  idle_timeout_seconds: 300      # 空闲隧道回收

  hosts:
    # 代理模式：C 无 Agent，B 通过 SSH 代为执行
    - id: "vm-web-01"
      addr: "192.168.122.100"
      mode: "proxy"
      ssh:
        user: "root"
        key_file: "/etc/opsagent/keys/id_rsa"
        port: 22

    # 隧道模式：C 有 Agent，B 透明转发
    - id: "vm-db-01"
      addr: "192.168.122.101"
      mode: "tunnel"

    # 自动模式：B 自动检测 C 是否有 Agent
    - id: "vm-app-01"
      addr: "192.168.122.102"
      mode: "auto"
      ssh:
        user: "root"
        key_file: "/etc/opsagent/keys/id_rsa"
        port: 22
```

### 5.2 C 的 Agent 配置（隧道模式）

```yaml
agent:
  id: "vm-db-01"
  name: "database-server"

grpc:
  server_addr: "192.168.1.10:18081"  # 指向 B 的 Gateway 端口
  enroll_token: "your-enroll-token"
  heartbeat_interval_seconds: 15
  mtls:
    cert_file: "/etc/opsagent/certs/client.crt"
    key_file: "/etc/opsagent/certs/client.key"
    ca_file: "/etc/opsagent/certs/ca.crt"
```

---

## 6. 异常处理

| 场景 | 平台表现 | 处理方式 |
|------|----------|----------|
| B 与 A 断连 | 所有通过 B 的主机显示离线 | B 重连 A 后自动重建隧道 |
| B 与 C 断连 | 对应主机显示离线 | 与直连主机断连表现一致 |
| 隧道建连超时 | 主机连接失败 | 平台标记主机离线，心跳超时后显示 |
| B 重启 | 隧道丢失，主机暂时离线 | C 的 Agent 自动重连 B 触发隧道重建 |
| C 无 Agent 且 SSH 不通 | 主机不可达 | B 上报主机离线状态 |

---

## 7. 可观测性

### 7.1 Prometheus 指标

B 的 Agent 暴露以下指标：

```prometheus
# 当前活跃隧道数
opsagent_gateway_tunnels_active

# 隧道流量统计（字节）
opsagent_gateway_tunnel_bytes_total

# 隧道错误计数
opsagent_gateway_tunnel_errors_total

# 代理模式请求数
opsagent_gateway_proxy_requests_total

# 代理执行延迟（秒）
opsagent_gateway_proxy_latency_seconds
```

### 7.2 健康检查

B 的 `/healthz` 端点返回 gateway 子系统状态：

```json
{
  "status": "healthy",
  "subsystems": {
    "collector": "healthy",
    "grpc": "healthy",
    "gateway": {
      "status": "running",
      "details": {
        "active_tunnels": 15,
        "max_tunnels": 100
      }
    }
  }
}
```

### 7.3 审计日志

B 的 Agent 记录以下审计事件：

```json
{"event_type": "gateway.started", "component": "gateway", "action": "start", "status": "success"}
{"event_type": "gateway.tunnel.close", "component": "gateway", "action": "tunnel_close", "details": {"tunnel_id": "xxx", "reason": "idle_timeout"}}
{"event_type": "gateway.proxy.exec", "component": "gateway", "action": "proxy_command", "details": {"host_id": "vm-web-01", "command": "uptime"}}
```

---

## 8. 安全考虑

1. **隧道鉴权**：C 连 B 使用 mTLS + enroll_token；B 连 A 使用现有 mTLS
2. **隧道隔离**：每个 tunnel_id 绑定唯一 C 主机，平台验证 tunnel_id 与目标主机匹配
3. **代理模式安全**：SSH 密钥只存 B 本地，代理执行走现有白名单策略
4. **隧道限流**：最大隧道数限制（默认 100），空闲隧道自动回收（默认 300 秒）
5. **密码保护**：配置文件中的 SSH 密码在 diff 输出中自动脱敏

---

## 9. 平台 UI 变更

### 9.1 添加主机页面

新增字段：
- **跳板机**：下拉选择，列出所有 gateway 类型 Agent
- **连接模式**：当选择跳板机后显示
  - 隧道模式（C 有 Agent）
  - 代理模式（C 无 Agent）
  - 自动检测

### 9.2 主机列表

- 显示连接方式：直连 / 通过网关
- 显示网关状态：在线 / 离线

### 9.3 主机详情

- 显示隧道状态（隧道模式）
- 显示 SSH 连接状态（代理模式）
