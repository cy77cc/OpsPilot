package cluster

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
)

var criticalPolicyNamespaces = map[string]struct{}{
	"default":     {},
	"kube-system": {},
}

// PolicySimulationInput carries the minimal inputs needed for phase 2 policy simulation.
type PolicySimulationInput struct {
	Base               *PolicyDefinition
	Candidate          *PolicyDefinition
	ExistingPolicies   []PolicyDefinition
	NamespacePodCounts map[string]int
	CNICapabilityGap   bool
	RollbackAvailable  bool
}

// PolicySimulationResult combines simulation findings with release risk metadata.
type PolicySimulationResult struct {
	Passed bool `json:"passed" yaml:"passed"`
	PolicySimulationStatus
	PolicyReleaseStatus
}

type policySimulationResultOutput struct {
	Passed         bool                      `json:"passed" yaml:"passed"`
	BlockingIssues []PolicyIssue             `json:"blocking_issues,omitempty" yaml:"blocking_issues,omitempty"`
	Warnings       []PolicyWarning           `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	ImpactSummary  policyImpactSummaryOutput `json:"impact_summary,omitempty" yaml:"impact_summary,omitempty"`
	RiskScore      int                       `json:"risk_score" yaml:"risk_score"`
	RiskLevel      PolicyRiskLevel           `json:"risk_level" yaml:"risk_level"`
}

type policyImpactSummaryOutput struct {
	AffectedPods       int      `json:"affected_pods" yaml:"affected_pods"`
	AffectedNamespaces []string `json:"affected_namespaces" yaml:"affected_namespaces"`
	NewDeniedFlows     []string `json:"new_denied_flows" yaml:"new_denied_flows"`
}

type policyRiskFactors struct {
	BlocksCriticalNamespace bool
	HTTPRuleCount           int
	CNICapabilityGap        bool
	RollbackAvailable       bool
}

// MarshalJSON keeps the simulation output contract aligned with the spec's snake_case field names.
func (r PolicySimulationResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.outputContract())
}

// MarshalYAML keeps the simulation output contract aligned with the spec's snake_case field names.
func (r PolicySimulationResult) MarshalYAML() (any, error) {
	return r.outputContract(), nil
}

// SimulatePolicy evaluates a candidate policy and returns simulation findings plus risk.
func SimulatePolicy(input PolicySimulationInput) PolicySimulationResult {
	startedAt := time.Now()
	defer ObserveSimulationEvaluationDurationSince(startedAt)

	candidate := normalizedPolicyDefinition(input.Candidate)
	recordPolicyRuleMetrics(candidate)
	impact := buildImpactSummary(candidate, input.NamespacePodCounts)
	blockingIssues := detectBlockingIssues(input, candidate, impact.AffectedNamespaces)
	warnings := detectWarnings(candidate, impact)

	riskScore, riskLevel := calculatePolicyRiskScore(policyRiskFactors{
		BlocksCriticalNamespace: hasIssueCode(blockingIssues, PolicyErrorCriticalNamespaceBlocked),
		HTTPRuleCount:           countHTTPRules(candidate),
		CNICapabilityGap:        input.CNICapabilityGap,
		RollbackAvailable:       input.RollbackAvailable,
	})

	return PolicySimulationResult{
		Passed: simulationPassed(blockingIssues, riskLevel),
		PolicySimulationStatus: PolicySimulationStatus{
			BlockingIssues: blockingIssues,
			Warnings:       warnings,
			ImpactSummary:  impact,
		},
		PolicyReleaseStatus: PolicyReleaseStatus{
			RiskScore: riskScore,
			RiskLevel: riskLevel,
		},
	}
}

func recordPolicyRuleMetrics(candidate PolicyDefinition) {
	policyName := strings.TrimSpace(candidate.Metadata.Name)
	namespace := strings.TrimSpace(candidate.Metadata.Namespace)

	for _, rule := range candidate.Spec.Ingress {
		action := strings.TrimSpace(string(rule.Action))
		if action == "" {
			action = string(PolicyRuleActionAllow)
		}
		RecordPolicyHit(policyName, action, "ingress", namespace)
		if rule.Action == PolicyRuleActionDeny {
			RecordPolicyDeny(policyName, namespace)
		}
	}

	for _, rule := range candidate.Spec.Egress {
		action := strings.TrimSpace(string(rule.Action))
		if action == "" {
			action = string(PolicyRuleActionAllow)
		}
		RecordPolicyHit(policyName, action, "egress", namespace)
		if rule.Action == PolicyRuleActionDeny {
			RecordPolicyDeny(policyName, namespace)
		}
	}
}

func calculatePolicyRiskScore(factors policyRiskFactors) (int, PolicyRiskLevel) {
	score := 0

	if factors.BlocksCriticalNamespace {
		score += 40
	}
	if factors.HTTPRuleCount > 0 {
		httpRisk := factors.HTTPRuleCount * 2
		if httpRisk > 20 {
			httpRisk = 20
		}
		score += httpRisk
	}
	if factors.CNICapabilityGap {
		score += 30
	}
	if !factors.RollbackAvailable {
		score += 10
	}
	if score > 100 {
		score = 100
	}

	switch {
	case score >= 70:
		return score, PolicyRiskLevelCritical
	case score >= 40:
		return score, PolicyRiskLevelHigh
	case score >= 20:
		return score, PolicyRiskLevelMedium
	default:
		return score, PolicyRiskLevelLow
	}
}

func detectBlockingIssues(
	input PolicySimulationInput,
	candidate PolicyDefinition,
	affectedNamespaces []string,
) []PolicyIssue {
	var issues []PolicyIssue

	if blocksCriticalNamespace(candidate, affectedNamespaces) {
		issues = append(issues,
			PolicyIssue{
				Code:       PolicyErrorSimulationBlockingConflict,
				Message:    "candidate policy introduces a blocking conflict for a critical namespace",
				Severity:   PolicyIssueSeverityBlocking,
				Suggestion: "adjust the deny scope or add explicit allow exceptions for critical system traffic",
			},
			PolicyIssue{
				Code:       PolicyErrorCriticalNamespaceBlocked,
				Message:    "candidate policy would block traffic for kube-system/default namespaces",
				Severity:   PolicyIssueSeverityBlocking,
				Suggestion: "add exception rules for kube-system/default before publishing",
			},
		)
	}

	if input.CNICapabilityGap {
		issues = append(issues, PolicyIssue{
			Code:       PolicyErrorCNISemanticGap,
			Message:    "candidate policy relies on semantics the target CNI cannot preserve",
			Severity:   PolicyIssueSeverityBlocking,
			Suggestion: "remove the unsupported semantics or choose a compatible CNI adapter",
		})
	}

	if overlapsConflictingPolicy(input.Base, input.ExistingPolicies, candidate) {
		issues = append(issues, PolicyIssue{
			Code:       PolicyErrorSimulationBlockingConflict,
			Message:    "candidate policy conflicts with an existing policy targeting the same workload",
			Severity:   PolicyIssueSeverityBlocking,
			Suggestion: "reconcile conflicting allow/deny rules before publishing",
		})
	}

	return issues
}

func simulationPassed(blockingIssues []PolicyIssue, _ PolicyRiskLevel) bool {
	return len(blockingIssues) == 0
}

func detectWarnings(candidate PolicyDefinition, impact PolicyImpactSummary) []PolicyWarning {
	var warnings []PolicyWarning

	if len(impact.AffectedNamespaces) > 1 || impact.AffectedPods >= 10 {
		warnings = append(warnings, PolicyWarning{
			Code:    PolicyWarningImpactScopeExpanded,
			Message: "policy change expands the affected workload scope",
		})
	}
	if countHTTPRules(candidate) > 0 || countDNSRules(candidate) > 0 {
		warnings = append(warnings, PolicyWarning{
			Code:    PolicyWarningL7RuleSimplified,
			Message: "L7 rules may be simplified by downstream adapters during translation",
		})
	}
	if hasNonCriticalNamespace(impact.AffectedNamespaces) {
		warnings = append(warnings, PolicyWarning{
			Code:    PolicyWarningNonCriticalNamespaceChange,
			Message: "policy change affects non-critical namespaces and should be audited",
		})
	}

	return warnings
}

func buildImpactSummary(candidate PolicyDefinition, namespacePodCounts map[string]int) PolicyImpactSummary {
	namespaces := collectAffectedNamespaces(candidate, namespacePodCounts)
	affectedPods := 0
	for _, namespace := range namespaces {
		affectedPods += namespacePodCounts[namespace]
	}

	return PolicyImpactSummary{
		AffectedPods:       affectedPods,
		AffectedNamespaces: namespaces,
		NewDeniedFlows:     collectDeniedFlows(candidate),
	}
}

func collectAffectedNamespaces(candidate PolicyDefinition, namespacePodCounts map[string]int) []string {
	set := map[string]struct{}{}
	knownNamespaces := knownNamespaces(namespacePodCounts, candidate.Metadata.Namespace)

	addNamespace(set, candidate.Metadata.Namespace)
	addSelectorNamespaces(set, candidate.Spec.Target.NamespaceSelector, knownNamespaces)
	for _, rule := range candidate.Spec.Ingress {
		if rule.From != nil {
			addSelectorNamespaces(set, rule.From.NamespaceSelector, knownNamespaces)
		}
	}
	for _, rule := range candidate.Spec.Egress {
		if rule.To != nil {
			addSelectorNamespaces(set, rule.To.NamespaceSelector, knownNamespaces)
		}
	}

	namespaces := make([]string, 0, len(set))
	for namespace := range set {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

func knownNamespaces(namespacePodCounts map[string]int, currentNamespace string) []string {
	set := map[string]struct{}{}
	addNamespace(set, currentNamespace)
	for namespace := range namespacePodCounts {
		addNamespace(set, namespace)
	}
	for namespace := range criticalPolicyNamespaces {
		addNamespace(set, namespace)
	}

	namespaces := make([]string, 0, len(set))
	for namespace := range set {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

func addSelectorNamespaces(set map[string]struct{}, selector *PolicyLabelSelector, knownNamespaces []string) {
	if selector == nil {
		return
	}
	for _, namespace := range selectorExplicitNamespaceNames(selector) {
		addNamespace(set, namespace)
	}
	for _, namespace := range knownNamespaces {
		if selectorMatchesNamespace(selector, namespace) {
			addNamespace(set, namespace)
		}
	}
}

func selectorMatchesNamespace(selector *PolicyLabelSelector, namespace string) bool {
	if selector == nil {
		return false
	}

	labels := namespaceNameLabels(namespace)
	for key, value := range selector.MatchLabels {
		labelValue, ok := labels[key]
		if !ok || labelValue != value {
			return false
		}
	}
	for _, requirement := range selector.MatchExpressions {
		if !selectorRequirementMatches(labels, requirement) {
			return false
		}
	}
	return true
}

func selectorRequirementMatches(labels map[string]string, requirement PolicyLabelSelectorRequirement) bool {
	value, present := labels[requirement.Key]

	switch requirement.Operator {
	case "In":
		return present && stringInSlice(value, requirement.Values)
	case "NotIn":
		return present && !stringInSlice(value, requirement.Values)
	case "Exists":
		return present
	case "DoesNotExist":
		return !present
	default:
		return false
	}
}

func namespaceNameLabels(namespace string) map[string]string {
	return map[string]string{
		"name":                        namespace,
		"kubernetes.io/metadata.name": namespace,
	}
}

func selectorExplicitNamespaceNames(selector *PolicyLabelSelector) []string {
	if selector == nil {
		return nil
	}

	set := map[string]struct{}{}
	for _, key := range []string{"name", "kubernetes.io/metadata.name"} {
		addNamespace(set, selector.MatchLabels[key])
	}
	for _, requirement := range selector.MatchExpressions {
		if requirement.Operator != "In" {
			continue
		}
		if requirement.Key != "name" && requirement.Key != "kubernetes.io/metadata.name" {
			continue
		}
		for _, value := range requirement.Values {
			addNamespace(set, value)
		}
	}

	namespaces := make([]string, 0, len(set))
	for namespace := range set {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

func stringInSlice(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if candidate == value {
			return true
		}
	}
	return false
}

func addNamespace(set map[string]struct{}, namespace string) {
	if namespace == "" {
		return
	}
	set[namespace] = struct{}{}
}

func collectDeniedFlows(candidate PolicyDefinition) []string {
	var flows []string

	for _, rule := range candidate.Spec.Ingress {
		if rule.Action == PolicyRuleActionDeny && rule.Name != "" {
			flows = append(flows, "ingress:"+rule.Name)
		}
	}
	for _, rule := range candidate.Spec.Egress {
		if rule.Action == PolicyRuleActionDeny && rule.Name != "" {
			flows = append(flows, "egress:"+rule.Name)
		}
	}

	return flows
}

func countHTTPRules(candidate PolicyDefinition) int {
	count := 0
	for _, rule := range candidate.Spec.Ingress {
		count += len(rule.HTTP)
	}
	return count
}

func countDNSRules(candidate PolicyDefinition) int {
	count := 0
	for _, rule := range candidate.Spec.Ingress {
		count += len(rule.DNS)
	}
	return count
}

func blocksCriticalNamespace(candidate PolicyDefinition, affectedNamespaces []string) bool {
	if !hasDenyRules(candidate) {
		return false
	}
	for _, namespace := range affectedNamespaces {
		if isCriticalNamespace(namespace) {
			return true
		}
	}
	return false
}

func hasDenyRules(candidate PolicyDefinition) bool {
	for _, rule := range candidate.Spec.Ingress {
		if rule.Action == PolicyRuleActionDeny {
			return true
		}
	}
	for _, rule := range candidate.Spec.Egress {
		if rule.Action == PolicyRuleActionDeny {
			return true
		}
	}
	return false
}

func hasNonCriticalNamespace(namespaces []string) bool {
	for _, namespace := range namespaces {
		if namespace != "" && !isCriticalNamespace(namespace) {
			return true
		}
	}
	return false
}

func isCriticalNamespace(namespace string) bool {
	_, ok := criticalPolicyNamespaces[namespace]
	return ok
}

func hasIssueCode(issues []PolicyIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func overlapsConflictingPolicy(base *PolicyDefinition, existing []PolicyDefinition, candidate PolicyDefinition) bool {
	if base != nil && policiesConflict(*base, candidate) {
		return true
	}
	for _, policy := range existing {
		if policiesConflict(policy, candidate) {
			return true
		}
	}
	return false
}

func policiesConflict(existing PolicyDefinition, candidate PolicyDefinition) bool {
	if existing.Metadata.Namespace != candidate.Metadata.Namespace {
		return false
	}
	if !reflect.DeepEqual(existing.Spec.Target, candidate.Spec.Target) {
		return false
	}
	return ingressRulesConflict(existing.Spec.Ingress, candidate.Spec.Ingress) ||
		egressRulesConflict(existing.Spec.Egress, candidate.Spec.Egress)
}

func ingressRulesConflict(existing []PolicyIngressRule, candidate []PolicyIngressRule) bool {
	for _, left := range existing {
		for _, right := range candidate {
			if !ruleActionsConflict(left.Action, right.Action) {
				continue
			}
			if !portsOverlap(left.Ports, right.Ports) {
				continue
			}
			if peersOverlap(left.From, right.From) {
				return true
			}
		}
	}
	return false
}

func egressRulesConflict(existing []PolicyEgressRule, candidate []PolicyEgressRule) bool {
	for _, left := range existing {
		for _, right := range candidate {
			if !ruleActionsConflict(left.Action, right.Action) {
				continue
			}
			if !portsOverlap(left.Ports, right.Ports) {
				continue
			}
			if peersOverlap(left.To, right.To) {
				return true
			}
		}
	}
	return false
}

func ruleActionsConflict(left PolicyRuleAction, right PolicyRuleAction) bool {
	return left != "" && right != "" && left != right
}

func portsOverlap(left []PolicyPort, right []PolicyPort) bool {
	if len(left) == 0 || len(right) == 0 {
		return true
	}
	for _, leftPort := range left {
		for _, rightPort := range right {
			if policyPortsOverlap(leftPort, rightPort) {
				return true
			}
		}
	}
	return false
}

func policyPortsOverlap(left PolicyPort, right PolicyPort) bool {
	if !protocolsOverlap(left.Protocol, right.Protocol) {
		return false
	}

	leftStart, leftEnd, leftNumeric := numericPortRange(left)
	rightStart, rightEnd, rightNumeric := numericPortRange(right)
	if leftNumeric && rightNumeric {
		return leftStart <= rightEnd && rightStart <= leftEnd
	}

	leftValue := strings.TrimSpace(left.Port.String())
	rightValue := strings.TrimSpace(right.Port.String())
	if leftValue == "" || rightValue == "" {
		return true
	}
	return leftValue == rightValue
}

func protocolsOverlap(left string, right string) bool {
	if left == "" || right == "" {
		return true
	}
	return strings.EqualFold(left, right)
}

func numericPortRange(port PolicyPort) (int, int, bool) {
	start, ok := parseCanonicalInt(strings.TrimSpace(port.Port.String()))
	if !ok {
		return 0, 0, false
	}
	end := start
	if port.EndPort > 0 {
		end = int(port.EndPort)
	}
	if end < start {
		end = start
	}
	return start, end, true
}

func peersOverlap(left *PolicyPeer, right *PolicyPeer) bool {
	if left == nil || right == nil {
		return true
	}
	if !stringDimensionOverlaps(left.ServiceAccount, right.ServiceAccount) {
		return false
	}
	if !stringDimensionOverlaps(left.FQDN, right.FQDN) {
		return false
	}
	if !ipBlockOverlap(left.IPBlock, right.IPBlock) {
		return false
	}
	if !selectorsOverlap(left.NamespaceSelector, right.NamespaceSelector, selectorExplicitNamespaceNames) {
		return false
	}
	if !selectorsOverlap(left.PodSelector, right.PodSelector, nil) {
		return false
	}
	return true
}

func stringDimensionOverlaps(left string, right string) bool {
	return left == "" || right == "" || left == right
}

func ipBlockOverlap(left *PolicyIPBlock, right *PolicyIPBlock) bool {
	if left == nil || right == nil {
		return true
	}
	return reflect.DeepEqual(*left, *right)
}

func selectorsOverlap(
	left *PolicyLabelSelector,
	right *PolicyLabelSelector,
	explicitNames func(*PolicyLabelSelector) []string,
) bool {
	if left == nil || right == nil {
		return true
	}
	if reflect.DeepEqual(left, right) {
		return true
	}
	if selectorsContradict(left, right) {
		return false
	}
	if explicitNames != nil {
		leftNames := explicitNames(left)
		rightNames := explicitNames(right)
		if len(leftNames) > 0 && len(rightNames) > 0 {
			return slicesOverlap(leftNames, rightNames)
		}
	}
	return false
}

func selectorsContradict(left *PolicyLabelSelector, right *PolicyLabelSelector) bool {
	for key, value := range left.MatchLabels {
		if other, ok := right.MatchLabels[key]; ok && other != value {
			return true
		}
	}
	for key, value := range right.MatchLabels {
		if other, ok := left.MatchLabels[key]; ok && other != value {
			return true
		}
	}
	return false
}

func slicesOverlap(left []string, right []string) bool {
	set := map[string]struct{}{}
	for _, value := range left {
		set[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}

func normalizedPolicyDefinition(def *PolicyDefinition) PolicyDefinition {
	if def == nil {
		policy := PolicyDefinition{}
		policy.ApplyDefaults()
		return policy
	}
	policy := *def
	policy.ApplyDefaults()
	return policy
}

func (r PolicySimulationResult) outputContract() policySimulationResultOutput {
	return policySimulationResultOutput{
		Passed:         r.Passed,
		BlockingIssues: r.BlockingIssues,
		Warnings:       r.Warnings,
		ImpactSummary: policyImpactSummaryOutput{
			AffectedPods:       r.ImpactSummary.AffectedPods,
			AffectedNamespaces: r.ImpactSummary.AffectedNamespaces,
			NewDeniedFlows:     r.ImpactSummary.NewDeniedFlows,
		},
		RiskScore: r.RiskScore,
		RiskLevel: r.RiskLevel,
	}
}
