package logic

import (
	"context"
	"strings"
	"testing"

	hostmodel "github.com/cy77cc/OpsPilot/internal/modules/host/model"
	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"github.com/cy77cc/OpsPilot/internal/svc"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHostPluginServiceForTest(t *testing.T) *Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&hostmodel.Node{},
		&hostmodel.SSHKey{},
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginVersion{},
		&hostpluginmodel.HostPluginInstance{},
		&hostpluginmodel.HostPluginTask{},
		&hostpluginmodel.HostPluginTaskLog{},
	); err != nil {
		t.Fatalf("auto migrate hostplugin install models: %v", err)
	}

	plugin := hostpluginmodel.HostPlugin{
		PluginKey:      "opsagent",
		Name:           "OpsAgent",
		Category:       "host-observability",
		Description:    "agent runtime",
		DefaultVersion: "nodeagentx-dc57fbc-dirty",
		Status:         "active",
	}
	if err := db.Create(&plugin).Error; err != nil {
		t.Fatalf("create plugin: %v", err)
	}

	versions := []hostpluginmodel.HostPluginVersion{
		{
			PluginID:         plugin.ID,
			Version:          "nodeagentx-dc57fbc-dirty",
			Arch:             "amd64",
			PackagePath:      "/tmp/releases/nodeagentx-linux-amd64.tar.gz",
			InstallEntry:     "install.sh",
			UpgradeEntry:     "upgrade.sh",
			UninstallEntry:   "uninstall.sh",
			Checksum:         "sha256-amd64",
			CapabilitiesJSON: "[]",
			ConfigSchemaJSON: "{}",
		},
		{
			PluginID:         plugin.ID,
			Version:          "nodeagentx-dc57fbc-dirty",
			Arch:             "arm64",
			PackagePath:      "/tmp/releases/nodeagentx-linux-arm64.tar.gz",
			InstallEntry:     "install.sh",
			UpgradeEntry:     "upgrade.sh",
			UninstallEntry:   "uninstall.sh",
			Checksum:         "sha256-arm64",
			CapabilitiesJSON: "[]",
			ConfigSchemaJSON: "{}",
		},
	}
	if err := db.Create(&versions).Error; err != nil {
		t.Fatalf("create plugin versions: %v", err)
	}

	return NewService(&svc.ServiceContext{DB: db})
}

func TestResolvePackageForHost_SelectsByArchitecture(t *testing.T) {
	svc := newHostPluginServiceForTest(t)
	version, err := svc.ResolveVersionForHost(context.Background(), "opsagent", "nodeagentx-dc57fbc-dirty", "amd64")
	if err != nil {
		t.Fatalf("resolve package: %v", err)
	}
	if !strings.Contains(version.PackagePath, "linux-amd64.tar.gz") {
		t.Fatalf("expected amd64 package path, got %s", version.PackagePath)
	}
}
