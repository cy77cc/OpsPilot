# 任务清单

## 概述

本文档记录 Plan-Execute-Replan 思考过程可视化功能的开发任务。

说明：本任务已按 cleanup 后的 AI 主路径重映射执行，具体落点见
`docs/refactor/plan-execute-replan-implementation-remap.md`。原文中的
`useAIChat / ThoughtChain* / ThinkingProcessPanel*` 文件级要求，语义上已
收口到 `Copilot.tsx -> turnLifecycle.ts -> messageBlocks.ts ->
AssistantMessageBlocks.tsx`。

**预估总工时**: 11-12 小时

---

## Phase 1: 后端事件增强

**预估工时**: 3 小时

### 1.1 事件定义

- [x] `internal/ai/events/events.go`: 新增事件常量
  - `EventPhaseStarted`
  - `EventPhaseComplete`
  - `EventPlanGenerated`
  - `EventStepStarted`
  - `EventStepComplete`
  - `EventReplanTriggered`
  - `EventReplanComplete`

### 1.2 数据结构

- [x] `internal/ai/runtime/runtime.go`: 新增数据结构
  - `PlanStep` 结构体
  - `PhaseInfo` 结构体
  - `StepInfo` 结构体
  - `ToolCallInfo` 结构体
  - `ToolResultInfo` 结构体
  - `ReplanInfo` 结构体

### 1.3 SSE 转换器

- [x] `internal/ai/runtime/sse_converter.go`: 新增转换方法
  - `OnPhaseStarted(phase, title string) StreamEvent`
  - `OnPhaseComplete(phase, status string) StreamEvent`
  - `OnPlanGenerated(planID string, steps []PlanStep) []StreamEvent`
  - `OnStepStarted(info *StepInfo) StreamEvent`
  - `OnToolCall(info *ToolCallInfo) StreamEvent`
  - `OnToolResult(info *ToolResultInfo) StreamEvent`
  - `OnStepComplete(stepID, status, summary string) StreamEvent`
  - `OnReplanTriggered(info *ReplanInfo) StreamEvent`

### 1.4 阶段检测器

- [x] `internal/ai/runtime/phase_detector.go`: 新建文件
  - `PhaseDetector` 结构体
  - `Detect(event *adk.AgentEvent) string` 方法
  - `NextStepID() string` 方法

### 1.5 计划解析器

- [x] `internal/ai/runtime/plan_parser.go`: 新建文件
  - `PlanParser` 结构体
  - `Parse(event *adk.AgentEvent) *ParsedPlan` 方法
  - JSON 格式解析
  - 编号列表解析

### 1.6 Orchestrator 增强

- [x] `internal/ai/orchestrator.go`: 修改 streamExecution
  - 集成 PhaseDetector
  - 集成 PlanParser
  - 发送阶段事件
  - 发送步骤事件
  - 发送工具事件

### 1.7 单元测试

- [x] `internal/ai/runtime/phase_detector_test.go`: 新建测试
- [x] `internal/ai/runtime/plan_parser_test.go`: 新建测试

---

## Phase 2: 前端状态管理

**预估工时**: 2 小时

### 2.1 类型定义

- [x] `web/src/components/AI/types.ts`: 新增类型
  - `PlanStep` 接口
  - `ToolExecution` 接口
  - 以 `ChatTurn` / `TurnBlock` / block payload 承载过程可视化语义
  - `ApprovalRequest` 接口扩展
  - SSE 事件类型: `SSEPhaseStartedEvent`, `SSEPlanGeneratedEvent` 等

### 2.2 事件处理器

- [x] `web/src/components/AI/Copilot.tsx` + `turnLifecycle.ts`: 新增处理器
  - `applyPhaseStarted`
  - `applyPlanGenerated`
  - `applyStepStarted`
  - `applyStepComplete`
  - `applyReplanTriggered`
  - tool/approval 事件映射到 turn/block 主路径

### 2.3 辅助函数

- [x] `web/src/components/AI/turnLifecycle.ts` + `messageBlocks.ts`: 新增辅助函数
  - phase/step/replan 到 block 的 reducer 语义
  - plan/tool/status block payload 归一化

### 2.4 SSE API 扩展

- [x] `web/src/api/modules/ai.ts`: 新增 SSE 事件类型
- [x] `web/src/api/modules/ai.ts`: 扩展 `AIChatStreamHandlers` 接口

---

## Phase 3: UI 组件实现

**预估工时**: 4 小时

### 3.1 思考过程面板

- [x] `web/src/components/AI/components/AssistantMessageBlocks.tsx`: 以 block 面板承载过程可视化
  - 使用现有 assistant block 渲染主容器
  - 统一承载 plan/status/tool/approval 展示

### 3.2 阶段时间线

- [x] `web/src/components/AI/components/AssistantMessageBlocks.tsx`: 通过 status/plan/tool block 顺序表达阶段
  - 渲染各阶段状态图标与元信息
  - 以现有 block 结构替代独立 timeline 容器

### 3.3 步骤列表

- [x] `web/src/components/AI/components/AssistantMessageBlocks.tsx`: 渲染 plan steps 列表
  - 展示步骤状态图标
  - 展示 tool hint / tool descriptor

### 3.4 工具执行时间线

- [x] `web/src/components/AI/components/ToolCard.tsx`: 渲染工具执行卡片
  - 显示工具名称、参数、结果
  - 状态图标与摘要展示

### 3.5 审批弹窗

- [x] `web/src/components/AI/components/ConfirmationPanel.tsx`: 审批确认 UI 已接入现有主路径
  - 风险等级标签
  - 详情展示
  - 确认/取消按钮

### 3.6 样式文件

- [x] 过程可视化样式已并入现有 block 组件
  - status/plan/tool/approval 块样式已增强
  - 未新增独立 `thinking-process.css`

---

## Phase 4: 集成与测试

**预估工时**: 2 小时

### 4.1 组件集成

- [x] `web/src/components/AI/Copilot.tsx`: 集成 turn/block 主路径
  - 注册新的 phase/plan/step/replan/tool 事件
  - 修改 AssistantMessage 渲染逻辑，优先展示 turn blocks

### 4.2 后端集成测试

- [x] 启动后端服务
- [x] 测试 SSE 事件流
- [x] 验证事件序列正确

### 4.3 前端集成测试

- [x] 启动前端服务
- [x] 测试 UI 渲染
- [x] 验证步骤状态更新
- [x] 验证审批弹窗功能

### 4.4 样式调整

- [x] 调整响应式布局
- [x] 调整颜色和间距
- [ ] 测试暗色主题

### 4.5 文档更新

- [x] 更新实现重映射文档
- [x] 更新 README
- [x] 更新 API 文档

---

## 验收检查

### 功能验收

- [x] 前端能正确显示 Planning/Executing/Replanning 阶段
- [x] 步骤列表能实时更新状态 (pending → running → completed)
- [x] 工具调用参数和结果能正确展示
- [x] 审批弹窗能正常工作
- [x] 用户批准/拒绝后流程能继续

### 性能验收

- [ ] SSE 事件延迟 < 100ms
- [ ] 前端渲染无明显卡顿
- [ ] 内存无泄漏

### 代码验收

- [x] 后端新增代码有单元测试
- [x] 前端组件有 TypeScript 类型检查
- [x] 代码符合项目规范

---

## 风险与缓解

| 风险 | 缓解措施 | 负责人 |
|------|---------|--------|
| Planner 输出格式不稳定 | 使用 structured output 或明确 Prompt 格式要求 | 后端 |
| RunPath 格式变化 | 多重检测手段 (AgentName + 消息内容特征) | 后端 |
| SSE 事件过多 | 节流处理，合并高频事件 | 前端 |
| 审批流程卡死 | 添加超时机制和重试按钮 | 前后端 |

---

## 时间线

| 阶段 | 开始日期 | 结束日期 | 状态 |
|------|---------|---------|------|
| Phase 1: 后端 | 2026-03-14 | 2026-03-14 | 已完成 |
| Phase 2: 前端状态 | 2026-03-14 | 2026-03-14 | 已完成 |
| Phase 3: UI 组件 | 2026-03-14 | 2026-03-14 | 已完成 |
| Phase 4: 集成测试 | 2026-03-14 | - | 进行中 |

---

## 相关文档

- [proposal.md](./proposal.md) - 提案文档
- [design-backend.md](./design-backend.md) - 后端实现方案
- [design-frontend.md](./design-frontend.md) - 前端实现方案
- [specs/ai-streaming-events/spec.md](./specs/ai-streaming-events/spec.md) - SSE 事件规范变更
- [specs/ai-runtime-core/spec.md](./specs/ai-runtime-core/spec.md) - Runtime 规范变更
