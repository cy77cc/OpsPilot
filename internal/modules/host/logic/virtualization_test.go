package logic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	model "github.com/cy77cc/OpsPilot/internal/modules/host/model"
)

func TestKVMProvision_EncryptsSSHPassword(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)
	if err := db.AutoMigrate(&model.HostVirtualizationTask{}); err != nil {
		t.Fatalf("auto migrate host virtualization task: %v", err)
	}

	host := &model.Node{
		Name:    "kvm-parent-host",
		IP:      "10.20.0.1",
		Port:    22,
		SSHUser: "root",
		Status:  "online",
		Source:  "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(host).Error; err != nil {
		t.Fatalf("seed parent host: %v", err)
	}

	const plainPassword = "KVM-Child-Password"
	task, node, err := hostSvc.KVMProvision(context.Background(), 3001, uint64(host.ID), KVMProvisionReq{
		Name:     "kvm-child-node",
		IP:       "10.20.0.11",
		SSHUser:  "ubuntu",
		Password: plainPassword,
		CPU:      2,
		MemoryMB: 2048,
		DiskGB:   20,
	})
	if err != nil {
		t.Fatalf("kvm provision: %v", err)
	}
	if task == nil || node == nil {
		t.Fatal("expected task and node to be created")
	}

	var persisted model.Node
	if err := db.WithContext(context.Background()).First(&persisted, node.ID).Error; err != nil {
		t.Fatalf("reload provisioned node: %v", err)
	}
	assertCipherRoundTrip(t, persisted.SSHPassword, plainPassword)
}

func TestKVMProvision_SanitizesTaskRequestJSON(t *testing.T) {
	hostSvc, db := newHostLogicTestService(t)
	if err := db.AutoMigrate(&model.HostVirtualizationTask{}); err != nil {
		t.Fatalf("auto migrate host virtualization task: %v", err)
	}

	host := &model.Node{
		Name:    "kvm-parent-host-sanitize",
		IP:      "10.20.0.2",
		Port:    22,
		SSHUser: "root",
		Status:  "online",
		Source:  "manual_ssh",
	}
	if err := db.WithContext(context.Background()).Create(host).Error; err != nil {
		t.Fatalf("seed parent host: %v", err)
	}

	const plainPassword = "KVM-Task-Plain-Password"
	task, node, err := hostSvc.KVMProvision(context.Background(), 3002, uint64(host.ID), KVMProvisionReq{
		Name:     "kvm-child-sanitize",
		IP:       "10.20.0.12",
		SSHUser:  "ubuntu",
		Password: plainPassword,
	})
	if err != nil {
		t.Fatalf("kvm provision: %v", err)
	}
	if task == nil || node == nil {
		t.Fatal("expected task and node to be created")
	}
	if strings.Contains(task.RequestJSON, plainPassword) {
		t.Fatalf("task request json leaked plaintext password: %s", task.RequestJSON)
	}

	var requestPayload map[string]any
	if err := json.Unmarshal([]byte(task.RequestJSON), &requestPayload); err != nil {
		t.Fatalf("decode task request json: %v", err)
	}
	if got := requestPayload["password"]; got != redactedVirtualizationPassword {
		t.Fatalf("expected task request password to be redacted, got %#v", got)
	}

	var persistedTask model.HostVirtualizationTask
	if err := db.WithContext(context.Background()).Where("id = ?", task.ID).First(&persistedTask).Error; err != nil {
		t.Fatalf("reload virtualization task: %v", err)
	}
	if strings.Contains(persistedTask.RequestJSON, plainPassword) {
		t.Fatalf("persisted task request json leaked plaintext password: %s", persistedTask.RequestJSON)
	}
}
