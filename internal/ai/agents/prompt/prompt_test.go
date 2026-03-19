package prompt

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestRouterPrompt_EncodesPlatformRoutingAndSafetyBoundaries(t *testing.T) {
	for _, fragment := range []string{
		"AI-native PaaS platform",
		"whether the user needs live runtime evidence",
		"whether the user requests a mutating action",
		"prefer DiagnosisAgent first",
		"always transfer to exactly one sub-agent",
	} {
		if !strings.Contains(ROUTERPROMPT, fragment) {
			t.Fatalf("expected router prompt to contain %q", fragment)
		}
	}
}

func TestChangeExecutorPrompt_EncodesApprovalAwareExecutionRules(t *testing.T) {
	msgs, err := ChangeExecutorPrompt.Format(t.Context(), map[string]any{
		"input":          "scale deployment nginx to 3 replicas",
		"plan":           `{"steps":["precheck target","scale deployment","verify rollout"]}`,
		"executed_steps": "",
		"step":           "precheck target",
	})
	if err != nil {
		t.Fatalf("format change executor prompt: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected formatted messages, got %d", len(msgs))
	}

	content := msgs[0].Content + "\n" + msgs[1].Content
	for _, fragment := range []string{
		"approval-aware change executor",
		"Do not skip ahead to a write operation",
		"precheck",
		"verification",
		"stop at the current boundary",
		"Prefer one high-signal batch or aggregate tool call",
		"Keep tool usage tight",
		"Stop after enough evidence",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected change executor prompt content to contain %q", fragment)
		}
	}
}

func TestDiagnosisExecutorPrompt_EncodesBatchFirstAndToolBudgetRules(t *testing.T) {
	msgs, err := DiagnosisExecutorPrompt.Format(t.Context(), map[string]any{
		"input":          "check why nginx is unhealthy",
		"plan":           `{"steps":["inspect target health","check recent events","summarize findings"]}`,
		"executed_steps": "",
		"step":           "inspect target health",
	})
	if err != nil {
		t.Fatalf("format diagnosis executor prompt: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatalf("expected formatted messages, got %d", len(msgs))
	}

	content := msgs[0].Content + "\n" + msgs[1].Content
	for _, fragment := range []string{
		"diagnosis executor",
		"Prefer one high-signal batch or aggregate tool call",
		"Keep tool usage tight",
		"Stop after enough evidence",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected diagnosis executor prompt content to contain %q", fragment)
		}
	}
}

func TestPlannerPrompts_RequireIdentifierResolutionBeforeScopedToolCalls(t *testing.T) {
	tests := []struct {
		name string
		load func(t *testing.T) string
	}{
		{
			name: "diagnosis planner",
			load: func(t *testing.T) string {
				t.Helper()
				msgs, err := DiagnosisPlannerPrompt.Format(t.Context(), map[string]any{
					"input": []*schema.Message{schema.UserMessage("check nginx pod status")},
				})
				if err != nil {
					t.Fatalf("format diagnosis planner prompt: %v", err)
				}
				if len(msgs) == 0 {
					t.Fatal("expected diagnosis planner prompt messages")
				}
				return msgs[0].Content
			},
		},
		{
			name: "change planner",
			load: func(t *testing.T) string {
				t.Helper()
				msgs, err := ChangePlannerPrompt.Format(t.Context(), map[string]any{
					"input": []*schema.Message{schema.UserMessage("restart nginx deployment")},
				})
				if err != nil {
					t.Fatalf("format change planner prompt: %v", err)
				}
				if len(msgs) == 0 {
					t.Fatal("expected change planner prompt messages")
				}
				return msgs[0].Content
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := tt.load(t)
			for _, fragment := range []string{
				"resolve the required identifiers",
				"Never call or plan a Kubernetes tool with an assumed or omitted cluster_id",
				"ask for clarification instead of guessing",
			} {
				if !strings.Contains(content, fragment) {
					t.Fatalf("expected planner prompt content to contain %q", fragment)
				}
			}
		})
	}
}
