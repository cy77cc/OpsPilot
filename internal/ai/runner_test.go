package ai

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

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
