# 实现计划：AI 对话 Runtime 持久化与懒加载

**关联设计文档**: `docs/superpowers/specs/2026-03-18-ai-message-runtime-persistence-design.md`
**创建日期**: 2026-03-18

## 概述

实现 AI 对话运行时状态的持久化存储，并支持前端按需懒加载，解决历史对话显示效果与实时对话不一致的问题。

## 任务列表

### Phase 1: 后端数据模型与类型定义

- [ ] **Task 1.1**: 在 `internal/model/ai.go` 增加 `RuntimeJSON` 字段
  - 在 `AIChatMessage` 结构体添加 `RuntimeJSON string` 字段
  - 添加 GORM 标签：`gorm:"column:runtime_json;type:longtext"`
  - JSON 标签设为 `json:"-"` 避免在普通响应中返回

- [ ] **Task 1.2**: 创建 `internal/ai/runtime/types.go` 定义持久化类型
  - 定义 `PersistedRuntime` 结构体
  - 定义 `PersistedPlan`, `PersistedStep`, `PersistedActivity`, `PersistedSummary`, `PersistedStatus` 结构体
  - 添加完整的中文注释

### Phase 2: 后端状态跟踪与存储

- [ ] **Task 2.1**: 扩展 `ProjectionState` 添加持久化字段
  - 在 `internal/ai/runtime/project.go` 的 `ProjectionState` 添加 `Persisted *PersistedRuntime` 字段
  - 修改 `NewStreamProjector()` 初始化 `Persisted`

- [ ] **Task 2.2**: 实现 `GetPersistedState()` 方法
  - 在 `StreamProjector` 添加 `GetPersistedState() *PersistedRuntime` 方法

- [ ] **Task 2.3**: 修改 `projectNormalizedEvent` 更新持久化状态
  - 在 `NormalizedKindHandoff` 分支更新 `Phase` 和添加活动
  - 在 `NormalizedKindInterrupt` 分支添加审批活动
  - 在 `NormalizedKindToolCall` 分支添加工具调用活动
  - 在 `NormalizedKindToolResult` 分支更新活动状态
  - 在 `NormalizedKindMessage` 分支更新 plan

- [ ] **Task 2.4**: 添加 `buildPersistedPlanFromSteps` 辅助函数
  - 从步骤字符串数组构建 `PersistedPlan`

- [ ] **Task 2.5**: 修改 `Finish()` 设置最终状态
  - 设置 `Phase = "completed"`
  - 设置 `Status = { Kind: "completed", Label: "已生成" }`
  - 清除 `ActiveStepIndex` 并标记所有步骤为 done

- [ ] **Task 2.6**: 修改 `logic.Chat` Step 7 保存 runtime
  - 在持久化结果时调用 `projector.GetPersistedState()`
  - 序列化并存储到 `runtime_json` 字段

### Phase 3: 后端 API 实现

- [ ] **Task 3.1**: 在 `internal/dao/ai/chat.go` 新增 `GetMessage` 方法
  - 根据 ID 查询单条消息
  - 处理 `gorm.ErrRecordNotFound` 返回 nil

- [ ] **Task 3.2**: 在 `internal/service/ai/logic/logic.go` 新增 `GetMessageWithOwnership` 方法
  - 调用 `ChatDAO.GetMessage` 获取消息
  - 验证会话所有权
  - 返回消息或 nil（无权限）

- [ ] **Task 3.3**: 在 `internal/service/ai/handler/session.go` 新增 `GetMessageRuntime` handler
  - 定义 `MessageRuntimeResponse` 响应结构体
  - 调用 `logic.GetMessageWithOwnership`
  - 解析 `runtime_json` 并返回
  - 使用 `httpx.OK()` 返回响应

- [ ] **Task 3.4**: 修改 `GetSession` 返回 `has_runtime` 标志
  - 在消息响应中添加 `has_runtime` 字段
  - 基于 `runtime_json != ""` 计算

- [ ] **Task 3.5**: 在 `internal/service/ai/routes.go` 注册路由
  - 添加 `messages.GET("/:id/runtime", h.GetMessageRuntime)`

### Phase 4: 前端 API 与类型定义

- [ ] **Task 4.1**: 在 `web/src/api/modules/ai.ts` 增加 `getMessageRuntime` API
  - 定义响应类型
  - 调用 `apiService.get('/ai/messages/${id}/runtime')`

- [ ] **Task 4.2**: 在 `web/src/components/AI/types.ts` 扩展 `XChatMessage` 类型
  - 添加 `id?: string` 字段
  - 添加 `hasRuntime?: boolean` 字段

### Phase 5: 前端组件改造

- [ ] **Task 5.1**: 修改 `web/src/components/AI/historyRuntime.ts`
  - 添加 `MAX_CACHE_SIZE = 50` 常量
  - 添加 `runtimeCache` Map 和 `cacheOrder` 数组
  - 实现 `evictOldest()` LRU 淘汰函数
  - 实现 `loadMessageRuntime()` 懒加载函数
  - 修改 `hydrateAssistantHistoryMessage()` 传递 id 和 hasRuntime

- [ ] **Task 5.2**: 修改 `web/src/components/AI/AssistantReply.tsx`
  - 扩展 `AssistantReplyProps` 接口
  - 添加 `localRuntime`, `loading`, `expanded` 状态
  - 实现"展开详情"按钮和 loading skeleton
  - 处理懒加载场景

### Phase 6: 测试与验证

- [ ] **Task 6.1**: 编写后端单元测试
  - `PersistedRuntime` 序列化/反序列化测试
  - `GetPersistedState()` 测试
  - `GetMessageWithOwnership` 权限验证测试

- [ ] **Task 6.2**: 编写前端单元测试
  - `loadMessageRuntime` 缓存测试
  - `hydrateAssistantHistoryMessage` 测试

- [ ] **Task 6.3**: 集成测试
  - 新对话 runtime 存储验证
  - 历史对话懒加载验证
  - 边界情况测试（空 runtime、JSON 错误、权限）

## 依赖关系

```
Phase 1 (类型定义)
    ↓
Phase 2 (状态跟踪) → Phase 3 (API)
    ↓                      ↓
Phase 6 (测试) ← Phase 4 (前端类型) ← Phase 5 (前端组件)
```

## 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| `runtime_json` 数据量大 | 性能影响 | 前端懒加载，不随会话返回 |
| 状态跟踪遗漏 | 数据不完整 | 完善测试覆盖所有事件类型 |
| 前端缓存一致性 | 显示过时数据 | LRU 淘汰，考虑添加 TTL |

## 验收标准

1. 新对话完成后，数据库中 `runtime_json` 包含完整的 plan、activities、summary
2. 历史对话加载时，消息列表不包含 runtime 数据
3. 点击"展开详情"后，正确显示步骤折叠、活动记录、摘要
4. 缓存工作正常，避免重复请求
5. 所有单元测试和集成测试通过
