package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

type Planner struct {
	runner *adk.Runner
}

type Input struct {
	Message string
	Rewrite rewrite.Output
}

type DecisionType string

const (
	DecisionClarify     DecisionType = "clarify"
	DecisionReject      DecisionType = "reject"
	DecisionDirectReply DecisionType = "direct_reply"
	DecisionPlan        DecisionType = "plan"
)

type Decision struct {
	Type       DecisionType     `json:"type"`
	Message    string           `json:"message,omitempty"`
	Reason     string           `json:"reason,omitempty"`
	Candidates []map[string]any `json:"candidates,omitempty"`
	Plan       *ExecutionPlan   `json:"plan,omitempty"`
	Narrative  string           `json:"narrative"`
}

type ExecutionPlan struct {
	PlanID    string            `json:"plan_id"`
	Goal      string            `json:"goal"`
	Resolved  ResolvedResources `json:"resolved"`
	Narrative string            `json:"narrative"`
	Steps     []PlanStep        `json:"steps"`
}

type ResolvedResources struct {
	ServiceName string   `json:"service_name,omitempty"`
	ClusterName string   `json:"cluster_name,omitempty"`
	HostNames   []string `json:"host_names,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
}

type PlanStep struct {
	StepID    string         `json:"step_id"`
	Title     string         `json:"title"`
	Expert    string         `json:"expert"`
	Intent    string         `json:"intent"`
	Task      string         `json:"task"`
	Input     map[string]any `json:"input,omitempty"`
	DependsOn []string       `json:"depends_on,omitempty"`
	Mode      string         `json:"mode"`
	Risk      string         `json:"risk"`
	Narrative string         `json:"narrative,omitempty"`
}

func New(runner *adk.Runner) *Planner {
	return &Planner{runner: runner}
}

func (p *Planner) Plan(ctx context.Context, in Input) (Decision, error) {
	out := heuristicPlan(in)

	if p == nil || p.runner == nil {
		return out, nil
	}
	raw, err := runADKPlanner(ctx, p.runner, buildPromptInput(in))
	if err != nil {
		return out, nil
	}

	parsed, err := ParseDecision(strings.TrimSpace(raw))
	if err != nil {
		return out, nil
	}
	return mergeDecision(out, parsed), nil
}

func ParseDecision(raw string) (Decision, error) {
	var out Decision
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return Decision{}, err
	}
	if out.Type == "" {
		return Decision{}, fmt.Errorf("planner decision missing type")
	}
	return out, nil
}

func heuristicPlan(in Input) Decision {
	rewritten := in.Rewrite
	if len(rewritten.AmbiguityFlags) > 0 {
		return Decision{
			Type:      DecisionClarify,
			Message:   "我需要先确认目标资源后再继续规划。",
			Narrative: "Rewrite 阶段识别到目标资源仍有歧义，Planner 选择先澄清而不是继续生成执行计划。",
			Candidates: []map[string]any{
				{"kind": "resource_target", "message": strings.Join(rewritten.AmbiguityFlags, ", ")},
			},
		}
	}

	if rewritten.OperationMode == "query" && looksLikeDirectReply(in.Message) {
		return Decision{
			Type:      DecisionDirectReply,
			Message:   "我可以继续帮你做服务排查、发布检查、监控分析和运维辅助。你可以直接说目标对象和想做的事。",
			Narrative: "该请求更适合直接回答，不需要进入执行计划。",
		}
	}

	planID := uuid.NewString()
	steps := buildPlanSteps(rewritten)
	return Decision{
		Type:      DecisionPlan,
		Narrative: "Planner 已根据 Rewrite 输出生成初步执行计划，后续执行层可按 step 依赖继续推进。",
		Plan: &ExecutionPlan{
			PlanID: planID,
			Goal:   firstNonEmpty(rewritten.NormalizedGoal, strings.TrimSpace(in.Message)),
			Resolved: ResolvedResources{
				ServiceName: rewritten.ResourceHints.ServiceName,
				ClusterName: rewritten.ResourceHints.ClusterName,
				Namespace:   rewritten.ResourceHints.Namespace,
			},
			Narrative: "该计划围绕用户当前目标组织调查步骤，并保留结构化的 mode/risk 信息。",
			Steps:     steps,
		},
	}
}

func buildPromptInput(in Input) string {
	data, _ := json.Marshal(in.Rewrite)
	return "message: " + strings.TrimSpace(in.Message) + "\nrewrite: " + string(data)
}

func buildPlanSteps(r rewrite.Output) []PlanStep {
	mode := "readonly"
	risk := "low"
	if r.OperationMode == "mutate" {
		mode = "mutating"
		risk = "high"
	}

	steps := make([]PlanStep, 0, 3)
	appendStep := func(id, title, expert, intent, task string, dependsOn ...string) {
		steps = append(steps, PlanStep{
			StepID:    id,
			Title:     title,
			Expert:    expert,
			Intent:    intent,
			Task:      task,
			DependsOn: dependsOn,
			Mode:      mode,
			Risk:      risk,
			Narrative: task,
			Input: map[string]any{
				"service_name": r.ResourceHints.ServiceName,
				"cluster_name": r.ResourceHints.ClusterName,
				"namespace":    r.ResourceHints.Namespace,
			},
		})
	}

	stepNo := 1
	appendFromDomain := func(domain string) {
		id := fmt.Sprintf("step-%d", stepNo)
		stepNo++
		switch domain {
		case "observability":
			appendStep(id, "检查监控与异常", "observability", "collect_evidence", "检查延迟、错误率、日志或告警信号。")
		case "delivery":
			appendStep(id, "核对近期发布", "delivery", "check_release", "核对近期发布、流水线或变更记录。")
		case "k8s":
			appendStep(id, "检查集群工作负载", "k8s", "inspect_workload", "检查相关工作负载、Pod 状态与命名空间上下文。")
		case "hostops":
			appendStep(id, "检查主机层运行状态", "hostops", "inspect_hosts", "检查主机资源或运行状态是否异常。")
		default:
			appendStep(id, "检查服务状态", "service", "inspect_service", "检查服务当前状态、配置和运行表现。")
		}
	}

	for _, domain := range r.DomainHints {
		appendFromDomain(domain)
		if len(steps) >= 3 {
			break
		}
	}
	if len(steps) == 0 {
		appendFromDomain("service")
	}
	return steps
}

func looksLikeDirectReply(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))
	return strings.HasPrefix(lower, "你能") ||
		strings.HasPrefix(lower, "你会") ||
		strings.Contains(lower, "help") ||
		strings.Contains(lower, "怎么用")
}

func mergeDecision(fallback, parsed Decision) Decision {
	if parsed.Type == "" {
		return fallback
	}
	if strings.TrimSpace(parsed.Narrative) == "" {
		parsed.Narrative = fallback.Narrative
	}
	if parsed.Type == DecisionPlan && parsed.Plan == nil {
		parsed.Plan = fallback.Plan
	}
	if parsed.Type == DecisionClarify && strings.TrimSpace(parsed.Message) == "" {
		parsed.Message = fallback.Message
	}
	if parsed.Type == DecisionDirectReply && strings.TrimSpace(parsed.Message) == "" {
		parsed.Message = fallback.Message
	}
	return parsed
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
