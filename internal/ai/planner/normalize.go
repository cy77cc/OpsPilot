// Package planner 实现 AI 编排的规划阶段。
//
// 本文件负责规划决策的规范化、步骤输入填充和执行前提校验。
package planner

import (
	"fmt"
	"strings"
)

// primaryID 返回列表中第一个非零 ID，无则返回 0。
func primaryID(refs []ResourceRef) int {
	for _, ref := range refs {
		if ref.ID > 0 {
			return ref.ID
		}
	}
	return 0
}

// primaryName 返回列表中第一个非空名称，无则返回 ""。
func primaryName(refs []ResourceRef) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.Name) != "" {
			return strings.TrimSpace(ref.Name)
		}
	}
	return ""
}

// allIDs 提取列表中所有 ID > 0 的值。
func allIDs(refs []ResourceRef) []int {
	out := make([]int, 0, len(refs))
	for _, ref := range refs {
		if ref.ID > 0 {
			out = append(out, ref.ID)
		}
	}
	return out
}

func normalizeDecision(base *ExecutionPlan, parsed Decision) (Decision, error) {
	parsed = canonicalizeDecision(parsed)
	if parsed.Type == "" {
		return Decision{}, &PlanningError{
			Code:              "planning_invalid",
			UserVisibleReason: "AI 规划结果不可执行，请稍后重试或手动在页面中执行操作。",
		}
	}
	if parsed.Type == DecisionPlan {
		var err error
		parsed.Plan, err = canonicalizePlan(base, parsed.Plan)
		if err != nil {
			return Decision{}, err
		}
		if err := validatePlanPrerequisites(parsed.Plan); err != nil {
			return Decision{}, err
		}
	}
	return parsed, nil
}

func canonicalizeDecision(parsed Decision) Decision {
	if parsed.Type == "" {
		return Decision{}
	}
	return parsed
}

func canonicalizePlan(base, parsed *ExecutionPlan) (*ExecutionPlan, error) {
	if parsed == nil {
		return nil, &PlanningError{
			Code:              "planning_invalid",
			UserVisibleReason: "AI 规划结果不可执行，请稍后重试或手动在页面中执行操作。",
			Cause:             fmt.Errorf("planner plan is nil"),
		}
	}
	if base == nil {
		base = &ExecutionPlan{}
	}
	parsed.Resolved.Namespace = firstNonEmpty(parsed.Resolved.Namespace, base.Resolved.Namespace)
	if len(parsed.Resolved.Services) == 0 {
		parsed.Resolved.Services = cloneResourceRefs(base.Resolved.Services)
	}
	if len(parsed.Resolved.Clusters) == 0 {
		parsed.Resolved.Clusters = cloneResourceRefs(base.Resolved.Clusters)
	}
	if len(parsed.Resolved.Hosts) == 0 {
		parsed.Resolved.Hosts = cloneResourceRefs(base.Resolved.Hosts)
	}
	if len(parsed.Resolved.Pods) == 0 {
		parsed.Resolved.Pods = clonePodRefs(base.Resolved.Pods)
	}
	if parsed.Resolved.Scope == nil && base.Resolved.Scope != nil {
		parsed.Resolved.Scope = cloneScope(base.Resolved.Scope)
	}
	if strings.TrimSpace(parsed.Narrative) == "" {
		parsed.Narrative = base.Narrative
	}
	commonInput := baseStepInput(base)
	for i := range parsed.Steps {
		step := parsed.Steps[i]
		if strings.TrimSpace(step.StepID) == "" {
			step.StepID = fmt.Sprintf("step-%d", i+1)
		}
		mode, risk, err := normalizeModeRisk(step.Mode, step.Risk)
		if err != nil {
			return nil, &PlanningError{
				Code:              "planning_invalid",
				UserVisibleReason: "AI 规划结果包含无效的步骤模式或风险等级，当前无法执行。",
				Cause:             fmt.Errorf("planner step %s has invalid mode/risk: %w", strings.TrimSpace(step.StepID), err),
			}
		}
		step.Mode, step.Risk = mode, risk
		if len(step.DependsOn) > 0 {
			step.DependsOn = dedupe(step.DependsOn)
		}
		step.Input = mergeStepInput(commonInput, step.Input)
		step.Input = populateStepInput(step, parsed.Resolved)
		parsed.Steps[i] = step
	}
	return parsed, nil
}

func baseStepInput(base *ExecutionPlan) map[string]any {
	if base == nil || len(base.Steps) == 0 {
		return map[string]any{}
	}
	return cloneInput(base.Steps[0].Input)
}

func mergeStepInput(base, step map[string]any) map[string]any {
	out := cloneInput(base)
	for key, value := range step {
		out[key] = value
	}
	return out
}

func populateStepInput(step PlanStep, resolved ResolvedResources) map[string]any {
	input := cloneInput(step.Input)
	switch step.Expert {
	case "k8s":
		if clusterID := primaryID(resolved.Clusters); clusterID > 0 && looseIntValue(input["cluster_id"]) == 0 {
			input["cluster_id"] = clusterID
		}
		if resolved.Namespace != "" && strings.TrimSpace(looseStringValue(input["namespace"])) == "" {
			input["namespace"] = resolved.Namespace
		}
		if podName := primaryPodName(resolved.Pods); podName != "" && strings.TrimSpace(looseStringValue(input["pod"])) == "" {
			input["pod"] = podName
		}
		if len(resolved.Pods) > 1 && len(podRefsFromInput(input["pods"])) == 0 {
			input["pods"] = podRefsToAny(resolved.Pods)
		}
		if resolved.Scope != nil && input["scope"] == nil {
			input["scope"] = scopeToMap(resolved.Scope)
		}
	case "service":
		if serviceID := primaryID(resolved.Services); serviceID > 0 && looseIntValue(input["service_id"]) == 0 {
			input["service_id"] = serviceID
		}
		if clusterID := primaryID(resolved.Clusters); clusterID > 0 && looseIntValue(input["cluster_id"]) == 0 {
			input["cluster_id"] = clusterID
		}
		if len(resolved.Services) > 1 && len(intSliceValue(input["service_ids"])) == 0 {
			input["service_ids"] = resourceIDs(resolved.Services)
		}
		if resolved.Scope != nil && input["scope"] == nil {
			input["scope"] = scopeToMap(resolved.Scope)
		}
	case "hostops":
		hostIDs := allIDs(resolved.Hosts)
		if len(hostIDs) == 1 && looseIntValue(input["host_id"]) == 0 {
			input["host_id"] = hostIDs[0]
		}
		if len(hostIDs) > 1 && len(intSliceValue(input["host_ids"])) == 0 {
			input["host_ids"] = hostIDs
		}
		if resolved.Scope != nil && input["scope"] == nil {
			input["scope"] = scopeToMap(resolved.Scope)
		}
	}
	return input
}

// primaryPodName 返回第一个非空 Pod 名称。
func primaryPodName(pods []PodRef) string {
	for _, pod := range pods {
		if name := strings.TrimSpace(pod.Name); name != "" {
			return name
		}
	}
	return ""
}

func validatePlanPrerequisites(plan *ExecutionPlan) error {
	if plan == nil {
		return &PlanningError{
			Code:              "planning_invalid",
			UserVisibleReason: "AI 规划结果不可执行，请稍后重试或手动在页面中执行操作。",
			Cause:             fmt.Errorf("planner plan is nil"),
		}
	}
	if strings.TrimSpace(plan.PlanID) == "" || strings.TrimSpace(plan.Goal) == "" || len(plan.Steps) == 0 {
		return &PlanningError{
			Code:              "planning_invalid",
			UserVisibleReason: "AI 规划结果缺少必要字段，当前无法执行。",
			Cause:             fmt.Errorf("planner plan is missing plan_id, goal, or steps"),
		}
	}
	for _, step := range plan.Steps {
		if strings.TrimSpace(step.Title) == "" || strings.TrimSpace(step.Expert) == "" || strings.TrimSpace(step.Task) == "" {
			return &PlanningError{
				Code:              "planning_invalid",
				UserVisibleReason: "AI 规划结果缺少必要步骤字段，当前无法执行。",
				Cause:             fmt.Errorf("planner step %s missing title, expert, or task", strings.TrimSpace(step.StepID)),
			}
		}
		if !isSupportedExpert(step.Expert) {
			return &PlanningError{
				Code:              "planning_invalid",
				UserVisibleReason: "AI 规划结果包含未知专家类型，当前无法执行。",
				Cause:             fmt.Errorf("planner step %s has unsupported expert %q", strings.TrimSpace(step.StepID), strings.TrimSpace(step.Expert)),
			}
		}
		if !isSupportedMode(step.Mode) {
			return &PlanningError{
				Code:              "planning_invalid",
				UserVisibleReason: "AI 规划结果包含无效的步骤模式，当前无法执行。",
				Cause:             fmt.Errorf("planner step %s has unsupported mode %q", strings.TrimSpace(step.StepID), strings.TrimSpace(step.Mode)),
			}
		}
		if !isSupportedRisk(step.Risk) {
			return &PlanningError{
				Code:              "planning_invalid",
				UserVisibleReason: "AI 规划结果包含无效的风险等级，当前无法执行。",
				Cause:             fmt.Errorf("planner step %s has unsupported risk %q", strings.TrimSpace(step.StepID), strings.TrimSpace(step.Risk)),
			}
		}
		switch step.Expert {
		case "k8s":
			if resolvedClusterID(step, plan.Resolved) <= 0 {
				return &PlanningError{
					Code:              "planning_invalid",
					UserVisibleReason: "AI 规划结果缺少 cluster_id，当前无法执行 Kubernetes 相关步骤。",
					Cause:             fmt.Errorf("kubernetes step missing cluster_id"),
				}
			}
			if requiresK8sPodTarget(step, plan.Resolved) && resolvedPodName(step, plan.Resolved) == "" {
				return &PlanningError{
					Code:              "planning_invalid",
					UserVisibleReason: "AI 规划结果缺少 pod 标识，当前无法执行 Kubernetes Pod 相关步骤。",
					Cause:             fmt.Errorf("kubernetes pod step missing pod target"),
				}
			}
		case "service":
			if requiresServiceID(step) && resolvedServiceID(step, plan.Resolved) <= 0 {
				return &PlanningError{
					Code:              "planning_invalid",
					UserVisibleReason: "AI 规划结果缺少 service_id，当前无法执行服务相关步骤。",
					Cause:             fmt.Errorf("service step missing service_id"),
				}
			}
			if requiresClusterID(step) && resolvedClusterID(step, plan.Resolved) <= 0 {
				return &PlanningError{
					Code:              "planning_invalid",
					UserVisibleReason: "AI 规划结果缺少 cluster_id，当前无法执行服务部署相关步骤。",
					Cause:             fmt.Errorf("service mutating step missing cluster_id"),
				}
			}
		case "hostops":
			if requiresHostTarget(step, plan.Resolved) && len(resolvedHostIDs(step, plan.Resolved)) == 0 && !hasResolvedScope(step, plan.Resolved, "host") {
				return &PlanningError{
					Code:              "planning_invalid",
					UserVisibleReason: "AI 规划结果缺少 host_id 或 host_ids，当前无法执行主机相关步骤。",
					Cause:             fmt.Errorf("hostops step missing host target"),
				}
			}
		}
	}
	return nil
}

func requiresServiceID(step PlanStep) bool {
	return step.Expert == "service"
}

func requiresClusterID(step PlanStep) bool {
	return step.Expert == "service" && step.Mode == "mutating"
}

func requiresK8sPodTarget(step PlanStep, resolved ResolvedResources) bool {
	if resolvedPodName(step, resolved) != "" {
		return true
	}
	return hasTargetType(step, "pod")
}

func requiresHostTarget(step PlanStep, resolved ResolvedResources) bool {
	if hasResolvedScope(step, resolved, "host") {
		return false
	}
	if len(resolvedHostIDs(step, resolved)) > 0 {
		return true
	}
	if len(resolved.Hosts) > 0 {
		return true
	}
	return hasTargetType(step, "host")
}

func resolvedClusterID(step PlanStep, resolved ResolvedResources) int {
	if clusterID := looseIntValue(step.Input["cluster_id"]); clusterID > 0 {
		return clusterID
	}
	return primaryID(resolved.Clusters)
}

func resolvedServiceID(step PlanStep, resolved ResolvedResources) int {
	if serviceID := looseIntValue(step.Input["service_id"]); serviceID > 0 {
		return serviceID
	}
	if ids := intSliceValue(step.Input["service_ids"]); len(ids) == 1 {
		return ids[0]
	}
	return primaryID(resolved.Services)
}

func resolvedPodName(step PlanStep, resolved ResolvedResources) string {
	if pod := strings.TrimSpace(looseStringValue(step.Input["pod"])); pod != "" {
		return pod
	}
	if name := primaryPodName(resolved.Pods); name != "" {
		return name
	}
	return targetNameForType(step, "pod")
}

func resolvedHostIDs(step PlanStep, resolved ResolvedResources) []int {
	if hostID := looseIntValue(step.Input["host_id"]); hostID > 0 {
		return []int{hostID}
	}
	if hostIDs := intSliceValue(step.Input["host_ids"]); len(hostIDs) > 0 {
		return hostIDs
	}
	return allIDs(resolved.Hosts)
}

func hasResolvedScope(step PlanStep, resolved ResolvedResources, resourceType string) bool {
	if scope := parseResourceScope(step.Input["scope"]); scope != nil {
		return strings.EqualFold(strings.TrimSpace(scope.ResourceType), resourceType) && strings.TrimSpace(scope.Kind) != ""
	}
	if resolved.Scope == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(resolved.Scope.ResourceType), resourceType) && strings.TrimSpace(resolved.Scope.Kind) != ""
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

func targetNameForType(step PlanStep, want string) string {
	raw, ok := step.Input["normalized_request"].(map[string]any)
	if !ok {
		return ""
	}
	targets, ok := raw["targets"].([]any)
	if !ok {
		return ""
	}
	for _, target := range targets {
		item, ok := target.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(looseStringValue(item["type"])), strings.TrimSpace(want)) {
			return strings.TrimSpace(looseStringValue(item["name"]))
		}
	}
	return ""
}

func isSupportedExpert(expert string) bool {
	switch strings.TrimSpace(expert) {
	case "hostops", "k8s", "service", "delivery", "observability":
		return true
	default:
		return false
	}
}

func isSupportedMode(mode string) bool {
	return strings.TrimSpace(mode) == "readonly" || strings.TrimSpace(mode) == "mutating"
}

func isSupportedRisk(risk string) bool {
	switch strings.TrimSpace(risk) {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

func normalizeModeRisk(mode, risk string) (string, string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "mutating", "mutate", "write", "apply", "edit":
		return "mutating", normalizedRisk(risk, "high"), nil
	case "analysis", "query", "inspect", "read", "readonly", "":
		return "readonly", normalizedRisk(risk, "low"), nil
	default:
		return "", "", fmt.Errorf("unsupported mode %q", strings.TrimSpace(mode))
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

func cloneResourceRefs(in []ResourceRef) []ResourceRef {
	if len(in) == 0 {
		return nil
	}
	out := make([]ResourceRef, len(in))
	copy(out, in)
	return out
}

func clonePodRefs(in []PodRef) []PodRef {
	if len(in) == 0 {
		return nil
	}
	out := make([]PodRef, len(in))
	copy(out, in)
	return out
}

func cloneScope(in *ResourceScope) *ResourceScope {
	if in == nil {
		return nil
	}
	out := &ResourceScope{
		Kind:         in.Kind,
		ResourceType: in.ResourceType,
	}
	if len(in.Selector) > 0 {
		out.Selector = cloneInput(in.Selector)
	}
	return out
}

func scopeToMap(scope *ResourceScope) map[string]any {
	if scope == nil {
		return nil
	}
	out := map[string]any{
		"kind":          scope.Kind,
		"resource_type": scope.ResourceType,
	}
	if len(scope.Selector) > 0 {
		out["selector"] = cloneInput(scope.Selector)
	}
	return out
}

func resourceIDs(refs []ResourceRef) []int {
	out := make([]int, 0, len(refs))
	for _, ref := range refs {
		if ref.ID > 0 {
			out = append(out, ref.ID)
		}
	}
	return out
}

func podRefsFromInput(value any) []PodRef {
	return parsePodRefs(value)
}

func podRefsToAny(refs []PodRef) []map[string]any {
	out := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		out = append(out, map[string]any{
			"name":       ref.Name,
			"namespace":  ref.Namespace,
			"cluster_id": ref.ClusterID,
		})
	}
	return out
}
