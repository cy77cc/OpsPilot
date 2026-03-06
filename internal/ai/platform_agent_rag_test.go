package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/rag"
)

type fakeRAGRetriever struct{}

func (f fakeRAGRetriever) Retrieve(_ context.Context, query string, topK int) (*rag.RAGContext, error) {
	if query == "" || topK <= 0 {
		return &rag.RAGContext{}, nil
	}
	return &rag.RAGContext{ToolExamples: []rag.ToolExample{{ToolName: "deployment.release", Intent: "deploy", ParamsJSON: `{}`}}}, nil
}

func (f fakeRAGRetriever) BuildAugmentedPrompt(query string, context *rag.RAGContext) string {
	if context == nil || len(context.ToolExamples) == 0 {
		return query
	}
	return "[RAG]\n" + query
}

func TestInjectRAGIntoMessages(t *testing.T) {
	agent := &PlatformRunner{ragRetriever: fakeRAGRetriever{}}
	messages := []*schema.Message{
		schema.SystemMessage("sys"),
		schema.UserMessage("deploy service to prod"),
	}
	out := agent.injectRAGIntoMessages(context.Background(), messages)
	if len(out) != len(messages) {
		t.Fatalf("unexpected message count: %d", len(out))
	}
	if !strings.Contains(out[1].Content, "[RAG]") {
		t.Fatalf("expected augmented user message, got: %s", out[1].Content)
	}
	if messages[1].Content == out[1].Content {
		t.Fatalf("expected copy-on-write behavior")
	}
}

func TestInjectRAGIntoMessagesWithoutRetriever(t *testing.T) {
	agent := &PlatformRunner{}
	messages := []*schema.Message{schema.UserMessage("check status")}
	out := agent.injectRAGIntoMessages(context.Background(), messages)
	if out[0].Content != messages[0].Content {
		t.Fatalf("expected unchanged messages without rag retriever")
	}
}
