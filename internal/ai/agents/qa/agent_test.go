package qa

import (
	"context"
	"testing"
)

func TestAgent_AnswerReturnsVisibleText(t *testing.T) {
	agent := NewAgent()

	result, err := agent.Answer(context.Background(), Request{
		Message: "What does a namespace do?",
	})
	if err != nil {
		t.Fatalf("answer question: %v", err)
	}
	if result.Text == "" {
		t.Fatalf("expected visible answer text, got %#v", result)
	}
}
