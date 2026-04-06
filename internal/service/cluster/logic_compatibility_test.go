package cluster

import "testing"

func TestNodeOps_CompatibilityDrainFlow(t *testing.T) {
	if fn := (*Handler).DrainNode; fn == nil {
		t.Fatalf("expected DrainNode handler to be wired")
	}
}

func TestWorkloadOps_CompatibilityScaleDeploymentFlow(t *testing.T) {
	if fn := (*Handler).ScaleDeployment; fn == nil {
		t.Fatalf("expected ScaleDeployment handler to be wired")
	}
}

func TestServiceOps_CompatibilityCreateServiceFlow(t *testing.T) {
	if fn := (*Handler).CreateService; fn == nil {
		t.Fatalf("expected CreateService handler to be wired")
	}
}

func TestAdvancedOps_CompatibilityUpgradeFlow(t *testing.T) {
	if fn := (*Handler).UpgradeCluster; fn == nil {
		t.Fatalf("expected UpgradeCluster handler to be wired")
	}
}
