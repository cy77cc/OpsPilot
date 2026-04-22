package user

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/cy77cc/OpsPilot/internal/constants"
	usermodel "github.com/cy77cc/OpsPilot/internal/modules/user/model"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeUserCache struct {
	values map[string]string
}

func newFakeUserCache() *fakeUserCache {
	return &fakeUserCache{values: make(map[string]string)}
}

func (f *fakeUserCache) SetEx(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	return f.Set(context.Background(), key, value, 0)
}

func (f *fakeUserCache) Del(_ context.Context, keys ...string) *redis.IntCmd {
	var removed int64
	for _, key := range keys {
		if _, ok := f.values[key]; ok {
			delete(f.values, key)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

func (f *fakeUserCache) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := f.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

func (f *fakeUserCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	switch typed := value.(type) {
	case string:
		f.values[key] = typed
	default:
		buf, err := json.Marshal(typed)
		if err != nil {
			return redis.NewStatusResult("", err)
		}
		f.values[key] = string(buf)
	}
	return redis.NewStatusResult("OK", nil)
}

func (f *fakeUserCache) TTL(_ context.Context, key string) *redis.DurationCmd {
	if _, ok := f.values[key]; !ok {
		return redis.NewDurationResult(-1, redis.Nil)
	}
	return redis.NewDurationResult(time.Minute, nil)
}

func (f *fakeUserCache) Expire(_ context.Context, key string, _ time.Duration) *redis.BoolCmd {
	if _, ok := f.values[key]; !ok {
		return redis.NewBoolResult(false, redis.Nil)
	}
	return redis.NewBoolResult(true, nil)
}

func TestUpdate_InvalidatesUserIDAndUsernameCaches(t *testing.T) {
	t.Parallel()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&usermodel.User{}); err != nil {
		t.Fatalf("auto migrate user: %v", err)
	}

	user := &usermodel.User{
		Username:     "olduser01",
		PasswordHash: "hashed-password",
		Email:        "old@example.com",
		Status:       1,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	cache := newFakeUserCache()
	idKey := constants.UserIdKey + "1"
	oldUsernameKey := constants.UserNameKey + "olduser01"
	newUsernameKey := constants.UserNameKey + "newuser01"
	cache.values[idKey] = `{"id":1}`
	cache.values[oldUsernameKey] = `{"username":"olduser01"}`
	cache.values[newUsernameKey] = `{"username":"newuser01"}`

	dao := NewUserDAO(db, nil, cache)
	user.Username = "newuser01"
	user.Email = "new@example.com"

	if err := dao.Update(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	if _, ok := cache.values[idKey]; ok {
		t.Fatalf("expected user id cache %q to be invalidated", idKey)
	}
	if _, ok := cache.values[oldUsernameKey]; ok {
		t.Fatalf("expected old username cache %q to be invalidated", oldUsernameKey)
	}
	if _, ok := cache.values[newUsernameKey]; ok {
		t.Fatalf("expected new username cache %q to be invalidated", newUsernameKey)
	}

	var saved usermodel.User
	if err := db.First(&saved, user.ID).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if saved.Username != "newuser01" {
		t.Fatalf("expected username to be updated, got %q", saved.Username)
	}
}
