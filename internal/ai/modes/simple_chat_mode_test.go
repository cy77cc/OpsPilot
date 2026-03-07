package modes

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/types"
)

// fakeToolCallingModel is a test double for ToolCallingChatModel
type fakeToolCallingModel struct{}

func (m *fakeToolCallingModel) Generate(_ context.Context, input []*schema.Message, _ ...modelcomponent.Option) (*schema.Message, error) {
	last := ""
	for i := len(input) - 1; i >= 0; i-- {
		if input[i] != nil && input[i].Role == schema.User {
			last = strings.TrimSpace(input[i].Content)
			break
		}
	}
	if strings.Contains(strings.ToLower(last), "error") {
		return nil, fmt.Errorf("synthetic model error")
	}
	return schema.AssistantMessage("ok: "+last, nil), nil
}

func (m *fakeToolCallingModel) Stream(_ context.Context, input []*schema.Message, _ ...modelcomponent.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, err := m.Generate(context.Background(), input)
	if err != nil {
		return nil, err
	}
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()
	return sr, nil
}

func (m *fakeToolCallingModel) WithTools(_ []*schema.ToolInfo) (modelcomponent.ToolCallingChatModel, error) {
	return m, nil
}

// fakeEmptyModel returns empty content
type fakeEmptyModel struct{}

func (m *fakeEmptyModel) Generate(_ context.Context, _ []*schema.Message, _ ...modelcomponent.Option) (*schema.Message, error) {
	return schema.AssistantMessage("", nil), nil
}

func (m *fakeEmptyModel) Stream(_ context.Context, input []*schema.Message, _ ...modelcomponent.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, _ := m.Generate(context.Background(), input)
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()
	return sr, nil
}

func (m *fakeEmptyModel) WithTools(_ []*schema.ToolInfo) (modelcomponent.ToolCallingChatModel, error) {
	return m, nil
}

func collectAgentResults(iter *adk.AsyncIterator[*types.AgentResult]) []*types.AgentResult {
	out := make([]*types.AgentResult, 0)
	for {
		item, ok := iter.Next()
		if !ok {
			break
		}
		if item != nil {
			out = append(out, item)
		}
	}
	return out
}

func TestSimpleChatModeExecuteReturnsTextResult(t *testing.T) {
	mode := NewSimpleChatMode(&fakeToolCallingModel{})
	iter, gen := adk.NewAsyncIteratorPair[*types.AgentResult]()

	go func() {
		defer gen.Close()
		mode.Execute(context.Background(), "什么是 Pod", gen)
	}()

	results := collectAgentResults(iter)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "text" {
		t.Fatalf("expected text result, got %s", results[0].Type)
	}
	if results[0].Content == "" {
		t.Fatalf("expected non-empty content")
	}
}

func TestSimpleChatModeExecuteNilModelReturnsError(t *testing.T) {
	mode := NewSimpleChatMode(nil)
	iter, gen := adk.NewAsyncIteratorPair[*types.AgentResult]()

	go func() {
		defer gen.Close()
		mode.Execute(context.Background(), "你好", gen)
	}()

	results := collectAgentResults(iter)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "error" {
		t.Fatalf("expected error result, got %s", results[0].Type)
	}
}

func TestSimpleChatModeExecuteModelErrorReturnsError(t *testing.T) {
	mode := NewSimpleChatMode(&fakeToolCallingModel{})
	iter, gen := adk.NewAsyncIteratorPair[*types.AgentResult]()

	go func() {
		defer gen.Close()
		mode.Execute(context.Background(), "trigger error", gen)
	}()

	results := collectAgentResults(iter)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "error" {
		t.Fatalf("expected error result, got %s", results[0].Type)
	}
}

func TestSimpleChatModeExecuteEmptyContentFallsBackToDefault(t *testing.T) {
	mode := NewSimpleChatMode(&fakeEmptyModel{})
	iter, gen := adk.NewAsyncIteratorPair[*types.AgentResult]()

	go func() {
		defer gen.Close()
		mode.Execute(context.Background(), "你好", gen)
	}()

	results := collectAgentResults(iter)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Type != "text" {
		t.Fatalf("expected text result, got %s", results[0].Type)
	}
	if results[0].Content != "无输出。" {
		t.Fatalf("unexpected fallback content: %q", results[0].Content)
	}
}
