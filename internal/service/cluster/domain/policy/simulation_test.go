package policy

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSimulationBlockingConflictForCriticalNamespace(t *testing.T) {
	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "deny-kube-system", "kube-system", nil, []PolicyIngressRule{
			{
				Name:   "deny-all",
				Action: PolicyRuleActionDeny,
			},
		}, nil),
		NamespacePodCounts: map[string]int{
			"kube-system": 3,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if result.Passed {
		t.Fatalf("expected simulation to fail when critical namespace traffic is blocked")
	}
	assertIssueCodes(t, result.BlockingIssues, []string{
		PolicyErrorSimulationBlockingConflict,
		PolicyErrorCriticalNamespaceBlocked,
	})
	if result.RiskScore != 40 {
		t.Fatalf("expected risk score 40 for critical namespace blocking, got %d", result.RiskScore)
	}
	if result.RiskLevel != PolicyRiskLevelHigh {
		t.Fatalf("expected risk level %q, got %q", PolicyRiskLevelHigh, result.RiskLevel)
	}
	if result.ImpactSummary.AffectedPods != 3 {
		t.Fatalf("expected 3 affected pods, got %d", result.ImpactSummary.AffectedPods)
	}
	if !reflect.DeepEqual(result.ImpactSummary.AffectedNamespaces, []string{"kube-system"}) {
		t.Fatalf("expected affected namespaces [kube-system], got %v", result.ImpactSummary.AffectedNamespaces)
	}
	if !reflect.DeepEqual(result.ImpactSummary.NewDeniedFlows, []string{"ingress:deny-all"}) {
		t.Fatalf("expected denied flows [ingress:deny-all], got %v", result.ImpactSummary.NewDeniedFlows)
	}
}

func TestSimulationBlockingConflictForCriticalNamespaceSelectedByMatchExpression(t *testing.T) {
	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "deny-system-by-expression", "prod", nil, []PolicyIngressRule{
			{
				Name:   "deny-kube-system",
				Action: PolicyRuleActionDeny,
				From: &PolicyPeer{
					NamespaceSelector: &PolicyLabelSelector{
						MatchExpressions: []PolicyLabelSelectorRequirement{
							{
								Key:      "kubernetes.io/metadata.name",
								Operator: "In",
								Values:   []string{"kube-system"},
							},
						},
					},
				},
			},
		}, nil),
		NamespacePodCounts: map[string]int{
			"kube-system": 3,
			"prod":        5,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if result.Passed {
		t.Fatalf("expected simulation to fail when a matchExpression selects a critical namespace")
	}
	assertIssueCodes(t, result.BlockingIssues, []string{
		PolicyErrorSimulationBlockingConflict,
		PolicyErrorCriticalNamespaceBlocked,
	})
	expectedNamespaces := []string{"kube-system", "prod"}
	if !reflect.DeepEqual(result.ImpactSummary.AffectedNamespaces, expectedNamespaces) {
		t.Fatalf("expected affected namespaces %v, got %v", expectedNamespaces, result.ImpactSummary.AffectedNamespaces)
	}
}

func TestRiskScoreCriticalThreshold(t *testing.T) {
	score, level := calculatePolicyRiskScore(policyRiskFactors{
		BlocksCriticalNamespace: true,
		HTTPRuleCount:           15,
		RollbackAvailable:       false,
	})

	if score != 70 {
		t.Fatalf("expected risk score 70, got %d", score)
	}
	if level != PolicyRiskLevelCritical {
		t.Fatalf("expected risk level %q, got %q", PolicyRiskLevelCritical, level)
	}
}

func TestSimulationCriticalRiskWithoutBlockingIssuesStillPasses(t *testing.T) {
	if !simulationPassed(nil, PolicyRiskLevelCritical) {
		t.Fatalf("expected critical risk without blocking issues to pass simulation")
	}
	if simulationPassed([]PolicyIssue{{Code: PolicyErrorCNISemanticGap}}, PolicyRiskLevelCritical) {
		t.Fatalf("expected blocking issues to fail simulation even when risk is critical")
	}
}

func TestSimulationWarnings(t *testing.T) {
	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "allow-payments", "payments", nil, []PolicyIngressRule{
			{
				Name:   "allow-api",
				Action: PolicyRuleActionAllow,
				HTTP: []PolicyHTTPRule{
					{Method: "GET", Path: "/healthz"},
				},
				From: &PolicyPeer{
					NamespaceSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"name": "finance",
						},
					},
				},
			},
		}, nil),
		NamespacePodCounts: map[string]int{
			"payments": 4,
			"finance":  8,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if !result.Passed {
		t.Fatalf("expected simulation to pass when only warnings are present")
	}
	assertWarningCodes(t, result.Warnings, []string{
		PolicyWarningImpactScopeExpanded,
		PolicyWarningL7RuleSimplified,
		PolicyWarningNonCriticalNamespaceChange,
	})
	if result.RiskScore != 2 {
		t.Fatalf("expected risk score 2 from one HTTP rule, got %d", result.RiskScore)
	}
	if result.RiskLevel != PolicyRiskLevelLow {
		t.Fatalf("expected risk level %q, got %q", PolicyRiskLevelLow, result.RiskLevel)
	}
}

func TestSimulationImpactSummary(t *testing.T) {
	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "deny-prod", "prod", nil, []PolicyIngressRule{
			{
				Name:   "deny-admin",
				Action: PolicyRuleActionDeny,
				From: &PolicyPeer{
					NamespaceSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"name": "default",
						},
					},
				},
			},
		}, []PolicyEgressRule{
			{
				Name:   "deny-external",
				Action: PolicyRuleActionDeny,
			},
		}),
		NamespacePodCounts: map[string]int{
			"default": 2,
			"prod":    5,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	expectedNamespaces := []string{"default", "prod"}
	if !reflect.DeepEqual(result.ImpactSummary.AffectedNamespaces, expectedNamespaces) {
		t.Fatalf("expected affected namespaces %v, got %v", expectedNamespaces, result.ImpactSummary.AffectedNamespaces)
	}
	if result.ImpactSummary.AffectedPods != 7 {
		t.Fatalf("expected 7 affected pods, got %d", result.ImpactSummary.AffectedPods)
	}
	expectedFlows := []string{"ingress:deny-admin", "egress:deny-external"}
	if !reflect.DeepEqual(result.ImpactSummary.NewDeniedFlows, expectedFlows) {
		t.Fatalf("expected denied flows %v, got %v", expectedFlows, result.ImpactSummary.NewDeniedFlows)
	}
}

func TestSimulationBlockingConflictFromExistingPolicyUsesBlockingSeverity(t *testing.T) {
	target := &PolicyTarget{
		PodSelector: &PolicyLabelSelector{
			MatchLabels: map[string]string{
				"app": "api",
			},
		},
	}

	input := PolicySimulationInput{
		Base: mustPolicyDefinition(t, "allow-api", "prod", target, []PolicyIngressRule{
			{
				Name:   "allow-api",
				Action: PolicyRuleActionAllow,
			},
		}, nil),
		Candidate: mustPolicyDefinition(t, "deny-api", "prod", target, []PolicyIngressRule{
			{
				Name:   "deny-api",
				Action: PolicyRuleActionDeny,
			},
		}, nil),
		ExistingPolicies: []PolicyDefinition{
			*mustPolicyDefinition(t, "allow-api-copy", "prod", target, []PolicyIngressRule{
				{
					Name:   "allow-api-copy",
					Action: PolicyRuleActionAllow,
				},
			}, nil),
		},
		NamespacePodCounts: map[string]int{
			"prod": 6,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if result.Passed {
		t.Fatalf("expected simulation to fail for overlapping conflicting policies")
	}
	if len(result.BlockingIssues) != 1 {
		t.Fatalf("expected exactly one overlap conflict issue, got %d", len(result.BlockingIssues))
	}
	if result.BlockingIssues[0].Code != PolicyErrorSimulationBlockingConflict {
		t.Fatalf("expected issue code %q, got %q", PolicyErrorSimulationBlockingConflict, result.BlockingIssues[0].Code)
	}
	if result.BlockingIssues[0].Severity != PolicyIssueSeverityBlocking {
		t.Fatalf("expected overlap conflict severity %q, got %q", PolicyIssueSeverityBlocking, result.BlockingIssues[0].Severity)
	}
}

func TestSimulationConflictIgnoresNonOverlappingPorts(t *testing.T) {
	target := &PolicyTarget{
		PodSelector: &PolicyLabelSelector{
			MatchLabels: map[string]string{
				"app": "api",
			},
		},
	}

	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "deny-api-443", "prod", target, []PolicyIngressRule{
			{
				Name:   "deny-api-443",
				Action: PolicyRuleActionDeny,
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("443")},
				},
			},
		}, nil),
		ExistingPolicies: []PolicyDefinition{
			*mustPolicyDefinition(t, "allow-api-80", "prod", target, []PolicyIngressRule{
				{
					Name:   "allow-api-80",
					Action: PolicyRuleActionAllow,
					Ports: []PolicyPort{
						{Protocol: "TCP", Port: PolicyPortValue("80")},
					},
				},
			}, nil),
		},
		NamespacePodCounts: map[string]int{
			"prod": 6,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if !result.Passed {
		t.Fatalf("expected simulation to pass when allow/deny rules do not overlap on ports, got issues %v", result.BlockingIssues)
	}
	if len(result.BlockingIssues) != 0 {
		t.Fatalf("expected no blocking issues for non-overlapping ports, got %v", result.BlockingIssues)
	}
}

func TestSimulationConflictIgnoresDifferentPeerScope(t *testing.T) {
	target := &PolicyTarget{
		PodSelector: &PolicyLabelSelector{
			MatchLabels: map[string]string{
				"app": "api",
			},
		},
	}

	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "deny-api-engineering", "prod", target, []PolicyIngressRule{
			{
				Name:   "deny-api-engineering",
				Action: PolicyRuleActionDeny,
				From: &PolicyPeer{
					NamespaceSelector: &PolicyLabelSelector{
						MatchLabels: map[string]string{
							"name": "engineering",
						},
					},
				},
				Ports: []PolicyPort{
					{Protocol: "TCP", Port: PolicyPortValue("80")},
				},
			},
		}, nil),
		ExistingPolicies: []PolicyDefinition{
			*mustPolicyDefinition(t, "allow-api-finance", "prod", target, []PolicyIngressRule{
				{
					Name:   "allow-api-finance",
					Action: PolicyRuleActionAllow,
					From: &PolicyPeer{
						NamespaceSelector: &PolicyLabelSelector{
							MatchLabels: map[string]string{
								"name": "finance",
							},
						},
					},
					Ports: []PolicyPort{
						{Protocol: "TCP", Port: PolicyPortValue("80")},
					},
				},
			}, nil),
		},
		NamespacePodCounts: map[string]int{
			"prod": 6,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if !result.Passed {
		t.Fatalf("expected simulation to pass when allow/deny rules target different peer scopes, got issues %v", result.BlockingIssues)
	}
	if len(result.BlockingIssues) != 0 {
		t.Fatalf("expected no blocking issues for different peer scopes, got %v", result.BlockingIssues)
	}
}

func TestSimulationBlockingConflictIncludesIndependentBlockers(t *testing.T) {
	target := &PolicyTarget{
		PodSelector: &PolicyLabelSelector{
			MatchLabels: map[string]string{
				"app": "api",
			},
		},
	}

	input := PolicySimulationInput{
		Candidate: mustPolicyDefinition(t, "deny-api", "prod", target, []PolicyIngressRule{
			{
				Name:   "deny-api",
				Action: PolicyRuleActionDeny,
			},
		}, nil),
		ExistingPolicies: []PolicyDefinition{
			*mustPolicyDefinition(t, "allow-api", "prod", target, []PolicyIngressRule{
				{
					Name:   "allow-api",
					Action: PolicyRuleActionAllow,
				},
			}, nil),
		},
		CNICapabilityGap: true,
		NamespacePodCounts: map[string]int{
			"prod": 6,
		},
		RollbackAvailable: true,
	}

	result := SimulatePolicy(input)

	if result.Passed {
		t.Fatalf("expected simulation to fail when independent blockers are present")
	}
	assertIssueCodes(t, result.BlockingIssues, []string{
		PolicyErrorCNISemanticGap,
		PolicyErrorSimulationBlockingConflict,
	})
}

func TestSimulationZeroRiskJSONUsesSnakeCaseContract(t *testing.T) {
	result := PolicySimulationResult{
		Passed: true,
		PolicySimulationStatus: PolicySimulationStatus{
			BlockingIssues: []PolicyIssue{
				{Code: PolicyErrorSimulationBlockingConflict},
			},
			Warnings: []PolicyWarning{
				{Code: PolicyWarningImpactScopeExpanded},
			},
			ImpactSummary: PolicyImpactSummary{
				AffectedPods:       2,
				AffectedNamespaces: []string{"prod"},
			},
		},
		PolicyReleaseStatus: PolicyReleaseStatus{
			RiskScore: 0,
			RiskLevel: PolicyRiskLevelLow,
		},
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	expectedKeys := []string{"passed", "blocking_issues", "warnings", "impact_summary", "risk_score", "risk_level"}
	for _, key := range expectedKeys {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected key %q in payload, got %v", key, payload)
		}
	}

	impactSummary, ok := payload["impact_summary"].(map[string]any)
	if !ok {
		t.Fatalf("expected impact_summary object, got %T (%v)", payload["impact_summary"], payload["impact_summary"])
	}

	expectedImpactKeys := []string{"affected_pods", "affected_namespaces", "new_denied_flows"}
	for _, key := range expectedImpactKeys {
		if _, ok := impactSummary[key]; !ok {
			t.Fatalf("expected nested key %q in impact_summary, got %v", key, impactSummary)
		}
	}

	if payload["risk_score"] != float64(0) {
		t.Fatalf("expected risk_score 0 in payload, got %v", payload["risk_score"])
	}
	if payload["risk_level"] != string(PolicyRiskLevelLow) {
		t.Fatalf("expected risk_level %q in payload, got %v", PolicyRiskLevelLow, payload["risk_level"])
	}

	unexpectedKeys := []string{"blockingIssues", "impactSummary", "riskScore", "riskLevel"}
	for _, key := range unexpectedKeys {
		if _, ok := payload[key]; ok {
			t.Fatalf("did not expect camelCase key %q in payload, got %v", key, payload)
		}
	}

	unexpectedImpactKeys := []string{"affectedPods", "affectedNamespaces", "newDeniedFlows"}
	for _, key := range unexpectedImpactKeys {
		if _, ok := impactSummary[key]; ok {
			t.Fatalf("did not expect camelCase nested key %q in impact_summary, got %v", key, impactSummary)
		}
	}
}

func mustPolicyDefinition(
	t *testing.T,
	name string,
	namespace string,
	target *PolicyTarget,
	ingress []PolicyIngressRule,
	egress []PolicyEgressRule,
) *PolicyDefinition {
	t.Helper()

	def := PolicyDefinition{
		Metadata: PolicyObjectMetadata{
			Name:      name,
			Namespace: namespace,
		},
		Spec: PolicyDefinitionSpec{
			Target:      PolicyTarget{},
			PolicyTypes: []PolicyType{PolicyTypeIngress, PolicyTypeEgress},
			Ingress:     ingress,
			Egress:      egress,
		},
	}
	if target != nil {
		def.Spec.Target = *target
	}
	def.ApplyDefaults()
	return &def
}

func assertIssueCodes(t *testing.T, issues []PolicyIssue, expected []string) {
	t.Helper()

	var codes []string
	for _, issue := range issues {
		codes = append(codes, issue.Code)
	}
	if !reflect.DeepEqual(codes, expected) {
		t.Fatalf("expected issue codes %v, got %v", expected, codes)
	}
}

func assertWarningCodes(t *testing.T, warnings []PolicyWarning, expected []string) {
	t.Helper()

	var codes []string
	for _, warning := range warnings {
		codes = append(codes, warning.Code)
	}
	if !reflect.DeepEqual(codes, expected) {
		t.Fatalf("expected warning codes %v, got %v", expected, codes)
	}
}
