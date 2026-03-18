# AI 助手回复组件优化设计

**日期**: 2026-03-18
**状态**: 设计阶段

## 概述

优化前端 AI 助手回复组件的三个问题：
1. Steps 显示逻辑 - 当前步骤展开，其他折叠
2. tool_call/tool_result 事件丢失 - 补充后端事件生成
3. 流式输出优化 - 后端缓冲批量发送

## 问题分析

### 问题 1: Steps 显示逻辑

**当前行为**:
- `AssistantReply.tsx` 显示所有 steps
- `activeStepIndex` 控制哪些 steps 可见（index <= activeStepIndex）
- 非当前步骤完全不显示内容

**期望行为**:
- 显示所有 steps 的标题
- 只有 `activeStepIndex` 对应的步骤展开
- 其他步骤折叠显示标题行

**步骤状态来源**:

步骤的 `status` 字段由后端 `reconcilePlan()` 函数（`replyRuntime.ts`）计算：
- `done`: 已完成的步骤（`index < completed`）
- `active`: 当前执行的步骤（`index === completed && !isFinal`）
- `pending`: 未开始的步骤（`index > completed`）

前端不需要额外计算状态，直接使用 `step.status` 即可。

### 问题 2: tool_call/tool_result 事件丢失

**根因分析**:

当前代码路径：`logic.go` → `StreamProjector` → `NormalizeAgentEvent` + `projectNormalizedEvent`

**当前状态**:

| 层级 | tool_call 状态 | tool_result 状态 |
|------|---------------|-----------------|
| normalize.go | 未创建 | 已创建（schema.Tool case，第 107-118 行） |
| project.go | 未处理（落入 default: return nil） | 未处理（落入 default: return nil） |
| streamer.go | 已处理（projectAssistantMessage） | 已处理（projectToolMessage） |

**关键发现**: `streamer.go` 有正确的处理逻辑，但 `StreamProjector` 使用的是 `normalize.go` + `project.go` 路径，而不是 `streamer.go`。

**需要修改**:
1. `normalize.go`: 添加 ToolCalls 提取，创建 `NormalizedKindToolCall` 事件
2. `project.go`: 添加 `NormalizedKindToolCall` 和 `NormalizedKindToolResult` 的投影处理

**期望行为**:
- Executor 发起工具调用时，发送 `tool_call` SSE 事件
- 工具执行完成后，发送 `tool_result` SSE 事件
- 前端能看到工具调用的执行状态

### 问题 3: 流式输出优化

**当前行为**:
- 每个 LLM token 都触发前端更新
- 频繁的 setState 导致性能问题

**期望行为**:
- 后端累积到一定数量或时间后批量发送 delta 事件
- 减少 frontend 刷新频率
- 保持流式体验感

## 设计方案

### 方案 1: Steps 显示优化

**修改文件**: `web/src/components/AI/AssistantReply.tsx`

**改动点**:

1. 修改步骤渲染逻辑，所有步骤都显示标题
2. 只有 `activeStepIndex` 对应的步骤展开 content 和 activities
3. 其他步骤显示折叠状态指示器

**实现细节**:

```tsx
// 当前实现
const visiblePlanSteps = runtime?.plan?.steps?.filter((_, index) => (
  activeStepIndex === undefined ? true : index <= activeStepIndex
)) || [];

// 改为：显示所有步骤
const planSteps = runtime?.plan?.steps || [];

// 渲染逻辑
{planSteps.map((step, index) => {
  const isActive = activeStepIndex === index;
  const isDone = step.status === 'done' || (activeStepIndex !== undefined && index < activeStepIndex);
  const isExpanded = isActive;

  return (
    <div key={step.id}>
      <div className={styles.planStepHeader}>
        {/* 状态图标 */}
        {isDone ? <CheckIcon /> : isActive ? <LoadingIcon /> : <PendingIcon />}
        <span>{step.title}</span>
      </div>
      {/* 只有当前步骤展开 */}
      {isExpanded && (
        <div className={styles.planStepBody}>
          {/* content 和 activities */}
        </div>
      )}
    </div>
  );
})}
```

### 方案 2: tool_call/tool_result 事件补充

**修改文件**:
- `internal/ai/runtime/normalize.go`
- `internal/ai/runtime/project.go`

**改动点**:

#### 2.1 normalize.go 修改

**当前状态**:
- `NormalizedKindToolResult` 已在 `schema.Tool` case 中创建（第 107-118 行）
- `NormalizedKindToolCall` 从未创建，需要添加 ToolCalls 提取

修改 `normalizeMessageOutput()` 函数，在 `schema.Assistant` case 中提取 ToolCalls：

```go
func normalizeMessageOutput(event *adk.AgentEvent) []NormalizedEvent {
    if event == nil || event.Output == nil || event.Output.MessageOutput == nil {
        return nil
    }

    message, err := event.Output.MessageOutput.GetMessage()
    if err != nil || message == nil {
        return nil
    }

    switch message.Role {
    case schema.Assistant:
        events := make([]NormalizedEvent, 0, 2)

        // 1. 添加消息事件（如果有内容）
        if strings.TrimSpace(message.Content) != "" {
            events = append(events, NormalizedEvent{
                Kind:      NormalizedKindMessage,
                AgentName: event.AgentName,
                Message: &NormalizedMessage{
                    Role:        string(message.Role),
                    Content:     message.Content,
                    IsStreaming: event.Output.MessageOutput.IsStreaming,
                },
                Raw: event,
            })
        }

        // 2. 添加工具调用事件（如果有 ToolCalls）- 新增
        for _, tc := range message.ToolCalls {
            events = append(events, NormalizedEvent{
                Kind:      NormalizedKindToolCall,
                AgentName: event.AgentName,
                Tool: &NormalizedTool{
                    CallID:    tc.ID,
                    ToolName:  tc.Function.Name,
                    Arguments: decodeToolArguments(tc.Function.Arguments),
                    Phase:     "call",
                },
                Raw: event,
            })
        }

        return events

    case schema.Tool:
        // 已有实现，无需修改
        return []NormalizedEvent{{
            Kind:      NormalizedKindToolResult,
            AgentName: event.AgentName,
            Tool: &NormalizedTool{
                CallID:   message.ToolCallID,
                ToolName: message.ToolName,
                Content:  message.Content,
                Phase:    "result",
            },
            Raw: event,
        }}

    default:
        return nil
    }
}

// decodeToolArguments 解析工具调用参数 - 新增函数
// 如果 JSON 解析失败，返回包含原始字符串的 map
func decodeToolArguments(raw string) map[string]any {
    if strings.TrimSpace(raw) == "" {
        return map[string]any{}
    }
    var payload map[string]any
    if err := json.Unmarshal([]byte(raw), &payload); err == nil && payload != nil {
        return payload
    }
    return map[string]any{"raw": raw}
}
```

#### 2.2 project.go 修改

**当前状态**: `projectNormalizedEvent()` 的 switch 中没有 `NormalizedKindToolCall` 和 `NormalizedKindToolResult` case，它们落入 `default: return nil`。

添加这两个 case：

```go
func projectNormalizedEvent(event NormalizedEvent, state *ProjectionState) []PublicStreamEvent {
    switch event.Kind {
    case NormalizedKindHandoff:
        // ... 现有逻辑
    case NormalizedKindInterrupt:
        // ... 现有逻辑
    case NormalizedKindToolCall:
        // 新增 case
        if event.Tool == nil {
            return nil
        }
        return []PublicStreamEvent{{
            Event: "tool_call",
            Data: map[string]any{
                "call_id":   event.Tool.CallID,
                "tool_name": event.Tool.ToolName,
                "arguments": event.Tool.Arguments,
            },
        }}
    case NormalizedKindToolResult:
        // 新增 case
        if event.Tool == nil {
            return nil
        }
        return []PublicStreamEvent{{
            Event: "tool_result",
            Data: map[string]any{
                "call_id":   event.Tool.CallID,
                "tool_name": event.Tool.ToolName,
                "content":   event.Tool.Content,
            },
        }}
    case NormalizedKindMessage:
        return projectNormalizedMessage(event, state)
    default:
        return nil
    }
}
```

### 方案 3: 流式输出缓冲

**新增文件**: `internal/ai/runtime/delta_buffer.go`

**修改文件**:
- `internal/ai/runtime/projector.go`
- `internal/service/ai/logic/logic.go`

**设计思路**:

缓冲策略：
- 累积普通 delta 内容，达到阈值或收到非 delta 事件时发送
- planner/replanner 的 envelope 内容**不缓冲**，需要立即解析
- 非事件类型（tool_call, plan, replan 等）到达时，先刷新缓冲区再发送

**为什么选择 MinChunkSize=50, MaxWaitMs=100**:
- 50 字符约等于 1-2 行中文或 5-10 个英文单词，用户能感知到内容更新
- 100ms 是人眼感知的阈值，低于这个值用户感觉是"实时"的
- 可通过配置文件调整

#### 3.1 DeltaBuffer 结构

```go
package runtime

import (
    "strings"
    "sync"
    "time"
)

// DeltaBufferConfig 缓冲配置
type DeltaBufferConfig struct {
    MinChunkSize int // 最小累积字符数，默认 50
    MaxWaitMs    int // 最大等待毫秒数，默认 100
}

// DeltaBuffer 累积 delta 内容并批量发送
//
// 使用方式：
//   - Append(): 添加内容，返回需要立即发送的事件（达到阈值时）
//   - Flush(): 强制刷新所有缓冲内容
//   - ShouldFlushByTime(): 检查是否因超时需要刷新（调用方定时检查）
type DeltaBuffer struct {
    config    DeltaBufferConfig
    mu        sync.Mutex
    content   strings.Builder
    agent     string
    lastAppend time.Time
}

func NewDeltaBuffer(config DeltaBufferConfig) *DeltaBuffer {
    if config.MinChunkSize <= 0 {
        config.MinChunkSize = 50
    }
    if config.MaxWaitMs <= 0 {
        config.MaxWaitMs = 100
    }
    return &DeltaBuffer{
        config: config,
    }
}

// Append 添加 delta 内容
// 返回值：需要立即发送的事件（达到 MinChunkSize 阈值时）
func (b *DeltaBuffer) Append(content, agent string) []PublicStreamEvent {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.content.WriteString(content)
    if b.agent == "" {
        b.agent = agent
    }
    b.lastAppend = time.Now()

    // 只有达到 MinChunkSize 才立即发送
    if b.content.Len() >= b.config.MinChunkSize {
        return b.flush()
    }
    return nil
}

// ShouldFlushByTime 检查是否因超时需要刷新
// 返回值：true 表示有超时内容需要发送
func (b *DeltaBuffer) ShouldFlushByTime() bool {
    b.mu.Lock()
    defer b.mu.Unlock()

    if b.content.Len() == 0 {
        return false
    }
    elapsed := time.Since(b.lastAppend).Milliseconds()
    return elapsed >= int64(b.config.MaxWaitMs)
}

// Flush 强制刷新剩余内容
func (b *DeltaBuffer) Flush() []PublicStreamEvent {
    b.mu.Lock()
    defer b.mu.Unlock()
    return b.flush()
}

func (b *DeltaBuffer) flush() []PublicStreamEvent {
    content := b.content.String()
    if content == "" {
        return nil
    }

    event := PublicStreamEvent{
        Event: "delta",
        Data: map[string]any{
            "content": content,
            "agent":   b.agent,
        },
    }

    b.content.Reset()
    b.agent = ""

    return []PublicStreamEvent{event}
}
```

#### 3.2 StreamProjector 集成

```go
type StreamProjector struct {
    state  ProjectionState
    buffer *DeltaBuffer
}

func NewStreamProjector() *StreamProjector {
    return &StreamProjector{
        buffer: NewDeltaBuffer(DeltaBufferConfig{
            MinChunkSize: 50,
            MaxWaitMs:    100,
        }),
    }
}

// Consume 消费 ADK 事件，返回需要发送的 SSE 事件
func (p *StreamProjector) Consume(event *adk.AgentEvent) []PublicStreamEvent {
    normalized := NormalizeAgentEvent(event)
    events := make([]PublicStreamEvent, 0, len(normalized))

    for _, n := range normalized {
        // 只有普通 agent 的 delta 事件走缓冲
        // planner/replanner 需要立即解析 envelope，不缓冲
        if n.Kind == NormalizedKindMessage && n.Message != nil {
            agent := strings.TrimSpace(n.AgentName)
            if agent != "planner" && agent != "replanner" {
                // 累积到缓冲区
                if buffered := p.buffer.Append(n.Message.Content, agent); len(buffered) > 0 {
                    events = append(events, buffered...)
                }
                continue
            }
        }

        // 非 delta 事件：先刷新缓冲区，再发送当前事件
        if flushed := p.buffer.Flush(); len(flushed) > 0 {
            events = append(events, flushed...)
        }
        events = append(events, projectNormalizedEvent(n, &p.state)...)
    }

    // 检查超时刷新
    if p.buffer.ShouldFlushByTime() {
        if flushed := p.buffer.Flush(); len(flushed) > 0 {
            events = append(events, flushed...)
        }
    }

    return events
}

// FlushBuffer 刷新缓冲区（公开方法供调用方使用）
func (p *StreamProjector) FlushBuffer() []PublicStreamEvent {
    return p.buffer.Flush()
}

func (p *StreamProjector) Finish(runID string) PublicStreamEvent {
    return doneEvent(runID, p.state.ReplanIteration)
}
```

#### 3.3 logic.go 调整

在 `Chat()` 和 `ResumeApproval()` 方法中，发送 done 事件前先刷新缓冲区：

```go
// Step 8: 发送 done 事件前先刷新缓冲
if remaining := projector.FlushBuffer(); len(remaining) > 0 {
    for _, e := range remaining {
        emit(e.Event, e.Data)
    }
}
done := projector.Finish(run.ID)
emit(done.Event, done.Data)
```

#### 3.4 与现有 Envelope 缓冲的关系

现有 `ProjectionState` 中的 `PendingPlannerJSON` 和 `PendingReplannerJSON` 用于缓冲 planner/replanner 的 JSON envelope，以便完整解析 `{"steps": [...]}` 或 `{"response": "..."}`。

新增的 `DeltaBuffer` 只缓冲普通 agent 的文本内容，两者互不干扰：

| 缓冲类型 | 用途 | 内容来源 |
|---------|------|---------|
| PendingPlannerJSON | 解析完整 JSON envelope | planner agent |
| PendingReplannerJSON | 解析完整 JSON envelope | replanner agent |
| DeltaBuffer | 批量发送文本内容 | 其他 agent (executor, diagnosis 等) |

## 实现计划

### Phase 1: 后端事件补充（优先级最高）
1. 修改 `normalize.go` 添加 ToolCalls 提取
2. 修改 `project.go` 添加 tool_call/tool_result 事件发送
3. 编写单元测试验证

### Phase 2: 流式输出优化
1. 新增 `delta_buffer.go` 缓冲组件
2. 修改 `projector.go` 集成缓冲逻辑
3. 修改 `logic.go` 处理 Finish 时的刷新
4. 编写单元测试验证

### Phase 3: 前端 Steps 显示
1. 修改 `AssistantReply.tsx` 渲染逻辑
2. 添加步骤状态图标（完成/进行中/等待）
3. 编写组件测试验证

## 风险评估

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 缓冲延迟过高 | 用户体验下降 | 配置合理的默认值（50字符/100ms） |
| tool_call 事件过多 | 前端渲染压力 | 前端限制显示数量 |
| Steps 状态不同步 | UI 显示错误 | 确保 activeStepIndex 实时更新 |

## 测试计划

### 后端单元测试

#### normalize_test.go

```go
func TestNormalizeAgentEvent_AssistantWithToolCalls(t *testing.T) {
    tests := []struct {
        name      string
        event     *adk.AgentEvent
        wantKinds []NormalizedKind
        wantLen   int
    }{
        {
            name: "assistant with content and tool_calls",
            event: &adk.AgentEvent{
                AgentName: "executor",
                Output: &adk.AgentOutput{
                    MessageOutput: &adk.MessageVariant{
                        Message: &schema.Message{
                            Role:    schema.Assistant,
                            Content: "some content",
                            ToolCalls: []schema.ToolCall{
                                {ID: "call-1", Function: schema.FunctionCall{Name: "tool_a", Arguments: `{"arg": "value"}`}},
                            },
                        },
                    },
                },
            },
            wantKinds: []NormalizedKind{NormalizedKindMessage, NormalizedKindToolCall},
            wantLen:   2,
        },
        {
            name: "assistant with only tool_calls",
            event: &adk.AgentEvent{
                AgentName: "executor",
                Output: &adk.AgentOutput{
                    MessageOutput: &adk.MessageVariant{
                        Message: &schema.Message{
                            Role:      schema.Assistant,
                            ToolCalls: []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "tool_a"}}},
                        },
                    },
                },
            },
            wantKinds: []NormalizedKind{NormalizedKindToolCall},
            wantLen:   1,
        },
        {
            name: "assistant with nil tool_calls",
            event: &adk.AgentEvent{
                AgentName: "executor",
                Output: &adk.AgentOutput{
                    MessageOutput: &adk.MessageVariant{
                        Message: &schema.Message{
                            Role:      schema.Assistant,
                            Content:   "content",
                            ToolCalls: nil,
                        },
                    },
                },
            },
            wantKinds: []NormalizedKind{NormalizedKindMessage},
            wantLen:   1,
        },
        {
            name: "assistant with empty tool_calls",
            event: &adk.AgentEvent{
                AgentName: "executor",
                Output: &adk.AgentOutput{
                    MessageOutput: &adk.MessageVariant{
                        Message: &schema.Message{
                            Role:      schema.Assistant,
                            Content:   "content",
                            ToolCalls: []schema.ToolCall{},
                        },
                    },
                },
            },
            wantKinds: []NormalizedKind{NormalizedKindMessage},
            wantLen:   1,
        },
    }
    // ... test implementation
}

func TestDecodeToolArguments(t *testing.T) {
    tests := []struct {
        name string
        raw  string
        want map[string]any
    }{
        {name: "valid json", raw: `{"key": "value"}`, want: map[string]any{"key": "value"}},
        {name: "invalid json", raw: `not json`, want: map[string]any{"raw": "not json"}},
        {name: "empty string", raw: ``, want: map[string]any{}},
        {name: "nested json", raw: `{"nested": {"key": "value"}}`, want: map[string]any{"nested": map[string]any{"key": "value"}}},
    }
    // ... test implementation
}
```

#### project_test.go

```go
func TestProjectNormalizedEvent_ToolCall(t *testing.T) {
    event := NormalizedEvent{
        Kind:      NormalizedKindToolCall,
        AgentName: "executor",
        Tool: &NormalizedTool{
            CallID:    "call-123",
            ToolName:  "k8s_query",
            Arguments: map[string]any{"namespace": "default"},
        },
    }

    got := projectNormalizedEvent(event, &ProjectionState{})

    if len(got) != 1 {
        t.Fatalf("expected 1 event, got %d", len(got))
    }
    if got[0].Event != "tool_call" {
        t.Fatalf("expected event tool_call, got %s", got[0].Event)
    }
    // verify data fields
}

func TestProjectNormalizedEvent_ToolResult(t *testing.T) {
    // Similar test for tool_result
}
```

#### delta_buffer_test.go

```go
func TestDeltaBuffer_Append_MinChunkSize(t *testing.T) {
    buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 10, MaxWaitMs: 1000})

    // Under threshold - no flush
    got := buf.Append("hello", "agent")
    if len(got) != 0 {
        t.Fatalf("expected no flush, got %d events", len(got))
    }

    // Reach threshold - should flush
    got = buf.Append(" world!!!", "agent") // total 14 chars
    if len(got) != 1 {
        t.Fatalf("expected 1 event, got %d", len(got))
    }
}

func TestDeltaBuffer_Flush(t *testing.T) {
    buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 100, MaxWaitMs: 1000})

    buf.Append("some content", "agent")
    got := buf.Flush()

    if len(got) != 1 {
        t.Fatalf("expected 1 event, got %d", len(got))
    }
    // Verify content
}

func TestDeltaBuffer_ShouldFlushByTime(t *testing.T) {
    buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 1000, MaxWaitMs: 50})

    buf.Append("content", "agent")
    time.Sleep(60 * time.Millisecond)

    if !buf.ShouldFlushByTime() {
        t.Fatal("expected ShouldFlushByTime to return true")
    }
}
```

### 集成测试

验证完整事件流：
```
meta -> plan -> tool_call -> tool_result -> delta (buffered) -> done
```

### 前端测试

#### AssistantReply.test.tsx

```tsx
describe('AssistantReply steps display', () => {
  it('shows all steps with titles', () => {
    const runtime = {
      activities: [],
      plan: {
        steps: [
          { id: '1', title: 'Step 1', status: 'done' },
          { id: '2', title: 'Step 2', status: 'active' },
          { id: '3', title: 'Step 3', status: 'pending' },
        ],
        activeStepIndex: 1,
      },
    };

    render(<AssistantReply content="" runtime={runtime} />);

    // All steps should have their titles visible
    expect(screen.getByText('Step 1')).toBeVisible();
    expect(screen.getByText('Step 2')).toBeVisible();
    expect(screen.getByText('Step 3')).toBeVisible();
  });

  it('expands only active step', () => {
    const runtime = {
      activities: [],
      plan: {
        steps: [
          { id: '1', title: 'Step 1', status: 'done', content: 'done content' },
          { id: '2', title: 'Step 2', status: 'active', content: 'active content' },
        ],
        activeStepIndex: 1,
      },
    };

    render(<AssistantReply content="" runtime={runtime} />);

    // Only active step content should be visible
    expect(screen.queryByText('done content')).not.toBeInTheDocument();
    expect(screen.getByText('active content')).toBeVisible();
  });
});
```
