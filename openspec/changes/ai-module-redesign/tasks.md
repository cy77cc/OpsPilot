# Tasks: AI Module Redesign

## Phase 1: 核心运行时修复 (P0 - 阻塞编译)

- [x] 1.1 创建 `internal/ai/runtime/` 目录结构
  - runtime.go - Runtime 接口定义
  - context_processor.go - 上下文处理器（混合模式注入）
  - scene_resolver.go - 场景配置解析器（硬编码默认 + 数据库覆盖）
  - checkoint.go - Redis Checkpoint Store 实现

- [x] 1.2 实现 ContextProcessor
  - BuildPlannerInput() - 构建 Planner 输入
  - buildSystemPrompt() - 构建场景感知的 System Prompt
  - filterTools() - 根据场景筛选工具
  - selectExamples() - 选择相关示例

- [x] 1.3 实现 SceneConfigResolver（混合方案）
  - Resolve() - 解析两级场景 key (domain:sub)
  - applySubSceneRule() - 应用子场景规则
  - loadDomainConfig() - 加载一级场景配置
  - getSubSceneRule() - 获取子场景规则（硬编码）
  - merge() - 合并默认配置和数据库配置
  - WatchChanges() - 监听配置变更
  - ReloadCache() - 清除缓存

- [x] 1.4 增强 Planner
  - 创建 prompt.go - Planner Prompt 定义
  - 实现 GenInputFn - 注入场景上下文
  - 结构化输出生成（Plan JSON Schema）
  - 参数依赖拓扑排序

- [x] 1.5 增强 Executor
  - 更新 prompt.go - 增加场景感知和审批说明
  - 实现 GenInputFn - 注入场景上下文和约束
  - 传递 RuntimeContext 到工具执行

- [x] 1.6 实现 Replanner
  - 创建 prompt.go - Replanner Prompt 定义
  - 实现 GenInputFn - 注入执行上下文
  - 决策逻辑：submit_result / create_plan / continue

- [x] 1.7 修复 handler.go 编译错误
  - 移除对不存在的 aiv2 包的引用
  - 使用新的 runtime 包
  - 集成 ContextProcessor

- [x] 1.8 实现 SSE Event Converter（ThoughtChain 统一模型）
  - 创建 runtime/sse_converter.go
  - OnPlannerStart() - 发送 stage_delta(stage=plan, status=loading)
  - OnPlanCreated() - 发送 stage_delta(stage=plan, status=success, content)
  - OnToolCallStart() - 发送 step_update(status=loading)
  - OnToolResult() - 发送 step_update(status=success/error)
  - OnApprovalRequired() - 发送 stage_delta(user_action) + approval_required
  - OnApprovalResult() - 发送 stage_delta(user_action, success/abort)
  - OnTextDelta() - 发送 delta(content_chunk)
  - OnExecuteComplete() - 发送 stage_delta(execute, success)
  - OnDone() - 发送 done
  - OnError() - 发送 error(stage, message)

## Phase 2: 审批流程 (P0 - 核心功能)

- [x] 2.1 实现 ApprovalDecisionMaker
  - Decide() - 判断是否需要审批
  - applyPolicy() - 应用审批策略
  - matchSkipCondition() - 匹配跳过条件

- [x] 2.2 实现 ApprovalGate 中间件
  - 创建 tools/approval/gate.go
  - InvokableRun() - 工具包装执行
  - triggerInterrupt() - 触发审批中断
  - handleResume() - 处理恢复执行

- [x] 2.3 实现 SummaryRenderer
  - Render() - 渲染操作摘要
  - generateActionSummary() - 生成操作描述
  - 自定义模板支持

- [x] 2.4 工具元数据标注
  - 为现有工具添加 ToolMeta
  - 标注 mode (readonly/mutating)
  - 标注 risk (low/medium/high)
  - 标注 category 和 tags

- [x] 2.5 工具包装器注册
  - 更新 ToolRegistry
  - 变更工具自动包装 ApprovalGate
  - 存储元数据

## Phase 3: 接口补齐 (P0 - 前端依赖)

- [x] 3.1 GET /ai/capabilities - 工具能力列表
  - 创建 handler/capabilities.go
  - 从 Tool Registry 读取工具元数据
  - 返回前端兼容的 JSON 格式

- [x] 3.2 GET /ai/tools/:name/params/hints - 参数提示
  - 创建 handler/tool_hints.go
  - 实现 HintResolver
  - 支持静态、动态、远程三种模式
  - 参数依赖拓扑排序

- [x] 3.3 实现内置数据源
  - NamespaceDataSource
  - DeploymentDataSource
  - PodDataSource
  - HostDataSource
  - ClusterDataSource

- [x] 3.4 POST /ai/tools/preview - 工具预览
  - 创建 handler/tool_preview.go
  - 实现 dry-run 逻辑
  - 返回预期变更摘要

- [x] 3.5 POST /ai/tools/execute - 工具执行
  - 创建 handler/tool_execute.go
  - 验证 checkpoint_id
  - 创建执行记录
  - 异步执行工具

- [x] 3.6 GET /ai/executions/:id - 执行状态
  - 创建 handler/execution.go
  - 从 Redis/MySQL 查询执行状态
  - 返回执行结果或错误

- [x] 3.7 POST /ai/resume/step/stream - SSE 流式恢复
  - 创建 handler/resume_stream.go
  - 接收 checkpoint_id + approved
  - 调用 Runtime.ResumeWithParams()
  - 通过 SSE Event Converter 发送 stage_delta/step_update 事件
  - 返回 SSE 流

- [x] 3.8 POST /ai/approvals - 创建审批
  - 创建 handler/approval_create.go
  - 生成审批 ID 和 checkpoint_id
  - 持久化到 MySQL（包含 checkpoint_id）
  - 存储临时状态到 Redis

- [x] 3.9 GET /ai/approvals - 审批列表
  - 创建 handler/approval_list.go
  - 支持按 status 筛选
  - 分页查询

- [x] 3.10 GET /ai/approvals/:id - 审批详情
  - 创建 handler/approval_get.go
  - 返回审批完整信息（包含 checkpoint_id）

- [x] 3.11 POST /ai/approvals/:id/approve - 批准审批
  - 创建 handler/approval_approve.go
  - 更新审批状态
  - 创建执行记录
  - 异步调用 Runtime.ResumeWithParams()
  - 返回 execution_id 供轮询

- [x] 3.12 POST /ai/approvals/:id/reject - 拒绝审批
  - 创建 handler/approval_reject.go
  - 更新审批状态
  - 返回拒绝消息

- [x] 3.13 GET /ai/scene/:scene/tools - 场景工具
  - 创建 handler/scene_tools.go
  - 使用 SceneResolver 解析两级场景
  - 根据场景筛选可用工具
  - 返回工具描述和使用提示

- [x] 3.14 GET /ai/scene/:scene/prompts - 场景提示词
  - 创建 handler/scene_prompts.go
  - 从配置或数据库读取预定义提示词
  - 返回提示词列表

- [x] 3.15 场景配置管理接口
  - GET /ai/scene/configs - 获取所有场景配置
  - GET /ai/scene/configs/:scene - 获取指定场景配置
  - PUT /ai/scene/configs/:scene - 更新场景配置
  - DELETE /ai/scene/configs/:scene - 删除场景配置（恢复默认）

## Phase 3.5: 前端适配 (P0 - 前端依赖)

- [x] 3.16 更新前端类型定义
  - web/src/api/modules/ai.ts
  - AIInterruptApprovalResponse 改用 checkpoint_id
  - 添加 ApprovalRequiredEvent 类型（包含 checkpoint_id）
  - 添加 StageDeltaEvent, StepUpdateEvent 类型

- [x] 3.17 更新 useAIChat hook
  - handleApprovalRequired() 使用 checkpoint_id
  - confirmApproval() 传递 checkpoint_id
  - 添加 onStageDelta, onStepUpdate 事件处理

- [x] 3.18 重构 Copilot.tsx 事件处理（ThoughtChain 统一模型）
  - 删除 Turn/Block 相关事件处理（onTurnStarted, onBlockOpen 等）
  - 实现 onStageDelta - 更新 thoughtChain 阶段状态
  - 实现 onStepUpdate - 更新 execute 阶段的 details
  - 简化消息渲染逻辑
  - **UI 组件实现:**
    - ThoughtChainStageCard - 阶段卡片组件
    - ThoughtChainDetailItem - 工具调用详情组件
    - ApprovalConfirmationPanel - 审批确认面板
    - RecommendedActions - 推荐操作区域
  - **交互实现:**
    - 阶段执行中 → 卡片自动展开
    - 阶段完成后 → 卡片自动折叠
    - 用户可手动展开/折叠

- [x] 3.19 清理废弃代码
  - 删除 turnLifecycle.ts
  - 删除 messageBlocks.ts
  - 删除 AssistantMessageBlocks.tsx
  - 删除 ChatTurn, TurnBlock 类型定义
  - 简化 ChatMessage 类型（移除 turn 字段）

## Phase 4: 场景感知 (P1 - 体验优化)

- [x] 4.1 Scene 上下文注入中间件
  - 创建 middleware/scene_context.go
  - 从请求中提取 scene/route/page
  - 注入到 RuntimeContext

- [x] 4.2 页面级资源选择感知
  - 解析 selectedResources 参数
  - 传递给工具执行上下文
  - 工具可感知用户选中的资源

- [x] 4.3 项目上下文传递
  - 解析 X-Project-ID header
  - 传递给工具执行上下文
  - 工具可感知当前项目

- [x] 4.4 环境感知
  - 从项目配置或命名空间推断环境
  - 传递给 ApprovalDecisionMaker

## Phase 5: 可观测性 (P2 - 运维支持)

- [x] 5.1 执行日志记录
  - 记录工具调用参数和结果
  - 记录执行耗时
  - 存储到 ai_executions 表

- [x] 5.2 性能指标采集
  - Prometheus metrics
  - 工具调用次数、耗时、成功率
  - Agent 执行次数、耗时

- [x] 5.3 成本追踪
  - LLM token 使用量
  - 按会话统计成本

## Phase 6: 数据库迁移

- [x] 6.1 创建 ai_scene_configs 表
- [x] 6.2 创建 ai_approvals 表
- [x] 6.3 创建 ai_executions 表
- [x] 6.4 初始化默认场景配置数据

## Testing

- [x] T.1 Runtime 接口单元测试
- [x] T.2 ApprovalDecisionMaker 单元测试
- [x] T.3 ApprovalGate 中间件单元测试
- [x] T.4 Checkpoint Store 单元测试
- [x] T.5 SceneConfigResolver 单元测试
- [x] T.6 HintResolver 单元测试
- [x] T.7 SSE Event Converter 单元测试（ThoughtChain 事件生成）
- [x] T.8 SceneResolver 两级场景解析测试
- [x] T.9 SSE 流式对话集成测试（ThoughtChain 阶段流转）
- [x] T.10 Interrupt/Resume 流程集成测试
- [x] T.11 审批流程端到端测试
- [x] T.12 参数提示级联查询测试
- [x] T.13 前端 ThoughtChain 事件处理测试
- [x] T.14 前端 checkpoint_id 集成测试

## Dependencies

- Eino ADK: `github.com/cloudwego/eino/adk`
- Redis: `github.com/redis/go-redis/v9`
- MySQL: GORM
- Gin: HTTP framework
- Ristretto: 本地缓存 (`github.com/dgraph-io/ristretto`)

## Implementation Order

```
Phase 1 (P0) ─────────────────────────────────────────┐
│ 1.1 Directory Structure                              │
│ 1.2 ContextProcessor                                 │
│ 1.3 SceneConfigResolver (混合方案)                    │
│ 1.4 Planner Enhancement                              │
│ 1.5 Executor Enhancement                             │
│ 1.6 Replanner Implementation                         │
│ 1.7 Handler Fix                                      │
│ 1.8 SSE Event Converter (ThoughtChain)               │
└──────────────────────────────────────────────────────┘
                         │
                         ▼
Phase 2 (P0) ─────────────────────────────────────────┐
│ 2.1 ApprovalDecisionMaker                            │
│ 2.2 ApprovalGate                                     │
│ 2.3 SummaryRenderer                                  │
│ 2.4 Tool Meta Annotation                             │
│ 2.5 Tool Registry Update                             │
└──────────────────────────────────────────────────────┘
                         │
                         ▼
Phase 3 (P0) ─────────────────────────────────────────┐
│ 3.1-3.3 Capabilities & Hints                         │
│ 3.4-3.6 Tool Preview & Execute                       │
│ 3.7 Resume Stream (SSE)                              │
│ 3.8-3.12 Approval APIs                               │
│ 3.13-3.15 Scene APIs                                 │
└──────────────────────────────────────────────────────┘
                         │
                         ▼
Phase 3.5 (P0) ────────────────────────────────────────┐
│ 3.16 Frontend Type Definitions                       │
│ 3.17 useAIChat Hook Update                           │
│ 3.18 Copilot.tsx Refactor (ThoughtChain)             │
│ 3.19 Cleanup Turn/Block Code                         │
└──────────────────────────────────────────────────────┘
                         │
                         ▼
Phase 4-6 (P1-P2) ────────────────────────────────────┐
│ 4.1-4.4 Scene Context                                │
│ 5.1-5.3 Observability                                │
│ 6.1-6.4 Database Migration                           │
└──────────────────────────────────────────────────────┘
```
