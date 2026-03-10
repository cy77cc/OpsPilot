package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
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
	ServiceID   int      `json:"service_id,omitempty"`
	ClusterName string   `json:"cluster_name,omitempty"`
	ClusterID   int      `json:"cluster_id,omitempty"`
	HostNames   []string `json:"host_names,omitempty"`
	HostIDs     []int    `json:"host_ids,omitempty"`
	Namespace   string   `json:"namespace,omitempty"`
	PodName     string   `json:"pod_name,omitempty"`
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
	base := buildBaseDecision(in)

	if p == nil || p.runner == nil {
		return base, nil
	}
	raw, err := runADKPlanner(ctx, p.runner, buildPromptInput(in))
	if err != nil {
		return base, nil
	}

	parsed, err := ParseDecision(strings.TrimSpace(raw))
	if err != nil {
		return base, nil
	}
	return normalizeDecision(base, parsed), nil
}

func ParseDecision(raw string) (Decision, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return Decision{}, err
	}

	out := Decision{
		Type:       DecisionType(strings.TrimSpace(looseStringValue(payload["type"]))),
		Message:    strings.TrimSpace(looseStringValue(payload["message"])),
		Reason:     strings.TrimSpace(looseStringValue(payload["reason"])),
		Narrative:  strings.TrimSpace(looseStringValue(payload["narrative"])),
		Candidates: mapSliceValue(payload["candidates"]),
	}
	if out.Type == "" {
		return Decision{}, fmt.Errorf("planner decision missing type")
	}

	planValue, hasPlan := payload["plan"]
	if out.Type == DecisionPlan || hasPlan {
		plan, err := parseExecutionPlan(planValue)
		if err != nil {
			return Decision{}, err
		}
		out.Plan = plan
	}
	return out, nil
}

func buildBaseDecision(in Input) Decision {
	rewritten := in.Rewrite
	ambiguities := dedupe(append(append([]string(nil), rewritten.AmbiguityFlags...), rewritten.Ambiguities...))
	if len(ambiguities) > 0 {
		return Decision{
			Type:       DecisionClarify,
			Message:    "我需要先确认目标资源后再继续规划。",
			Narrative:  "Rewrite 输出仍有未消解歧义，Planner 先请求补充信息。",
			Candidates: buildClarifyCandidates(ambiguities),
		}
	}

	planID := uuid.NewString()
	goal := firstNonEmpty(rewritten.NormalizedGoal, strings.TrimSpace(in.Message))
	mode, risk := normalizeStepMode(rewritten.OperationMode)
	expert := pickPrimaryExpert(rewritten)
	return Decision{
		Type:      DecisionPlan,
		Narrative: "Planner 模型不可用时，使用最小结构化计划继续交给执行层处理。",
		Plan: &ExecutionPlan{
			PlanID: planID,
			Goal:   goal,
			Resolved: ResolvedResources{
				ServiceName: rewritten.ResourceHints.ServiceName,
				ServiceID:   rewritten.ResourceHints.ServiceID,
				ClusterName: rewritten.ResourceHints.ClusterName,
				ClusterID:   rewritten.ResourceHints.ClusterID,
				Namespace:   rewritten.ResourceHints.Namespace,
				PodName:     collectPodName(rewritten),
				HostNames:   collectHostNames(rewritten),
				HostIDs:     collectHostIDs(rewritten),
			},
			Narrative: "该计划是 Planner 失败时的最小兜底结构，保留用户目标与已知资源线索。",
			Steps: []PlanStep{
				{
					StepID:    "step-1",
					Title:     "处理用户请求",
					Expert:    expert,
					Intent:    "handle_request",
					Task:      goal,
					Mode:      mode,
					Risk:      risk,
					Narrative: goal,
					Input: map[string]any{
						"message":            strings.TrimSpace(in.Message),
						"normalized_request": rewritten.NormalizedRequest,
						"resource_hints":     rewritten.ResourceHints,
					},
				},
			},
		},
	}
}

func buildPromptInput(in Input) string {
	data, _ := json.Marshal(in.Rewrite)
	return "message: " + strings.TrimSpace(in.Message) + "\nrewrite: " + string(data)
}

func normalizeDecision(base, parsed Decision) Decision {
	if parsed.Type == "" {
		return base
	}
	if strings.TrimSpace(parsed.Narrative) == "" {
		parsed.Narrative = base.Narrative
	}
	if parsed.Type == DecisionPlan && parsed.Plan == nil {
		parsed.Plan = base.Plan
	}
	if parsed.Type == DecisionClarify && strings.TrimSpace(parsed.Message) == "" {
		parsed.Message = base.Message
	}
	if parsed.Type == DecisionDirectReply && strings.TrimSpace(parsed.Message) == "" {
		parsed.Message = base.Message
	}
	if parsed.Type == DecisionClarify {
		parsed = normalizeClarifyDecision(base, parsed)
	}
	if parsed.Type == DecisionPlan {
		parsed.Plan = normalizePlan(base.Plan, parsed.Plan)
		if clarify := validatePlanPrerequisites(parsed.Plan); clarify.Type != "" {
			return clarify
		}
	}
	return parsed
}

func normalizePlan(base, parsed *ExecutionPlan) *ExecutionPlan {
	if parsed == nil {
		return base
	}
	if base == nil {
		return parsed
	}
	if strings.TrimSpace(parsed.PlanID) == "" {
		parsed.PlanID = base.PlanID
	}
	if strings.TrimSpace(parsed.Goal) == "" {
		parsed.Goal = base.Goal
	}
	parsed.Resolved.ServiceName = firstNonEmpty(parsed.Resolved.ServiceName, base.Resolved.ServiceName)
	if parsed.Resolved.ServiceID == 0 {
		parsed.Resolved.ServiceID = base.Resolved.ServiceID
	}
	parsed.Resolved.ClusterName = firstNonEmpty(parsed.Resolved.ClusterName, base.Resolved.ClusterName)
	if parsed.Resolved.ClusterID == 0 {
		parsed.Resolved.ClusterID = base.Resolved.ClusterID
	}
	parsed.Resolved.Namespace = firstNonEmpty(parsed.Resolved.Namespace, base.Resolved.Namespace)
	parsed.Resolved.PodName = firstNonEmpty(parsed.Resolved.PodName, base.Resolved.PodName)
	if len(parsed.Resolved.HostNames) == 0 {
		parsed.Resolved.HostNames = base.Resolved.HostNames
	}
	if len(parsed.Resolved.HostIDs) == 0 {
		parsed.Resolved.HostIDs = append([]int(nil), base.Resolved.HostIDs...)
	}
	if strings.TrimSpace(parsed.Narrative) == "" {
		parsed.Narrative = base.Narrative
	}
	if len(parsed.Steps) == 0 {
		parsed.Steps = base.Steps
	}
	for i := range parsed.Steps {
		step := parsed.Steps[i]
		if strings.TrimSpace(step.StepID) == "" {
			step.StepID = fmt.Sprintf("step-%d", i+1)
		}
		if strings.TrimSpace(step.Title) == "" {
			step.Title = fmt.Sprintf("步骤 %d", i+1)
		}
		if strings.TrimSpace(step.Expert) == "" {
			if i < len(base.Steps) {
				step.Expert = base.Steps[i].Expert
			}
			if strings.TrimSpace(step.Expert) == "" {
				step.Expert = "service"
			}
		}
		step.Expert = normalizeExpertName(step.Expert, parsed.Steps, i)
		step.Mode, step.Risk = normalizeModeRisk(step.Mode, step.Risk)
		if len(step.DependsOn) > 0 {
			step.DependsOn = dedupe(step.DependsOn)
		}
		if strings.TrimSpace(step.Narrative) == "" {
			step.Narrative = firstNonEmpty(step.Task, step.Title)
		}
		step.Input = populateStepInput(step, parsed.Resolved)
		parsed.Steps[i] = step
	}
	return parsed
}

func parseExecutionPlan(value any) (*ExecutionPlan, error) {
	if value == nil {
		return nil, nil
	}
	planMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("planner plan must be an object")
	}
	out := &ExecutionPlan{
		PlanID:    strings.TrimSpace(looseStringValue(planMap["plan_id"])),
		Goal:      strings.TrimSpace(looseStringValue(planMap["goal"])),
		Narrative: strings.TrimSpace(looseStringValue(planMap["narrative"])),
		Resolved:  parseResolvedResources(planMap["resolved"]),
		Steps:     parsePlanSteps(planMap["steps"]),
	}
	return out, nil
}

func parseResolvedResources(value any) ResolvedResources {
	raw, ok := value.(map[string]any)
	if !ok {
		return ResolvedResources{}
	}
	serviceName := firstNonEmpty(
		looseStringValue(raw["service_name"]),
		looseStringValue(raw["service"]),
	)
	serviceID := looseIntValue(raw["service_id"])
	clusterName := firstNonEmpty(
		looseStringValue(raw["cluster_name"]),
		looseStringValue(raw["cluster"]),
	)
	clusterID := looseIntValue(raw["cluster_id"])
	if clusterName == "" && clusterID > 0 {
		clusterName = strconv.Itoa(clusterID)
	}
	namespace := firstNonEmpty(
		looseStringValue(raw["namespace"]),
		looseStringValue(raw["ns"]),
	)
	podName := firstNonEmpty(
		looseStringValue(raw["pod_name"]),
		looseStringValue(raw["pod"]),
	)
	hosts := stringSliceValue(raw["host_names"])
	if len(hosts) == 0 {
		hosts = stringSliceValue(raw["hosts"])
	}
	hostIDs := intSliceValue(raw["host_ids"])
	if len(hostIDs) == 0 {
		if hostID := looseIntValue(raw["host_id"]); hostID > 0 {
			hostIDs = []int{hostID}
		}
	}
	return ResolvedResources{
		ServiceName: serviceName,
		ServiceID:   serviceID,
		ClusterName: clusterName,
		ClusterID:   clusterID,
		Namespace:   namespace,
		HostNames:   hosts,
		HostIDs:     hostIDs,
		PodName:     podName,
	}
}

func parsePlanSteps(value any) []PlanStep {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]PlanStep, 0, len(items))
	for i, item := range items {
		raw, ok := item.(map[string]any)
		if !ok {
			continue
		}
		step := PlanStep{
			StepID:    firstNonEmpty(looseStringValue(raw["step_id"]), fmt.Sprintf("step-%d", i+1)),
			Title:     strings.TrimSpace(looseStringValue(raw["title"])),
			Expert:    strings.TrimSpace(looseStringValue(raw["expert"])),
			Intent:    strings.TrimSpace(looseStringValue(raw["intent"])),
			Task:      strings.TrimSpace(looseStringValue(raw["task"])),
			DependsOn: stringSliceValue(raw["depends_on"]),
			Mode:      strings.TrimSpace(looseStringValue(raw["mode"])),
			Risk:      strings.TrimSpace(looseStringValue(raw["risk"])),
			Narrative: strings.TrimSpace(looseStringValue(raw["narrative"])),
		}
		if input, ok := raw["input"].(map[string]any); ok {
			step.Input = input
		}
		out = append(out, step)
	}
	return out
}

func looseStringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		if v == float32(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	default:
		return ""
	}
}

func looseIntValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		out, _ := strconv.Atoi(v.String())
		return out
	case string:
		out, _ := strconv.Atoi(strings.TrimSpace(v))
		return out
	default:
		return 0
	}
}

func stringSliceValue(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(looseStringValue(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func intSliceValue(value any) []int {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		if number := looseIntValue(item); number > 0 {
			out = append(out, number)
		}
	}
	return out
}

func mapSliceValue(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func normalizeExpertName(expert string, steps []PlanStep, index int) string {
	switch strings.ToLower(strings.TrimSpace(expert)) {
	case "host", "hostops", "os":
		return "hostops"
	case "k8s", "kubernetes", "cluster", "pod":
		return "k8s"
	case "service", "app", "application":
		return "service"
	case "delivery", "deploy", "deployment", "cicd", "pipeline":
		return "delivery"
	case "observability", "monitor", "metrics", "topology", "audit":
		return "observability"
	case "analysis":
		if index > 0 && strings.TrimSpace(steps[index-1].Expert) != "" {
			return normalizeExpertName(steps[index-1].Expert, steps, index-1)
		}
		return "observability"
	default:
		return strings.TrimSpace(expert)
	}
}

func normalizeModeRisk(mode, risk string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "mutating", "mutate", "write", "apply", "edit":
		return "mutating", normalizedRisk(risk, "high")
	case "analysis", "query", "inspect", "read", "readonly", "":
		return "readonly", normalizedRisk(risk, "low")
	default:
		return "readonly", normalizedRisk(risk, "low")
	}
}

func normalizedRisk(risk, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(risk))
	default:
		return fallback
	}
}

func populateStepInput(step PlanStep, resolved ResolvedResources) map[string]any {
	input := cloneInput(step.Input)
	switch step.Expert {
	case "k8s":
		if resolved.ClusterID > 0 && looseIntValue(input["cluster_id"]) == 0 {
			input["cluster_id"] = resolved.ClusterID
		}
		if resolved.Namespace != "" && strings.TrimSpace(looseStringValue(input["namespace"])) == "" {
			input["namespace"] = resolved.Namespace
		}
		if resolved.PodName != "" && strings.TrimSpace(looseStringValue(input["pod"])) == "" {
			input["pod"] = resolved.PodName
		}
	case "service":
		if resolved.ServiceID > 0 && looseIntValue(input["service_id"]) == 0 {
			input["service_id"] = resolved.ServiceID
		}
		if resolved.ClusterID > 0 && looseIntValue(input["cluster_id"]) == 0 {
			input["cluster_id"] = resolved.ClusterID
		}
	case "hostops":
		if len(resolved.HostIDs) == 1 && looseIntValue(input["host_id"]) == 0 {
			input["host_id"] = resolved.HostIDs[0]
		}
		if len(resolved.HostIDs) > 1 && len(intSliceValue(input["host_ids"])) == 0 {
			hostIDs := make([]int, len(resolved.HostIDs))
			copy(hostIDs, resolved.HostIDs)
			input["host_ids"] = hostIDs
		}
	}
	return input
}

func cloneInput(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func validatePlanPrerequisites(plan *ExecutionPlan) Decision {
	if plan == nil {
		return Decision{}
	}
	for _, step := range plan.Steps {
		switch step.Expert {
		case "k8s":
			clusterID := looseIntValue(step.Input["cluster_id"])
			if clusterID <= 0 {
				clusterID = plan.Resolved.ClusterID
			}
			if clusterID <= 0 {
				return Decision{
					Type:      DecisionClarify,
					Message:   "需要先明确目标集群后才能执行 Kubernetes 相关步骤。",
					Narrative: "Kubernetes 步骤缺少 cluster_id，Planner 不能继续交给 Executor。",
				}
			}
			if requiresK8sPodTarget(step, plan.Resolved) && strings.TrimSpace(looseStringValue(step.Input["pod"])) == "" {
				return Decision{
					Type:      DecisionClarify,
					Message:   "需要先明确目标 Pod 名称后才能继续执行。",
					Narrative: "Kubernetes Pod 相关步骤缺少 pod 标识。",
				}
			}
		case "service":
			if requiresServiceID(step) && looseIntValue(step.Input["service_id"]) <= 0 {
				return Decision{
					Type:      DecisionClarify,
					Message:   "需要先明确目标服务后才能继续执行服务相关步骤。",
					Narrative: "Service 步骤缺少 service_id。",
				}
			}
			if requiresClusterID(step) && looseIntValue(step.Input["cluster_id"]) <= 0 {
				return Decision{
					Type:      DecisionClarify,
					Message:   "需要先明确目标集群后才能执行服务部署相关步骤。",
					Narrative: "Service 部署步骤缺少 cluster_id。",
				}
			}
		case "hostops":
			if requiresHostTarget(step, plan.Resolved) && looseIntValue(step.Input["host_id"]) <= 0 && len(intSliceValue(step.Input["host_ids"])) == 0 {
				return Decision{
					Type:      DecisionClarify,
					Message:   "需要先明确目标主机后才能继续执行主机相关步骤。",
					Narrative: "HostOps 步骤缺少 host_id 或 host_ids。",
				}
			}
		}
	}
	return Decision{}
}

func requiresServiceID(step PlanStep) bool {
	return step.Expert == "service"
}

func requiresClusterID(step PlanStep) bool {
	text := strings.ToLower(strings.Join([]string{step.Title, step.Intent, step.Task, step.Narrative, step.Mode}, " "))
	return strings.Contains(text, "deploy") || strings.Contains(text, "release") || strings.Contains(text, "cluster") || step.Mode == "mutating"
}

func requiresK8sPodTarget(step PlanStep, resolved ResolvedResources) bool {
	if strings.TrimSpace(looseStringValue(step.Input["pod"])) != "" {
		return true
	}
	if strings.TrimSpace(resolved.PodName) != "" {
		return true
	}
	return hasTargetType(step, "pod")
}

func requiresHostTarget(step PlanStep, resolved ResolvedResources) bool {
	if looseIntValue(step.Input["host_id"]) > 0 || len(intSliceValue(step.Input["host_ids"])) > 0 {
		return true
	}
	if len(resolved.HostIDs) > 0 || len(resolved.HostNames) > 0 {
		return true
	}
	return hasTargetType(step, "host")
}

func hasTargetType(step PlanStep, want string) bool {
	raw, ok := step.Input["normalized_request"].(map[string]any)
	if !ok {
		return false
	}
	targets, ok := raw["targets"].([]any)
	if !ok {
		return false
	}
	for _, target := range targets {
		item, ok := target.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(looseStringValue(item["type"])), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func buildClarifyCandidates(ambiguities []string) []map[string]any {
	out := make([]map[string]any, 0, len(ambiguities))
	for _, item := range ambiguities {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, map[string]any{
			"kind":    "ambiguity",
			"message": item,
		})
	}
	return out
}

func normalizeStepMode(mode string) (string, string) {
	if strings.TrimSpace(mode) == "mutate" {
		return "mutating", "high"
	}
	return "readonly", "low"
}

func pickPrimaryExpert(r rewrite.Output) string {
	for _, domain := range r.DomainHints {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			return domain
		}
	}
	for _, target := range r.NormalizedRequest.Targets {
		switch strings.TrimSpace(target.Type) {
		case "host":
			return "hostops"
		case "cluster", "namespace", "pod", "deployment":
			return "k8s"
		case "pipeline":
			return "delivery"
		case "service":
			return "service"
		}
	}
	return "service"
}

func collectHostNames(r rewrite.Output) []string {
	if strings.TrimSpace(r.ResourceHints.HostName) != "" {
		return []string{strings.TrimSpace(r.ResourceHints.HostName)}
	}
	hosts := make([]string, 0, len(r.NormalizedRequest.Targets))
	for _, target := range r.NormalizedRequest.Targets {
		if strings.TrimSpace(target.Type) != "host" {
			continue
		}
		name := strings.TrimSpace(target.Name)
		if name == "" {
			continue
		}
		hosts = append(hosts, name)
	}
	return dedupe(hosts)
}

func collectHostIDs(r rewrite.Output) []int {
	if r.ResourceHints.HostID > 0 {
		return []int{r.ResourceHints.HostID}
	}
	return nil
}

func collectPodName(r rewrite.Output) string {
	for _, target := range r.NormalizedRequest.Targets {
		if !strings.EqualFold(strings.TrimSpace(target.Type), "pod") {
			continue
		}
		name := strings.TrimSpace(target.Name)
		if name != "" {
			return name
		}
	}
	return ""
}

func normalizeClarifyDecision(base, parsed Decision) Decision {
	if parsed.Type != DecisionClarify {
		return parsed
	}
	message := strings.TrimSpace(parsed.Message)
	if message == "" || base.Plan == nil {
		return parsed
	}
	lower := strings.ToLower(message)
	if strings.Contains(lower, "pod") && strings.Contains(lower, "名称") && strings.TrimSpace(base.Plan.Resolved.PodName) != "" {
		if base.Plan.Resolved.ClusterID <= 0 {
			parsed.Message = "需要先明确目标集群后才能继续执行。"
			parsed.Narrative = "目标 Pod 已明确，但缺少可执行的集群上下文。"
			return parsed
		}
		parsed.Message = "需要补充其他执行上下文后才能继续。"
		parsed.Narrative = "目标 Pod 已明确，原始 clarify 与已解析资源矛盾。"
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

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
