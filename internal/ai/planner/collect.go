// Package planner 实现 AI 编排的规划阶段。
//
// 本文件负责从 rewrite.Output 中提取资源引用，构建 ExecutionPlan 的基础上下文。
package planner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

// buildBasePlanContext 从输入构建基础计划上下文。
func buildBasePlanContext(in Input) *ExecutionPlan {
	rewritten := in.Rewrite
	goal := firstNonEmpty(rewritten.NormalizedGoal, strings.TrimSpace(in.Message))
	resolved := ResolvedResources{
		Namespace: rewritten.ResourceHints.Namespace,
		Services:  collectServices(rewritten),
		Clusters:  collectClusters(rewritten),
		Hosts:     collectHosts(rewritten),
		Pods:      collectPods(rewritten),
		Scope:     detectScope(rewritten),
	}
	return &ExecutionPlan{
		PlanID:   uuid.NewString(),
		Goal:     goal,
		Resolved: resolved,
		Steps: []PlanStep{{
			StepID: "step-1",
			Input: map[string]any{
				"message":            strings.TrimSpace(in.Message),
				"normalized_request": rewritten.NormalizedRequest,
				"resource_hints":     rewritten.ResourceHints,
				"scope":              scopeToMap(detectScope(rewritten)),
			},
		}},
	}
}

func buildPromptInput(in Input) string {
	data, _ := json.Marshal(in.Rewrite.SemanticContract())
	return "message: " + strings.TrimSpace(in.Message) + "\nrewrite: " + string(data)
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

func collectServices(r rewrite.Output) []ResourceRef {
	refs := make([]ResourceRef, 0, 1)
	if r.ResourceHints.ServiceID > 0 || strings.TrimSpace(r.ResourceHints.ServiceName) != "" {
		refs = append(refs, ResourceRef{ID: r.ResourceHints.ServiceID, Name: strings.TrimSpace(r.ResourceHints.ServiceName)})
	}
	for _, target := range r.NormalizedRequest.Targets {
		if !strings.EqualFold(strings.TrimSpace(target.Type), "service") {
			continue
		}
		name := strings.TrimSpace(target.Name)
		if name == "" || isAllKeyword(name) {
			continue
		}
		refs = append(refs, ResourceRef{Name: name})
	}
	return dedupeResourceRefs(refs)
}

func collectClusters(r rewrite.Output) []ResourceRef {
	refs := make([]ResourceRef, 0, 1)
	if r.ResourceHints.ClusterID > 0 || strings.TrimSpace(r.ResourceHints.ClusterName) != "" {
		refs = append(refs, ResourceRef{ID: r.ResourceHints.ClusterID, Name: strings.TrimSpace(r.ResourceHints.ClusterName)})
	}
	for _, target := range r.NormalizedRequest.Targets {
		if !strings.EqualFold(strings.TrimSpace(target.Type), "cluster") {
			continue
		}
		name := strings.TrimSpace(target.Name)
		if name == "" || isAllKeyword(name) {
			continue
		}
		refs = append(refs, ResourceRef{Name: name})
	}
	return dedupeResourceRefs(refs)
}

func collectHosts(r rewrite.Output) []ResourceRef {
	refs := make([]ResourceRef, 0, len(r.NormalizedRequest.Targets)+1)
	if r.ResourceHints.HostID > 0 || strings.TrimSpace(r.ResourceHints.HostName) != "" {
		refs = append(refs, ResourceRef{ID: r.ResourceHints.HostID, Name: strings.TrimSpace(r.ResourceHints.HostName)})
	}
	for _, target := range r.NormalizedRequest.Targets {
		if !strings.EqualFold(strings.TrimSpace(target.Type), "host") {
			continue
		}
		name := strings.TrimSpace(target.Name)
		if name == "" || isAllKeyword(name) {
			continue
		}
		refs = append(refs, ResourceRef{Name: name})
	}
	return dedupeResourceRefs(refs)
}

func collectPods(r rewrite.Output) []PodRef {
	pods := make([]PodRef, 0, len(r.NormalizedRequest.Targets))
	for _, target := range r.NormalizedRequest.Targets {
		if !strings.EqualFold(strings.TrimSpace(target.Type), "pod") {
			continue
		}
		name := strings.TrimSpace(target.Name)
		if name == "" || isAllKeyword(name) {
			continue
		}
		pods = append(pods, PodRef{
			Name:      name,
			Namespace: strings.TrimSpace(r.ResourceHints.Namespace),
			ClusterID: r.ResourceHints.ClusterID,
		})
	}
	if len(pods) == 0 && strings.TrimSpace(r.ResourceHints.Namespace) != "" && strings.TrimSpace(collectPodName(r)) != "" {
		pods = append(pods, PodRef{
			Name:      strings.TrimSpace(collectPodName(r)),
			Namespace: strings.TrimSpace(r.ResourceHints.Namespace),
			ClusterID: r.ResourceHints.ClusterID,
		})
	}
	return dedupePodRefs(pods)
}

func detectScope(r rewrite.Output) *ResourceScope {
	for _, target := range r.NormalizedRequest.Targets {
		name := strings.TrimSpace(target.Name)
		if !isAllKeyword(name) {
			continue
		}
		scope := &ResourceScope{
			Kind:         "all",
			ResourceType: strings.TrimSpace(target.Type),
			Selector:     map[string]any{},
		}
		if ns := strings.TrimSpace(r.ResourceHints.Namespace); ns != "" {
			scope.Selector["namespace"] = ns
		}
		if r.ResourceHints.ClusterID > 0 {
			scope.Selector["cluster_id"] = r.ResourceHints.ClusterID
		}
		if clusterName := strings.TrimSpace(r.ResourceHints.ClusterName); clusterName != "" {
			scope.Selector["cluster_name"] = clusterName
		}
		if len(scope.Selector) == 0 {
			scope.Selector = nil
		}
		return scope
	}
	return nil
}

func dedupeResourceRefs(values []ResourceRef) []ResourceRef {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]ResourceRef, 0, len(values))
	for _, value := range values {
		key := fmt.Sprintf("%d:%s", value.ID, strings.TrimSpace(value.Name))
		if value.ID == 0 && strings.TrimSpace(value.Name) == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value.Name = strings.TrimSpace(value.Name)
		out = append(out, value)
	}
	return out
}

func dedupePodRefs(values []PodRef) []PodRef {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]PodRef, 0, len(values))
	for _, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		value.Namespace = strings.TrimSpace(value.Namespace)
		if value.Name == "" {
			continue
		}
		key := fmt.Sprintf("%s:%s:%d", value.Name, value.Namespace, value.ClusterID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func isAllKeyword(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all", "*", "全部", "所有":
		return true
	default:
		return false
	}
}
