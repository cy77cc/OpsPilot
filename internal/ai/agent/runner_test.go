package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
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

func newRunnerForQueryTest(t *testing.T) *PlatformRunner {
	t.Helper()

	runner, err := NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		t.Fatalf("new platform runner failed: %v", err)
	}
	if runner == nil {
		t.Fatalf("expected non-nil platform runner")
	}
	return runner
}

func TestPlatformRunnerQuery(t *testing.T) {
	runner := newRunnerForQueryTest(t)

	iter := runner.Query(context.Background(), "sess-1", "status")
	var content strings.Builder
	for {
		ev, ok := iter.Next()
		if !ok {
			break
		}
		if ev == nil || ev.Err != nil || ev.Output == nil || ev.Output.MessageOutput == nil {
			continue
		}
		if msg := ev.Output.MessageOutput.Message; msg != nil {
			content.WriteString(msg.Content)
		}
		if stream := ev.Output.MessageOutput.MessageStream; stream != nil {
			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("unexpected stream error: %v", err)
				}
				if chunk != nil {
					content.WriteString(chunk.Content)
				}
			}
			stream.Close()
		}
	}

	if got := content.String(); !strings.Contains(got, "ok: status") {
		t.Fatalf("unexpected query output: %q", got)
	}
}

func TestPlatformRunnerGenerate(t *testing.T) {
	runner := newRunnerForQueryTest(t)

	out, err := runner.Generate(context.Background(), []*schema.Message{schema.UserMessage("status")})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if out == nil || !strings.Contains(out.Content, "ok: status") {
		t.Fatalf("unexpected output: %#v", out)
	}
}
