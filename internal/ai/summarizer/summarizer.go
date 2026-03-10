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
	base := buildBaseSummary(in)
	if s == nil || s.runner == nil {
		return base, nil
	}
	raw, err := runADKSummarizer(ctx, s.runner, buildPromptInput(in))
	if err != nil {
		return base, nil
	}
	var parsed SummaryOutput
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil {
		return base, nil
	}
	return normalizeSummary(base, parsed), nil
}

func buildBaseSummary(in Input) SummaryOutput {
	output := SummaryOutput{
		Summary:    "本轮执行已结束，结果已交给 Summarizer 生成结构化结论。",
		Conclusion: "可继续查看正文回答获取自然语言结论。",
		Narrative:  "Summarizer 模型不可用时，使用当前执行状态生成最小结构化摘要。",
	}
	if in.State.PendingApproval != nil && in.State.PendingApproval.Status == "pending" {
		title := firstNonEmpty(in.State.PendingApproval.Title, in.State.PendingApproval.StepID)
		output.Summary = fmt.Sprintf("步骤 %q 正在等待审批。", title)
		output.Conclusion = "当前计划已暂停，等待你确认后继续执行。"
		output.NextActions = []string{"确认当前审批请求"}
		output.Narrative = "执行链到达审批节点，当前只能回报暂停状态。"
		return output
	}

	completed, failed, blocked := summarizeStepCounts(in.Steps)
	evidenceCount := countEvidence(in.Steps)

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
		output.Narrative = "当前执行状态显示仍有失败或阻断步骤。"
		return output
	}
	if completed > 0 && evidenceCount == 0 {
		output.Summary = fmt.Sprintf("已完成 %d 个步骤，但还没有收集到足够的执行证据。", completed)
		output.Conclusion = "当前只能给出初步判断，仍需补充执行证据后再确认结论。"
		output.NextActions = []string{"补充关键步骤的执行证据", "基于新增证据重新总结结论"}
		output.NeedMoreInvestigation = true
		output.ReplanHint = &ReplanHint{
			Reason:          "completed_steps_without_evidence",
			Focus:           "补充已完成步骤对应的工具输出或观察证据",
			MissingEvidence: []string{"step_evidence"},
		}
		output.Narrative = "步骤虽然完成，但缺少可支撑结论的 StepResult/Evidence。"
		return output
	}

	goal := strings.TrimSpace(in.Message)
	if in.Plan != nil && strings.TrimSpace(in.Plan.Goal) != "" {
		goal = strings.TrimSpace(in.Plan.Goal)
	}
	output.Summary = fmt.Sprintf("已围绕目标“%s”完成 %d 个步骤，并收集到 %d 条执行证据。", goal, completed, evidenceCount)
	output.Conclusion = "当前结论基于已执行步骤及其证据生成，可继续查看正文回答获取自然语言说明。"
	output.NextActions = []string{"查看最终结论正文"}
	output.Narrative = "当前总结基于 StepResult 与执行证据汇总得出。"
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

func normalizeSummary(base, parsed SummaryOutput) SummaryOutput {
	if strings.TrimSpace(parsed.Summary) == "" {
		parsed.Summary = base.Summary
	}
	if strings.TrimSpace(parsed.Conclusion) == "" {
		parsed.Conclusion = base.Conclusion
	}
	if len(parsed.NextActions) == 0 {
		parsed.NextActions = base.NextActions
	}
	if strings.TrimSpace(parsed.Narrative) == "" {
		parsed.Narrative = base.Narrative
	}
	if parsed.ReplanHint == nil && base.ReplanHint != nil {
		parsed.ReplanHint = base.ReplanHint
	}
	if parsed.NeedMoreInvestigation {
		parsed.Conclusion = qualifyUncertainConclusion(parsed.Conclusion)
		parsed.Narrative = qualifyUncertainNarrative(parsed.Narrative)
		return parsed
	}
	parsed.NeedMoreInvestigation = base.NeedMoreInvestigation
	if parsed.NeedMoreInvestigation {
		if parsed.ReplanHint == nil {
			parsed.ReplanHint = base.ReplanHint
		}
		parsed.Conclusion = qualifyUncertainConclusion(parsed.Conclusion)
		parsed.Narrative = qualifyUncertainNarrative(parsed.Narrative)
	}
	return parsed
}

func summarizeStepCounts(steps []executor.StepResult) (completed, failed, blocked int) {
	for _, step := range steps {
		switch step.Status {
		case runtime.StepCompleted:
			completed++
		case runtime.StepFailed:
			failed++
		case runtime.StepBlocked, runtime.StepCancelled:
			blocked++
		}
	}
	return completed, failed, blocked
}

func countEvidence(steps []executor.StepResult) int {
	total := 0
	for _, step := range steps {
		total += len(step.Evidence)
	}
	return total
}

func qualifyUncertainConclusion(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "当前仅能基于现有证据给出初步判断，仍需进一步调查。"
	}
	if containsUncertaintyMarker(text) {
		return text
	}
	return text + " 当前仅为基于现有证据的初步判断，仍需进一步调查。"
}

func qualifyUncertainNarrative(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "当前叙述仅基于已完成步骤和现有证据，不足部分仍待补充确认。"
	}
	if containsUncertaintyMarker(text) && strings.Contains(text, "证据") {
		return text
	}
	if !strings.Contains(text, "证据") {
		text += " 以上内容仅基于当前执行证据。"
	}
	if !containsUncertaintyMarker(text) {
		text += " 仍存在待确认的不确定性。"
	}
	return strings.TrimSpace(text)
}

func containsUncertaintyMarker(text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	markers := []string{"可能", "初步", "待确认", "不确定", "证据不足", "进一步调查", "尚不能", "仍需"}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
