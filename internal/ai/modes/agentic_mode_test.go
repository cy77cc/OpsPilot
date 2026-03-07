package modes

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/ai/types"
)

func TestNewAgenticModeNilModel(t *testing.T) {
	mode, err := NewAgenticMode(context.Background(), nil, aitools.PlatformDeps{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mode == nil {
		t.Fatalf("expected non-nil mode")
	}
}

func TestAgenticModeExecuteNilRunnerReturnsError(t *testing.T) {
	mode := &AgenticMode{}
	iter, gen := adk.NewAsyncIteratorPair[*types.AgentResult]()
	go func() {
		defer gen.Close()
		mode.Execute(context.Background(), "sess-1", "status", gen)
	}()

	results := collectAgentResults(iter)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	last := results[0]
	if last.Type != "error" {
		t.Fatalf("expected error result, got %s", last.Type)
	}
}

func TestAgenticModeProcessEventError(t *testing.T) {
	mode := &AgenticMode{}

	result := mode.processEvent(&adk.AgentEvent{Err: fmt.Errorf("boom")})
	if result == nil || result.Type != "error" {
		t.Fatalf("expected error result, got %#v", result)
	}
}

func TestAgenticModeProcessEventToolResult(t *testing.T) {
	mode := &AgenticMode{}

	result := mode.processEvent(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: &schema.Message{
					Role:     schema.Tool,
					ToolName: "host_list_inventory",
					Content:  "{\"ok\":true}",
				},
			},
		},
	})
	if result == nil || result.Type != "tool_result" {
		t.Fatalf("expected tool_result, got %#v", result)
	}
	if result.ToolName != "host_list_inventory" {
		t.Fatalf("unexpected tool name: %q", result.ToolName)
	}
}

func TestAgenticModeProcessEventTextResult(t *testing.T) {
	mode := &AgenticMode{}

	result := mode.processEvent(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: schema.AssistantMessage("done", nil),
			},
		},
	})
	if result == nil || result.Type != "text" {
		t.Fatalf("expected text result, got %#v", result)
	}
	if result.Content != "done" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

func TestAgenticModeProcessEventTextStreamResult(t *testing.T) {
	mode := &AgenticMode{}
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(schema.AssistantMessage("part1", nil), nil)
		sw.Send(schema.AssistantMessage(" part2", nil), nil)
	}()

	result := mode.processEvent(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				MessageStream: sr,
			},
		},
	})
	if result == nil || result.Type != "text" {
		t.Fatalf("expected text result, got %#v", result)
	}
	if result.Content != "part1part2" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

func TestAgenticModeProcessEventTextStreamError(t *testing.T) {
	mode := &AgenticMode{}
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(nil, fmt.Errorf("stream boom"))
	}()

	result := mode.processEvent(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				MessageStream: sr,
			},
		},
	})
	if result == nil || result.Type != "error" {
		t.Fatalf("expected error result, got %#v", result)
	}
	if result.Content != "stream boom" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
}

func TestProcessInterruptApprovalInfo(t *testing.T) {
	result := processInterrupt(&adk.InterruptInfo{
		Data: &aitools.ApprovalInfo{
			ToolName:        "host_batch_exec_apply",
			ArgumentsInJSON: "{\"host_ids\":[2]}",
			Risk:            aitools.ToolRiskHigh,
			Preview:         map[string]any{"host_ids": []any{2}},
		},
		InterruptContexts: []*adk.InterruptCtx{
			{ID: "call-1", IsRootCause: true},
		},
	})
	if result == nil || result.Type != "ask_user" || result.Ask == nil {
		t.Fatalf("expected ask_user result, got %#v", result)
	}
	if result.Ask.ID != "call-1" {
		t.Fatalf("unexpected ask id: %q", result.Ask.ID)
	}
	if result.Ask.Risk != string(aitools.ToolRiskHigh) {
		t.Fatalf("unexpected risk: %q", result.Ask.Risk)
	}
}

func TestProcessInterruptReviewEditInfo(t *testing.T) {
	result := processInterrupt(&adk.InterruptInfo{
		Data: &aitools.ReviewEditInfo{
			ToolName:        "service_deploy_apply",
			ArgumentsInJSON: "{\"service_id\":1}",
		},
		InterruptContexts: []*adk.InterruptCtx{
			{ID: "call-2", IsRootCause: true},
		},
	})
	if result == nil || result.Type != "ask_user" || result.Ask == nil {
		t.Fatalf("expected ask_user result, got %#v", result)
	}
	if result.Ask.Risk != "medium" {
		t.Fatalf("unexpected risk: %q", result.Ask.Risk)
	}
}

func TestProcessInterruptDefaultPayload(t *testing.T) {
	result := processInterrupt(&adk.InterruptInfo{
		Data: "manual input required",
		InterruptContexts: []*adk.InterruptCtx{
			{ID: "call-3", IsRootCause: true},
		},
	})
	if result == nil || result.Type != "ask_user" || result.Ask == nil {
		t.Fatalf("expected ask_user result, got %#v", result)
	}
	if result.Ask.ID != "call-3" {
		t.Fatalf("unexpected ask id: %q", result.Ask.ID)
	}
}

func TestInterruptTargetsOnlyIncludeRootCause(t *testing.T) {
	targets := interruptTargets([]*adk.InterruptCtx{
		{ID: "nested", IsRootCause: false},
		{ID: "root", IsRootCause: true},
		{ID: "", IsRootCause: true},
	})
	if len(targets) != 1 || targets[0] != "root" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}
