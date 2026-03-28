package logic

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	toolcomp "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	airuntime "github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

func TestProcessAgentIterator_InterruptEventStopsWithInterrupted(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		gen.Send(&adk.AgentEvent{
			AgentName: "executor",
			Err: toolcomp.StatefulInterrupt(
				adk.AppendAddressSegment(context.Background(), adk.AddressSegmentTool, "host_exec"),
				map[string]any{
					"approval_id":     "approval-iterator-1",
					"call_id":         "call-iterator-approval-1",
					"tool_name":       "host_exec",
					"preview":         map[string]any{"command": "systemctl restart nginx"},
					"timeout_seconds": 300,
				},
				"state",
			),
		})
	}()

	var emitted []airuntime.PublicStreamEvent
	res, err := processAgentIterator(context.Background(), iteratorProcessInput{
		Iterator:  iter,
		Projector: airuntime.NewStreamProjector(),
		Emit: func(event string, data any) {
			emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
		},
	})
	if err != nil {
		t.Fatalf("process iterator: %v", err)
	}
	if !res.Interrupted {
		t.Fatalf("expected interrupted result, got %#v", res)
	}
	if res.FatalErr != nil {
		t.Fatalf("expected no fatal error, got %v", res.FatalErr)
	}

	var sawApproval bool
	for _, event := range emitted {
		if event.Event == "tool_approval" {
			sawApproval = true
			break
		}
	}
	if !sawApproval {
		t.Fatalf("expected emitted tool_approval event, got %#v", emitted)
	}
}

func TestProcessAgentIterator_RecoverableToolErrorMarksToolErrorAndCircuitBreak(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		gen.Send(adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
			{ID: "call-1", Function: schema.FunctionCall{Name: "host_exec", Arguments: `{"host_id":1,"command":"uptime"}`}},
		}), nil, schema.Assistant, ""))
		gen.Send(&adk.AgentEvent{
			AgentName: "executor",
			Err:       errors.New("[NodeRunError] failed to stream tool call call-1: [LocalFunc] failed to invoke tool, toolName=host_exec, err=command denied"),
		})
		gen.Send(adk.EventFromMessage(schema.AssistantMessage("", []schema.ToolCall{
			{ID: "call-2", Function: schema.FunctionCall{Name: "host_exec", Arguments: `{"host_id":1,"command":"uptime"}`}},
		}), nil, schema.Assistant, ""))
		gen.Send(&adk.AgentEvent{
			AgentName: "executor",
			Err:       errors.New("[NodeRunError] failed to stream tool call call-2: [LocalFunc] failed to invoke tool, toolName=host_exec, err=command denied"),
		})
		gen.Send(adk.EventFromMessage(schema.AssistantMessage("should not be consumed after circuit break", nil), nil, schema.Assistant, ""))
	}()

	var emitted []airuntime.PublicStreamEvent
	res, err := processAgentIterator(context.Background(), iteratorProcessInput{
		Iterator:  iter,
		Projector: airuntime.NewStreamProjector(),
		Emit: func(event string, data any) {
			emitted = append(emitted, airuntime.PublicStreamEvent{Event: event, Data: data})
		},
	})
	if err != nil {
		t.Fatalf("process iterator: %v", err)
	}
	if !res.HasToolErrors {
		t.Fatalf("expected tool errors to be recorded, got %#v", res)
	}
	if !res.CircuitBroken {
		t.Fatalf("expected circuit breaker to trip, got %#v", res)
	}
	for _, event := range emitted {
		data, _ := event.Data.(map[string]any)
		if strings.Contains(stringValue(data, "content"), "should not be consumed") {
			t.Fatalf("expected circuit break to stop later events, got %#v", emitted)
		}
	}
}

func TestProcessAgentIterator_FatalIteratorErrorFlushesSummaryIntoResult(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		gen.Send(adk.EventFromMessage(schema.AssistantMessage("partial summary", nil), nil, schema.Assistant, ""))
		gen.Send(&adk.AgentEvent{
			AgentName: "planner",
			Err:       errors.New("iterator exploded"),
		})
	}()

	res, err := processAgentIterator(context.Background(), iteratorProcessInput{
		Iterator:  iter,
		Projector: airuntime.NewStreamProjector(),
		Emit:      func(string, any) {},
	})
	if err != nil {
		t.Fatalf("process iterator: %v", err)
	}
	if res.FatalErr == nil {
		t.Fatalf("expected fatal error in result, got %#v", res)
	}
	if !strings.Contains(res.FatalErr.Error(), "iterator event: iterator exploded") {
		t.Fatalf("unexpected fatal error: %v", res.FatalErr)
	}
	if !strings.Contains(res.SummaryText, "partial summary") {
		t.Fatalf("expected summary flush before fatal return, got %#v", res)
	}
}

func TestProcessAgentIterator_FatalStreamRecvErrorReturnsAssistantSnapshot(t *testing.T) {
	streamReader, streamWriter := schema.Pipe[*schema.Message](2)
	go func() {
		defer streamWriter.Close()
		streamWriter.Send(schema.AssistantMessage("partial answer", nil), nil)
		streamWriter.Send(nil, errors.New("read tcp 10.0.0.8:443->10.0.0.9:1234: i/o timeout"))
	}()

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		gen.Send(&adk.AgentEvent{
			AgentName: "executor",
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming:   true,
					MessageStream: streamReader,
				},
			},
		})
	}()

	res, err := processAgentIterator(context.Background(), iteratorProcessInput{
		Iterator:  iter,
		Projector: airuntime.NewStreamProjector(),
		Emit:      func(string, any) {},
	})
	if err != nil {
		t.Fatalf("process iterator: %v", err)
	}
	if res.FatalErr == nil {
		t.Fatalf("expected fatal recv error, got %#v", res)
	}
	if !strings.Contains(res.FatalErr.Error(), "i/o timeout") {
		t.Fatalf("unexpected fatal error: %v", res.FatalErr)
	}
	if res.AssistantSnapshot != "partial answer" {
		t.Fatalf("expected assistant snapshot to capture streamed content, got %#v", res)
	}
}
