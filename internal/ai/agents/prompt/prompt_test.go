package prompt

import (
	"strings"
	"testing"
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
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected change executor prompt content to contain %q", fragment)
		}
	}
}
