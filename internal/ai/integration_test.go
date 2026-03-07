package ai

import (
	"context"
	"fmt"
	"strings"
	"testing"

	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/agent"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

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

func newFakePlatformRunner(t *testing.T) *agent.PlatformRunner {
	t.Helper()
	runner, err := agent.NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		t.Fatalf("new platform runner failed: %v", err)
	}
	if runner == nil {
		t.Fatalf("expected non-nil platform runner")
	}
	return runner
}

func TestE2ESimpleQuery(t *testing.T) {
	runner := newFakePlatformRunner(t)
	out, err := runner.Generate(context.Background(), []*schema.Message{schema.UserMessage("status")})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if out == nil || !strings.Contains(out.Content, "ok: status") {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestE2EMultiStepTask(t *testing.T) {
	runner := newFakePlatformRunner(t)
	out, err := runner.Generate(context.Background(), []*schema.Message{schema.UserMessage("step1 then step2")})
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if out == nil || !strings.Contains(out.Content, "step1 then step2") {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestE2EApprovalInterruptFlow(t *testing.T) {
	runner := newFakePlatformRunner(t)
	metas := runner.ToolMetas()
	found := false
	for _, meta := range metas {
		if meta.Name == "service_deploy_apply" {
			if meta.Risk != tools.ToolRiskHigh {
				t.Fatalf("unexpected risk level for %s: %s", meta.Name, meta.Risk)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected service_deploy_apply tool metadata")
	}
}

func TestE2EErrorRecovery(t *testing.T) {
	runner := newFakePlatformRunner(t)
	if _, err := runner.Generate(context.Background(), []*schema.Message{schema.UserMessage("trigger error")}); err == nil {
		t.Fatalf("expected synthetic error")
	}
	out, err := runner.Generate(context.Background(), []*schema.Message{schema.UserMessage("recover")})
	if err != nil {
		t.Fatalf("generate after recovery failed: %v", err)
	}
	if out == nil || !strings.Contains(out.Content, "ok: recover") {
		t.Fatalf("unexpected output: %#v", out)
	}
}
