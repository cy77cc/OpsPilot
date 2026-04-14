// Package stream 实现 AI Agent 迭代器处理和事件流投影。
package stream

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/modules/ai/agent/runtime"
)

// IteratorConsumeKind 定义迭代器事件类型。
type IteratorConsumeKind string

const (
	IteratorConsumeInterrupt       IteratorConsumeKind = "interrupt"
	IteratorConsumeRecoverableTool IteratorConsumeKind = "recoverable_tool_error"
	IteratorConsumeStreamTool      IteratorConsumeKind = "stream_tool_error"
	IteratorConsumeStreamChunk     IteratorConsumeKind = "stream_chunk"
	IteratorConsumeEvent           IteratorConsumeKind = "event"
	IteratorConsumeFlush           IteratorConsumeKind = "flush"
)

// RunUpdate 定义运行状态更新。
type RunUpdate struct {
	AssistantType string
	IntentType    string
}

// IteratorProcessResult 定义迭代器处理结果。
type IteratorProcessResult struct {
	Interrupted       bool
	HasToolErrors     bool
	CircuitBroken     bool
	SummaryText       string
	AssistantSnapshot string
	FatalErr          error
}

// IteratorProcessInput 定义迭代器处理输入。
type IteratorProcessInput struct {
	Iterator         *adk.AsyncIterator[*adk.AgentEvent]
	Projector        *airuntime.StreamProjector
	Emit             func(event string, data any)
	ConsumeProjected func(kind IteratorConsumeKind, events []airuntime.PublicStreamEvent) error
	HandleRunUpdate  func(update RunUpdate)
}

// ProcessAgentIterator 处理 Agent 异步迭代器事件流。
func ProcessAgentIterator(_ context.Context, input IteratorProcessInput) (IteratorProcessResult, error) {
	result := IteratorProcessResult{}
	if input.Iterator == nil {
		return result, nil
	}
	if input.Projector == nil {
		input.Projector = airuntime.NewStreamProjector()
	}
	if input.Emit == nil {
		input.Emit = func(string, any) {}
	}
	if input.HandleRunUpdate == nil {
		input.HandleRunUpdate = func(RunUpdate) {}
	}

	var (
		summaryContent    strings.Builder
		assistantSnapshot strings.Builder
		toolFailures      = newToolFailureTracker()
	)

	processProjected := func(kind IteratorConsumeKind, events []airuntime.PublicStreamEvent) error {
		if len(events) == 0 {
			return nil
		}
		update := AccumulateProjectedEvents(events, &summaryContent)
		if input.ConsumeProjected != nil {
			if err := input.ConsumeProjected(kind, events); err != nil {
				return wrapIteratorConsumeError(kind, err)
			}
		} else {
			for _, event := range events {
				input.Emit(event.Event, event.Data)
			}
		}
		if update.AssistantType != "" || update.IntentType != "" {
			input.HandleRunUpdate(update)
		}
		return nil
	}

	flushProjected := func() error {
		events := input.Projector.FlushBuffer()
		toolFailures.recordProjectedEvents(events)
		if err := processProjected(IteratorConsumeFlush, events); err != nil {
			return err
		}
		result.SummaryText = summaryContent.String()
		result.AssistantSnapshot = assistantSnapshot.String()
		return nil
	}

	for {
		if persisted := input.Projector.GetPersistedState(); persisted != nil && !persisted.CanFinalizeDone() {
			result.Interrupted = true
			break
		}
		event, ok := input.Iterator.Next()
		if !ok {
			break
		}
		if interruptEvent, ok := RecoverableInterruptEventFromEvent(event); ok {
			projected := input.Projector.Consume(interruptEvent)
			toolFailures.recordProjectedEvents(projected)
			if err := processProjected(IteratorConsumeInterrupt, projected); err != nil {
				return result, err
			}
			continue
		}
		if event.Err != nil {
			if recoverable, ok := RecoverableToolErrorFromEvent(event); ok {
				result.HasToolErrors = true
				if _, count, tripped := toolFailures.recordFailure(recoverable.Info); tripped && count > 0 {
					result.CircuitBroken = true
				}
				projected := input.Projector.Consume(recoverable.Event)
				toolFailures.recordProjectedEvents(projected)
				if err := processProjected(IteratorConsumeRecoverableTool, projected); err != nil {
					return result, err
				}
				if result.CircuitBroken {
					break
				}
				continue
			}
			if !IsBusinessToolResultEvent(event) {
				if err := flushProjected(); err != nil {
					return result, err
				}
				result.FatalErr = fmt.Errorf("iterator event: %w", event.Err)
				return result, nil
			}
			result.HasToolErrors = true
		}
		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			for {
				msg, err := event.Output.MessageOutput.MessageStream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					if interruptEvent, ok := RecoverableInterruptEventFromErr(err, event.AgentName); ok {
						projected := input.Projector.Consume(interruptEvent)
						toolFailures.recordProjectedEvents(projected)
						if consumeErr := processProjected(IteratorConsumeInterrupt, projected); consumeErr != nil {
							return result, consumeErr
						}
						break
					}
					if recoverable, ok := RecoverableToolErrorFromErr(err, event.AgentName); ok {
						result.HasToolErrors = true
						if _, count, tripped := toolFailures.recordFailure(recoverable.Info); tripped && count > 0 {
							result.CircuitBroken = true
						}
						projected := input.Projector.Consume(recoverable.Event)
						toolFailures.recordProjectedEvents(projected)
						if consumeErr := processProjected(IteratorConsumeStreamTool, projected); consumeErr != nil {
							return result, consumeErr
						}
						if result.CircuitBroken {
							break
						}
						continue
					}
					if err := flushProjected(); err != nil {
						return result, err
					}
					result.FatalErr = err
					return result, nil
				}
				if msg == nil {
					continue
				}
				chunkEvent := adk.EventFromMessage(msg, nil, msg.Role, msg.ToolName)
				chunkEvent.AgentName = event.AgentName
				projected := input.Projector.Consume(chunkEvent)
				toolFailures.recordProjectedEvents(projected)
				if err := processProjected(IteratorConsumeStreamChunk, projected); err != nil {
					return result, err
				}
				if msg.Role == schema.Assistant {
					assistantSnapshot.WriteString(msg.Content)
				}
			}
			if result.CircuitBroken {
				break
			}
			continue
		}
		projected := input.Projector.Consume(event)
		toolFailures.recordProjectedEvents(projected)
		if err := processProjected(IteratorConsumeEvent, projected); err != nil {
			return result, err
		}
	}
	if err := flushProjected(); err != nil {
		return result, err
	}
	if persisted := input.Projector.GetPersistedState(); persisted != nil && !persisted.CanFinalizeDone() {
		result.Interrupted = true
	}
	return result, nil
}

func wrapIteratorConsumeError(kind IteratorConsumeKind, err error) error {
	switch kind {
	case IteratorConsumeInterrupt:
		return fmt.Errorf("persist projected interrupt event: %w", err)
	case IteratorConsumeRecoverableTool:
		return fmt.Errorf("persist recoverable tool error: %w", err)
	case IteratorConsumeStreamTool:
		return fmt.Errorf("persist projected tool error event: %w", err)
	case IteratorConsumeStreamChunk:
		return fmt.Errorf("persist projected stream chunk: %w", err)
	case IteratorConsumeEvent:
		return fmt.Errorf("persist projected event: %w", err)
	case IteratorConsumeFlush:
		return fmt.Errorf("flush projected events: %w", err)
	default:
		return err
	}
}
