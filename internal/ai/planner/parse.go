// Package planner 实现 AI 编排的规划阶段。
//
// 本文件负责将 LLM 输出的宽松 JSON 解析为结构化的 Decision 和 ExecutionPlan。
// looseStringValue / looseIntValue 等函数专门处理模型可能输出的非标准类型。
package planner

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseDecision 解析规划器输出的 JSON 字符串。
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
	// 读取 LLM 可能输出的平铺字段，用于归并到列表
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
	hostNames := stringSliceValue(raw["host_names"])
	if len(hostNames) == 0 {
		hostNames = stringSliceValue(raw["hosts"])
	}
	hostIDs := intSliceValue(raw["host_ids"])
	if len(hostIDs) == 0 {
		if hostID := looseIntValue(raw["host_id"]); hostID > 0 {
			hostIDs = []int{hostID}
		}
	}
	// 解析列表字段，平铺字段作为 fallback 归并
	services := parseResourceRefs(raw["services"])
	if len(services) == 0 && (serviceID > 0 || serviceName != "") {
		services = []ResourceRef{{ID: serviceID, Name: serviceName}}
	}
	clusters := parseResourceRefs(raw["clusters"])
	if len(clusters) == 0 && (clusterID > 0 || clusterName != "") {
		clusters = []ResourceRef{{ID: clusterID, Name: clusterName}}
	}
	hostRefs := parseResourceRefs(raw["hosts"])
	if len(hostRefs) == 0 {
		for i, id := range hostIDs {
			ref := ResourceRef{ID: id}
			if i < len(hostNames) {
				ref.Name = hostNames[i]
			}
			hostRefs = append(hostRefs, ref)
		}
	}
	if len(hostRefs) == 0 {
		for _, name := range hostNames {
			hostRefs = append(hostRefs, ResourceRef{Name: name})
		}
	}
	pods := parsePodRefs(raw["pods"])
	if len(pods) == 0 && podName != "" {
		pods = []PodRef{{Name: podName, Namespace: namespace, ClusterID: clusterID}}
	}
	scope := parseResourceScope(raw["scope"])
	return ResolvedResources{
		Namespace: namespace,
		Services:  services,
		Clusters:  clusters,
		Hosts:     hostRefs,
		Pods:      pods,
		Scope:     scope,
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

func parseResourceRefs(value any) []ResourceRef {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]ResourceRef, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ref := ResourceRef{
			ID:   looseIntValue(row["id"]),
			Name: strings.TrimSpace(firstNonEmpty(looseStringValue(row["name"]), looseStringValue(row["label"]))),
		}
		if ref.ID > 0 || ref.Name != "" {
			out = append(out, ref)
		}
	}
	return dedupeResourceRefs(out)
}

func parsePodRefs(value any) []PodRef {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]PodRef, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		pod := PodRef{
			Name:      strings.TrimSpace(firstNonEmpty(looseStringValue(row["name"]), looseStringValue(row["pod"]))),
			Namespace: strings.TrimSpace(looseStringValue(row["namespace"])),
			ClusterID: looseIntValue(row["cluster_id"]),
		}
		if pod.Name != "" {
			out = append(out, pod)
		}
	}
	return dedupePodRefs(out)
}

func parseResourceScope(value any) *ResourceScope {
	row, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	scope := &ResourceScope{
		Kind:         strings.TrimSpace(looseStringValue(row["kind"])),
		ResourceType: strings.TrimSpace(looseStringValue(row["resource_type"])),
	}
	if selector, ok := row["selector"].(map[string]any); ok && len(selector) > 0 {
		scope.Selector = cloneInput(selector)
	}
	if scope.Kind == "" || scope.ResourceType == "" {
		return nil
	}
	return scope
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
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text := strings.TrimSpace(looseStringValue(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func intSliceValue(value any) []int {
	switch v := value.(type) {
	case []int:
		out := make([]int, 0, len(v))
		for _, item := range v {
			if item > 0 {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]int, 0, len(v))
		for _, item := range v {
			if number := looseIntValue(item); number > 0 {
				out = append(out, number)
			}
		}
		return out
	default:
		return nil
	}
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
