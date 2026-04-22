// Package user 提供用户相关的数据访问对象。
//
// 本文件实现用户 DAO，支持 Redis 缓存和延迟双删策略。
package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type userCacheClient interface {
	SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// UserDAO 是用户数据访问对象。
type UserDAO struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   userCacheClient             // Redis 客户端
}

// NewUserDAO 创建用户 DAO 实例。
func NewUserDAO(db *gorm.DB, cache *expirable.LRU[string, any], rdb userCacheClient) *UserDAO {
	return &UserDAO{db: db, cache: cache, rdb: rdb}
}

// Create 创建用户并缓存到 Redis。
func (d *UserDAO) Create(ctx context.Context, user *model.User) error {
	if err := d.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	key := fmt.Sprintf("%s%d", constants.UserIdKey, user.ID)
	if d.rdb != nil {
		if bs, err := json.Marshal(&user); err == nil {
			d.rdb.SetEx(ctx, key, bs, constants.RdbTTL)
		}
	}
	return nil
}

// Update 更新用户，使用延迟双删策略保证缓存一致性。
func (d *UserDAO) Update(ctx context.Context, user *model.User) error {
	var current model.User
	if d.rdb != nil {
		if err := d.db.WithContext(ctx).
			Select("id", "username").
			First(&current, user.ID).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
	}

	if err := d.db.WithContext(ctx).Save(user).Error; err != nil {
		return err
	}

	return d.invalidateUserCaches(ctx, user.ID, current.Username, user.Username)
}

// Delete 删除用户并清除缓存。
func (d *UserDAO) Delete(ctx context.Context, id model.UserID) error {
	key := fmt.Sprintf("%s%d", constants.UserIdKey, id)
	if d.rdb != nil {
		d.rdb.Del(ctx, key)
	}
	return d.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// FindOneById 根据用户 ID 查询用户，优先从 Redis 获取。
func (d *UserDAO) FindOneById(ctx context.Context, id model.UserID) (*model.User, error) {
	var user model.User
	// 先从redis获取数据
	key := fmt.Sprintf("%s%d", constants.UserIdKey, id)
	if d.rdb != nil {
		buf, err := d.rdb.Get(ctx, key).Bytes()
		if err == nil {
			if err := json.Unmarshal(buf, &user); err == nil {
				// 续约，加时间，方式缓存雪崩，穿透
				_ = extendUserCacheTTL(ctx, d.rdb, key)
				return &user, nil
			}
		}
	}
	err := d.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&user)
	if err == nil && d.rdb != nil {
		d.rdb.Set(ctx, key, b, constants.RdbTTL)
	}

	return &user, nil
}

// FindOneByUsername 根据用户名查询用户，优先从 Redis 获取。
func (d *UserDAO) FindOneByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	key := fmt.Sprintf("%s%s", constants.UserNameKey, username)
	if d.rdb != nil {
		buf, err := d.rdb.Get(ctx, key).Bytes()
		if err == nil {
			if err := json.Unmarshal(buf, &user); err == nil {
				// 不处理error，可以容忍失败
				_ = extendUserCacheTTL(ctx, d.rdb, key)
				return &user, nil
			}
		}
	}
	err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&user)
	if err == nil && d.rdb != nil {
		d.rdb.Set(ctx, key, b, constants.RdbTTL)
	}

	return &user, nil
}

func (d *UserDAO) invalidateUserCaches(ctx context.Context, id model.UserID, usernames ...string) error {
	if d.rdb == nil {
		return nil
	}

	keys := []string{fmt.Sprintf("%s%d", constants.UserIdKey, id)}
	seen := map[string]struct{}{keys[0]: {}}
	for _, username := range usernames {
		username = strings.TrimSpace(username)
		if username == "" {
			continue
		}
		key := fmt.Sprintf("%s%s", constants.UserNameKey, username)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	return d.rdb.Del(ctx, keys...).Err()
}

func extendUserCacheTTL(ctx context.Context, cache userCacheClient, key string) error {
	ttl, err := cache.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl < 0 {
		ttl = 0
	}
	return cache.Expire(ctx, key, ttl+constants.RdbAddTTL).Err()
}
