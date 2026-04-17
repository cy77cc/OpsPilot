package bootstrap

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/storage"
	"github.com/cy77cc/OpsPilot/internal/core/storage/migration"
	"gorm.io/gorm"
)

// RunBootstrapMigrations applies versioned and development-only migrations.
func RunBootstrapMigrations() error {
	db := storage.MustNewDB()
	sqlDB, err := db.DB()
	if err == nil {
		defer sqlDB.Close()
	}

	if err := migration.RunMigrations(db); err != nil {
		return fmt.Errorf("run migrations failed: %w", err)
	}

	if config.CFG.App.AutoMigrate {
		if err := migration.RunDevAutoMigrate(db); err != nil {
			return fmt.Errorf("run dev auto migrate failed: %w", err)
		}
	}

	// Fix host unique index
	if err := fixHostUniqueIndex(db); err != nil {
		return fmt.Errorf("fix host unique index failed: %w", err)
	}

	return nil
}

func fixHostUniqueIndex(db *gorm.DB) error {
	// 1. 清理存量空字符串数据
	if err := db.Exec("UPDATE nodes SET provider = NULL WHERE provider = ''").Error; err != nil {
		return err
	}
	if err := db.Exec("UPDATE nodes SET provider_instance_id = NULL WHERE provider_instance_id = ''").Error; err != nil {
		return err
	}

	// 2. 重建索引
	// 先删除旧索引（如果存在且不是部分索引）
	if err := db.Exec("DROP INDEX IF EXISTS idx_provider_instance").Error; err != nil {
		return err
	}

	// 3. 创建部分索引
	return db.Exec(`
		CREATE UNIQUE INDEX idx_provider_instance 
		ON nodes(provider, provider_instance_id) 
		WHERE provider IS NOT NULL AND provider != ''
	`).Error
}
