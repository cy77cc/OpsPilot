package runtime

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type ApprovalToolSpec struct {
	Name        string
	DisplayName string
	Description string
	Mode        string
	Risk        string
	Category    string
}

type ApprovalCheckRequest struct {
	ToolName       string
	Mode           string
	Risk           string
	Scene          string
	Environment    string
	Namespace      string
	Params         map[string]any
	RuntimeContext RuntimeContext
}

type ApprovalDecision struct {
	NeedApproval    bool
	Reason          string
	Environment     string
	PolicySource    string
	SummaryTemplate string
	Tool            ApprovalToolSpec
}

type ApprovalInterruptInfo struct {
	PlanID          string         `json:"plan_id,omitempty"`
	StepID          string         `json:"step_id,omitempty"`
	ToolName        string         `json:"tool_name,omitempty"`
	ToolDisplayName string         `json:"tool_display_name,omitempty"`
	Mode            string         `json:"mode,omitempty"`
	RiskLevel       string         `json:"risk_level,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Params          map[string]any `json:"params,omitempty"`
	Environment     string         `json:"environment,omitempty"`
	Namespace       string         `json:"namespace,omitempty"`
}

type ApprovalDecisionMaker struct {
	resolveScene func(string) ResolvedScene
	lookupTool   func(string) (ApprovalToolSpec, bool)
}

type ApprovalDecisionMakerOptions struct {
	ResolveScene func(string) ResolvedScene
	LookupTool   func(string) (ApprovalToolSpec, bool)
}

func NewApprovalDecisionMaker(opts ApprovalDecisionMakerOptions) *ApprovalDecisionMaker {
	return &ApprovalDecisionMaker{
		resolveScene: opts.ResolveScene,
		lookupTool:   opts.LookupTool,
	}
}

func (m *ApprovalDecisionMaker) Decide(_ context.Context, req ApprovalCheckRequest) (ApprovalDecision, error) {
	toolSpec, ok := m.lookup(req)
	if !ok {
		return ApprovalDecision{}, fmt.Errorf("approval tool metadata not found for %q", strings.TrimSpace(req.ToolName))
	}
	if strings.TrimSpace(req.Mode) == "" {
		req.Mode = toolSpec.Mode
	}
	if strings.TrimSpace(req.Risk) == "" {
		req.Risk = toolSpec.Risk
	}
	if strings.EqualFold(toolSpec.Mode, "readonly") {
		return ApprovalDecision{
			NeedApproval: false,
			Reason:       "readonly tool bypasses approval",
			Tool:         toolSpec,
		}, nil
	}

	scene := m.resolve(req.Scene)
	override := ToolApprovalOverride{}
	if scene.SceneConfig.ApprovalConfig != nil {
		override = scene.SceneConfig.ApprovalConfig.ToolOverrides[toolSpec.Name]
	}
	if override.ForceApproval {
		return ApprovalDecision{
			NeedApproval:    true,
			Reason:          "tool override forces approval",
			PolicySource:    "tool_override",
			Environment:     inferEnvironment(req),
			SummaryTemplate: strings.TrimSpace(override.SummaryTemplate),
			Tool:            toolSpec,
		}, nil
	}
	if override.SkipApproval {
		return ApprovalDecision{
			NeedApproval:    false,
			Reason:          "tool override skips approval",
			PolicySource:    "tool_override",
			Environment:     inferEnvironment(req),
			SummaryTemplate: strings.TrimSpace(override.SummaryTemplate),
			Tool:            toolSpec,
		}, nil
	}

	policy, source := m.applyPolicy(scene, req)
	environment := inferEnvironment(req)
	if m.matchSkipCondition(policy.SkipConditions, req, environment) {
		return ApprovalDecision{
			NeedApproval:    false,
			Reason:          "approval skipped by policy condition",
			PolicySource:    source,
			Environment:     environment,
			SummaryTemplate: strings.TrimSpace(override.SummaryTemplate),
			Tool:            toolSpec,
		}, nil
	}
	if policy.RequireForAllMutating {
		return ApprovalDecision{
			NeedApproval:    true,
			Reason:          "policy requires approval for all mutating tools",
			PolicySource:    source,
			Environment:     environment,
			SummaryTemplate: strings.TrimSpace(override.SummaryTemplate),
			Tool:            toolSpec,
		}, nil
	}
	if containsFold(policy.RequireApprovalFor, toolSpec.Risk) {
		return ApprovalDecision{
			NeedApproval:    true,
			Reason:          "tool risk matches approval policy",
			PolicySource:    source,
			Environment:     environment,
			SummaryTemplate: strings.TrimSpace(override.SummaryTemplate),
			Tool:            toolSpec,
		}, nil
	}
	return ApprovalDecision{
		NeedApproval:    false,
		Reason:          "policy does not require approval",
		PolicySource:    source,
		Environment:     environment,
		SummaryTemplate: strings.TrimSpace(override.SummaryTemplate),
		Tool:            toolSpec,
	}, nil
}

func (m *ApprovalDecisionMaker) applyPolicy(scene ResolvedScene, req ApprovalCheckRequest) (ApprovalPolicy, string) {
	cfg := scene.SceneConfig.ApprovalConfig
	if cfg == nil {
		return defaultApprovalPolicyForTool(req), "implicit_default"
	}
	policy := cfg.DefaultPolicy
	source := "scene_default"
	environment := inferEnvironment(req)
	if environment != "" {
		if envPolicy, ok := cfg.EnvironmentPolicies[environment]; ok {
			policy = mergeApprovalPolicy(policy, envPolicy)
			source = "environment:" + environment
		}
	}
	return policy, source
}

func (m *ApprovalDecisionMaker) matchSkipCondition(conditions []SkipCondition, req ApprovalCheckRequest, environment string) bool {
	for _, condition := range conditions {
		switch strings.TrimSpace(strings.ToLower(condition.Type)) {
		case "environment":
			if wildcardMatch(environment, condition.Pattern) {
				return true
			}
		case "namespace":
			if wildcardMatch(firstNonBlank(req.Namespace, namespaceFromRequest(req)), condition.Pattern) {
				return true
			}
		case "tool", "tool_name":
			if wildcardMatch(strings.TrimSpace(req.ToolName), condition.Pattern) {
				return true
			}
		case "scene":
			if wildcardMatch(strings.TrimSpace(req.Scene), condition.Pattern) {
				return true
			}
		}
	}
	return false
}

func (m *ApprovalDecisionMaker) resolve(scene string) ResolvedScene {
	if m == nil || m.resolveScene == nil {
		return defaultSceneConfigResolver().Resolve(scene)
	}
	return m.resolveScene(scene)
}

func (m *ApprovalDecisionMaker) lookup(req ApprovalCheckRequest) (ApprovalToolSpec, bool) {
	if m != nil && m.lookupTool != nil {
		if spec, ok := m.lookupTool(strings.TrimSpace(req.ToolName)); ok {
			return normalizeToolSpec(spec, req), true
		}
	}
	if strings.TrimSpace(req.ToolName) == "" {
		return ApprovalToolSpec{}, false
	}
	return normalizeToolSpec(ApprovalToolSpec{Name: req.ToolName}, req), true
}

func normalizeToolSpec(spec ApprovalToolSpec, req ApprovalCheckRequest) ApprovalToolSpec {
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = strings.TrimSpace(req.ToolName)
	}
	if strings.TrimSpace(spec.Mode) == "" {
		spec.Mode = strings.TrimSpace(req.Mode)
	}
	if strings.TrimSpace(spec.Risk) == "" {
		spec.Risk = strings.TrimSpace(req.Risk)
	}
	return spec
}

func mergeApprovalPolicy(base, override ApprovalPolicy) ApprovalPolicy {
	out := base
	if override.RequireForAllMutating {
		out.RequireForAllMutating = true
	}
	if len(override.RequireApprovalFor) > 0 {
		out.RequireApprovalFor = dedupeStrings(append([]string(nil), override.RequireApprovalFor...))
	}
	if len(override.SkipConditions) > 0 {
		out.SkipConditions = append(append([]SkipCondition(nil), base.SkipConditions...), override.SkipConditions...)
	}
	return out
}

func defaultApprovalPolicyForTool(req ApprovalCheckRequest) ApprovalPolicy {
	policy := ApprovalPolicy{}
	switch strings.TrimSpace(strings.ToLower(req.Risk)) {
	case "high":
		policy.RequireApprovalFor = []string{"high"}
	case "medium":
		policy.RequireApprovalFor = []string{"medium", "high"}
	case "low":
		policy.RequireApprovalFor = nil
	default:
		policy.RequireForAllMutating = strings.EqualFold(strings.TrimSpace(req.Mode), "mutating")
	}
	return policy
}

func inferEnvironment(req ApprovalCheckRequest) string {
	candidates := []string{
		strings.TrimSpace(req.Environment),
		stringValue(req.Params["environment"]),
		stringValue(req.Params["env"]),
		stringValue(req.RuntimeContext.Metadata["environment"]),
		stringValue(req.RuntimeContext.Metadata["env"]),
		stringValue(req.RuntimeContext.UserContext["environment"]),
		stringValue(req.RuntimeContext.UserContext["env"]),
		environmentFromNamespace(firstNonBlank(req.Namespace, namespaceFromRequest(req))),
		environmentFromProject(firstNonBlank(req.RuntimeContext.ProjectName, req.RuntimeContext.ProjectID)),
	}
	for _, candidate := range candidates {
		if normalized := normalizeEnvironment(candidate); normalized != "" {
			return normalized
		}
	}
	return ""
}

func namespaceFromRequest(req ApprovalCheckRequest) string {
	if namespace := stringValue(req.Params["namespace"]); namespace != "" {
		return namespace
	}
	for _, resource := range req.RuntimeContext.SelectedResources {
		if strings.TrimSpace(resource.Namespace) != "" {
			return strings.TrimSpace(resource.Namespace)
		}
	}
	return ""
}

func environmentFromNamespace(namespace string) string {
	namespace = strings.TrimSpace(strings.ToLower(namespace))
	switch {
	case namespace == "":
		return ""
	case namespace == "prod", namespace == "production", strings.Contains(namespace, "prod"):
		return "production"
	case namespace == "stage", namespace == "staging", strings.Contains(namespace, "staging"):
		return "staging"
	case namespace == "dev", namespace == "development", strings.Contains(namespace, "dev"):
		return "dev"
	case strings.Contains(namespace, "test"):
		return "test"
	default:
		return ""
	}
}

func environmentFromProject(project string) string {
	project = strings.TrimSpace(strings.ToLower(project))
	switch {
	case strings.Contains(project, "prod"):
		return "production"
	case strings.Contains(project, "stag"):
		return "staging"
	case strings.Contains(project, "dev"):
		return "dev"
	default:
		return ""
	}
}

func normalizeEnvironment(environment string) string {
	switch strings.TrimSpace(strings.ToLower(environment)) {
	case "":
		return ""
	case "prod", "production":
		return "production"
	case "stage", "staging":
		return "staging"
	case "dev", "development":
		return "dev"
	case "test", "testing":
		return "test"
	default:
		return strings.TrimSpace(strings.ToLower(environment))
	}
}

func wildcardMatch(value, pattern string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	pattern = strings.TrimSpace(strings.ToLower(pattern))
	if value == "" || pattern == "" {
		return false
	}
	if !strings.ContainsAny(pattern, "*?[") {
		return value == pattern
	}
	matched, err := filepath.Match(pattern, value)
	return err == nil && matched
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		normalized := strings.TrimSpace(strings.ToLower(item))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringValue(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		return ""
	}
}
