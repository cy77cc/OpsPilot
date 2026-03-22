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

const defaultPrefix = "ai:checkpoint:"

type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

type Metadata struct {
	SessionID    string
	RunID        string
	CheckpointID string
	UserID       uint64
	Scene        string
}

type Store struct {
	dao    *aidao.AICheckpointDAO
	redis  redisClient
	prefix string
	ttl    time.Duration
}

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

func (s *Store) redisKey(checkpointID string) string {
	return s.prefix + checkpointID
}
