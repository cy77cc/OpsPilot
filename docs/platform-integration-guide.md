# OpsAgent Platform Integration Guide

> 本文档面向平台侧开发者，说明如何部署 OpsAgent Agent，以及如何在平台端编写 gRPC 服务来接收指标、下发命令。

---

## 目录

1. [架构概览](#1-架构概览)
2. [Agent 安装部署](#2-agent-安装部署)
3. [gRPC Proto 定义](#3-grpc-proto-定义)
4. [平台端服务实现](#4-平台端服务实现)
5. [消息交互流程](#5-消息交互流程)
6. [完整平台端示例 (Go)](#6-完整平台端示例-go)
7. [配置参考](#7-配置参考)
8. [故障排查](#8-故障排查)

---

## 1. 架构概览

```
┌─────────────────────────────────────────────────────┐
│                   Platform (你的服务)                  │
│                                                       │
│   ┌───────────────────────────────────────────────┐  │
│   │         gRPC Server (AgentService)            │  │
│   │                                               │  │
│   │  ┌─────────────┐    ┌─────────────────────┐  │  │
│   │  │  收到 Metrics │    │  下发 ExecuteCommand │  │  │
│   │  │  → 存储/告警  │    │  → 等待 ExecResult   │  │  │
│   │  └─────────────┘    └─────────────────────┘  │  │
│   └───────────────────────┬───────────────────────┘  │
└───────────────────────────┼───────────────────────────┘
                            │ 双向流 (stream)
                            │
┌───────────────────────────┼───────────────────────────┐
│                   OpsAgent Agent                     │
│                           │                             │
│   ┌───────────────────────┴───────────────────────┐   │
│   │              gRPC Client                       │   │
│   │  连接 → 注册 → 心跳 → 收指标 → 发结果          │   │
│   └───────────────────────┬───────────────────────┘   │
│                           │                             │
│   ┌───────────┐  ┌───────┴───────┐  ┌─────────────┐  │
│   │ Collector  │  │   Sandbox     │  │  Executor   │  │
│   │ Pipeline   │  │   Executor    │  │  (local)    │  │
│   │ CPU/Mem/   │  │   nsjail 隔离  │  │  直接执行    │  │
│   │ Disk/Net   │  │   命令/脚本    │  │             │  │
│   └───────────┘  └───────────────┘  └─────────────┘  │
└───────────────────────────────────────────────────────┘
```

**核心通信方式**: Agent 主动连接平台的 gRPC 双向流，平台通过同一个流下发指令。

---

## 2. Agent 安装部署

### 2.1 系统要求

| 项目 | 要求 |
|------|------|
| OS | Linux (amd64/arm64) |
| Go | 1.21+ (仅编译时) |
| nsjail | 可选，sandbox 功能需要 |
| cgroup v2 | 可选，资源限制需要 |

### 2.2 方式一：安装包部署（推荐）

打包脚本会交叉编译 x86_64 和 arm64 两个架构的安装包，内含二进制、配置文件、systemd 服务文件和安装脚本。

**打包**（在开发机上执行）：

```bash
# 打包两个架构
make package

# 仅打包某个架构
make package-amd64
make package-arm64

# 指定版本号
VERSION=1.0.0 make package
```

产物在 `dist/` 目录：

```
dist/
├── opsagent-dev-linux-amd64.tar.gz
├── opsagent-dev-linux-arm64.tar.gz
├── amd64/
│   └── opsagent          # x86_64 二进制
└── arm64/
    └── opsagent-arm64    # arm64 二进制
```

**安装**（在目标机器上执行）：

```bash
# 解压
tar xzf opsagent-<version>-linux-amd64.tar.gz
cd opsagent-<version>-linux-amd64

# 一键安装（需要 root）
sudo ./install.sh
```

安装脚本会自动完成：

| 步骤 | 说明 |
|------|------|
| 安装二进制 | `/usr/local/bin/opsagent` |
| 安装配置 | `/etc/opsagent/config.yaml`（已有则不覆盖，新配置存为 `.new`） |
| 安装 systemd 服务 | `/etc/systemd/system/opsagent.service` |
| 创建日志目录 | `/var/log/opsagent/` |

安装完成后按提示操作：

```bash
# 1. 编辑配置
sudo vim /etc/opsagent/config.yaml

# 2. 启动服务
sudo systemctl start opsagent

# 3. 开机自启
sudo systemctl enable opsagent

# 4. 查看状态
sudo systemctl status opsagent

# 5. 查看日志
sudo journalctl -u opsagent -f
```

### 2.3 方式二：源码编译

```bash
git clone <repo-url> opsagent
cd opsagent

# 编译当前架构
make build
# 产物: bin/opsagent

# 交叉编译两个架构
make build-all
# 产物: bin/opsagent-amd64, bin/opsagent-arm64

# 手动安装
sudo cp bin/opsagent /usr/local/bin/opsagent
sudo mkdir -p /etc/opsagent
sudo cp configs/config.yaml /etc/opsagent/config.yaml
```

### 2.4 配置

配置文件路径：`/etc/opsagent/config.yaml`

**最小配置** (仅指标采集 + gRPC 连接):

```yaml
agent:
  id: "agent-prod-001"        # 唯一标识，建议 hostname 或 UUID
  name: "web-server-01"       # 可读名称
  interval_seconds: 10        # 指标采集间隔

server:
  listen_addr: "127.0.0.1:18080"  # 本地 API 监听地址

executor:
  timeout_seconds: 10
  max_output_bytes: 65536
  allowed_commands:
    - uptime
    - df
    - free
    - hostname

reporter:
  mode: "stdout"

grpc:
  server_addr: "platform.example.com:443"  # 平台 gRPC 地址
  enroll_token: "your-enrollment-token"     # 注册令牌
  mtls:
    cert_file: "/etc/opsagent/certs/client.crt"
    key_file: "/etc/opsagent/certs/client.key"
    ca_file: "/etc/opsagent/certs/ca.crt"
  heartbeat_interval_seconds: 15
  reconnect_initial_backoff_ms: 1000
  reconnect_max_backoff_ms: 30000

collector:
  inputs:
    - type: cpu
      config:
        totalcpu: true
    - type: memory
      config: {}
    - type: disk
      config: {}
    - type: net
      config: {}
    - type: process
      config:
        top_n: 10
  processors:
    - type: tagger
      config:
        tags:
          env: "production"
          region: "cn-east"
  outputs:
    - type: http
      config:
        url: "https://metrics.example.com/api/v1/push"
        timeout: 5
```

**启用 Sandbox** (需要 nsjail):

```yaml
sandbox:
  enabled: true
  nsjail_path: "/usr/bin/nsjail"
  base_workdir: "/tmp/opsagent/sandbox"
  default_timeout_seconds: 30
  max_concurrent_tasks: 4
  cgroup_base_path: "/sys/fs/cgroup/opsagent"
  audit_log_path: "/var/log/opsagent/audit.log"
  policy:
    allowed_commands:
      - echo
      - ls
      - cat
      - grep
      - wc
      - df
      - free
    blocked_commands:
      - rm
      - mkfs
      - dd
      - shutdown
    blocked_keywords:
      - "rm -rf /"
    allowed_interpreters:
      - bash
      - python3
    script_max_bytes: 65536
    shell_injection_check: true
```

### 2.5 Systemd 服务管理

安装包自带 systemd 服务文件，支持以下操作：

```bash
# 启动 / 停止 / 重启
sudo systemctl start opsagent
sudo systemctl stop opsagent
sudo systemctl restart opsagent

# 查看状态
sudo systemctl status opsagent

# 开机自启 / 取消自启
sudo systemctl enable opsagent
sudo systemctl disable opsagent

# 查看日志
sudo journalctl -u opsagent -f           # 实时跟踪
sudo journalctl -u opsagent --since today # 今日日志
sudo journalctl -u opsagent -n 100        # 最近 100 行
```

服务文件特性：

| 特性 | 说明 |
|------|------|
| 自动重启 | 崩溃后 5 秒自动重启 (`Restart=always`) |
| 网络依赖 | 等待网络就绪后启动 (`After=network-online.target`) |
| 安全加固 | `ProtectSystem=strict`, `ProtectHome=true`, `PrivateTmp=true` |
| 日志 | 通过 journald 管理，`LOG_LEVEL=info` 可在服务文件中修改 |

### 2.6 卸载

安装包内含卸载脚本，会停止服务、删除二进制和 systemd 服务文件，配置和日志目录会交互式确认是否删除：

```bash
sudo ./uninstall.sh
```

卸载流程：

| 步骤 | 说明 |
|------|------|
| 停止服务 | `systemctl stop opsagent` |
| 禁用自启 | `systemctl disable opsagent` |
| 删除服务文件 | `/etc/systemd/system/opsagent.service` |
| 删除二进制 | `/usr/local/bin/opsagent` |
| 删除配置 | `/etc/opsagent/`（交互确认） |
| 删除日志 | `/var/log/opsagent/`（交互确认） |
| 删除临时目录 | `/tmp/opsagent/` |

### 2.7 验证安装

```bash
# 检查 binary
opsagent --help

# 检查 sandbox 前置条件（源码编译时）
make sandbox-check

# 运行 smoke test（源码编译时）
./scripts/smoke-test.sh

# 检查本地 API
curl http://127.0.0.1:18080/api/v1/health

# 检查 Prometheus 指标
curl http://127.0.0.1:18080/metrics
```

---

## 3. gRPC Proto 定义

完整 proto 定义在 `proto/agent.proto`。核心 service:

```protobuf
service AgentService {
  // Agent 主动调用，建立双向流
  rpc Connect(stream AgentMessage) returns (stream PlatformMessage);
}
```

### 3.1 Agent → Platform (AgentMessage)

Agent 发送给平台的消息:

```protobuf
message AgentMessage {
  oneof payload {
    AgentRegistration registration = 1;  // 首次连接注册
    Heartbeat heartbeat = 2;             // 周期心跳
    MetricBatch metrics = 3;             // 指标批次
    ExecOutput exec_output = 4;          // 命令执行实时输出
    ExecResult exec_result = 5;          // 命令执行结果
    Ack ack = 6;                         // 确认消息
  }
}
```

| 消息类型 | 触发时机 | 关键字段 |
|---------|---------|---------|
| `AgentRegistration` | 连接建立后立即发送 | `agent_id`, `token`, `agent_info`, `capabilities` |
| `Heartbeat` | 每 15s (可配置) | `agent_id`, `timestamp_ms`, `status`, `agent_info` |
| `MetricBatch` | 每个采集周期 | `metrics[]` (name, tags, fields, timestamp_ms, type) |
| `ExecOutput` | 命令执行过程中实时输出 | `task_id`, `stream` (stdout/stderr), `data` |
| `ExecResult` | 命令执行完成 | `task_id`, `exit_code`, `duration_ms`, `timed_out`, `stats` |

### 3.2 Platform → Agent (PlatformMessage)

平台发送给 Agent 的消息:

```protobuf
message PlatformMessage {
  oneof payload {
    ExecuteCommand exec_command = 1;  // 执行命令
    ExecuteScript exec_script = 2;    // 执行脚本
    CancelJob cancel_job = 3;         // 取消任务
    ConfigUpdate config_update = 4;   // 配置更新
    Ack ack = 5;                      // 确认消息
  }
}
```

| 消息类型 | 用途 | 关键字段 |
|---------|------|---------|
| `ExecuteCommand` | 在 Agent 上执行命令 | `task_id`, `command`, `args[]`, `env{}`, `timeout_seconds`, `sandbox` |
| `ExecuteScript` | 在 Agent 上执行脚本 | `task_id`, `interpreter`, `script`, `args[]`, `env{}`, `timeout_seconds`, `sandbox` |
| `CancelJob` | 取消正在执行的任务 | `task_id`, `reason` |
| `ConfigUpdate` | 推送配置更新 | `config_yaml`, `version` |

---

## 4. 平台端服务实现

### 4.1 核心逻辑

### 4.0 OpsPilot 当前实现说明

当前 OpsPilot 平台侧实现与本文档的关系如下：

- 主机添加流程支持可选安装 `opsagent` 插件，平台会先记录插件实例与安装任务，再通过 SSH/SFTP 在主机侧分发安装包并执行安装脚本。
- 平台已经实现 `AgentService.Connect` 的真实 gRPC 注册入口，连接建立后会记录在线 session、刷新 `host_plugin_instances.runtime_status/last_seen_at`。
- `Heartbeat` 会更新插件实例在线状态；`MetricBatch` 会聚合写入 `host_health_snapshots`；配置 `Ack` 会回写 `host_plugin_config_revisions.delivery_status`。
- AI 主机命令执行已切换为依赖在线 `opsagent` session；没有在线插件实例时不会回退到 SSH。

如果你在对接的不是 OpsPilot，而是另一套平台，那么仍然可以沿用本文档中的协议和时序，只需要自行实现相应的数据落库与任务调度。

平台端需要实现 `AgentService` 的 `Connect` 方法:

```
1. Agent 调用 Connect → 平台收到 stream
2. 从 stream.Recv() 读取第一条消息 → 应为 AgentRegistration
3. 验证 token，注册 Agent
4. 启动 goroutine 循环 Recv() 处理 Agent 消息:
   - Heartbeat → 更新 Agent 状态
   - MetricBatch → 存储指标
   - ExecOutput → 转发给等待的调用方
   - ExecResult → 通知等待的调用方
5. 通过 stream.Send() 下发 PlatformMessage
```

### 4.2 Proto 代码生成

```bash
# 从 proto 生成 Go 代码
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/agent.proto
```

生成的代码在 `internal/grpcclient/proto/` 目录，包括:
- `agent.pb.go` — 消息类型
- `agent_grpc.pb.go` — gRPC 客户端/服务端接口

---

## 5. 消息交互流程

### 5.1 Agent 注册与心跳

```
Agent                                Platform
  │                                     │
  │──── Connect(stream) ──────────────>│
  │                                     │
  │──── AgentRegistration ────────────>│  // agent_id + token + info
  │                                     │  // 验证 token, 注册 Agent
  │<──── Ack (success=true) ───────────│
  │                                     │
  │──── Heartbeat ───────────────────>│  // 每 15s
  │                                     │  // 更新 last_seen
  │──── Heartbeat ───────────────────>│
  │     ...                             │
```

### 5.2 指标上报

```
Agent                                Platform
  │                                     │
  │──── MetricBatch ─────────────────>│  // cpu, memory, disk, net, process
  │                                     │  // 存储到时序数据库
  │                                     │
  │──── MetricBatch ─────────────────>│  // 下一个采集周期
  │     ...                             │
```

**MetricBatch 结构示例**:

```json
{
  "metrics": [
    {
      "name": "cpu",
      "tags": {"cpu": "cpu-total"},
      "fields": [{"key": "usage_percent", "double_value": 45.2}],
      "timestamp_ms": 1714300000000,
      "type": "GAUGE"
    },
    {
      "name": "memory",
      "tags": {},
      "fields": [
        {"key": "total_bytes", "int_value": 17179869184},
        {"key": "used_percent", "double_value": 62.5}
      ],
      "timestamp_ms": 1714300000000,
      "type": "GAUGE"
    }
  ]
}
```

### 5.3 下发命令执行

```
Platform                             Agent
  │                                     │
  │──── ExecuteCommand ───────────────>│  // task_id + command + args
  │                                     │  // 验证 policy (白名单/黑名单)
  │                                     │  // 如果 sandbox 启用 → nsjail 隔离执行
  │                                     │  // 否则 → 直接 exec
  │                                     │
  │<──── ExecOutput (stdout) ─────────│  // 实时输出 (可选)
  │<──── ExecOutput (stdout) ─────────│
  │<──── ExecResult ──────────────────│  // exit_code + duration + stats
  │                                     │
  │──── Ack ─────────────────────────>│  // 确认收到结果
```

**ExecuteCommand 示例**:

```json
{
  "task_id": "task-20260428-001",
  "command": "df",
  "args": ["-h", "/"],
  "env": {"LANG": "C"},
  "timeout_seconds": 10,
  "sandbox": {
    "memory_mb": 128,
    "cpu_quota_pct": 50,
    "max_pids": 32,
    "network_mode": "disabled"
  }
}
```

**ExecResult 示例**:

```json
{
  "task_id": "task-20260428-001",
  "exit_code": 0,
  "duration_ms": 120,
  "timed_out": false,
  "truncated": false,
  "killed": false,
  "stats": {
    "peak_memory_bytes": 2048000,
    "cpu_time_user_ms": 10,
    "cpu_time_system_ms": 5,
    "process_count": 1,
    "bytes_written": 1024,
    "bytes_read": 0
  }
}
```

### 5.4 下发脚本执行

```
Platform                             Agent
  │                                     │
  │──── ExecuteScript ────────────────>│  // task_id + interpreter + script
  │                                     │  // 通过 sandbox 隔离执行
  │<──── ExecOutput (stdout) ─────────│  // 实时流式输出
  │<──── ExecOutput (stdout) ─────────│
  │<──── ExecResult ──────────────────│
```

**ExecuteScript 示例**:

```json
{
  "task_id": "task-20260428-002",
  "interpreter": "bash",
  "script": "echo 'Disk usage:' && df -h && echo 'Memory:' && free -h",
  "timeout_seconds": 30,
  "sandbox": {
    "memory_mb": 256,
    "network_mode": "disabled"
  }
}
```

---

## 6. 完整平台端示例 (Go)

以下是一个完整的平台端 gRPC 服务实现:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "your-project/proto" // 替换为你的 proto 包路径
)

// AgentServer 实现 AgentService gRPC 服务。
type AgentServer struct {
	pb.UnimplementedAgentServiceServer

	mu     sync.RWMutex
	agents map[string]*AgentSession // agent_id → session
}

// AgentSession 代表一个已连接的 Agent。
type AgentSession struct {
	AgentID  string
	Stream   pb.AgentService_ConnectServer
	Info     *pb.AgentInfo
	LastSeen int64

	// 用于等待命令结果
	resultCh chan *pb.ExecResult
	outputCh chan *pb.ExecOutput
}

func NewAgentServer() *AgentServer {
	return &AgentServer{
		agents: make(map[string]*AgentSession),
	}
}

// Connect 是核心方法: 处理 Agent 的双向流连接。
func (s *AgentServer) Connect(stream pb.AgentService_ConnectServer) error {
	// 1. 读取注册消息
	regMsg, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive registration: %v", err)
	}

	reg := regMsg.GetRegistration()
	if reg == nil {
		return status.Errorf(codes.InvalidArgument, "first message must be registration")
	}

	// 2. 验证 token
	if !s.validateToken(reg.GetToken()) {
		// 发送失败 ack
		stream.Send(&pb.PlatformMessage{
			Payload: &pb.PlatformMessage_Ack{
				Ack: &pb.Ack{
					RefId:   "registration",
					Success: false,
					Error:   "invalid token",
				},
			},
		})
		return status.Errorf(codes.Unauthenticated, "invalid token")
	}

	agentID := reg.GetAgentId()
	log.Printf("[+] Agent connected: %s (host=%s, os=%s)",
		agentID, reg.GetAgentInfo().GetHostname(), reg.GetAgentInfo().GetOs())

	// 3. 注册 session
	session := &AgentSession{
		AgentID:  agentID,
		Stream:   stream,
		Info:     reg.GetAgentInfo(),
		resultCh: make(chan *pb.ExecResult, 10),
		outputCh: make(chan *pb.ExecOutput, 100),
	}

	s.mu.Lock()
	s.agents[agentID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.agents, agentID)
		s.mu.Unlock()
		log.Printf("[-] Agent disconnected: %s", agentID)
	}()

	// 4. 发送注册成功 ack
	stream.Send(&pb.PlatformMessage{
		Payload: &pb.PlatformMessage_Ack{
			Ack: &pb.Ack{
				RefId:   "registration",
				Success: true,
			},
		},
	})

	// 5. 消息接收循环
	for {
		msg, err := stream.Recv()
		if err != nil {
			return err // stream 关闭
		}

		switch p := msg.Payload.(type) {
		case *pb.AgentMessage_Heartbeat:
			hb := p.Heartbeat
			session.LastSeen = hb.GetTimestampMs()
			log.Printf("[HB] %s status=%s", agentID, hb.GetStatus())

		case *pb.AgentMessage_Metrics:
			batch := p.Metrics
			log.Printf("[METRICS] %s: %d metrics", agentID, len(batch.GetMetrics()))
			// TODO: 写入时序数据库 (InfluxDB/Prometheus/etc.)
			for _, m := range batch.GetMetrics() {
				s.processMetric(agentID, m)
			}

		case *pb.AgentMessage_ExecOutput:
			out := p.ExecOutput
			log.Printf("[OUTPUT] %s [%s]: %s", out.GetTaskId(), out.GetStream(), string(out.GetData()))
			session.outputCh <- out

		case *pb.AgentMessage_ExecResult:
			res := p.ExecResult
			log.Printf("[RESULT] %s: exit_code=%d duration=%dms",
				res.GetTaskId(), res.GetExitCode(), res.GetDurationMs())
			session.resultCh <- res

		case *pb.AgentMessage_Ack:
			ack := p.Ack
			log.Printf("[ACK] %s: success=%v", ack.GetRefId(), ack.GetSuccess())
		}
	}
}

// ExecuteCommand 向指定 Agent 下发命令执行。
func (s *AgentServer) ExecuteCommand(ctx context.Context, agentID string, cmd *pb.ExecuteCommand) (*pb.ExecResult, error) {
	s.mu.RLock()
	session, ok := s.agents[agentID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not connected", agentID)
	}

	// 清空之前的结果
	for {
		select {
		case <-session.resultCh:
		default:
			goto drained
		}
	}
drained:

	// 发送命令
	msg := &pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ExecCommand{
			ExecCommand: cmd,
		},
	}
	if err := session.Stream.Send(msg); err != nil {
		return nil, fmt.Errorf("send command: %w", err)
	}

	// 等待结果 (带超时)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-session.resultCh:
		return result, nil
	}
}

// ExecuteScript 向指定 Agent 下发脚本执行。
func (s *AgentServer) ExecuteScript(ctx context.Context, agentID string, script *pb.ExecuteScript) (*pb.ExecResult, error) {
	s.mu.RLock()
	session, ok := s.agents[agentID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("agent %s not connected", agentID)
	}

	msg := &pb.PlatformMessage{
		Payload: &pb.PlatformMessage_ExecScript{
			ExecScript: script,
		},
	}
	if err := session.Stream.Send(msg); err != nil {
		return nil, fmt.Errorf("send script: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-session.resultCh:
		return result, nil
	}
}

// ListAgents 返回所有已连接的 Agent。
func (s *AgentServer) ListAgents() []*AgentSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AgentSession, 0, len(s.agents))
	for _, session := range s.agents {
		result = append(result, session)
	}
	return result
}

func (s *AgentServer) validateToken(token string) bool {
	// TODO: 实现真实的 token 验证逻辑
	return token != ""
}

func (s *AgentServer) processMetric(agentID string, m *pb.Metric) {
	// TODO: 写入你的时序数据库
	// 示例: InfluxDB, Prometheus Remote Write, VictoriaMetrics, etc.
	log.Printf("  metric: %s %v %v", m.GetName(), m.GetTags(), m.GetFields())
}

func main() {
	lis, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	// TODO: 配置 TLS
	srv := grpc.NewServer()
	agentSrv := NewAgentServer()
	pb.RegisterAgentServiceServer(srv, agentSrv)

	log.Println("Platform gRPC server listening on :443")
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
```

### 6.1 调用示例

```go
// 向 Agent 下发命令
result, err := agentSrv.ExecuteCommand(ctx, "agent-prod-001", &pb.ExecuteCommand{
	TaskId:         "task-001",
	Command:        "df",
	Args:           []string{"-h", "/"},
	TimeoutSeconds: 10,
	Sandbox: &pb.SandboxConfig{
		MemoryMb:    128,
		CpuQuotaPct: 50,
		MaxPids:     32,
		NetworkMode: "disabled",
	},
})
if err != nil {
	log.Printf("execute failed: %v", err)
} else {
	log.Printf("exit_code=%d, duration=%dms", result.GetExitCode(), result.GetDurationMs())
}

// 向 Agent 下发脚本
result, err := agentSrv.ExecuteScript(ctx, "agent-prod-001", &pb.ExecuteScript{
	TaskId:      "task-002",
	Interpreter: "bash",
	Script:      "echo '=== System Info ===' && uname -a && uptime && free -h",
	TimeoutSeconds: 30,
})
```

---

## 7. 配置参考

### 7.1 Agent 配置完整字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `agent.id` | string | (必填) | Agent 唯一标识 |
| `agent.name` | string | (必填) | Agent 可读名称 |
| `agent.interval_seconds` | int | 10 | 指标采集间隔 (秒) |
| `server.listen_addr` | string | 0.0.0.0:18080 | 本地 API 监听地址 |
| `grpc.server_addr` | string | (必填) | 平台 gRPC 地址 |
| `grpc.enroll_token` | string | "" | 注册令牌 |
| `grpc.mtls.cert_file` | string | "" | 客户端证书路径 |
| `grpc.mtls.key_file` | string | "" | 客户端私钥路径 |
| `grpc.mtls.ca_file` | string | "" | CA 证书路径 |
| `grpc.heartbeat_interval_seconds` | int | 15 | 心跳间隔 |
| `grpc.reconnect_initial_backoff_ms` | int | 1000 | 重连初始退避 (ms) |
| `grpc.reconnect_max_backoff_ms` | int | 30000 | 重连最大退避 (ms) |
| `sandbox.enabled` | bool | false | 是否启用 sandbox |
| `sandbox.nsjail_path` | string | /usr/bin/nsjail | nsjail 路径 |
| `sandbox.default_timeout_seconds` | int | 30 | 默认执行超时 |
| `sandbox.max_concurrent_tasks` | int | 4 | 最大并发任务数 |
| `collector.inputs[]` | list | - | 采集插件列表 |
| `collector.processors[]` | list | - | 处理插件列表 |
| `collector.outputs[]` | list | - | 输出插件列表 |

### 7.2 可用采集插件

| 插件 | type | 可选 config |
|------|------|------------|
| CPU | `cpu` | `totalcpu: true`, `percpu: false` |
| 内存 | `memory` | 无 |
| 磁盘 | `disk` | `mount_points: ["/", "/data"]` |
| 网络 | `net` | 无 |
| 进程 | `process` | `top_n: 10` |

### 7.3 可用处理插件

| 插件 | type | config |
|------|------|--------|
| 标签器 | `tagger` | `tags: {env: "prod", region: "east"}` |
| 正则替换 | `regex` | `tags: [{key: "host", pattern: "...", replacement: "..."}]` |

### 7.4 可用聚合插件

| 插件 | type | config |
|------|------|--------|
| 平均值 | `avg` | `fields: ["usage_percent"]` |
| 求和 | `sum` | `fields: ["bytes_sent"]` |

### 7.5 可用输出插件

| 插件 | type | config |
|------|------|--------|
| HTTP | `http` | `url`, `timeout`, `batch_size`, `retry_count` |
| Prometheus | `prometheus` | `path`, `addr` |
| Prometheus Remote Write | `prometheus_remote_write` | `url`, `timeout` |

---

## 8. 故障排查

### 8.1 Agent 无法连接平台

```bash
# 检查网络连通性
nc -zv platform.example.com 443

# 检查证书
openssl x509 -in /etc/opsagent/certs/client.crt -noout -dates

# 查看 Agent 日志 (开启 debug)
LOG_LEVEL=debug ./bin/opsagent run --config /etc/opsagent/config.yaml
```

### 8.2 Sandbox 命令被拒绝

```bash
# 检查 nsjail
which nsjail
nsjail --version

# 检查 cgroup
cat /sys/fs/cgroup/cgroup.controllers

# 检查 Agent 日志中的 policy 错误
journalctl -u opsagent | grep "policy"
```

### 8.3 指标未到达平台

```bash
# 检查 Agent 本地 Prometheus 端点
curl http://127.0.0.1:18080/metrics

# 检查 collector 配置
# 确保 collector.inputs 中至少有一个 input 配置正确

# 检查 gRPC 连接状态
curl http://127.0.0.1:18080/api/v1/health
```

### 8.4 常见错误码

| 场景 | exit_code | 说明 |
|------|-----------|------|
| 正常退出 | 0 | 命令执行成功 |
| 命令错误 | 1-125 | 命令自身返回的错误码 |
| 超时被杀 | -1 | 执行超时, `timed_out=true` |
| Policy 拒绝 | N/A | gRPC 返回 error, 不产生 ExecResult |
| Sandbox 错误 | N/A | cgroup/nsjail 配置问题 |
