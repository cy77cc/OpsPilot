package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/core/config"
	"github.com/redis/go-redis/v9"
)

func NewRdb() (redis.UniversalClient, error) {
	if !config.CFG.Redis.Enable {
		return nil, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         config.CFG.Redis.Addr,
		Password:     config.CFG.Redis.Password,
		DB:           config.CFG.Redis.DB,
		PoolSize:     config.CFG.Redis.PoolSize,
		MinIdleConns: config.CFG.Redis.MinIdleConns,
		DialTimeout:  config.CFG.Redis.DialTimeout,
		ReadTimeout:  config.CFG.Redis.ReadTimeout,
		WriteTimeout: config.CFG.Redis.WriteTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}

	return rdb, nil
}
