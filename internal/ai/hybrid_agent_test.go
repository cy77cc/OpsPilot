package ai

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
	aitools "github.com/cy77cc/k8s-manage/internal/ai/tools"
	"github.com/cy77cc/k8s-manage/internal/ai/types"
)

func collectAgentResults(iter *adk.AsyncIterator[*types.AgentResult]) []*types.AgentResult {
	var results []*types.AgentResult
	for {
		item, ok := iter.Next()
		if !ok {
			break
		}
		results = append(results, item)
	}
	return results
}

func TestNewHybridAgent(t *testing.T) {
	agent, err := NewHybridAgent(context.Background(), &fakeToolCallingModel{}, &fakeClassifierModel{reply: "simple"}, aitools.PlatformDeps{}, nil)
	if err != nil {
		t.Fatalf("new hybrid agent failed: %v", err)
	}
	if agent == nil {
		t.Fatalf("expected non-nil hybrid agent")
	}
}

func TestHybridAgentQueryRoutesToSimpleChat(t *testing.T) {
	agent, err := NewHybridAgent(context.Background(), &fakeToolCallingModel{}, &fakeClassifierModel{reply: "simple"}, aitools.PlatformDeps{}, nil)
	if err != nil {
		t.Fatalf("new hybrid agent failed: %v", err)
	}

	results := collectAgentResults(agent.Query(context.Background(), "sess-1", "什么是 Pod"))
	if len(results) == 0 {
		t.Fatalf("expected results")
	}
	last := results[len(results)-1]
	if last.Type != "text" {
		t.Fatalf("expected text result, got %s", last.Type)
	}
}

func TestNewHybridAgentFallsBackToChatModelForClassifier(t *testing.T) {
	agent, err := NewHybridAgent(context.Background(), &fakeToolCallingModel{}, nil, aitools.PlatformDeps{}, nil)
	if err != nil {
		t.Fatalf("new hybrid agent failed: %v", err)
	}
	if agent == nil {
		t.Fatalf("expected non-nil agent")
	}
}

func TestHybridAgentResumeWithoutAgenticModeReturnsError(t *testing.T) {
	agent := &HybridAgent{}
	if _, err := agent.Resume(context.Background(), "sess-1", "ask-1", true); err == nil {
		t.Fatalf("expected error")
	}
}
