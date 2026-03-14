# Proposal: Plan-Execute-Replan 思考过程可视化

## Summary

实现 AI 助手的"思考过程"可视化，展示 Plan-Execute-Replan 编排模式的完整执行流程，让用户能看到 AI 的规划、执行步骤、工具调用和动态调整过程。

## Motivation

### 1.1 问题陈述

当前 AI 助手在执行复杂任务时，用户只能看到最终的文本回复，无法了解 AI 的"思考过程"。这导致：

1. **信任问题**: 用户不知道 AI 做了什么操作，特别是涉及高危操作时
2. **调试困难**: 当 AI 执行出错时，难以定位问题发生在哪个阶段
3. **交互割裂**: 审批确认界面简单，无法展示完整的操作上下文

### 1.2 参考实现

CloudWeGo Eino ADK 提供了 `planexecute` 预构建模式，核心流程：

```
User Query → Planner (生成计划) → Executor (执行步骤) → Replanner (动态调整) → Result
                              ↓
                        Human-in-the-Loop (关键操作需审批)
```

测试输出示例位于: `/root/learn/eino-examples/adk/human-in-the-loop/6_plan-execute-replan/output.md`

### 1.3 目标效果

实现类似 ChatGPT Deep Research / Claude Thinking 的效果：

```
┌─────────────────────────────────────────────────────────────┐
│ ▼ 思考过程                                          [已完成] │
├─────────────────────────────────────────────────────────────┤
│  ✓ 意图识别                                                 │
│  ✓ 整理执行步骤 (3 步)                                       │
│  ● 执行步骤                                          运行中  │
│    ┌─────────────────────────────────────────────────────┐  │
│    │ ✓ 1. 获取 Pod 列表                     [k8s_list]  │  │
│    │ ● 2. 检查异常 Pod                     [k8s_describe]│  │
│    │ ○ 3. 重启异常 Pod (需确认)            [k8s_restart] │  │
│    └─────────────────────────────────────────────────────┘  │
│  ○ 等待确认                                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 技术方案

### 2.1 SSE 事件契约

前后端对齐的事件类型定义：

| 事件类型 | 阶段 | 数据字段 | UI 渲染 |
|---------|------|---------|--------|
| `phase_started` | 生命周期 | `phase`, `title`, `status` | 阶段标题 + Loading |
| `phase_complete` | 生命周期 | `phase`, `status` | 阶段完成标记 |
| `plan_generated` | Planning | `plan_id`, `steps[]` | 步骤列表 |
| `step_started` | Executing | `step_id`, `title`, `tool_name`, `params` | 步骤运行中 |
| `tool_call` | Executing | `step_id`, `tool_name`, `arguments` | 工具调用卡片 |
| `tool_result` | Executing | `step_id`, `tool_name`, `result`, `status` | 工具结果 |
| `step_complete` | Executing | `step_id`, `status`, `summary` | 步骤完成 |
| `replan_triggered` | Replanning | `reason`, `completed_steps` | 重规划提示 |
| `approval_required` | UserAction | `approval_id`, `tool_name`, `risk`, `summary` | 审批弹窗 |

### 2.2 完整事件流示例

```json
// 1. 元信息
{"type": "meta", "data": {"session_id": "sess-1", "plan_id": "plan-1"}}

// 2. 规划阶段
{"type": "phase_started", "data": {"phase": "planning", "title": "整理执行步骤"}}
{"type": "delta", "data": {"content_chunk": "正在分析需求..."}}
{"type": "plan_generated", "data": {"plan_id": "plan-1", "steps": [
  {"id": "step-1", "content": "获取 Pod 列表", "tool_hint": "k8s_list_pods"},
  {"id": "step-2", "content": "检查异常 Pod", "tool_hint": "k8s_describe_pod"},
  {"id": "step-3", "content": "重启异常 Pod", "tool_hint": "k8s_restart_pod"}
], "total": 3}}
{"type": "phase_complete", "data": {"phase": "planning", "status": "success"}}

// 3. 执行阶段
{"type": "phase_started", "data": {"phase": "executing", "title": "执行步骤"}}
{"type": "step_started", "data": {"step_id": "step-1", "title": "获取 Pod 列表", "tool_name": "k8s_list_pods"}}
{"type": "tool_call", "data": {"step_id": "step-1", "tool_name": "k8s_list_pods", "arguments": {"namespace": "default"}}}
{"type": "tool_result", "data": {"step_id": "step-1", "tool_name": "k8s_list_pods", "result": "3 pods found", "status": "success"}}
{"type": "step_complete", "data": {"step_id": "step-1", "status": "success", "summary": "发现 3 个 Pod"}}

// 4. 步骤 2 执行...
// 5. 步骤 3 需要审批
{"type": "step_started", "data": {"step_id": "step-3", "title": "重启异常 Pod", "tool_name": "k8s_restart_pod"}}
{"type": "approval_required", "data": {"approval_id": "apr-1", "tool_name": "k8s_restart_pod", "risk": "high", "summary": "即将重启 Pod app-xxx"}}

// 6. 用户批准后继续
// 7. 完成
{"type": "done", "data": {"status": "completed"}}
```

### 2.3 架构图

```
┌─────────────────────────────────────────────────────────────────────┐
│                              Backend                                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐          │
│  │   Planner    │───▶│   Executor   │───▶│  Replanner   │          │
│  │  (生成计划)   │    │  (执行步骤)   │    │  (动态调整)   │          │
│  └──────────────┘    └──────────────┘    └──────────────┘          │
│         │                   │                    │                  │
│         └───────────────────┼────────────────────┘                  │
│                             ▼                                       │
│                  ┌──────────────────┐                               │
│                  │   Orchestrator   │                               │
│                  │  (事件解析+分发)  │                               │
│                  └────────┬─────────┘                               │
│                           │                                         │
│                           ▼                                         │
│                  ┌──────────────────┐                               │
│                  │  SSE Converter   │                               │
│                  │  (事件→SSE转换)  │                               │
│                  └────────┬─────────┘                               │
│                           │                                         │
└───────────────────────────┼─────────────────────────────────────────┘
                            │ SSE Stream
                            ▼
┌─────────────────────────────────────────────────────────────────────┐
│                             Frontend                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────┐                                               │
│  │   useAIChat      │ (状态管理)                                    │
│  │  Event Handlers  │                                               │
│  └────────┬─────────┘                                               │
│           │                                                          │
│           ▼                                                          │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                 ThinkingProcessPanel                          │   │
│  │  ┌────────────────────────────────────────────────────────┐  │   │
│  │  │                   StageTimeline                         │  │   │
│  │  │  ├─ RewriteStage (意图识别)                             │  │   │
│  │  │  ├─ PlanStage (规划)                                    │  │   │
│  │  │  │   └─ PlanStepsList (步骤列表)                        │  │   │
│  │  │  ├─ ExecuteStage (执行)                                 │  │   │
│  │  │  │   └─ ToolExecutionTimeline                           │  │   │
│  │  │  │       └─ ToolCard × N                                │  │   │
│  │  │  └─ UserActionStage (审阅)                              │  │   │
│  │  │      └─ ApprovalModal                                   │  │   │
│  │  └────────────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. 文件变更清单

### 3.1 后端变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/ai/events/events.go` | 修改 | 新增事件常量 |
| `internal/ai/runtime/sse_converter.go` | 修改 | 新增事件转换方法 |
| `internal/ai/orchestrator.go` | 修改 | 增强事件解析和发送 |
| `internal/ai/runtime/phase_detector.go` | 新建 | 阶段检测器 |
| `internal/ai/runtime/plan_parser.go` | 新建 | 计划解析器 |

### 3.2 前端变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `web/src/components/AI/types.ts` | 修改 | 新增类型定义 |
| `web/src/components/AI/hooks/useAIChat.ts` | 修改 | 新增事件处理器 |
| `web/src/components/AI/components/PlanStepsList.tsx` | 新建 | 步骤列表组件 |
| `web/src/components/AI/components/StageTimeline.tsx` | 新建 | 阶段时间线组件 |
| `web/src/components/AI/components/ThinkingProcessPanel.tsx` | 新建 | 思考过程面板 |
| `web/src/components/AI/components/ToolExecutionTimeline.tsx` | 新建 | 工具执行时间线 |
| `web/src/components/AI/components/ApprovalModal.tsx` | 新建 | 审批确认弹窗 |
| `web/src/components/AI/Copilot.tsx` | 修改 | 集成新组件 |

---

## 4. 实施计划

### Phase 1: 后端事件增强 (预估 3h)

- [ ] 1.1 在 `events.go` 新增事件常量
- [ ] 1.2 在 `sse_converter.go` 实现事件转换方法
- [ ] 1.3 实现 `phase_detector.go` 从 RunPath 推断阶段
- [ ] 1.4 修改 `orchestrator.go` 发送阶段事件
- [ ] 1.5 实现 `plan_parser.go` 解析结构化计划
- [ ] 1.6 后端单元测试

### Phase 2: 前端状态管理 (预估 2h)

- [ ] 2.1 扩展 `types.ts` 类型定义
- [ ] 2.2 在 `useAIChat.ts` 新增事件处理器
- [ ] 2.3 实现 `upsertThoughtStage` 支持 steps 更新

### Phase 3: UI 组件实现 (预估 4h)

- [ ] 3.1 实现 `PlanStepsList` 组件
- [ ] 3.2 实现 `StageTimeline` 组件
- [ ] 3.3 实现 `ThinkingProcessPanel` 组件
- [ ] 3.4 实现 `ToolExecutionTimeline` 组件
- [ ] 3.5 实现 `ApprovalModal` 组件

### Phase 4: 集成与测试 (预估 2h)

- [ ] 4.1 在 `Copilot.tsx` 集成新组件
- [ ] 4.2 端到端测试 SSE 事件流
- [ ] 4.3 UI 样式调整
- [ ] 4.4 文档更新

---

## 5. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| Planner 输出格式不稳定 | 无法解析结构化计划 | 使用 structured output 或明确 Prompt 格式要求 |
| RunPath 格式变化 | 阶段检测失败 | 多重检测手段 (AgentName + 消息内容特征) |
| SSE 事件过多 | 前端渲染性能 | 节流处理，合并高频事件 |

---

## 6. 验收标准

1. **功能验收**
   - [ ] 前端能正确显示 Planning/Executing/Replanning 阶段
   - [ ] 步骤列表能实时更新状态 (pending → running → completed)
   - [ ] 工具调用参数和结果能正确展示
   - [ ] 审批弹窗能正常工作

2. **性能验收**
   - [ ] SSE 事件延迟 < 100ms
   - [ ] 前端渲染无明显卡顿

3. **代码验收**
   - [ ] 后端新增代码有单元测试
   - [ ] 前端组件有 TypeScript 类型检查
