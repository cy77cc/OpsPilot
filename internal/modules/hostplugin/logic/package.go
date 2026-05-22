package logic

import (
	"context"
	"errors"
	"os"

	hostpluginmodel "github.com/cy77cc/OpsPilot/internal/modules/hostplugin/model"
	"gorm.io/gorm"
)

// ListPackages returns all uploaded plugin packages.
func (s *Service) ListPackages(ctx context.Context) ([]hostpluginmodel.HostPluginPackage, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}
	return s.ListPackagesWithDB(ctx, db)
}

// ListPackagesWithDB returns all packages using the provided DB (for testing).
func (s *Service) ListPackagesWithDB(ctx context.Context, db *gorm.DB) ([]hostpluginmodel.HostPluginPackage, error) {
	var pkgs []hostpluginmodel.HostPluginPackage
	err := db.WithContext(ctx).Order("id DESC").Find(&pkgs).Error
	return pkgs, err
}

// CreatePackage records a new uploaded package and auto-creates a version entry.
func (s *Service) CreatePackage(ctx context.Context, pkg *hostpluginmodel.HostPluginPackage) error {
	db := s.db()
	if db == nil {
		return errors.New("hostplugin service: db is required")
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(pkg).Error; err != nil {
			return err
		}

		// Auto-create version entry
		var plugin hostpluginmodel.HostPlugin
		if err := tx.Where("plugin_key = ?", pkg.PluginKey).First(&plugin).Error; err != nil {
			return err
		}

		version := hostpluginmodel.HostPluginVersion{
			PluginID:         plugin.ID,
			Version:          pkg.Version,
			Arch:             pkg.Arch,
			PackagePath:      pkg.StoragePath,
			InstallEntry:     "install.sh",
			UpgradeEntry:     "install.sh",
			UninstallEntry:   "uninstall.sh",
			Checksum:         pkg.Checksum,
			CapabilitiesJSON: `["metrics.collect","exec.shell","exec.script.shell"]`,
			ConfigSchemaJSON: `{}`,
		}
		return tx.Where("plugin_id = ? AND version = ? AND arch = ?", plugin.ID, pkg.Version, pkg.Arch).
			FirstOrCreate(&version).Error
	})
}

// DeletePackage removes a package record and its file from disk.
func (s *Service) DeletePackage(ctx context.Context, packageID uint64) error {
	db := s.db()
	if db == nil {
		return errors.New("hostplugin service: db is required")
	}
	return s.DeletePackageWithDB(ctx, db, packageID)
}

// DeletePackageWithDB deletes using the provided DB (for testing).
func (s *Service) DeletePackageWithDB(ctx context.Context, db *gorm.DB, packageID uint64) error {
	var pkg hostpluginmodel.HostPluginPackage
	if err := db.WithContext(ctx).First(&pkg, packageID).Error; err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete version entry
		var plugin hostpluginmodel.HostPlugin
		if err := tx.Where("plugin_key = ?", pkg.PluginKey).First(&plugin).Error; err == nil {
			tx.Where("plugin_id = ? AND version = ? AND arch = ?", plugin.ID, pkg.Version, pkg.Arch).
				Delete(&hostpluginmodel.HostPluginVersion{})
		}

		// Delete package record
		if err := tx.Delete(&hostpluginmodel.HostPluginPackage{}, packageID).Error; err != nil {
			return err
		}

		// Delete file from disk (ignore error if file doesn't exist)
		_ = os.Remove(pkg.StoragePath)
		return nil
	})
}

// GetPackage returns a package by ID.
func (s *Service) GetPackage(ctx context.Context, packageID uint64) (*hostpluginmodel.HostPluginPackage, error) {
	db := s.db()
	if db == nil {
		return nil, errors.New("hostplugin service: db is required")
	}

	var pkg hostpluginmodel.HostPluginPackage
	if err := db.WithContext(ctx).First(&pkg, packageID).Error; err != nil {
		return nil, err
	}
	return &pkg, nil
}
