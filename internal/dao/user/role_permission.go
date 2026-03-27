// Package user 提供用户相关的数据访问对象。
//
// 本文件实现角色权限关联 DAO，管理角色与权限的绑定关系。
package user

import (
	"context"

	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RolePermissionDAO 是角色权限关联数据访问对象。
type RolePermissionDAO struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   redis.UniversalClient       // Redis 客户端
}

// NewRolePermissionDAO 创建角色权限关联 DAO 实例。
func NewRolePermissionDAO(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *RolePermissionDAO {
	return &RolePermissionDAO{db: db, cache: cache, rdb: rdb}
}

// Create 创建角色权限关联。
func (d *RolePermissionDAO) Create(ctx context.Context, rolePermission *model.RolePermission) error {
	return d.db.WithContext(ctx).Create(rolePermission).Error
}

// Delete 删除角色权限关联。
func (d *RolePermissionDAO) Delete(ctx context.Context, id model.UserID) error {
	return d.db.WithContext(ctx).Delete(&model.RolePermission{}, id).Error
}

// GetByRoleID 根据角色 ID 查询所有权限关联。
func (d *RolePermissionDAO) GetByRoleID(ctx context.Context, roleID model.UserID) ([]model.RolePermission, error) {
	var rolePermissions []model.RolePermission
	err := d.db.WithContext(ctx).Where("role_id = ?", roleID).Find(&rolePermissions).Error
	if err != nil {
		return nil, err
	}
	return rolePermissions, nil
}
