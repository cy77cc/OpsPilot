# Plan-Execute-Replan 可视化实现指导

> 本文档提供前后端实现的具体指导，包括文件变更、代码示例和测试策略。

## 一、实现概览

### 1.1 实现目标

将现有的 `stage_delta` 和 `step_update` 事件升级为更明确的阶段和步骤生命周期事件，支持前端展示 Plan-Execute-Replan 执行过程。

### 1.2 核心变更

| 组件 | 变更类型 | 影响范围 |
|------|---------|---------|
| 后端事件定义 | 新增 | `internal/ai/events/events.go` |
| 后端数据结构 | 新增 | `internal/ai/runtime/runtime.go` |
| 后端事件转换 | 重构 | `internal/ai/runtime/sse_converter.go` |
| 后端阶段检测 | 新增 | `internal/ai/runtime/phase_detector.go` |
| 前端类型定义 | 扩展 | `web/src/components/AI/types.ts` |
| 前端事件处理 | 扩展 | `web/src/components/AI/hooks/useAIChat.ts` |

---

## 二、后端实现

### 2.1 文件变更清单

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `internal/ai/events/events.go` | 修改 | 新增 6 个事件常量 |
| `internal/ai/runtime/runtime.go` | 修改 | 新增 5 个数据结构 |
| `internal/ai/runtime/sse_converter.go` | 重构 | 新增 8 个转换方法 |
| `internal/ai/runtime/phase_detector.go` | 新建 | 阶段检测器 |
| `internal/ai/runtime/phase_detector_test.go` | 新建 | 单元测试 |
| `internal/ai/orchestrator.go` | 修改 | 集成阶段检测 |

### 2.2 events/events.go 变更

**位置:** 第 36 行后添加

```go
    // === 新增: 阶段生命周期事件 ===
    PhaseStarted    Name = "phase_started"    // 阶段开始
    PhaseComplete   Name = "phase_complete"   // 阶段完成
    PlanGenerated   Name = "plan_generated"   // 计划生成
    StepStarted     Name = "step_started"     // 步骤开始
    StepComplete    Name = "step_complete"    // 步骤完成
    ReplanTriggered Name = "replan_triggered" // 重规划触发
```

### 2.3 runtime/runtime.go 变更

**位置:** 第 57 行（PlanStep 结构后）添加

```go
// PhaseInfo 阶段信息
type PhaseInfo struct {
    Phase  string `json:"phase"`
    Title  string `json:"title"`
    Status string `json:"status"`
}

// StepInfo 步骤信息
type StepInfo struct {
    StepID    string         `json:"step_id"`
    Title     string         `json:"title,omitempty"`
    ToolName  string         `json:"tool_name,omitempty"`
    Params    map[string]any `json:"params,omitempty"`
    Status    string         `json:"status"`
    StartedAt string         `json:"started_at,omitempty"`
    Summary   string         `json:"summary,omitempty"`
}

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
    StepID    string         `json:"step_id"`
    ToolName  string         `json:"tool_name"`
    Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolResultInfo 工具结果信息
type ToolResultInfo struct {
    StepID   string `json:"step_id"`
    ToolName string `json:"tool_name"`
    Result   string `json:"result,omitempty"`
    Status   string `json:"status"`
    Duration int64  `json:"duration,omitempty"`
}

// ReplanInfo 重规划信息
type ReplanInfo struct {
    Reason         string `json:"reason"`
    CompletedSteps int    `json:"completed_steps"`
}
```

**同时在常量区添加:**

```go
const (
    // ... 现有常量 ...

    // === 新增事件常量 ===
    EventPhaseStarted    EventType = events.PhaseStarted
    EventPhaseComplete   EventType = events.PhaseComplete
    EventPlanGenerated   EventType = events.PlanGenerated
    EventStepStarted     EventType = events.StepStarted
    EventStepComplete    EventType = events.StepComplete
    EventReplanTriggered EventType = events.ReplanTriggered
)
```

### 2.4 runtime/sse_converter.go 变更

**新增方法:**

```go
// OnPhaseStarted 发送阶段开始事件
func (c *SSEConverter) OnPhaseStarted(phase, title string) StreamEvent {
    return StreamEvent{
        Type: EventPhaseStarted,
        Data: map[string]any{
            "phase":  phase,
            "title":  title,
            "status": "loading",
        },
    }
}

// OnPhaseComplete 发送阶段完成事件
func (c *SSEConverter) OnPhaseComplete(phase, status string) StreamEvent {
    return StreamEvent{
        Type: EventPhaseComplete,
        Data: map[string]any{
            "phase":  phase,
            "status": status,
        },
    }
}

// OnPlanGenerated 发送计划生成事件
func (c *SSEConverter) OnPlanGenerated(planID string, steps []PlanStep) StreamEvent {
    return StreamEvent{
        Type: EventPlanGenerated,
        Data: map[string]any{
            "plan_id": planID,
            "steps":   steps,
            "total":   len(steps),
        },
    }
}

// OnStepStarted 发送步骤开始事件
func (c *SSEConverter) OnStepStarted(info *StepInfo) StreamEvent {
    if info == nil {
        return StreamEvent{}
    }
    data := map[string]any{
        "step_id": info.StepID,
        "status":  "running",
    }
    if info.Title != "" {
        data["title"] = info.Title
    }
    if info.ToolName != "" {
        data["tool_name"] = info.ToolName
    }
    if info.Params != nil {
        data["params"] = info.Params
    }
    return StreamEvent{Type: EventStepStarted, Data: data}
}

// OnStepComplete 发送步骤完成事件
func (c *SSEConverter) OnStepComplete(stepID, status, summary string) StreamEvent {
    data := map[string]any{
        "step_id": stepID,
        "status":  status,
    }
    if summary != "" {
        data["summary"] = summary
    }
    return StreamEvent{Type: EventStepComplete, Data: data}
}

// OnToolCallWithStep 发送工具调用事件（带 step_id）
func (c *SSEConverter) OnToolCallWithStep(info *ToolCallInfo) StreamEvent {
    if info == nil {
        return StreamEvent{}
    }
    data := map[string]any{
        "step_id":   info.StepID,
        "tool_name": info.ToolName,
    }
    if info.Arguments != nil {
        data["arguments"] = info.Arguments
    }
    return StreamEvent{Type: EventToolCall, Data: data}
}

// OnToolResultWithStep 发送工具结果事件（带 step_id）
func (c *SSEConverter) OnToolResultWithStep(info *ToolResultInfo) StreamEvent {
    if info == nil {
        return StreamEvent{}
    }
    data := map[string]any{
        "step_id":   info.StepID,
        "tool_name": info.ToolName,
        "status":    info.Status,
    }
    if info.Result != "" {
        data["result"] = info.Result
    }
    if info.Duration > 0 {
        data["duration"] = info.Duration
    }
    return StreamEvent{Type: EventToolResult, Data: data}
}

// OnReplanTriggered 发送重规划触发事件
func (c *SSEConverter) OnReplanTriggered(info *ReplanInfo) StreamEvent {
    if info == nil {
        return StreamEvent{}
    }
    return StreamEvent{
        Type: EventReplanTriggered,
        Data: map[string]any{
            "reason":          info.Reason,
            "completed_steps": info.CompletedSteps,
        },
    }
}
```

### 2.5 runtime/phase_detector.go (新建)

```go
// Package runtime 提供阶段检测功能
package runtime

import (
    "fmt"
    "strings"
    "sync/atomic"
)

// PhaseDetector 从 ADK 事件推断当前执行阶段
type PhaseDetector struct {
    currentPhase atomic.Value // string
    stepCounter  int32
}

// NewPhaseDetector 创建阶段检测器
func NewPhaseDetector() *PhaseDetector {
    d := &PhaseDetector{}
    d.currentPhase.Store("planning")
    return d
}

// Detect 从 RunPath 和 AgentName 推断当前阶段
func (d *PhaseDetector) Detect(runPath, agentName string) string {
    // 方法1: 从 RunPath 推断
    path := strings.ToLower(runPath)
    if path != "" {
        switch {
        case strings.Contains(path, "replanner"):
            d.currentPhase.Store("replanning")
            return "replanning"
        case strings.Contains(path, "executor"):
            d.currentPhase.Store("executing")
            return "executing"
        case strings.Contains(path, "planner"):
            d.currentPhase.Store("planning")
            return "planning"
        }
    }

    // 方法2: 从 AgentName 推断
    name := strings.ToLower(agentName)
    if name != "" {
        switch {
        case name == "replanner":
            d.currentPhase.Store("replanning")
            return "replanning"
        case name == "executor":
            d.currentPhase.Store("executing")
            return "executing"
        case name == "planner":
            d.currentPhase.Store("planning")
            return "planning"
        }
    }

    return d.CurrentPhase()
}

// CurrentPhase 获取当前阶段
func (d *PhaseDetector) CurrentPhase() string {
    v := d.currentPhase.Load()
    if s, ok := v.(string); ok {
        return s
    }
    return "planning"
}

// SetPhase 设置当前阶段
func (d *PhaseDetector) SetPhase(phase string) {
    d.currentPhase.Store(phase)
}

// NextStepID 生成下一个步骤ID
func (d *PhaseDetector) NextStepID() string {
    n := atomic.AddInt32(&d.stepCounter, 1)
    return fmt.Sprintf("step-%d", n)
}

// Reset 重置检测器状态
func (d *PhaseDetector) Reset() {
    d.currentPhase.Store("planning")
    atomic.StoreInt32(&d.stepCounter, 0)
}
```

### 2.6 orchestrator.go 集成示例

```go
// 在 streamExecution 方法中集成阶段检测

func (o *Orchestrator) streamExecution(
    ctx context.Context,
    iter *adk.AsyncIterator[*adk.AgentEvent],
    state *ExecutionState,
    emit StreamEmitter,
) error {

    detector := NewPhaseDetector()
    lastPhase := ""
    stepIDMap := make(map[string]string) // toolCallID -> stepID

    for {
        event, ok := iter.Next()
        if !ok {
            break
        }

        if event == nil || event.Err != nil {
            if event != nil && event.Err != nil {
                emit(o.converter.OnError("execution", event.Err))
            }
            continue
        }

        // === 阶段检测 ===
        currentPhase := detector.Detect(event.RunPath, event.AgentName)
        if currentPhase != lastPhase {
            emit(o.converter.OnPhaseStarted(currentPhase, phaseTitle(currentPhase)))
            lastPhase = currentPhase
        }

        // === 处理其他事件 ===
        // ... 现有逻辑 ...

        // === 工具调用 ===
        if toolCalls := extractToolCalls(event); len(toolCalls) > 0 {
            for _, tc := range toolCalls {
                stepID := detector.NextStepID()
                stepIDMap[tc.ID] = stepID

                emit(o.converter.OnStepStarted(&StepInfo{
                    StepID:   stepID,
                    Title:    tc.Description,
                    ToolName: tc.Name,
                    Params:   tc.Arguments,
                }))

                emit(o.converter.OnToolCallWithStep(&ToolCallInfo{
                    StepID:    stepID,
                    ToolName:  tc.Name,
                    Arguments: tc.Arguments,
                }))
            }
        }

        // === 工具结果 ===
        if toolResult := extractToolResult(event); toolResult != nil {
            stepID := stepIDMap[toolResult.ToolCallID]
            if stepID == "" {
                stepID = detector.NextStepID()
            }

            emit(o.converter.OnToolResultWithStep(&ToolResultInfo{
                StepID:   stepID,
                ToolName: toolResult.ToolName,
                Result:   toolResult.Content,
                Status:   "success",
            }))

            emit(o.converter.OnStepComplete(stepID, "success", truncateSummary(toolResult.Content)))
        }

        // === Replan 检测 ===
        if isReplanTriggered(event) {
            emit(o.converter.OnReplanTriggered(&ReplanInfo{
                Reason:         "根据执行结果调整计划",
                CompletedSteps: len(state.Steps),
            }))
        }
    }

    // === 完成 ===
    if lastPhase != "" {
        emit(o.converter.OnPhaseComplete(lastPhase, "success"))
    }
    emit(o.converter.OnDone("completed"))

    return nil
}

// 辅助函数
func phaseTitle(phase string) string {
    titles := map[string]string{
        "planning":   "整理执行步骤",
        "executing":  "执行步骤",
        "replanning": "动态调整计划",
    }
    return titles[phase]
}
```

---

## 三、前端实现

### 3.1 文件变更清单

| 文件路径 | 操作 | 说明 |
|---------|------|------|
| `web/src/components/AI/types.ts` | 修改 | 新增 SSE 事件类型 |
| `web/src/api/modules/ai.ts` | 修改 | 扩展 handlers 接口 |
| `web/src/components/AI/hooks/useAIChat.ts` | 修改 | 新增事件处理器 |

### 3.2 types.ts 变更

**在 SSEEventType 中添加:**

```typescript
export type SSEEventType =
  // ... 现有类型 ...
  // === 新增 ===
  | 'phase_started'
  | 'phase_complete'
  | 'plan_generated'
  | 'step_started'
  | 'step_complete'
  | 'replan_triggered';
```

**新增事件类型定义:**

```typescript
// 阶段事件
export interface SSEPhaseStartedEvent {
  type: 'phase_started';
  data: {
    phase: 'planning' | 'executing' | 'replanning';
    title: string;
    status: 'loading';
  };
}

export interface SSEPhaseCompleteEvent {
  type: 'phase_complete';
  data: {
    phase: 'planning' | 'executing' | 'replanning';
    status: 'success' | 'error';
  };
}

// 计划事件
export interface SSEPlanGeneratedEvent {
  type: 'plan_generated';
  data: {
    plan_id: string;
    steps: Array<{ id: string; content: string; tool_hint?: string }>;
    total: number;
  };
}

// 步骤事件
export interface SSEStepStartedEvent {
  type: 'step_started';
  data: {
    step_id: string;
    title?: string;
    tool_name?: string;
    params?: Record<string, unknown>;
    status: 'running';
  };
}

export interface SSEStepCompleteEvent {
  type: 'step_complete';
  data: {
    step_id: string;
    status: 'success' | 'error';
    summary?: string;
  };
}

// 重规划事件
export interface SSEReplanTriggeredEvent {
  type: 'replan_triggered';
  data: {
    reason: string;
    completed_steps: number;
  };
}
```

### 3.3 api/modules/ai.ts 变更

**扩展 AIChatStreamHandlers:**

```typescript
export interface AIChatStreamHandlers {
  // 现有 handlers...
  onMeta?: (payload: SSEMetaEvent) => void;
  onDelta?: (payload: SSEDeltaEvent) => void;
  onThinkingDelta?: (payload: SSEThinkingDeltaEvent) => void;
  onToolCall?: (payload: SSEToolCallEvent) => void;
  onToolResult?: (payload: SSEToolResultEvent) => void;
  onApprovalRequired?: (payload: SSEApprovalRequiredEvent) => void;
  onDone?: (payload: SSEDoneEvent) => void;
  onError?: (payload: SSEErrorEvent) => void;

  // === 新增 handlers ===
  onPhaseStarted?: (payload: SSEPhaseStartedEvent['data']) => void;
  onPhaseComplete?: (payload: SSEPhaseCompleteEvent['data']) => void;
  onPlanGenerated?: (payload: SSEPlanGeneratedEvent['data']) => void;
  onStepStarted?: (payload: SSEStepStartedEvent['data']) => void;
  onStepComplete?: (payload: SSEStepCompleteEvent['data']) => void;
  onReplanTriggered?: (payload: SSEReplanTriggeredEvent['data']) => void;
}
```

### 3.4 useAIChat.ts 事件处理器

**新增事件处理函数:**

```typescript
/**
 * 将阶段名映射到 ThoughtStageKey
 */
function mapPhaseToStage(phase: string): ThoughtStageKey {
  const mapping: Record<string, ThoughtStageKey> = {
    planning: 'plan',
    executing: 'execute',
    replanning: 'plan',
  };
  return mapping[phase] || 'execute';
}

/**
 * 更新或插入 ThoughtStageItem
 */
function upsertThoughtStage(
  stages: ThoughtStageItem[] | undefined,
  update: Partial<ThoughtStageItem> & { key: ThoughtStageKey }
): ThoughtStageItem[] {
  const result = stages ? [...stages] : [];
  const index = result.findIndex(s => s.key === update.key);

  if (index >= 0) {
    result[index] = { ...result[index], ...update };
  } else {
    result.push({
      key: update.key,
      title: '',
      status: 'loading',
      ...update,
    });
  }

  return result;
}

/**
 * 批量更新多个 ThoughtStageItem
 */
function upsertMultipleStages(
  stages: ThoughtStageItem[],
  updates: Array<Partial<ThoughtStageItem> & { key: ThoughtStageKey }>
): ThoughtStageItem[] {
  let result = [...stages];
  for (const update of updates) {
    result = upsertThoughtStage(result, update);
  }
  return result;
}

/**
 * 处理 phase_started 事件
 */
export function handlePhaseStarted(
  payload: SSEPhaseStartedEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) {
  const { phase, title } = payload;
  const stageKey = mapPhaseToStage(phase);

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    return {
      ...item,
      thoughtChain: upsertThoughtStage(item.thoughtChain, {
        key: stageKey,
        title,
        status: 'loading',
      }),
    };
  }));
}

/**
 * 处理 plan_generated 事件
 */
export function handlePlanGenerated(
  payload: SSEPlanGeneratedEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) {
  const { steps } = payload;

  const planSteps: PlanStep[] = steps.map((s, i) => ({
    id: s.id || `step-${i + 1}`,
    content: s.content,
    tool_hint: s.tool_hint,
    status: 'pending',
  }));

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    return {
      ...item,
      thoughtChain: upsertThoughtStage(item.thoughtChain, {
        key: 'plan',
        status: 'success',
        steps: planSteps,
      }),
    };
  }));
}

/**
 * 处理 step_started 事件
 */
export function handleStepStarted(
  payload: SSEStepStartedEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) {
  const { step_id, title, tool_name, params } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    const thoughtChain = item.thoughtChain || [];
    const planStage = thoughtChain.find(s => s.key === 'plan');
    const executeStage = thoughtChain.find(s => s.key === 'execute');

    // 更新 plan 阶段的步骤状态
    const planSteps = (planStage as ExtendedThoughtStageItem)?.steps || [];
    const stepIndex = planSteps.findIndex(s => s.id === step_id);
    const updatedSteps = [...planSteps];
    if (stepIndex >= 0) {
      updatedSteps[stepIndex] = { ...updatedSteps[stepIndex], status: 'running' };
    }

    // 更新 execute 阶段
    const executions = executeStage?.details || [];
    const newExecution: ThoughtStageDetailItem = {
      id: `exec-${step_id}`,
      label: title || tool_name || '执行步骤',
      status: 'loading',
      kind: 'tool',
      tool: tool_name,
      params,
    };

    return {
      ...item,
      thoughtChain: upsertMultipleStages(thoughtChain, [
        { key: 'plan', steps: updatedSteps, currentStepIndex: stepIndex },
        { key: 'execute', status: 'loading', details: [...executions, newExecution] },
      ]),
    };
  }));
}

/**
 * 处理 step_complete 事件
 */
export function handleStepComplete(
  payload: SSEStepCompleteEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) {
  const { step_id, status, summary } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    const thoughtChain = item.thoughtChain || [];
    const planStage = thoughtChain.find(s => s.key === 'plan');
    const executeStage = thoughtChain.find(s => s.key === 'execute');

    // 更新 plan 阶段的步骤状态
    const planSteps = (planStage as ExtendedThoughtStageItem)?.steps || [];
    const stepIndex = planSteps.findIndex(s => s.id === step_id);
    const updatedSteps = [...planSteps];
    if (stepIndex >= 0) {
      updatedSteps[stepIndex] = {
        ...updatedSteps[stepIndex],
        status: status === 'success' ? 'completed' : 'failed',
        result: { ok: status === 'success', summary },
      };
    }

    // 更新 execute 阶段
    const executions = executeStage?.details || [];
    const execIndex = executions.findIndex(e => e.id === `exec-${step_id}`);
    const updatedExecutions = [...executions];
    if (execIndex >= 0) {
      updatedExecutions[execIndex] = {
        ...updatedExecutions[execIndex],
        status: status === 'success' ? 'success' : 'error',
        result: { ok: status === 'success', data: summary },
      };
    }

    return {
      ...item,
      thoughtChain: upsertMultipleStages(thoughtChain, [
        { key: 'plan', steps: updatedSteps },
        { key: 'execute', details: updatedExecutions },
      ]),
    };
  }));
}

/**
 * 处理 replan_triggered 事件
 */
export function handleReplanTriggered(
  payload: SSEReplanTriggeredEvent['data'],
  setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>,
  assistantId: string
) {
  const { reason } = payload;

  setMessages(prev => prev.map(item => {
    if (item.id !== assistantId) return item;

    return {
      ...item,
      thoughtChain: upsertThoughtStage(item.thoughtChain, {
        key: 'plan',
        status: 'loading',
        title: '动态调整计划',
        description: reason || '根据执行结果重新规划',
        replanReason: reason,
      }),
    };
  }));
}
```

---

## 四、测试策略

### 4.1 后端单元测试

```go
// internal/ai/runtime/sse_converter_test.go

func TestOnPhaseStarted(t *testing.T) {
    c := NewSSEConverter()
    event := c.OnPhaseStarted("planning", "整理执行步骤")

    assert.Equal(t, EventPhaseStarted, event.Type)
    assert.Equal(t, "planning", event.Data["phase"])
    assert.Equal(t, "整理执行步骤", event.Data["title"])
}

func TestOnPlanGenerated(t *testing.T) {
    c := NewSSEConverter()
    steps := []PlanStep{
        {ID: "step-1", Content: "查询状态"},
    }

    event := c.OnPlanGenerated("plan-123", steps)

    assert.Equal(t, EventPlanGenerated, event.Type)
    assert.Equal(t, "plan-123", event.Data["plan_id"])
    assert.Equal(t, 1, event.Data["total"])
}

func TestPhaseDetector(t *testing.T) {
    d := NewPhaseDetector()

    assert.Equal(t, "planning", d.Detect("[{planner}]", ""))
    assert.Equal(t, "executing", d.Detect("[{executor}]", ""))
    assert.Equal(t, "replanning", d.Detect("[{replanner}]", ""))
}
```

### 4.2 前端单元测试

```typescript
// web/src/components/AI/hooks/useAIChat.test.ts

describe('Event Handlers', () => {
  const mockSetMessages = jest.fn();
  const assistantId = 'assistant-1';

  beforeEach(() => {
    mockSetMessages.mockClear();
  });

  describe('handlePhaseStarted', () => {
    it('should create plan stage with loading status', () => {
      handlePhaseStarted(
        { phase: 'planning', title: '整理执行步骤', status: 'loading' },
        mockSetMessages,
        assistantId
      );

      const updateFn = mockSetMessages.mock.calls[0][0];
      const result = updateFn([{ id: assistantId, thoughtChain: [] }]);

      expect(result[0].thoughtChain).toContainEqual(
        expect.objectContaining({ key: 'plan', status: 'loading' })
      );
    });
  });

  describe('handlePlanGenerated', () => {
    it('should update plan stage with steps', () => {
      handlePlanGenerated(
        {
          plan_id: 'plan-1',
          steps: [{ id: 'step-1', content: '查询状态' }],
          total: 1,
        },
        mockSetMessages,
        assistantId
      );

      const updateFn = mockSetMessages.mock.calls[0][0];
      const result = updateFn([{ id: assistantId, thoughtChain: [] }]);

      expect(result[0].thoughtChain[0].steps).toHaveLength(1);
    });
  });
});
```

---

## 五、迁移检查清单

### 5.1 后端检查项

- [ ] `events/events.go` 新增 6 个事件常量
- [ ] `runtime/runtime.go` 新增 5 个数据结构
- [ ] `sse_converter.go` 新增 8 个转换方法
- [ ] `phase_detector.go` 新建并实现
- [ ] `orchestrator.go` 集成阶段检测逻辑
- [ ] 单元测试覆盖所有新方法

### 5.2 前端检查项

- [ ] `types.ts` 新增 SSE 事件类型
- [ ] `api/modules/ai.ts` 扩展 handlers 接口
- [ ] `useAIChat.ts` 新增事件处理函数
- [ ] 组件正确渲染步骤状态
- [ ] 单元测试覆盖事件处理

### 5.3 集成测试

- [ ] 正常执行流程：规划 → 执行 → 完成
- [ ] 重规划流程：执行中断 → 重规划 → 继续执行
- [ ] 审批流程：审批请求 → 用户确认 → 继续执行
- [ ] 错误处理：步骤失败 → 错误事件 → 前端展示
