package ai

import (
	"context"
	"testing"
	"time"

	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
)

type fakeClassifierModel struct {
	reply string
}

func (m *fakeClassifierModel) Generate(_ context.Context, input []*schema.Message, _ ...modelcomponent.Option) (*schema.Message, error) {
	return schema.AssistantMessage(m.reply, nil), nil
}

func (m *fakeClassifierModel) Stream(_ context.Context, input []*schema.Message, _ ...modelcomponent.Option) (*schema.StreamReader[*schema.Message], error) {
	msg, _ := m.Generate(context.Background(), input)
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()
	return sr, nil
}

func (m *fakeClassifierModel) WithTools(_ []*schema.ToolInfo) (modelcomponent.ToolCallingChatModel, error) {
	return m, nil
}

func TestAgentFirstResponseCompletesWithinTwoSeconds(t *testing.T) {
	agent, err := NewHybridAgent(
		context.Background(),
		&fakeToolCallingModel{},
		&fakeClassifierModel{reply: "agentic"},
		aitools.PlatformDeps{},
		nil,
	)
	if err != nil {
		t.Fatalf("new hybrid agent failed: %v", err)
	}

	start := time.Now()
	iter := agent.Query(context.Background(), "sess-agent-perf", "查看 pod 日志")

	first, ok := iter.Next()
	if !ok {
		t.Fatalf("expected first agent result")
	}
	if first == nil {
		t.Fatalf("expected non-nil first agent result")
	}

	elapsed := time.Since(start)
	if elapsed >= 2*time.Second {
		t.Fatalf("expected agent first response under 2s, got %s", elapsed)
	}
}
