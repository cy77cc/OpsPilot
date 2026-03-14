# AI Runtime Core Specification

## Overview

本规范定义 AI Runtime 核心功能增强，支持 Plan-Execute-Replan 可视化所需的阶段检测、计划解析和 SSE 流式事件输出。这些功能使前端能够实时展示 Agent 的执行进度和状态变化。

## MODIFIED Requirements

### REQ-RT-003: SSE 流式输出 (Enhanced)

系统 SHALL 通过 SSE 流式输出 Agent 执行事件，支持新增的阶段事件类型。

#### Scenario: 基础事件流输出
- **WHEN** 用户发送对话请求
- **AND** Agent 开始执行
- **THEN** 系统通过 SSE 流式输出事件
- **AND** 支持的事件类型包括: meta, delta, thinking_delta, tool_call, tool_result, approval_required, done, error

#### Scenario: 阶段事件输出
- **WHEN** Agent 执行阶段发生变化
- **THEN** 系统发送 phase_started 事件
- **AND** 事件体包含 phase 字段 (planning/executing/replanning)
- **AND** 事件体包含 timestamp 字段

#### Scenario: 计划生成事件输出
- **WHEN** Planner 完成计划生成
- **THEN** 系统发送 plan_generated 事件
- **AND** 事件体包含 planId 字段
- **AND** 事件体包含 steps 数组

#### Scenario: 步骤执行事件输出
- **WHEN** Executor 开始执行某个步骤
- **THEN** 系统发送 step_started 事件
- **AND** 事件体包含 stepId 和 title 字段

#### Scenario: 步骤完成事件输出
- **WHEN** Executor 完成某个步骤执行
- **THEN** 系统发送 step_complete 事件
- **AND** 事件体包含 stepId、status 和 summary 字段

#### Scenario: 重规划事件输出
- **WHEN** Replanner 触发重新规划
- **THEN** 系统发送 replan_triggered 事件
- **AND** 事件体包含 reason 字段说明触发原因

## NEW Requirements

### REQ-RT-005: Phase Detection

系统 SHALL 在 Orchestrator 中检测当前执行阶段，并输出阶段变化事件。

#### Scenario: planning 阶段检测
- **WHEN** RunPath 包含 "planner"
- **OR** AgentName 为 "planner"
- **THEN** 系统推断当前阶段为 "planning"
- **AND** 发送 phase_started 事件，phase 字段值为 "planning"

#### Scenario: executing 阶段检测
- **WHEN** RunPath 包含 "executor"
- **OR** AgentName 为 "executor"
- **THEN** 系统推断当前阶段为 "executing"
- **AND** 发送 phase_started 事件，phase 字段值为 "executing"

#### Scenario: replanning 阶段检测
- **WHEN** RunPath 包含 "replan"
- **OR** AgentName 为 "replanner"
- **THEN** 系统推断当前阶段为 "replanning"
- **AND** 发送 phase_started 事件，phase 字段值为 "replanning"

#### Scenario: 阶段变化检测
- **WHEN** 检测到的阶段与前一阶段不同
- **THEN** 系统发送 phase_complete 事件表示前一阶段结束
- **AND** 发送 phase_started 事件表示新阶段开始

#### Scenario: 未知阶段处理
- **WHEN** RunPath 和 AgentName 均无法识别阶段
- **THEN** 系统保持当前阶段不变
- **AND** 不发送阶段变化事件

### REQ-RT-006: Plan Parsing

系统 SHALL 从 Planner 输出解析结构化计划，生成 plan_generated 事件。

#### Scenario: JSON 格式计划解析成功
- **WHEN** Planner 输出包含 JSON 格式的步骤列表
- **AND** JSON 格式符合 PlanStep 结构定义
- **THEN** 系统成功解析计划
- **AND** 发送 plan_generated 事件
- **AND** 事件体包含结构化的 steps 数组
- **AND** 每个步骤包含 id、content、toolHint 字段

#### Scenario: 带代码块标记的计划解析
- **WHEN** Planner 输出包含 markdown 代码块标记 (```json ... ```)
- **THEN** 系统提取代码块内的 JSON 内容
- **AND** 解析为结构化计划
- **AND** 发送 plan_generated 事件

#### Scenario: 计划解析失败处理
- **WHEN** Planner 输出不包含有效 JSON
- **OR** JSON 格式不符合预期结构
- **THEN** 系统记录解析错误日志
- **AND** 不发送 plan_generated 事件
- **AND** 继续处理后续事件流

#### Scenario: 空计划处理
- **WHEN** 解析得到的 steps 数组为空
- **THEN** 系统发送 plan_generated 事件
- **AND** steps 数组为空数组 []

#### Scenario: 计划 ID 生成
- **WHEN** 成功解析计划
- **THEN** 系统生成唯一的 planId
- **AND** planId 格式为 "plan-{timestamp}-{random}"

## NEW Interfaces

### PhaseDetector

```go
// PhaseDetector 从 ADK 事件推断当前执行阶段
type PhaseDetector struct {
    currentPhase string
    stepCounter  int
}

// Detect 从事件推断当前阶段
// 返回: planning, executing, replanning
func (d *PhaseDetector) Detect(event *adk.AgentEvent) string

// NextStepID 生成下一个步骤ID
func (d *PhaseDetector) NextStepID() string
```

### PlanParser

```go
// PlanParser 从 LLM 输出解析结构化计划
type PlanParser struct{}

// Parse 从 Planner 输出解析计划步骤
func (p *PlanParser) Parse(event *adk.AgentEvent) *ParsedPlan

type ParsedPlan struct {
    PlanID string
    Steps  []PlanStep
}

type PlanStep struct {
    ID       string
    Content  string
    ToolHint string
}
```

### StepInfo

```go
// StepInfo 步骤执行信息
type StepInfo struct {
    StepID    string
    Title     string
    ToolName  string
    Params    map[string]any
    Status    string  // running, success, error
    StartedAt string
    Summary   string
}
```

## Event Types

### Phase Events

| Event Type | Description | Payload |
|------------|-------------|---------|
| phase_started | 阶段开始 | `{ phase: string, timestamp: string }` |
| phase_complete | 阶段完成 | `{ phase: string, timestamp: string }` |

### Plan Events

| Event Type | Description | Payload |
|------------|-------------|---------|
| plan_generated | 计划生成完成 | `{ planId: string, steps: PlanStep[] }` |
| replan_triggered | 重规划触发 | `{ reason: string, timestamp: string }` |

### Step Events

| Event Type | Description | Payload |
|------------|-------------|---------|
| step_started | 步骤开始执行 | `{ stepId: string, title: string }` |
| step_complete | 步骤执行完成 | `{ stepId: string, status: string, summary: string }` |
