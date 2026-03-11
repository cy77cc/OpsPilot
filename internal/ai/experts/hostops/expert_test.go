package hostops

import (
	"context"
	"testing"

	"github.com/cy77cc/OpsPilot/internal/ai/tools/common"
)

func TestHostopsExpertIncludesHostInventoryTool(t *testing.T) {
	exp := New(common.PlatformDeps{})
	tools := exp.Tools(context.Background())
	var hasInventory bool
	var hasExecByTarget bool
	for _, item := range tools {
		info, err := item.Info(context.Background())
		if err != nil || info == nil {
			continue
		}
		switch info.Name {
		case "host_list_inventory":
			hasInventory = true
		case "host_exec_by_target":
			hasExecByTarget = true
		}
	}
	if !hasInventory || !hasExecByTarget {
		t.Fatalf("expected semantic host tools missing: inventory=%t exec_by_target=%t", hasInventory, hasExecByTarget)
	}
}
