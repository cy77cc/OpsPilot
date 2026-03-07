# AI Assistant ADK Architecture

## Capability Overview

定义基于 eino ADK 的 AI 助手架构能力，包括 Plan-Execute-Replan 模式、StatefulInterrupt 审批机制、CheckPointStore 持久化。

## Capability Specification

### 1. Agent Architecture

**Requirement:** AI 助手 MUST 使用 eino ADK 标准架构构建，采用 Plan-Execute-Replan 模式，并且该编排架构的宿主 MUST 位于 `internal/ai` 中，而不是位于 HTTP handler 本地流程中。

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

### 2. Tool Approval Mechanism

**Requirement:** 高风险工具必须使用 `tool.StatefulInterrupt` 实现标准的审批中断流程。

**Approval Flow:**
```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  Tool   │────▶│Interrupt│────▶│  User   │────▶│ Resume  │
│  Call   │     │  Save   │     │ Approve │     │Execute  │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
                      │
                      ▼
               ┌─────────────┐
               │CheckPoint   │
               │   Store     │
               └─────────────┘
```

**Required Types:**
```go
type ApprovalInfo struct {
    ToolName        string         `json:"tool_name"`
    ArgumentsInJSON string         `json:"arguments"`
    Risk            ToolRisk       `json:"risk"`
    Preview         map[string]any `json:"preview"`
}

type ApprovalResult struct {
    Approved         bool    `json:"approved"`
    DisapproveReason *string `json:"disapprove_reason,omitempty"`
}
```

**Acceptance Criteria:**
- [ ] `ApprovalInfo` 和 `ApprovalResult` 已注册 schema
- [ ] 高风险工具使用 `ApprovableTool` 包装
- [ ] 中断状态保存到 CheckPointStore
- [ ] 恢复时正确读取审批结果

### 3. Tool Review Mechanism

**Requirement:** 中风险工具必须使用 `tool.StatefulInterrupt` 实现参数审核编辑流程。

**Required Types:**
```go
type ReviewEditInfo struct {
    ToolName        string `json:"tool_name"`
    ArgumentsInJSON string `json:"arguments"`
}

type ReviewEditResult struct {
    EditedArgumentsInJSON *string `json:"edited_arguments,omitempty"`
    NoNeedToEdit          bool    `json:"no_need_to_edit"`
    Disapproved           bool    `json:"disapproved"`
    DisapproveReason      *string `json:"disapprove_reason,omitempty"`
}
```

**Acceptance Criteria:**
- [ ] `ReviewEditInfo` 和 `ReviewEditResult` 已注册 schema
- [ ] 中风险工具使用 `ReviewableTool` 包装
- [ ] 用户可编辑参数后执行
- [ ] 用户可拒绝执行

### 4. CheckPoint Persistence

**Requirement:** 必须实现 `compose.CheckPointStore` 接口，使用数据库持久化。

**Interface:**
```go
type CheckPointStore interface {
    Set(ctx context.Context, key string, value []byte) error
    Get(ctx context.Context, key string) ([]byte, bool, error)
}
```

**Database Schema:**
```sql
CREATE TABLE ai_checkpoints (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `key` VARCHAR(255) NOT NULL UNIQUE,
    value MEDIUMBLOB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_key (`key`)
);
```

**Acceptance Criteria:**
- [ ] 实现 `DBCheckPointStore` 结构体
- [ ] Set 方法支持 upsert
- [ ] Get 方法正确处理 not found 场景
- [ ] 支持并发访问

### 5. Tool Risk Classification

**Requirement:** 所有工具必须明确风险等级，并应用对应包装器。

**Risk Levels:**
| Level | Wrapper | Behavior |
|-------|---------|----------|
| Low | None | Direct execution |
| Medium | ReviewableTool | User can edit parameters |
| High | ApprovableTool | Requires explicit approval |

**Tool Classification:**
```go
type ToolRisk string

const (
    ToolRiskLow    ToolRisk = "low"
    ToolRiskMedium ToolRisk = "medium"
    ToolRiskHigh   ToolRisk = "high"
)
```

**Acceptance Criteria:**
- [ ] 所有工具定义风险等级
- [ ] 高风险工具有预览函数
- [ ] 构建函数正确应用包装器

### 6. SSE Event Format

**Requirement:** SSE 事件格式 MUST 与现有前端兼容，并且事件语义的来源 MUST 位于 AI core 中，gateway SHALL 仅负责传输封装与兼容序列化。

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

### 7. HTTP Handler

**Requirement:** HTTP 处理器 MUST 作为 AI gateway 使用 ADK orchestration entrypoint，代码量控制在 200 行以内，并且 MUST NOT 继续在 handler 内承载主要编排逻辑。

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

## Integration Points

### With Existing Systems

| System | Integration |
|--------|-------------|
| Database | `ai_checkpoints` table, existing session tables |
| RBAC | Tool permission checking |
| Audit | Command execution logging |
| MCP | MCP proxy tools |

### Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| github.com/cloudwego/eino | latest | ADK framework |
| github.com/cloudwego/eino-ext | latest | Model providers |

## Testing Requirements

### Unit Tests
- [ ] `ApprovableTool` 中断恢复逻辑
- [ ] `ReviewableTool` 参数编辑逻辑
- [ ] `DBCheckPointStore` 读写操作
- [ ] 工具风险分级验证

### Integration Tests
- [ ] Plan-Execute-Replan 完整流程
- [ ] 审批中断恢复流程
- [ ] SSE 事件流验证

### End-to-End Tests
- [ ] 用户对话完整流程
- [ ] 审批确认交互流程

## Migration Notes

### Breaking Changes
- 无 API 破坏性变更
- SSE 事件格式保持兼容

### Data Migration
- 新增 `ai_checkpoints` 表
- 现有会话数据无需迁移

### Rollback Strategy
- 功能开关控制新旧实现
- CheckPoint 表保留不影响旧逻辑
