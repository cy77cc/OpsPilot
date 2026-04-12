package security

import clustercontracts "github.com/cy77cc/OpsPilot/internal/modules/cluster/contracts"

const (
	ClusterModePlatformManaged = clustercontracts.ClusterModePlatformManaged
	ClusterModeExternalManaged = clustercontracts.ClusterModeExternalManaged
)

type Phase3GateDecision = clustercontracts.Phase3GateDecision

const (
	Phase3GateDecisionAllowed          Phase3GateDecision = clustercontracts.Phase3GateDecisionAllowed
	Phase3GateDecisionApprovalRequired Phase3GateDecision = clustercontracts.Phase3GateDecisionApprovalRequired
	Phase3GateDecisionRejected         Phase3GateDecision = clustercontracts.Phase3GateDecisionRejected
	Phase3GateDecisionBlocked          Phase3GateDecision = clustercontracts.Phase3GateDecisionBlocked
)

type RuntimeContainResult = clustercontracts.RuntimeContainResult
