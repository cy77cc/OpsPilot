## MODIFIED Requirements

### Requirement: Agent Architecture

AI 助手 MUST 使用 eino ADK 标准架构构建，采用 Plan-Execute-Replan 模式，并且该编排架构的宿主 MUST 位于 `internal/ai` 中，而不是位于 HTTP handler 本地流程中。

```
┌─────────────────────────────────────────────────────────────┐
│                    AI Core Orchestrator                     │
│                     (hosted in internal/ai)                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐              │
│  │ Planner  │───▶│ Executor │───▶│Replanner │──┐           │
│  └──────────┘    └──────────┘    └──────────┘  │           │
│       │              │                           │           │
│       │              ▼                           │           │
│       │        ┌──────────┐                      │           │
│       │        │  Tools   │                      │           │
│       │        └──────────┘                      │           │
│       │              │                           │           │
│       └──────────────┴───────────────────────────┘           │
│                      │                                       │
│                      ▼                                       │
│              ┌──────────────┐                               │
│              │ CheckPoint   │                               │
│              │    Store     │                               │
│              └──────────────┘                               │
│                                                              │
└─────────────────────────────────────────────────────────────┘
                     ▲
                     │ delegate
┌─────────────────────────────────────────────────────────────┐
│                     AI Gateway Layer                         │
│       (internal/service/ai routes/handlers/SSE transport)   │
└─────────────────────────────────────────────────────────────┘
```

**Acceptance Criteria:**
- [ ] 使用 `planexecute.New()` 创建 Agent
- [ ] 支持 Planner/Executor/Replanner 三个组件
- [ ] 最大迭代次数可配置（默认 20）
- [ ] `internal/service/ai` 通过委托 AI core 触发 ADK 编排，而不是在 handler 中本地承载编排所有权

#### Scenario: orchestration ownership lives in the AI core
- **WHEN** reviewers inspect the ADK architecture
- **THEN** the Plan-Execute-Replan orchestration host MUST be defined under `internal/ai`
- **AND** gateway handlers MUST delegate into the AI core instead of acting as the orchestration owner

### Requirement: SSE Event Format

SSE 事件格式 MUST 与现有前端兼容，并且事件语义的来源 MUST 位于 AI core 中，gateway SHALL 仅负责传输封装与兼容序列化。

**Event Types:**
| Event | Description |
|-------|-------------|
| `delta` | Streaming content chunk |
| `approval_required` | Tool needs approval |
| `review_required` | Tool needs parameter review |
| `tool_call` | Tool invocation |
| `tool_result` | Tool result |
| `error` | Error occurred |
| `done` | Execution complete |

**Approval Event Payload:**
```json
{
    "tool": "host_batch_exec_apply",
    "arguments": "{\"host_ids\":[1,2,3],\"command\":\"df -h\"}",
    "risk": "high",
    "preview": {
        "target_count": 3,
        "resolved_hosts": ["host1", "host2", "host3"]
    }
}
```

**Acceptance Criteria:**
- [ ] 事件格式与现有实现一致
- [ ] 前端无需改动
- [ ] 事件的业务语义由 `internal/ai` 产生，transport 兼容包装由 gateway 负责

#### Scenario: gateway preserves transport compatibility while AI core owns event meaning
- **WHEN** AI execution events are streamed to the frontend
- **THEN** the SSE family MUST remain compatible with the current frontend
- **AND** the semantic meaning of execution, interrupt, and progress events MUST be produced by the AI core

### Requirement: HTTP Handler

HTTP 处理器 MUST 作为 AI gateway 使用 ADK orchestration entrypoint，代码量控制在 200 行以内，并且 MUST NOT 继续在 handler 内承载主要编排逻辑。

**Handler Pattern:**
```go
func (h *handler) chat(c *gin.Context) {
    result := h.aiCore.RunChat(ctx, request)
    h.streamResult(c, result)
}
```

**Acceptance Criteria:**
- [ ] handler 负责 request bind、auth/session shell、SSE transport 与错误收尾
- [ ] handler 通过 AI core orchestration entrypoint 执行 chat/resume 流程
- [ ] handler 不直接成为 Planner/Executor/Replanner 的宿主

#### Scenario: handler acts as gateway rather than orchestration owner
- **WHEN** a chat or resume request is processed
- **THEN** the HTTP handler MUST delegate orchestration to the AI core
- **AND** the handler MUST remain limited to gateway and transport responsibilities
