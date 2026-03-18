# AI 助手回复组件优化实现计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 优化前端 AI 助手回复组件，修复 tool_call/tool_result 事件丢失问题，实现流式输出缓冲优化。

**Architecture:**
- 后端新增 ToolCalls 提取逻辑和 DeltaBuffer 缓冲组件
- 前端修改 Steps 显示逻辑，所有步骤显示但只有当前步骤展开

**Tech Stack:** Go 1.26, React 19, Ant Design 6

---

## 文件结构

```
internal/ai/runtime/
├── delta_buffer.go       # 新增 - delta 缓冲组件
├── delta_buffer_test.go  # 新增 - delta 缓冲测试
├── normalize.go          # 修改 - 添加 ToolCalls 提取
├── normalize_test.go     # 修改 - 添加 ToolCalls 测试
├── project.go            # 修改 - 添加 tool_call/tool_result 投影
├── project_test.go       # 修改 - 添加投影测试
└── projector.go          # 修改 - 集成 DeltaBuffer

internal/service/ai/logic/
└── logic.go              # 修改 - 调用 FlushBuffer

web/src/components/AI/
├── AssistantReply.tsx    # 修改 - Steps 显示逻辑
└── __tests__/
    └── AssistantReply.test.tsx  # 修改 - 更新测试 + 添加新测试
```

---

## Chunk 1: 后端 tool_call/tool_result 事件补充

### Task 1: normalize.go - 添加 ToolCalls 提取

**Files:**
- Modify: `internal/ai/runtime/normalize.go`
- Test: `internal/ai/runtime/normalize_test.go`

- [ ] **Step 1: 编写 ToolCalls 提取的测试用例**

在 `internal/ai/runtime/normalize_test.go` 添加测试：

**重要**: `decodeToolArguments` 函数已存在于 `streamer.go` (同一 package)，可直接使用，无需重复定义。

```go
func TestNormalizeAgentEvent_AssistantWithToolCalls(t *testing.T) {
	tests := []struct {
		name       string
		event      *adk.AgentEvent
		wantKinds  []NormalizedKind
		wantLen    int
		wantToolID string
	}{
		{
			name: "assistant with content and tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:    schema.Assistant,
						Content: "some content",
						ToolCalls: []schema.ToolCall{
							{ID: "call-1", Function: schema.FunctionCall{Name: "tool_a", Arguments: `{"arg": "value"}`}},
						},
					}},
				},
			},
			wantKinds:  []NormalizedKind{NormalizedKindMessage, NormalizedKindToolCall},
			wantLen:    2,
			wantToolID: "call-1",
		},
		{
			name: "assistant with only tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:      schema.Assistant,
						ToolCalls: []schema.ToolCall{{ID: "call-1", Function: schema.FunctionCall{Name: "tool_a"}}},
					}},
				},
			},
			wantKinds:  []NormalizedKind{NormalizedKindToolCall},
			wantLen:    1,
			wantToolID: "call-1",
		},
		{
			name: "assistant with nil tool_calls",
			event: &adk.AgentEvent{
				AgentName: "executor",
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:      schema.Assistant,
						Content:   "content",
						ToolCalls: nil,
					}},
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
					MessageOutput: &adk.MessageVariant{Message: &schema.Message{
						Role:      schema.Assistant,
						Content:   "content",
						ToolCalls: []schema.ToolCall{},
					}},
				},
			},
			wantKinds: []NormalizedKind{NormalizedKindMessage},
			wantLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeAgentEvent(tt.event)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d events, got %d", tt.wantLen, len(got))
			}
			for i, kind := range tt.wantKinds {
				if got[i].Kind != kind {
					t.Errorf("event %d: expected kind %s, got %s", i, kind, got[i].Kind)
				}
				// 验证 tool call ID
				if kind == NormalizedKindToolCall && tt.wantToolID != "" {
					if got[i].Tool == nil || got[i].Tool.CallID != tt.wantToolID {
						t.Errorf("expected tool call_id=%s, got %v", tt.wantToolID, got[i].Tool)
					}
				}
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/ai/runtime/... -run TestNormalizeAgentEvent_AssistantWithToolCalls -v`
Expected: FAIL (ToolCalls not extracted yet)

- [ ] **Step 3: 实现 ToolCalls 提取**

在 `internal/ai/runtime/normalize.go` 中修改：

首先更新 import 块（添加 `strings`）：

```go
import (
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)
```

然后修改 `normalizeMessageOutput()` 函数：

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

		// 添加消息事件（如果有内容）
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

		// 添加工具调用事件（如果有 ToolCalls）
		// 使用 streamer.go 中已存在的 decodeToolArguments 函数（同一 package）
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
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/ai/runtime/... -run TestNormalizeAgentEvent_AssistantWithToolCalls -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/normalize.go internal/ai/runtime/normalize_test.go
git commit -m "feat(ai): add ToolCalls extraction in normalizeMessageOutput

- Extract ToolCalls from assistant messages
- Create NormalizedKindToolCall events
- Reuse existing decodeToolArguments from streamer.go

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 2: project.go - 添加 tool_call/tool_result 投影

**Files:**
- Modify: `internal/ai/runtime/project.go`
- Test: `internal/ai/runtime/project_test.go`

- [ ] **Step 1: 编写投影测试用例**

在 `internal/ai/runtime/project_test.go` 添加测试：

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

	data, ok := got[0].Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be map[string]any")
	}
	if data["call_id"] != "call-123" {
		t.Errorf("expected call_id=call-123, got %v", data["call_id"])
	}
	if data["tool_name"] != "k8s_query" {
		t.Errorf("expected tool_name=k8s_query, got %v", data["tool_name"])
	}
}

func TestProjectNormalizedEvent_ToolResult(t *testing.T) {
	event := NormalizedEvent{
		Kind:      NormalizedKindToolResult,
		AgentName: "executor",
		Tool: &NormalizedTool{
			CallID:   "call-123",
			ToolName: "k8s_query",
			Content:  "tool output",
		},
	}

	got := projectNormalizedEvent(event, &ProjectionState{})

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Event != "tool_result" {
		t.Fatalf("expected event tool_result, got %s", got[0].Event)
	}

	data, ok := got[0].Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be map[string]any")
	}
	if data["content"] != "tool output" {
		t.Errorf("expected content='tool output', got %v", data["content"])
	}
}

func TestProjectNormalizedEvent_ToolCall_NilTool(t *testing.T) {
	event := NormalizedEvent{
		Kind:      NormalizedKindToolCall,
		AgentName: "executor",
		Tool:      nil,
	}

	got := projectNormalizedEvent(event, &ProjectionState{})

	if len(got) != 0 {
		t.Fatalf("expected 0 events for nil tool, got %d", len(got))
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/ai/runtime/... -run TestProjectNormalizedEvent_ToolCall -v`
Expected: FAIL (NormalizedKindToolCall not handled)

- [ ] **Step 3: 实现投影处理**

在 `internal/ai/runtime/project.go` 的 `projectNormalizedEvent()` 函数中添加 case：

```go
case NormalizedKindToolCall:
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
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/ai/runtime/... -run TestProjectNormalizedEvent_Tool -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/project.go internal/ai/runtime/project_test.go
git commit -m "feat(ai): add tool_call and tool_result event projection

- Add NormalizedKindToolCall case in projectNormalizedEvent
- Add NormalizedKindToolResult case in projectNormalizedEvent
- Handle nil tool gracefully

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: 流式输出缓冲优化

### Task 3: 新增 DeltaBuffer 组件

**Files:**
- Create: `internal/ai/runtime/delta_buffer.go`
- Create: `internal/ai/runtime/delta_buffer_test.go`

- [ ] **Step 1: 编写 DeltaBuffer 测试用例**

创建 `internal/ai/runtime/delta_buffer_test.go`：

```go
package runtime

import (
	"testing"
	"time"
)

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
	if got[0].Event != "delta" {
		t.Errorf("expected event delta, got %s", got[0].Event)
	}

	data := got[0].Data.(map[string]any)
	if data["content"] != "hello world!!!" {
		t.Errorf("expected content='hello world!!!', got %v", data["content"])
	}
}

func TestDeltaBuffer_Flush(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 100, MaxWaitMs: 1000})

	buf.Append("some content", "agent")
	got := buf.Flush()

	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Event != "delta" {
		t.Errorf("expected event delta, got %s", got[0].Event)
	}
}

func TestDeltaBuffer_Flush_Empty(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{})

	got := buf.Flush()
	if len(got) != 0 {
		t.Fatalf("expected 0 events for empty buffer, got %d", len(got))
	}
}

func TestDeltaBuffer_ShouldFlushByTime(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{MinChunkSize: 1000, MaxWaitMs: 50})

	buf.Append("content", "agent")

	// Immediately after append - should not flush
	if buf.ShouldFlushByTime() {
		t.Fatal("expected ShouldFlushByTime to return false immediately")
	}

	time.Sleep(60 * time.Millisecond)

	if !buf.ShouldFlushByTime() {
		t.Fatal("expected ShouldFlushByTime to return true after timeout")
	}
}

func TestDeltaBuffer_DefaultConfig(t *testing.T) {
	buf := NewDeltaBuffer(DeltaBufferConfig{})

	// Default values should be applied
	if buf.config.MinChunkSize != 50 {
		t.Errorf("expected default MinChunkSize=50, got %d", buf.config.MinChunkSize)
	}
	if buf.config.MaxWaitMs != 100 {
		t.Errorf("expected default MaxWaitMs=100, got %d", buf.config.MaxWaitMs)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./internal/ai/runtime/... -run TestDeltaBuffer -v`
Expected: FAIL (DeltaBuffer not implemented)

- [ ] **Step 3: 实现 DeltaBuffer**

创建 `internal/ai/runtime/delta_buffer.go`：

```go
// Package runtime 提供 AI 运行时的 delta 缓冲组件。
//
// DeltaBuffer 累积 delta 内容并批量发送，减少前端刷新频率。
package runtime

import (
	"strings"
	"sync"
	"time"
)

// DeltaBufferConfig 缓冲配置。
type DeltaBufferConfig struct {
	MinChunkSize int // 最小累积字符数，默认 50
	MaxWaitMs    int // 最大等待毫秒数，默认 100
}

// DeltaBuffer 累积 delta 内容并批量发送。
type DeltaBuffer struct {
	config     DeltaBufferConfig
	mu         sync.Mutex
	content    strings.Builder
	agent      string
	lastAppend time.Time
}

// NewDeltaBuffer 创建 DeltaBuffer 实例。
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

// Append 添加 delta 内容。
// 返回值：需要立即发送的事件（达到 MinChunkSize 阈值时）。
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

// ShouldFlushByTime 检查是否因超时需要刷新。
func (b *DeltaBuffer) ShouldFlushByTime() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.content.Len() == 0 {
		return false
	}
	elapsed := time.Since(b.lastAppend).Milliseconds()
	return elapsed >= int64(b.config.MaxWaitMs)
}

// Flush 强制刷新剩余内容。
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

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./internal/ai/runtime/... -run TestDeltaBuffer -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/runtime/delta_buffer.go internal/ai/runtime/delta_buffer_test.go
git commit -m "feat(ai): add DeltaBuffer for streaming optimization

- Buffer delta content until MinChunkSize or MaxWaitMs
- Reduce frontend refresh frequency
- Thread-safe implementation

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 4: StreamProjector 集成 DeltaBuffer

**Files:**
- Modify: `internal/ai/runtime/projector.go`

- [ ] **Step 1: 修改 StreamProjector 结构**

在 `internal/ai/runtime/projector.go` 中修改。**注意**: 保留现有的 `Fail` 方法。

```go
package runtime

import (
	"strings"

	"github.com/cloudwego/eino/adk"
)

// StreamProjector 消费 ADK 事件并投影为前端可消费的 SSE 事件。
type StreamProjector struct {
	state  ProjectionState
	buffer *DeltaBuffer
}

// NewStreamProjector 创建 StreamProjector 实例。
func NewStreamProjector() *StreamProjector {
	return &StreamProjector{
		buffer: NewDeltaBuffer(DeltaBufferConfig{
			MinChunkSize: 50,
			MaxWaitMs:    100,
		}),
	}
}

// Consume 消费 ADK 事件，返回需要发送的 SSE 事件。
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

// FlushBuffer 刷新缓冲区（公开方法供调用方使用）。
func (p *StreamProjector) FlushBuffer() []PublicStreamEvent {
	return p.buffer.Flush()
}

// Finish 返回 done 事件。
func (p *StreamProjector) Finish(runID string) PublicStreamEvent {
	return doneEvent(runID, p.state.ReplanIteration)
}

// Fail 返回 error 事件（保留现有方法）。
func (p *StreamProjector) Fail(runID string, err error) PublicStreamEvent {
	// 刷新缓冲区
	p.buffer.Flush()
	p.state.RunPhase = "failed"
	return errorEvent(runID, err)
}
```

- [ ] **Step 2: 运行测试验证**

Run: `go test ./internal/ai/runtime/... -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ai/runtime/projector.go
git commit -m "feat(ai): integrate DeltaBuffer into StreamProjector

- Buffer delta events for non-planner/replanner agents
- Flush buffer before non-delta events
- Add FlushBuffer method for final flush

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

### Task 5: logic.go 集成

**Files:**
- Modify: `internal/service/ai/logic/logic.go`

- [ ] **Step 1: 修改 Chat() 方法的结束逻辑**

在 `internal/service/ai/logic/logic.go` 的 `Chat()` 方法中，找到发送 done 事件的部分（约第 282-284 行），修改为：

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

- [ ] **Step 2: 修改 ResumeApproval() 方法的结束逻辑**

同样在 `ResumeApproval()` 方法中（约第 815-817 行），修改为：

```go
	// 刷新缓冲区
	if remaining := projector.FlushBuffer(); len(remaining) > 0 {
		for _, e := range remaining {
			emit(e.Event, e.Data)
		}
	}
	done := projector.Finish(task.RunID)
	emit(done.Event, done.Data)
```

- [ ] **Step 3: 运行测试验证**

Run: `go test ./internal/service/ai/... -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/service/ai/logic/logic.go
git commit -m "feat(ai): flush DeltaBuffer before done event

- Call FlushBuffer before sending done in Chat()
- Call FlushBuffer before sending done in ResumeApproval()

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: 前端 Steps 显示优化

### Task 6: AssistantReply.tsx 修改

**Files:**
- Modify: `web/src/components/AI/AssistantReply.tsx`
- Test: `web/src/components/AI/__tests__/AssistantReply.test.tsx`

- [ ] **Step 1: 更新现有测试用例**

**重要**: 现有测试 `only reveals the current step while hiding future steps` 需要修改以匹配新行为。

在 `web/src/components/AI/__tests__/AssistantReply.test.tsx` 中：

1. 找到 `only reveals the current step while hiding future steps` 测试
2. 修改为 `shows all steps but only expands current step`

```tsx
it('shows all steps but only expands current step', () => {
  const runtime: AssistantReplyRuntime = {
    activities: [],
    plan: {
      steps: [
        { id: '1', title: '检查服务器状态', status: 'done' },
        { id: '2', title: '批量执行健康检查', status: 'active' },
        { id: '3', title: '汇总检查结果', status: 'pending' },
      ],
      activeStepIndex: 1,
    },
  };

  render(<AssistantReply content="" runtime={runtime} status="loading" />);

  // 所有步骤标题都应该可见
  expect(screen.getByText('检查服务器状态')).toBeVisible();
  expect(screen.getByText('批量执行健康检查')).toBeVisible();
  expect(screen.getByText('汇总检查结果')).toBeVisible();
});
```

- [ ] **Step 2: 添加新的测试用例**

```tsx
describe('AssistantReply steps display', () => {
  it('expands only active step content', () => {
    const runtime = {
      activities: [],
      plan: {
        steps: [
          { id: '1', title: 'Step 1', status: 'done' as const, content: 'done content' },
          { id: '2', title: 'Step 2', status: 'active' as const, content: 'active content' },
        ],
        activeStepIndex: 1,
      },
    };

    render(<AssistantReply content="" runtime={runtime} status="loading" />);

    // 只有当前步骤内容应该可见
    expect(screen.queryByText('done content')).not.toBeInTheDocument();
    expect(screen.getByText('active content')).toBeVisible();
  });

  it('shows all steps collapsed when activeStepIndex is undefined', () => {
    const runtime = {
      activities: [],
      plan: {
        steps: [
          { id: '1', title: 'Step 1', status: 'done' as const, content: 'content 1' },
          { id: '2', title: 'Step 2', status: 'done' as const, content: 'content 2' },
        ],
        activeStepIndex: undefined,
      },
    };

    render(<AssistantReply content="" runtime={runtime} status="success" />);

    // 所有步骤标题都应该可见
    expect(screen.getByText('Step 1')).toBeVisible();
    expect(screen.getByText('Step 2')).toBeVisible();
    // 但没有内容展开
    expect(screen.queryByText('content 1')).not.toBeInTheDocument();
    expect(screen.queryByText('content 2')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 3: 运行测试验证失败**

Run: `cd web && npm test -- --testPathPattern="AssistantReply.test" -u`
Expected: FAIL (steps logic not updated)

- [ ] **Step 4: 修改 AssistantReply.tsx**

修改 `web/src/components/AI/AssistantReply.tsx`：

```tsx
// 修改 visiblePlanSteps 逻辑，显示所有步骤
const planSteps = runtime?.plan?.steps || [];

// 修改渲染逻辑
{planSteps.length ? (
  <div className={styles.planSteps}>
    {planSteps.map((step, index) => {
      const isActive = activeStepIndex === index;
      const isDone = step.status === 'done';
      const isExpanded = isActive && activeStepIndex !== undefined;

      return (
        <div key={step.id} className={styles.planStep}>
          <div className={styles.planStepHeader}>
            <span className={styles.planStepLine} />
            <span className={styles.planStepTitle}>
              {isDone ? '✓ ' : isActive ? '◐ ' : '○ '}
              {step.title}
            </span>
            <span className={styles.planStepLine} />
          </div>
          {isExpanded ? (
            <div className={styles.planStepBody}>
              {step.content ? (
                <div className={styles.stepMarkdown}>
                  <XMarkdown
                    content={step.content}
                    streaming={{
                      hasNextChunk: status === 'loading' || status === 'updating',
                      enableAnimation: true,
                      animationConfig: {
                        fadeDuration: 180,
                        easing: 'ease-out',
                      },
                    }}
                  />
                </div>
              ) : null}
              {runtime?.activities?.filter((activity) => activity.stepIndex === index).map((activity) => (
                <div key={activity.id} className={styles.activity}>
                  <span>{activity.label}</span>
                  {activity.detail ? <span className={styles.activityDetail}>{activity.detail}</span> : null}
                </div>
              )) || <div className={styles.activityDetail}>执行中</div>}
            </div>
          ) : null}
        </div>
      );
    })}
  </div>
) : null}
```

- [ ] **Step 5: 运行测试验证通过**

Run: `cd web && npm test -- --testPathPattern="AssistantReply.test" -u`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/components/AI/AssistantReply.tsx web/src/components/AI/__tests__/AssistantReply.test.tsx
git commit -m "feat(web): show all steps with only active expanded

- Display all step titles regardless of status
- Only expand current active step content
- Show status indicator (✓ done, ◐ active, ○ pending)
- Update existing test to match new behavior

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 验收清单

- [ ] 运行所有后端测试通过: `make test`
- [ ] 运行所有前端测试通过: `make web-test`
- [ ] 手动测试 AI 对话功能，验证:
  - [ ] Steps 显示所有步骤，只有当前步骤展开
  - [ ] tool_call/tool_result 事件正确发送
  - [ ] 流式输出不会过于频繁刷新
