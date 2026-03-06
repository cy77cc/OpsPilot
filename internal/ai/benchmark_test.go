package ai

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

func BenchmarkPlatformRunnerGenerate(b *testing.B) {
	agent, err := NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		b.Fatalf("new platform runner failed: %v", err)
	}
	msgs := []*schema.Message{schema.UserMessage("benchmark query")}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := agent.Generate(context.Background(), msgs); err != nil {
			b.Fatalf("generate failed: %v", err)
		}
	}
}

func BenchmarkPlatformRunnerRunTool(b *testing.B) {
	agent, err := NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		b.Fatalf("new platform runner failed: %v", err)
	}
	params := map[string]any{"resource": "pods", "namespace": "default", "limit": 1}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := agent.RunTool(context.Background(), "k8s_list_resources", params); err != nil {
			b.Fatalf("run tool failed: %v", err)
		}
	}
}
