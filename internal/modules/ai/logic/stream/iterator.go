// Package stream 实现 AI Agent 迭代器处理和事件流投影。
package stream

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/OpsPilot/internal/core/logger"
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
	Iterator                *adk.AsyncIterator[*adk.AgentEvent]
	Projector               *airuntime.StreamProjector
	Emit                    func(event string, data any)
	ConsumeProjected        func(kind IteratorConsumeKind, events []airuntime.PublicStreamEvent) error
	HandleRunUpdate         func(update RunUpdate)
	CircuitBreakerThreshold int // 0 uses default of 2
}

// ProcessAgentIterator 处理 Agent 异步迭代器事件流。
func ProcessAgentIterator(ctx context.Context, input IteratorProcessInput) (IteratorProcessResult, error) {
	result := IteratorProcessResult{}
	logger.L().Info("[AI-DEBUG] ProcessAgentIterator ENTERED",
		logger.String("iterator_nil", fmt.Sprintf("%v", input.Iterator == nil)),
		logger.String("ctx_nil", fmt.Sprintf("%v", ctx == nil)))
	if input.Iterator == nil {
		logger.L().Info("[AI-DEBUG] ProcessAgentIterator: iterator is nil, returning early")
		return result, nil
	}
	if ctx == nil {
		return result, fmt.Errorf("context is required")
	}
	if deadline, ok := ctx.Deadline(); ok {
		logger.L().Info("[AI-DEBUG] ProcessAgentIterator: context has deadline",
			logger.String("deadline", deadline.Format(time.RFC3339)))
		// Independent timer to diagnose if context deadline fires
		time.AfterFunc(time.Until(deadline), func() {
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: TIMER FIRED at deadline",
				logger.String("ctx_err", fmt.Sprintf("%v", ctx.Err())),
				logger.String("deadline", deadline.Format(time.RFC3339)))
		})
	} else {
		logger.L().Info("[AI-DEBUG] ProcessAgentIterator: context has NO deadline")
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
		toolFailures      = newToolFailureTracker(input.CircuitBreakerThreshold)
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

	// Quick timer to verify timers work in this goroutine context
	time.AfterFunc(10*time.Second, func() {
		logger.L().Info("[AI-DEBUG] ProcessAgentIterator: 10s TIMER FIRED",
			logger.String("ctx_err", fmt.Sprintf("%v", ctx.Err())))
	})

	iterationCount := 0
	for {
		if ctx.Err() != nil {
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: ctx cancelled",
				logger.Error(ctx.Err()),
				logger.Int("iteration", iterationCount))
			result.FatalErr = ctx.Err()
			break
		}
		if persisted := input.Projector.GetPersistedState(); persisted != nil && !persisted.CanFinalizeDone() {
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: CanFinalizeDone=false, interrupted",
				logger.Int("iteration", iterationCount))
			result.Interrupted = true
			break
		}
		// Use goroutine + select to allow context cancellation while waiting for Next().
		// AsyncIterator.Next() blocks on an unbounded channel that doesn't respect context.
		type nextResult struct {
			event *adk.AgentEvent
			ok    bool
		}
		nextCh := make(chan nextResult, 1)
		go func() {
			e, ok := input.Iterator.Next()
			nextCh <- nextResult{event: e, ok: ok}
		}()
		var event *adk.AgentEvent
		var ok bool
		select {
		case nr := <-nextCh:
			fmt.Fprintf(os.Stderr, "[AI-DEBUG] ProcessAgentIterator: nextCh returned, ok=%v, iteration=%d\n", nr.ok, iterationCount)
			event = nr.event
			ok = nr.ok
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "[AI-DEBUG] ProcessAgentIterator: ctx.Done FIRED, err=%v, iteration=%d\n", ctx.Err(), iterationCount)
			stdlog.Printf("[AI-DEBUG] ProcessAgentIterator: ctx.Done FIRED, err=%v, iteration=%d", ctx.Err(), iterationCount)
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: ctx cancelled while waiting for Next()",
				logger.Error(ctx.Err()),
				logger.Int("iteration", iterationCount))
			result.FatalErr = ctx.Err()
		}
		if result.FatalErr != nil {
			break
		}
		iterationCount++
		if !ok {
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: iterator exhausted",
				logger.Int("iteration", iterationCount))
			break
		}
		hasOutput := event.Output != nil
		hasAction := event.Action != nil
		var eventErr string
		if event.Err != nil {
			eventErr = event.Err.Error()
		}
		// Deep inspection of event output
		if hasOutput && event.Output.MessageOutput != nil {
			mo := event.Output.MessageOutput
			contentLen := len(mo.Message.Content)
			toolCallCount := len(mo.Message.ToolCalls)
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: event received",
				logger.Int("iteration", iterationCount),
				logger.String("agentName", event.AgentName),
				logger.String("err", eventErr),
				logger.String("hasOutput", "true"),
				logger.String("hasAction", fmt.Sprintf("%v", hasAction)),
				logger.String("role", string(mo.Role)),
				logger.Int("contentLen", contentLen),
				logger.Int("toolCallCount", toolCallCount),
				logger.String("isStreaming", fmt.Sprintf("%v", mo.IsStreaming)),
				logger.String("streamNil", fmt.Sprintf("%v", mo.MessageStream == nil)))
		} else {
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: event received",
				logger.Int("iteration", iterationCount),
				logger.String("agentName", event.AgentName),
				logger.String("err", eventErr),
				logger.String("hasOutput", fmt.Sprintf("%v", hasOutput)),
				logger.String("hasAction", fmt.Sprintf("%v", hasAction)))
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
				return result, result.FatalErr
			}
			result.HasToolErrors = true
		}
		if event.Output != nil && event.Output.MessageOutput != nil && event.Output.MessageOutput.IsStreaming && event.Output.MessageOutput.MessageStream != nil {
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: taking STREAMING path",
				logger.Int("iteration", iterationCount),
				logger.String("agentName", event.AgentName))
			chunkCount := 0
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
					return result, result.FatalErr
				}
				if msg == nil {
					continue
				}
				chunkCount++
				logger.L().Info("[AI-DEBUG] ProcessAgentIterator: stream chunk",
					logger.Int("iteration", iterationCount),
					logger.Int("chunk", chunkCount),
					logger.String("role", string(msg.Role)),
					logger.Int("contentLen", len(msg.Content)),
					logger.Int("toolCalls", len(msg.ToolCalls)))
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
			logger.L().Info("[AI-DEBUG] ProcessAgentIterator: streaming path complete",
				logger.Int("iteration", iterationCount),
				logger.Int("totalChunks", chunkCount))
			if result.CircuitBroken {
				break
			}
			continue
		}
		logger.L().Info("[AI-DEBUG] ProcessAgentIterator: taking DEFAULT path (non-streaming)",
			logger.Int("iteration", iterationCount),
			logger.String("agentName", event.AgentName))
		projected := input.Projector.Consume(event)
		logger.L().Info("[AI-DEBUG] ProcessAgentIterator: projector.Consume returned",
			logger.Int("iteration", iterationCount),
			logger.Int("projectedCount", len(projected)))
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
