# Delta Spec: AI 助手模块

## 新增能力

### PlatformRunner API

```yaml
capability: ai.runner
version: 1.0.0
status: draft

operations:
  - name: Query
    description: 执行用户查询
    input:
      session_id: string (会话标识)
      message: string (用户消息)
      context: object (运行时上下文，可选)
    output:
      events: stream<AgentEvent>

  - name: Resume
    description: 恢复中断的执行
    input:
      session_id: string (会话标识)
      tool_options: array<tool.Option> (恢复选项)
    output:
      events: stream<AgentEvent>
```

### CheckPointStore 接口

```yaml
capability: ai.checkpoint
version: 1.0.0
status: draft

interface:
  - name: Set
    input:
      key: string
      value: bytes
    output: error

  - name: Get
    input:
      key: string
    output:
      value: bytes
      exists: bool
      error: error

implementations:
  - InMemoryCheckPointStore (本地开发/测试)
  - RedisCheckPointStore (生产环境)
```

## API 变更

### 新增路由

```
POST /api/ai/approval/respond
```

请求体:
```json
{
  "session_id": "string",
  "approved": true,
  "reason": "string (optional)"
}
```

响应:
```json
{
  "ok": true,
  "data": {
    "status": "resumed"
  }
}
```

## 废弃 API

以下内部 API 将被移除:

- `internal/ai.PlatformAgent.Stream` → 使用 `PlatformRunner.Query`
- `internal/ai.PlatformAgent.Generate` → 使用 `PlatformRunner.Query`
- `internal/service/ai.ConfirmationService` → 使用 `adk.Runner` 内置机制

## 兼容性

### SSE 事件格式 (保持不变)

```
event: delta
data: {"contentChunk": "..."}

event: tool_call
data: {"tool": "host_list_inventory", "call_id": "...", "payload": {...}}

event: tool_result
data: {"content": "..."}

event: approval_required
data: {"tool": "host_batch_exec_apply", "arguments": {...}, "risk": "high"}

event: done
data: {"reason": "agent_exit"}
```

### 工具定义 (保持不变)

现有工具定义 (`internal/ai/tools/tools_registry.go`) 无需修改。
