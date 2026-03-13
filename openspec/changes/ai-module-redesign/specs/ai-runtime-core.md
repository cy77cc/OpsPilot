# Spec: AI Runtime Core

## Overview

定义 AI 运行时核心能力，包括 Plan-Execute Agent 架构、Checkpoint 持久化、SSE 流式输出。

## Requirements

### REQ-RT-001: Plan-Execute Agent 架构

系统 SHALL 使用 Plan-Execute-Replanner 模式执行 AI 对话。

**验收标准:**
- GIVEN 用户发送对话请求
- WHEN 请求进入 Runtime
- THEN 系统依次执行 Planner → Executor → Replanner
- AND 最大迭代次数为 20 次

### REQ-RT-002: Redis Checkpoint Store

系统 SHALL 使用 Redis 持久化 Agent 执行状态。

**验收标准:**
- GIVEN Agent 执行过程中产生中断
- WHEN 需要恢复执行
- THEN 系统能从 Redis 恢复执行状态
- AND Checkpoint TTL 为 24 小时

### REQ-RT-003: SSE 流式输出

系统 SHALL 通过 SSE 流式输出 Agent 执行事件。

**验收标准:**
- GIVEN 用户发送对话请求
- WHEN Agent 开始执行
- THEN 系统通过 SSE 流式输出事件
- AND 支持的事件类型包括: meta, delta, thinking_delta, tool_call, tool_result, approval_required, done, error

### REQ-RT-004: 场景上下文注入

系统 SHALL 支持注入页面级场景上下文。

**验收标准:**
- GIVEN 请求包含 scene, currentPage, selectedResources
- WHEN Agent 执行工具调用
- THEN 上下文信息对工具可见
- AND 工具可以基于上下文做出智能决策

## Interfaces

### Runtime Interface

```go
type Runtime interface {
    Run(ctx context.Context, req RunRequest, emit StreamEmitter) error
    Resume(ctx context.Context, req ResumeRequest) (*ResumeResult, error)
    ResumeStream(ctx context.Context, req ResumeRequest, emit StreamEmitter) (*ResumeResult, error)
}
```

### RunRequest

```go
type RunRequest struct {
    SessionID      string         // 会话 ID
    Message        string         // 用户消息
    RuntimeContext RuntimeContext // 运行时上下文
}

type RuntimeContext struct {
    Scene             string              // 场景标识
    Route             string              // 路由路径
    ProjectID         string              // 项目 ID
    CurrentPage       string              // 当前页面
    SelectedResources []SelectedResource  // 选中的资源
    UserContext       map[string]any      // 用户上下文
    Metadata          map[string]any      // 其他元数据
}
```

### StreamEvent

```go
type StreamEvent struct {
    Type    EventType
    Data    map[string]any
}
```

## Dependencies

- Eino ADK (`github.com/cloudwego/eino/adk`)
- Redis (`github.com/redis/go-redis/v9`)
- Gin (`github.com/gin-gonic/gin`)
