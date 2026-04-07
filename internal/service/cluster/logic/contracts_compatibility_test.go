package logic

import (
	"testing"

	clustercontracts "github.com/cy77cc/OpsPilot/internal/service/cluster/contracts"
)

func TestClusterContracts_TypeAliasCompatibility(t *testing.T) {
	var _ clustercontracts.ClusterDetail = ClusterDetail{}
	var _ = OperationStateCompleted
	var _ = ClusterModePlatformManaged
}
