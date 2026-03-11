// Package cmd 提供命令行入口。
//
// 本文件实现数据库迁移命令，支持 up/down/status 子命令。
package cmd

import (
	"fmt"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/storage"
	"github.com/cy77cc/OpsPilot/storage/migration"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	upSteps   int // 向前迁移步数
	downSteps int // 回滚步数
)

// migrateCMD 是数据库迁移父命令。
var migrateCMD = &cobra.Command{
	Use:   "migrate",
	Short: "database migration commands",
}

// migrateUpCMD 执行向上迁移。
var migrateUpCMD = &cobra.Command{
	Use:   "up",
	Short: "apply versioned migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, cleanup := mustInitMigrationDeps()
		defer cleanup()
		return migration.Migrate(db, migration.DirectionUp, upSteps)
	},
}

// migrateDownCMD 执行回滚迁移。
var migrateDownCMD = &cobra.Command{
	Use:   "down",
	Short: "rollback versioned migrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, cleanup := mustInitMigrationDeps()
		defer cleanup()
		return migration.Migrate(db, migration.DirectionDown, downSteps)
	},
}

// migrateStatusCMD 打印迁移状态。
var migrateStatusCMD = &cobra.Command{
	Use:   "status",
	Short: "print migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, cleanup := mustInitMigrationDeps()
		defer cleanup()

		items, err := migration.Status(db)
		if err != nil {
			return err
		}
		for _, item := range items {
			state := "pending"
			at := "-"
			if item.Applied {
				state = "applied"
				if item.AppliedAt != nil {
					at = item.AppliedAt.Format("2006-01-02 15:04:05")
				}
			}
			fmt.Printf("%s\t%s\t%s\t%s\n", item.Version, item.Name, state, at)
		}
		return nil
	},
}

// mustInitMigrationDeps 初始化迁移依赖。
//
// 返回数据库连接和清理函数。
func mustInitMigrationDeps() (*gorm.DB, func()) {
	config.MustNewConfig()
	logger.Init(logger.MustNewZapLogger())
	db := storage.MustNewDB()
	cleanup := func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
	return db, cleanup
}

func init() {
	migrateUpCMD.Flags().IntVar(&upSteps, "steps", 0, "number of steps to run, 0 means all")
	migrateDownCMD.Flags().IntVar(&downSteps, "steps", 1, "number of steps to rollback")

	migrateCMD.AddCommand(migrateUpCMD)
	migrateCMD.AddCommand(migrateDownCMD)
	migrateCMD.AddCommand(migrateStatusCMD)
	rootCMD.AddCommand(migrateCMD)
}
