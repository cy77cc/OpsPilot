package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TODO 这里还有可以优化的点，可以支持读写分离的模式

func NewDB() (*gorm.DB, error) {
	var dialector gorm.Dialector
	var maxOpenConns, maxIdleConns int
	var connMaxLifetime time.Duration

	if config.CFG.MySQL.Enable {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=%t&loc=%s",
			config.CFG.MySQL.User,
			config.CFG.MySQL.Password,
			config.CFG.MySQL.Host,
			config.CFG.MySQL.Port,
			config.CFG.MySQL.Database,
			config.CFG.MySQL.Charset,
			config.CFG.MySQL.ParseTime,
			config.CFG.MySQL.Loc,
		)
		dialector = mysql.Open(dsn)
		maxOpenConns = config.CFG.MySQL.MaxOpenConns
		maxIdleConns = config.CFG.MySQL.MaxIdleConns
		connMaxLifetime = config.CFG.MySQL.ConnMaxLifetime
	} else if config.CFG.Postgres.Enable {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.CFG.Postgres.Host,
			config.CFG.Postgres.Port,
			config.CFG.Postgres.User,
			config.CFG.Postgres.Password,
			config.CFG.Postgres.Database,
			config.CFG.Postgres.SSLMode,
		)
		dialector = postgres.Open(dsn)
		maxOpenConns = config.CFG.Postgres.MaxOpenConns
		maxIdleConns = config.CFG.Postgres.MaxIdleConns
		connMaxLifetime = config.CFG.Postgres.ConnMaxLifetime
	} else if config.CFG.SQLite.Enable {
		dialector = sqlite.Open(config.CFG.SQLite.File)
		maxOpenConns = config.CFG.SQLite.MaxOpenConns
		maxIdleConns = config.CFG.SQLite.MaxIdleConns
		connMaxLifetime = config.CFG.SQLite.ConnMaxLifetime
	} else {
		return nil, errors.New("no database configured")
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	if config.CFG.App.Debug {
		db = db.Debug()
	}
	return db, nil
}
