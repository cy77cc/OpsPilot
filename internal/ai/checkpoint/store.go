// Package checkpoint 实现 AI 检查点存储。
//
// 检查点用于 Human-in-the-Loop 工作流，当需要人工审批时暂停执行，
// 审批完成后通过检查点恢复执行状态。
//
// 存储策略：
//   - 主存储：MySQL 数据库（通过 AICheckpointDAO）
//   - 缓存层：Redis（可选，用于加速读取）
//
// 使用流程：
//  1. 调用 ContextWithMetadata 将元数据注入上下文
//  2. 调用 Set 保存检查点
//  3. 调用 Get 恢复检查点
package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	aidao "github.com/cy77cc/OpsPilot/internal/dao/ai"
	"github.com/cy77cc/OpsPilot/internal/model"
	"github.com/cy77cc/OpsPilot/internal/runtimectx"
	"github.com/redis/go-redis/v9"
)

// defaultPrefix Redis 键前缀。
const defaultPrefix = "ai:checkpoint:"

// redisClient Redis 客户端接口。
type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

// Metadata 检查点元数据。
type Metadata struct {
	SessionID    string
	RunID        string
	CheckpointID string
	UserID       uint64
	Scene        string
}

// Store 检查点存储。
//
// 实现双层存储策略：Redis 缓存 + MySQL 持久化。
type Store struct {
	dao    *aidao.AICheckpointDAO
	redis  redisClient
	prefix string
	ttl    time.Duration
}

// NewStore 创建检查点存储实例。
//
// 参数:
//   - dao: 数据库 DAO
//   - redisClient: Redis 客户端（可选）
//   - prefix: Redis 键前缀（空则使用默认值）
//
// 返回: 存储实例
func NewStore(dao *aidao.AICheckpointDAO, redisClient redisClient, prefix string) *Store {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		prefix = defaultPrefix
	}
	return &Store{
		dao:    dao,
		redis:  redisClient,
		prefix: prefix,
		ttl:    24 * time.Hour,
	}
}

// ContextWithMetadata 将元数据注入上下文。
//
// 用于在保存检查点时自动获取会话信息。
// 如果 CheckpointID 为空，则保留已有值。
//
// 参数:
//   - ctx: 原始上下文
//   - meta: 元数据
//
// 返回: 包含元数据的上下文
func ContextWithMetadata(ctx context.Context, meta Metadata) context.Context {
	existing := runtimectx.AIMetadataFrom(ctx)
	if strings.TrimSpace(meta.CheckpointID) == "" {
		meta.CheckpointID = existing.CheckpointID
	}
	return runtimectx.WithAIMetadata(ctx, runtimectx.AIMetadata{
		SessionID:    meta.SessionID,
		RunID:        meta.RunID,
		CheckpointID: meta.CheckpointID,
		UserID:       meta.UserID,
		Scene:        meta.Scene,
	})
}

// metadataFromContext 从上下文中提取元数据。
func metadataFromContext(ctx context.Context) Metadata {
	meta := runtimectx.AIMetadataFrom(ctx)
	return Metadata{
		SessionID:    meta.SessionID,
		RunID:        meta.RunID,
		CheckpointID: meta.CheckpointID,
		UserID:       meta.UserID,
		Scene:        meta.Scene,
	}
}

// Get 获取检查点数据。
//
// 读取策略：先查 Redis，未命中则查数据库并回填缓存。
//
// 参数:
//   - ctx: 上下文
//   - checkpointID: 检查点 ID
//
// 返回:
//   - []byte: 检查点数据
//   - bool: 是否存在
//   - error: 错误信息
func (s *Store) Get(ctx context.Context, checkpointID string) ([]byte, bool, error) {
	checkpointID = strings.TrimSpace(checkpointID)
	if checkpointID == "" {
		return nil, false, nil
	}

	if s.redis != nil {
		raw, err := s.redis.Get(ctx, s.redisKey(checkpointID)).Bytes()
		switch {
		case err == nil:
			return append([]byte(nil), raw...), true, nil
		case errors.Is(err, redis.Nil):
		default:
			return nil, false, err
		}
	}

	if s.dao == nil {
		return nil, false, nil
	}

	record, err := s.dao.Get(ctx, checkpointID)
	if err != nil || record == nil {
		return nil, false, err
	}
	payload := append([]byte(nil), record.Payload...)
	if s.redis != nil {
		_ = s.redis.Set(ctx, s.redisKey(checkpointID), payload, s.ttl).Err()
	}
	return payload, true, nil
}

// Set 保存检查点数据。
//
// 写入策略：同时写入数据库和 Redis（双写）。
//
// 参数:
//   - ctx: 上下文（应包含元数据）
//   - checkpointID: 检查点 ID
//   - checkpoint: 检查点数据
//
// 返回: 错误信息
func (s *Store) Set(ctx context.Context, checkpointID string, checkpoint []byte) error {
	checkpointID = strings.TrimSpace(checkpointID)
	if checkpointID == "" {
		return fmt.Errorf("checkpoint id is empty")
	}
	payload := append([]byte(nil), checkpoint...)
	expiresAt := time.Now().Add(s.ttl)

	if s.dao != nil {
		meta := metadataFromContext(ctx)
		if err := s.dao.Upsert(ctx, &model.AICheckpoint{
			CheckpointID: checkpointID,
			SessionID:    strings.TrimSpace(meta.SessionID),
			RunID:        strings.TrimSpace(meta.RunID),
			UserID:       meta.UserID,
			Scene:        strings.TrimSpace(meta.Scene),
			Payload:      payload,
			ExpiresAt:    &expiresAt,
		}); err != nil {
			return err
		}
	}

	if s.redis != nil {
		if err := s.redis.Set(ctx, s.redisKey(checkpointID), payload, s.ttl).Err(); err != nil {
			return err
		}
	}
	return nil
}

// redisKey 生成 Redis 键。
func (s *Store) redisKey(checkpointID string) string {
	return s.prefix + checkpointID
}
