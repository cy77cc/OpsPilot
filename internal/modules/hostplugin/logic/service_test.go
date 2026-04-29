package logic

import (
	"testing"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHostPluginModels_AutoMigrateAndPersist(t *testing.T) {
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

	if !db.Migrator().HasTable(&hostpluginmodel.HostPluginInstance{}) {
		t.Fatalf("expected host_plugin_instances table")
	}
}
