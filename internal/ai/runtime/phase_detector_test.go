package runtime

import (
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestPhaseDetectorDetectByAgentName(t *testing.T) {
	t.Parallel()

	detector := NewPhaseDetector()

	tests := []struct {
		name  string
		event *adk.AgentEvent
		want  string
	}{
		{
			name: "planner agent",
			event: &adk.AgentEvent{
				AgentName: "planner",
			},
			want: "planning",
		},
		{
			name: "executor agent",
			event: &adk.AgentEvent{
				AgentName: "executor",
			},
			want: "executing",
		},
		{
			name: "replanner agent",
			event: &adk.AgentEvent{
				AgentName: "replanner",
			},
			want: "replanning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detector.Detect(tt.event); got != tt.want {
				t.Fatalf("Detect() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPhaseDetectorDetectByMessageContentFallback(t *testing.T) {
	t.Parallel()

	detector := NewPhaseDetector()
	got := detector.Detect(&adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming: false,
				Message:     schema.AssistantMessage("正在重新规划后续执行步骤", nil),
			},
		},
	})
	if got != "replanning" {
		t.Fatalf("Detect() = %q, want replanning", got)
	}
}

func TestPhaseDetectorNextStepID(t *testing.T) {
	t.Parallel()

	detector := NewPhaseDetector()
	if got := detector.NextStepID(); got != "step-1" {
		t.Fatalf("NextStepID() = %q, want step-1", got)
	}
	if got := detector.NextStepID(); got != "step-2" {
		t.Fatalf("NextStepID() = %q, want step-2", got)
	}
}
