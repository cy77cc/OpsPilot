// Package node 提供节点相关的数据访问对象。
//
// 本文件实现节点 DAO，支持 Redis 缓存和延迟双删策略。
package node

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// NodeDao 是节点数据访问对象。
type NodeDao struct {
	db    *gorm.DB                    // GORM 数据库实例
	cache *expirable.LRU[string, any] // 本地 LRU 缓存
	rdb   redis.UniversalClient       // Redis 客户端
}

// NewNodeDao 创建节点 DAO 实例。
func NewNodeDao(db *gorm.DB, cache *expirable.LRU[string, any], rdb redis.UniversalClient) *NodeDao {
	return &NodeDao{
		db:    db,
		cache: cache,
		rdb:   rdb,
	}
}

// Create 创建节点并缓存到 Redis。
func (d *NodeDao) Create(ctx context.Context, node *model.Node) error {
	if err := d.db.WithContext(ctx).Create(node).Error; err != nil {
		return err
	}

	key := fmt.Sprintf("%s%d", constants.NodeKey, node.ID)

	if d.rdb != nil {
		if bs, err := json.Marshal(node); err == nil {
			d.rdb.SetEx(ctx, key, bs, constants.RdbTTL)
		}
	}

	return nil
}

// Update 更新节点，使用延迟双删策略保证缓存一致性。
func (d *NodeDao) Update(ctx context.Context, node *model.Node) error {
	// 双删策略
	key := fmt.Sprintf("%s%d", constants.NodeKey, node.ID)
	if d.rdb != nil {
		if err := d.rdb.Del(ctx, key).Err(); err != nil {
			return nil
		}
	}

	if err := d.db.WithContext(ctx).Save(node).Error; err != nil {
		return err
	}

	time.Sleep(50 * time.Millisecond)
	if d.rdb != nil {
		if err := d.rdb.Del(ctx, key).Err(); err != nil {
			return nil
		}
	}
	return nil
}

// Delete 删除节点并清除缓存。
func (d *NodeDao) Delete(ctx context.Context, id model.NodeID) error {
	key := fmt.Sprintf("%s%d", constants.NodeKey, id)
	if d.rdb != nil {
		if err := d.rdb.Del(ctx, key).Err(); err != nil {
			return nil
		}
	}

	if err := d.db.WithContext(ctx).Delete(&model.Node{}, id).Error; err != nil {
		return err
	}
	return nil
}

// FindSSHKeyByID 根据节点 ID 查询 SSH 密钥，优先从 Redis 获取。
func (d *NodeDao) FindSSHKeyByID(ctx context.Context, id model.NodeID) (*model.SSHKey, error) {
	key := fmt.Sprintf("%s%d", constants.SSHKey, id)
	var data model.SSHKey
	if d.rdb != nil {
		bs, err := d.rdb.Get(ctx, key).Bytes()
		if err == nil {
			if err := json.Unmarshal(bs, &data); err != nil {
				return &data, nil
			}
		}
	}

	if err := d.db.WithContext(ctx).First(&data, id).Error; err != nil {
		return nil, err
	}

	// 保存到redis
	b, err := json.Marshal(data)
	if err == nil && d.rdb != nil {
		d.rdb.SetNX(ctx, key, b, constants.RdbTTL)
	}

	return &data, nil

}
