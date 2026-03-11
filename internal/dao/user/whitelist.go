// Package user 提供用户相关的数据访问对象。
//
// 本文件实现 JWT 白名单 DAO，管理 Token 的有效性状态。
package user

import (
	"context"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// WhiteListDao 是 JWT 白名单数据访问对象。
type WhiteListDao struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   redis.UniversalClient       // Redis 客户端
}

// NewWhiteListDao 创建白名单 DAO 实例。
func NewWhiteListDao(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *WhiteListDao {
	return &WhiteListDao{db: db, cache: cache, rdb: rdb}
}

// AddToWhitelist 将 Token 添加到白名单。
//
// TTL 设置为 Token 的剩余有效期。
func (w *WhiteListDao) AddToWhitelist(ctx context.Context, token string, exp time.Time) error {
	ttl := time.Until(exp)
	return w.rdb.Set(ctx, constants.JwtWhiteListKey+token, 1, ttl).Err()
}

// DeleteToken 从白名单移除 Token（用于登出）。
func (w *WhiteListDao) DeleteToken(ctx context.Context, token string) error {
	return w.rdb.Del(ctx, constants.JwtWhiteListKey+token).Err()
}

// IsWhitelisted 检查 Token 是否在白名单中。
func (w *WhiteListDao) IsWhitelisted(ctx context.Context, token string) (bool, error) {
	res, err := w.rdb.Exists(ctx, constants.JwtWhiteListKey+token).Result()
	return res > 0, err
}
