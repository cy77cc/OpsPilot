package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/executor"
	"github.com/cy77cc/OpsPilot/internal/ai/planner"
	"github.com/cy77cc/OpsPilot/internal/ai/runtime"
)

type Input struct {
	Message string
	Plan    *planner.ExecutionPlan
	State   runtime.ExecutionState
	Steps   []executor.StepResult
}

type ReplanHint struct {
	Reason          string   `json:"reason,omitempty"`
	Focus           string   `json:"focus,omitempty"`
	MissingEvidence []string `json:"missing_evidence,omitempty"`
}

type SummaryOutput struct {
	Summary               string      `json:"summary"`
	Conclusion            string      `json:"conclusion,omitempty"`
	NextActions           []string    `json:"next_actions,omitempty"`
	NeedMoreInvestigation bool        `json:"need_more_investigation"`
	Narrative             string      `json:"narrative"`
	ReplanHint            *ReplanHint `json:"replan_hint,omitempty"`
}

type Summarizer struct {
	runner *adk.Runner
}

func New(runner *adk.Runner) *Summarizer {
	return &Summarizer{runner: runner}
}

func (s *Summarizer) Summarize(ctx context.Context, in Input) (SummaryOutput, error) {
	out := heuristicSummary(in)
	if s == nil || s.runner == nil {
		return out, nil
	}
	raw, err := runADKSummarizer(ctx, s.runner, buildPromptInput(in))
	if err != nil {
		return out, nil
	}
	var parsed SummaryOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil {
		return out, nil
	}
	if strings.TrimSpace(parsed.Summary) == "" {
		return out, nil
	}
	if strings.TrimSpace(parsed.Narrative) == "" {
		parsed.Narrative = out.Narrative
	}
	if strings.TrimSpace(parsed.Conclusion) == "" {
		parsed.Conclusion = out.Conclusion
	}
	if len(parsed.NextActions) == 0 {
		parsed.NextActions = out.NextActions
	}
	if parsed.ReplanHint == nil {
		parsed.ReplanHint = out.ReplanHint
	}
	return parsed, nil
}

func heuristicSummary(in Input) SummaryOutput {
	output := SummaryOutput{
		Summary:   "本轮 AI 编排已生成阶段结果。",
		Narrative: "Summarizer 基于当前步骤结果生成结构化结论，并保留不确定性说明。",
	}
	if in.State.PendingApproval != nil && in.State.PendingApproval.Status == "pending" {
		title := firstNonEmpty(in.State.PendingApproval.Title, in.State.PendingApproval.StepID)
		output.Summary = fmt.Sprintf("步骤 %q 正在等待审批。", title)
		output.Conclusion = "当前计划已暂停，等待你确认后继续执行。"
		output.NextActions = []string{"确认当前审批请求", "如有需要补充审批原因"}
		output.Narrative = "执行链已到达需要用户决策的节点，因此先向用户回报当前状态，而不是伪造完成结论。"
		return output
	}

	completed := 0
	failed := 0
	blocked := 0
	for _, step := range in.Steps {
		switch step.Status {
		case runtime.StepCompleted:
			completed++
		case runtime.StepFailed:
			failed++
		case runtime.StepBlocked, runtime.StepCancelled:
			blocked++
		}
	}

	if failed > 0 || blocked > 0 {
		output.Summary = "当前证据不足以形成稳定结论。"
		output.Conclusion = "执行链中存在失败或阻断步骤，建议补充调查后再继续决策。"
		output.NextActions = []string{"检查失败步骤的错误详情", "补充缺失证据后重新规划"}
		output.NeedMoreInvestigation = true
		output.ReplanHint = &ReplanHint{
			Reason:          "executor_has_failed_or_blocked_steps",
			Focus:           "补充失败步骤所需的资源信息或调查证据",
			MissingEvidence: []string{"failed_or_blocked_step_evidence"},
		}
		output.Narrative = "Summarizer 发现执行链中仍有失败或阻断步骤，因此明确标记 need_more_investigation=true。"
		return output
	}

	goal := strings.TrimSpace(in.Message)
	if in.Plan != nil && strings.TrimSpace(in.Plan.Goal) != "" {
		goal = strings.TrimSpace(in.Plan.Goal)
	}
	output.Summary = fmt.Sprintf("已围绕目标“%s”完成 %d 个步骤。", goal, completed)
	output.Conclusion = "当前执行链已经完成本轮计划，可继续查看正文回答获取自然语言说明。"
	output.NextActions = []string{"查看最终结论正文", "如需更深入调查，可继续追加问题"}
	return output
}

func buildPromptInput(in Input) string {
	data, _ := json.Marshal(in)
	return string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
