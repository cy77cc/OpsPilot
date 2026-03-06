# Proposal: 重构 AI 助手模块 - 使用 eino 官方推荐模式

## Summary

重构 AI 助手模块，使用 eino 官方推荐的 `ChatModelAgent + Runner` 模式替代当前的 `planexecute` 模式，解决工具调用问题并完善中断恢复机制。

## Motivation

### 问题背景

当前 AI 助手使用 `planexecute` 模式（Planner → Executor → Replanner），存在以下问题：

1. **工具调用失败**: AI 经常输出 JSON 格式的"步骤计划"而非实际执行工具调用
   ```
   用户请求: "输出一条信息到火山云服务器..."
   AI 输出: {"steps": ["调用 host_list_inventory(keyword='火山云服务器')..."]}
   ```
   模型认为需要先"规划"，而非直接执行工具。

2. **中断恢复机制分散**: 当前使用自定义的 `ConfirmationService` + 数据库轮询，与 eino 原生的 `Runner` + `CheckPointStore` 机制不一致。

3. **System Prompt 分散**: 没有统一的 Agent Instruction，导致模型行为不一致。

### 解决方案

采用 eino 官方推荐的模式：

```
当前架构:                          目标架构:
┌─────────────────────┐           ┌─────────────────────┐
│ planexecute         │           │ ChatModelAgent      │
│ ┌─────────────────┐ │           │ ┌─────────────────┐ │
│ │ Planner         │ │           │ │ Instruction     │ │
│ └────────┬────────┘ │           │ │ (统一 Prompt)   │ │
│          ▼          │    ───>   │ └────────┬────────┘ │
│ ┌─────────────────┐ │           │          ▼          │
│ │ Executor        │ │           │ ┌─────────────────┐ │
│ └────────┬────────┘ │           │ │ Runner          │ │
│          ▼          │           │ │ (中断恢复)      │ │
│ ┌─────────────────┐ │           │ └─────────────────┘ │
│ │ Replanner       │ │           └─────────────────────┘
│ └─────────────────┘ │
└─────────────────────┘
```

## Scope

### In Scope

- 重构 Agent 创建逻辑（`internal/ai/agent.go`）
- 新增 Runner 封装层（`internal/ai/runner.go`）
- 实现 CheckPointStore（支持 Redis/内存存储）
- 调整 Chat Handler 使用新 API
- 统一 Agent Instruction 定义

### Out of Scope

- 前端重构（保持现有风格）
- 工具定义修改（保持现有工具不变）
- 审批 UI 完善（后续独立变更）

## Key Decisions

| 决策点 | 选择 | 理由 |
|--------|------|------|
| Agent 类型 | `adk.NewChatModelAgent` | 官方推荐，工具调用更可靠 |
| 执行管理 | `adk.Runner` | 原生支持中断恢复 |
| 检查点存储 | Redis + 内存降级 | 支持分布式部署，本地开发降级为内存 |
| Instruction 管理 | 集中定义在 Agent 配置 | 模型行为一致性 |

## Success Criteria

1. AI 正确执行工具调用，不再输出"计划 JSON"
2. 中断恢复流程正常工作（审批 → 恢复 → 继续）
3. 现有功能保持兼容（SSE 事件流、工具追踪等）
4. 单元测试覆盖核心逻辑

## Risks & Mitigations

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 模型兼容性 | 某些模型可能不支持 ChatModelAgent | 保留 planexecute 作为 fallback |
| 检查点存储可靠性 | Redis 不可用时影响中断恢复 | 实现内存存储降级 |
| 现有行为变更 | 用户感知的响应可能变化 | 充分测试，渐进发布 |

## Timeline

- Phase 1: 核心 Agent 重构 (1-2 天)
- Phase 2: Runner + CheckPointStore 实现 (1 天)
- Phase 3: Handler 集成 + 测试 (1 天)
- Phase 4: 验证 + 文档 (0.5 天)

## References

- eino-examples: `eino-examples/adk/intro/chatmodel/`
- eino-examples: `eino-examples/adk/intro/http-sse-service/`
- 当前实现: `internal/ai/agent.go`, `internal/service/ai/chat_handler.go`
