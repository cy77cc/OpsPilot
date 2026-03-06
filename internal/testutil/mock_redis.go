package testutil

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// MockRedisClient implements redis.UniversalClient for testing purposes.
type MockRedisClient struct {
	data map[string]string
	exp  map[string]time.Time
}

// NewMockRedisClient creates a new MockRedisClient.
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]string),
		exp:  make(map[string]time.Time),
	}
}

// Set implements redis.UniversalClient.
func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	m.data[key] = value.(string)
	if expiration > 0 {
		m.exp[key] = time.Now().Add(expiration)
	}
	return redis.NewStatusResult("OK", nil)
}

// SetEx implements redis.UniversalClient.
func (m *MockRedisClient) SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return m.Set(ctx, key, value, expiration)
}

// Get implements redis.UniversalClient.
func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	val, ok := m.data[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(val, nil)
}

// Del implements redis.UniversalClient.
func (m *MockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			delete(m.exp, key)
			count++
		}
	}
	return redis.NewIntResult(count, nil)
}

// Exists implements redis.UniversalClient.
func (m *MockRedisClient) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	count := int64(0)
	for _, key := range keys {
		if val, ok := m.data[key]; ok {
			// Check expiration
			if exp, hasExp := m.exp[key]; hasExp {
				if time.Now().Before(exp) {
					count++
				} else {
					// Expired, clean up
					delete(m.data, key)
					delete(m.exp, key)
				}
			} else {
				_ = val // Just to avoid unused variable warning
				count++
			}
		}
	}
	return redis.NewIntResult(count, nil)
}

// Expire implements redis.UniversalClient.
func (m *MockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if _, ok := m.data[key]; ok {
		m.exp[key] = time.Now().Add(expiration)
		return redis.NewBoolResult(true, nil)
	}
	return redis.NewBoolResult(false, nil)
}

// TTL implements redis.UniversalClient.
func (m *MockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	if exp, ok := m.exp[key]; ok {
		return redis.NewDurationResult(time.Until(exp), nil)
	}
	return redis.NewDurationResult(-1, nil)
}

// Ping implements redis.UniversalClient.
func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusResult("PONG", nil)
}

// Close implements redis.UniversalClient.
func (m *MockRedisClient) Close() error {
	return nil
}

// The following methods are required to satisfy redis.UniversalClient interface
// but are not used in our auth tests.

func (m *MockRedisClient) Do(ctx context.Context, args ...interface{}) *redis.Cmd {
	return redis.NewCmd(ctx, args...)
}

func (m *MockRedisClient) Process(ctx context.Context, cmd redis.Cmder) error {
	return nil
}

func (m *MockRedisClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return nil
}

func (m *MockRedisClient) PSubscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return nil
}

func (m *MockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult(nil, nil)
}

func (m *MockRedisClient) SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd {
	return redis.NewBoolResult(false, nil)
}

func (m *MockRedisClient) LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) LPop(ctx context.Context, key string) *redis.StringCmd {
	return redis.NewStringResult("", redis.Nil)
}

func (m *MockRedisClient) RPop(ctx context.Context, key string) *redis.StringCmd {
	return redis.NewStringResult("", redis.Nil)
}

func (m *MockRedisClient) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return redis.NewStringSliceResult(nil, nil)
}

func (m *MockRedisClient) HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return redis.NewStringResult("", redis.Nil)
}

func (m *MockRedisClient) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return redis.NewMapStringStringResult(nil, nil)
}

func (m *MockRedisClient) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) Decr(ctx context.Context, key string) *redis.IntCmd {
	return redis.NewIntResult(0, nil)
}

func (m *MockRedisClient) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return redis.NewSliceResult(nil, nil)
}

func (m *MockRedisClient) MSet(ctx context.Context, values ...interface{}) *redis.StatusCmd {
	return redis.NewStatusResult("OK", nil)
}

func (m *MockRedisClient) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	return redis.NewScanCmd(ctx, nil, nil, 0, nil)
}

func (m *MockRedisClient) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult(nil, nil)
}

func (m *MockRedisClient) FlushDB(ctx context.Context) *redis.StatusCmd {
	m.data = make(map[string]string)
	m.exp = make(map[string]time.Time)
	return redis.NewStatusResult("OK", nil)
}

// AddUniversalClientMethods adds remaining methods needed to satisfy the interface.
// These are minimal implementations for testing purposes.

func (m *MockRedisClient) WrapProcess(fn func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error) {
}

func (m *MockRedisClient) WrapProcessPipeline(fn func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error) {
}

func (m *MockRedisClient) Pipeline() redis.Pipeliner {
	return nil
}

func (m *MockRedisClient) Pipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return nil, nil
}

func (m *MockRedisClient) TxPipeline() redis.Pipeliner {
	return nil
}

func (m *MockRedisClient) TxPipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return nil, nil
}

func (m *MockRedisClient) Watch(ctx context.Context, fn func(*redis.Tx) error, keys ...string) error {
	return nil
}
