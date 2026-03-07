package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/redis/go-redis/v9"
)

type InMemoryCheckPointStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewInMemoryCheckPointStore() *InMemoryCheckPointStore {
	return &InMemoryCheckPointStore{data: make(map[string][]byte)}
}

func (s *InMemoryCheckPointStore) Set(_ context.Context, key string, value []byte) error {
	if s == nil {
		return fmt.Errorf("checkpoint store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = append([]byte(nil), value...)
	return nil
}

func (s *InMemoryCheckPointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	if s == nil {
		return nil, false, fmt.Errorf("checkpoint store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.data[key]
	if !ok {
		return nil, false, nil
	}
	return append([]byte(nil), val...), true, nil
}

type RedisCheckPointStore struct {
	client redis.UniversalClient
}

func NewRedisCheckPointStore(client redis.UniversalClient) *RedisCheckPointStore {
	return &RedisCheckPointStore{client: client}
}

func (s *RedisCheckPointStore) Set(ctx context.Context, key string, value []byte) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("checkpoint store redis client is nil")
	}
	if err := s.client.Set(ctx, key, value, 0).Err(); err != nil {
		return fmt.Errorf("save checkpoint %q: %w", key, err)
	}
	return nil
}

func (s *RedisCheckPointStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s == nil || s.client == nil {
		return nil, false, fmt.Errorf("checkpoint store redis client is nil")
	}
	val, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get checkpoint %q: %w", key, err)
	}
	return val, true, nil
}
