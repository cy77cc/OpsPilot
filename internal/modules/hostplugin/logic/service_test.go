package logic

import (
	"context"
	"testing"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openHostPluginTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	err = db.AutoMigrate(
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginVersion{},
		&hostpluginmodel.HostPluginInstance{},
		&hostpluginmodel.HostPluginConfigRevision{},
		&hostpluginmodel.HostPluginTask{},
		&hostpluginmodel.HostPluginTaskLog{},
	)
	if err != nil {
		t.Fatalf("auto migrate hostplugin models: %v", err)
	}

	return db
}

func TestHostPluginModels_AutoMigratePersistsAndDefaults(t *testing.T) {
	db := openHostPluginTestDB(t)

	if !db.Migrator().HasTable(&hostpluginmodel.HostPluginInstance{}) {
		t.Fatalf("expected host_plugin_instances table")
	}

	plugin := hostpluginmodel.HostPlugin{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "runtime",
		Description:    "agent runtime",
		DefaultVersion: "v1.0.0",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("create host plugin: %v", err)
	}

	var storedPlugin hostpluginmodel.HostPlugin
	if err := db.Where("plugin_key = ?", "opsagent").First(&storedPlugin).Error; err != nil {
		t.Fatalf("read host plugin: %v", err)
	}
	if storedPlugin.Name != "OpsAgent" {
		t.Fatalf("expected persisted plugin name OpsAgent, got %q", storedPlugin.Name)
	}
	if storedPlugin.Status != "active" {
		t.Fatalf("expected default plugin status active, got %q", storedPlugin.Status)
	}

	instance := hostpluginmodel.HostPluginInstance{
		HostID:           42,
		PluginID:         plugin.ID,
		DesiredVersion:   "v1.0.0",
		CapabilitiesJSON: `["exec.shell"]`,
		LastError:        "",
	}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatalf("create host plugin instance: %v", err)
	}

	var storedInstance hostpluginmodel.HostPluginInstance
	if err := db.First(&storedInstance, instance.ID).Error; err != nil {
		t.Fatalf("read host plugin instance: %v", err)
	}
	if storedInstance.InstallStatus != "pending" {
		t.Fatalf("expected default install status pending, got %q", storedInstance.InstallStatus)
	}
	if storedInstance.RuntimeStatus != "pending_online" {
		t.Fatalf("expected default runtime status pending_online, got %q", storedInstance.RuntimeStatus)
	}
	if storedInstance.HealthStatus != "unknown" {
		t.Fatalf("expected default health status unknown, got %q", storedInstance.HealthStatus)
	}
}

func TestEnsureDefaultCatalogSeedsOpsAgent(t *testing.T) {
	db := openHostPluginTestDB(t)
	svc := NewService(&svc.ServiceContext{DB: db})

	if err := svc.EnsureDefaultCatalog(context.Background()); err != nil {
		t.Fatalf("seed catalog: %v", err)
	}

	var count int64
	if err := db.Model(&hostpluginmodel.HostPlugin{}).Where("plugin_key = ?", "opsagent").Count(&count).Error; err != nil {
		t.Fatalf("count plugin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected opsagent catalog entry, got %d", count)
	}
}
