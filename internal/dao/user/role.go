// Package user 提供用户相关的数据访问对象。
//
// 本文件实现角色 DAO，提供角色的 CRUD 操作。
package user

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RoleDAO 是角色数据访问对象。
type RoleDAO struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   redis.UniversalClient       // Redis 客户端
}

// NewRoleDAO 创建角色 DAO 实例。
func NewRoleDAO(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *RoleDAO {
	return &RoleDAO{db: db, cache: cache, rdb: rdb}
}

// Create 创建角色。
func (d *RoleDAO) Create(ctx context.Context, role *model.Role) error {
	return d.db.WithContext(ctx).Create(role).Error
}

// Update 更新角色。
func (d *RoleDAO) Update(ctx context.Context, role *model.Role) error {
	return d.db.WithContext(ctx).Save(role).Error
}

// Delete 删除角色。
func (d *RoleDAO) Delete(ctx context.Context, id model.UserID) error {
	return d.db.WithContext(ctx).Delete(&model.Role{}, id).Error
}

// GetByID 根据角色 ID 查询角色。
func (d *RoleDAO) GetByID(ctx context.Context, id model.UserID) (*model.Role, error) {
	var role model.Role
	err := d.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}
