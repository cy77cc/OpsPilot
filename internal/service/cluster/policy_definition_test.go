package cluster

import (
	"encoding/json"
	"reflect"
	"testing"

	yamlv3 "gopkg.in/yaml.v3"
)

func TestPolicyDefinitionApplyDefaults(t *testing.T) {
	def := PolicyDefinition{}

	def.ApplyDefaults()

	if def.APIVersion != PolicyDefinitionAPIVersion {
		t.Fatalf("expected apiVersion %q, got %q", PolicyDefinitionAPIVersion, def.APIVersion)
	}
	if def.Kind != PolicyDefinitionKind {
		t.Fatalf("expected kind %q, got %q", PolicyDefinitionKind, def.Kind)
	}
	if def.Spec.PolicyTypes == nil {
		t.Fatalf("expected policyTypes to be initialized")
	}
	if def.Spec.Ingress == nil {
		t.Fatalf("expected ingress to be initialized")
	}
	if def.Spec.Egress == nil {
		t.Fatalf("expected egress to be initialized")
	}
}

func TestPolicyDefinitionJSONRoundTrip(t *testing.T) {
	input := []byte(`{
		"metadata": {"name": "allow-web", "namespace": "prod"},
		"spec": {
			"target": {
				"podSelector": {"matchLabels": {"app": "web"}}
			},
			"policyTypes": ["Ingress", "Egress"],
			"ingress": [{
				"name": "allow-api",
				"action": "Allow",
				"from": {
					"namespaceSelector": {"matchLabels": {"name": "api"}}
				},
				"ports": [{
					"protocol": "TCP",
					"port": 80,
					"endPort": 90
				}],
				"http": [{
					"method": "GET",
					"path": "/healthz"
				}],
				"dns": [{
					"matchPattern": "*.svc.cluster.local"
				}]
			}],
			"egress": [{
				"name": "allow-external",
				"action": "Allow",
				"to": {
					"fqdn": "api.example.com"
				},
				"ports": [{
					"protocol": "TCP",
					"port": "https"
				}]
			}],
			"advanced": {
				"order": 10,
				"doNotTrack": true
			}
		}
	}`)

	var def PolicyDefinition
	if err := json.Unmarshal(input, &def); err != nil {
		t.Fatalf("unexpected json unmarshal error: %v", err)
	}

	def.ApplyDefaults()

	if def.Spec.Ingress[0].Ports[0].Port.String() != "80" {
		t.Fatalf("expected numeric port to normalize to string 80, got %q", def.Spec.Ingress[0].Ports[0].Port.String())
	}
	if def.Spec.Egress[0].Ports[0].Port.String() != "https" {
		t.Fatalf("expected named port preserved, got %q", def.Spec.Egress[0].Ports[0].Port.String())
	}
	if def.Spec.Advanced.Order == nil || *def.Spec.Advanced.Order != 10 {
		t.Fatalf("expected advanced order 10, got %#v", def.Spec.Advanced.Order)
	}

	encoded, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("unexpected json marshal error: %v", err)
	}

	var roundTrip map[string]any
	if err := json.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("unexpected json round-trip error: %v", err)
	}

	spec := roundTrip["spec"].(map[string]any)
	ingress := spec["ingress"].([]any)
	ports := ingress[0].(map[string]any)["ports"].([]any)
	if portValue, ok := ports[0].(map[string]any)["port"].(float64); !ok || portValue != 80 {
		t.Fatalf("expected numeric port to marshal as 80, got %#v", ports[0].(map[string]any)["port"])
	}
	egress := spec["egress"].([]any)
	egressPorts := egress[0].(map[string]any)["ports"].([]any)
	if egressPorts[0].(map[string]any)["port"] != "https" {
		t.Fatalf("expected named port to marshal as https, got %#v", egressPorts[0].(map[string]any)["port"])
	}
}

func TestPolicyDefinitionYAMLRoundTrip(t *testing.T) {
	input := []byte(`
metadata:
  name: allow-web
  namespace: prod
spec:
  target:
    podSelector:
      matchLabels:
        app: web
  policyTypes:
    - Ingress
  ingress:
    - name: allow-api
      action: Allow
      from:
        ipBlock:
          cidr: 10.0.0.0/24
          except:
            - 10.0.0.8/32
      ports:
        - protocol: TCP
          port: 443
      http:
        - method: GET
          path: /readyz
  egress:
    - name: allow-dns
      action: Allow
      to:
        fqdn: api.example.com
      ports:
        - protocol: UDP
          port: domain
  advanced:
    applyOnForward: true
`)

	var def PolicyDefinition
	if err := yamlv3.Unmarshal(input, &def); err != nil {
		t.Fatalf("unexpected yaml unmarshal error: %v", err)
	}

	def.ApplyDefaults()

	if def.APIVersion != PolicyDefinitionAPIVersion {
		t.Fatalf("expected default apiVersion %q, got %q", PolicyDefinitionAPIVersion, def.APIVersion)
	}
	if def.Spec.Ingress[0].From == nil || def.Spec.Ingress[0].From.IPBlock == nil {
		t.Fatalf("expected ingress from ipBlock to be present")
	}
	if def.Spec.Ingress[0].Ports[0].Port.String() != "443" {
		t.Fatalf("expected yaml numeric port to normalize to string 443, got %q", def.Spec.Ingress[0].Ports[0].Port.String())
	}
	if def.Spec.Egress[0].Ports[0].Port.String() != "domain" {
		t.Fatalf("expected yaml named port preserved, got %q", def.Spec.Egress[0].Ports[0].Port.String())
	}

	encoded, err := yamlv3.Marshal(def)
	if err != nil {
		t.Fatalf("unexpected yaml marshal error: %v", err)
	}

	var roundTrip PolicyDefinition
	if err := yamlv3.Unmarshal(encoded, &roundTrip); err != nil {
		t.Fatalf("unexpected yaml round-trip error: %v", err)
	}
	if !reflect.DeepEqual(def, roundTrip) {
		t.Fatalf("expected yaml round-trip to preserve definition\nexpected: %#v\nactual: %#v", def, roundTrip)
	}
}

func TestPolicyStateContractAndDefaults(t *testing.T) {
	expectedStates := []PolicyReleaseState{
		PolicyReleaseStateDraft,
		PolicyReleaseStateSimulationPending,
		PolicyReleaseStateSimulationPassed,
		PolicyReleaseStateSimulationFailed,
		PolicyReleaseStateApprovalRequired,
		PolicyReleaseStateApplying,
		PolicyReleaseStateApprovalRejected,
		PolicyReleaseStateApplied,
		PolicyReleaseStateApplyFailed,
		PolicyReleaseStateActive,
		PolicyReleaseStateRollbackApplied,
	}

	var actualStates []PolicyReleaseState
	for _, state := range AllPolicyReleaseStates() {
		actualStates = append(actualStates, state)
	}
	if !reflect.DeepEqual(expectedStates, actualStates) {
		t.Fatalf("expected policy release states %v, got %v", expectedStates, actualStates)
	}

	status := PolicyReleaseStatus{}
	status.ApplyDefaults()
	if status.Phase != PolicyReleaseStateDraft {
		t.Fatalf("expected default phase %q, got %q", PolicyReleaseStateDraft, status.Phase)
	}
	if status.RiskLevel != PolicyRiskLevelLow {
		t.Fatalf("expected default risk level %q, got %q", PolicyRiskLevelLow, status.RiskLevel)
	}
}

func TestPolicyErrorCodeContract(t *testing.T) {
	expected := []string{
		PolicyErrorSimulationBlockingConflict,
		PolicyErrorCNISemanticGap,
		PolicyErrorApprovalTokenInvalid,
		PolicyErrorApplyValidationFailed,
		PolicyErrorFlannelNetpolDisabled,
		PolicyErrorFlannelL7NotSupported,
		PolicyErrorFlannelOnlyStandardNP,
		PolicyErrorCriticalNamespaceBlocked,
		PolicyWarningCNICapabilityDowngrade,
		PolicyWarningImpactScopeExpanded,
		PolicyWarningNonCriticalNamespaceChange,
		PolicyWarningL7RuleSimplified,
	}

	if !reflect.DeepEqual(expected, AllPolicyErrorCodes()) {
		t.Fatalf("expected policy error codes %v, got %v", expected, AllPolicyErrorCodes())
	}
}
