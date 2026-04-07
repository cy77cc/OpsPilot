// Package policy 提供集群网络策略的领域模型和业务规则。
//
// 本文件定义 Phase 2 网络策略 DSL、发布状态常量与错误码契约。
package policy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	clustercontracts "github.com/cy77cc/OpsPilot/internal/service/cluster/contracts"
	yamlv3 "gopkg.in/yaml.v3"
)

const (
	// PolicyDefinitionAPIVersion 是统一策略 DSL 的 API 版本。
	PolicyDefinitionAPIVersion = "opspilot.io/v1alpha1"
	// PolicyDefinitionKind 是统一策略 DSL 的 Kind。
	PolicyDefinitionKind = "NetworkPolicyDefinition"
)

// Import shared types from contracts
type PolicyReleaseState = clustercontracts.PolicyReleaseState
type PolicyRiskLevel = clustercontracts.PolicyRiskLevel
type PolicyIssueSeverity = clustercontracts.PolicyIssueSeverity
type PolicyReference = clustercontracts.PolicyReference
type PolicyTargetCluster = clustercontracts.PolicyTargetCluster
type PolicyReleaseStatus = clustercontracts.PolicyReleaseStatus
type PolicyIssue = clustercontracts.PolicyIssue
type PolicyWarning = clustercontracts.PolicyWarning
type PolicyImpactSummary = clustercontracts.PolicyImpactSummary
type PolicySimulationStatus = clustercontracts.PolicySimulationStatus
type PolicyApprovalStatus = clustercontracts.PolicyApprovalStatus
type PolicyAuditStatus = clustercontracts.PolicyAuditStatus

// Re-export constants for convenience
const (
	PolicyReleaseStateDraft             = clustercontracts.PolicyReleaseStateDraft
	PolicyReleaseStateSimulationPending = clustercontracts.PolicyReleaseStateSimulationPending
	PolicyReleaseStateSimulationPassed  = clustercontracts.PolicyReleaseStateSimulationPassed
	PolicyReleaseStateSimulationFailed  = clustercontracts.PolicyReleaseStateSimulationFailed
	PolicyReleaseStateApprovalRequired  = clustercontracts.PolicyReleaseStateApprovalRequired
	PolicyReleaseStateApplying          = clustercontracts.PolicyReleaseStateApplying
	PolicyReleaseStateApprovalRejected  = clustercontracts.PolicyReleaseStateApprovalRejected
	PolicyReleaseStateApplied           = clustercontracts.PolicyReleaseStateApplied
	PolicyReleaseStateApplyFailed       = clustercontracts.PolicyReleaseStateApplyFailed
	PolicyReleaseStateActive            = clustercontracts.PolicyReleaseStateActive
	PolicyReleaseStateRollbackApplied   = clustercontracts.PolicyReleaseStateRollbackApplied
)

const (
	PolicyRiskLevelLow      = clustercontracts.PolicyRiskLevelLow
	PolicyRiskLevelMedium   = clustercontracts.PolicyRiskLevelMedium
	PolicyRiskLevelHigh     = clustercontracts.PolicyRiskLevelHigh
	PolicyRiskLevelCritical = clustercontracts.PolicyRiskLevelCritical
)

const (
	PolicyIssueSeverityBlocking = clustercontracts.PolicyIssueSeverityBlocking
	PolicyIssueSeverityHigh     = clustercontracts.PolicyIssueSeverityHigh
	PolicyIssueSeverityMedium   = clustercontracts.PolicyIssueSeverityMedium
	PolicyIssueSeverityLow      = clustercontracts.PolicyIssueSeverityLow
)

const (
	PolicyErrorSimulationBlockingConflict = "SIMULATION_BLOCKING_CONFLICT"
	PolicyErrorCNISemanticGap             = "CNI_SEMANTIC_GAP"
	PolicyErrorApprovalTokenInvalid       = "APPROVAL_TOKEN_INVALID"
	PolicyErrorApplyValidationFailed      = "APPLY_VALIDATION_FAILED"
	PolicyErrorFlannelNetpolDisabled      = "FLANNEL_NETPOL_DISABLED"
	PolicyErrorFlannelL7NotSupported      = "FLANNEL_L7_NOT_SUPPORTED"
	PolicyErrorFlannelOnlyStandardNP      = "FLANNEL_ONLY_STANDARD_NP"
	PolicyErrorCriticalNamespaceBlocked   = "CRITICAL_NAMESPACE_BLOCKED"

	PolicyWarningCNICapabilityDowngrade     = "CNI_CAPABILITY_DOWNGRADE"
	PolicyWarningImpactScopeExpanded        = "IMPACT_SCOPE_EXPANDED"
	PolicyWarningNonCriticalNamespaceChange = "NON_CRITICAL_NAMESPACE_CHANGE"
	PolicyWarningL7RuleSimplified           = "L7_RULE_SIMPLIFIED"
)

// PolicyDefinition 表示统一的 NetworkPolicyDefinition DSL。
type PolicyDefinition struct {
	APIVersion string               `json:"apiVersion" yaml:"apiVersion"`
	Kind       string               `json:"kind" yaml:"kind"`
	Metadata   PolicyObjectMetadata `json:"metadata" yaml:"metadata"`
	Spec       PolicyDefinitionSpec `json:"spec" yaml:"spec"`
}

// PolicyObjectMetadata 表示策略元数据。
type PolicyObjectMetadata struct {
	Name      string            `json:"name" yaml:"name"`
	Namespace string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

// PolicyDefinitionSpec 表示统一 DSL 的 spec。
type PolicyDefinitionSpec struct {
	Target      PolicyTarget          `json:"target" yaml:"target"`
	PolicyTypes []PolicyType          `json:"policyTypes" yaml:"policyTypes"`
	Ingress     []PolicyIngressRule   `json:"ingress" yaml:"ingress"`
	Egress      []PolicyEgressRule    `json:"egress" yaml:"egress"`
	Advanced    PolicyAdvancedOptions `json:"advanced,omitempty" yaml:"advanced,omitempty"`
}

// PolicyType 表示策略方向类型。
type PolicyType string

const (
	PolicyTypeIngress PolicyType = "Ingress"
	PolicyTypeEgress  PolicyType = "Egress"
)

// PolicyTarget 表示规则目标选择器。
type PolicyTarget struct {
	PodSelector       *PolicyLabelSelector `json:"podSelector,omitempty" yaml:"podSelector,omitempty"`
	NamespaceSelector *PolicyLabelSelector `json:"namespaceSelector,omitempty" yaml:"namespaceSelector,omitempty"`
}

// PolicyLabelSelector 表示 DSL 内部的标签选择器。
type PolicyLabelSelector struct {
	MatchLabels      map[string]string                `json:"matchLabels,omitempty" yaml:"matchLabels,omitempty"`
	MatchExpressions []PolicyLabelSelectorRequirement `json:"matchExpressions,omitempty" yaml:"matchExpressions,omitempty"`
}

// PolicyLabelSelectorRequirement 表示 matchExpressions 条件。
type PolicyLabelSelectorRequirement struct {
	Key      string   `json:"key" yaml:"key"`
	Operator string   `json:"operator" yaml:"operator"`
	Values   []string `json:"values,omitempty" yaml:"values,omitempty"`
}

// PolicyIngressRule 表示入站规则。
type PolicyIngressRule struct {
	Name   string           `json:"name,omitempty" yaml:"name,omitempty"`
	Action PolicyRuleAction `json:"action,omitempty" yaml:"action,omitempty"`
	From   *PolicyPeer      `json:"from,omitempty" yaml:"from,omitempty"`
	Ports  []PolicyPort     `json:"ports,omitempty" yaml:"ports,omitempty"`
	HTTP   []PolicyHTTPRule `json:"http,omitempty" yaml:"http,omitempty"`
	DNS    []PolicyDNSRule  `json:"dns,omitempty" yaml:"dns,omitempty"`
}

// PolicyEgressRule 表示出站规则。
type PolicyEgressRule struct {
	Name   string           `json:"name,omitempty" yaml:"name,omitempty"`
	Action PolicyRuleAction `json:"action,omitempty" yaml:"action,omitempty"`
	To     *PolicyPeer      `json:"to,omitempty" yaml:"to,omitempty"`
	Ports  []PolicyPort     `json:"ports,omitempty" yaml:"ports,omitempty"`
}

// PolicyRuleAction 表示规则动作。
type PolicyRuleAction string

const (
	PolicyRuleActionAllow PolicyRuleAction = "Allow"
	PolicyRuleActionDeny  PolicyRuleAction = "Deny"
)

// PolicyPeer 表示对端选择器。
type PolicyPeer struct {
	PodSelector       *PolicyLabelSelector `json:"podSelector,omitempty" yaml:"podSelector,omitempty"`
	NamespaceSelector *PolicyLabelSelector `json:"namespaceSelector,omitempty" yaml:"namespaceSelector,omitempty"`
	IPBlock           *PolicyIPBlock       `json:"ipBlock,omitempty" yaml:"ipBlock,omitempty"`
	ServiceAccount    string               `json:"serviceAccount,omitempty" yaml:"serviceAccount,omitempty"`
	FQDN              string               `json:"fqdn,omitempty" yaml:"fqdn,omitempty"`
}

// PolicyIPBlock 表示 IP 范围定义。
type PolicyIPBlock struct {
	CIDR   string   `json:"cidr" yaml:"cidr"`
	Except []string `json:"except,omitempty" yaml:"except,omitempty"`
}

// PolicyPort 表示端口规则。
type PolicyPort struct {
	Protocol string          `json:"protocol,omitempty" yaml:"protocol,omitempty"`
	Port     PolicyPortValue `json:"port,omitempty" yaml:"port,omitempty"`
	EndPort  int32           `json:"endPort,omitempty" yaml:"endPort,omitempty"`
}

// PolicyPortValue 表示可以为数字或命名端口的序列化值。
type PolicyPortValue string

// String 返回端口的规范字符串表示。
func (v PolicyPortValue) String() string {
	return string(v)
}

// MarshalJSON 保持数字端口输出为 JSON number，命名端口输出为字符串。
func (v PolicyPortValue) MarshalJSON() ([]byte, error) {
	if n, ok := parseCanonicalInt(string(v)); ok {
		return json.Marshal(n)
	}
	return json.Marshal(string(v))
}

// UnmarshalJSON 支持从数字或字符串读取端口值。
func (v *PolicyPortValue) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*v = ""
		return nil
	}

	var asInt int
	if err := json.Unmarshal(data, &asInt); err == nil {
		*v = PolicyPortValue(strconv.Itoa(asInt))
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		*v = PolicyPortValue(asString)
		return nil
	}

	return fmt.Errorf("invalid policy port value: %s", string(data))
}

// MarshalYAML 保持数字端口输出为 YAML number，命名端口输出为字符串。
func (v PolicyPortValue) MarshalYAML() (any, error) {
	if n, ok := parseCanonicalInt(string(v)); ok {
		return n, nil
	}
	return string(v), nil
}

// UnmarshalYAML 支持从数字或字符串读取端口值。
func (v *PolicyPortValue) UnmarshalYAML(node *yamlv3.Node) error {
	switch node.Kind {
	case yamlv3.ScalarNode:
		raw := strings.TrimSpace(node.Value)
		if raw == "" {
			*v = ""
			return nil
		}
		*v = PolicyPortValue(raw)
		return nil
	default:
		return fmt.Errorf("invalid policy port yaml node kind: %d", node.Kind)
	}
}

// PolicyHTTPRule 表示 L7 HTTP 规则。
type PolicyHTTPRule struct {
	Method string `json:"method,omitempty" yaml:"method,omitempty"`
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
}

// PolicyDNSRule 表示 L7 DNS 规则。
type PolicyDNSRule struct {
	MatchPattern string `json:"matchPattern,omitempty" yaml:"matchPattern,omitempty"`
}

// PolicyAdvancedOptions 表示 Calico 等高级选项。
type PolicyAdvancedOptions struct {
	Order          *float64 `json:"order,omitempty" yaml:"order,omitempty"`
	DoNotTrack     bool     `json:"doNotTrack,omitempty" yaml:"doNotTrack,omitempty"`
	ApplyOnForward bool     `json:"applyOnForward,omitempty" yaml:"applyOnForward,omitempty"`
}

// ApplyDefaults 为策略 DSL 补齐稳定默认值。
func (d *PolicyDefinition) ApplyDefaults() {
	if d.APIVersion == "" {
		d.APIVersion = PolicyDefinitionAPIVersion
	}
	if d.Kind == "" {
		d.Kind = PolicyDefinitionKind
	}
	if d.Spec.PolicyTypes == nil {
		d.Spec.PolicyTypes = []PolicyType{}
	}
	if d.Spec.Ingress == nil {
		d.Spec.Ingress = []PolicyIngressRule{}
	}
	if d.Spec.Egress == nil {
		d.Spec.Egress = []PolicyEgressRule{}
	}
}

// AllPolicyErrorCodes 返回 Phase 2 固定错误码与告警码集合。
func AllPolicyErrorCodes() []string {
	return []string{
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
}

func parseCanonicalInt(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	if strconv.Itoa(n) != raw {
		return 0, false
	}
	return n, true
}
