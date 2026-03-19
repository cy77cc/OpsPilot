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

// FlushBuffer 刷新所有缓冲区（公开方法供调用方使用）。
func (p *StreamProjector) FlushBuffer() []PublicStreamEvent {
	events := p.buffer.Flush()
	// 刷新 replanner response 缓冲区
	if flushed := FlushReplannerBuffer(&p.state, "replanner"); len(flushed) > 0 {
		events = append(events, flushed...)
	}
	return events
}

// Finish 返回 done 事件，并设置持久化状态的最终值。
func (p *StreamProjector) Finish(runID string) PublicStreamEvent {
	// 设置最终状态
	if p.state.Persisted != nil {
		p.state.Persisted.Phase = "completed"
		p.state.Persisted.PhaseLabel = "已完成"
		p.state.Persisted.Status = &PersistedStatus{
			Kind:  "completed",
			Label: "已生成",
		}
		// 清除活动步骤索引，标记所有步骤为完成
		if p.state.Persisted.Plan != nil {
			p.state.Persisted.Plan.ActiveStepIndex = -1
			for i := range p.state.Persisted.Plan.Steps {
				p.state.Persisted.Plan.Steps[i].Status = "done"
			}
		}
	}
	return doneEvent(runID, p.state.ReplanIteration)
}

// Fail 返回 error 事件（保留现有方法）。
func (p *StreamProjector) Fail(runID string, err error) PublicStreamEvent {
	// 刷新缓冲区
	p.buffer.Flush()
	p.state.RunPhase = "failed"
	return errorEvent(runID, err)
}
