# Design: AI 助手模块重构

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Request Flow                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   HTTP Request                                                              │
│       │                                                                     │
│       ▼                                                                     │
│   ┌─────────────────────────────────────────────────────────────────┐      │
│   │                    chat_handler.go                               │      │
│   │  - 解析请求                                                      │      │
│   │  - 初始化 SSE                                                    │      │
│   │  - 调用 PlatformRunner                                          │      │
│   └────────────────────────────┬────────────────────────────────────┘      │
│                                │                                            │
│                                ▼                                            │
│   ┌─────────────────────────────────────────────────────────────────┐      │
│   │                    runner.go (新增)                              │      │
│   │  - 封装 adk.Runner                                              │      │
│   │  - 管理 CheckPointStore                                         │      │
│   │  - Query / Resume API                                           │      │
│   └────────────────────────────┬────────────────────────────────────┘      │
│                                │                                            │
│                                ▼                                            │
│   ┌─────────────────────────────────────────────────────────────────┐      │
│   │                    agent.go (重构)                               │      │
│   │  - NewPlatformAgent() 使用 ChatModelAgent                       │      │
│   │  - 统一 Instruction 定义                                         │      │
│   └────────────────────────────┬────────────────────────────────────┘      │
│                                │                                            │
│                                ▼                                            │
│   ┌─────────────────────────────────────────────────────────────────┐      │
│   │                    tools/builder.go                              │      │
│   │  - 工具构建（保持不变）                                          │      │
│   │  - 风险包装（保持不变）                                          │      │
│   └─────────────────────────────────────────────────────────────────┘      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Component Design

### 1. PlatformRunner (新增)

封装 `adk.Runner`，提供查询和恢复能力。

```go
// internal/ai/runner.go

package ai

import (
    "context"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
    "github.com/cloudwego/eino/schema"
)

// PlatformRunner 封装 ADK Runner，管理 Agent 执行生命周期
type PlatformRunner struct {
    runner  *adk.Runner
    store   compose.CheckPointStore
    agent   adk.Agent
    model   model.ToolCallingChatModel
}

// RunnerConfig Runner 配置
type RunnerConfig struct {
    EnableStreaming bool
    RedisClient     *redis.Client  // 可选，用于分布式检查点存储
}

// NewPlatformRunner 创建 PlatformRunner
func NewPlatformRunner(ctx context.Context, chatModel model.ToolCallingChatModel, deps tools.PlatformDeps, cfg *RunnerConfig) (*PlatformRunner, error) {
    // 1. 构建工具
    allTools, err := tools.BuildAllTools(ctx, deps)
    if err != nil {
        return nil, fmt.Errorf("build tools: %w", err)
    }

    // 2. 创建 Agent
    agent, err := newPlatformAgent(ctx, chatModel, allTools)
    if err != nil {
        return nil, fmt.Errorf("create agent: %w", err)
    }

    // 3. 创建检查点存储
    var store compose.CheckPointStore
    if cfg.RedisClient != nil {
        store = NewRedisCheckPointStore(cfg.RedisClient)
    } else {
        store = NewInMemoryCheckPointStore()
    }

    // 4. 创建 Runner
    runner := adk.NewRunner(ctx, adk.RunnerConfig{
        EnableStreaming: cfg.EnableStreaming,
        Agent:           agent,
        CheckPointStore: store,
    })

    return &PlatformRunner{
        runner: runner,
        store:  store,
        agent:  agent,
        model:  chatModel,
    }, nil
}

// Query 执行查询
func (r *PlatformRunner) Query(ctx context.Context, sessionID, message string, opts ...adk.RunnerOption) *adk.Iterator {
    return r.runner.Query(ctx, message, append(opts, adk.WithCheckPointID(sessionID))...)
}

// Resume 恢复执行（用于审批后）
func (r *PlatformRunner) Resume(ctx context.Context, sessionID string, toolOpts []tool.Option, opts ...adk.RunnerOption) (*adk.Iterator, error) {
    return r.runner.Resume(ctx, sessionID, append(opts, adk.WithToolOptions(toolOpts))...)
}

// Close 清理资源
func (r *PlatformRunner) Close() error {
    // 清理 MCP 连接等
    return nil
}
```

### 2. Agent 创建重构

```go
// internal/ai/agent.go

package ai

import (
    "context"
    "fmt"

    "github.com/cloudwego/eino/adk"
    "github.com/cloudwego/eino/components/model"
    "github.com/cloudwego/eino/components/tool"
    "github.com/cloudwego/eino/compose"
)

const platformAgentInstruction = `你是一个专业的智能运维助手，具备以下核心能力：

## 核心能力

### 主机管理
- host_list_inventory: 查询主机资产清单
- host_ssh_exec_readonly: 在主机上执行只读命令
- host_batch_exec_preview: 批量命令预检查
- host_batch_exec_apply: 批量执行命令（需审批）

### K8s 运维
- k8s_list_resources: 列出 K8s 资源（pods/services/deployments/nodes）
- k8s_get_events: 获取 K8s 事件
- k8s_get_pod_logs: 获取 Pod 日志

### 服务管理
- service_list_inventory: 查询服务清单
- service_get_detail: 获取服务详情
- service_deploy_preview: 预览服务部署
- service_deploy_apply: 执行服务部署（需审批）

### 监控与诊断
- os_get_cpu_mem: 获取 CPU/内存信息
- os_get_disk_fs: 获取磁盘使用情况
- os_get_net_stat: 获取网络状态
- monitor_alert_active: 查询活跃告警

## 执行规则

1. **直接执行**: 用户请求时，直接调用相应工具，不要输出计划或步骤
2. **参数解析**: 从用户输入中提取必要参数，如主机名、命令等
3. **风险操作**: 高风险操作会自动触发审批流程，等待用户确认
4. **结果呈现**: 工具执行后，以清晰的方式呈现结果

## 交互示例

用户: "查看香港云服务器的磁盘使用情况"
正确做法:
  1. 调用 host_list_inventory(keyword="香港云服务器") 获取主机 ID
  2. 调用 host_ssh_exec_readonly(host_id=X, command="df -h")
  3. 呈现结果

错误做法:
  - 输出 {"steps": [...]} 这样的计划 JSON
  - 只说"我将调用..."而不实际调用

## 注意事项

- 不要编造不存在的工具
- 参数不足时询问用户
- 执行失败时给出清晰的原因和建议
`

// newPlatformAgent 创建平台运维 Agent
func newPlatformAgent(ctx context.Context, chatModel model.ToolCallingChatModel, allTools []tool.BaseTool) (adk.Agent, error) {
    if chatModel == nil {
        return nil, fmt.Errorf("chat model is nil")
    }

    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "PlatformOps",
        Description: "智能运维助手，可以执行主机管理、K8s 操作、服务部署、监控诊断等任务",
        Instruction: platformAgentInstruction,
        Model:       chatModel,
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: allTools,
            },
        },
    })
}
```

### 3. CheckPointStore 实现

```go
// internal/ai/checkpoint_store.go

package ai

import (
    "context"
    "encoding/json"
    "time"

    "github.com/cloudwego/eino/compose"
    "github.com/redis/go-redis/v9"
)

// InMemoryCheckPointStore 内存存储（本地开发/测试）
type InMemoryCheckPointStore struct {
    data map[string][]byte
}

func NewInMemoryCheckPointStore() *InMemoryCheckPointStore {
    return &InMemoryCheckPointStore{data: make(map[string][]byte)}
}

func (s *InMemoryCheckPointStore) Set(ctx context.Context, key string, value []byte) error {
    s.data[key] = value
    return nil
}

func (s *InMemoryCheckPointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
    v, ok := s.data[key]
    return v, ok, nil
}

// RedisCheckPointStore Redis 存储（生产环境）
type RedisCheckPointStore struct {
    client *redis.Client
    ttl    time.Duration
}

func NewRedisCheckPointStore(client *redis.Client) *RedisCheckPointStore {
    return &RedisCheckPointStore{
        client: client,
        ttl:    30 * time.Minute, // 检查点过期时间
    }
}

func (s *RedisCheckPointStore) Set(ctx context.Context, key string, value []byte) error {
    return s.client.Set(ctx, "checkpoint:"+key, value, s.ttl).Err()
}

func (s *RedisCheckPointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
    val, err := s.client.Get(ctx, "checkpoint:"+key).Bytes()
    if err == redis.Nil {
        return nil, false, nil
    }
    if err != nil {
        return nil, false, err
    }
    return val, true, nil
}
```

### 4. Chat Handler 调整

```go
// internal/service/ai/chat_handler.go (关键改动部分)

func (h *handler) chatWithADK(c *gin.Context, req chatRequest, uid uint64, msg string) {
    // ... 初始化 SSE ...

    ctx := h.buildToolContext(c.Request.Context(), uid, "", scene, msg, runtime, emit, tracker)

    // 使用新的 Runner API
    iter := h.runner.Query(ctx, req.SessionID, msg)

    var assistantContent, reasoningContent strings.Builder

    for {
        event, ok := iter.Next()
        if !ok {
            break
        }

        if err := h.processADKEvent(emit, tracker, event, &assistantContent, &reasoningContent); err != nil {
            // 处理错误
            break
        }
    }

    // ... 后续处理 ...
}

// handleApprovalResponse 处理审批响应
func (h *handler) handleApprovalResponse(c *gin.Context) {
    var req struct {
        SessionID string `json:"session_id"`
        Approved  bool   `json:"approved"`
        Reason    string `json:"reason,omitempty"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        httpx.BindErr(c, err)
        return
    }

    ctx := c.Request.Context()

    // 构建审批结果
    var toolOpts []tool.Option
    if req.Approved {
        toolOpts = append(toolOpts, tools.WithApprovalResult(&tools.ApprovalResult{
            Approved: true,
        }))
    } else {
        toolOpts = append(toolOpts, tools.WithApprovalResult(&tools.ApprovalResult{
            Approved:         false,
            DisapproveReason: &req.Reason,
        }))
    }

    // 恢复执行
    iter, err := h.runner.Resume(ctx, req.SessionID, toolOpts)
    if err != nil {
        httpx.Fail(c, xcode.ServerError, err.Error())
        return
    }

    // 流式返回结果
    // ...
}
```

## File Changes Summary

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/ai/agent.go` | 重构 | 改用 ChatModelAgent，统一 Instruction |
| `internal/ai/runner.go` | 新增 | PlatformRunner 封装 |
| `internal/ai/checkpoint_store.go` | 新增 | 检查点存储实现 |
| `internal/ai/runtime_agent.go` | 删除 | 功能合并到 runner.go |
| `internal/service/ai/chat_handler.go` | 修改 | 使用新 Runner API |
| `internal/service/ai/routes.go` | 修改 | 新增审批响应路由 |

## Testing Strategy

1. **单元测试**
   - `runner_test.go`: 测试 Query/Resume 流程
   - `checkpoint_store_test.go`: 测试存储读写
   - `agent_test.go`: 测试 Agent 创建

2. **集成测试**
   - 端到端工具调用测试
   - 中断恢复流程测试
   - SSE 事件流测试

3. **手动验证**
   - 使用实际模型测试工具调用
   - 验证审批流程
