package logic

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

type iteratorConsumeKind string

const (
	iteratorConsumeInterrupt       iteratorConsumeKind = "interrupt"
	iteratorConsumeRecoverableTool iteratorConsumeKind = "recoverable_tool_error"
	iteratorConsumeStreamTool      iteratorConsumeKind = "stream_tool_error"
	iteratorConsumeStreamChunk     iteratorConsumeKind = "stream_chunk"
	iteratorConsumeEvent           iteratorConsumeKind = "event"
	iteratorConsumeFlush           iteratorConsumeKind = "flush"
)

type IteratorProcessResult struct {
	Interrupted       bool
	HasToolErrors     bool
	CircuitBroken     bool
	SummaryText       string
	AssistantSnapshot string
	FatalErr          error
}

type iteratorProcessInput struct {
	Iterator         *adk.AsyncIterator[*adk.AgentEvent]
	Projector        *airuntime.StreamProjector
	Emit             EventEmitter
	ConsumeProjected func(kind iteratorConsumeKind, events []airuntime.PublicStreamEvent) error
	HandleRunUpdate  func(update projectedRunUpdate)
}

func processAgentIterator(_ context.Context, input iteratorProcessInput) (IteratorProcessResult, error) {
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
		input.HandleRunUpdate = func(projectedRunUpdate) {}
	}

	var (
		summaryContent    strings.Builder
		assistantSnapshot strings.Builder
		toolFailures      = newToolFailureTracker()
	)

	processProjected := func(kind iteratorConsumeKind, events []airuntime.PublicStreamEvent) error {
		if len(events) == 0 {
			return nil
		}

		update := accumulateProjectedEvents(events, &summaryContent)
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
		if err := processProjected(iteratorConsumeFlush, events); err != nil {
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

		if interruptEvent, ok := recoverableInterruptEventFromEvent(event); ok {
			projected := input.Projector.Consume(interruptEvent)
			toolFailures.recordProjectedEvents(projected)
			if err := processProjected(iteratorConsumeInterrupt, projected); err != nil {
				return result, err
			}
			continue
		}

		if event.Err != nil {
			if recoverable, ok := recoverableToolErrorFromEvent(event); ok {
				result.HasToolErrors = true
				if _, count, tripped := toolFailures.recordFailure(recoverable.Info); tripped && count > 0 {
					result.CircuitBroken = true
				}
				projected := input.Projector.Consume(recoverable.Event)
				toolFailures.recordProjectedEvents(projected)
				if err := processProjected(iteratorConsumeRecoverableTool, projected); err != nil {
					return result, err
				}
				if result.CircuitBroken {
					break
				}
				continue
			}

			if !isBusinessToolResultEvent(event) {
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
					if interruptEvent, ok := recoverableInterruptEventFromErr(err, event.AgentName); ok {
						projected := input.Projector.Consume(interruptEvent)
						toolFailures.recordProjectedEvents(projected)
						if consumeErr := processProjected(iteratorConsumeInterrupt, projected); consumeErr != nil {
							return result, consumeErr
						}
						break
					}

					if recoverable, ok := recoverableToolErrorFromErr(err, event.AgentName); ok {
						result.HasToolErrors = true
						if _, count, tripped := toolFailures.recordFailure(recoverable.Info); tripped && count > 0 {
							result.CircuitBroken = true
						}
						projected := input.Projector.Consume(recoverable.Event)
						toolFailures.recordProjectedEvents(projected)
						if consumeErr := processProjected(iteratorConsumeStreamTool, projected); consumeErr != nil {
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
				if err := processProjected(iteratorConsumeStreamChunk, projected); err != nil {
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
		if err := processProjected(iteratorConsumeEvent, projected); err != nil {
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

func wrapIteratorConsumeError(kind iteratorConsumeKind, err error) error {
	switch kind {
	case iteratorConsumeInterrupt:
		return fmt.Errorf("persist projected interrupt event: %w", err)
	case iteratorConsumeRecoverableTool:
		return fmt.Errorf("persist recoverable tool error: %w", err)
	case iteratorConsumeStreamTool:
		return fmt.Errorf("persist projected tool error event: %w", err)
	case iteratorConsumeStreamChunk:
		return fmt.Errorf("persist projected stream chunk: %w", err)
	case iteratorConsumeEvent:
		return fmt.Errorf("persist projected event: %w", err)
	case iteratorConsumeFlush:
		return fmt.Errorf("flush projected events: %w", err)
	default:
		return err
	}
}
