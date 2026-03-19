package runtime

import (
	"encoding/json"
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
		state: ProjectionState{
			Persisted: &PersistedRuntime{},
		},
	}
}

// GetPersistedState 获取累积的持久化状态。
//
// 在流结束时调用此方法获取完整的 runtime 数据用于存储。
func (p *StreamProjector) GetPersistedState() *PersistedRuntime {
	return p.state.Persisted
}

// Consume 消费 ADK 事件，返回需要发送的 SSE 事件。
func (p *StreamProjector) Consume(event *adk.AgentEvent) []PublicStreamEvent {
	normalized := NormalizeAgentEvent(event)
	events := make([]PublicStreamEvent, 0, len(normalized))

	for _, n := range normalized {
		if n.Kind == NormalizedKindToolCall {
			if merged, ok := p.absorbToolCall(n); ok {
				if flushed := p.buffer.Flush(); len(flushed) > 0 {
					events = append(events, flushed...)
				}
				events = append(events, projectNormalizedEvent(merged, &p.state)...)
			}
			continue
		}

		if merged, ok := p.flushPendingToolCall(); ok {
			events = append(events, projectNormalizedEvent(merged, &p.state)...)
		}

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

	if merged, ok := p.flushPendingToolCallIfComplete(); ok {
		events = append(events, projectNormalizedEvent(merged, &p.state)...)
	}

	return events
}

// FlushBuffer 刷新所有缓冲区（公开方法供调用方使用）。
func (p *StreamProjector) FlushBuffer() []PublicStreamEvent {
	events := p.buffer.Flush()
	if merged, ok := p.flushPendingToolCall(); ok {
		events = append(events, projectNormalizedEvent(merged, &p.state)...)
	}
	// 刷新 replanner response 缓冲区
	if flushed := FlushReplannerBuffer(&p.state, "replanner"); len(flushed) > 0 {
		events = append(events, flushed...)
	}
	return events
}

// Finish 返回 done 事件，并设置持久化状态的最终值。
func (p *StreamProjector) Finish(runID string) PublicStreamEvent {
	// 设置最终状态
	statusKind := p.finalStatusKind()
	if p.state.Persisted != nil {
		statusLabel := "已生成"
		if statusKind == "completed_with_tool_errors" {
			statusLabel = "已完成，部分步骤失败"
		}
		p.state.Persisted.Phase = "completed"
		p.state.Persisted.PhaseLabel = "已完成"
		p.state.Persisted.Status = &PersistedStatus{
			Kind:  statusKind,
			Label: statusLabel,
		}
		// 清除活动步骤索引，标记所有步骤为完成
		if p.state.Persisted.Plan != nil {
			p.state.Persisted.Plan.ActiveStepIndex = -1
			for i := range p.state.Persisted.Plan.Steps {
				p.state.Persisted.Plan.Steps[i].Status = "done"
			}
		}
	}
	return doneEvent(runID, p.state.ReplanIteration, statusKind)
}

// Fail 返回 error 事件（保留现有方法）。
func (p *StreamProjector) Fail(runID string, err error) PublicStreamEvent {
	p.state.RunPhase = "failed"
	if p.state.Persisted != nil {
		p.state.Persisted.Phase = "failed_runtime"
		p.state.Persisted.PhaseLabel = "运行失败"
		p.state.Persisted.Status = &PersistedStatus{
			Kind:  "failed_runtime",
			Label: "运行失败",
		}
	}
	return errorEvent(runID, err)
}

func (p *StreamProjector) finalStatusKind() string {
	if p == nil || p.state.Persisted == nil {
		return "completed"
	}
	for _, activity := range p.state.Persisted.Activities {
		if activity.Status == "error" {
			return "completed_with_tool_errors"
		}
	}
	return "completed"
}

func (p *StreamProjector) absorbToolCall(event NormalizedEvent) (NormalizedEvent, bool) {
	if event.Tool == nil {
		return NormalizedEvent{}, false
	}

	rawChunk, hasRaw := extractRawToolArguments(event.Tool.Arguments)
	if p.state.PendingToolCall == nil {
		if strings.TrimSpace(event.Tool.CallID) == "" && strings.TrimSpace(event.Tool.ToolName) == "" && strings.TrimSpace(rawChunk) == "" && !hasRaw {
			return NormalizedEvent{}, false
		}
		p.state.PendingToolCall = &PendingToolCallState{}
	}

	pending := p.state.PendingToolCall
	if strings.TrimSpace(event.AgentName) != "" {
		pending.AgentName = strings.TrimSpace(event.AgentName)
	}
	if strings.TrimSpace(event.Tool.CallID) != "" {
		pending.CallID = strings.TrimSpace(event.Tool.CallID)
	}
	if strings.TrimSpace(event.Tool.ToolName) != "" {
		pending.ToolName = strings.TrimSpace(event.Tool.ToolName)
	}
	if hasRaw {
		pending.ArgumentsRaw += rawChunk
	} else if len(event.Tool.Arguments) > 0 {
		pending.Arguments = event.Tool.Arguments
	}

	if !pendingToolCallComplete(pending) {
		return NormalizedEvent{}, false
	}

	return p.flushPendingToolCall()
}

func (p *StreamProjector) flushPendingToolCallIfComplete() (NormalizedEvent, bool) {
	if !pendingToolCallComplete(p.state.PendingToolCall) {
		return NormalizedEvent{}, false
	}
	return p.flushPendingToolCall()
}

func (p *StreamProjector) flushPendingToolCall() (NormalizedEvent, bool) {
	if p.state.PendingToolCall == nil {
		return NormalizedEvent{}, false
	}

	pending := p.state.PendingToolCall
	p.state.PendingToolCall = nil

	arguments := pending.Arguments
	if len(arguments) == 0 {
		arguments = decodeToolArguments(pending.ArgumentsRaw)
	}

	return NormalizedEvent{
		Kind:      NormalizedKindToolCall,
		AgentName: pending.AgentName,
		Tool: &NormalizedTool{
			CallID:    pending.CallID,
			ToolName:  pending.ToolName,
			Arguments: arguments,
			Phase:     "call",
		},
	}, true
}

func pendingToolCallComplete(pending *PendingToolCallState) bool {
	if pending == nil || strings.TrimSpace(pending.CallID) == "" || strings.TrimSpace(pending.ToolName) == "" {
		return false
	}
	if len(pending.Arguments) > 0 {
		return true
	}
	if strings.TrimSpace(pending.ArgumentsRaw) == "" {
		return false
	}
	var payload map[string]any
	return json.Unmarshal([]byte(pending.ArgumentsRaw), &payload) == nil
}

func extractRawToolArguments(arguments map[string]any) (string, bool) {
	if len(arguments) != 1 {
		return "", false
	}
	raw, ok := arguments["raw"].(string)
	return raw, ok
}
