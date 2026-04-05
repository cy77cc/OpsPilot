package cluster

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	flannelNetworkPolicyAPIVersion = "networking.k8s.io/v1"
	flannelNetworkPolicyKind       = "NetworkPolicy"
	flannelEnableSuggestion        = "helm upgrade flannel --set netpol.enabled=true"
)

// FlannelPolicyAdapter translates the phase-2 DSL into standard Kubernetes NetworkPolicy objects.
type FlannelPolicyAdapter struct {
	NetpolEnabled              bool
	PolicyEnforcementAvailable bool
}

// FlannelTranslationError reports Flannel-specific policy guard failures.
type FlannelTranslationError struct {
	Code       string
	Field      string
	Message    string
	Suggestion string
}

func (e *FlannelTranslationError) Error() string {
	if e == nil {
		return ""
	}
	message := e.Message
	if e.Field != "" {
		message = fmt.Sprintf("%s: %s", e.Field, message)
	}
	if e.Suggestion != "" {
		return fmt.Sprintf("%s (suggestion: %s)", message, e.Suggestion)
	}
	return message
}

// ToNetworkPolicy translates the DSL into a standard networking.k8s.io/v1 NetworkPolicy.
func (a FlannelPolicyAdapter) ToNetworkPolicy(def *PolicyDefinition) (networkingv1.NetworkPolicy, []PolicyWarning, error) {
	if err := a.ensureNetpolEnabled("cluster.flannel.netpol.enabled", "Flannel network policy controller is not enabled"); err != nil {
		return networkingv1.NetworkPolicy{}, nil, err
	}

	policyDef := normalizedPolicyDefinition(def)
	if err := validateFlannelPolicyDefinition(policyDef); err != nil {
		return networkingv1.NetworkPolicy{}, nil, err
	}

	policy := networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: flannelNetworkPolicyAPIVersion,
			Kind:       flannelNetworkPolicyKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      policyDef.Metadata.Name,
			Namespace: policyDef.Metadata.Namespace,
			Labels:    cloneStringMap(policyDef.Metadata.Labels),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: translateFlannelLabelSelector(policyDef.Spec.Target.PodSelector),
			PolicyTypes: translateFlannelPolicyTypes(policyDef.Spec.PolicyTypes),
			Ingress:     make([]networkingv1.NetworkPolicyIngressRule, 0, len(policyDef.Spec.Ingress)),
			Egress:      make([]networkingv1.NetworkPolicyEgressRule, 0, len(policyDef.Spec.Egress)),
		},
	}

	for _, rule := range policyDef.Spec.Ingress {
		policy.Spec.Ingress = append(policy.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From:  translateFlannelPeer(rule.From),
			Ports: translateFlannelPorts(rule.Ports),
		})
	}
	for _, rule := range policyDef.Spec.Egress {
		policy.Spec.Egress = append(policy.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
			To:    translateFlannelPeer(rule.To),
			Ports: translateFlannelPorts(rule.Ports),
		})
	}

	warnings := []PolicyWarning{{
		Code:    PolicyErrorFlannelOnlyStandardNP,
		Message: "Flannel adapter only supports the standard Kubernetes NetworkPolicy subset",
	}}

	return policy, warnings, nil
}

// EnsurePublishable blocks release when Flannel cannot actually enforce network policy.
func (a FlannelPolicyAdapter) EnsurePublishable() error {
	if err := a.ensureNetpolEnabled("publish.flannel.policyEnforcement", "publish blocked because Flannel network policy enforcement is unavailable"); err != nil {
		return err
	}
	if a.PolicyEnforcementAvailable {
		return nil
	}
	return &FlannelTranslationError{
		Code:       PolicyErrorFlannelNetpolDisabled,
		Field:      "publish.flannel.policyEnforcement",
		Message:    "publish blocked because Flannel network policy enforcement is unavailable",
		Suggestion: flannelEnableSuggestion,
	}
}

func (a FlannelPolicyAdapter) ensureNetpolEnabled(field string, message string) error {
	if a.NetpolEnabled {
		return nil
	}
	return &FlannelTranslationError{
		Code:       PolicyErrorFlannelNetpolDisabled,
		Field:      field,
		Message:    message,
		Suggestion: flannelEnableSuggestion,
	}
}

func validateFlannelPolicyDefinition(def PolicyDefinition) error {
	if def.Spec.Advanced.Order != nil {
		return unsupportedFlannelField("spec.advanced.order", "advanced.order is not supported by standard Kubernetes NetworkPolicy")
	}
	if def.Spec.Advanced.DoNotTrack {
		return unsupportedFlannelField("spec.advanced.doNotTrack", "advanced.doNotTrack is not supported by standard Kubernetes NetworkPolicy")
	}
	if def.Spec.Advanced.ApplyOnForward {
		return unsupportedFlannelField("spec.advanced.applyOnForward", "advanced.applyOnForward is not supported by standard Kubernetes NetworkPolicy")
	}
	for idx, policyType := range def.Spec.PolicyTypes {
		switch policyType {
		case PolicyTypeIngress, PolicyTypeEgress:
		default:
			return unsupportedFlannelField(
				fmt.Sprintf("spec.policyTypes[%d]", idx),
				fmt.Sprintf("unsupported policy type %q for standard Kubernetes NetworkPolicy", policyType),
			)
		}
	}

	for idx, rule := range def.Spec.Ingress {
		if err := validateFlannelIngressRule(rule, idx); err != nil {
			return err
		}
	}
	for idx, rule := range def.Spec.Egress {
		if err := validateFlannelEgressRule(rule, idx); err != nil {
			return err
		}
	}
	return nil
}

func validateFlannelIngressRule(rule PolicyIngressRule, index int) error {
	if rule.Action != "" && rule.Action != PolicyRuleActionAllow {
		return unsupportedFlannelField(
			fmt.Sprintf("spec.ingress[%d].action", index),
			"deny actions are not supported by standard Kubernetes NetworkPolicy",
		)
	}
	if len(rule.HTTP) > 0 {
		return unsupportedFlannelL7Field(fmt.Sprintf("spec.ingress[%d].http", index))
	}
	if len(rule.DNS) > 0 {
		return unsupportedFlannelL7Field(fmt.Sprintf("spec.ingress[%d].dns", index))
	}
	return validateFlannelPeer(rule.From, fmt.Sprintf("spec.ingress[%d].from", index))
}

func validateFlannelEgressRule(rule PolicyEgressRule, index int) error {
	if rule.Action != "" && rule.Action != PolicyRuleActionAllow {
		return unsupportedFlannelField(
			fmt.Sprintf("spec.egress[%d].action", index),
			"deny actions are not supported by standard Kubernetes NetworkPolicy",
		)
	}
	return validateFlannelPeer(rule.To, fmt.Sprintf("spec.egress[%d].to", index))
}

func validateFlannelPeer(peer *PolicyPeer, fieldPath string) error {
	if peer == nil {
		return nil
	}
	if peer.ServiceAccount != "" {
		return unsupportedFlannelField(fieldPath+".serviceAccount", "serviceAccount is not supported by standard Kubernetes NetworkPolicy")
	}
	if peer.FQDN != "" {
		return unsupportedFlannelField(fieldPath+".fqdn", "fqdn is not supported by standard Kubernetes NetworkPolicy")
	}
	return nil
}

func translateFlannelLabelSelector(selector *PolicyLabelSelector) metav1.LabelSelector {
	if selector == nil {
		return metav1.LabelSelector{}
	}

	translated := metav1.LabelSelector{
		MatchLabels: cloneStringMap(selector.MatchLabels),
	}
	if len(selector.MatchExpressions) == 0 {
		return translated
	}

	translated.MatchExpressions = make([]metav1.LabelSelectorRequirement, 0, len(selector.MatchExpressions))
	for _, expr := range selector.MatchExpressions {
		translated.MatchExpressions = append(translated.MatchExpressions, metav1.LabelSelectorRequirement{
			Key:      expr.Key,
			Operator: metav1.LabelSelectorOperator(expr.Operator),
			Values:   cloneStrings(expr.Values),
		})
	}
	return translated
}

func translateFlannelPolicyTypes(types []PolicyType) []networkingv1.PolicyType {
	if len(types) == 0 {
		return nil
	}

	translated := make([]networkingv1.PolicyType, 0, len(types))
	for _, policyType := range types {
		switch policyType {
		case PolicyTypeIngress:
			translated = append(translated, networkingv1.PolicyTypeIngress)
		case PolicyTypeEgress:
			translated = append(translated, networkingv1.PolicyTypeEgress)
		}
	}
	return translated
}

func translateFlannelPeer(peer *PolicyPeer) []networkingv1.NetworkPolicyPeer {
	if peer == nil {
		return nil
	}

	translated := make([]networkingv1.NetworkPolicyPeer, 0, 3)
	if peer.PodSelector != nil || peer.NamespaceSelector != nil {
		combined := networkingv1.NetworkPolicyPeer{}
		if peer.PodSelector != nil {
			combined.PodSelector = labelSelectorPtr(translateFlannelLabelSelector(peer.PodSelector))
		}
		if peer.NamespaceSelector != nil {
			combined.NamespaceSelector = labelSelectorPtr(translateFlannelLabelSelector(peer.NamespaceSelector))
		}
		translated = append(translated, combined)
	}
	if peer.IPBlock != nil {
		translated = append(translated, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR:   peer.IPBlock.CIDR,
				Except: cloneStrings(peer.IPBlock.Except),
			},
		})
	}
	return translated
}

func translateFlannelPorts(ports []PolicyPort) []networkingv1.NetworkPolicyPort {
	if len(ports) == 0 {
		return nil
	}

	translated := make([]networkingv1.NetworkPolicyPort, 0, len(ports))
	for _, port := range ports {
		translatedPort := networkingv1.NetworkPolicyPort{}

		protocol := strings.ToUpper(strings.TrimSpace(port.Protocol))
		switch protocol {
		case "":
		case string(corev1.ProtocolTCP):
			p := corev1.ProtocolTCP
			translatedPort.Protocol = &p
		case string(corev1.ProtocolUDP):
			p := corev1.ProtocolUDP
			translatedPort.Protocol = &p
		case string(corev1.ProtocolSCTP):
			p := corev1.ProtocolSCTP
			translatedPort.Protocol = &p
		default:
			p := corev1.Protocol(protocol)
			translatedPort.Protocol = &p
		}

		if value := port.Port.String(); value != "" {
			if n, ok := parseCanonicalInt(value); ok {
				intOrString := intstr.FromInt32(int32(n))
				translatedPort.Port = &intOrString
			} else {
				intOrString := intstr.FromString(value)
				translatedPort.Port = &intOrString
			}
		}
		if port.EndPort != 0 {
			endPort := port.EndPort
			translatedPort.EndPort = &endPort
		}

		translated = append(translated, translatedPort)
	}
	return translated
}

func labelSelectorPtr(selector metav1.LabelSelector) *metav1.LabelSelector {
	return &selector
}

func unsupportedFlannelL7Field(field string) error {
	RecordCNIAdapterTranslationError("flannel", PolicyErrorFlannelL7NotSupported)
	return &FlannelTranslationError{
		Code:       PolicyErrorFlannelL7NotSupported,
		Field:      field,
		Message:    "L7 rules are not supported by the Flannel adapter",
		Suggestion: "upgrade to Cilium or remove the L7 rules",
	}
}

func unsupportedFlannelField(field string, message string) error {
	RecordCNIAdapterTranslationError("flannel", PolicyErrorFlannelOnlyStandardNP)
	return &FlannelTranslationError{
		Code:       PolicyErrorFlannelOnlyStandardNP,
		Field:      field,
		Message:    message,
		Suggestion: "use the standard Kubernetes NetworkPolicy L3/L4 subset",
	}
}
