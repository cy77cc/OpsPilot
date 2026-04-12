// Package cache 提供两级缓存实现。
//
// 本文件实现 L1（内存）+ L2（Redis）两级缓存的门面模式。
// 提供 Get/Set/Delete 操作，支持自动回填和命中率统计。
package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// ErrCacheMiss 表示缓存未命中的错误。
var ErrCacheMiss = errors.New("cache miss")

// 缓存来源常量，用于 GetOrLoad 返回值。
const (
	SourceL1   = "l1"   // L1 内存缓存命中
	SourceL2   = "l2"   // L2 Redis 缓存命中
	SourceLoad = "load" // 从加载器获取
)

// L2Store 定义二级缓存接口。
type L2Store interface {
	Get(ctx context.Context, key string) (string, error)          // 获取缓存
	Set(ctx context.Context, key string, val string, ttl time.Duration) error // 设置缓存
	Delete(ctx context.Context, keys ...string) error             // 删除缓存
}

// Stats 包含缓存命中率统计。
type Stats struct {
	L1Hits        int64 `json:"l1_hits"`        // L1 命中次数
	L2Hits        int64 `json:"l2_hits"`        // L2 命中次数
	Misses        int64 `json:"misses"`         // 未命中次数
	FallbackLoads int64 `json:"fallback_loads"` // 回退加载次数
}

// Facade 实现两级缓存门面。
//
// L1 为内存 LRU 缓存，L2 为 Redis 等外部缓存。
// 所有计数器使用 atomic 保证线程安全。
type Facade struct {
	l1 *expirable.LRU[string, string] // 内存 LRU 缓存
	l2 L2Store                         // 二级缓存存储

	l1Hits        atomic.Int64 // L1 命中计数
	l2Hits        atomic.Int64 // L2 命中计数
	misses        atomic.Int64 // 未命中计数
	fallbackLoads atomic.Int64 // 回退加载计数
}

// NewFacade 创建两级缓存门面实例。
func NewFacade(l1 *expirable.LRU[string, string], l2 L2Store) *Facade {
	return &Facade{l1: l1, l2: l2}
}

// Get 从 L1 缓存获取值。
//
// 仅检查 L1，不穿透到 L2。
// 返回值和是否命中的布尔值。
func (f *Facade) Get(key string) (string, bool) {
	if f == nil || f.l1 == nil {
		return "", false
	}
	v, ok := f.l1.Get(key)
	if ok {
		f.l1Hits.Add(1)
	}
	return v, ok
}

// Set 同时设置 L1 和 L2 缓存。
//
// L1 立即写入，L2 异步写入（忽略错误）。
func (f *Facade) Set(ctx context.Context, key, val string, ttl time.Duration) {
	if f == nil {
		return
	}
	if f.l1 != nil {
		f.l1.Add(key, val)
	}
	if f.l2 != nil {
		_ = f.l2.Set(ctx, key, val, ttl)
	}
}

// Delete 同时删除 L1 和 L2 缓存。
func (f *Facade) Delete(ctx context.Context, keys ...string) {
	if f == nil {
		return
	}
	if f.l1 != nil {
		for _, key := range keys {
			f.l1.Remove(key)
		}
	}
	if f.l2 != nil {
		_ = f.l2.Delete(ctx, keys...)
	}
}

// GetOrLoad 实现缓存穿透加载模式。
//
// 查找顺序：L1 → L2 → loader 函数。
// 返回值、来源标识和错误。
func (f *Facade) GetOrLoad(ctx context.Context, key string, ttl time.Duration, loader func(context.Context) (string, error)) (string, string, error) {
	if v, ok := f.Get(key); ok {
		return v, SourceL1, nil
	}
	f.misses.Add(1)

	if f != nil && f.l2 != nil {
		if v, err := f.l2.Get(ctx, key); err == nil {
			f.l2Hits.Add(1)
			if f.l1 != nil {
				f.l1.Add(key, v)
			}
			return v, SourceL2, nil
		} else if !errors.Is(err, ErrCacheMiss) {
			f.fallbackLoads.Add(1)
		}
	}

	v, err := loader(ctx)
	if err != nil {
		return "", "", err
	}
	f.Set(ctx, key, v, ttl)
	return v, SourceLoad, nil
}

// Stats 返回缓存命中率统计快照。
func (f *Facade) Stats() Stats {
	if f == nil {
		return Stats{}
	}
	return Stats{
		L1Hits:        f.l1Hits.Load(),
		L2Hits:        f.l2Hits.Load(),
		Misses:        f.misses.Load(),
		FallbackLoads: f.fallbackLoads.Load(),
	}
}
