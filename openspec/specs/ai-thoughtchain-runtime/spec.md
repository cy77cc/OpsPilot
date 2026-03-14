# Spec: AI ThoughtChain Runtime

## Overview

本规范定义 AI 运行时的 ThoughtChain 原生事件协议，将 plan/execute/tool/replan/approval 统一为用户叙事型链节点，确保过程展示与最终答案的严格分离，以及会话持久化和历史恢复能重建相同的链路与最终答案关系。

## Requirements

### REQ-TCR-001: AI runtime MUST emit native ThoughtChain lifecycle events

The system MUST emit a native event family for user-visible ThoughtChain lifecycle rather than requiring the frontend to infer chain semantics from generic phase or tool events.

#### Scenario: chain nodes are opened and closed by the runtime
- **WHEN** a chat turn enters planning, execution, replanning, or approval waiting
- **THEN** the runtime MUST emit `chain_node_open`, `chain_node_patch`, and `chain_node_close` events with stable node identity
- **AND** the frontend MUST be able to render the user-visible chain directly from those events without reconstructing stage ownership from unrelated event families

#### Scenario: tool result updates do not create extra narrative nodes
- **WHEN** a tool call produces progress, result, or summary data
- **THEN** the runtime MUST update the active tool node details
- **AND** the runtime MUST NOT create a separate user-visible `tool_result` chain node by default

### REQ-TCR-002: process chain MUST complete before final answer starts

The system MUST keep the live UI focused on the ThoughtChain while execution is in progress, and it MUST only start final-answer streaming after the process chain has reached a collapsed completed state.

#### Scenario: final answer starts only after chain collapse
- **WHEN** the runtime finishes all process-chain work for a turn
- **THEN** it MUST emit `chain_collapsed` before `final_answer_started`
- **AND** final answer content MUST be delivered only through `final_answer_delta`
- **AND** process-chain content MUST NOT continue streaming as normal final-answer prose

### REQ-TCR-003: approval waits MUST behave as normal chain nodes

Approval-required states MUST be represented as first-class ThoughtChain nodes rather than detached side panels or out-of-band flow interruptions.

#### Scenario: approval node pauses chain progression
- **WHEN** execution reaches a step that requires approval
- **THEN** the runtime MUST open an `approval` chain node
- **AND** the UI MUST render approval interaction within that node's detail area
- **AND** approval acceptance or rejection MUST close or patch the same node before the chain proceeds or terminates

### REQ-TCR-004: session replay MUST preserve chain and final-answer relationship

The persisted chat session model MUST preserve enough lifecycle state to reconstruct the same user-visible relationship between the collapsed ThoughtChain and the final answer during history replay.

#### Scenario: completed session replays collapsed chain and answer
- **WHEN** a client restores a completed AI session
- **THEN** the session detail response MUST allow the client to render a collapsed completed ThoughtChain and the final answer separately
- **AND** planner JSON, tool arguments, or replanning notes MUST NOT be replayed as ordinary final-answer prose

---

## Data Structures

### Chain Node Events

```json
// chain_node_open
{
  "type": "chain_node_open",
  "data": {
    "node_id": "node-planning-xxx",
    "node_type": "planning",
    "title": "整理执行步骤",
    "status": "loading"
  }
}

// chain_node_patch
{
  "type": "chain_node_patch",
  "data": {
    "node_id": "node-planning-xxx",
    "title": "整理执行步骤",
    "status": "done",
    "details": {
      "steps": [
        { "id": "step-1", "content": "检查集群状态" },
        { "id": "step-2", "content": "获取部署列表" }
      ]
    }
  }
}

// chain_node_close
{
  "type": "chain_node_close",
  "data": {
    "node_id": "node-planning-xxx",
    "status": "success"
  }
}
```

### Chain Lifecycle Events

```json
// chain_collapsed
{
  "type": "chain_collapsed",
  "data": {
    "turn_id": "turn-xxx",
    "node_count": 4,
    "summary": "思考完成"
  }
}

// final_answer_started
{
  "type": "final_answer_started",
  "data": {
    "turn_id": "turn-xxx"
  }
}

// final_answer_delta
{
  "type": "final_answer_delta",
  "data": {
    "turn_id": "turn-xxx",
    "content": "根据检查结果",
    "is_append": true
  }
}
```

### Node Types

| Node Type | Title (Chinese) | Description |
|-----------|-----------------|-------------|
| planning | 整理执行步骤 | Planner generates execution plan |
| executing | 执行步骤 | Executor runs steps |
| tool | 调用工具 | Tool invocation with parameters and results |
| replanning | 动态调整计划 | Replanner adjusts the plan |
| approval | 等待确认 | Approval gate waiting for user confirmation |

---

## Event Flow

### Normal Execution Flow with ThoughtChain

```
1. turn_started              -> 会话开始
2. chain_node_open           -> node_type: "planning"
3. chain_node_patch          -> details: { steps: [...] }
4. chain_node_close          -> status: "success"
5. chain_node_open           -> node_type: "executing"
6. chain_node_open           -> node_type: "tool" (nested under executing)
7. chain_node_patch          -> tool result
8. chain_node_close          -> tool complete
... (repeat 6-8 for each tool)
9. chain_node_close          -> executing complete
10. chain_collapsed          -> chain completed
11. final_answer_started     -> final answer begins
12. final_answer_delta       -> streaming answer content
...
13. done                     -> turn complete
```

### Approval Flow

```
... (during executing)
a. chain_node_open           -> node_type: "approval"
b. [UI shows approval interaction within node]
c. user approves/rejects
d. chain_node_patch          -> approval result
e. chain_node_close          -> approval resolved
... (continue or terminate based on result)
```

### Replan Flow

```
... (during executing)
a. chain_node_open           -> node_type: "replanning"
b. chain_node_patch          -> new plan details
c. chain_node_close          -> replanning complete
d. chain_node_open           -> node_type: "executing" (new phase)
... (continue execution)
```

---

## Relationship to Existing Specs

### ai-streaming-events

This spec extends `ai-streaming-events` by introducing chain-native events that complement the existing phase/tool events. During a transition period, both event families may be emitted simultaneously.

### ai-turn-lifecycle-storage

The chain node structure must be persisted as part of the turn/block model defined in `ai-turn-lifecycle-storage`. Chain nodes map to blocks, and the collapsed chain state must be preserved for session replay.

### ai-pre-execution-approval-gate

This spec integrates approval gating into the chain model as defined in `ai-pre-execution-approval-gate`. The approval becomes a visible chain node rather than an out-of-band interruption.

---

## Migration Strategy

### Phase 1: Parallel Emission

- Backend emits both existing events and new chain-native events
- Frontend can opt-in to chain-native rendering via feature flag

### Phase 2: Chain-Native Primary

- Frontend uses chain-native events as primary rendering source
- Existing events remain as fallback for compatibility

### Phase 3: Legacy Event Deprecation

- Confirm stability of chain-native rendering
- Remove redundant legacy event emission

### Feature Flag Configuration

```yaml
# configs/config.yaml
feature_flags:
  ai_thoughtchain_events: true  # Enable chain-native events
  ai_assistant_v2: true         # Use Plan-Execute runtime
```
