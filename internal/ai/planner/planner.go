// Package planner 实现 AI 编排的规划阶段。
//
// Planner 负责解析用户意图，解析资源引用，并生成执行计划。
// 输出四种决策类型: clarify (需要澄清)、reject (拒绝)、direct_reply (直接回复)、plan (生成计划)。
package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/google/uuid"

	"github.com/cy77cc/OpsPilot/internal/ai/availability"
	"github.com/cy77cc/OpsPilot/internal/ai/rewrite"
)

// Planner 是规划器核心，负责生成执行计划。
type Planner struct {
	runner *adk.Runner                                              // ADK 运行器
	runFn  func(context.Context, Input, func(string)) (Decision, error) // 执行函数
}

// Input 是规划器的输入结构。
type Input struct {
	Message string        // 用户原始消息
	Rewrite rewrite.Output // Rewrite 阶段的输出
}

// DecisionType 定义决策类型。
type DecisionType string

const (
	DecisionClarify     DecisionType = "clarify"      // 需要用户澄清
	DecisionReject      DecisionType = "reject"       // 拒绝执行
	DecisionDirectReply DecisionType = "direct_reply" // 直接回复
	DecisionPlan        DecisionType = "plan"         // 生成执行计划
)

// Decision 表示规划器的决策输出。
type Decision struct {
	Type       DecisionType     `json:"type"`                // 决策类型
	Message    string           `json:"message,omitempty"`   // 消息内容
	Reason     string           `json:"reason,omitempty"`    // 原因说明
	Candidates []map[string]any `json:"candidates,omitempty"` // 候选项 (澄清时使用)
	Plan       *ExecutionPlan   `json:"plan,omitempty"`      // 执行计划
	Narrative  string           `json:"narrative"`           // 自然语言描述
}

// PlanningError 表示规划错误。
type PlanningError struct {
	Code              string // 错误码
	UserVisibleReason string // 用户可见原因
	Cause             error  // 原始错误
}

// Error 实现 error 接口。
func (e *PlanningError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", strings.TrimSpace(e.Code), e.Cause)
	}
	return firstNonEmpty(e.Code, "planning_unavailable")
}

// Unwrap 返回原始错误。
func (e *PlanningError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// UserVisibleMessage 返回用户可见的错误消息。
func (e *PlanningError) UserVisibleMessage() string {
	if e == nil {
		return availability.UnavailableMessage(availability.LayerPlanner)
	}
	return firstNonEmpty(e.UserVisibleReason, availability.UnavailableMessage(availability.LayerPlanner))
}

// ExecutionPlan 表示执行计划。
type ExecutionPlan struct {
	PlanID    string            `json:"plan_id"`       // 计划唯一标识
	Goal      string            `json:"goal"`          // 执行目标
	Resolved  ResolvedResources `json:"resolved"`      // 已解析的资源
	Narrative string            `json:"narrative"`     // 自然语言描述
	Steps     []PlanStep        `json:"steps"`         // 执行步骤列表
}

// ResourceRef 表示资源引用。
type ResourceRef struct {
	ID   int    `json:"id,omitempty"`   // 资源 ID
	Name string `json:"name,omitempty"` // 资源名称
}

// PodRef 表示 Pod 引用。
type PodRef struct {
	Name      string `json:"name,omitempty"`      // Pod 名称
	Namespace string `json:"namespace,omitempty"` // 命名空间
	ClusterID int    `json:"cluster_id,omitempty"` // 集群 ID
}

// ResourceScope 表示资源范围。
type ResourceScope struct {
	Kind         string         `json:"kind,omitempty"`         // 范围类型 (all/filtered/single)
	ResourceType string         `json:"resource_type,omitempty"` // 资源类型
	Selector     map[string]any `json:"selector,omitempty"`     // 选择器
}

// ResolvedResources 表示已解析的资源引用。
type ResolvedResources struct {
	ServiceName string         `json:"service_name,omitempty"` // 服务名称
	ServiceID   int            `json:"service_id,omitempty"`   // 服务 ID
	ClusterName string         `json:"cluster_name,omitempty"` // 集群名称
	ClusterID   int            `json:"cluster_id,omitempty"`   // 集群 ID
	HostNames   []string       `json:"host_names,omitempty"`   // 主机名称列表
	HostIDs     []int          `json:"host_ids,omitempty"`     // 主机 ID 列表
	Namespace   string         `json:"namespace,omitempty"`     // 命名空间
	PodName     string         `json:"pod_name,omitempty"`      // Pod 名称
	Services    []ResourceRef  `json:"services,omitempty"`      // 服务引用列表
	Clusters    []ResourceRef  `json:"clusters,omitempty"`      // 集群引用列表
	Hosts       []ResourceRef  `json:"hosts,omitempty"`         // 主机引用列表
	Pods        []PodRef       `json:"pods,omitempty"`          // Pod 引用列表
	Scope       *ResourceScope `json:"scope,omitempty"`         // 资源范围
}

// PlanStep 表示执行计划中的单个步骤。
type PlanStep struct {
	StepID    string         `json:"step_id"`              // 步骤唯一标识
	Title     string         `json:"title"`                // 步骤标题
	Expert    string         `json:"expert"`               // 专家名称
	Intent    string         `json:"intent"`               // 意图
	Task      string         `json:"task"`                 // 任务描述
	Input     map[string]any `json:"input,omitempty"`      // 输入参数
	DependsOn []string       `json:"depends_on,omitempty"` // 依赖的步骤 ID
	Mode      string         `json:"mode"`                 // 操作模式 (readonly/mutating)
	Risk      string         `json:"risk"`                 // 风险等级 (low/medium/high)
	Narrative string         `json:"narrative,omitempty"`  // 自然语言描述
}

// New 创建新的规划器实例。
func New(runner *adk.Runner) *Planner {
	return &Planner{runner: runner}
}

// NewWithFunc 使用自定义执行函数创建规划器。
func NewWithFunc(runFn func(context.Context, Input, func(string)) (Decision, error)) *Planner {
	return &Planner{runFn: runFn}
}

// Plan 执行规划，返回决策结果。
func (p *Planner) Plan(ctx context.Context, in Input) (Decision, error) {
	return p.plan(ctx, in, nil)
}

// PlanStream 执行规划并支持流式输出。
func (p *Planner) PlanStream(ctx context.Context, in Input, onDelta func(string)) (Decision, error) {
	return p.plan(ctx, in, onDelta)
}

// plan 执行规划的核心逻辑。
func (p *Planner) plan(ctx context.Context, in Input, onDelta func(string)) (Decision, error) {
	if p != nil && p.runFn != nil {
		return p.runFn(ctx, in, onDelta)
	}
	if p == nil || p.runner == nil {
		return Decision{}, &PlanningError{
			Code:              "planner_runner_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerPlanner),
		}
	}
	raw, err := runADKPlanner(ctx, p.runner, buildPromptInput(in), onDelta)
	if err != nil {
		return Decision{}, &PlanningError{
			Code:              "planner_model_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerPlanner),
			Cause:             err,
		}
	}

	parsed, err := ParseDecision(strings.TrimSpace(raw))
	if err != nil {
		return Decision{}, &PlanningError{
			Code:              "planner_invalid_json",
			UserVisibleReason: availability.InvalidOutputMessage(availability.LayerPlanner),
			Cause:             err,
		}
	}
	return normalizeDecision(buildBasePlanContext(in), parsed)
}

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

// buildBasePlanContext 从输入构建基础计划上下文。
func buildBasePlanContext(in Input) *ExecutionPlan {
	rewritten := in.Rewrite
	goal := firstNonEmpty(rewritten.NormalizedGoal, strings.TrimSpace(in.Message))
	resolved := ResolvedResources{
		ServiceName: rewritten.ResourceHints.ServiceName,
		ServiceID:   rewritten.ResourceHints.ServiceID,
		ClusterName: rewritten.ResourceHints.ClusterName,
		ClusterID:   rewritten.ResourceHints.ClusterID,
		Namespace:   rewritten.ResourceHints.Namespace,
		PodName:     collectPodName(rewritten),
		HostNames:   collectHostNames(rewritten),
		HostIDs:     collectHostIDs(rewritten),
		Services:    collectServices(rewritten),
		Clusters:    collectClusters(rewritten),
		Hosts:       collectHosts(rewritten),
		Pods:        collectPods(rewritten),
		Scope:       detectScope(rewritten),
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
	services := parseResourceRefs(raw["services"])
	if len(services) == 0 && serviceID > 0 {
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
		ServiceName: serviceName,
		ServiceID:   serviceID,
		ClusterName: clusterName,
		ClusterID:   clusterID,
		Namespace:   namespace,
		HostNames:   hostNames,
		HostIDs:     hostIDs,
		PodName:     podName,
		Services:    services,
		Clusters:    clusters,
		Hosts:       hostRefs,
		Pods:        pods,
		Scope:       scope,
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
		if len(resolved.Pods) > 1 && len(podRefsFromInput(input["pods"])) == 0 {
			input["pods"] = podRefsToAny(resolved.Pods)
		}
		if resolved.Scope != nil && input["scope"] == nil {
			input["scope"] = scopeToMap(resolved.Scope)
		}
	case "service":
		if resolved.ServiceID > 0 && looseIntValue(input["service_id"]) == 0 {
			input["service_id"] = resolved.ServiceID
		}
		if resolved.ClusterID > 0 && looseIntValue(input["cluster_id"]) == 0 {
			input["cluster_id"] = resolved.ClusterID
		}
		if len(resolved.Services) > 1 && len(intSliceValue(input["service_ids"])) == 0 {
			input["service_ids"] = resourceIDs(resolved.Services)
		}
		if resolved.Scope != nil && input["scope"] == nil {
			input["scope"] = scopeToMap(resolved.Scope)
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
		if resolved.Scope != nil && input["scope"] == nil {
			input["scope"] = scopeToMap(resolved.Scope)
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
	if len(resolved.HostIDs) > 0 || len(resolved.HostNames) > 0 || len(resolved.Hosts) > 0 {
		return true
	}
	return hasTargetType(step, "host")
}

func resolvedClusterID(step PlanStep, resolved ResolvedResources) int {
	if clusterID := looseIntValue(step.Input["cluster_id"]); clusterID > 0 {
		return clusterID
	}
	return resolved.ClusterID
}

func resolvedServiceID(step PlanStep, resolved ResolvedResources) int {
	if serviceID := looseIntValue(step.Input["service_id"]); serviceID > 0 {
		return serviceID
	}
	if ids := intSliceValue(step.Input["service_ids"]); len(ids) == 1 {
		return ids[0]
	}
	if len(resolved.Services) == 1 && resolved.Services[0].ID > 0 {
		return resolved.Services[0].ID
	}
	return resolved.ServiceID
}

func resolvedPodName(step PlanStep, resolved ResolvedResources) string {
	if pod := strings.TrimSpace(looseStringValue(step.Input["pod"])); pod != "" {
		return pod
	}
	if pod := strings.TrimSpace(resolved.PodName); pod != "" {
		return pod
	}
	if len(resolved.Pods) == 1 && strings.TrimSpace(resolved.Pods[0].Name) != "" {
		return strings.TrimSpace(resolved.Pods[0].Name)
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
	if len(resolved.HostIDs) > 0 {
		hostIDs := make([]int, len(resolved.HostIDs))
		copy(hostIDs, resolved.HostIDs)
		return hostIDs
	}
	if len(resolved.Hosts) > 0 {
		return resourceIDs(resolved.Hosts)
	}
	return nil
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

func isAllKeyword(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "all", "*", "全部", "所有":
		return true
	default:
		return false
	}
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
