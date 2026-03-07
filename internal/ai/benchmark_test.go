package ai

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/cy77cc/k8s-manage/internal/ai/agent"
	"github.com/cy77cc/k8s-manage/internal/ai/tools"
)

func BenchmarkPlatformRunnerGenerate(b *testing.B) {
	runner, err := agent.NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		b.Fatalf("new platform runner failed: %v", err)
	}
	msgs := []*schema.Message{schema.UserMessage("benchmark query")}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := runner.Generate(context.Background(), msgs); err != nil {
			b.Fatalf("generate failed: %v", err)
		}
	}
}

func BenchmarkPlatformRunnerRunTool(b *testing.B) {
	runner, err := agent.NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		b.Fatalf("new platform runner failed: %v", err)
	}
	params := map[string]any{"resource": "pods", "namespace": "default", "limit": 1}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := runner.RunTool(context.Background(), "k8s_query", params); err != nil {
			b.Fatalf("run tool failed: %v", err)
		}
	}
}

func BenchmarkPlatformRunnerGenerateParallel(b *testing.B) {
	runner, err := agent.NewPlatformRunner(context.Background(), &fakeToolCallingModel{}, tools.PlatformDeps{}, nil)
	if err != nil {
		b.Fatalf("new platform runner failed: %v", err)
	}

	var seq uint64
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := atomic.AddUint64(&seq, 1)
			msgs := []*schema.Message{schema.UserMessage("benchmark parallel query")}
			if _, err := runner.Generate(context.Background(), msgs); err != nil {
				b.Fatalf("generate failed on iteration %d: %v", n, err)
			}
		}
	})
}
