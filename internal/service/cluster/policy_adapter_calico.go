package cluster

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const (
	calicoAPIVersion       = "crd.projectcalico.org/v1"
	calicoPolicyKind       = "NetworkPolicy"
	calicoGlobalPolicyKind = "GlobalNetworkPolicy"
	calicoSelectorAll      = "all()"
)

// CalicoPolicyAdapter translates the phase-2 DSL into a Calico-oriented policy shape.
type CalicoPolicyAdapter struct{}

// CalicoNetworkPolicy is the minimal Calico namespaced policy contract needed by phase-2.
type CalicoNetworkPolicy struct {
	APIVersion string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   PolicyObjectMetadata    `json:"metadata" yaml:"metadata"`
	Spec       CalicoNetworkPolicySpec `json:"spec" yaml:"spec"`
}

// CalicoGlobalNetworkPolicy is the minimal Calico cluster-scoped policy contract needed by phase-2.
type CalicoGlobalNetworkPolicy struct {
	APIVersion string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   PolicyObjectMetadata    `json:"metadata" yaml:"metadata"`
	Spec       CalicoNetworkPolicySpec `json:"spec" yaml:"spec"`
}

// CalicoNetworkPolicySpec contains the translated Calico policy fields used in phase-2.
type CalicoNetworkPolicySpec struct {
	Selector          string       `json:"selector,omitempty" yaml:"selector,omitempty"`
	NamespaceSelector string       `json:"namespaceSelector,omitempty" yaml:"namespaceSelector,omitempty"`
	Order             *float64     `json:"order,omitempty" yaml:"order,omitempty"`
	DoNotTrack        bool         `json:"doNotTrack,omitempty" yaml:"doNotTrack,omitempty"`
	ApplyOnForward    bool         `json:"applyOnForward,omitempty" yaml:"applyOnForward,omitempty"`
	Types             []PolicyType `json:"types,omitempty" yaml:"types,omitempty"`
	Ingress           []CalicoRule `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	Egress            []CalicoRule `json:"egress,omitempty" yaml:"egress,omitempty"`
}

// CalicoRule contains the translated Calico rule fields used in phase-2.
type CalicoRule struct {
	Action      string           `json:"action,omitempty" yaml:"action,omitempty"`
	Source      CalicoEntityRule `json:"source,omitempty" yaml:"source,omitempty"`
	Destination CalicoEntityRule `json:"destination,omitempty" yaml:"destination,omitempty"`
}

// CalicoEntityRule contains peer/port selection fields shared by ingress and egress rules.
type CalicoEntityRule struct {
	Selector               string   `json:"selector,omitempty" yaml:"selector,omitempty"`
	NamespaceSelector      string   `json:"namespaceSelector,omitempty" yaml:"namespaceSelector,omitempty"`
	ServiceAccountSelector string   `json:"serviceAccountSelector,omitempty" yaml:"serviceAccountSelector,omitempty"`
	Nets                   []string `json:"nets,omitempty" yaml:"nets,omitempty"`
	NotNets                []string `json:"notNets,omitempty" yaml:"notNets,omitempty"`
	Ports                  []string `json:"ports,omitempty" yaml:"ports,omitempty"`
}

// CalicoTranslationError reports unsupported or lossy Calico mappings.
type CalicoTranslationError struct {
	Code    string
	Field   string
	Message string
}

func (e *CalicoTranslationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ToCalicoPolicy translates the DSL into a Calico NetworkPolicy.
func ToCalicoPolicy(def *PolicyDefinition) (CalicoNetworkPolicy, []PolicyWarning, error) {
	return CalicoPolicyAdapter{}.ToCalicoPolicy(def)
}

// ToCalicoGlobalPolicy translates the DSL into a Calico GlobalNetworkPolicy.
func ToCalicoGlobalPolicy(def *PolicyDefinition) (CalicoGlobalNetworkPolicy, []PolicyWarning, error) {
	return CalicoPolicyAdapter{}.ToCalicoGlobalPolicy(def)
}

// ToCalicoPolicy translates the DSL into a Calico NetworkPolicy.
func (CalicoPolicyAdapter) ToCalicoPolicy(def *PolicyDefinition) (CalicoNetworkPolicy, []PolicyWarning, error) {
	translated, warnings, err := translateCalicoPolicy(def, true)
	if err != nil {
		return CalicoNetworkPolicy{}, nil, err
	}

	return CalicoNetworkPolicy{
		APIVersion: calicoAPIVersion,
		Kind:       calicoPolicyKind,
		Metadata:   translated.Metadata,
		Spec:       translated.Spec,
	}, warnings, nil
}

// ToCalicoGlobalPolicy translates the DSL into a Calico GlobalNetworkPolicy.
func (CalicoPolicyAdapter) ToCalicoGlobalPolicy(def *PolicyDefinition) (CalicoGlobalNetworkPolicy, []PolicyWarning, error) {
	translated, warnings, err := translateCalicoPolicy(def, false)
	if err != nil {
		return CalicoGlobalNetworkPolicy{}, nil, err
	}

	return CalicoGlobalNetworkPolicy{
		APIVersion: calicoAPIVersion,
		Kind:       calicoGlobalPolicyKind,
		Metadata:   translated.Metadata,
		Spec:       translated.Spec,
	}, warnings, nil
}

func translateCalicoPolicy(def *PolicyDefinition, includeNamespace bool) (CalicoNetworkPolicy, []PolicyWarning, error) {
	policyDef := normalizedPolicyDefinition(def)

	metadata := PolicyObjectMetadata{
		Name:   policyDef.Metadata.Name,
		Labels: cloneStringMap(policyDef.Metadata.Labels),
	}
	if includeNamespace {
		metadata.Namespace = policyDef.Metadata.Namespace
	}

	policy := CalicoNetworkPolicy{
		Metadata: metadata,
		Spec: CalicoNetworkPolicySpec{
			Order:          policyDef.Spec.Advanced.Order,
			DoNotTrack:     policyDef.Spec.Advanced.DoNotTrack,
			ApplyOnForward: policyDef.Spec.Advanced.ApplyOnForward,
			Types:          clonePolicyTypes(policyDef.Spec.PolicyTypes),
		},
	}
	selector, err := selectorOrAll(policyDef.Spec.Target.PodSelector, "spec.target.podSelector")
	if err != nil {
		return CalicoNetworkPolicy{}, nil, err
	}
	policy.Spec.Selector = selector
	if policyDef.Spec.Target.NamespaceSelector != nil {
		namespaceSelector, err := toCalicoSelector(policyDef.Spec.Target.NamespaceSelector, "spec.target.namespaceSelector")
		if err != nil {
			return CalicoNetworkPolicy{}, nil, err
		}
		policy.Spec.NamespaceSelector = namespaceSelector
	}

	warningSet := map[string]PolicyWarning{}

	for idx, rule := range policyDef.Spec.Ingress {
		translated, ruleWarnings, err := translateCalicoIngressRule(rule, idx)
		if err != nil {
			return CalicoNetworkPolicy{}, nil, err
		}
		policy.Spec.Ingress = append(policy.Spec.Ingress, translated)
		for _, warning := range ruleWarnings {
			warningSet[warning.Code] = warning
		}
	}

	for idx, rule := range policyDef.Spec.Egress {
		translated, ruleWarnings, err := translateCalicoEgressRule(rule, idx)
		if err != nil {
			return CalicoNetworkPolicy{}, nil, err
		}
		policy.Spec.Egress = append(policy.Spec.Egress, translated)
		for _, warning := range ruleWarnings {
			warningSet[warning.Code] = warning
		}
	}

	return policy, orderedPolicyWarnings(warningSet), nil
}

func translateCalicoIngressRule(rule PolicyIngressRule, index int) (CalicoRule, []PolicyWarning, error) {
	warnings := l7DowngradeWarnings(
		fmt.Sprintf("spec.ingress[%d].http", index),
		fmt.Sprintf("spec.ingress[%d].dns", index),
		len(rule.HTTP) > 0,
		len(rule.DNS) > 0,
	)

	source, err := translateCalicoPeer(rule.From, fmt.Sprintf("spec.ingress[%d].from", index))
	if err != nil {
		return CalicoRule{}, nil, err
	}

	translated := CalicoRule{
		Action:      string(rule.Action),
		Source:      source,
		Destination: CalicoEntityRule{Ports: translateCalicoPorts(rule.Ports)},
	}

	return translated, warnings, nil
}

func translateCalicoEgressRule(rule PolicyEgressRule, index int) (CalicoRule, []PolicyWarning, error) {
	if rule.To != nil && rule.To.FQDN != "" {
		return CalicoRule{}, nil, unsupportedCalicoField(
			fmt.Sprintf("spec.egress[%d].to.fqdn", index),
			"fqdn is not supported by the Calico adapter",
		)
	}

	warnings := l7DowngradeWarnings(
		fmt.Sprintf("spec.egress[%d].http", index),
		fmt.Sprintf("spec.egress[%d].dns", index),
		false,
		false,
	)

	destination, err := translateCalicoPeer(rule.To, fmt.Sprintf("spec.egress[%d].to", index))
	if err != nil {
		return CalicoRule{}, nil, err
	}
	destination.Ports = translateCalicoPorts(rule.Ports)

	translated := CalicoRule{
		Action:      string(rule.Action),
		Destination: destination,
	}

	return translated, warnings, nil
}

func translateCalicoPeer(peer *PolicyPeer, fieldPath string) (CalicoEntityRule, error) {
	if peer == nil {
		return CalicoEntityRule{}, nil
	}

	selector, err := selectorOrEmpty(peer.PodSelector, fieldPath+".podSelector")
	if err != nil {
		return CalicoEntityRule{}, err
	}
	translated := CalicoEntityRule{Selector: selector}
	if peer.NamespaceSelector != nil {
		namespaceSelector, err := toCalicoSelector(peer.NamespaceSelector, fieldPath+".namespaceSelector")
		if err != nil {
			return CalicoEntityRule{}, err
		}
		translated.NamespaceSelector = namespaceSelector
	}
	if peer.ServiceAccount != "" {
		translated.ServiceAccountSelector = fmt.Sprintf("name == %s", strconv.Quote(peer.ServiceAccount))
	}
	if peer.IPBlock != nil {
		translated.Nets = []string{peer.IPBlock.CIDR}
		translated.NotNets = cloneStrings(peer.IPBlock.Except)
	}

	return translated, nil
}

func selectorOrAll(selector *PolicyLabelSelector, fieldPath string) (string, error) {
	value, err := toCalicoSelector(selector, fieldPath)
	if err != nil {
		return "", err
	}
	if value == "" {
		return calicoSelectorAll, nil
	}
	return value, nil
}

func selectorOrEmpty(selector *PolicyLabelSelector, fieldPath string) (string, error) {
	return toCalicoSelector(selector, fieldPath)
}

func toCalicoSelector(selector *PolicyLabelSelector, fieldPath string) (string, error) {
	if selector == nil {
		return "", nil
	}

	parts := make([]string, 0, len(selector.MatchLabels)+len(selector.MatchExpressions))

	if len(selector.MatchLabels) > 0 {
		keys := make([]string, 0, len(selector.MatchLabels))
		for key := range selector.MatchLabels {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s == %s", key, strconv.Quote(selector.MatchLabels[key])))
		}
	}

	for idx, expr := range selector.MatchExpressions {
		translated, err := toCalicoSelectorRequirement(expr, fmt.Sprintf("%s.matchExpressions[%d].operator", fieldPath, idx))
		if err != nil {
			return "", err
		}
		parts = append(parts, translated)
	}

	return strings.Join(parts, " && "), nil
}

func toCalicoSelectorRequirement(requirement PolicyLabelSelectorRequirement, fieldPath string) (string, error) {
	switch requirement.Operator {
	case "In":
		return fmt.Sprintf("%s in %s", requirement.Key, calicoSetLiteral(requirement.Values)), nil
	case "NotIn":
		return fmt.Sprintf("%s not in %s", requirement.Key, calicoSetLiteral(requirement.Values)), nil
	case "Exists":
		return fmt.Sprintf("has(%s)", requirement.Key), nil
	case "DoesNotExist":
		return fmt.Sprintf("! has(%s)", requirement.Key), nil
	default:
		return "", unsupportedCalicoField(fieldPath, fmt.Sprintf("matchExpressions operator %q is not supported by the Calico adapter", requirement.Operator))
	}
}

func calicoSetLiteral(values []string) string {
	if len(values) == 0 {
		return "{}"
	}

	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "{" + strings.Join(quoted, ", ") + "}"
}

func translateCalicoPorts(ports []PolicyPort) []string {
	if len(ports) == 0 {
		return nil
	}

	translated := make([]string, 0, len(ports))
	for _, port := range ports {
		if value := port.Port.String(); value != "" {
			translated = append(translated, value)
		}
	}
	return translated
}

func clonePolicyTypes(types []PolicyType) []PolicyType {
	if len(types) == 0 {
		return nil
	}

	cloned := make([]PolicyType, len(types))
	copy(cloned, types)
	return cloned
}

func l7DowngradeWarnings(httpField string, dnsField string, hasHTTP bool, hasDNS bool) []PolicyWarning {
	if !hasHTTP && !hasDNS {
		return nil
	}

	warnings := []PolicyWarning{
		{
			Code:    PolicyWarningCNICapabilityDowngrade,
			Message: "Calico adapter downgraded unsupported L7 rules to L3/L4 semantics",
		},
		{
			Code:    PolicyWarningL7RuleSimplified,
			Message: "Calico adapter omitted unsupported L7 HTTP/DNS rules",
		},
	}

	if hasHTTP && !hasDNS {
		warnings[1].Message = fmt.Sprintf("%s omitted because Calico only supports L3/L4 policy semantics", httpField)
	}
	if hasDNS && !hasHTTP {
		warnings[1].Message = fmt.Sprintf("%s omitted because Calico only supports L3/L4 policy semantics", dnsField)
	}
	if hasHTTP && hasDNS {
		warnings[1].Message = fmt.Sprintf("%s and %s omitted because Calico only supports L3/L4 policy semantics", httpField, dnsField)
	}

	return warnings
}

func orderedPolicyWarnings(set map[string]PolicyWarning) []PolicyWarning {
	if len(set) == 0 {
		return nil
	}

	order := []string{
		PolicyWarningCNICapabilityDowngrade,
		PolicyWarningL7RuleSimplified,
		PolicyWarningImpactScopeExpanded,
		PolicyWarningNonCriticalNamespaceChange,
	}

	warnings := make([]PolicyWarning, 0, len(set))
	for _, code := range order {
		if warning, ok := set[code]; ok {
			warnings = append(warnings, warning)
			delete(set, code)
		}
	}

	if len(set) == 0 {
		return warnings
	}

	extraCodes := make([]string, 0, len(set))
	for code := range set {
		extraCodes = append(extraCodes, code)
	}
	slices.Sort(extraCodes)
	for _, code := range extraCodes {
		warnings = append(warnings, set[code])
	}
	return warnings
}

func unsupportedCalicoField(field string, message string) error {
	RecordCNIAdapterTranslationError("calico", PolicyErrorCNISemanticGap)
	return &CalicoTranslationError{
		Code:    PolicyErrorCNISemanticGap,
		Field:   field,
		Message: message,
	}
}
