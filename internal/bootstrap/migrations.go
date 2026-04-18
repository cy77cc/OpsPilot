package bootstrap

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/cy77cc/OpsPilot/internal/core/storage"
	"github.com/cy77cc/OpsPilot/internal/core/storage/migration"
	aimodel "github.com/cy77cc/OpsPilot/internal/modules/ai/model"
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
	if err := ensureAIAlertHealJobEventIndex(db); err != nil {
		return fmt.Errorf("ensure ai alert heal job event index failed: %w", err)
	}

	if config.CFG.App.AutoMigrate {
		if err := migration.RunDevAutoMigrate(db); err != nil {
			return fmt.Errorf("run dev auto migrate failed: %w", err)
		}

		// Fix host unique index only for PostgreSQL (MySQL doesn't support partial index WHERE clause)
		if db.Dialector.Name() == "postgres" {
			if err := fixHostUniqueIndex(db); err != nil {
				return fmt.Errorf("fix host unique index failed: %w", err)
			}
		}
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

func ensureAIAlertHealJobEventIndex(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&aimodel.AIAlertHealJob{}) {
		return nil
	}
	if db.Migrator().HasIndex(&aimodel.AIAlertHealJob{}, "uk_ai_alert_heal_jobs_event_id") {
		return nil
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasIndex(&aimodel.AIAlertHealJob{}, "uk_ai_alert_heal_jobs_event_id") {
			return nil
		}
		if err := dedupeAIAlertHealJobsByEventID(tx); err != nil {
			return err
		}
		return tx.Migrator().CreateIndex(&aimodel.AIAlertHealJob{}, "uk_ai_alert_heal_jobs_event_id")
	})
}

func dedupeAIAlertHealJobsByEventID(db *gorm.DB) error {
	type duplicateEvent struct {
		EventID string `gorm:"column:event_id"`
	}

	var duplicates []duplicateEvent
	if err := db.Table("ai_alert_heal_jobs").
		Select("event_id").
		Group("event_id").
		Having("COUNT(*) > 1").
		Order("event_id ASC").
		Find(&duplicates).Error; err != nil {
		return err
	}

	for _, duplicate := range duplicates {
		var keepID string
		if err := db.Table("ai_alert_heal_jobs").
			Select("id").
			Where("event_id = ?", duplicate.EventID).
			Order("created_at ASC").
			Order("id ASC").
			Limit(1).
			Scan(&keepID).Error; err != nil {
			return err
		}
		if keepID == "" {
			continue
		}
		if err := db.Exec(
			"DELETE FROM ai_alert_heal_jobs WHERE event_id = ? AND id <> ?",
			duplicate.EventID,
			keepID,
		).Error; err != nil {
			return err
		}
	}

	return nil
}
