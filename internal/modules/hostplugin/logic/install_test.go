package logic

import (
	"context"
	"path/filepath"
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

func TestBuildInstallPlan_UsesRemoteTarballPath(t *testing.T) {
	svc := newHostPluginServiceForTest(t)
	instance := &hostpluginmodel.HostPluginInstance{ID: 42}
	version := &hostpluginmodel.HostPluginVersion{
		PackagePath:  "/controller/releases/nodeagentx-linux-amd64.tar.gz",
		InstallEntry: "install.sh",
	}

	plan := svc.buildInstallPlan(instance, version)

	if plan.remotePackagePath != "/tmp/opspilot/plugins/42/nodeagentx-linux-amd64.tar.gz" {
		t.Fatalf("unexpected remote package path: %s", plan.remotePackagePath)
	}
	if got := filepath.Base(plan.localPackagePath); got != "nodeagentx-linux-amd64.tar.gz" {
		t.Fatalf("expected local package basename to be preserved, got %s", got)
	}
	if len(plan.commands) != 2 {
		t.Fatalf("expected 2 post-upload commands, got %d", len(plan.commands))
	}
	if strings.Contains(plan.commands[0], "/controller/releases/nodeagentx-linux-amd64.tar.gz") {
		t.Fatalf("untar command should not reference controller-local path: %s", plan.commands[0])
	}
	if !strings.Contains(plan.commands[0], plan.remotePackagePath) {
		t.Fatalf("untar command should reference remote package path: %s", plan.commands[0])
	}
	if !strings.Contains(plan.commands[1], "cd '/tmp/opspilot/plugins/42'") {
		t.Fatalf("install command should run inside remote work dir: %s", plan.commands[1])
	}
}
