package executor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/experts/hostops"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

func TestParseExpertResultRecoversHalfStructuredOutput(t *testing.T) {
	raw := `Summary: 已成功检查 payment-api 服务状态
Observed: service_id=42 状态正常
Inference: 当前没有发现异常
Next: 如需更深排查可继续查看最近日志`

	out, err := parseExpertResult(raw)
	if err != nil {
		t.Fatalf("parseExpertResult() error = %v", err)
	}
	if out.Summary == "" {
		t.Fatalf("Summary should not be empty: %#v", out)
	}
	if len(out.ObservedFacts) == 0 {
		t.Fatalf("ObservedFacts should be recovered: %#v", out)
	}
	if len(out.Inferences) == 0 {
		t.Fatalf("Inferences should be recovered: %#v", out)
	}
	if len(out.NextActions) == 0 {
		t.Fatalf("NextActions should be recovered: %#v", out)
	}
}

func TestBuildExpertRequestUsesStructuredEnvelope(t *testing.T) {
	raw := buildExpertRequest(Request{
		Message: "查看 payment-api 状态",
		Plan: planner.ExecutionPlan{
			Goal: "检查 payment-api 当前运行状态",
		},
		RuntimeContext: runtime.ContextSnapshot{
			Scene:       "scene:service",
			CurrentPage: "/services/42",
			ResourceIDs: []string{"42"},
		},
	}, planner.PlanStep{
		StepID: "step-1",
		Title:  "检查服务状态",
		Expert: "service",
		Intent: "inspect_service",
		Task:   "inspect payment-api",
		Mode:   "readonly",
		Risk:   "low",
		Input: map[string]any{
			"service_id": 42,
		},
		DependsOn: []string{"step-0"},
	})

	var envelope expertRequestEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("buildExpertRequest() produced invalid json: %v", err)
	}
	if envelope.UserMessage != "查看 payment-api 状态" {
		t.Fatalf("UserMessage = %q", envelope.UserMessage)
	}
	if envelope.Step.ID != "step-1" || envelope.Step.Expert != "service" {
		t.Fatalf("Step = %#v", envelope.Step)
	}
	if got := envelope.Step.Input["service_id"]; got != 42.0 && got != 42 {
		t.Fatalf("service_id = %#v, want 42", got)
	}
	if !envelope.HostConstraints.ApprovalAlreadyDecided || !envelope.HostConstraints.UseOnlyProvidedTools || !envelope.HostConstraints.StayInAssignedDomain {
		t.Fatalf("HostConstraints = %#v", envelope.HostConstraints)
	}
	if envelope.RuntimeContext["scene"] != "scene:service" {
		t.Fatalf("RuntimeContext = %#v", envelope.RuntimeContext)
	}
}

func TestHostopsExpertPromptRequiresMutatingApplyFlow(t *testing.T) {
	prompt := expertSystemPrompt(hostops.New(common.PlatformDeps{}))

	for _, expected := range []string{
		`If step.mode is "mutating", do NOT use host_exec or host_exec_by_target.`,
		`always use host_batch_exec_preview first and then host_batch_exec_apply`,
		`Never downgrade an approved mutating host step into a readonly diagnostic tool call.`,
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("hostops expert prompt missing %q\nprompt=%s", expected, prompt)
		}
	}
}
