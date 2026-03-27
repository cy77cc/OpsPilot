package storage

import (
	"context"
	"log"
	"time"

	"github.com/cy77cc/OpsPilot/internal/config"
	"github.com/redis/go-redis/v9"
)

func MustNewRdb() redis.UniversalClient {
	if !config.CFG.Redis.Enable {
		return nil
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
		log.Fatalf("failed to connect redis: %v", err)
	}

	return rdb
}
