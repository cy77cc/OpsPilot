package cluster

import (
	"fmt"
	"strings"
)

const (
	ciliumAPIVersion          = "cilium.io/v2"
	ciliumPolicyKind          = "CiliumNetworkPolicy"
	ciliumClusterwideKind     = "CiliumClusterwideNetworkPolicy"
	ciliumNamespaceLabelKey   = "k8s:io.kubernetes.pod.namespace"
	ciliumNamespaceNameLabel  = "kubernetes.io/metadata.name"
	ciliumNamespaceAliasLabel = "name"
)

// CiliumPolicyAdapter translates the phase-2 DSL into a Cilium-oriented policy shape.
type CiliumPolicyAdapter struct{}

// CiliumNetworkPolicy is the minimal Cilium policy output contract needed by phase-2.
type CiliumNetworkPolicy struct {
	APIVersion string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   PolicyObjectMetadata    `json:"metadata" yaml:"metadata"`
	Spec       CiliumNetworkPolicySpec `json:"spec" yaml:"spec"`
}

// CiliumClusterwideNetworkPolicy is the cluster-scoped Cilium policy output contract needed by phase-2.
type CiliumClusterwideNetworkPolicy struct {
	APIVersion string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   PolicyObjectMetadata    `json:"metadata" yaml:"metadata"`
	Spec       CiliumNetworkPolicySpec `json:"spec" yaml:"spec"`
}

// CiliumNetworkPolicySpec contains the translated Cilium policy rules.
type CiliumNetworkPolicySpec struct {
	EndpointSelector CiliumEndpointSelector `json:"endpointSelector" yaml:"endpointSelector"`
	Ingress          []CiliumIngressRule    `json:"ingress,omitempty" yaml:"ingress,omitempty"`
	IngressDeny      []CiliumIngressRule    `json:"ingressDeny,omitempty" yaml:"ingressDeny,omitempty"`
	Egress           []CiliumEgressRule     `json:"egress,omitempty" yaml:"egress,omitempty"`
	EgressDeny       []CiliumEgressRule     `json:"egressDeny,omitempty" yaml:"egressDeny,omitempty"`
}

// CiliumEndpointSelector mirrors the selector shape used in Cilium peer rules.
type CiliumEndpointSelector struct {
	MatchLabels      map[string]string                `json:"matchLabels,omitempty" yaml:"matchLabels,omitempty"`
	MatchExpressions []CiliumLabelSelectorRequirement `json:"matchExpressions,omitempty" yaml:"matchExpressions,omitempty"`
}

// CiliumLabelSelectorRequirement mirrors Cilium matchExpressions entries.
type CiliumLabelSelectorRequirement struct {
	Key      string   `json:"key" yaml:"key"`
	Operator string   `json:"operator" yaml:"operator"`
	Values   []string `json:"values,omitempty" yaml:"values,omitempty"`
}

// CiliumIngressRule contains ingress peer, CIDR, and port rules.
type CiliumIngressRule struct {
	FromEndpoints []CiliumEndpointSelector `json:"fromEndpoints,omitempty" yaml:"fromEndpoints,omitempty"`
	FromCIDRSet   []CiliumCIDRRule         `json:"fromCIDRSet,omitempty" yaml:"fromCIDRSet,omitempty"`
	ToPorts       []CiliumPortRule         `json:"toPorts,omitempty" yaml:"toPorts,omitempty"`
}

// CiliumEgressRule contains egress peer, CIDR, FQDN, and port rules.
type CiliumEgressRule struct {
	ToEndpoints []CiliumEndpointSelector `json:"toEndpoints,omitempty" yaml:"toEndpoints,omitempty"`
	ToCIDRSet   []CiliumCIDRRule         `json:"toCIDRSet,omitempty" yaml:"toCIDRSet,omitempty"`
	ToFQDNs     []CiliumFQDNSelector     `json:"toFQDNs,omitempty" yaml:"toFQDNs,omitempty"`
	ToPorts     []CiliumPortRule         `json:"toPorts,omitempty" yaml:"toPorts,omitempty"`
}

// CiliumCIDRRule mirrors Cilium's CIDRSet entries.
type CiliumCIDRRule struct {
	CIDR   string   `json:"cidr" yaml:"cidr"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
}

// CiliumPortRule mirrors Cilium's toPorts entries.
type CiliumPortRule struct {
	Ports []CiliumPortProtocol `json:"ports,omitempty" yaml:"ports,omitempty"`
	Rules *CiliumL7Rules       `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// CiliumPortProtocol mirrors Cilium's port/protocol pairs.
type CiliumPortProtocol struct {
	Port     string `json:"port,omitempty" yaml:"port,omitempty"`
	Protocol string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// CiliumL7Rules keeps HTTP and DNS rules disjoint, matching Cilium's union semantics.
type CiliumL7Rules struct {
	HTTP []CiliumHTTPRule `json:"http,omitempty" yaml:"http,omitempty"`
	DNS  []CiliumDNSRule  `json:"dns,omitempty" yaml:"dns,omitempty"`
}

// CiliumHTTPRule mirrors Cilium's HTTP port rule fields used in phase-2.
type CiliumHTTPRule struct {
	Method string `json:"method,omitempty" yaml:"method,omitempty"`
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
}

// CiliumDNSRule mirrors Cilium's DNS matchName/matchPattern forms.
type CiliumDNSRule struct {
	MatchName    string `json:"matchName,omitempty" yaml:"matchName,omitempty"`
	MatchPattern string `json:"matchPattern,omitempty" yaml:"matchPattern,omitempty"`
}

// CiliumFQDNSelector mirrors Cilium's toFQDNs selector fields.
type CiliumFQDNSelector struct {
	MatchName    string `json:"matchName,omitempty" yaml:"matchName,omitempty"`
	MatchPattern string `json:"matchPattern,omitempty" yaml:"matchPattern,omitempty"`
}

// CiliumTranslationError reports unsupported or lossy Cilium mappings.
type CiliumTranslationError struct {
	Code    string
	Field   string
	Message string
}

func (e *CiliumTranslationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ToCiliumPolicy translates the DSL into a CiliumNetworkPolicy.
func ToCiliumPolicy(def *PolicyDefinition) (CiliumNetworkPolicy, error) {
	return CiliumPolicyAdapter{}.ToCiliumPolicy(def)
}

// ToCiliumClusterwidePolicy translates the DSL into a CiliumClusterwideNetworkPolicy.
func ToCiliumClusterwidePolicy(def *PolicyDefinition) (CiliumClusterwideNetworkPolicy, error) {
	return CiliumPolicyAdapter{}.ToCiliumClusterwidePolicy(def)
}

// ToCiliumPolicy translates the DSL into a CiliumNetworkPolicy.
func (CiliumPolicyAdapter) ToCiliumPolicy(def *PolicyDefinition) (CiliumNetworkPolicy, error) {
	translated, err := translateCiliumPolicy(def, true)
	if err != nil {
		return CiliumNetworkPolicy{}, err
	}

	return CiliumNetworkPolicy{
		APIVersion: ciliumAPIVersion,
		Kind:       ciliumPolicyKind,
		Metadata:   translated.Metadata,
		Spec:       translated.Spec,
	}, nil
}

// ToCiliumClusterwidePolicy translates the DSL into a CiliumClusterwideNetworkPolicy.
func (CiliumPolicyAdapter) ToCiliumClusterwidePolicy(def *PolicyDefinition) (CiliumClusterwideNetworkPolicy, error) {
	translated, err := translateCiliumPolicy(def, false)
	if err != nil {
		return CiliumClusterwideNetworkPolicy{}, err
	}

	return CiliumClusterwideNetworkPolicy{
		APIVersion: ciliumAPIVersion,
		Kind:       ciliumClusterwideKind,
		Metadata:   translated.Metadata,
		Spec:       translated.Spec,
	}, nil
}

func translateCiliumPolicy(def *PolicyDefinition, includeNamespace bool) (CiliumNetworkPolicy, error) {
	policyDef := normalizedPolicyDefinition(def)

	if policyDef.Spec.Advanced.Order != nil {
		return CiliumNetworkPolicy{}, unsupportedCiliumField("spec.advanced.order", "advanced.order is not supported by the Cilium adapter")
	}

	metadata := PolicyObjectMetadata{
		Name:   policyDef.Metadata.Name,
		Labels: cloneStringMap(policyDef.Metadata.Labels),
	}
	if includeNamespace {
		metadata.Namespace = policyDef.Metadata.Namespace
	}

	policy := CiliumNetworkPolicy{
		Metadata: metadata,
		Spec: CiliumNetworkPolicySpec{
			EndpointSelector: translateCiliumTargetSelector(policyDef.Spec.Target),
		},
	}

	for idx, rule := range policyDef.Spec.Ingress {
		translated, err := translateCiliumIngressRule(rule, idx)
		if err != nil {
			return CiliumNetworkPolicy{}, err
		}
		if rule.Action == PolicyRuleActionDeny {
			policy.Spec.IngressDeny = append(policy.Spec.IngressDeny, translated)
			continue
		}
		policy.Spec.Ingress = append(policy.Spec.Ingress, translated)
	}

	for idx, rule := range policyDef.Spec.Egress {
		translated, err := translateCiliumEgressRule(rule, idx)
		if err != nil {
			return CiliumNetworkPolicy{}, err
		}
		if rule.Action == PolicyRuleActionDeny {
			policy.Spec.EgressDeny = append(policy.Spec.EgressDeny, translated)
			continue
		}
		policy.Spec.Egress = append(policy.Spec.Egress, translated)
	}

	return policy, nil
}

func translateCiliumTargetSelector(target PolicyTarget) CiliumEndpointSelector {
	selector := translateCiliumPodSelector(target.PodSelector)
	if target.NamespaceSelector != nil {
		mergeCiliumNamespaceSelector(&selector, target.NamespaceSelector)
	}
	return selector
}

func translateCiliumIngressRule(rule PolicyIngressRule, index int) (CiliumIngressRule, error) {
	var translated CiliumIngressRule

	if rule.From != nil {
		if rule.From.ServiceAccount != "" {
			return CiliumIngressRule{}, unsupportedCiliumField(
				fmt.Sprintf("spec.ingress[%d].from.serviceAccount", index),
				"serviceAccount is not supported by the Cilium adapter",
			)
		}

		if selector, ok := translateCiliumPeerSelector(rule.From); ok {
			translated.FromEndpoints = []CiliumEndpointSelector{selector}
		}
		if rule.From.IPBlock != nil {
			translated.FromCIDRSet = []CiliumCIDRRule{translateCiliumCIDRRule(rule.From.IPBlock)}
		}
	}

	translated.ToPorts = buildCiliumPortRules(rule.Ports, rule.HTTP, rule.DNS)
	return translated, nil
}

func translateCiliumEgressRule(rule PolicyEgressRule, index int) (CiliumEgressRule, error) {
	var translated CiliumEgressRule

	if rule.To != nil {
		if rule.To.ServiceAccount != "" {
			return CiliumEgressRule{}, unsupportedCiliumField(
				fmt.Sprintf("spec.egress[%d].to.serviceAccount", index),
				"serviceAccount is not supported by the Cilium adapter",
			)
		}

		if selector, ok := translateCiliumPeerSelector(rule.To); ok {
			translated.ToEndpoints = []CiliumEndpointSelector{selector}
		}
		if rule.To.IPBlock != nil {
			translated.ToCIDRSet = []CiliumCIDRRule{translateCiliumCIDRRule(rule.To.IPBlock)}
		}
		if rule.To.FQDN != "" {
			translated.ToFQDNs = []CiliumFQDNSelector{translateCiliumFQDNSelector(rule.To.FQDN)}
		}
	}

	translated.ToPorts = buildCiliumPortRules(rule.Ports, nil, nil)
	return translated, nil
}

func translateCiliumPodSelector(selector *PolicyLabelSelector) CiliumEndpointSelector {
	if selector == nil {
		return CiliumEndpointSelector{}
	}

	translated := CiliumEndpointSelector{
		MatchLabels: cloneStringMap(selector.MatchLabels),
	}
	if len(selector.MatchExpressions) > 0 {
		translated.MatchExpressions = make([]CiliumLabelSelectorRequirement, 0, len(selector.MatchExpressions))
		for _, expr := range selector.MatchExpressions {
			translated.MatchExpressions = append(translated.MatchExpressions, CiliumLabelSelectorRequirement{
				Key:      expr.Key,
				Operator: expr.Operator,
				Values:   cloneStrings(expr.Values),
			})
		}
	}
	return translated
}

func translateCiliumPeerSelector(peer *PolicyPeer) (CiliumEndpointSelector, bool) {
	if peer == nil {
		return CiliumEndpointSelector{}, false
	}

	var selector CiliumEndpointSelector
	if peer.PodSelector != nil {
		selector = translateCiliumPodSelector(peer.PodSelector)
	}

	if peer.NamespaceSelector != nil {
		mergeCiliumNamespaceSelector(&selector, peer.NamespaceSelector)
	}

	return selector, hasCiliumEndpointSelectorContent(selector)
}

func mergeCiliumNamespaceSelector(target *CiliumEndpointSelector, namespaceSelector *PolicyLabelSelector) {
	if target == nil || namespaceSelector == nil {
		return
	}

	if target.MatchLabels == nil && len(namespaceSelector.MatchLabels) > 0 {
		target.MatchLabels = map[string]string{}
	}
	for key, value := range namespaceSelector.MatchLabels {
		target.MatchLabels[translateCiliumNamespaceKey(key)] = value
	}
	if len(namespaceSelector.MatchExpressions) > 0 {
		target.MatchExpressions = append(target.MatchExpressions, translateCiliumNamespaceExpressions(namespaceSelector.MatchExpressions)...)
	}
}

func translateCiliumNamespaceExpressions(expressions []PolicyLabelSelectorRequirement) []CiliumLabelSelectorRequirement {
	if len(expressions) == 0 {
		return nil
	}

	translated := make([]CiliumLabelSelectorRequirement, 0, len(expressions))
	for _, expr := range expressions {
		translated = append(translated, CiliumLabelSelectorRequirement{
			Key:      translateCiliumNamespaceKey(expr.Key),
			Operator: expr.Operator,
			Values:   cloneStrings(expr.Values),
		})
	}
	return translated
}

func translateCiliumNamespaceKey(key string) string {
	switch key {
	case ciliumNamespaceLabelKey:
		return key
	case ciliumNamespaceNameLabel, ciliumNamespaceAliasLabel, "io.kubernetes.pod.namespace":
		return ciliumNamespaceLabelKey
	default:
		if strings.HasPrefix(key, "k8s:") {
			return key
		}
		return "k8s:" + key
	}
}

func translateCiliumCIDRRule(block *PolicyIPBlock) CiliumCIDRRule {
	if block == nil {
		return CiliumCIDRRule{}
	}
	return CiliumCIDRRule{
		CIDR:   block.CIDR,
		Except: cloneStrings(block.Except),
	}
}

func buildCiliumPortRules(ports []PolicyPort, httpRules []PolicyHTTPRule, dnsRules []PolicyDNSRule) []CiliumPortRule {
	basePorts := translateCiliumPorts(ports)
	if len(basePorts) == 0 && len(httpRules) == 0 && len(dnsRules) == 0 {
		return nil
	}

	var rules []CiliumPortRule
	if len(httpRules) > 0 {
		rules = append(rules, CiliumPortRule{
			Ports: cloneCiliumPorts(basePorts),
			Rules: &CiliumL7Rules{HTTP: translateCiliumHTTPRules(httpRules)},
		})
	}
	if len(dnsRules) > 0 {
		rules = append(rules, CiliumPortRule{
			Ports: cloneCiliumPorts(basePorts),
			Rules: &CiliumL7Rules{DNS: translateCiliumDNSRules(dnsRules)},
		})
	}
	if len(rules) > 0 {
		return rules
	}

	return []CiliumPortRule{{Ports: basePorts}}
}

func translateCiliumPorts(ports []PolicyPort) []CiliumPortProtocol {
	if len(ports) == 0 {
		return nil
	}

	translated := make([]CiliumPortProtocol, 0, len(ports))
	for _, port := range ports {
		protocol := strings.ToUpper(strings.TrimSpace(port.Protocol))
		if protocol == "" {
			protocol = "ANY"
		}
		translated = append(translated, CiliumPortProtocol{
			Port:     port.Port.String(),
			Protocol: protocol,
		})
	}
	return translated
}

func translateCiliumHTTPRules(rules []PolicyHTTPRule) []CiliumHTTPRule {
	if len(rules) == 0 {
		return nil
	}

	translated := make([]CiliumHTTPRule, 0, len(rules))
	for _, rule := range rules {
		translated = append(translated, CiliumHTTPRule{
			Method: rule.Method,
			Path:   rule.Path,
		})
	}
	return translated
}

func translateCiliumDNSRules(rules []PolicyDNSRule) []CiliumDNSRule {
	if len(rules) == 0 {
		return nil
	}

	translated := make([]CiliumDNSRule, 0, len(rules))
	for _, rule := range rules {
		translated = append(translated, translateCiliumDNSRule(rule.MatchPattern))
	}
	return translated
}

func translateCiliumDNSRule(pattern string) CiliumDNSRule {
	if strings.Contains(pattern, "*") {
		return CiliumDNSRule{MatchPattern: pattern}
	}
	return CiliumDNSRule{MatchName: pattern}
}

func translateCiliumFQDNSelector(value string) CiliumFQDNSelector {
	if strings.Contains(value, "*") {
		return CiliumFQDNSelector{MatchPattern: value}
	}
	return CiliumFQDNSelector{MatchName: value}
}

func unsupportedCiliumField(field string, message string) error {
	RecordCNIAdapterTranslationError("cilium", PolicyErrorCNISemanticGap)
	return &CiliumTranslationError{
		Code:    PolicyErrorCNISemanticGap,
		Field:   field,
		Message: message,
	}
}

func hasCiliumEndpointSelectorContent(selector CiliumEndpointSelector) bool {
	return len(selector.MatchLabels) > 0 || len(selector.MatchExpressions) > 0
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	cloned := make([]string, len(input))
	copy(cloned, input)
	return cloned
}

func cloneCiliumPorts(input []CiliumPortProtocol) []CiliumPortProtocol {
	if len(input) == 0 {
		return nil
	}
	cloned := make([]CiliumPortProtocol, len(input))
	copy(cloned, input)
	return cloned
}
