package cluster

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestFlannelAdapterMapsStandardNetworkPolicySubset(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"allow-api",
		"prod",
		&PolicyTarget{
			PodSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"app": "api",
				},
			},
		},
		[]PolicyIngressRule{
			{
				Name:   "allow-web",
				Action: PolicyRuleActionAllow,
				From: &PolicyPeer{
					PodSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"app": "web",
						},
					},
					NamespaceSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "frontend",
						},
					},
					IPBlock: &PolicyIPBlock{
						CIDR:   "10.10.0.0/16",
						Except: []string{"10.10.1.0/24"},
					},
				},
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("8080")},
				},
			},
		},
		[]PolicyEgressRule{
			{
				Name:   "allow-db",
				Action: PolicyRuleActionAllow,
				To: &PolicyPeer{
					PodSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"app": "db",
						},
					},
					IPBlock: &PolicyIPBlock{
						CIDR: "192.168.0.0/24",
					},
				},
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("5432")},
				},
			},
		},
	)

	policy, warnings, err := FlannelPolicyAdapter{
		NetpolEnabled:              true,
		PolicyEnforcementAvailable: true,
	}.ToNetworkPolicy(def)
	if err != nil {
		t.Fatalf("expected translation to succeed, got error: %v", err)
	}

	if policy.APIVersion != "networking.k8s.io/v1" {
		t.Fatalf("expected apiVersion networking.k8s.io/v1, got %q", policy.APIVersion)
	}
	if policy.Kind != "NetworkPolicy" {
		t.Fatalf("expected kind NetworkPolicy, got %q", policy.Kind)
	}
	if policy.Name != def.Metadata.Name || policy.Namespace != def.Metadata.Namespace {
		t.Fatalf("expected metadata to be preserved, got %#v", policy.ObjectMeta)
	}
	if got := collectWarningCodes(warnings); !reflect.DeepEqual(got, []string{PolicyErrorFlannelOnlyStandardNP}) {
		t.Fatalf("expected standard subset warning, got %v", got)
	}

	expectedTarget := map[string]string{"app": "api"}
	if !reflect.DeepEqual(policy.Spec.PodSelector.MatchLabels, expectedTarget) {
		t.Fatalf("expected podSelector %v, got %v", expectedTarget, policy.Spec.PodSelector.MatchLabels)
	}
	expectedTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress}
	if !reflect.DeepEqual(policy.Spec.PolicyTypes, expectedTypes) {
		t.Fatalf("expected policy types %#v, got %#v", expectedTypes, policy.Spec.PolicyTypes)
	}

	if len(policy.Spec.Ingress) != 1 {
		t.Fatalf("expected 1 ingress rule, got %d", len(policy.Spec.Ingress))
	}
	ingress := policy.Spec.Ingress[0]
	if len(ingress.From) != 2 {
		t.Fatalf("expected 2 ingress peers, got %d", len(ingress.From))
	}
	if got := ingress.From[0].PodSelector; got == nil || !reflect.DeepEqual(got.MatchLabels, map[string]string{"app": "web"}) {
		t.Fatalf("expected first ingress peer pod selector app=web, got %#v", got)
	}
	if got := ingress.From[0].NamespaceSelector; got == nil || !reflect.DeepEqual(got.MatchLabels, map[string]string{"kubernetes.io/metadata.name": "frontend"}) {
		t.Fatalf("expected first ingress peer namespace selector frontend, got %#v", got)
	}
	if got := ingress.From[1].IPBlock; got == nil || got.CIDR != "10.10.0.0/16" || !reflect.DeepEqual(got.Except, []string{"10.10.1.0/24"}) {
		t.Fatalf("expected second ingress peer IPBlock, got %#v", got)
	}
	if len(ingress.Ports) != 1 {
		t.Fatalf("expected 1 ingress port, got %d", len(ingress.Ports))
	}
	if got := ingress.Ports[0].Protocol; got == nil || *got != corev1.ProtocolTCP {
		t.Fatalf("expected ingress protocol TCP, got %#v", got)
	}
	if got := ingress.Ports[0].Port; got == nil || got.String() != "8080" {
		t.Fatalf("expected ingress port 8080, got %#v", got)
	}

	if len(policy.Spec.Egress) != 1 {
		t.Fatalf("expected 1 egress rule, got %d", len(policy.Spec.Egress))
	}
	egress := policy.Spec.Egress[0]
	if len(egress.To) != 2 {
		t.Fatalf("expected 2 egress peers, got %d", len(egress.To))
	}
	if got := egress.To[0].PodSelector; got == nil || !reflect.DeepEqual(got.MatchLabels, map[string]string{"app": "db"}) {
		t.Fatalf("expected first egress peer pod selector app=db, got %#v", got)
	}
	if got := egress.To[1].IPBlock; got == nil || got.CIDR != "192.168.0.0/24" {
		t.Fatalf("expected second egress peer IPBlock, got %#v", got)
	}
	if len(egress.Ports) != 1 {
		t.Fatalf("expected 1 egress port, got %d", len(egress.Ports))
	}
	if got := egress.Ports[0].Port; got == nil || got.String() != "5432" {
		t.Fatalf("expected egress port 5432, got %#v", got)
	}
}

func TestFlannelAdapterNetpolDisabledBlocksTranslation(t *testing.T) {
	def := mustPolicyDefinition(t, "allow-api", "prod", nil, nil, nil)

	_, _, err := FlannelPolicyAdapter{}.ToNetworkPolicy(def)
	if err == nil {
		t.Fatalf("expected translation to be blocked when netpol is disabled")
	}

	var translationErr *FlannelTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected FlannelTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorFlannelNetpolDisabled {
		t.Fatalf("expected error code %q, got %q", PolicyErrorFlannelNetpolDisabled, translationErr.Code)
	}
	if translationErr.Field != "cluster.flannel.netpol.enabled" {
		t.Fatalf("expected field cluster.flannel.netpol.enabled, got %q", translationErr.Field)
	}
	if !strings.Contains(translationErr.Suggestion, "helm upgrade flannel --set netpol.enabled=true") {
		t.Fatalf("expected helm suggestion, got %q", translationErr.Suggestion)
	}
}

func TestFlannelAdapterL7UnsupportedBlocksTranslation(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"allow-health",
		"prod",
		nil,
		[]PolicyIngressRule{
			{
				Name:   "allow-http",
				Action: PolicyRuleActionAllow,
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("8080")},
				},
				HTTP: []PolicyHTTPRule{
					{Method: "GET", Path: "/healthz"},
				},
			},
		},
		nil,
	)

	_, _, err := FlannelPolicyAdapter{
		NetpolEnabled:              true,
		PolicyEnforcementAvailable: true,
	}.ToNetworkPolicy(def)
	if err == nil {
		t.Fatalf("expected L7 rules to be rejected")
	}

	var translationErr *FlannelTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected FlannelTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorFlannelL7NotSupported {
		t.Fatalf("expected error code %q, got %q", PolicyErrorFlannelL7NotSupported, translationErr.Code)
	}
	if translationErr.Field != "spec.ingress[0].http" {
		t.Fatalf("expected field spec.ingress[0].http, got %q", translationErr.Field)
	}
	if !strings.Contains(strings.ToLower(translationErr.Message), "l7") {
		t.Fatalf("expected L7 rejection message, got %q", translationErr.Message)
	}
	if !strings.Contains(strings.ToLower(translationErr.Suggestion), "cilium") {
		t.Fatalf("expected suggestion to mention Cilium, got %q", translationErr.Suggestion)
	}
}

func TestFlannelAdapterRejectsNonStandardDenyAction(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"deny-api",
		"prod",
		nil,
		[]PolicyIngressRule{
			{
				Name:   "deny-web",
				Action: PolicyRuleActionDeny,
			},
		},
		nil,
	)

	_, _, err := FlannelPolicyAdapter{
		NetpolEnabled:              true,
		PolicyEnforcementAvailable: true,
	}.ToNetworkPolicy(def)
	if err == nil {
		t.Fatalf("expected deny action to be rejected")
	}

	var translationErr *FlannelTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected FlannelTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorFlannelOnlyStandardNP {
		t.Fatalf("expected error code %q, got %q", PolicyErrorFlannelOnlyStandardNP, translationErr.Code)
	}
	if translationErr.Field != "spec.ingress[0].action" {
		t.Fatalf("expected field spec.ingress[0].action, got %q", translationErr.Field)
	}
}

func TestFlannelAdapterRejectsInvalidPolicyType(t *testing.T) {
	def := mustPolicyDefinition(t, "allow-api", "prod", nil, nil, nil)
	def.Spec.PolicyTypes = []PolicyType{PolicyTypeIngress, PolicyType("Sideways")}

	_, _, err := FlannelPolicyAdapter{
		NetpolEnabled:              true,
		PolicyEnforcementAvailable: true,
	}.ToNetworkPolicy(def)
	if err == nil {
		t.Fatalf("expected invalid policy type to be rejected")
	}

	var translationErr *FlannelTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected FlannelTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorFlannelOnlyStandardNP {
		t.Fatalf("expected error code %q, got %q", PolicyErrorFlannelOnlyStandardNP, translationErr.Code)
	}
	if translationErr.Field != "spec.policyTypes[1]" {
		t.Fatalf("expected field spec.policyTypes[1], got %q", translationErr.Field)
	}
	if !strings.Contains(strings.ToLower(translationErr.Message), "unsupported policy type") {
		t.Fatalf("expected unsupported policy type message, got %q", translationErr.Message)
	}
}

func TestFlannelAdapterPublishBlockedWhenEnforcementUnavailable(t *testing.T) {
	err := FlannelPolicyAdapter{
		NetpolEnabled:              true,
		PolicyEnforcementAvailable: false,
	}.EnsurePublishable()
	if err == nil {
		t.Fatalf("expected publish guard to block when enforcement is unavailable")
	}

	var translationErr *FlannelTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected FlannelTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorFlannelNetpolDisabled {
		t.Fatalf("expected error code %q, got %q", PolicyErrorFlannelNetpolDisabled, translationErr.Code)
	}
	if translationErr.Field != "publish.flannel.policyEnforcement" {
		t.Fatalf("expected field publish.flannel.policyEnforcement, got %q", translationErr.Field)
	}
	if !strings.Contains(strings.ToLower(translationErr.Message), "publish blocked") {
		t.Fatalf("expected publish blocked message, got %q", translationErr.Message)
	}
}
