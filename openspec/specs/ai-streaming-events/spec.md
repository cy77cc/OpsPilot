# Spec: AI Streaming Events

## Overview

本规范定义 AI 聊天的 SSE 流式事件类型，确保跨流式传输、审批门控、恢复和兼容性客户端的助手输出语义一致性。同时支持 Plan-Execute-Replan 可视化所需的阶段生命周期、计划生成、步骤执行和重规划事件。

## Requirements

### REQ-SE-001: AI Chat Stream Events (Core)

系统 SHALL 发送 Server-Sent Events (SSE)，确保跨流式传输、审批门控、恢复和兼容性客户端的助手输出语义一致性。

必需的兼容性事件类型：
- `meta`: 流开始时的会话元数据
- `stage_delta`: rewrite、plan、approval-gate messaging、execute 和 summary 阶段的增量模型阶段输出
- `delta`: 最终回答内容的增量输出
- `thinking_delta`: 模型推理/思考内容
- `tool_call`: 工具调用，包含名称和参数
- `tool_result`: 工具执行完成结果
- `approval_required`: 执行开始前需要用户确认的审批门控
- `heartbeat`: 周期性保活事件
- `done`: 流完成
- `error`: 错误通知

必需的 turn/block 原生事件类型：
- `turn_started`: 助手 turn 生命周期开始
- `block_open`: 可渲染块被创建
- `block_delta`: 块的增量内容或状态更新
- `block_replace`: 结构化卡片的块载荷替换
- `block_close`: 块完成
- `turn_state`: 助手 turn 状态或阶段转换
- `turn_done`: 助手 turn 完成

#### Scenario: 助手文本 delta 保持追加模式
- **WHEN** 运行时在一个助手 turn 中收到累积的提供商内容快照
- **THEN** 发出的 `delta` 事件 MUST 仅包含新增的助手可见文本
- **AND** 按顺序连接所有发出的 `delta` 块 MUST 能重建最终助手回答
- **AND** 系统 MUST NOT 在后续 `delta` 中重新发送完整累积文本

#### Scenario: 阶段 delta 携带语义规划元数据
- **WHEN** 运行时进入规划或更新生成的计划
- **THEN** 为 plan 阶段发出的每个 `stage_delta` 事件 MUST 包含稳定的阶段标识符和状态
- **AND** 规划里程碑 MUST 包含用户可见的语义字段，如 `title`、`description` 和 `steps`（当数据存在时）
- **AND** 消费者 MUST NOT 被迫仅从硬编码的前端模板合成核心计划语义

#### Scenario: 内部结构化载荷不泄露到助手文本
- **WHEN** 模型或运行时产生内部结构化载荷，如工具参数或计划 JSON
- **THEN** 该载荷 MUST 通过结构化生命周期事件路由或从用户可见文本流中过滤
- **AND** 助手 `delta` 流 MUST NOT 将原始内部载荷（如 `{"steps": [...]}`）作为普通散文暴露

#### Scenario: 流暴露 turn 和 block 生命周期
- **WHEN** 为流式请求创建助手 turn
- **THEN** 系统必须发出 `turn_started`
- **AND** 用户可见的状态、文本、工具、审批、证据和错误表面 MUST 通过 block 生命周期事件表示
- **AND** block 更新 MUST 标识所属的 `turn_id` 和 `block_id`

#### Scenario: 不可用阶段被显式报告
- **WHEN** 模型支持的阶段在流式传输期间变得不可用
- **THEN** 系统必须为该阶段发出显式的用户可见失败或不可用信号
- **AND** 流 MUST NOT 替换代码生成的语义内容假装该阶段成功完成

#### Scenario: 审批门控出现在执行之前
- **WHEN** AI 识别需要审批的计划步骤
- **THEN** 流必须在门控步骤产生 `tool_call` 或 `tool_result` 之前发出 `approval_required`
- **AND** 流必须包含该门控的可恢复运行时标识
- **AND** 用户可见生命周期 MUST 显示 turn 等待审批，而非已执行该步骤

#### Scenario: 心跳维持连接
- **WHEN** 流式会话持续超过 10 秒
- **THEN** 系统每 10 秒发出 `heartbeat` 事件，包含时间戳

### REQ-SE-002: Checkpoint-based Resume

系统 SHALL 支持通过稳定的运行时标识而非模型特定的检查点语义恢复中断的会话，恢复执行 MUST 继续原始助手 turn 生命周期至执行和摘要阶段。

恢复输入必须基于：
- `session_id`
- `plan_id`
- `step_id`
- 审批决定

兼容性别名 MAY 为旧客户端存在，但运行时语义 MUST 基于计划-步骤。

#### Scenario: 审批后恢复
- **WHEN** 用户为中断会话提交审批
- **THEN** 系统恢复特定 `session_id + plan_id + step_id` 的执行
- **AND** 从该运行时点继续发出 SSE 事件
- **AND** 恢复的事件 MUST 继续在先前活跃的助手 `turn_id` 上
- **AND** 如果执行成功完成，恢复的流 MUST 继续进入摘要阶段后再完成

#### Scenario: 拒绝后恢复
- **WHEN** 用户为中断会话拒绝审批
- **THEN** 系统返回拒绝消息
- **AND** 不执行中断的步骤
- **AND** 中断的助手 turn MUST 进入终止的取消或拒绝状态

#### Scenario: 旧版恢复兼容性不重新定义运行时标识
- **WHEN** 旧客户端发送检查点风格的恢复请求
- **THEN** 系统 MAY 将该请求转换为当前运行时标识符
- **AND** 规范运行时标识 MUST 保持 `session_id + plan_id + step_id`

### REQ-SE-010: Phase Lifecycle Events

系统 SHALL 发送阶段生命周期事件以标识当前执行阶段。

事件类型：
- `phase_started`: 新阶段开始时发送
- `phase_complete`: 阶段完成时发送

#### Scenario: Planning Phase Lifecycle
- **WHEN** 运行时开始规划阶段
- **THEN** 系统必须发送 `phase_started` 事件，包含 `phase: "planning"` 和 `status: "loading"`
- **AND** 规划完成时发送 `phase_complete` 事件，包含 `phase: "planning"` 和 `status: "success"`

#### Scenario: Executing Phase Lifecycle
- **WHEN** 运行时进入执行阶段
- **THEN** 系统必须发送 `phase_started` 事件，包含 `phase: "executing"`
- **AND** 所有步骤执行完成后发送 `phase_complete` 事件

#### Scenario: Replanning Phase Lifecycle
- **WHEN** 运行时触发重规划
- **THEN** 系统必须发送 `phase_started` 事件，包含 `phase: "replanning"`
- **AND** 重规划完成时发送 `phase_complete` 事件

### REQ-SE-011: Plan Generated Event

系统 SHALL 在 Planner 完成时发送结构化的计划事件。

#### Scenario: Plan Generated Successfully
- **WHEN** Planner 成功生成执行计划
- **THEN** 系统必须发送 `plan_generated` 事件
- **AND** 事件必须包含 `plan_id`、`steps` 数组和 `total` 字段
- **AND** 每个步骤必须有唯一的 `id` 和 `content`

### REQ-SE-012: Step Lifecycle Events

系统 SHALL 在执行过程中发送步骤级别的事件。

事件类型：
- `step_started`: 步骤开始执行时发送
- `step_complete`: 步骤执行完成时发送

#### Scenario: Step Execution Lifecycle
- **WHEN** Executor 开始执行某个步骤
- **THEN** 系统必须发送 `step_started` 事件，包含 `step_id` 和 `status: "running"`
- **AND** 步骤完成时发送 `step_complete` 事件，包含 `step_id`、`status` 和可选的 `summary`

#### Scenario: Step Execution Failure
- **WHEN** 步骤执行失败
- **THEN** 系统必须发送 `step_complete` 事件，包含 `status: "error"`

### REQ-SE-013: Replan Triggered Event

系统 SHALL 在 Replanner 被触发时发送事件。

#### Scenario: Replan Triggered
- **WHEN** 执行结果需要调整计划
- **THEN** 系统必须发送 `replan_triggered` 事件
- **AND** 事件必须包含 `reason` 和 `completed_steps` 字段

### REQ-SE-014: Enhanced Tool Events

系统 SHALL 发送增强的工具事件，包含步骤关联。

#### Scenario: Tool Call with Step Association
- **WHEN** 工具被调用执行
- **THEN** 系统必须发送 `tool_call` 事件，包含 `step_id` 字段
- **AND** 工具执行完成后发送 `tool_result` 事件，包含 `step_id`、`status` 和可选的 `duration`

---

## Data Structures

### Phase Events

```json
// phase_started
{
  "type": "phase_started",
  "data": {
    "phase": "planning",
    "title": "整理执行步骤",
    "status": "loading"
  }
}

// phase_complete
{
  "type": "phase_complete",
  "data": {
    "phase": "planning",
    "status": "success"
  }
}
```

### Plan Generated Event

```json
{
  "type": "plan_generated",
  "data": {
    "plan_id": "plan-xxx",
    "steps": [
      { "id": "step-1", "content": "检查集群状态", "tool_hint": "get_cluster_info" },
      { "id": "step-2", "content": "获取部署列表", "tool_hint": "list_deployments" }
    ],
    "total": 2
  }
}
```

### Step Events

```json
// step_started
{
  "type": "step_started",
  "data": {
    "step_id": "step-1",
    "title": "检查集群状态",
    "tool_name": "get_cluster_info",
    "params": {},
    "status": "running"
  }
}

// step_complete
{
  "type": "step_complete",
  "data": {
    "step_id": "step-1",
    "status": "success",
    "summary": "集群状态正常"
  }
}
```

### Replan Event

```json
{
  "type": "replan_triggered",
  "data": {
    "reason": "步骤执行失败，需要调整计划",
    "completed_steps": 2
  }
}
```

### Enhanced Tool Events

```json
// tool_call
{
  "type": "tool_call",
  "data": {
    "step_id": "step-1",
    "tool_name": "get_cluster_info",
    "arguments": {}
  }
}

// tool_result
{
  "type": "tool_result",
  "data": {
    "step_id": "step-1",
    "tool_name": "get_cluster_info",
    "result": "集群状态: Ready",
    "status": "success",
    "duration": 150
  }
}
```

---

## Event Flow

### Normal Execution Flow

```
1. turn_started           → 会话开始
2. phase_started          → phase: "planning"
3. plan_generated         → steps: [...]
4. phase_complete         → phase: "planning"
5. phase_started          → phase: "executing"
6. step_started           → step_id: "step-1"
7. tool_call              → step_id: "step-1"
8. tool_result            → step_id: "step-1"
9. step_complete          → step_id: "step-1"
... (重复 6-9)
10. phase_complete        → phase: "executing"
11. done                  → 执行完成
```

### Replan Flow

```
... (执行中)
a. replan_triggered       → reason: "...", completed_steps: 2
b. phase_complete         → phase: "executing"
c. phase_started          → phase: "replanning"
d. plan_generated         → steps: [...] (新计划)
e. phase_complete         → phase: "replanning"
f. phase_started          → phase: "executing"
... (继续执行)
```

---

## Compatibility Strategy

### Backward Compatibility

1. **保留旧事件**: `stage_delta` 和 `step_update` 继续发送
2. **双事件模式**: 新旧事件同时发送，前端优先处理新事件
3. **Feature Flag**: 通过 `ai_enhanced_events` 控制新事件的启用

### Migration Path

```
阶段 1: 后端同时发送新旧事件
阶段 2: 前端优先使用新事件，旧事件作为 fallback
阶段 3: 确认稳定后，移除旧事件支持
```

### Feature Flag Configuration

```yaml
# configs/config.yaml
feature_flags:
  ai_enhanced_events: true  # 启用增强事件
  ai_assistant_v2: true     # 使用 Plan-Execute 运行时
```

---

## Phase Title Mapping

| Phase | Title (Chinese) | Title (English) |
|-------|-----------------|-----------------|
| planning | 整理执行步骤 | Planning |
| executing | 执行步骤 | Executing |
| replanning | 动态调整计划 | Replanning |
