// Package user 提供用户相关的数据访问对象。
//
// 本文件实现权限 DAO，提供权限的 CRUD 操作。
package user

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// PermissionDAO 是权限数据访问对象。
type PermissionDAO struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   redis.UniversalClient       // Redis 客户端
}

// NewPermissionDAO 创建权限 DAO 实例。
func NewPermissionDAO(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *PermissionDAO {
	return &PermissionDAO{db: db, cache: cache, rdb: rdb}
}

// Create 创建权限。
func (d *PermissionDAO) Create(ctx context.Context, permission *model.Permission) error {
	return d.db.WithContext(ctx).Create(permission).Error
}

// Update 更新权限。
func (d *PermissionDAO) Update(ctx context.Context, permission *model.Permission) error {
	return d.db.WithContext(ctx).Save(permission).Error
}

// Delete 删除权限。
func (d *PermissionDAO) Delete(ctx context.Context, id model.UserID) error {
	return d.db.WithContext(ctx).Delete(&model.Permission{}, id).Error
}

// GetByID 根据权限 ID 查询权限。
func (d *PermissionDAO) GetByID(ctx context.Context, id model.UserID) (*model.Permission, error) {
	var permission model.Permission
	err := d.db.WithContext(ctx).First(&permission, id).Error
	if err != nil {
		return nil, err
	}
	return &permission, nil
}
