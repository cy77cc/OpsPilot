package policy

import (
	"errors"
	"reflect"
	"testing"
)

func TestCiliumAdapterToCiliumPolicyMapsCoreFields(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"allow-api",
		"prod",
		&PolicyTarget{
			PodSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"app":  "api",
					"tier": "backend",
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
				HTTP: []PolicyHTTPRule{
					{Method: "GET", Path: "/healthz"},
				},
			},
			{
				Name:   "allow-dns-lookups",
				Action: PolicyRuleActionAllow,
				Ports: []PolicyPort{
					{Protocol: "ANY", Port: PolicyPortValue("53")},
				},
				DNS: []PolicyDNSRule{
					{MatchPattern: "*.svc.cluster.local"},
				},
			},
		},
		[]PolicyEgressRule{
			{
				Name:   "allow-payments",
				Action: PolicyRuleActionAllow,
				To: &PolicyPeer{
					PodSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"app": "payments",
						},
					},
					NamespaceSelector: &PolicyLabelSelector{
						MatchExpressions: []PolicyLabelSelectorRequirement{
							{
								Key:      "kubernetes.io/metadata.name",
								Operator: "In",
								Values:   []string{"finance"},
							},
						},
					},
					IPBlock: &PolicyIPBlock{
						CIDR: "192.168.0.0/24",
					},
					FQDN: "*.payments.example.com",
				},
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("443")},
				},
			},
		},
	)

	policy, err := ToCiliumPolicy(def)
	if err != nil {
		t.Fatalf("expected translation to succeed, got error: %v", err)
	}

	if policy.APIVersion != "cilium.io/v2" {
		t.Fatalf("expected apiVersion cilium.io/v2, got %q", policy.APIVersion)
	}
	if policy.Kind != "CiliumNetworkPolicy" {
		t.Fatalf("expected kind CiliumNetworkPolicy, got %q", policy.Kind)
	}
	if policy.Metadata.Name != def.Metadata.Name || policy.Metadata.Namespace != def.Metadata.Namespace {
		t.Fatalf("expected metadata to be preserved, got %#v", policy.Metadata)
	}

	expectedEndpointSelector := map[string]string{
		"app":  "api",
		"tier": "backend",
	}
	if !reflect.DeepEqual(policy.Spec.EndpointSelector.MatchLabels, expectedEndpointSelector) {
		t.Fatalf("expected endpoint selector %v, got %v", expectedEndpointSelector, policy.Spec.EndpointSelector.MatchLabels)
	}

	if len(policy.Spec.Ingress) != 2 {
		t.Fatalf("expected 2 ingress rules, got %d", len(policy.Spec.Ingress))
	}
	ingress := policy.Spec.Ingress[0]
	if len(ingress.FromEndpoints) != 1 {
		t.Fatalf("expected 1 ingress fromEndpoints selector, got %d", len(ingress.FromEndpoints))
	}
	expectedIngressPeerLabels := map[string]string{
		"app":                             "web",
		"k8s:io.kubernetes.pod.namespace": "frontend",
	}
	if !reflect.DeepEqual(ingress.FromEndpoints[0].MatchLabels, expectedIngressPeerLabels) {
		t.Fatalf("expected ingress peer labels %v, got %v", expectedIngressPeerLabels, ingress.FromEndpoints[0].MatchLabels)
	}
	expectedIngressCIDR := []CiliumCIDRRule{{CIDR: "10.10.0.0/16", Except: []string{"10.10.1.0/24"}}}
	if !reflect.DeepEqual(ingress.FromCIDRSet, expectedIngressCIDR) {
		t.Fatalf("expected ingress CIDR set %v, got %v", expectedIngressCIDR, ingress.FromCIDRSet)
	}
	if len(ingress.ToPorts) != 1 {
		t.Fatalf("expected 1 ingress toPorts entry, got %d", len(ingress.ToPorts))
	}
	if got := ingress.ToPorts[0].Ports; !reflect.DeepEqual(got, []CiliumPortProtocol{{Port: "8080", Protocol: "TCP"}}) {
		t.Fatalf("expected ingress ports %#v, got %#v", []CiliumPortProtocol{{Port: "8080", Protocol: "TCP"}}, got)
	}
	if got := ingress.ToPorts[0].Rules; got == nil {
		t.Fatalf("expected ingress L7 rules to be present")
	} else {
		if !reflect.DeepEqual(got.HTTP, []CiliumHTTPRule{{Method: "GET", Path: "/healthz"}}) {
			t.Fatalf("expected ingress HTTP rules %#v, got %#v", []CiliumHTTPRule{{Method: "GET", Path: "/healthz"}}, got.HTTP)
		}
		if len(got.DNS) != 0 {
			t.Fatalf("expected HTTP-only toPorts rules, got DNS rules %#v", got.DNS)
		}
	}

	dnsIngress := policy.Spec.Ingress[1]
	if len(dnsIngress.ToPorts) != 1 {
		t.Fatalf("expected 1 DNS ingress toPorts entry, got %d", len(dnsIngress.ToPorts))
	}
	if got := dnsIngress.ToPorts[0].Ports; !reflect.DeepEqual(got, []CiliumPortProtocol{{Port: "53", Protocol: "ANY"}}) {
		t.Fatalf("expected DNS ingress ports %#v, got %#v", []CiliumPortProtocol{{Port: "53", Protocol: "ANY"}}, got)
	}
	if got := dnsIngress.ToPorts[0].Rules; got == nil {
		t.Fatalf("expected DNS ingress rules to be present")
	} else {
		if !reflect.DeepEqual(got.DNS, []CiliumDNSRule{{MatchPattern: "*.svc.cluster.local"}}) {
			t.Fatalf("expected ingress DNS rules %#v, got %#v", []CiliumDNSRule{{MatchPattern: "*.svc.cluster.local"}}, got.DNS)
		}
		if len(got.HTTP) != 0 {
			t.Fatalf("expected DNS-only toPorts rules, got HTTP rules %#v", got.HTTP)
		}
	}

	if len(policy.Spec.Egress) != 1 {
		t.Fatalf("expected 1 egress rule, got %d", len(policy.Spec.Egress))
	}
	egress := policy.Spec.Egress[0]
	if len(egress.ToEndpoints) != 1 {
		t.Fatalf("expected 1 egress toEndpoints selector, got %d", len(egress.ToEndpoints))
	}
	if len(egress.ToEndpoints[0].MatchExpressions) != 1 {
		t.Fatalf("expected translated namespace matchExpression, got %#v", egress.ToEndpoints[0].MatchExpressions)
	}
	if got := egress.ToEndpoints[0].MatchExpressions[0]; got.Key != "k8s:io.kubernetes.pod.namespace" || got.Operator != "In" || !reflect.DeepEqual(got.Values, []string{"finance"}) {
		t.Fatalf("unexpected egress namespace expression: %#v", got)
	}
	if got := egress.ToEndpoints[0].MatchLabels["app"]; got != "payments" {
		t.Fatalf("expected egress pod selector label app=payments, got %q", got)
	}
	expectedEgressCIDR := []CiliumCIDRRule{{CIDR: "192.168.0.0/24"}}
	if !reflect.DeepEqual(egress.ToCIDRSet, expectedEgressCIDR) {
		t.Fatalf("expected egress CIDR set %v, got %v", expectedEgressCIDR, egress.ToCIDRSet)
	}
	if !reflect.DeepEqual(egress.ToFQDNs, []CiliumFQDNSelector{{MatchPattern: "*.payments.example.com"}}) {
		t.Fatalf("expected translated FQDN selector, got %#v", egress.ToFQDNs)
	}
	if len(egress.ToPorts) != 1 {
		t.Fatalf("expected 1 egress toPorts entry, got %d", len(egress.ToPorts))
	}
	if got := egress.ToPorts[0].Ports; !reflect.DeepEqual(got, []CiliumPortProtocol{{Port: "443", Protocol: "TCP"}}) {
		t.Fatalf("expected egress ports %#v, got %#v", []CiliumPortProtocol{{Port: "443", Protocol: "TCP"}}, got)
	}
}

func TestCiliumAdapterToCiliumClusterwidePolicyMapsCoreFields(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"clusterwide-gateway",
		"prod",
		&PolicyTarget{
			PodSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"role": "gateway",
				},
			},
			NamespaceSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"name": "shared-services",
				},
			},
		},
		[]PolicyIngressRule{
			{
				Name:   "allow-frontend-health",
				Action: PolicyRuleActionAllow,
				From: &PolicyPeer{
					PodSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"app": "frontend",
						},
					},
					NamespaceSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "frontend",
						},
					},
				},
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("8443")},
				},
				HTTP: []PolicyHTTPRule{
					{Method: "GET", Path: "/readyz"},
				},
			},
		},
		[]PolicyEgressRule{
			{
				Name:   "allow-api",
				Action: PolicyRuleActionAllow,
				To: &PolicyPeer{
					FQDN: "api.example.com",
				},
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("443")},
				},
			},
		},
	)
	def.Metadata.Labels = map[string]string{"team": "platform"}

	policy, err := ToCiliumClusterwidePolicy(def)
	if err != nil {
		t.Fatalf("expected clusterwide translation to succeed, got error: %v", err)
	}

	if policy.APIVersion != "cilium.io/v2" {
		t.Fatalf("expected apiVersion cilium.io/v2, got %q", policy.APIVersion)
	}
	if policy.Kind != "CiliumClusterwideNetworkPolicy" {
		t.Fatalf("expected kind CiliumClusterwideNetworkPolicy, got %q", policy.Kind)
	}
	if policy.Metadata.Name != def.Metadata.Name {
		t.Fatalf("expected metadata name %q, got %q", def.Metadata.Name, policy.Metadata.Name)
	}
	if policy.Metadata.Namespace != "" {
		t.Fatalf("expected clusterwide policy metadata namespace to be empty, got %q", policy.Metadata.Namespace)
	}
	if !reflect.DeepEqual(policy.Metadata.Labels, def.Metadata.Labels) {
		t.Fatalf("expected metadata labels %#v, got %#v", def.Metadata.Labels, policy.Metadata.Labels)
	}

	expectedEndpointSelector := map[string]string{
		"role":                            "gateway",
		"k8s:io.kubernetes.pod.namespace": "shared-services",
	}
	if !reflect.DeepEqual(policy.Spec.EndpointSelector.MatchLabels, expectedEndpointSelector) {
		t.Fatalf("expected endpoint selector %v, got %v", expectedEndpointSelector, policy.Spec.EndpointSelector.MatchLabels)
	}

	if len(policy.Spec.Ingress) != 1 {
		t.Fatalf("expected 1 ingress rule, got %d", len(policy.Spec.Ingress))
	}
	ingress := policy.Spec.Ingress[0]
	if len(ingress.FromEndpoints) != 1 {
		t.Fatalf("expected 1 ingress fromEndpoints selector, got %d", len(ingress.FromEndpoints))
	}
	expectedIngressPeerLabels := map[string]string{
		"app":                             "frontend",
		"k8s:io.kubernetes.pod.namespace": "frontend",
	}
	if !reflect.DeepEqual(ingress.FromEndpoints[0].MatchLabels, expectedIngressPeerLabels) {
		t.Fatalf("expected ingress peer labels %v, got %v", expectedIngressPeerLabels, ingress.FromEndpoints[0].MatchLabels)
	}
	if len(ingress.ToPorts) != 1 {
		t.Fatalf("expected 1 ingress toPorts entry, got %d", len(ingress.ToPorts))
	}
	if got := ingress.ToPorts[0].Ports; !reflect.DeepEqual(got, []CiliumPortProtocol{{Port: "8443", Protocol: "TCP"}}) {
		t.Fatalf("expected ingress ports %#v, got %#v", []CiliumPortProtocol{{Port: "8443", Protocol: "TCP"}}, got)
	}
	if got := ingress.ToPorts[0].Rules; got == nil {
		t.Fatalf("expected ingress L7 rules to be present")
	} else if !reflect.DeepEqual(got.HTTP, []CiliumHTTPRule{{Method: "GET", Path: "/readyz"}}) {
		t.Fatalf("expected ingress HTTP rules %#v, got %#v", []CiliumHTTPRule{{Method: "GET", Path: "/readyz"}}, got.HTTP)
	}

	if len(policy.Spec.Egress) != 1 {
		t.Fatalf("expected 1 egress rule, got %d", len(policy.Spec.Egress))
	}
	egress := policy.Spec.Egress[0]
	if !reflect.DeepEqual(egress.ToFQDNs, []CiliumFQDNSelector{{MatchName: "api.example.com"}}) {
		t.Fatalf("expected translated FQDN selector, got %#v", egress.ToFQDNs)
	}
	if len(egress.ToPorts) != 1 {
		t.Fatalf("expected 1 egress toPorts entry, got %d", len(egress.ToPorts))
	}
	if got := egress.ToPorts[0].Ports; !reflect.DeepEqual(got, []CiliumPortProtocol{{Port: "443", Protocol: "TCP"}}) {
		t.Fatalf("expected egress ports %#v, got %#v", []CiliumPortProtocol{{Port: "443", Protocol: "TCP"}}, got)
	}
}

func TestCiliumAdapterUnsupportedFieldServiceAccount(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"reject-service-account",
		"prod",
		nil,
		[]PolicyIngressRule{
			{
				Name:   "bad-peer",
				Action: PolicyRuleActionAllow,
				From: &PolicyPeer{
					ServiceAccount: "payments",
				},
			},
		},
		nil,
	)

	_, err := ToCiliumPolicy(def)
	if err == nil {
		t.Fatalf("expected serviceAccount usage to be rejected")
	}

	var translationErr *CiliumTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected CiliumTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorCNISemanticGap {
		t.Fatalf("expected error code %q, got %q", PolicyErrorCNISemanticGap, translationErr.Code)
	}
	if translationErr.Field != "spec.ingress[0].from.serviceAccount" {
		t.Fatalf("expected field spec.ingress[0].from.serviceAccount, got %q", translationErr.Field)
	}
}

func TestCiliumAdapterUnsupportedFieldOrder(t *testing.T) {
	order := 10.0
	def := mustPolicyDefinition(
		t,
		"reject-order",
		"prod",
		nil,
		nil,
		nil,
	)
	def.Spec.Advanced.Order = &order

	_, err := CiliumPolicyAdapter{}.ToCiliumPolicy(def)
	if err == nil {
		t.Fatalf("expected advanced.order usage to be rejected")
	}

	var translationErr *CiliumTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected CiliumTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorCNISemanticGap {
		t.Fatalf("expected error code %q, got %q", PolicyErrorCNISemanticGap, translationErr.Code)
	}
	if translationErr.Field != "spec.advanced.order" {
		t.Fatalf("expected field spec.advanced.order, got %q", translationErr.Field)
	}
}

func TestCiliumAdapterUnsupportedFieldEgressServiceAccount(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"reject-egress-service-account",
		"prod",
		nil,
		nil,
		[]PolicyEgressRule{
			{
				Name:   "bad-egress-peer",
				Action: PolicyRuleActionAllow,
				To: &PolicyPeer{
					ServiceAccount: "payments",
				},
			},
		},
	)

	_, err := ToCiliumPolicy(def)
	if err == nil {
		t.Fatalf("expected egress serviceAccount usage to be rejected")
	}

	var translationErr *CiliumTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected CiliumTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorCNISemanticGap {
		t.Fatalf("expected error code %q, got %q", PolicyErrorCNISemanticGap, translationErr.Code)
	}
	if translationErr.Field != "spec.egress[0].to.serviceAccount" {
		t.Fatalf("expected field spec.egress[0].to.serviceAccount, got %q", translationErr.Field)
	}
}

func TestCiliumAdapterToCiliumPolicyPreservesTargetNamespaceSelector(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"allow-cross-namespace",
		"prod",
		&PolicyTarget{
			PodSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"app": "api",
				},
			},
			NamespaceSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": "payments",
				},
				MatchExpressions: []PolicyLabelSelectorRequirement{
					{
						Key:      "name",
						Operator: "In",
						Values:   []string{"payments"},
					},
				},
			},
		},
		nil,
		nil,
	)

	policy, err := ToCiliumPolicy(def)
	if err != nil {
		t.Fatalf("expected translation to succeed, got error: %v", err)
	}

	expectedLabels := map[string]string{
		"app":                             "api",
		"k8s:io.kubernetes.pod.namespace": "payments",
	}
	if !reflect.DeepEqual(policy.Spec.EndpointSelector.MatchLabels, expectedLabels) {
		t.Fatalf("expected endpoint selector labels %v, got %v", expectedLabels, policy.Spec.EndpointSelector.MatchLabels)
	}

	expectedExpressions := []CiliumLabelSelectorRequirement{
		{
			Key:      "k8s:io.kubernetes.pod.namespace",
			Operator: "In",
			Values:   []string{"payments"},
		},
	}
	if !reflect.DeepEqual(policy.Spec.EndpointSelector.MatchExpressions, expectedExpressions) {
		t.Fatalf("expected endpoint selector expressions %#v, got %#v", expectedExpressions, policy.Spec.EndpointSelector.MatchExpressions)
	}
}
