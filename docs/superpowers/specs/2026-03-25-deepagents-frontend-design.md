# DeepAgents 前端交互迁移设计（项目对齐版）

## 概述

### 背景

AI 后端从 Plan-Execute 刚性流程迁移到 DeepAgents 动态路由后，前端不能再假设“单一线性步骤流”。
当前项目已经具备 runtime 增量渲染、审批交互、步骤懒加载能力，因此本设计以**增量演进**为原则，避免重写聊天框架。

### 设计目标

1. 在不破坏现有 `Drawer + Bubble.List + runtime` 架构的前提下，支持动态多 Agent 流。
2. 保持 HITL 审批链路清晰、可追踪、可恢复。
3. 控制长对话/长日志场景下的前端性能风险。
4. 为 `write_ops_todos` 提供结构化可视化承载（Task Board）。

### 非目标

1. 不重写聊天基础组件（不替换 `useXChat` / `Bubble.List`）。
2. 不引入第二套平行消息模型（统一使用 `XChatMessage.runtime`）。
3. 本期不做全量消息虚拟列表重构。

---

## 现状约束（以代码为准）

### 现有关键实现

1. SSE 事件在 `ai.ts` 分发，`PlatformChatProvider` 汇总到 `runtime`。
2. `AssistantReply` 已支持：
   - plan/replan 可视化
   - tool approval 交互
   - completed step 懒加载
3. `ToolReference` 已具备审批状态机与按钮防重复提交语义。

### 必须兼容的现有事件

`meta` / `agent_handoff` / `plan` / `replan` / `delta` / `tool_call` / `tool_approval` / `tool_result` / `run_state` / `done` / `error`

说明：迁移期间允许新增事件，不允许破坏现有事件语义。

---

## 前端架构设计

### 1. 核心消息流与性能分层

#### 1.1 消息语义

1. 主消息流仍保持 user/assistant 交替。
2. 动态执行轨迹（handoff/tool/plan）继续承载在 assistant message 的 `runtime` 字段。

#### 1.2 性能策略（分阶段）

1. Phase A（本期）
   - 保持 `Bubble.List`，不做消息级虚拟列表。
   - 强化“重内容延迟呈现”：
     - step 内容继续懒加载（沿用现有机制）。
     - 大 tool result 默认摘要显示，全文延迟拉取。
2. Phase B（后续）
   - 若出现长会话性能瓶颈，再引入消息级虚拟滚动。

#### 1.3 大内容展示策略

采用“流内折叠 + 阈值外置查看”的混合方案：

1. 默认：聊天流内代码块折叠与懒加载。
2. 当内容超过阈值（建议 >200 行或 >64KB）：
   - 显示“查看原文”入口。
   - 在侧栏/弹层展示完整文本，避免撑爆消息流 DOM。

#### 1.4 历史记录视口级渲染

针对历史会话中的重内容块（长日志/大 JSON/YAML）增加视口级挂载控制：

1. 使用 `IntersectionObserver` 监听消息块可见性。
2. 未进入视口时仅渲染轻量占位（标题、摘要、固定高度容器）。
3. 进入视口后再挂载全文节点与语法高亮。
4. 滚出视口后可降级为等高占位，释放复杂 DOM 与高亮对象引用。

约束：占位高度必须稳定，避免滚动跳动（layout shift）。

---

### 2. 动态 Task Board（OpsTODO）

#### 2.1 事件协议

新增 SSE 事件：`ops_plan_updated`

Payload（建议）：

```json
{
  "todos": [
    {
      "id": "todo-1",
      "content": "检查 default 命名空间 Pod 健康",
      "status": "in_progress",
      "cluster": "prod-cluster-a",
      "namespace": "default",
      "resourceType": "Pod",
      "riskLevel": "medium"
    }
  ]
}
```

协议约束：`ops_plan_updated` 每次下发 **完整快照**（full snapshot），不采用增量 patch。
前端接收后直接覆盖 `runtime.todos`，避免回放与重连场景的 merge 歧义。

#### 2.2 前端数据模型

在现有 runtime 扩展，避免平行模型：

```ts
export interface RuntimeTodoItem {
  id: string;
  content: string;
  status: 'pending' | 'in_progress' | 'completed';
  cluster?: string;
  namespace?: string;
  resourceType?: string;
  riskLevel?: 'low' | 'medium' | 'high' | 'critical';
}

export interface AssistantReplyRuntime {
  // existing fields ...
  todos?: RuntimeTodoItem[];
}
```

#### 2.3 UI 位置

优先方案（本期）：在 `AssistantReply` 顶部渲染 Task Board 卡片（可折叠）。

备选方案（后续）：在 `CopilotSurface` 右侧抽屉增加独立任务面板。

---

### 3. HITL 风险审计卡片

#### 3.1 展示原则

审批卡片必须在消息流中醒目展示：

1. 工具名（如 `k8s_restart_deployment`）
2. 目标资源（cluster/namespace/resource）
3. 风险级别（high/critical）
4. 超时时间与审批状态

#### 3.2 交互状态机

1. waiting-approval
2. submitting
3. approved_resuming / approved_retrying / approved_done
4. rejected / expired / approved_failed_terminal / refresh-needed

要求：按钮在提交后锁定，避免重复操作；状态以事件回放为准，前端本地态仅作过渡。

---

### 4. Sub-Agent 活动指示

#### 4.1 指示来源

1. `agent_handoff.to`
2. `tool_call.agent`
3. `delta.agent`

#### 4.2 展示策略

1. 在输入框上方或最后一条 assistant 消息底部显示“当前执行者”。
2. 标签文案示例：`[K8sAgent] 正在获取 Pod 日志...`
3. 指示更新增加 300-500ms 防抖，避免高频 handoff 导致视觉闪烁。

#### 4.3 兼容要求

扩展 agent label 映射时，保留未知 agent 的兜底文案（如“助手”）。

---

### 5. 失败重试与熔断反馈

后端若触发“同子任务失败超过 3 次”熔断：

1. 前端停止 loading 动画。
2. 在 `runtime.status` 明确展示“需人工介入”。
3. 不再自动推进“正在生成”状态。

### 6. 审批断线重连兜底

当审批进入 `submitting` 后，如果 SSE 中断或确认事件迟迟未到，前端需避免状态僵死：

1. 为 `submitting` 增加本地超时（建议 15 秒）。
2. 超时后将卡片状态转为 `refresh-needed`，展示“状态可能已更新，请刷新”。
3. 提供“刷新审批状态/重试查询”入口，主动拉取审批票据最新状态。
4. 若查询结果已终态（approved/rejected/expired），立即覆盖本地状态并解锁 UI。

---

## 接口与类型变更清单

1. `web/src/api/modules/ai.ts`
   - 新增 `A2UIOpsPlanUpdatedEvent` 类型
   - `A2UIStreamHandlers` 新增 `onOpsPlanUpdated`
   - `dispatchAIStreamEvent` 新增 `ops_plan_updated` 分发
2. `web/src/components/AI/types.ts`
   - `AssistantReplyRuntime` 增加 `todos?: RuntimeTodoItem[]`
3. `web/src/components/AI/providers/PlatformChatProvider.ts`
   - 消费 `onOpsPlanUpdated` 并写入 runtime
4. `web/src/components/AI/AssistantReply.tsx`
   - 新增 Task Board 折叠卡片渲染
   - `submitting` 超时后的 refresh-needed 兜底提示与刷新入口

---

## 迁移计划（前端）

### Phase 1: 协议与模型

1. 新增 `ops_plan_updated` 事件类型和解析逻辑。
2. 扩展 runtime 类型定义。

### Phase 2: 运行时接入

1. `PlatformChatProvider` 接入 `onOpsPlanUpdated`。
2. 与现有 `plan/replan` 并行工作，互不覆盖。

### Phase 3: 交互渲染

1. `AssistantReply` 渲染 Task Board。
2. 审批卡片增强风险字段展示。
3. Agent 活动标签增强。

### Phase 4: 性能与回归

1. 长日志与大 JSON 的阈值外置展示。
2. 历史消息块 `IntersectionObserver` 视口挂载与滚出回收。
3. 回归审批、重连、历史会话回放场景。

---

## 验收标准

1. 功能验收
   - [ ] 收到 `ops_plan_updated` 后，Task Board 在 1 次渲染周期内更新。
   - [ ] `ops_plan_updated` 以全量快照覆盖，不出现 todos 合并错乱。
   - [ ] `tool_approval` 卡片可正确完成批准/拒绝并锁定状态。
   - [ ] `submitting` 超时后可自动进入 `refresh-needed` 并支持状态刷新。
   - [ ] Agent 活动标签能随 handoff/tool_call 变化。
   - [ ] Agent 标签在高频切换时无明显闪烁（防抖生效）。
   - [ ] 历史会话加载后，runtime 中 todos 与审批状态可重建。

2. 性能验收
   - [ ] 单条超长结果不会导致主消息流明显卡顿（交互无阻塞）。
   - [ ] 连续 100 条消息场景下输入和滚动保持可用。
   - [ ] 历史重内容块仅在进入视口时挂载全文，滚出后可回收复杂 DOM。

3. 稳定性验收
   - [ ] 未知 SSE 事件不影响主流程（仅观测，不报错）。
   - [ ] 审批重复点击不会触发重复请求。

---

## 风险与回滚

1. 风险：新增事件导致前后端协议偏差。
   - 缓解：保持新增事件可选；旧事件路径不受影响。
2. 风险：Task Board 与 plan/replan 冲突导致 UI 重复。
   - 缓解：定义优先级（todos 作为结构化主视图，plan/replan 作为辅助轨迹）。
3. 回滚：
   - 前端保留开关（是否渲染 todos 卡片）。
   - 后端可临时停发 `ops_plan_updated`，不影响基础聊天流程。
