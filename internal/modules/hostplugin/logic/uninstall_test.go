package logic

import (
	"context"
	"testing"
	"time"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/gorm"
)

func openUninstallTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := openHostPluginTestDB(t)
	if err := db.AutoMigrate(&hostmodel.Node{}); err != nil {
		t.Fatalf("auto migrate host model: %v", err)
	}
	return db
}

func TestService_UninstallOnHost(t *testing.T) {
	db := openUninstallTestDB(t)
	ctx := context.Background()

	// Seed a host
	host := &hostmodel.Node{
		IP:      "10.0.0.1",
		Port:    22,
		SSHUser: "root",
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("create host: %v", err)
	}

	// Seed plugin and instance
	plugin := hostpluginmodel.HostPlugin{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "host-observability",
		Description:    "test",
		DefaultVersion: "v1.0.0",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	instance := hostpluginmodel.HostPluginInstance{
		HostID:           uint64(host.ID),
		PluginID:         plugin.ID,
		DesiredVersion:   "v1.0.0",
		InstalledVersion: "v1.0.0",
		InstallStatus:    "succeeded",
		RuntimeStatus:    "online",
		AgentID:          "opsagent-host-1-instance-1",
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	s := NewService(&svc.ServiceContext{DB: db})

	// Mock the uninstall execution to avoid actual SSH calls
	originalExec := executeHostPluginUninstallPlan
	executeHostPluginUninstallPlan = func(_ context.Context, _ *Service, _ *hostmodel.Node, _ *hostpluginmodel.HostPluginTask) error {
		return nil
	}
	defer func() { executeHostPluginUninstallPlan = originalExec }()

	taskID, err := s.UninstallOnHost(ctx, uint64(host.ID), instance.ID)
	if err != nil {
		t.Fatalf("UninstallOnHost() error: %v", err)
	}
	if taskID == 0 {
		t.Fatal("UninstallOnHost() returned 0 task ID")
	}

	// Verify task was created with correct operation
	var task hostpluginmodel.HostPluginTask
	if err := db.First(&task, taskID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.Operation != "uninstall" {
		t.Fatalf("task.Operation = %q, want %q", task.Operation, "uninstall")
	}
	if task.InstanceID != instance.ID {
		t.Fatalf("task.InstanceID = %d, want %d", task.InstanceID, instance.ID)
	}

	// Wait for async goroutine to finish updating instance status
	deadline := time.Now().Add(5 * time.Second)
	var updated hostpluginmodel.HostPluginInstance
	for time.Now().Before(deadline) {
		if err := db.First(&updated, instance.ID).Error; err != nil {
			t.Fatalf("load updated instance: %v", err)
		}
		if updated.RuntimeStatus == "uninstalled" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if updated.RuntimeStatus != "uninstalled" {
		t.Fatalf("RuntimeStatus = %q, want %q", updated.RuntimeStatus, "uninstalled")
	}
	if updated.InstallStatus != "uninstalled" {
		t.Fatalf("InstallStatus = %q, want %q", updated.InstallStatus, "uninstalled")
	}
}

func TestService_UninstallOnHost_AlreadyUninstalled(t *testing.T) {
	db := openUninstallTestDB(t)
	ctx := context.Background()

	host := &hostmodel.Node{
		IP:      "10.0.0.2",
		Port:    22,
		SSHUser: "root",
	}
	if err := db.Create(&host).Error; err != nil {
		t.Fatalf("create host: %v", err)
	}

	plugin := hostpluginmodel.HostPlugin{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "host-observability",
		Description:    "test",
		DefaultVersion: "v1.0.0",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	instance := hostpluginmodel.HostPluginInstance{
		HostID:         uint64(host.ID),
		PluginID:       plugin.ID,
		DesiredVersion: "v1.0.0",
		InstallStatus:  "uninstalled",
		RuntimeStatus:  "uninstalled",
		AgentID:        "opsagent-host-2-instance-1",
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create instance: %v", err)
	}

	s := NewService(&svc.ServiceContext{DB: db})

	_, err := s.UninstallOnHost(ctx, uint64(host.ID), instance.ID)
	if err == nil {
		t.Fatal("UninstallOnHost() expected error for already uninstalled, got nil")
	}
	if err.Error() != "plugin is already uninstalled" {
		t.Fatalf("UninstallOnHost() error = %q, want %q", err.Error(), "plugin is already uninstalled")
	}
}

func TestService_UninstallOnHost_InstanceNotFound(t *testing.T) {
	db := openUninstallTestDB(t)
	ctx := context.Background()

	s := NewService(&svc.ServiceContext{DB: db})

	_, err := s.UninstallOnHost(ctx, 999, 999)
	if err == nil {
		t.Fatal("UninstallOnHost() expected error for non-existent instance, got nil")
	}
	if err.Error() != "plugin instance not found for this host" {
		t.Fatalf("UninstallOnHost() error = %q, want %q", err.Error(), "plugin instance not found for this host")
	}
}
