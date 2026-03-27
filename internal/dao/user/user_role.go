// Package user 提供用户相关的数据访问对象。
//
// 本文件实现用户角色关联 DAO，管理用户与角色的绑定关系。
package user

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// UserRoleDAO 是用户角色关联数据访问对象。
type UserRoleDAO struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   redis.UniversalClient       // Redis 客户端
}

// NewUserRoleDAO 创建用户角色关联 DAO 实例。
func NewUserRoleDAO(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *UserRoleDAO {
	return &UserRoleDAO{db: db, cache: cache, rdb: rdb}
}

// Create 创建用户角色关联。
func (d *UserRoleDAO) Create(ctx context.Context, userRole *model.UserRole) error {
	return d.db.WithContext(ctx).Create(userRole).Error
}

// Delete 删除用户角色关联。
func (d *UserRoleDAO) Delete(ctx context.Context, id model.UserID) error {
	return d.db.WithContext(ctx).Delete(&model.UserRole{}, id).Error
}

// GetByUserID 根据用户 ID 查询所有角色关联。
func (d *UserRoleDAO) GetByUserID(ctx context.Context, userID model.UserID) ([]model.UserRole, error) {
	var userRoles []model.UserRole
	err := d.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userRoles).Error
	if err != nil {
		return nil, err
	}
	return userRoles, nil
}
