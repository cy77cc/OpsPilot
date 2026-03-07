package classifier

import (
	"context"
	"fmt"
	"testing"

	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type fakeClassifierModel struct {
	reply string
	err   error
}

func (m *fakeClassifierModel) Generate(_ context.Context, _ []*schema.Message, _ ...modelcomponent.Option) (*schema.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return schema.AssistantMessage(m.reply, nil), nil
}

func (m *fakeClassifierModel) Stream(_ context.Context, _ []*schema.Message, _ ...modelcomponent.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *fakeClassifierModel) WithTools(_ []*schema.ToolInfo) (modelcomponent.ToolCallingChatModel, error) {
	return m, nil
}

func TestIntentClassifierClassifyNilModelFallsBackToSimple(t *testing.T) {
	classifier := NewIntentClassifier(nil)

	intent, err := classifier.Classify(context.Background(), "查看 pod 日志")
	if err != nil {
		t.Fatalf("classify returned error: %v", err)
	}
	if intent != IntentSimple {
		t.Fatalf("expected simple fallback, got %s", intent)
	}
}

func TestIntentClassifierClassifyModelErrorFallsBackToSimple(t *testing.T) {
	classifier := NewIntentClassifier(&fakeClassifierModel{err: fmt.Errorf("synthetic failure")})

	intent, err := classifier.Classify(context.Background(), "执行主机命令")
	if err != nil {
		t.Fatalf("classify returned error: %v", err)
	}
	if intent != IntentSimple {
		t.Fatalf("expected simple fallback, got %s", intent)
	}
}

func TestIntentClassifierClassifyAgentic(t *testing.T) {
	classifier := NewIntentClassifier(&fakeClassifierModel{reply: "agentic"})

	intent, err := classifier.Classify(context.Background(), "查看生产环境 pod 日志")
	if err != nil {
		t.Fatalf("classify returned error: %v", err)
	}
	if intent != IntentAgentic {
		t.Fatalf("expected agentic, got %s", intent)
	}
}

func TestIntentClassifierClassifySimple(t *testing.T) {
	classifier := NewIntentClassifier(&fakeClassifierModel{reply: "simple"})

	intent, err := classifier.Classify(context.Background(), "什么是 Pod")
	if err != nil {
		t.Fatalf("classify returned error: %v", err)
	}
	if intent != IntentSimple {
		t.Fatalf("expected simple, got %s", intent)
	}
}

func TestParseIntentFallsBackToSimpleForUnexpectedOutput(t *testing.T) {
	intent := parseIntent(schema.AssistantMessage("I think this is unclear", nil))
	if intent != IntentSimple {
		t.Fatalf("expected simple fallback, got %s", intent)
	}
}

func TestParseIntentAcceptsMixedCaseAgentic(t *testing.T) {
	intent := parseIntent(schema.AssistantMessage("Agentic", nil))
	if intent != IntentAgentic {
		t.Fatalf("expected agentic, got %s", intent)
	}
}

func TestParseIntentNilMessageFallsBackToSimple(t *testing.T) {
	intent := parseIntent(nil)
	if intent != IntentSimple {
		t.Fatalf("expected simple fallback, got %s", intent)
	}
}
