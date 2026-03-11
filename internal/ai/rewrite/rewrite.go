// Package rewrite 实现 AI 编排的改写阶段。
//
// Rewriter 负责将用户口语化输入改写为结构化的任务表达。
// 输出包含归一化目标、操作模式、资源提示和检索信息。
package rewrite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cy77cc/OpsPilot/internal/ai/availability"
)

// Output 表示改写阶段的输出。
type Output struct {
	RawUserInput      string            `json:"raw_user_input,omitempty"`   // 原始用户输入
	NormalizedRequest NormalizedRequest `json:"normalized_request,omitempty"` // 归一化请求
	Ambiguities       []string          `json:"ambiguities,omitempty"`       // 歧义列表
	Assumptions       []string          `json:"assumptions,omitempty"`       // 假设列表
	NormalizedGoal    string            `json:"normalized_goal"`             // 归一化目标
	OperationMode     string            `json:"operation_mode"`              // 操作模式
	ResourceHints     ResourceHints     `json:"resource_hints,omitempty"`    // 资源提示
	DomainHints       []string          `json:"domain_hints,omitempty"`      // 领域提示
	AmbiguityFlags    []string          `json:"ambiguity_flags,omitempty"`   // 歧义标志
	RetrievalIntent   string            `json:"retrieval_intent,omitempty"`  // 检索意图
	RetrievalQueries  []string          `json:"retrieval_queries,omitempty"` // 检索查询
	RetrievalKeywords []string          `json:"retrieval_keywords,omitempty"` // 检索关键词
	KnowledgeScope    []string          `json:"knowledge_scope,omitempty"`   // 知识范围
	RequiresRAG       bool              `json:"requires_rag,omitempty"`      // 是否需要 RAG
	Narrative         string            `json:"narrative"`                   // 自然语言描述
}

// SemanticContract 定义改写阶段的语义契约。
type SemanticContract struct {
	RawUserInput      string            `json:"raw_user_input,omitempty"`
	NormalizedGoal    string            `json:"normalized_goal,omitempty"`
	OperationMode     string            `json:"operation_mode,omitempty"`
	NormalizedRequest NormalizedRequest `json:"normalized_request,omitempty"`
	ResourceHints     ResourceHints     `json:"resource_hints,omitempty"`
	DomainHints       []string          `json:"domain_hints,omitempty"`
	Ambiguities       []string          `json:"ambiguities,omitempty"`
	AmbiguityFlags    []string          `json:"ambiguity_flags,omitempty"`
	RetrievalIntent   string            `json:"retrieval_intent,omitempty"`
	RetrievalQueries  []string          `json:"retrieval_queries,omitempty"`
	RetrievalKeywords []string          `json:"retrieval_keywords,omitempty"`
	KnowledgeScope    []string          `json:"knowledge_scope,omitempty"`
	RequiresRAG       bool              `json:"requires_rag,omitempty"`
}

// ModelUnavailableError 表示模型不可用错误。
type ModelUnavailableError struct {
	Code              string // 错误码
	UserVisibleReason string // 用户可见原因
	Cause             error  // 原始错误
}

func (e *ModelUnavailableError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", strings.TrimSpace(e.Code), e.Cause)
	}
	if strings.TrimSpace(e.Code) != "" {
		return strings.TrimSpace(e.Code)
	}
	return "rewrite unavailable"
}

func (e *ModelUnavailableError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *ModelUnavailableError) UserVisibleMessage() string {
	if e == nil {
		return availability.UnavailableMessage(availability.LayerRewrite)
	}
	return firstNonEmpty(e.UserVisibleReason, availability.UnavailableMessage(availability.LayerRewrite))
}

type ResourceHints struct {
	ServiceName string `json:"service_name,omitempty"`
	ServiceID   int    `json:"service_id,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	ClusterID   int    `json:"cluster_id,omitempty"`
	HostName    string `json:"host_name,omitempty"`
	HostID      int    `json:"host_id,omitempty"`
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
	runFn  func(context.Context, Input, func(string)) (Output, error)
}

func New(runner *adk.Runner) *Rewriter {
	return &Rewriter{runner: runner}
}

func NewWithFunc(runFn func(context.Context, Input, func(string)) (Output, error)) *Rewriter {
	return &Rewriter{runFn: runFn}
}

func (r *Rewriter) Rewrite(ctx context.Context, in Input) (Output, error) {
	return r.rewrite(ctx, in, nil)
}

func (r *Rewriter) RewriteStream(ctx context.Context, in Input, onDelta func(string)) (Output, error) {
	return r.rewrite(ctx, in, onDelta)
}

func (r *Rewriter) rewrite(ctx context.Context, in Input, onDelta func(string)) (Output, error) {
	if r != nil && r.runFn != nil {
		return r.runFn(ctx, in, onDelta)
	}
	base := buildBaseOutput(in)

	if r == nil || r.runner == nil {
		return Output{}, &ModelUnavailableError{
			Code:              "rewrite_runner_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerRewrite),
		}
	}
	raw, err := runADKRewrite(ctx, r.runner, buildPromptInput(in), onDelta)
	if err != nil {
		return Output{}, &ModelUnavailableError{
			Code:              "rewrite_model_unavailable",
			UserVisibleReason: availability.UnavailableMessage(availability.LayerRewrite),
			Cause:             err,
		}
	}
	return parseModelOutput(base, raw)
}

func buildBaseOutput(in Input) Output {
	message := strings.TrimSpace(in.Message)
	hints := detectResourceHints(in.SelectedResources)
	return Output{
		RawUserInput:  message,
		ResourceHints: hints,
		NormalizedRequest: NormalizedRequest{
			Targets: buildTargets(in.SelectedResources),
		},
	}
}

func (out Output) SemanticContract() SemanticContract {
	return SemanticContract{
		RawUserInput:      strings.TrimSpace(out.RawUserInput),
		NormalizedGoal:    strings.TrimSpace(out.NormalizedGoal),
		OperationMode:     strings.TrimSpace(out.OperationMode),
		NormalizedRequest: out.NormalizedRequest,
		ResourceHints:     out.ResourceHints,
		DomainHints:       dedupe(out.DomainHints),
		Ambiguities:       dedupe(out.Ambiguities),
		AmbiguityFlags:    dedupe(out.AmbiguityFlags),
		RetrievalIntent:   strings.TrimSpace(out.RetrievalIntent),
		RetrievalQueries:  dedupe(out.RetrievalQueries),
		RetrievalKeywords: dedupe(out.RetrievalKeywords),
		KnowledgeScope:    dedupe(out.KnowledgeScope),
		RequiresRAG:       out.RequiresRAG,
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

func parseModelOutput(base Output, raw string) (Output, error) {
	var parsed Output
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &parsed); err != nil {
		return Output{}, &ModelUnavailableError{
			Code:              "rewrite_invalid_json",
			UserVisibleReason: availability.InvalidOutputMessage(availability.LayerRewrite),
			Cause:             err,
		}
	}
	return normalizeOutput(base, parsed), nil
}

func mergeNormalizedRequest(parsed, base NormalizedRequest) NormalizedRequest {
	if strings.TrimSpace(parsed.Intent) == "" {
		parsed.Intent = base.Intent
	}
	if len(parsed.Targets) == 0 {
		parsed.Targets = base.Targets
	}
	if len(parsed.Symptoms) == 0 {
		parsed.Symptoms = base.Symptoms
	}
	if parsed.Context == (RequestContext{}) {
		parsed.Context = base.Context
	}
	if len(parsed.UserHypotheses) == 0 {
		parsed.UserHypotheses = base.UserHypotheses
	}
	if strings.TrimSpace(parsed.Priority) == "" {
		parsed.Priority = base.Priority
	}
	return parsed
}

func normalizeOutput(base, parsed Output) Output {
	parsed.RawUserInput = firstNonEmpty(parsed.RawUserInput, base.RawUserInput)
	parsed.OperationMode = normalizeMode(parsed.OperationMode, "")
	if parsed.ResourceHints == (ResourceHints{}) {
		parsed.ResourceHints = base.ResourceHints
	}
	parsed.NormalizedRequest = mergeNormalizedRequest(parsed.NormalizedRequest, base.NormalizedRequest)
	parsed.Assumptions = dedupe(parsed.Assumptions)
	parsed.Ambiguities = dedupe(parsed.Ambiguities)
	parsed.DomainHints = dedupe(parsed.DomainHints)
	parsed.AmbiguityFlags = dedupe(parsed.AmbiguityFlags)
	parsed.RetrievalIntent = firstNonEmpty(parsed.RetrievalIntent)
	parsed.RetrievalQueries = dedupe(parsed.RetrievalQueries)
	parsed.RetrievalKeywords = dedupe(parsed.RetrievalKeywords)
	parsed.KnowledgeScope = dedupe(parsed.KnowledgeScope)
	parsed.Narrative = strings.TrimSpace(parsed.Narrative)
	return parsed
}

func buildTargets(resources []SelectedResource) []RequestTarget {
	targets := make([]RequestTarget, 0, len(resources))
	for _, item := range resources {
		targetType := strings.ToLower(strings.TrimSpace(item.Type))
		targetName := firstNonEmpty(item.Name, item.ID)
		if targetType == "" || targetName == "" {
			continue
		}
		targets = append(targets, RequestTarget{Type: targetType, Name: targetName})
	}
	return targets
}

func detectResourceHints(resources []SelectedResource) ResourceHints {
	hints := ResourceHints{}
	for _, item := range resources {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "service":
			hints.ServiceName = firstNonEmpty(item.Name, item.ID)
			hints.ServiceID = parseResourceID(item.ID)
		case "cluster":
			hints.ClusterName = firstNonEmpty(item.Name, item.ID)
			hints.ClusterID = parseResourceID(item.ID)
		case "host":
			hints.HostName = firstNonEmpty(item.Name, item.ID)
			hints.HostID = parseResourceID(item.ID)
		case "namespace":
			hints.Namespace = firstNonEmpty(item.Name, item.ID)
		}
	}
	return hints
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

func parseResourceID(raw string) int {
	for _, r := range strings.TrimSpace(raw) {
		if r < '0' || r > '9' {
			return 0
		}
	}
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	value := 0
	for _, r := range raw {
		value = value*10 + int(r-'0')
	}
	return value
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
