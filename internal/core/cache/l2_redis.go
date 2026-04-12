// Package cache 提供两级缓存实现。
//
// 本文件实现基于 Redis 的 L2 缓存存储。
// 作为 Facade 的二级缓存后端使用。
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisL2 是基于 Redis 的 L2 缓存实现。
type RedisL2 struct {
	client redis.UniversalClient // Redis 客户端
}

// NewRedisL2 创建 Redis L2 缓存实例。
func NewRedisL2(client redis.UniversalClient) *RedisL2 {
	return &RedisL2{client: client}
}

// Get 从 Redis 获取缓存值。
//
// 如果键不存在返回 ErrCacheMiss。
func (r *RedisL2) Get(ctx context.Context, key string) (string, error) {
	if r == nil || r.client == nil {
		return "", ErrCacheMiss
	}
	v, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}
	return v, err
}

// Set 设置 Redis 缓存值并指定过期时间。
func (r *RedisL2) Set(ctx context.Context, key string, val string, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Set(ctx, key, val, ttl).Err()
}

// Delete 删除 Redis 中的指定键。
func (r *RedisL2) Delete(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil || len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}
