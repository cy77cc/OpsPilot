// Package cmd 提供命令行入口。
//
// 本文件实现根命令，是应用程序的主入口点。
// 支持配置文件路径和调试模式标志。
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/cy77cc/OpsPilot/internal/logger"
	"github.com/cy77cc/OpsPilot/internal/server"
	"github.com/cy77cc/OpsPilot/storage"
	"github.com/cy77cc/OpsPilot/storage/migration"
	"github.com/cy77cc/OpsPilot/version"
	"github.com/spf13/cobra"
)

// rootCMD 是根命令，启动应用程序。
var (
	rootCMD = &cobra.Command{
		Use:     "OpsPilot",
		Short:   "OpsPilot is a tool to manage k8s cluster",
		Version: version.VERSION,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.MustNewConfig()
			logger.Init(logger.MustNewZapLogger())
			if err := runBootstrapMigrations(); err != nil {
				return err
			}
			ctx := cmd.Context()
			return server.Start(ctx)
		},
	}
)

// runBootstrapMigrations 执行启动时的数据库迁移。
//
// 包括版本化迁移和开发模式的自动迁移（如果启用）。
func runBootstrapMigrations() error {
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
	return nil
}

// Execute 是命令行入口函数。
//
// 注册信号处理，支持优雅关闭。
// 配置文件默认为 configs/config.yaml。
func Execute() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTRAP)
	defer stop()
	var cfgFile string
	rootCMD.PersistentFlags().StringVar(&cfgFile, "config", "configs/config.yaml", "config file path")
	config.SetConfigFile(cfgFile)
	rootCMD.Flags().BoolVar(&config.Debug, "debug", false, "enable debug mode")
	if err := rootCMD.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
