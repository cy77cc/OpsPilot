package logic

import (
	"context"
	"testing"
	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
)

func TestCreateMultipleManualHostsWithEmptyString(t *testing.T) {
	_, db := newHostLogicTestService(t)
	ctx := context.Background()

    empty := ""
	// First manual host with empty strings instead of NULLs
	node1 := &model.Node{
		Name:     "Host1",
		IP:       "192.168.1.1",
		Port:     22,
		SSHUser:  "root",
		Source:   "manual_ssh",
        Provider: &empty,
        ProviderID: &empty,
        Status: "online",
	}
	if err := db.WithContext(ctx).Create(node1).Error; err != nil {
		t.Fatalf("Failed to create first host: %v", err)
	}

	// Second manual host with same empty strings
	node2 := &model.Node{
		Name:     "Host2",
		IP:       "192.168.1.2",
		Port:     22,
		SSHUser:  "root",
		Source:   "manual_ssh",
        Provider: &empty,
        ProviderID: &empty,
        Status: "online",
	}
	err := db.WithContext(ctx).Create(node2).Error
	if err != nil {
		t.Logf("Expected failure with empty strings: %v", err)
	} else {
		t.Fatalf("Should have failed with duplicate empty strings")
	}
}
