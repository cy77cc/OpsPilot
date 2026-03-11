package service

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

func TestServiceExpertIncludesInventoryResolveTools(t *testing.T) {
	exp := New(common.PlatformDeps{})
	tools := exp.Tools(context.Background())
	var hasClusterInventory bool
	var hasServiceInventory bool
	var hasServiceStatusByTarget bool
	for _, item := range tools {
		info, err := item.Info(context.Background())
		if err != nil || info == nil {
			continue
		}
		switch info.Name {
		case "cluster_list_inventory":
			hasClusterInventory = true
		case "service_list_inventory":
			hasServiceInventory = true
		case "service_status_by_target":
			hasServiceStatusByTarget = true
		}
	}
	if !hasClusterInventory || !hasServiceInventory || !hasServiceStatusByTarget {
		t.Fatalf("semantic service tools missing: cluster=%t service=%t status_by_target=%t", hasClusterInventory, hasServiceInventory, hasServiceStatusByTarget)
	}
}
