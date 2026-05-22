package logic

import (
	"context"
	"testing"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openPackageTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&hostpluginmodel.HostPlugin{},
		&hostpluginmodel.HostPluginVersion{},
		&hostpluginmodel.HostPluginInstance{},
		&hostpluginmodel.HostPluginConfigRevision{},
		&hostpluginmodel.HostPluginTask{},
		&hostpluginmodel.HostPluginTaskLog{},
		&hostpluginmodel.OpsAgentCA{},
		&hostpluginmodel.OpsAgentHostCert{},
		&hostpluginmodel.HostPluginPackage{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestService_ListPackages(t *testing.T) {
	db := openPackageTestDB(t)
	svc := &Service{svcCtx: nil}
	ctx := context.Background()

	pkgs, err := svc.ListPackagesWithDB(ctx, db)
	if err != nil {
		t.Fatalf("ListPackages() error: %v", err)
	}
	if len(pkgs) != 0 {
		t.Fatalf("ListPackages() = %d, want 0", len(pkgs))
	}

	// Insert a package
	db.Create(&hostpluginmodel.HostPluginPackage{
		PluginKey:   "opsagent",
		Version:     "v1.0.0",
		Arch:        "amd64",
		Filename:    "opsagent-v1.0.0-linux-amd64.tar.gz",
		StoragePath: "/tmp/test.tar.gz",
		Checksum:    "abc123",
		SizeBytes:   1024,
	})

	pkgs, err = svc.ListPackagesWithDB(ctx, db)
	if err != nil {
		t.Fatalf("ListPackages() error: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("ListPackages() = %d, want 1", len(pkgs))
	}
	if pkgs[0].PluginKey != "opsagent" {
		t.Fatalf("PluginKey = %q, want %q", pkgs[0].PluginKey, "opsagent")
	}
}

func TestService_DeletePackage(t *testing.T) {
	db := openPackageTestDB(t)
	svc := &Service{}
	ctx := context.Background()

	pkg := hostpluginmodel.HostPluginPackage{
		PluginKey:   "opsagent",
		Version:     "v1.0.0",
		Arch:        "amd64",
		Filename:    "test.tar.gz",
		StoragePath: "/tmp/test.tar.gz",
		Checksum:    "abc",
		SizeBytes:   100,
	}
	db.Create(&pkg)

	err := svc.DeletePackageWithDB(ctx, db, pkg.ID)
	if err != nil {
		t.Fatalf("DeletePackage() error: %v", err)
	}

	var count int64
	db.Model(&hostpluginmodel.HostPluginPackage{}).Count(&count)
	if count != 0 {
		t.Fatalf("package count = %d, want 0", count)
	}
}
