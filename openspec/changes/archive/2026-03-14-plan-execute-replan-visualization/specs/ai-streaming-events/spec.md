# Spec: AI Streaming Events (Plan-Execute-Replan Visualization)

## Overview

本规范定义用于支持 Plan-Execute-Replan 可视化的 SSE 事件类型和数据结构，包括阶段生命周期、计划生成、步骤执行和重规划事件。

## MODIFIED Requirements

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
