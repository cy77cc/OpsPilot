package experts

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestExpertExecutorStreamStepFallback(t *testing.T) {
	reg := &fakeRegistry{
		experts: map[string]*Expert{
			"service_expert": {Name: "service_expert"},
		},
	}
	exec := NewExpertExecutor(reg)
	stream, err := exec.StreamStep(context.Background(), &ExecutionStep{
		ExpertName: "service_expert",
		Task:       "analyze service",
	}, nil, "服务不可用")
	if err != nil {
		t.Fatalf("stream step: %v", err)
	}
	defer stream.Close()

	msg, recvErr := stream.Recv()
	if recvErr != nil {
		t.Fatalf("recv stream: %v", recvErr)
	}
	if msg == nil || !strings.Contains(msg.Content, "专家模型未初始化") {
		t.Fatalf("unexpected stream content: %#v", msg)
	}
	_, recvErr = stream.Recv()
	if recvErr == nil || recvErr != io.EOF {
		t.Fatalf("expected EOF, got: %v", recvErr)
	}
}

func TestOrchestratorStreamExecuteSequential(t *testing.T) {
	reg := &fakeRegistry{
		experts: map[string]*Expert{
			"service_expert": {Name: "service_expert"},
			"k8s_expert":     {Name: "k8s_expert"},
		},
	}
	orch := NewOrchestrator(reg, NewResultAggregator(AggregationTemplate, nil))
	stream, err := orch.StreamExecute(context.Background(), &ExecuteRequest{
		Message: "服务发布失败",
		Decision: &RouteDecision{
			PrimaryExpert: "service_expert",
			HelperExperts: []string{"k8s_expert"},
			Strategy:      StrategySequential,
			Source:        "scene",
		},
		History: []*schema.Message{},
	})
	if err != nil {
		t.Fatalf("stream execute: %v", err)
	}
	defer stream.Close()

	var combined strings.Builder
	for {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatalf("recv stream: %v", recvErr)
		}
		if msg != nil {
			combined.WriteString(msg.Content)
		}
	}
	out := combined.String()
	if !strings.Contains(out, "专家模型未初始化") {
		t.Fatalf("unexpected stream output: %s", out)
	}
	if !strings.Contains(out, "综合分析") {
		t.Fatalf("expected merged summary in stream output: %s", out)
	}
}
