package rewrite

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/adk"
)

type Output struct {
	RawUserInput      string            `json:"raw_user_input,omitempty"`
	NormalizedRequest NormalizedRequest `json:"normalized_request,omitempty"`
	Ambiguities       []string          `json:"ambiguities,omitempty"`
	Assumptions       []string          `json:"assumptions,omitempty"`
	NormalizedGoal    string            `json:"normalized_goal"`
	OperationMode     string            `json:"operation_mode"`
	ResourceHints     ResourceHints     `json:"resource_hints,omitempty"`
	DomainHints       []string          `json:"domain_hints,omitempty"`
	AmbiguityFlags    []string          `json:"ambiguity_flags,omitempty"`
	Narrative         string            `json:"narrative"`
}

type ResourceHints struct {
	ServiceName string `json:"service_name,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	HostName    string `json:"host_name,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

type NormalizedRequest struct {
	Intent         string           `json:"intent,omitempty"`
	Targets        []RequestTarget  `json:"targets,omitempty"`
	Symptoms       []RequestSymptom `json:"symptoms,omitempty"`
	Context        RequestContext   `json:"context,omitempty"`
	UserHypotheses []string         `json:"user_hypotheses,omitempty"`
	Priority       string           `json:"priority,omitempty"`
}

type RequestTarget struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type RequestSymptom struct {
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type RequestContext struct {
	TimeHint     string `json:"time_hint,omitempty"`
	TriggerEvent string `json:"trigger_event,omitempty"`
	Environment  string `json:"environment,omitempty"`
}

type Input struct {
	Message           string
	Scene             string
	CurrentPage       string
	SelectedResources []SelectedResource
}

type SelectedResource struct {
	Type string
	ID   string
	Name string
}

type Rewriter struct {
	runner *adk.Runner
}

func New(runner *adk.Runner) *Rewriter {
	return &Rewriter{runner: runner}
}

func (r *Rewriter) Rewrite(ctx context.Context, in Input) (Output, error) {
	out := heuristicRewrite(in)

	if r == nil || r.runner == nil {
		return out, nil
	}
	raw, err := runADKRewrite(ctx, r.runner, buildPromptInput(in))
	if err != nil {
		return out, nil
	}

	var parsed Output
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil {
		return out, nil
	}
	return normalizeOutput(in, parsed, out), nil
}

func heuristicRewrite(in Input) Output {
	message := strings.TrimSpace(in.Message)
	mode := detectMode(message)
	hints := detectResourceHints(message, in.SelectedResources)
	domains := detectDomains(message, in.Scene)
	ambiguity := make([]string, 0, 2)
	if !hasExplicitResourceTarget(message, hints, in.SelectedResources) {
		ambiguity = append(ambiguity, "resource_target_not_explicit")
	}

	goal := message
	if hints.ServiceName != "" && !strings.Contains(goal, hints.ServiceName) {
		goal = goal + "，目标服务：" + hints.ServiceName
	}
	normalizedRequest, assumptions := inferRequestMetadata(message, mode, hints)

	return Output{
		RawUserInput:      message,
		NormalizedRequest: normalizedRequest,
		Ambiguities:       append([]string(nil), ambiguity...),
		Assumptions:       assumptions,
		NormalizedGoal:    goal,
		OperationMode:     mode,
		ResourceHints:     hints,
		DomainHints:       domains,
		AmbiguityFlags:    ambiguity,
		Narrative:         buildNarrative(goal, mode, hints, domains, ambiguity),
	}
}

func buildPromptInput(in Input) string {
	var b strings.Builder
	b.WriteString("message: ")
	b.WriteString(strings.TrimSpace(in.Message))
	if strings.TrimSpace(in.Scene) != "" {
		b.WriteString("\nscene: ")
		b.WriteString(strings.TrimSpace(in.Scene))
	}
	if strings.TrimSpace(in.CurrentPage) != "" {
		b.WriteString("\ncurrent_page: ")
		b.WriteString(strings.TrimSpace(in.CurrentPage))
	}
	if len(in.SelectedResources) > 0 {
		b.WriteString("\nselected_resources:")
		for _, item := range in.SelectedResources {
			b.WriteString("\n- type=")
			b.WriteString(item.Type)
			if item.ID != "" {
				b.WriteString(", id=")
				b.WriteString(item.ID)
			}
			if item.Name != "" {
				b.WriteString(", name=")
				b.WriteString(item.Name)
			}
		}
	}
	return b.String()
}

func detectMode(message string) string {
	lower := strings.ToLower(message)
	switch {
	case containsAny(lower, "发布", "部署", "回滚", "重启", "删除", "apply", "rollback", "restart"):
		return "mutate"
	case containsAny(lower, "排查", "分析", "诊断", "看看", "查下", "check", "investigate"):
		return "investigate"
	default:
		return "query"
	}
}

func detectDomains(message, scene string) []string {
	lower := strings.ToLower(message + " " + scene)
	if mentionsAllHosts(lower) {
		return []string{"hostops"}
	}
	out := make([]string, 0, 3)
	appendIf := func(domain string, keywords ...string) {
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				out = append(out, domain)
				return
			}
		}
	}
	appendIf("service", "service", "服务", "api")
	appendIf("k8s", "k8s", "pod", "deployment", "cluster", "namespace")
	appendIf("hostops", "host", "主机", "机器", "节点")
	appendIf("delivery", "deploy", "发布", "rollout", "pipeline", "cicd")
	appendIf("observability", "监控", "latency", "error", "告警", "metrics", "日志")
	if len(out) == 0 {
		out = append(out, "service")
	}
	return dedupe(out)
}

func inferRequestMetadata(message, mode string, hints ResourceHints) (NormalizedRequest, []string) {
	lower := strings.ToLower(strings.TrimSpace(message))
	intent := "operations_request"
	switch {
	case mode == "mutate":
		if hints.ServiceName != "" {
			intent = "service_change_request"
		} else {
			intent = "infrastructure_change_request"
		}
	case containsAny(lower, "状态", "健康", "health", "status"):
		intent = "service_health_check"
	case containsAny(lower, "慢", "延迟", "latency", "性能"):
		intent = "performance_investigation"
	case containsAny(lower, "错误", "失败", "报错", "异常", "告警"):
		intent = "incident_diagnosis"
	}

	targets := make([]RequestTarget, 0, 2)
	switch {
	case mentionsAllHosts(lower):
		targets = append(targets, RequestTarget{Type: "host", Name: "all"})
	case hints.HostName != "":
		targets = append(targets, RequestTarget{Type: "host", Name: hints.HostName})
	}
	if hints.ServiceName != "" {
		targets = append(targets, RequestTarget{Type: "service", Name: hints.ServiceName})
	}
	if hints.ClusterName != "" {
		targets = append(targets, RequestTarget{Type: "cluster", Name: hints.ClusterName})
	}
	if hints.Namespace != "" {
		targets = append(targets, RequestTarget{Type: "namespace", Name: hints.Namespace})
	}

	ctx := RequestContext{}
	switch {
	case containsAny(lower, "刚刚", "刚才", "刚", "最近", "latest", "recent"):
		ctx.TimeHint = "recent"
	case containsAny(lower, "今天", "today"):
		ctx.TimeHint = "today"
	}
	if containsAny(lower, "发布", "部署", "上线", "rollout", "deploy") {
		ctx.TriggerEvent = "deployment_related"
	}
	if containsAny(lower, "生产", "prod", "线上") {
		ctx.Environment = "production"
	} else if containsAny(lower, "测试", "staging", "预发") {
		ctx.Environment = "staging"
	}
	hypotheses := make([]string, 0, 2)
	if containsAny(lower, "是不是", "是否", "会不会", "怀疑") {
		hypotheses = append(hypotheses, strings.TrimSpace(message))
	}
	priority := ""
	switch {
	case containsAny(lower, "紧急", "马上", "立刻", "urgent", "asap"):
		priority = "high"
	case containsAny(lower, "尽快", "优先", "priority"):
		priority = "medium"
	}
	assumptions := make([]string, 0, 2)
	if hints.ServiceName != "" {
		assumptions = append(assumptions, "service_target_inferred_from_request")
	}
	if hints.Namespace != "" {
		assumptions = append(assumptions, "namespace_inferred_from_request")
	}
	return NormalizedRequest{
		Intent:         intent,
		Targets:        targets,
		Context:        ctx,
		UserHypotheses: hypotheses,
		Priority:       priority,
	}, assumptions
}

func hasExplicitResourceTarget(message string, hints ResourceHints, resources []SelectedResource) bool {
	if len(resources) > 0 {
		return true
	}
	if hints != (ResourceHints{}) {
		return true
	}
	lower := strings.ToLower(strings.TrimSpace(message))
	return mentionsAllHosts(lower) ||
		containsAny(lower, "所有服务", "全部服务", "所有集群", "全部集群", "所有pod", "全部pod", "全部主机状态", "所有主机状态")
}

func mentionsAllHosts(lower string) bool {
	return containsAny(lower,
		"所有主机", "全部主机",
		"所有机器", "全部机器",
		"所有服务器", "全部服务器",
	)
}

func constrainDomains(parsed Output, heuristic []string) []string {
	for _, target := range parsed.NormalizedRequest.Targets {
		switch strings.TrimSpace(target.Type) {
		case "host":
			return []string{"hostops"}
		case "service":
			return []string{"service"}
		case "cluster", "namespace":
			return []string{"k8s"}
		}
	}
	if len(parsed.DomainHints) == 0 {
		return heuristic
	}
	return dedupe(parsed.DomainHints)
}

func constrainAmbiguity(parsed Output, resources []SelectedResource, fallback []string) []string {
	current := parsed.AmbiguityFlags
	if len(current) == 0 {
		current = fallback
	}
	if hasExplicitResourceTarget(parsed.RawUserInput, parsed.ResourceHints, resources) || len(parsed.NormalizedRequest.Targets) > 0 {
		out := make([]string, 0, len(current))
		for _, flag := range current {
			if strings.TrimSpace(flag) == "resource_target_not_explicit" {
				continue
			}
			out = append(out, flag)
		}
		return dedupe(out)
	}
	return dedupe(current)
}

func mergeNormalizedRequest(parsed, fallback NormalizedRequest) NormalizedRequest {
	if strings.TrimSpace(parsed.Intent) == "" {
		parsed.Intent = fallback.Intent
	}
	if len(parsed.Targets) == 0 {
		parsed.Targets = fallback.Targets
	}
	if len(parsed.Symptoms) == 0 {
		parsed.Symptoms = fallback.Symptoms
	}
	if parsed.Context == (RequestContext{}) {
		parsed.Context = fallback.Context
	}
	if len(parsed.UserHypotheses) == 0 {
		parsed.UserHypotheses = fallback.UserHypotheses
	}
	if strings.TrimSpace(parsed.Priority) == "" {
		parsed.Priority = fallback.Priority
	}
	return parsed
}

func normalizeOutput(in Input, parsed, fallback Output) Output {
	if strings.TrimSpace(parsed.NormalizedGoal) == "" {
		parsed.NormalizedGoal = fallback.NormalizedGoal
	}
	parsed.RawUserInput = firstNonEmpty(parsed.RawUserInput, strings.TrimSpace(in.Message))
	parsed.OperationMode = normalizeMode(parsed.OperationMode, fallback.OperationMode)
	if parsed.ResourceHints == (ResourceHints{}) {
		parsed.ResourceHints = fallback.ResourceHints
	}
	parsed.NormalizedRequest = mergeNormalizedRequest(parsed.NormalizedRequest, fallback.NormalizedRequest)
	parsed.Assumptions = dedupe(parsed.Assumptions)
	parsed.Ambiguities = dedupe(parsed.Ambiguities)
	parsed.DomainHints = constrainDomains(parsed, fallback.DomainHints)
	parsed.AmbiguityFlags = constrainAmbiguity(parsed, in.SelectedResources, fallback.AmbiguityFlags)
	parsed.Narrative = buildNarrative(parsed.NormalizedGoal, parsed.OperationMode, parsed.ResourceHints, parsed.DomainHints, parsed.AmbiguityFlags)
	return parsed
}

func detectResourceHints(message string, resources []SelectedResource) ResourceHints {
	hints := ResourceHints{}
	for _, item := range resources {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "service":
			hints.ServiceName = firstNonEmpty(item.Name, item.ID)
		case "cluster":
			hints.ClusterName = firstNonEmpty(item.Name, item.ID)
		case "host":
			hints.HostName = firstNonEmpty(item.Name, item.ID)
		case "namespace":
			hints.Namespace = firstNonEmpty(item.Name, item.ID)
		}
	}

	servicePattern := regexp.MustCompile(`([a-z0-9-]+(?:api|service|svc))`)
	if hints.ServiceName == "" {
		if match := servicePattern.FindString(strings.ToLower(message)); match != "" {
			hints.ServiceName = match
		}
	}
	namespacePattern := regexp.MustCompile(`namespace[:= ]([a-z0-9-]+)`)
	if hints.Namespace == "" {
		if groups := namespacePattern.FindStringSubmatch(strings.ToLower(message)); len(groups) > 1 {
			hints.Namespace = groups[1]
		}
	}
	return hints
}

func buildNarrative(goal, mode string, hints ResourceHints, domains, ambiguity []string) string {
	parts := []string{"用户请求已被规整为可执行任务。", "目标：" + goal + "。", "模式：" + mode + "。"}
	if hints.ServiceName != "" {
		parts = append(parts, "服务线索："+hints.ServiceName+"。")
	}
	if hints.ClusterName != "" {
		parts = append(parts, "集群线索："+hints.ClusterName+"。")
	}
	if len(domains) > 0 {
		parts = append(parts, "涉及领域："+strings.Join(domains, " / ")+"。")
	}
	if len(ambiguity) > 0 {
		parts = append(parts, "当前仍存在歧义："+strings.Join(ambiguity, ", ")+"。")
	}
	return strings.Join(parts, " ")
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func normalizeMode(value, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "query", "investigate", "mutate":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return fallback
	}
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
