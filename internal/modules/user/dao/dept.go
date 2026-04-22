package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type deptCacheClient interface {
	SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// DepartmentDAO 是部门数据访问对象。
type DepartmentDAO struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   deptCacheClient             // Redis 客户端
}

// NewDepartmentDAO 创建部门 DAO 实例。
func NewDepartmentDAO(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *DepartmentDAO {
	return &DepartmentDAO{db: db, cache: cache, rdb: rdb}
}

// Create 创建部门并缓存到 Redis。
func (d *DepartmentDAO) Create(ctx context.Context, dept *model.Department) error {
	if err := d.db.WithContext(ctx).Create(dept).Error; err != nil {
		return err
	}
	key := fmt.Sprintf("%s%d", constants.DeptIdKey, dept.ID)
	if d.rdb != nil {
		if bs, err := json.Marshal(&dept); err == nil {
			d.rdb.SetEx(ctx, key, bs, constants.RdbTTL)
		}
	}
	return nil
}

// Update 更新部门，并清除缓存。
func (d *DepartmentDAO) Update(ctx context.Context, dept *model.Department) error {
	if err := d.db.WithContext(ctx).Save(dept).Error; err != nil {
		return err
	}
	key := fmt.Sprintf("%s%d", constants.DeptIdKey, dept.ID)
	if d.rdb != nil {
		d.rdb.Del(ctx, key)
	}
	return nil
}

// Delete 删除部门并清除缓存。
func (d *DepartmentDAO) Delete(ctx context.Context, id model.UserID) error {
	key := fmt.Sprintf("%s%d", constants.DeptIdKey, id)
	if d.rdb != nil {
		d.rdb.Del(ctx, key)
	}
	return d.db.WithContext(ctx).Delete(&model.Department{}, id).Error
}

// FindByID 根据部门 ID 查询部门，优先从 Redis 获取。
func (d *DepartmentDAO) FindByID(ctx context.Context, id model.UserID) (*model.Department, error) {
	var dept model.Department
	key := fmt.Sprintf("%s%d", constants.DeptIdKey, id)
	if d.rdb != nil {
		buf, err := d.rdb.Get(ctx, key).Bytes()
		if err == nil {
			if err := json.Unmarshal(buf, &dept); err == nil {
				_ = d.extendCacheTTL(ctx, key)
				return &dept, nil
			}
		}
	}
	err := d.db.WithContext(ctx).First(&dept, id).Error
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(&dept); err == nil && d.rdb != nil {
		d.rdb.Set(ctx, key, b, constants.RdbTTL)
	}

	return &dept, nil
}

// FindAll 获取所有部门。
func (d *DepartmentDAO) FindAll(ctx context.Context) ([]model.Department, error) {
	var depts []model.Department
	err := d.db.WithContext(ctx).Find(&depts).Error
	return depts, err
}

// GetTree 获取部门树（目前返回所有部门，由逻辑层处理树结构）。
func (d *DepartmentDAO) GetTree(ctx context.Context) ([]model.Department, error) {
	return d.FindAll(ctx)
}

func (d *DepartmentDAO) extendCacheTTL(ctx context.Context, key string) error {
	if d.rdb == nil {
		return nil
	}
	ttl, err := d.rdb.TTL(ctx, key).Result()
	if err != nil {
		return err
	}
	if ttl < 0 {
		ttl = 0
	}
	return d.rdb.Expire(ctx, key, ttl+constants.RdbAddTTL).Err()
}
