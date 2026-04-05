package cluster

import (
	"errors"
	"reflect"
	"testing"
)

func TestCalicoAdapterSelectorConversion(t *testing.T) {
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
				MatchExpressions: []PolicyLabelSelectorRequirement{
					{
						Key:      "debug",
						Operator: "DoesNotExist",
					},
					{
						Key:      "env",
						Operator: "In",
						Values:   []string{"prod", "staging"},
					},
					{
						Key:      "version",
						Operator: "Exists",
					},
					{
						Key:      "zone",
						Operator: "NotIn",
						Values:   []string{"deprecated"},
					},
				},
			},
		},
		nil,
		nil,
	)

	policy, warnings, err := ToCalicoPolicy(def)
	if err != nil {
		t.Fatalf("expected translation to succeed, got error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	if policy.APIVersion != "crd.projectcalico.org/v1" {
		t.Fatalf("expected apiVersion crd.projectcalico.org/v1, got %q", policy.APIVersion)
	}
	if policy.Kind != "NetworkPolicy" {
		t.Fatalf("expected kind NetworkPolicy, got %q", policy.Kind)
	}
	if policy.Metadata.Name != def.Metadata.Name || policy.Metadata.Namespace != def.Metadata.Namespace {
		t.Fatalf("expected metadata to be preserved, got %#v", policy.Metadata)
	}

	expectedSelector := `app == "api" && tier == "backend" && ! has(debug) && env in {"prod", "staging"} && has(version) && zone not in {"deprecated"}`
	if policy.Spec.Selector != expectedSelector {
		t.Fatalf("expected selector %q, got %q", expectedSelector, policy.Spec.Selector)
	}
	if !reflect.DeepEqual(policy.Spec.Types, def.Spec.PolicyTypes) {
		t.Fatalf("expected types %#v, got %#v", def.Spec.PolicyTypes, policy.Spec.Types)
	}
}

func TestCalicoAdapterOrderMapping(t *testing.T) {
	order := 10.5
	def := mustPolicyDefinition(
		t,
		"allow-web",
		"prod",
		&PolicyTarget{
			PodSelector: &PolicyLabelSelector{
				MatchLabels: map[string]string{
					"app": "web",
				},
			},
		},
		nil,
		nil,
	)
	def.Spec.Advanced.Order = &order
	def.Spec.Advanced.DoNotTrack = true
	def.Spec.Advanced.ApplyOnForward = true

	policy, warnings, err := ToCalicoPolicy(def)
	if err != nil {
		t.Fatalf("expected translation to succeed, got error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if policy.Spec.Order == nil || *policy.Spec.Order != order {
		t.Fatalf("expected order %v, got %#v", order, policy.Spec.Order)
	}
	if !policy.Spec.DoNotTrack {
		t.Fatalf("expected doNotTrack to be true")
	}
	if !policy.Spec.ApplyOnForward {
		t.Fatalf("expected applyOnForward to be true")
	}
}

func TestCalicoAdapterSemanticGapFQDN(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"reject-fqdn",
		"prod",
		nil,
		nil,
		[]PolicyEgressRule{
			{
				Name:   "allow-external-api",
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

	_, _, err := ToCalicoPolicy(def)
	if err == nil {
		t.Fatalf("expected fqdn usage to be rejected")
	}

	var translationErr *CalicoTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected CalicoTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorCNISemanticGap {
		t.Fatalf("expected error code %q, got %q", PolicyErrorCNISemanticGap, translationErr.Code)
	}
	if translationErr.Field != "spec.egress[0].to.fqdn" {
		t.Fatalf("expected field spec.egress[0].to.fqdn, got %q", translationErr.Field)
	}
}

func TestCalicoAdapterSemanticGapUnsupportedSelectorOperator(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"reject-unknown-operator",
		"prod",
		&PolicyTarget{
			PodSelector: &PolicyLabelSelector{
				MatchExpressions: []PolicyLabelSelectorRequirement{
					{
						Key:      "version",
						Operator: "Gt",
						Values:   []string{"2"},
					},
				},
			},
		},
		nil,
		nil,
	)

	_, _, err := ToCalicoPolicy(def)
	if err == nil {
		t.Fatalf("expected unsupported selector operator to be rejected")
	}

	var translationErr *CalicoTranslationError
	if !errors.As(err, &translationErr) {
		t.Fatalf("expected CalicoTranslationError, got %T", err)
	}
	if translationErr.Code != PolicyErrorCNISemanticGap {
		t.Fatalf("expected error code %q, got %q", PolicyErrorCNISemanticGap, translationErr.Code)
	}
	if translationErr.Field != "spec.target.podSelector.matchExpressions[0].operator" {
		t.Fatalf("expected field spec.target.podSelector.matchExpressions[0].operator, got %q", translationErr.Field)
	}
}

func TestCalicoAdapterWarningOnL7Simplification(t *testing.T) {
	def := mustPolicyDefinition(
		t,
		"degrade-http",
		"prod",
		nil,
		[]PolicyIngressRule{
			{
				Name:   "allow-health",
				Action: PolicyRuleActionAllow,
				From: &PolicyPeer{
					PodSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"app": "web",
						},
					},
				},
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

	policy, warnings, err := ToCalicoPolicy(def)
	if err != nil {
		t.Fatalf("expected translation to succeed with warnings, got error: %v", err)
	}

	expectedWarnings := []string{
		PolicyWarningCNICapabilityDowngrade,
		PolicyWarningL7RuleSimplified,
	}
	if got := collectWarningCodes(warnings); !reflect.DeepEqual(got, expectedWarnings) {
		t.Fatalf("expected warnings %v, got %v", expectedWarnings, got)
	}
	if len(policy.Spec.Ingress) != 1 {
		t.Fatalf("expected 1 translated ingress rule, got %d", len(policy.Spec.Ingress))
	}
	if policy.Spec.Ingress[0].Action != "Allow" {
		t.Fatalf("expected action Allow, got %q", policy.Spec.Ingress[0].Action)
	}
	if !reflect.DeepEqual(policy.Spec.Ingress[0].Destination.Ports, []string{"8080"}) {
		t.Fatalf("expected translated ports [8080], got %#v", policy.Spec.Ingress[0].Destination.Ports)
	}
}

func collectWarningCodes(warnings []PolicyWarning) []string {
	if len(warnings) == 0 {
		return nil
	}

	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		codes = append(codes, warning.Code)
	}
	return codes
}
